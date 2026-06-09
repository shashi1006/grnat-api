-- Grant applications tracked by an organization
CREATE TABLE grant_applications (
    id                  UUID PRIMARY KEY DEFAULT uuid_generate_v4(),
    org_id              UUID NOT NULL REFERENCES organizations(id) ON DELETE CASCADE,
    grant_id            UUID NOT NULL REFERENCES grants(id),
    assigned_to         UUID REFERENCES users(id),
    status              TEXT NOT NULL DEFAULT 'prospect',
    -- Status flow: prospect -> researching -> drafting -> internal_review -> submitted -> awarded | rejected | withdrawn
    stage               TEXT NOT NULL DEFAULT 'pre_application',
    -- Stages: pre_application, application, post_submission
    priority            TEXT NOT NULL DEFAULT 'medium',     -- low, medium, high, critical
    compatibility_score NUMERIC(5,2),
    submission_date     DATE,
    award_amount        BIGINT,                             -- cents, if awarded
    award_date          DATE,
    rejection_reason    TEXT,
    notes               TEXT,
    internal_deadline   DATE,
    metadata            JSONB NOT NULL DEFAULT '{}',
    created_at          TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    updated_at          TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    UNIQUE(org_id, grant_id)
);

CREATE INDEX idx_applications_org_id ON grant_applications(org_id);
CREATE INDEX idx_applications_grant_id ON grant_applications(grant_id);
CREATE INDEX idx_applications_status ON grant_applications(status);
CREATE INDEX idx_applications_assigned ON grant_applications(assigned_to);

-- Application activities/audit log
CREATE TABLE application_activities (
    id              UUID PRIMARY KEY DEFAULT uuid_generate_v4(),
    application_id  UUID NOT NULL REFERENCES grant_applications(id) ON DELETE CASCADE,
    user_id         UUID REFERENCES users(id),
    activity_type   TEXT NOT NULL,   -- status_change, note_added, document_uploaded, assigned, etc.
    old_value       TEXT,
    new_value       TEXT,
    note            TEXT,
    created_at      TIMESTAMPTZ NOT NULL DEFAULT NOW()
);

CREATE INDEX idx_app_activities_app_id ON application_activities(application_id);

-- AI-generated narratives for application sections
CREATE TABLE ai_narratives (
    id              UUID PRIMARY KEY DEFAULT uuid_generate_v4(),
    org_id          UUID NOT NULL REFERENCES organizations(id) ON DELETE CASCADE,
    grant_id        UUID NOT NULL REFERENCES grants(id),
    application_id  UUID REFERENCES grant_applications(id) ON DELETE SET NULL,
    section_key     TEXT NOT NULL,   -- e.g. "need_statement", "project_description", "eval_plan"
    prompt_used     TEXT,
    content         TEXT NOT NULL,
    word_count      INT,
    model_used      TEXT NOT NULL DEFAULT 'claude-3-5-sonnet-20241022',
    tokens_in       INT,
    tokens_out      INT,
    is_approved     BOOLEAN DEFAULT FALSE,
    version         INT NOT NULL DEFAULT 1,
    created_at      TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    updated_at      TIMESTAMPTZ NOT NULL DEFAULT NOW()
);

CREATE INDEX idx_narratives_org_id ON ai_narratives(org_id);
CREATE INDEX idx_narratives_grant_id ON ai_narratives(grant_id);
CREATE INDEX idx_narratives_app_id ON ai_narratives(application_id);
CREATE INDEX idx_narratives_section ON ai_narratives(section_key);

-- Application documents
CREATE TABLE application_documents (
    id              UUID PRIMARY KEY DEFAULT uuid_generate_v4(),
    application_id  UUID NOT NULL REFERENCES grant_applications(id) ON DELETE CASCADE,
    name            TEXT NOT NULL,
    doc_type        TEXT NOT NULL,   -- narrative, budget, attachment, letter_of_support, etc.
    storage_url     TEXT,
    size_bytes      BIGINT,
    mime_type       TEXT,
    uploaded_by     UUID REFERENCES users(id),
    created_at      TIMESTAMPTZ NOT NULL DEFAULT NOW()
);

CREATE INDEX idx_app_docs_app_id ON application_documents(application_id);
