-- Create tables safely
CREATE TABLE IF NOT EXISTS students (
    id SERIAL PRIMARY KEY,
    name TEXT NOT NULL,
    score INT NOT NULL
);

CREATE TABLE IF NOT EXISTS users (
    id SERIAL PRIMARY KEY,
    email TEXT UNIQUE NOT NULL,
    password TEXT NOT NULL
);

-- Seed data (avoid duplicates)

INSERT INTO students (name, score)
SELECT 'Alice', 85
WHERE NOT EXISTS (
    SELECT 1 FROM students WHERE name = 'Alice'
);

INSERT INTO students (name, score)
SELECT 'Bob', 90
WHERE NOT EXISTS (
    SELECT 1 FROM students WHERE name = 'Bob'
);

INSERT INTO users (email, password)
VALUES ('admin@example.com', 'hashedpassword')
ON CONFLICT (email) DO NOTHING;