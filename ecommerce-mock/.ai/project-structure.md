# Project Structure

Mock e-commerce system with 6 microservices, each using a different stack.
All services run in Docker Compose and log to Fluent Bit via the fluentd driver.

---

## Root Layout

```
ecommerce-mock/
├── applications/          # One folder per microservice
│   ├── api-product/       # Go + Gin + PostgreSQL          :8081
│   ├── api-customer/      # .NET 9 + Serilog + PostgreSQL  :8082
│   ├── api-cart/          # Node.js + TypeScript + Fastify + MongoDB  :8083
│   ├── api-order/         # Go + Gin + PostgreSQL          :8084
│   ├── api-payment/       # .NET 9 + Serilog + MySQL       :8085
│   └── api-search/        # Rust + axum + Elasticsearch    :8086
├── infra/
│   ├── fluent-bit/        # fluent-bit.conf — collects stdout from all containers
│   ├── postgres/          # init.sql — creates databases: product_db, customer_db, order_db
│   ├── mongo/             # init.js  — creates cart_db
│   └── mysql/             # init.sql — creates payment_db
├── seeds/                 # Seed data scripts (run after containers are up)
│   ├── seed.sh            # Master bash script: truncate + seed all services in order
│   ├── api-product.sql    # 4 products (Product A–D)
│   ├── api-customer.sql   # 2 customers (customer_1, customer_2)
│   ├── api-order.sql      # 16 orders across statuses and customers
│   ├── api-payment.sql    # 16 payments linked to orders
│   └── api-cart.js        # 3 carts (customer_1, customer_2, ghost)
├── call-api/              # Python test scripts (stdlib only, no dependencies)
│   ├── get/2xx/run.py     # Calls all normal GET endpoints; asserts 2xx
│   └── get/5xx/run.py     # Calls all sim GET endpoints; asserts 5xx
├── db-schema-summary.md   # All table/collection schemas in one file
├── docker-compose.yml     # Full stack definition
└── .ai/                   # AI context docs (this folder)
    ├── project-structure.md
    └── api-structure.md
```

---

## Service Details

| Service | Stack | Port | Database | DB Name |
|---|---|---|---|---|
| api-product | Go + Gin + pgx/v5 | 8081 | PostgreSQL | product_db |
| api-customer | .NET 9 + EF Core + Npgsql | 8082 | PostgreSQL | customer_db |
| api-cart | Node.js + TypeScript + Fastify + Mongoose | 8083 | MongoDB | cart_db |
| api-order | Go + Gin + pgx/v5 | 8084 | PostgreSQL | order_db |
| api-payment | .NET 9 + EF Core + Pomelo (MySQL) | 8085 | MySQL | payment_db |
| api-search | Rust + axum + reqwest | 8086 | Elasticsearch | index: products |

---

## Infrastructure

| Component | Image | Purpose |
|---|---|---|
| PostgreSQL | postgres:16-alpine | Shared instance, 3 databases |
| MongoDB | mongo:7 | cart_db |
| MySQL | mysql:8 | payment_db |
| Elasticsearch | elasticsearch:8.13.0 | products index for api-search |
| Fluent Bit | fluent/fluent-bit:3.2 | Log aggregation — receives from all containers via fluentd driver |

All containers use `logging: driver: fluentd` with `fluentd-async: "true"` so apps are never blocked if Fluent Bit is slow.

---

## Application Structure per Service

### Go services (api-product, api-order)
```
main.go
config/config.go
handler/
  middleware.go      # RequestLogger + request_id injection
  <entity>.go        # Normal CRUD handlers
  <entity>-5x.go     # Sim handlers (intentional 5xx triggers)
logger/logger.go     # zap logger setup
model/<entity>.go    # Structs + request/response types
repository/<entity>.go  # pgxpool queries + migrations
```

### .NET services (api-customer, api-payment)
```
Program.cs           # Host setup, Serilog, EF migration, route mapping
Controllers/
  <Entity>Controller.cs
  AuthController.cs  # (api-customer only)
  SimController.cs   # Sim endpoints (intentional 5xx triggers)
Data/AppDbContext.cs
Models/
Services/
```

### Node.js service (api-cart)
```
src/
  index.ts           # Fastify setup, DB connect, server start
  config.ts
  db.ts              # Mongoose connect
  logger.ts          # Pino logger options
  models/cart.ts     # Mongoose schema + model
  routes/
    cart.ts          # Normal cart routes
    cart-5x.ts       # Sim routes (intentional 5xx triggers)
```

### Rust service (api-search)
```
src/main.rs          # axum server, ES ping loop, all routes inline
Cargo.toml
```

---

## Seed UUIDs (fixed across all services)

| Entity | UUID |
|---|---|
| Product A | `11111111-1111-1111-1111-000000000001` |
| Product B | `11111111-1111-1111-1111-000000000002` |
| Product C (discontinued) | `11111111-1111-1111-1111-000000000003` |
| Product D (discontinued) | `11111111-1111-1111-1111-000000000004` |
| Customer 1 | `22222222-2222-2222-2222-000000000001` |
| Customer 2 | `22222222-2222-2222-2222-000000000002` |
| Ghost customer | `99999999-9999-9999-9999-999999999999` |
| Order 1 (PENDING, Customer 1) | `33333333-0001-0000-0000-000000000001` |
| Payment 1 (for Order 1) | `44444444-0001-0000-0000-000000000001` |

---

## Startup Resilience

All services are designed to start regardless of dependency availability:
- **Go**: `pgxpool` uses `LazyConnect = true` — pool creation never dials; real connection on first query
- **.NET**: `EnsureCreated()` wrapped in `try/catch` — logs error and continues
- **Node.js**: MongoDB connect failure is caught and logged — server starts anyway
- **Rust**: After 10 failed ES pings (3s apart), logs error and starts serving anyway

No service has `depends_on` on Fluent Bit — observability must not block business services.
