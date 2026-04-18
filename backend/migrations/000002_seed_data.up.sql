-- ========================
-- USERS (admin + coaches)
-- ========================
INSERT INTO users (id, email, password, role)
VALUES
(1, 'coachA@test.com', 'password', 'coach'),
(2, 'coachB@test.com', 'password', 'coach')
ON CONFLICT (id) DO NOTHING;


-- ========================
-- COACHES (must come AFTER users)
-- ========================
INSERT INTO coaches (id, user_id, name)
VALUES
(1, 1, 'Coach A'),
(2, 2, 'Coach B')
ON CONFLICT (id) DO NOTHING;


-- ========================
-- STUDENTS (FIXED: added coach_id)
-- ========================
INSERT INTO students (id, coach_id, student_code, name)
VALUES
(1, 1, 'STU001', 'Alice'),  -- belongs to Coach 1
(2, 1, 'STU002', 'Bob'),    -- belongs to Coach 1
(3, 2, 'STU003', 'Charlie') -- belongs to Coach 2
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
-- TESTS (owned by coach)
-- ========================
INSERT INTO tests (id, title, subject_id, coach_id, duration)
VALUES
(1, 'Math Test', 1, 1, 60),
(2, 'Physics Test', 2, 2, 60)
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
-- ASSIGNMENTS (VALIDATED RELATIONS)
-- ========================
INSERT INTO assignments (id, student_id, test_id, coach_id)
VALUES
(1, 1, 1, 1), -- Alice → Math Test → Coach 1 
(2, 3, 2, 2), -- Charlie → Physics Test → Coach 2 
(3, 2, 1, 1)  -- Bob → Math Test → Coach 1 
ON CONFLICT (id) DO NOTHING;

-- ========================
-- ADMIN USER (secure)
-- ========================
INSERT INTO users (id, email, password, role)
VALUES (
    100,
    'admin@system.com',
    '$2a$10$7QJ8F...PUT_YOUR_HASH_HERE...',
    'admin'
)
ON CONFLICT (id) DO NOTHING;


-- Payload
-- {
--   "email": "admin@system.com",
--   "password": "admin123"
-- }