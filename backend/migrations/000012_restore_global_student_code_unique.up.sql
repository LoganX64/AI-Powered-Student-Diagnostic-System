-- H3: students authenticate with only their student_code (no tenant context),
-- so the code must be globally unique to avoid ambiguous / cross-tenant login.
-- Migration 000004 relaxed the global UNIQUE constraint to a tenant-scoped
-- partial index, which made GetIDByStudentCode return an arbitrary row when two
-- tenants reused the same code. Restore global uniqueness. Auto-generated codes
-- already embed the tenant id (T{tenant}{base36-6}), so this only rejects
-- manually assigned codes that collide across tenants.

DROP INDEX IF EXISTS idx_active_student_code;

-- Defensive: if duplicate student_code values already exist (from the relaxed
-- window), keep the earliest row per code and rename the rest so the global
-- constraint can be added without failing. Renamed rows get a _dup{id} suffix
-- (unique per row) and will need a fresh code issued to the affected student.
UPDATE students s
SET student_code = s.student_code || '_dup' || s.id
WHERE s.id NOT IN (
  SELECT MIN(id) FROM students GROUP BY student_code
);

ALTER TABLE students ADD CONSTRAINT students_student_code_key UNIQUE (student_code);
