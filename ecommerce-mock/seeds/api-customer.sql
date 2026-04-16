-- Seeds: api-customer (customer_db)
-- Run: psql -h localhost -U ecommerce -d customer_db -f seeds/api-customer.sql
--
-- Requires pgcrypto for BCrypt hashing (compatible with BCrypt.Net used by api-customer)
-- Passwords: customer_1@mail.com_pass / customer_2@mail.com_pass
--
-- Fixed UUIDs so other seeds (api-cart, api-order) can reference them:
--   customer_1 → 22222222-2222-2222-2222-000000000001
--   customer_2 → 22222222-2222-2222-2222-000000000002

CREATE EXTENSION IF NOT EXISTS pgcrypto;

INSERT INTO customers (id, email, password_hash, first_name, last_name, created_at, updated_at)
VALUES
  (
    '22222222-2222-2222-2222-000000000001',
    'customer_1@mail.com',
    crypt('customer_1@mail.com_pass', gen_salt('bf', 10)),
    'customer',
    '1',
    NOW(), NOW()
  ),
  (
    '22222222-2222-2222-2222-000000000002',
    'customer_2@mail.com',
    crypt('customer_2@mail.com_pass', gen_salt('bf', 10)),
    'customer',
    '2',
    NOW(), NOW()
  )
ON CONFLICT DO NOTHING;
