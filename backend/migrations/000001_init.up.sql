-- STUDENTS
CREATE TABLE students (
    id SERIAL PRIMARY KEY,
    student_code VARCHAR(50) UNIQUE NOT NULL,
    name VARCHAR(100) NOT NULL,
    created_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP
);

CREATE TABLE users (
    id SERIAL PRIMARY KEY,
    email VARCHAR(100) UNIQUE NOT NULL,
    password TEXT, -- nullable for Google OAuth
    role VARCHAR(20) CHECK (role IN ('admin','coach')) NOT NULL,
    created_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP
);

-- COACHES
CREATE TABLE coaches (
    id SERIAL PRIMARY KEY,
    user_id INT UNIQUE NOT NULL,
    name VARCHAR(100),

    created_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP,

    FOREIGN KEY (user_id) REFERENCES users(id) ON DELETE CASCADE
);

-- SUBJECTS
CREATE TABLE subjects (
    id SERIAL PRIMARY KEY,
    name VARCHAR(100) UNIQUE NOT NULL
);

-- TESTS
CREATE TABLE tests (
    id SERIAL PRIMARY KEY,
    title TEXT,
    subject_id INT,
    coach_id INT,
    duration INT,
    created_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP,

    FOREIGN KEY (subject_id) REFERENCES subjects(id),
    FOREIGN KEY (coach_id) REFERENCES coaches(id)
);

-- QUESTIONS
CREATE TABLE questions (
    id SERIAL PRIMARY KEY,
    test_id INT,

    question_text TEXT NOT NULL,

    option_a TEXT NOT NULL,
    option_b TEXT NOT NULL,
    option_c TEXT NOT NULL,
    option_d TEXT NOT NULL,

    correct_answer CHAR(1) CHECK (correct_answer IN ('A','B','C','D')),

    marks FLOAT NOT NULL,
    neg_marks FLOAT NOT NULL,

    importance CHAR(1) CHECK (importance IN ('A','B','C')),
    difficulty CHAR(1) CHECK (difficulty IN ('E','M','H')),
    type VARCHAR(20) CHECK (type IN ('Theory','Practical')),

    expected_time FLOAT,
    concept_tag VARCHAR(100),

    FOREIGN KEY (test_id) REFERENCES tests(id) ON DELETE CASCADE
);
-- ASSIGNMENTS
CREATE TABLE assignments (
    id SERIAL PRIMARY KEY,
    student_id INT,
    test_id INT,
    coach_id INT,

    status VARCHAR(20) DEFAULT 'assigned',
    assigned_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP,

    FOREIGN KEY (student_id) REFERENCES students(id),
    FOREIGN KEY (test_id) REFERENCES tests(id),
    FOREIGN KEY (coach_id) REFERENCES coaches(id)
);

-- ATTEMPTS
CREATE TABLE attempts (
    id SERIAL PRIMARY KEY,
    assignment_id INT NOT NULL,

    started_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP,
    submitted_at TIMESTAMP,

    FOREIGN KEY (assignment_id) REFERENCES assignments(id) ON DELETE CASCADE
);

-- ANSWER LOGS
CREATE TABLE answer_logs (
    id SERIAL PRIMARY KEY,
    attempt_id INT,
    question_id INT,

    selected_answer TEXT,
    is_correct BOOLEAN,

    time_spent FLOAT,

    marked_for_review BOOLEAN,
    revisited BOOLEAN,
    changed_answer BOOLEAN,
    was_initially_wrong BOOLEAN,

    created_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP,

    FOREIGN KEY (attempt_id) REFERENCES attempts(id) ON DELETE CASCADE,
    FOREIGN KEY (question_id) REFERENCES questions(id) ON DELETE CASCADE
);

-- ATTEMPT RESULTS (SQI OUTPUT)
CREATE TABLE attempt_results (
    id SERIAL PRIMARY KEY,
    attempt_id INT UNIQUE,

    sqi_score FLOAT,
    raw_score FLOAT,

    created_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP,

    FOREIGN KEY (attempt_id) REFERENCES attempts(id) ON DELETE CASCADE
);