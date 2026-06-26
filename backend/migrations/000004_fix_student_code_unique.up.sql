-- Drop the global UNIQUE constraint on student_code
ALTER TABLE students DROP CONSTRAINT students_student_code_key;

-- Add tenant-scoped UNIQUE index (excludes soft-deleted students)
CREATE UNIQUE INDEX idx_active_student_code
ON students (tenant_id, student_code)
WHERE deleted_at IS NULL;
