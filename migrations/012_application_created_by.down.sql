DROP INDEX IF EXISTS idx_applications_created_by;
ALTER TABLE grant_applications DROP COLUMN IF EXISTS created_by;
