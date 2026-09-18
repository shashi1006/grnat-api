DROP INDEX IF EXISTS idx_grants_submission_pathway;

ALTER TABLE compatibility_scores
    DROP COLUMN IF EXISTS subaward_only;

ALTER TABLE grants
    DROP COLUMN IF EXISTS eligible_applicants,
    DROP COLUMN IF EXISTS submission_pathway,
    DROP COLUMN IF EXISTS pass_through_note,
    DROP COLUMN IF EXISTS submission_requirements,
    DROP COLUMN IF EXISTS requirements_extracted_at;
