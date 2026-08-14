-- ─────────────────────────────────────────────────────────────
-- Exam tiering foundation: batches, assignment policy, jobs, timing
-- ─────────────────────────────────────────────────────────────

-- Batches (tenant-wide; no coach_id — all tenant admins/coaches manage)
CREATE TABLE IF NOT EXISTS batches (
    id         SERIAL PRIMARY KEY,
    tenant_id  INT NOT NULL,
    name       VARCHAR(150) NOT NULL,
    created_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP,
    FOREIGN KEY (tenant_id) REFERENCES tenants(id) ON DELETE CASCADE
);

-- Optional batch membership on students
ALTER TABLE students ADD COLUMN IF NOT EXISTS batch_id INT NULL REFERENCES batches(id) ON DELETE SET NULL;

-- Assignment integrity policy + pricing + delivery mode
ALTER TABLE assignments ADD COLUMN IF NOT EXISTS integrity_policy JSONB NOT NULL DEFAULT '{}'::jsonb;
ALTER TABLE assignments ADD COLUMN IF NOT EXISTS estimated_cost NUMERIC(12,2) NULL;
ALTER TABLE assignments ADD COLUMN IF NOT EXISTS delivery_mode VARCHAR(20) NOT NULL DEFAULT 'standard';

-- Jobs table (async SQI compute + progress)
CREATE TABLE IF NOT EXISTS jobs (
    id         SERIAL PRIMARY KEY,
    tenant_id  INT NOT NULL,
    type       VARCHAR(40) NOT NULL,
    payload    JSONB NOT NULL DEFAULT '{}'::jsonb,
    total      INT NOT NULL DEFAULT 0,
    done       INT NOT NULL DEFAULT 0,
    failed     INT NOT NULL DEFAULT 0,
    status     VARCHAR(20) NOT NULL DEFAULT 'pending',
    created_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP,
    updated_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP,
    FOREIGN KEY (tenant_id) REFERENCES tenants(id) ON DELETE CASCADE
);

-- attempts.status for server-authoritative timing (in_progress -> submitted)
ALTER TABLE attempts ADD COLUMN IF NOT EXISTS status VARCHAR(20) NOT NULL DEFAULT 'submitted';

-- answer_logs unique per attempt+question so autosave can upsert
CREATE UNIQUE INDEX IF NOT EXISTS idx_answer_logs_attempt_question
    ON answer_logs (attempt_id, question_id);
