BEGIN;

-- Result release state.
--
-- Scores are computed by a coach-requested SQI job, not on submit, and are
-- never shown to the student until explicitly released. released_at IS NULL is
-- therefore the default and means "computed but withheld".
--
-- released_by is ON DELETE SET NULL: the audit trail should survive a coach
-- leaving the organization without ever blocking that deletion.

ALTER TABLE attempt_results ADD COLUMN IF NOT EXISTS released_at TIMESTAMP NULL;

ALTER TABLE attempt_results ADD COLUMN IF NOT EXISTS released_by INT NULL
    REFERENCES users(id) ON DELETE SET NULL;

-- Listing released results for one student joins on released_at.
CREATE INDEX IF NOT EXISTS idx_attempt_results_released
    ON attempt_results (released_at) WHERE released_at IS NOT NULL;

COMMIT;