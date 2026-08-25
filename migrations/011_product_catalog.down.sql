DROP TABLE IF EXISTS org_product_selections;
DROP INDEX IF EXISTS idx_products_category;
ALTER TABLE products DROP COLUMN IF EXISTS catalog;
ALTER TABLE products DROP COLUMN IF EXISTS funding_alignment;
ALTER TABLE products DROP COLUMN IF EXISTS featured;
ALTER TABLE products DROP COLUMN IF EXISTS selection_type;
ALTER TABLE products DROP COLUMN IF EXISTS category;
