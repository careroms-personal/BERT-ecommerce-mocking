# Database Schema Summary

## api-product — PostgreSQL (`product_db`)

**Table: `products`**
| Column | Type | Notes |
|---|---|---|
| `id` | UUID PK | `gen_random_uuid()` |
| `name` | VARCHAR(255) | NOT NULL |
| `description` | TEXT | default `''` |
| `price` | NUMERIC(10,2) | NOT NULL |
| `category` | VARCHAR(100) | default `''` |
| `stock` | INTEGER | default `0` |
| `discontinued` | BOOLEAN | default `FALSE` |
| `created_at` | TIMESTAMPTZ | default `NOW()` |
| `updated_at` | TIMESTAMPTZ | default `NOW()` |

Indexes: `category`, `stock`

---

## api-customer — PostgreSQL (`customer_db`)

**Table: `customers`**
| Column | Type | Notes |
|---|---|---|
| `id` | UUID PK | `Guid.NewGuid()` |
| `email` | VARCHAR(255) | NOT NULL |
| `password_hash` | VARCHAR(255) | NOT NULL |
| `first_name` | VARCHAR(100) | NOT NULL |
| `last_name` | VARCHAR(100) | NOT NULL |
| `created_at` | TIMESTAMP | default `UtcNow` |
| `updated_at` | TIMESTAMP | default `UtcNow` |

**Table: `revoked_tokens`**
| Column | Type | Notes |
|---|---|---|
| `jti` | UUID PK | JWT ID |
| `customer_id` | UUID | |
| `revoked_at` | TIMESTAMP | default `UtcNow` |
| `expires_at` | TIMESTAMP | |

---

## api-cart — MongoDB (`cart_db`)

**Collection: `carts`**
| Field | Type | Notes |
|---|---|---|
| `_id` | ObjectId | auto |
| `userId` | String | unique index |
| `items` | Array | embedded `CartItem` |
| `createdAt` | Date | immutable |
| `updatedAt` | Date | updated on save, TTL index (86400s) |

**Embedded: `CartItem`**
| Field | Type | Notes |
|---|---|---|
| `itemId` | String | |
| `productId` | String | |
| `name` | String | |
| `price` | Number | |
| `quantity` | Number | min 1 |
| `addedAt` | Date | default `now` |

---

## api-order — PostgreSQL (`order_db`)

**Table: `orders`**
| Column | Type | Notes |
|---|---|---|
| `id` | UUID PK | `gen_random_uuid()` |
| `customer_id` | UUID | NOT NULL |
| `status` | VARCHAR(20) | `PENDING` / `CONFIRMED` / `SHIPPED` / `DELIVERED` / `CANCELLED` |
| `items` | JSONB | array of `{product_id, name, price, quantity}` |
| `total_amount` | NUMERIC(10,2) | NOT NULL |
| `created_at` | TIMESTAMPTZ | default `NOW()` |
| `updated_at` | TIMESTAMPTZ | default `NOW()` |

Indexes: `customer_id`, `status`

---

## api-payment — MySQL (`payment_db`)

**Table: `payments`**
| Column | Type | Notes |
|---|---|---|
| `id` | UUID PK | `Guid.NewGuid()` |
| `order_id` | UUID | NOT NULL |
| `customer_id` | UUID | NOT NULL |
| `amount` | DECIMAL | NOT NULL |
| `status` | VARCHAR(20) | `PENDING` / `PROCESSING` / `SUCCESS` / `FAILED` / `REFUNDED` |
| `provider_ref` | VARCHAR(100) | nullable |
| `failure_reason` | VARCHAR(100) | nullable |
| `retry_count` | INT | default `0` |
| `created_at` | TIMESTAMP | default `UtcNow` |
| `updated_at` | TIMESTAMP | default `UtcNow` |
