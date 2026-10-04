-- Index for per-test question lookups.
--
-- questions has a FK to tests but Postgres does not index FK columns, so both the
-- `has_questions` filter and the `question_count` column added to the tests list
-- would otherwise run a sequential scan of `questions` per test row. That list is
-- requested with limit=10000 by the assignment pickers, so the cost is real.
CREATE INDEX IF NOT EXISTS idx_questions_test_id ON questions (test_id);