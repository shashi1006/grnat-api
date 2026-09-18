-- Submission-pathway eligibility: distinguish who can SUBMIT an application
-- (eligible_applicants) from who can benefit from the funds (eligible_org_types).
-- Pass-through programs (HSGP, NSGP, JAG formula share, HPP) are administered by
-- a state-level agency; other orgs can only receive subawards.
ALTER TABLE grants
    ADD COLUMN eligible_applicants       TEXT[] NOT NULL DEFAULT '{}',
    ADD COLUMN submission_pathway        TEXT NOT NULL DEFAULT 'direct',  -- direct, pass_through, mixed
    ADD COLUMN pass_through_note         TEXT,
    ADD COLUMN submission_requirements   JSONB NOT NULL DEFAULT '{}',     -- required forms, narrative structure, set-asides
    ADD COLUMN requirements_extracted_at TIMESTAMPTZ;

ALTER TABLE compatibility_scores
    ADD COLUMN subaward_only BOOLEAN NOT NULL DEFAULT FALSE;

CREATE INDEX idx_grants_submission_pathway ON grants(submission_pathway);

-- Backfill imported federal grants whose direct applicants are restricted.

-- HSGP (FY2026 import + legacy FEMA HSGP): only the State Administrative
-- Agency may submit; one application per SAA. Also clear the fabricated
-- per-applicant award cap — real awards are state allocations.
UPDATE grants SET
    submission_pathway  = 'pass_through',
    eligible_applicants = '{state-administrative-agency}',
    pass_through_note   = 'Only the State Administrative Agency (SAA) may submit HSGP applications to FEMA. Other organizations can only receive subawards through their state SAA (e.g., NY DHSES for New York applicants).',
    max_award_amount    = NULL
WHERE title ILIKE '%Homeland Security Grant Program%';

-- NSGP: nonprofits submit an Investment Justification through their state
-- SAA, which files the consolidated application with FEMA.
UPDATE grants SET
    submission_pathway  = 'pass_through',
    eligible_applicants = '{state-administrative-agency}',
    pass_through_note   = 'Nonprofits apply through their state SAA (e.g., NY DHSES), which submits the consolidated NSGP application to FEMA. Direct applications to FEMA are not accepted.'
WHERE title ILIKE '%Nonprofit Security Grant%';

-- ASPR HPP: cooperative agreements go to state/territory health departments
-- and select localities; hospitals and coalitions are subrecipients.
UPDATE grants SET
    submission_pathway  = 'pass_through',
    eligible_applicants = '{state-administrative-agency,municipality-government}',
    pass_through_note   = 'HPP cooperative agreements are awarded to state, territorial, and select local health departments. Hospitals and health care coalitions participate as subrecipients of those awards.'
WHERE title ILIKE '%Hospital Preparedness%';

-- JAG: state formula share is SAA-administered; only units of local
-- government on the DOJ allocation list may apply directly. Universities
-- and nonprofits are not eligible applicants in either track.
UPDATE grants SET
    submission_pathway  = 'mixed',
    eligible_applicants = '{state-administrative-agency,municipality-government}',
    pass_through_note   = 'The state formula share is administered by the state administering agency. Only units of local government on the DOJ allocation list may apply for a direct JAG award; other organizations participate as subrecipients.'
WHERE title ILIKE '%Justice Assistance%' OR title ILIKE '%Byrne%';

-- COPS SVPP and STOP School Violence are direct-to-federal applications;
-- they keep the default 'direct' pathway.
