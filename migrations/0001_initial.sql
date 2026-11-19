CREATE TABLE IF NOT EXISTS spots (
  id TEXT PRIMARY KEY,
  prefecture TEXT NOT NULL,
  municipality TEXT NOT NULL,
  name TEXT NOT NULL,
  song_title TEXT NOT NULL,
  song_type TEXT NOT NULL,
  relation_type TEXT NOT NULL,
  evidence_summary TEXT NOT NULL,
  location_text TEXT NOT NULL,
  lat REAL,
  lon REAL,
  source_url TEXT NOT NULL,
  source_type TEXT NOT NULL,
  evidence_level TEXT NOT NULL,
  status TEXT NOT NULL,
  created_at TEXT NOT NULL DEFAULT CURRENT_TIMESTAMP
);

CREATE TABLE IF NOT EXISTS users (
  id TEXT PRIMARY KEY,
  created_at TEXT NOT NULL DEFAULT CURRENT_TIMESTAMP
);

CREATE TABLE IF NOT EXISTS questions (
  id INTEGER PRIMARY KEY AUTOINCREMENT,
  external_key TEXT UNIQUE,
  spot_id TEXT NOT NULL REFERENCES spots(id) ON DELETE CASCADE,
  prompt TEXT NOT NULL,
  choices_json TEXT NOT NULL,
  answer_index INTEGER NOT NULL CHECK(answer_index >= 0),
  sort_order INTEGER NOT NULL DEFAULT 0,
  active INTEGER NOT NULL DEFAULT 1 CHECK(active IN (0, 1)),
  source_url TEXT,
  source_title TEXT,
  evidence_level TEXT
);
CREATE INDEX IF NOT EXISTS idx_questions_spot_active ON questions(spot_id, active, sort_order);

CREATE TABLE IF NOT EXISTS attempts (
  id INTEGER PRIMARY KEY AUTOINCREMENT,
  user_id TEXT NOT NULL REFERENCES users(id),
  spot_id TEXT NOT NULL REFERENCES spots(id),
  correct_count INTEGER NOT NULL,
  total_count INTEGER NOT NULL,
  passed INTEGER NOT NULL CHECK(passed IN (0, 1)),
  created_at TEXT NOT NULL DEFAULT CURRENT_TIMESTAMP
);
CREATE INDEX IF NOT EXISTS idx_attempts_user_spot ON attempts(user_id, spot_id, created_at);

CREATE TABLE IF NOT EXISTS stamps (
  id INTEGER PRIMARY KEY AUTOINCREMENT,
  user_id TEXT NOT NULL REFERENCES users(id),
  spot_id TEXT NOT NULL REFERENCES spots(id),
  earned_at TEXT NOT NULL DEFAULT CURRENT_TIMESTAMP,
  UNIQUE(user_id, spot_id)
);