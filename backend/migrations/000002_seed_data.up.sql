-- ========================
-- STUDENTS
-- ========================
INSERT INTO students (id, student_code, name)
VALUES
(1, 'STU001', 'Alice'),
(2, 'STU002', 'Bob')
ON CONFLICT (id) DO NOTHING;

-- ========================
-- COACHES
-- ========================
INSERT INTO coaches (id, name, email)
VALUES
(1, 'Coach A', 'coachA@test.com'),
(2, 'Coach B', 'coachB@test.com')
ON CONFLICT (id) DO NOTHING;

-- ========================
-- SUBJECTS
-- ========================
INSERT INTO subjects (id, name)
VALUES
(1, 'Math'),
(2, 'Physics')
ON CONFLICT (id) DO NOTHING;

-- ========================
-- TESTS
-- ========================
INSERT INTO tests (id, subject_id)
VALUES
(1, 1),
(2, 2)
ON CONFLICT (id) DO NOTHING;

-- ========================
-- QUESTIONS
-- ========================
INSERT INTO questions 
(id, test_id, correct_answer, marks, neg_marks, importance, difficulty, type, expected_time)
VALUES
(1, 1, 'A', 4, 1, 'A', 'M', 'Theory', 30),
(2, 1, 'C', 4, 1, 'B', 'H', 'Theory', 45),
(3, 2, 'B', 5, 1, 'A', 'E', 'Practical', 25)
ON CONFLICT (id) DO NOTHING;

-- ========================
-- ASSIGNMENTS
-- ========================
INSERT INTO assignments (id, student_id, test_id, coach_id)
VALUES
(1, 1, 1, 1),
(2, 1, 2, 2),
(3, 2, 1, 1)
ON CONFLICT (id) DO NOTHING;