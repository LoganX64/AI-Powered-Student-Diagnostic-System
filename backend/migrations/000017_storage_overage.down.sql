BEGIN;

DROP TABLE IF EXISTS storage_chunks;

ALTER TABLE storage_usage DROP COLUMN IF EXISTS overage_bytes;

COMMIT;
