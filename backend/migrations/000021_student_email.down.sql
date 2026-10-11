BEGIN;

ALTER TABLE students DROP COLUMN IF EXISTS email;
DROP INDEX IF EXISTS idx_students_email_tenant;

COMMIT;