-- Organizations (tenants in the SaaS platform)
CREATE TABLE organizations (
    id            UUID PRIMARY KEY DEFAULT uuid_generate_v4(),
    name          TEXT NOT NULL,
    slug          TEXT NOT NULL UNIQUE,
    ein           TEXT,                          -- Employer Identification Number
    org_type      TEXT NOT NULL DEFAULT 'nonprofit',  -- nonprofit, government, tribal, faith, other
    mission       TEXT,
    address_line1 TEXT,
    address_line2 TEXT,
    city          TEXT,
    state         TEXT,
    zip           TEXT,
    county        TEXT,
    website       TEXT,
    phone         TEXT,
    logo_url      TEXT,
    plan          TEXT NOT NULL DEFAULT 'free',  -- free, starter, pro, enterprise
    plan_expires_at TIMESTAMPTZ,
    is_active     BOOLEAN NOT NULL DEFAULT TRUE,
    metadata      JSONB NOT NULL DEFAULT '{}',
    created_at    TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    updated_at    TIMESTAMPTZ NOT NULL DEFAULT NOW()
);

CREATE INDEX idx_organizations_slug ON organizations(slug);
CREATE INDEX idx_organizations_plan ON organizations(plan);
CREATE INDEX idx_organizations_state ON organizations(state);

-- Organization profile: detailed answers used for matching/scoring
CREATE TABLE organization_profiles (
    id                      UUID PRIMARY KEY DEFAULT uuid_generate_v4(),
    org_id                  UUID NOT NULL REFERENCES organizations(id) ON DELETE CASCADE,
    annual_budget           BIGINT,              -- in cents
    num_employees           INT,
    num_volunteers          INT,
    years_operating         INT,
    populations_served      TEXT[] NOT NULL DEFAULT '{}',   -- youth, elderly, veterans, homeless, etc.
    service_areas           TEXT[] NOT NULL DEFAULT '{}',   -- geographic focus areas
    program_areas           TEXT[] NOT NULL DEFAULT '{}',   -- education, housing, health, etc.
    focus_issues            TEXT[] NOT NULL DEFAULT '{}',   -- specific issues
    has_501c3               BOOLEAN DEFAULT FALSE,
    has_audited_financials  BOOLEAN DEFAULT FALSE,
    has_indirect_cost_rate  BOOLEAN DEFAULT FALSE,
    indirect_cost_rate_pct  NUMERIC(5,2),
    prior_federal_grants    BOOLEAN DEFAULT FALSE,
    narrative               TEXT,                -- free-text mission narrative
    embedding               VECTOR(1536),        -- pgvector embedding of profile narrative
    created_at              TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    updated_at              TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    UNIQUE(org_id)
);

CREATE INDEX idx_org_profiles_org_id ON organization_profiles(org_id);
CREATE INDEX idx_org_profiles_embedding ON organization_profiles USING ivfflat (embedding vector_cosine_ops) WITH (lists = 100);
CREATE INDEX idx_org_profiles_populations ON organization_profiles USING GIN(populations_served);
CREATE INDEX idx_org_profiles_program_areas ON organization_profiles USING GIN(program_areas);
