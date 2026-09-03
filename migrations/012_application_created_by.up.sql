ALTER TABLE grant_applications ADD COLUMN created_by UUID REFERENCES users(id);
CREATE INDEX idx_applications_created_by ON grant_applications(created_by);
