BEGIN;

ALTER TABLE questions DROP CONSTRAINT IF EXISTS questions_importance_check;
ALTER TABLE questions DROP CONSTRAINT IF EXISTS questions_type_check;

ALTER TABLE questions ALTER COLUMN importance TYPE VARCHAR(10);
ALTER TABLE questions ALTER COLUMN type TYPE VARCHAR(20);

UPDATE questions SET importance = 'high' WHERE importance = 'A';
UPDATE questions SET importance = 'medium' WHERE importance = 'B';
UPDATE questions SET importance = 'low' WHERE importance = 'C';

UPDATE questions SET type = 'mcq' WHERE type = 'Theory';
UPDATE questions SET type = 'integer' WHERE type = 'Practical';

ALTER TABLE questions ADD CONSTRAINT questions_importance_check
  CHECK (importance IN ('high', 'medium', 'low'));
ALTER TABLE questions ADD CONSTRAINT questions_type_check
  CHECK (type IN ('mcq', 'multi', 'integer'));

COMMIT;
