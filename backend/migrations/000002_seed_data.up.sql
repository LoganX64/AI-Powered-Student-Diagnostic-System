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
(id, test_id, question_text, option_a, option_b, option_c, option_d, correct_answer, marks, neg_marks, importance, difficulty, type, expected_time, concept_tag)
VALUES
(
    1, 1,
    'What is 2 + 2?',
    '3', '4', '5', '6',
    'B',
    4, 1, 'A', 'M', 'Theory', 30,
    'Addition'
),
(
    2, 1,
    'Solve: 5x = 20',
    '2', '3', '4', '5',
    'C',
    4, 1, 'B', 'H', 'Theory', 45,
    'Linear Equation'
),
(
    3, 2,
    'Speed formula?',
    'Distance/Time', 'Time/Distance', 'Velocity*Time', 'None',
    'A',
    5, 1, 'A', 'E', 'Practical', 25,
    'Physics Basics'
)
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