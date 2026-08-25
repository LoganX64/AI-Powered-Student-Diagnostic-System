DROP INDEX IF EXISTS idx_assignments_student_test_status;
ALTER TABLE assignments ADD CONSTRAINT assignments_student_id_test_id_key UNIQUE (student_id, test_id);
