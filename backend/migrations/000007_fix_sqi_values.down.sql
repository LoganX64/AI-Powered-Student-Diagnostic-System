BEGIN;

ALTER TABLE questions DROP CONSTRAINT IF EXISTS questions_importance_check;
ALTER TABLE questions DROP CONSTRAINT IF EXISTS questions_type_check;

UPDATE questions SET importance = CASE
  WHEN importance = 'high' THEN 'A'
  WHEN importance = 'medium' THEN 'B'
  WHEN importance = 'low' THEN 'C'
END;

UPDATE questions SET type = CASE
  WHEN type = 'mcq' THEN 'Theory'
  WHEN type = 'integer' THEN 'Practical'
  ELSE type
END;

ALTER TABLE questions ALTER COLUMN importance TYPE CHAR(1);
ALTER TABLE questions ALTER COLUMN type TYPE VARCHAR(20);

ALTER TABLE questions ADD CONSTRAINT questions_importance_check
  CHECK (importance IN ('A', 'B', 'C'));
ALTER TABLE questions ADD CONSTRAINT questions_type_check
  CHECK (type IN ('Theory', 'Practical'));

COMMIT;
