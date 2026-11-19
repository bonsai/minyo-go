ALTER TABLE questions ADD COLUMN external_key TEXT;
CREATE UNIQUE INDEX IF NOT EXISTS idx_questions_external_key
    ON questions(external_key);
