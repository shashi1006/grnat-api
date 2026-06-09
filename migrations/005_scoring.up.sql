-- Compatibility scores between an org and a grant
CREATE TABLE compatibility_scores (
    id                  UUID PRIMARY KEY DEFAULT uuid_generate_v4(),
    org_id              UUID NOT NULL REFERENCES organizations(id) ON DELETE CASCADE,
    grant_id            UUID NOT NULL REFERENCES grants(id) ON DELETE CASCADE,
    total_score         NUMERIC(5,2) NOT NULL DEFAULT 0,    -- 0-100
    tier                TEXT NOT NULL DEFAULT 'unknown',    -- strong_match, good_match, partial_match, low_match, no_match
    dimension_scores    JSONB NOT NULL DEFAULT '{}',        -- per-criterion breakdown
    disqualified        BOOLEAN NOT NULL DEFAULT FALSE,
    disqualify_reasons  TEXT[] NOT NULL DEFAULT '{}',
    strengths           TEXT[] NOT NULL DEFAULT '{}',
    gaps                TEXT[] NOT NULL DEFAULT '{}',
    recommendations     TEXT[] NOT NULL DEFAULT '{}',
    semantic_score      NUMERIC(5,2),                       -- cosine similarity score from RAG
    engine_version      TEXT NOT NULL DEFAULT 'v1',
    computed_at         TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    created_at          TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    updated_at          TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    UNIQUE(org_id, grant_id)
);

CREATE INDEX idx_compat_scores_org_id ON compatibility_scores(org_id);
CREATE INDEX idx_compat_scores_grant_id ON compatibility_scores(grant_id);
CREATE INDEX idx_compat_scores_total ON compatibility_scores(total_score DESC);
CREATE INDEX idx_compat_scores_tier ON compatibility_scores(tier);
