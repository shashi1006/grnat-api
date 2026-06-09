-- Products / services offered by ReadyGeneration
CREATE TABLE products (
    id              UUID PRIMARY KEY DEFAULT uuid_generate_v4(),
    slug            TEXT NOT NULL UNIQUE,
    name            TEXT NOT NULL,
    short_desc      TEXT,
    description     TEXT,
    price_cents     INT NOT NULL DEFAULT 0,
    price_type      TEXT NOT NULL DEFAULT 'one_time',   -- one_time, monthly, annual
    features        TEXT[] NOT NULL DEFAULT '{}',
    is_active       BOOLEAN NOT NULL DEFAULT TRUE,
    sort_order      INT NOT NULL DEFAULT 0,
    metadata        JSONB NOT NULL DEFAULT '{}',
    created_at      TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    updated_at      TIMESTAMPTZ NOT NULL DEFAULT NOW()
);

-- Organization subscriptions/purchases
CREATE TABLE org_subscriptions (
    id              UUID PRIMARY KEY DEFAULT uuid_generate_v4(),
    org_id          UUID NOT NULL REFERENCES organizations(id) ON DELETE CASCADE,
    product_id      UUID NOT NULL REFERENCES products(id),
    status          TEXT NOT NULL DEFAULT 'active',   -- active, canceled, expired, trialing
    started_at      TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    expires_at      TIMESTAMPTZ,
    canceled_at     TIMESTAMPTZ,
    payment_ref     TEXT,                             -- external payment processor reference
    metadata        JSONB NOT NULL DEFAULT '{}',
    created_at      TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    updated_at      TIMESTAMPTZ NOT NULL DEFAULT NOW()
);

CREATE INDEX idx_subscriptions_org_id ON org_subscriptions(org_id);
CREATE INDEX idx_subscriptions_status ON org_subscriptions(status);
