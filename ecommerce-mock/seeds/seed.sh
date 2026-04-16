#!/usr/bin/env bash
# ─────────────────────────────────────────────────────────────────────────────
# seed.sh — clear all data and re-seed every service database
#
# Usage (from project root):
#   bash seeds/seed.sh
#
# Requires Docker containers to be running:
#   ecommerce-postgres, ecommerce-mysql, ecommerce-mongodb
# ─────────────────────────────────────────────────────────────────────────────

set -euo pipefail

SEEDS_DIR="$(cd "$(dirname "$0")" && pwd)"

RED='\033[0;31m'
GREEN='\033[0;32m'
YELLOW='\033[1;33m'
CYAN='\033[0;36m'
NC='\033[0m'

log_step()  { echo -e "${CYAN}▶ $1${NC}"; }
log_ok()    { echo -e "${GREEN}✓ $1${NC}"; }
log_warn()  { echo -e "${YELLOW}⚠ $1${NC}"; }
log_error() { echo -e "${RED}✗ $1${NC}"; }

# ─────────────────────────────────────────────
# Check containers are running
# ─────────────────────────────────────────────
check_container() {
  local name=$1
  if ! docker inspect --format='{{.State.Running}}' "$name" 2>/dev/null | grep -q true; then
    log_error "Container '$name' is not running. Start it with: docker compose up -d"
    exit 1
  fi
}

log_step "Checking containers..."
check_container ecommerce-postgres
check_container ecommerce-mysql
check_container ecommerce-mongodb
log_ok "All containers running"

# ─────────────────────────────────────────────
# PostgreSQL helpers
# ─────────────────────────────────────────────
pg_exec() {
  local db=$1
  local sql=$2
  docker exec ecommerce-postgres psql -U ecommerce -d "$db" -c "$sql" -q
}

pg_file() {
  local db=$1
  local file=$2
  docker exec -i ecommerce-postgres psql -U ecommerce -d "$db" -q < "$file"
}

# ─────────────────────────────────────────────
# CLEAR
# ─────────────────────────────────────────────
echo ""
log_step "Clearing all data..."

# product_db
pg_exec product_db  "TRUNCATE TABLE products RESTART IDENTITY CASCADE;"
log_ok "product_db cleared"

# customer_db
pg_exec customer_db "TRUNCATE TABLE revoked_tokens RESTART IDENTITY CASCADE;"
pg_exec customer_db "TRUNCATE TABLE customers RESTART IDENTITY CASCADE;"
log_ok "customer_db cleared"

# order_db
pg_exec order_db    "TRUNCATE TABLE orders RESTART IDENTITY CASCADE;"
log_ok "order_db cleared"

# payment_db (MySQL)
docker exec ecommerce-mysql mysql -u ecommerce -pecommerce_pass -e \
  "SET FOREIGN_KEY_CHECKS=0; TRUNCATE TABLE payment_db.payments; SET FOREIGN_KEY_CHECKS=1;" 2>/dev/null
log_ok "payment_db cleared"

# cart_db (MongoDB)
docker exec ecommerce-mongodb mongosh \
  "mongodb://ecommerce:ecommerce_pass@localhost:27017/cart_db?authSource=cart_db" \
  --quiet --eval "db.carts.deleteMany({})" > /dev/null
log_ok "cart_db cleared"

# ─────────────────────────────────────────────
# SEED
# ─────────────────────────────────────────────
echo ""
log_step "Seeding databases..."

# 1. api-product
pg_file product_db  "$SEEDS_DIR/api-product.sql"
log_ok "api-product seeded"

# 2. api-customer
pg_file customer_db "$SEEDS_DIR/api-customer.sql"
log_ok "api-customer seeded"

# 3. api-order
pg_file order_db    "$SEEDS_DIR/api-order.sql"
log_ok "api-order seeded"

# 4. api-payment (MySQL)
docker exec -i ecommerce-mysql mysql -u ecommerce -pecommerce_pass payment_db 2>/dev/null \
  < "$SEEDS_DIR/api-payment.sql"
log_ok "api-payment seeded"

# 5. api-cart (MongoDB)
docker exec -i ecommerce-mongodb mongosh \
  "mongodb://ecommerce:ecommerce_pass@localhost:27017/cart_db?authSource=cart_db" \
  --quiet < "$SEEDS_DIR/api-cart.js" > /dev/null
log_ok "api-cart seeded"

# ─────────────────────────────────────────────
# Summary
# ─────────────────────────────────────────────
echo ""
echo -e "${GREEN}─────────────────────────────────────────${NC}"
echo -e "${GREEN}  Seed complete.${NC}"
echo -e "${GREEN}─────────────────────────────────────────${NC}"
echo ""
echo "  product_db  (postgres) → 4 products"
echo "  customer_db (postgres) → 2 customers"
echo "  order_db    (postgres) → 16 orders"
echo "  payment_db  (mysql)    → 16 payments"
echo "  cart_db     (mongo)    → 3 carts"
echo ""
