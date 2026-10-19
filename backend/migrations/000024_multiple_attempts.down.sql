BEGIN;

ALTER TABLE assignments DROP COLUMN IF EXISTS retake_granted_by;
ALTER TABLE assignments DROP COLUMN IF EXISTS retake_granted_at;

DROP INDEX IF EXISTS idx_attempts_one_in_progress;

-- Restoring the single-attempt constraint fails if retake rows exist; that is
-- intentional so a rollback cannot silently discard attempt history.
ALTER TABLE attempts ADD CONSTRAINT unique_assignment_attempt UNIQUE (assignment_id);

COMMIT;
