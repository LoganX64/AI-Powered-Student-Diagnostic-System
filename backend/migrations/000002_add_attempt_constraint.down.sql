-- Remove unique constraint
ALTER TABLE attempts DROP CONSTRAINT IF EXISTS unique_assignment_attempt;

-- Reset status back to default
UPDATE assignments SET status = 'assigned' WHERE status = 'submitted';
