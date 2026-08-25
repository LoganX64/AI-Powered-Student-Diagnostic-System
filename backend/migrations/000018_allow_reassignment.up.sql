-- Remove the unique constraint that prevents re-assigning the same test to a student.
-- Coaches/admins can now re-assign a test if the student has already submitted it.
ALTER TABLE assignments DROP CONSTRAINT IF EXISTS assignments_student_id_test_id_key;

-- Index for fast active-assignment lookups (HasActiveAssignment query).
CREATE INDEX idx_assignments_student_test_status ON assignments(student_id, test_id, status);
