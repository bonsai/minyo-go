import { readFile, writeFile } from "node:fs/promises";
import { tmpdir } from "node:os";
import { join } from "node:path";
import { execFileSync } from "node:child_process";

const remote = process.argv.includes("--remote");
const quote = value => value == null ? "NULL" : typeof value === "number" ? String(value) : "'" + String(value).replaceAll("'", "''") + "'";
const spots = JSON.parse(await readFile(new URL("../spots-verified.json", import.meta.url), "utf8")).records;
const quizzes = JSON.parse(await readFile(new URL("../quizzes.json", import.meta.url), "utf8")).questions;
if (!spots.length || !quizzes.length) throw new Error("seed source is empty");
const perSpot = new Map();
for (const q of quizzes) perSpot.set(q.spot_id, (perSpot.get(q.spot_id) || 0) + 1);
for (const [id, count] of perSpot) if (count !== 3) throw new Error(`${id} has ${count} questions; expected 3`);
const sql = [];
for (const s of spots) {
  sql.push(`INSERT INTO spots(id,prefecture,municipality,name,song_title,song_type,relation_type,evidence_summary,location_text,lat,lon,source_url,source_type,evidence_level,status)
    VALUES (${[s.id,s.prefecture,s.municipality,s.name,s.song_title,s.song_type,s.relation_type,s.evidence_summary,s.location_text,s.lat,s.lon,s.source_url,s.source_type,s.evidence_level,s.status].map(quote).join(",")})
    ON CONFLICT(id) DO UPDATE SET prefecture=excluded.prefecture,municipality=excluded.municipality,name=excluded.name,song_title=excluded.song_title,song_type=excluded.song_type,relation_type=excluded.relation_type,evidence_summary=excluded.evidence_summary,location_text=excluded.location_text,lat=excluded.lat,lon=excluded.lon,source_url=excluded.source_url,source_type=excluded.source_type,evidence_level=excluded.evidence_level,status=excluded.status;`);
}
for (const id of perSpot.keys()) sql.push(`UPDATE questions SET active=0 WHERE spot_id=${quote(id)};`);
for (const q of quizzes) {
  sql.push(`INSERT INTO questions(external_key,spot_id,prompt,choices_json,answer_index,sort_order,active,source_url,source_title,evidence_level)
    VALUES (${[q.key,q.spot_id,q.prompt,JSON.stringify(q.choices),q.answer_index,q.sort_order,1,q.source_url,q.source_title,q.evidence_level].map(quote).join(",")})
    ON CONFLICT(external_key) DO UPDATE SET spot_id=excluded.spot_id,prompt=excluded.prompt,choices_json=excluded.choices_json,answer_index=excluded.answer_index,sort_order=excluded.sort_order,active=1,source_url=excluded.source_url,source_title=excluded.source_title,evidence_level=excluded.evidence_level;`);
}
const file = join(tmpdir(), `minyo-go-seed-${Date.now()}.sql`);
await writeFile(file, sql.join("\n"));
const args = ["d1","execute","minyo-go",...(remote?["--remote"]:["--local"]),"--file",file];
execFileSync("npx",["wrangler",...args],{stdio:"inherit"});
console.log(`Seed submitted: ${spots.length} spots, ${quizzes.length} questions (${remote?"remote":"local"} D1)`);
