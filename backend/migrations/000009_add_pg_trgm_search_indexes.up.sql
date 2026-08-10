CREATE EXTENSION IF NOT EXISTS pg_trgm;

CREATE INDEX idx_coaches_name_trgm   ON coaches  USING gin (name gin_trgm_ops);
CREATE INDEX idx_students_name_trgm  ON students USING gin (name gin_trgm_ops);
CREATE INDEX idx_students_code_trgm  ON students USING gin (student_code gin_trgm_ops);
CREATE INDEX idx_tests_title_trgm    ON tests    USING gin (title gin_trgm_ops);
CREATE INDEX idx_subjects_name_trgm  ON subjects USING gin (name gin_trgm_ops);
