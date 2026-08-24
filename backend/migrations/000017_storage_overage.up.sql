BEGIN;

-- Exam video must be stored regardless of quota; we meter actual usage and
-- bill per-GB overage instead of hard-blocking mid-exam uploads.

ALTER TABLE storage_usage ADD COLUMN IF NOT EXISTS overage_bytes BIGINT NOT NULL DEFAULT 0;

CREATE TABLE IF NOT EXISTS storage_chunks (
    id            SERIAL PRIMARY KEY,
    tenant_id     INT NOT NULL REFERENCES tenants(id) ON DELETE CASCADE,
    assignment_id INT NOT NULL REFERENCES assignments(id) ON DELETE CASCADE,
    chunk_index   TEXT NOT NULL,
    bytes         BIGINT NOT NULL DEFAULT 0,
    created_at    TIMESTAMP DEFAULT CURRENT_TIMESTAMP,
    UNIQUE (assignment_id, chunk_index)
);

COMMIT;
