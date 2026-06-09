-- Grant programs (the core catalog)
CREATE TABLE grants (
    id                      UUID PRIMARY KEY DEFAULT uuid_generate_v4(),
    slug                    TEXT NOT NULL UNIQUE,
    title                   TEXT NOT NULL,
    funder_name             TEXT NOT NULL,
    funder_type             TEXT NOT NULL DEFAULT 'federal',  -- federal, state, local, private, foundation, corporate
    program_number          TEXT,                             -- e.g. CFDA/ALN number
    opportunity_number      TEXT,                             -- e.g. FOA-HHS-2024-001
    agency                  TEXT,
    sub_agency              TEXT,
    description             TEXT,
    synopsis                TEXT,
    full_nofo_text          TEXT,                             -- raw NOFO/FOA text
    category                TEXT,                             -- education, housing, health, etc.
    subcategory             TEXT,
    focus_areas             TEXT[] NOT NULL DEFAULT '{}',
    eligible_org_types      TEXT[] NOT NULL DEFAULT '{}',     -- nonprofit, government, tribal, etc.
    eligible_populations    TEXT[] NOT NULL DEFAULT '{}',     -- youth, veterans, etc.
    eligible_states         TEXT[] NOT NULL DEFAULT '{}',     -- empty = nationwide
    eligible_counties       TEXT[] NOT NULL DEFAULT '{}',
    requires_501c3          BOOLEAN NOT NULL DEFAULT FALSE,
    requires_audited_fin    BOOLEAN NOT NULL DEFAULT FALSE,
    requires_indirect_rate  BOOLEAN NOT NULL DEFAULT FALSE,
    requires_match          BOOLEAN NOT NULL DEFAULT FALSE,
    match_percentage        NUMERIC(5,2),
    min_award_amount        BIGINT,                           -- cents
    max_award_amount        BIGINT,                           -- cents
    avg_award_amount        BIGINT,                           -- cents
    total_funding_available BIGINT,                           -- cents
    num_awards_expected     INT,
    application_url         TEXT,
    faq_url                 TEXT,
    webinar_url             TEXT,
    status                  TEXT NOT NULL DEFAULT 'active',   -- active, closed, archived, draft
    deadline                DATE,
    open_date               DATE,
    period_of_performance   TEXT,                             -- e.g. "12 months", "3 years"
    is_recurring            BOOLEAN NOT NULL DEFAULT FALSE,
    recurrence_notes        TEXT,
    difficulty_level        TEXT NOT NULL DEFAULT 'medium',   -- easy, medium, hard, very_hard
    competition_level       TEXT NOT NULL DEFAULT 'medium',   -- low, medium, high, very_high
    tags                    TEXT[] NOT NULL DEFAULT '{}',
    embedding               VECTOR(1536),                     -- pgvector embedding of description+synopsis
    metadata                JSONB NOT NULL DEFAULT '{}',
    created_by              UUID REFERENCES users(id),
    created_at              TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    updated_at              TIMESTAMPTZ NOT NULL DEFAULT NOW()
);

CREATE INDEX idx_grants_slug ON grants(slug);
CREATE INDEX idx_grants_status ON grants(status);
CREATE INDEX idx_grants_funder_type ON grants(funder_type);
CREATE INDEX idx_grants_category ON grants(category);
CREATE INDEX idx_grants_deadline ON grants(deadline);
CREATE INDEX idx_grants_focus_areas ON grants USING GIN(focus_areas);
CREATE INDEX idx_grants_eligible_org_types ON grants USING GIN(eligible_org_types);
CREATE INDEX idx_grants_eligible_populations ON grants USING GIN(eligible_populations);
CREATE INDEX idx_grants_eligible_states ON grants USING GIN(eligible_states);
CREATE INDEX idx_grants_tags ON grants USING GIN(tags);
CREATE INDEX idx_grants_embedding ON grants USING ivfflat (embedding vector_cosine_ops) WITH (lists = 100);
CREATE INDEX idx_grants_title_trgm ON grants USING GIN(title gin_trgm_ops);

-- NOFO document chunks for RAG
CREATE TABLE nofo_chunks (
    id          UUID PRIMARY KEY DEFAULT uuid_generate_v4(),
    grant_id    UUID NOT NULL REFERENCES grants(id) ON DELETE CASCADE,
    chunk_index INT NOT NULL,
    section     TEXT,                             -- e.g. "Eligibility", "Evaluation Criteria"
    content     TEXT NOT NULL,
    token_count INT,
    embedding   VECTOR(1536),
    created_at  TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    UNIQUE(grant_id, chunk_index)
);

CREATE INDEX idx_nofo_chunks_grant_id ON nofo_chunks(grant_id);
CREATE INDEX idx_nofo_chunks_embedding ON nofo_chunks USING ivfflat (embedding vector_cosine_ops) WITH (lists = 100);

-- Grant scoring criteria weights (admin-configurable per grant)
CREATE TABLE grant_scoring_criteria (
    id              UUID PRIMARY KEY DEFAULT uuid_generate_v4(),
    grant_id        UUID NOT NULL REFERENCES grants(id) ON DELETE CASCADE,
    criterion_key   TEXT NOT NULL,   -- e.g. "org_type", "population_match", "geographic_match"
    weight          NUMERIC(5,2) NOT NULL DEFAULT 1.0,
    is_required     BOOLEAN NOT NULL DEFAULT FALSE,
    disqualifying   BOOLEAN NOT NULL DEFAULT FALSE,
    description     TEXT,
    created_at      TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    UNIQUE(grant_id, criterion_key)
);

CREATE INDEX idx_scoring_criteria_grant_id ON grant_scoring_criteria(grant_id);
