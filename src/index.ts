export interface Env {
  DB: D1Database;
  ALLOW_TEST_MODE?: string;
}

type Answer = { question_id: number; answer_index: number };
type AttemptBody = {
  user_id: string; spot_id: string; answers: Answer[];
  latitude?: number; longitude?: number; test_mode?: boolean;
};

const json = (data: unknown, status = 200) =>
  new Response(JSON.stringify(data), { status, headers: { "content-type": "application/json; charset=utf-8", "cache-control": "no-store" } });
const fail = (error: string, status: number) => json({ error }, status);

function distanceKm(a: number, b: number, c: number, d: number): number {
  const rad = (v: number) => v * Math.PI / 180;
  const x = rad(c - a), y = rad(d - b);
  const h = Math.sin(x / 2) ** 2 + Math.cos(rad(a)) * Math.cos(rad(c)) * Math.sin(y / 2) ** 2;
  return 6371 * 2 * Math.asin(Math.sqrt(h));
}

async function listSpots(env: Env) {
  const { results } = await env.DB.prepare(`SELECT id, prefecture, municipality, name, song_title, song_type,
    relation_type, location_text, lat, lon, status FROM spots ORDER BY prefecture, municipality, name`).all();
  return json({ spots: results, count: results.length });
}

async function getQuiz(env: Env, spotId: string) {
  const spot = await env.DB.prepare("SELECT id FROM spots WHERE id = ?").bind(spotId).first();
  if (!spot) return fail("spot not found", 404);
  const { results } = await env.DB.prepare(`SELECT id, prompt, choices_json FROM questions
    WHERE spot_id = ? AND active = 1 ORDER BY sort_order, id LIMIT 4`).bind(spotId).all<{id:number;prompt:string;choices_json:string}>();
  if (results.length !== 3) return fail("this spot does not have exactly three active questions yet", 409);
  return json({ spot_id: spotId, questions: results.map(q => ({ id:q.id, prompt:q.prompt, choices:JSON.parse(q.choices_json) })) });
}

async function submitAttempt(request: Request, env: Env) {
  let body: AttemptBody;
  try { body = await request.json() as AttemptBody; } catch { return fail("invalid request body", 400); }
  const userId = String(body.user_id || "").trim(), spotId = String(body.spot_id || "").trim();
  if (!userId || !spotId || userId.length > 128 || body.answers?.length !== 3) return fail("user_id, spot_id and exactly three answers are required", 400);
  const spot = await env.DB.prepare("SELECT lat, lon FROM spots WHERE id = ?").bind(spotId).first<{lat:number|null;lon:number|null}>();
  if (!spot) return fail("spot not found", 404);
  if (body.test_mode) {
    if (env.ALLOW_TEST_MODE !== "true") return fail("test mode is disabled on this server", 403);
  } else {
    const lat = body.latitude, lon = body.longitude;
    if (typeof lat !== "number" || typeof lon !== "number" || !Number.isFinite(lat) || !Number.isFinite(lon) || lat < -90 || lat > 90 || lon < -180 || lon > 180) return fail("valid latitude and longitude are required", 400);
    if (spot.lat == null || spot.lon == null) return fail("this spot has no field-verified check-in coordinates", 409);
    if (distanceKm(lat, lon, spot.lat, spot.lon) > 30) return fail("outside the 30 km GET radius", 403);
  }
  const { results: questions } = await env.DB.prepare(`SELECT id, answer_index FROM questions
    WHERE spot_id = ? AND active = 1 ORDER BY sort_order, id`).bind(spotId).all<{id:number;answer_index:number}>();
  if (questions.length !== 3) return fail("this spot does not have exactly three active questions yet", 409);
  const expected = new Map(questions.map(q => [q.id, q.answer_index]));
  const seen = new Set<number>();
  let correct = 0;
  for (const answer of body.answers) {
    if (!Number.isInteger(answer.question_id) || !Number.isInteger(answer.answer_index) || !expected.has(answer.question_id) || seen.has(answer.question_id)) return fail("answers must contain each active question exactly once", 400);
    seen.add(answer.question_id);
    if (expected.get(answer.question_id) === answer.answer_index) correct++;
  }
  if (seen.size !== 3) return fail("answers must contain each active question exactly once", 400);
  const passed = correct === 3;
  const batch = await env.DB.batch([
    env.DB.prepare("INSERT INTO users(id) VALUES (?) ON CONFLICT(id) DO NOTHING").bind(userId),
    env.DB.prepare("INSERT INTO attempts(user_id, spot_id, correct_count, total_count, passed) VALUES (?, ?, ?, 3, ?)").bind(userId, spotId, correct, passed ? 1 : 0),
    env.DB.prepare(`INSERT INTO stamps(user_id, spot_id) SELECT ?, ? WHERE ? = 1
      ON CONFLICT(user_id, spot_id) DO NOTHING`).bind(userId, spotId, passed ? 1 : 0)
  ]);
  const stampAwarded = passed && (batch[2]?.meta?.changes ?? 0) === 1;
  return json({ correct_count: correct, total_count: 3, passed, stamp_awarded: stampAwarded,
    stamp_already_owned: passed && !stampAwarded, retry_allowed: !passed });
}

async function getStamps(env: Env, userId: string) {
  const { results } = await env.DB.prepare(`SELECT st.spot_id, sp.name AS spot_name, sp.song_title, st.earned_at
    FROM stamps st JOIN spots sp ON sp.id = st.spot_id WHERE st.user_id = ? ORDER BY st.earned_at, st.id`).bind(userId).all();
  return json({ user_id: userId, stamps: results, count: results.length });
}

async function getProgress(env: Env, userId: string) {
  const { results } = await env.DB.prepare(`SELECT sp.id AS spot_id, sp.name AS spot_name, sp.song_title, st.earned_at
    FROM spots sp LEFT JOIN stamps st ON st.spot_id = sp.id AND st.user_id = ?
    ORDER BY sp.prefecture, sp.municipality, sp.name`).bind(userId).all<{spot_id:string;spot_name:string;song_title:string;earned_at:string|null}>();
  const collected = results.filter(x => x.earned_at !== null).length;
  return json({ user_id:userId, total_spots:results.length, collected_spots:collected,
    remaining_spots:results.length-collected, spots:results.map(x => ({...x, collected:x.earned_at !== null})) });
}

export default {
  async fetch(request: Request, env: Env): Promise<Response> {
    const url = new URL(request.url), path = url.pathname;
    try {
      if (request.method === "GET" && path === "/healthz") {
        await env.DB.prepare("SELECT 1").first();
        return json({ status:"ok" });
      }
      if (request.method === "GET" && path === "/api/spots") return await listSpots(env);
      const quiz = path.match(/^\/api\/spots\/([^/]+)\/quiz$/);
      if (request.method === "GET" && quiz) return await getQuiz(env, decodeURIComponent(quiz[1]));
      if (request.method === "POST" && path === "/api/attempts") return await submitAttempt(request, env);
      const stamps = path.match(/^\/api\/users\/([^/]+)\/stamps$/);
      if (request.method === "GET" && stamps) return await getStamps(env, decodeURIComponent(stamps[1]));
      const progress = path.match(/^\/api\/users\/([^/]+)\/progress$/);
      if (request.method === "GET" && progress) return await getProgress(env, decodeURIComponent(progress[1]));
      return fail("not found", 404);
    } catch (error) {
      console.error(error);
      return fail("internal server error", 500);
    }
  }
};