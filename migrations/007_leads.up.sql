-- Leads (organizations or individuals who have not yet registered)
CREATE TABLE leads (
    id                  UUID PRIMARY KEY DEFAULT uuid_generate_v4(),
    email               TEXT NOT NULL UNIQUE,
    first_name          TEXT,
    last_name           TEXT,
    org_name            TEXT,
    org_type            TEXT,
    phone               TEXT,
    city                TEXT,
    state               TEXT,
    zip                 TEXT,
    source              TEXT NOT NULL DEFAULT 'organic',   -- organic, referral, ad, webinar, demo
    utm_source          TEXT,
    utm_medium          TEXT,
    utm_campaign        TEXT,
    status              TEXT NOT NULL DEFAULT 'new',       -- new, contacted, qualified, unqualified, converted, churned
    score               INT NOT NULL DEFAULT 0,            -- lead score 0-100
    assigned_to         UUID REFERENCES users(id),
    converted_org_id    UUID REFERENCES organizations(id),
    converted_at        TIMESTAMPTZ,
    last_contacted_at   TIMESTAMPTZ,
    notes               TEXT,
    quiz_responses      JSONB NOT NULL DEFAULT '{}',       -- answers from lead qualification quiz
    interested_grants   TEXT[] NOT NULL DEFAULT '{}',      -- grant IDs they expressed interest in
    metadata            JSONB NOT NULL DEFAULT '{}',
    created_at          TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    updated_at          TIMESTAMPTZ NOT NULL DEFAULT NOW()
);

CREATE INDEX idx_leads_email ON leads(email);
CREATE INDEX idx_leads_status ON leads(status);
CREATE INDEX idx_leads_source ON leads(source);
CREATE INDEX idx_leads_assigned ON leads(assigned_to);
CREATE INDEX idx_leads_state ON leads(state);
CREATE INDEX idx_leads_score ON leads(score DESC);

-- Lead activities
CREATE TABLE lead_activities (
    id              UUID PRIMARY KEY DEFAULT uuid_generate_v4(),
    lead_id         UUID NOT NULL REFERENCES leads(id) ON DELETE CASCADE,
    user_id         UUID REFERENCES users(id),
    activity_type   TEXT NOT NULL,   -- email_sent, call_logged, meeting_scheduled, status_changed, note_added
    subject         TEXT,
    body            TEXT,
    old_value       TEXT,
    new_value       TEXT,
    scheduled_at    TIMESTAMPTZ,
    completed_at    TIMESTAMPTZ,
    created_at      TIMESTAMPTZ NOT NULL DEFAULT NOW()
);

CREATE INDEX idx_lead_activities_lead_id ON lead_activities(lead_id);
