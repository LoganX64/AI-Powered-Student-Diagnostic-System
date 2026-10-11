BEGIN;

DROP INDEX IF EXISTS idx_attempt_results_released;

ALTER TABLE attempt_results DROP COLUMN IF EXISTS released_by;
ALTER TABLE attempt_results DROP COLUMN IF EXISTS released_at;

COMMIT;