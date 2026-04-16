# api-order

**Stack:** Go + Gin + Zap + PostgreSQL  
**Port:** 8084  
**Base URL:** `http://localhost:8084`

> Order creation checks stock availability via **api-product**.  
> Status transitions: `PENDING → CONFIRMED → SHIPPED → DELIVERED`  
> Cancellation allowed only from `PENDING` or `CONFIRMED`. Cancelling `CONFIRMED` logs `CONFLICT_ERROR`.

---

## Endpoints

### GET /health
Returns service status.

**Response 200**
```json
{ "status": "ok", "service": "api-order" }
```

---

### POST /orders
Create a new order. Validates stock for each item via **api-product** before inserting.

**Request body**
```json
{
  "customer_id": "22222222-2222-2222-2222-000000000001",
  "items": [
    {
      "product_id": "11111111-1111-1111-1111-000000000001",
      "name": "Product A",
      "price": 200.00,
      "quantity": 2
    }
  ]
}
```

**Response 201**
```json
{
  "id": "33333333-0001-0000-0000-000000000001",
  "customer_id": "22222222-2222-2222-2222-000000000001",
  "status": "PENDING",
  "items": [ ... ],
  "total_amount": 400.00,
  "created_at": "2026-01-01T00:00:00Z",
  "updated_at": "2026-01-01T00:00:00Z"
}
```

**Response 400** — validation error  
**Response 503** — stock service unavailable  
**Response 504** — database timeout

---

### GET /orders/:id
Get a single order by UUID.

**Response 200** — order object  
**Response 404**
```json
{ "error": "order not found", "order_id": "<id>" }
```

---

### GET /orders/customer/:customerId
List all orders for a customer.

**Response 200**
```json
{ "data": [ ...orders ], "count": 6 }
```

---

### PUT /orders/:id/status
Update order status manually.

**Request body**
```json
{ "status": "CONFIRMED" }
```

Valid values: `PENDING`, `CONFIRMED`, `SHIPPED`, `DELIVERED`, `CANCELLED`

**Response 200** — updated order  
**Response 404** — order not found

---

### DELETE /orders/:id
Cancel an order. Only allowed from `PENDING` or `CONFIRMED` status.

**Response 200** — cancelled order  
**Response 404** — order not found  
**Response 409**
```json
{
  "error": "order cannot be cancelled",
  "order_id": "<id>",
  "current_status": "SHIPPED"
}
```

---

## Status machine

```
PENDING → CONFIRMED → SHIPPED → DELIVERED
    ↓           ↓
 CANCELLED   CANCELLED  (logs CONFLICT_ERROR when cancelling CONFIRMED)
```

---

## Error responses

| Status | Meaning |
|---|---|
| 400 | Validation error |
| 404 | Order not found |
| 409 | Cannot cancel (wrong status) |
| 500 | Database error |
| 503 | Stock service unavailable |
| 504 | Database timeout |
