-- Seeds: api-payment (payment_db)
-- Run: docker exec -i ecommerce-mysql mysql -u ecommerce -pecommerce_pass payment_db < seeds/api-payment.sql
--
-- Order UUIDs match seeds/api-order.sql
-- Customer UUIDs match seeds/api-customer.sql
--
-- Status distribution:
--   SUCCESS  → CONFIRMED / SHIPPED orders (payment passed)
--   PENDING  → PENDING orders (not yet processed)
--   FAILED   → CANCELLED orders where payment caused the cancellation
--   REFUNDED → CANCELLED orders where payment succeeded but order was later cancelled
--
-- retry_count > 0 + SUCCESS → succeeded after retries  (RETRY log)
-- retry_count = 3 + FAILED  → retry exhausted          (RETRY_EXHAUSTED log)

INSERT INTO payments (id, order_id, customer_id, amount, status, provider_ref, failure_reason, retry_count, created_at, updated_at)
VALUES

-- ── customer_1 — PENDING order → PENDING payment ─────────────────────────────
('44444444-0001-0000-0000-000000000001','33333333-0001-0000-0000-000000000001','22222222-2222-2222-2222-000000000001', 400.00,'PENDING',   NULL,               NULL,               0, NOW(), NOW()),

-- ── customer_1 — CONFIRMED orders → SUCCESS payments ─────────────────────────
('44444444-0001-0000-0000-000000000002','33333333-0001-0000-0000-000000000002','22222222-2222-2222-2222-000000000001', 600.00,'SUCCESS',  'PRV-10001',         NULL,               0, NOW(), NOW()),
('44444444-0001-0000-0000-000000000003','33333333-0001-0000-0000-000000000003','22222222-2222-2222-2222-000000000001', 800.00,'SUCCESS',  'PRV-10002',         NULL,               0, NOW(), NOW()),
-- succeeded after 1 retry
('44444444-0001-0000-0000-000000000004','33333333-0001-0000-0000-000000000004','22222222-2222-2222-2222-000000000001', 600.00,'SUCCESS',  'PRV-10003',         NULL,               1, NOW(), NOW()),

-- ── customer_1 — SHIPPED orders → SUCCESS payments ───────────────────────────
('44444444-0001-0000-0000-000000000005','33333333-0001-0000-0000-000000000005','22222222-2222-2222-2222-000000000001', 200.00,'SUCCESS',  'PRV-10004',         NULL,               0, NOW(), NOW()),
('44444444-0001-0000-0000-000000000006','33333333-0001-0000-0000-000000000006','22222222-2222-2222-2222-000000000001', 600.00,'SUCCESS',  'PRV-10005',         NULL,               0, NOW(), NOW()),

-- ── customer_2 — CANCELLED orders → FAILED payments ──────────────────────────
-- retry exhausted: card_declined
('44444444-0002-0000-0000-000000000001','33333333-0002-0000-0000-000000000001','22222222-2222-2222-2222-000000000002', 200.00,'FAILED',    NULL,               'card_declined',    3, NOW(), NOW()),
-- retry exhausted: insufficient_funds
('44444444-0002-0000-0000-000000000002','33333333-0002-0000-0000-000000000002','22222222-2222-2222-2222-000000000002', 500.00,'FAILED',    NULL,               'insufficient_funds',3,NOW(), NOW()),
-- failed after 1 retry: card_declined
('44444444-0002-0000-0000-000000000003','33333333-0002-0000-0000-000000000003','22222222-2222-2222-2222-000000000002', 800.00,'FAILED',    NULL,               'card_declined',    1, NOW(), NOW()),

-- ── customer_2 — SHIPPED orders → SUCCESS payments ───────────────────────────
('44444444-0002-0000-0000-000000000004','33333333-0002-0000-0000-000000000004','22222222-2222-2222-2222-000000000002', 400.00,'SUCCESS',  'PRV-20001',         NULL,               0, NOW(), NOW()),
('44444444-0002-0000-0000-000000000005','33333333-0002-0000-0000-000000000005','22222222-2222-2222-2222-000000000002', 400.00,'SUCCESS',  'PRV-20002',         NULL,               0, NOW(), NOW()),
-- succeeded after 2 retries
('44444444-0002-0000-0000-000000000006','33333333-0002-0000-0000-000000000006','22222222-2222-2222-2222-000000000002', 300.00,'SUCCESS',  'PRV-20003',         NULL,               2, NOW(), NOW()),
('44444444-0002-0000-0000-000000000007','33333333-0002-0000-0000-000000000007','22222222-2222-2222-2222-000000000002',1000.00,'SUCCESS',  'PRV-20004',         NULL,               0, NOW(), NOW()),

-- ── customer_2 — CANCELLED order → REFUNDED (paid then manually cancelled) ───
('44444444-0002-0000-0000-000000000008','33333333-0002-0000-0000-000000000008','22222222-2222-2222-2222-000000000002', 300.00,'REFUNDED', 'PRV-20005',         NULL,               0, NOW(), NOW()),

-- ── ghost_1 — PENDING order → PENDING payment ────────────────────────────────
('44444444-0088-0000-0000-000000000001','33333333-0088-0000-0000-000000000001','88888888-8888-8888-8888-000000000001', 200.00,'PENDING',   NULL,               NULL,               0, NOW(), NOW()),

-- ── ghost_2 — CANCELLED order → FAILED, retry exhausted: provider_timeout ────
('44444444-0088-0000-0000-000000000002','33333333-0088-0000-0000-000000000002','88888888-8888-8888-8888-000000000002', 400.00,'FAILED',    NULL,               'provider_timeout', 3, NOW(), NOW());
