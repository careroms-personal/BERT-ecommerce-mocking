# E-Commerce Mock Microservice Server — Project Summary
> Goal: Generate realistic, labeled log data for ML/LLM training (log categorization)

---

## 🎯 Project Goal

Build a **mock e-commerce microservice system** that:
- Runs real HTTP servers per service
- Fires real, authentic logs in each language's native format
- Can be hammered infinitely with a load generator
- Produces labeled log data for fine-tuning **DistilBERT / DeBERTa** for log categorization

---

## 🏗️ Architecture Overview

```
api-gateway
    ├── api-product       → PostgreSQL (product db)
    ├── api-cart          → MongoDB (cart db)
    ├── api-customer      → PostgreSQL (customer db)
    ├── api-order         → PostgreSQL (order db)
    ├── api-payment       → MySQL (payment db)
    └── api-search        → Elasticsearch
                                ↑
                        search-data-transfer
                        (syncs from product + order dbs)
```

---

## 🛠️ Technology Stack

| Service               | Language  | Logger    | Log Style              | Database      |
|-----------------------|-----------|-----------|------------------------|---------------|
| api-gateway           | Go        | Zap       | JSON structured        | ❌ No DB      |
| api-product           | Go        | Zap       | JSON structured        | PostgreSQL    |
| api-cart              | Node.js (TypeScript) | Pino | JSON + colorized  | MongoDB       |
| api-customer          | .NET      | Serilog   | JSON + .NET verbose    | PostgreSQL    |
| api-order             | Go        | Zap       | JSON structured        | PostgreSQL    |
| api-payment           | .NET      | Serilog   | .NET enterprise style  | MySQL         |
| api-search            | Rust      | tracing   | Rust structured        | Elasticsearch |
| search-data-transfer  | Rust      | tracing   | Rust structured        | ❌ No DB      |

> Mixed language + DB stack is intentional — mimics real-world microservice teams and makes ML model more robust/generalizable

### DB Selection Rationale
- **api-gateway** — stateless, routing only, no DB needed
- **api-product** — structured catalog, complex queries, stock transactions need ACID → PostgreSQL
- **api-cart** — flexible schema, cart items vary per user, MongoDB TTL index handles cart expiry natively → MongoDB
- **api-customer** — auth/identity, JWT blacklist, strict relational integrity → PostgreSQL
- **api-order** — ACID critical, order state machine, money related → PostgreSQL
- **api-payment** — fintech/enterprise MySQL tradition, battle tested for financial transactions → MySQL

---

## 📡 Service Endpoints

### api-gateway (Go)
- `GET /health`
- `POST /auth/login` → routes to api-customer
- `GET /products` → routes to api-product
- `GET /products/:id` → routes to api-product
- `GET /search?q=` → routes to api-search
- `POST /cart` → routes to api-cart
- `GET /cart/:userId` → routes to api-cart
- `POST /orders` → routes to api-order
- `GET /orders/:id` → routes to api-order
- `POST /payments` → routes to api-payment

### api-product (Go)
- `GET /products` — list with pagination
- `GET /products/:id` — product detail
- `POST /products` — create product
- `PUT /products/:id` — update product
- `GET /products/:id/stock` — stock check

### api-cart (Node.js)
- `GET /cart/:userId` — get cart
- `POST /cart/:userId/items` — add item
- `DELETE /cart/:userId/items/:itemId` — remove item
- `DELETE /cart/:userId` — clear cart
- `POST /cart/:userId/sync` — sync with product stock

### api-customer (.NET)
- `POST /auth/login` — login, returns JWT
- `POST /auth/logout` — logout, revoke token
- `POST /auth/refresh` — refresh JWT
- `POST /customers/register` — register
- `GET /customers/:id` — profile
- `PUT /customers/:id` — update profile

### api-order (Go)
- `POST /orders` — create order (from cart)
- `GET /orders/:id` — order detail
- `GET /orders/customer/:customerId` — order history
- `PUT /orders/:id/status` — update status
- `DELETE /orders/:id` — cancel order

### api-payment (.NET)
- `POST /payments` — process payment
- `GET /payments/:id` — payment status
- `POST /payments/:id/retry` — retry failed payment
- `POST /payments/webhook` — payment provider callback

### api-search (Rust)
- `GET /search?q=&category=&page=` — full text search
- `GET /search/suggest?q=` — autocomplete
- `POST /search/index` — manual re-index trigger

### search-data-transfer (Rust)
- Polls product db + order db on interval
- Syncs changes to Elasticsearch index
- Logs sync events, failures, record counts

---

## 🔄 User Journey Flow (Log Story)

```
1. REGISTER / LOGIN
   api-gateway → api-customer → JWT issued

2. BROWSE PRODUCTS
   api-gateway → api-product (list, paginate)

3. SEARCH
   api-gateway → api-search → Elasticsearch query

4. VIEW PRODUCT DETAIL
   api-gateway → api-product/:id + stock check

5. ADD TO CART
   api-gateway → api-cart → api-product (stock validate)

6. PLACE ORDER
   api-gateway → api-order → api-cart (read) → api-product (stock deduct)

7. PAYMENT
   api-gateway → api-payment → (mock payment provider)
              → success: api-order status update
              → fail: retry logic, eventually fail

8. ORDER CONFIRMATION
   api-order status = CONFIRMED
```

---

## 💥 Chaos / Error Scenarios (per service)

| Service      | Scenario                                      | Log Category        |
|--------------|-----------------------------------------------|---------------------|
| api-gateway  | Rate limit exceeded                           | `RATE_LIMIT`        |
| api-gateway  | Unknown route                                 | `ROUTING_ERROR`     |
| api-customer | Wrong password (multiple attempts)            | `AUTH_FAIL`         |
| api-customer | Token expired / invalid                       | `AUTH_EXPIRED`      |
| api-product  | Product not found                             | `NOT_FOUND`         |
| api-product  | Out of stock                                  | `STOCK_ERROR`       |
| api-cart     | Cart expired (session timeout)                | `CART_EXPIRED`      |
| api-cart     | Stock mismatch during sync                    | `SYNC_ERROR`        |
| api-order    | Order creation fail (db timeout)              | `DB_TIMEOUT`        |
| api-order    | Order cancel after payment                    | `CONFLICT_ERROR`    |
| api-payment  | Payment declined                              | `PAYMENT_FAIL`      |
| api-payment  | Payment timeout (provider slow)               | `PAYMENT_TIMEOUT`   |
| api-payment  | Retry exceeded max attempts                   | `RETRY_EXHAUSTED`   |
| api-search   | Elasticsearch down                            | `SEARCH_UNAVAILABLE`|
| search-transfer | DB connection lost during sync             | `SYNC_FAIL`         |

---

## 🏷️ Log Categories (ML Training Labels)

```
AUTH          - login, logout, register, token ops
AUTH_FAIL     - failed auth attempts
ROUTING       - gateway routing events
RATE_LIMIT    - rate limit hits
PRODUCT       - product fetch, list, update
STOCK         - stock check, deduction, out-of-stock
CART          - cart add, remove, clear, sync
ORDER         - order create, update, cancel, confirm
PAYMENT       - payment process, success, fail, retry
SEARCH        - search query, suggest, index
SYNC          - data transfer sync events
DB_ERROR      - database timeout, connection error
NOT_FOUND     - 404 resource not found
SYSTEM        - health check, startup, shutdown
ERROR         - unhandled / unexpected errors
```

---

## 📊 Load Generator

- Separate script/service that fires requests infinitely
- Configurable RPS (requests per second)
- Simulates multiple concurrent users (sessions)
- Randomizes happy path vs chaos scenarios
- Can run as Docker container alongside services

---

## 🐳 Infrastructure

- Each service runs as a **Docker container**
- `docker-compose.yml` orchestrates all services
- Shared network for inter-service communication
- **PostgreSQL** — api-product, api-customer, api-order
- **MongoDB** — api-cart (with TTL index for cart expiry)
- **MySQL** — api-payment
- Single Elasticsearch instance for api-search

### 📦 Log Collection — Fluent Bit

```
Each service stdout
      ↓
Fluent Bit (Docker log driver)
      ↓
Local volume ./logs/
      ↓
JSONL file per service
```

- Lightweight single container (~450KB)
- No code changes needed in any service
- Auto-collects from all containers via Docker logging driver
- Outputs one JSONL file per service — ready for ML training pipeline

```
logs/
├── api-gateway.jsonl
├── api-product.jsonl
├── api-cart.jsonl
├── api-customer.jsonl
├── api-order.jsonl
├── api-payment.jsonl
├── api-search.jsonl
└── search-data-transfer.jsonl
```

---

## 🤖 ML Training Target

| Item | Detail |
|---|---|
| Model | DistilBERT (start) → DeBERTa-v3-small (upgrade) |
| Task | Multi-class log categorization |
| Training data | 300K–500K labeled log lines |
| Hardware | ASUS Zephyrus G16, RTX 5070 Ti 12GB VRAM, 32GB RAM |
| Framework | HuggingFace Transformers + PyTorch |
| Serving | FastAPI inference API |
| Expected epochs | 3–5 epochs |
| Estimated train time | 3–10 hours (DistilBERT, 500K samples) |

---

## 📁 Suggested Project Structure

```
log-forge/
├── docker-compose.yml
├── fluent-bit.conf
├── logs/                         ← JSONL output per service
├── load-generator/
│   └── main.go
├── api-gateway/                  (Go)
├── api-product/                  (Go + PostgreSQL)
├── api-cart/                     (Node.js + MongoDB)
├── api-customer/                 (.NET + PostgreSQL)
├── api-order/                    (Go + PostgreSQL)
├── api-payment/                  (.NET + MySQL)
├── api-search/                   (Rust + Elasticsearch)
├── search-data-transfer/         (Rust)
└── ml-training/
    ├── preprocess.py
    ├── train.py
    └── serve.py                  (FastAPI)
```

---

## ✅ PoC Success Criteria

- [ ] All 8 services running via docker-compose
- [ ] Load generator fires requests continuously
- [ ] Each service produces authentic language-native logs
- [ ] Logs are labeled by category
- [ ] 300K+ log samples collected
- [ ] DistilBERT fine-tuned and achieving >90% categorization accuracy
- [ ] FastAPI inference endpoint working
