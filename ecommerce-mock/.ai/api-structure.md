# API Structure

All endpoints return JSON. All services log structured JSON via Serilog / zap / Pino.
Each log entry includes a `category` field for filtering (e.g. `DB_ERROR`, `NOT_FOUND`, `SYSTEM`).

Sim endpoints (`/sim/` or `/*/sim/`) intentionally trigger 5xx by running bad DB queries.
They exist for testing observability pipelines, not for production use.

---

## api-product — port 8081

| Method | Path | Description | Success |
|---|---|---|---|
| GET | `/health` | Service status | 200 |
| GET | `/products` | List products (`page`, `limit`, `category`) | 200 |
| GET | `/products/:id` | Get product by UUID | 200 / 404 |
| POST | `/products` | Create product | 201 / 400 |
| PUT | `/products/:id` | Partial update product | 200 / 404 |
| GET | `/products/:id/stock` | Stock level + availability | 200 / 404 |
| GET | `/products/sim/bad-column` | **SIM** — runs bad SQL → 500 | — |

Product fields: `id`, `name`, `description`, `price`, `category`, `stock`, `discontinued`, `created_at`, `updated_at`

---

## api-customer — port 8082

| Method | Path | Description | Success |
|---|---|---|---|
| GET | `/health` | Service status | 200 |
| POST | `/customers/register` | Create account (BCrypt password) | 201 / 409 |
| GET | `/customers/:id` | Get customer profile | 200 / 404 |
| PUT | `/customers/:id` | Partial update profile | 200 / 404 / 409 |
| POST | `/auth/login` | Login → JWT access + refresh tokens | 200 / 401 |
| POST | `/auth/logout` | Revoke access token | 200 / 401 |
| POST | `/auth/refresh` | Exchange refresh token for new pair | 200 / 401 |
| GET | `/sim/bad-column` | **SIM** — runs bad SQL → 500 | — |

Auth uses JWT (Bearer). Revoked tokens stored in `revoked_tokens` table.

---

## api-cart — port 8083

| Method | Path | Description | Success |
|---|---|---|---|
| GET | `/health` | Service status | 200 |
| GET | `/cart/:userId` | Get cart (410 if expired >24h) | 200 / 404 / 410 |
| POST | `/cart/:userId/items` | Add item (checks stock via api-product) | 201 / 404 / 409 |
| DELETE | `/cart/:userId/items/:itemId` | Remove specific item | 200 / 404 |
| DELETE | `/cart/:userId` | Clear entire cart | 200 |
| POST | `/cart/:userId/sync` | Sync quantities against current stock | 200 / 404 |
| GET | `/cart/sim/bad-query` | **SIM** — invalid MongoDB aggregation → 500 | — |

Cart expires after 24h of inactivity (MongoDB TTL index on `updatedAt`).

---

## api-order — port 8084

| Method | Path | Description | Success |
|---|---|---|---|
| GET | `/health` | Service status | 200 |
| POST | `/orders` | Create order (validates stock via api-product) | 201 / 400 / 503 |
| GET | `/orders/:id` | Get order by UUID | 200 / 404 |
| GET | `/orders/customer/:customerId` | List orders for customer | 200 |
| PUT | `/orders/:id/status` | Update order status | 200 / 404 |
| DELETE | `/orders/:id` | Cancel order (PENDING or CONFIRMED only) | 200 / 404 / 409 |
| GET | `/orders/sim/bad-column` | **SIM** — runs bad SQL → 500 | — |

Status machine: `PENDING → CONFIRMED → SHIPPED → DELIVERED`
Cancellation from `CONFIRMED` logs `CONFLICT_ERROR`.

---

## api-payment — port 8085

| Method | Path | Description | Success |
|---|---|---|---|
| GET | `/health` | Service status | 200 |
| POST | `/payments` | Process payment (MockPaymentProvider) | 200 / 402 / 500 |
| GET | `/payments/:id` | Get payment by UUID | 200 / 404 |
| POST | `/payments/:id/retry` | Retry failed payment | 200 / 400 / 402 / 404 / 429 |
| POST | `/payments/webhook` | External provider callback | 200 / 404 |
| GET | `/sim/bad-column` | **SIM** — runs bad SQL → 500 | — |

MockPaymentProvider rates (configurable via env): 75% success · 15% decline · 10% timeout.
Max retries: 3 (configurable via `Payment__MaxRetries`).
Status flow: `PROCESSING → SUCCESS` or `PROCESSING → FAILED → retry → ...`

---

## api-search — port 8086

| Method | Path | Description | Success |
|---|---|---|---|
| GET | `/health` | Service + ES status | 200 / 503 |
| GET | `/search?q=<term>` | Full-text search (`category`, `page` optional) | 200 / 400 / 503 |
| GET | `/search/suggest?q=<prefix>` | Autocomplete (min 2 chars) | 200 / 400 / 503 |
| POST | `/search/index` | Index a product document | 200 / 503 |

Returns 503 on all endpoints if Elasticsearch is unreachable.
No sim endpoint (Rust service excluded by design).

---

## Cross-service Dependencies

```
api-cart   ──stock check──▶  api-product
api-order  ──stock check──▶  api-product
api-search ──index sync──▶   (called by api-product or external transfer service)
```

---

## Common Log Categories

| Category | Meaning |
|---|---|
| `SYSTEM` | Startup, shutdown, health |
| `DB_ERROR` | Database query or connection failure |
| `NOT_FOUND` | 404 — entity does not exist |
| `ERROR` | General 5xx error |
| `AUTH` | Login, logout, token ops |
| `CART` | Cart read/write operations |
| `STOCK` | Stock check or deduction |
| `PAYMENT` | Payment processing |
| `PAYMENT_FAIL` | Payment declined (402) |
| `RETRY_EXHAUSTED` | Max retries hit (429) |
| `SYNC_ERROR` | Cart sync mismatch |
| `CONFLICT_ERROR` | Order cancel from wrong state |
| `SIM` | Simulation endpoint triggered |

---

## call-api Scripts

| Script | What it does | Pass condition |
|---|---|---|
| `call-api/get/2xx/run.py` | Calls all normal GET endpoints (24 calls) | All return 2xx |
| `call-api/get/5xx/run.py` | Calls all sim GET endpoints (5 calls) | All return 5xx |

Both scripts: stdlib only, `--base-url` flag, colored output, exit 1 on failure.
