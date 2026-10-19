BEGIN;

-- Allow more than one attempt per assignment (admin/coach-granted retakes)
-- while keeping at most one active attempt at a time.
ALTER TABLE attempts DROP CONSTRAINT IF EXISTS unique_assignment_attempt;

CREATE UNIQUE INDEX IF NOT EXISTS idx_attempts_one_in_progress
    ON attempts (assignment_id)
    WHERE status = 'in_progress';

-- Audit who granted a retake and when. The assignment status returns to
-- 'assigned' on grant so the student can start the next attempt.
ALTER TABLE assignments ADD COLUMN IF NOT EXISTS retake_granted_at TIMESTAMP NULL;
ALTER TABLE assignments ADD COLUMN IF NOT EXISTS retake_granted_by INT NULL
    REFERENCES users(id) ON DELETE SET NULL;

COMMIT;
