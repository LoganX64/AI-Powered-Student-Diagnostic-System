-- Prevent multiple attempts per assignment
ALTER TABLE attempts ADD CONSTRAINT unique_assignment_attempt UNIQUE (assignment_id);

-- Mark existing completed assignments as submitted
UPDATE assignments
SET status = 'submitted'
WHERE id IN (
    SELECT DISTINCT assignment_id FROM attempts WHERE submitted_at IS NOT NULL
);
