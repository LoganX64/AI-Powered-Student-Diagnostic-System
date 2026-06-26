-- Drop tenant-scoped index
DROP INDEX IF EXISTS idx_active_student_code;

-- Restore global UNIQUE constraint
ALTER TABLE students ADD CONSTRAINT students_student_code_key UNIQUE (student_code);
