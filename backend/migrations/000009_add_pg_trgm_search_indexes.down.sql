DROP INDEX IF EXISTS idx_coaches_name_trgm;
DROP INDEX IF EXISTS idx_students_name_trgm;
DROP INDEX IF EXISTS idx_students_code_trgm;
DROP INDEX IF EXISTS idx_tests_title_trgm;
DROP INDEX IF EXISTS idx_subjects_name_trgm;

DROP EXTENSION IF EXISTS pg_trgm;
