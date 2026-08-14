DROP INDEX IF EXISTS idx_answer_logs_attempt_question;

ALTER TABLE attempts DROP COLUMN IF EXISTS status;

ALTER TABLE assignments DROP COLUMN IF EXISTS delivery_mode;
ALTER TABLE assignments DROP COLUMN IF EXISTS estimated_cost;
ALTER TABLE assignments DROP COLUMN IF EXISTS integrity_policy;

ALTER TABLE students DROP COLUMN IF EXISTS batch_id;

DROP TABLE IF EXISTS jobs;

DROP TABLE IF EXISTS batches;
