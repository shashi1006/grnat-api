-- Analytics events
CREATE TABLE analytics_events (
    id          UUID PRIMARY KEY DEFAULT uuid_generate_v4(),
    org_id      UUID REFERENCES organizations(id) ON DELETE SET NULL,
    user_id     UUID REFERENCES users(id) ON DELETE SET NULL,
    lead_id     UUID REFERENCES leads(id) ON DELETE SET NULL,
    event_type  TEXT NOT NULL,    -- grant_viewed, score_computed, narrative_generated, application_created, etc.
    entity_type TEXT,             -- grant, application, lead, etc.
    entity_id   UUID,
    properties  JSONB NOT NULL DEFAULT '{}',
    ip_address  TEXT,
    user_agent  TEXT,
    created_at  TIMESTAMPTZ NOT NULL DEFAULT NOW()
);

CREATE INDEX idx_analytics_org_id ON analytics_events(org_id);
CREATE INDEX idx_analytics_user_id ON analytics_events(user_id);
CREATE INDEX idx_analytics_event_type ON analytics_events(event_type);
CREATE INDEX idx_analytics_created_at ON analytics_events(created_at DESC);
CREATE INDEX idx_analytics_entity ON analytics_events(entity_type, entity_id);

-- Aggregated metrics (materialized daily)
CREATE TABLE daily_metrics (
    id              UUID PRIMARY KEY DEFAULT uuid_generate_v4(),
    metric_date     DATE NOT NULL,
    org_id          UUID REFERENCES organizations(id) ON DELETE SET NULL,
    metric_key      TEXT NOT NULL,
    metric_value    NUMERIC NOT NULL DEFAULT 0,
    dimensions      JSONB NOT NULL DEFAULT '{}',
    created_at      TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    UNIQUE(metric_date, org_id, metric_key)
);

CREATE INDEX idx_daily_metrics_date ON daily_metrics(metric_date DESC);
CREATE INDEX idx_daily_metrics_org ON daily_metrics(org_id, metric_date DESC);

-- AI usage tracking
CREATE TABLE ai_usage_logs (
    id              UUID PRIMARY KEY DEFAULT uuid_generate_v4(),
    org_id          UUID REFERENCES organizations(id) ON DELETE SET NULL,
    user_id         UUID REFERENCES users(id) ON DELETE SET NULL,
    operation       TEXT NOT NULL,   -- narrative_generate, rag_query, embedding_create, score_explain
    model           TEXT NOT NULL,
    tokens_in       INT NOT NULL DEFAULT 0,
    tokens_out      INT NOT NULL DEFAULT 0,
    cost_usd_cents  INT NOT NULL DEFAULT 0,
    latency_ms      INT,
    success         BOOLEAN NOT NULL DEFAULT TRUE,
    error_message   TEXT,
    created_at      TIMESTAMPTZ NOT NULL DEFAULT NOW()
);

CREATE INDEX idx_ai_usage_org_id ON ai_usage_logs(org_id);
CREATE INDEX idx_ai_usage_created_at ON ai_usage_logs(created_at DESC);
