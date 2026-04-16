-- Seeds: api-product (product_db)
-- Run: psql -h localhost -U ecommerce -d product_db -f seeds/api-product.sql
--
-- Fixed UUIDs so other seeds (api-cart, api-order) can reference them:
--   Product A → 11111111-1111-1111-1111-000000000001
--   Product B → 11111111-1111-1111-1111-000000000002
--   Product C → 11111111-1111-1111-1111-000000000003
--   Product D → 11111111-1111-1111-1111-000000000004

-- Add discontinued column if the table was created before this column was added
ALTER TABLE products ADD COLUMN IF NOT EXISTS discontinued BOOLEAN NOT NULL DEFAULT FALSE;

INSERT INTO products (id, name, description, price, category, stock, discontinued)
VALUES
  ('11111111-1111-1111-1111-000000000001', 'Product A', 'I have stock',     200.00, 'prod1', 20, FALSE),
  ('11111111-1111-1111-1111-000000000002', 'Product B', 'I have no stock',  400.00, 'prod1',  0, FALSE),
  ('11111111-1111-1111-1111-000000000003', 'Product C', 'I am discontinue', 500.00, 'prod2',  1, TRUE),
  ('11111111-1111-1111-1111-000000000004', 'Product D', 'I am discontinue', 100.00, 'prod2',  1, TRUE)
ON CONFLICT DO NOTHING;
