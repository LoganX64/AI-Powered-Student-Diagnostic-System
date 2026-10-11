BEGIN;

-- Students currently authenticate with student_code only and have no email on
-- file. Result-release notifications need a deliverable address, so add one as
-- nullable: existing demo students are unaffected and simply receive no email.

ALTER TABLE students ADD COLUMN IF NOT EXISTS email VARCHAR(254) NULL;

-- Partial unique index rather than a table-level UNIQUE: NULLs would collide
-- under a plain UNIQUE, and every student without an email shares one.
-- Scoped per tenant so two organizations may hold the same address.
CREATE UNIQUE INDEX IF NOT EXISTS idx_students_email_tenant
    ON students (tenant_id, email) WHERE email IS NOT NULL;

COMMIT;