-- Extend products with the fields needed by the Funding OS wizard's
-- "Recommended Preparedness Solutions" step, and record what an org selects.
ALTER TABLE products ADD COLUMN IF NOT EXISTS category TEXT;
ALTER TABLE products ADD COLUMN IF NOT EXISTS selection_type TEXT; -- configurationCards | baseProductWithToggles
ALTER TABLE products ADD COLUMN IF NOT EXISTS featured BOOLEAN NOT NULL DEFAULT FALSE;
ALTER TABLE products ADD COLUMN IF NOT EXISTS funding_alignment TEXT[] NOT NULL DEFAULT '{}';
ALTER TABLE products ADD COLUMN IF NOT EXISTS catalog JSONB NOT NULL DEFAULT '{}';

CREATE INDEX IF NOT EXISTS idx_products_category ON products(category);

-- An organization's selected product + configuration + add-ons from the
-- solutions step, with the computed price so later wizard steps (funding
-- roadmap, application package) can reuse it without recomputing.
CREATE TABLE IF NOT EXISTS org_product_selections (
    id               UUID PRIMARY KEY DEFAULT uuid_generate_v4(),
    org_id           UUID NOT NULL REFERENCES organizations(id) ON DELETE CASCADE,
    product_id       UUID NOT NULL REFERENCES products(id),
    configuration_id TEXT,
    selected_addons  TEXT[] NOT NULL DEFAULT '{}',
    quantity         INT NOT NULL DEFAULT 1,
    unit_price_cents BIGINT NOT NULL DEFAULT 0,
    subtotal_cents   BIGINT NOT NULL DEFAULT 0,
    created_at       TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    updated_at       TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    UNIQUE(org_id, product_id)
);

CREATE INDEX IF NOT EXISTS idx_org_product_selections_org_id ON org_product_selections(org_id);
