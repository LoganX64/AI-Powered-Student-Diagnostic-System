ALTER TABLE tests ADD COLUMN subject_name VARCHAR(100);

UPDATE tests SET subject_name = s.name FROM subjects s WHERE tests.subject_id = s.id;
