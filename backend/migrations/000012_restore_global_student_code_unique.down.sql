-- Revert H3 fix: drop the global UNIQUE constraint and restore the
-- tenant-scoped partial index (allows the same code in different tenants).

ALTER TABLE students DROP CONSTRAINT IF EXISTS students_student_code_key;

CREATE UNIQUE INDEX idx_active_student_code
ON students (tenant_id, student_code)
WHERE deleted_at IS NULL;
