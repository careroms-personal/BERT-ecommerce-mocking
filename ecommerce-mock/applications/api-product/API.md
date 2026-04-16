# api-product

**Stack:** Go + Gin + PostgreSQL  
**Port:** 8081  
**Base URL:** `http://localhost:8081`

---

## Endpoints

### GET /health
Returns service status.

**Response 200**
```json
{ "status": "ok", "service": "api-product" }
```

---

### GET /products
List products with pagination and optional category filter.

**Query params**
| Param | Type | Default | Description |
|---|---|---|---|
| `page` | int | 1 | Page number |
| `limit` | int | 20 | Items per page (max 100) |
| `category` | string | — | Filter by category |

**Response 200**
```json
{
  "data": [
    {
      "id": "11111111-1111-1111-1111-000000000001",
      "name": "Product A",
      "description": "I have stock",
      "price": 200.00,
      "category": "prod1",
      "stock": 20,
      "discontinued": false,
      "created_at": "2026-01-01T00:00:00Z",
      "updated_at": "2026-01-01T00:00:00Z"
    }
  ],
  "total": 4,
  "page": 1,
  "limit": 20
}
```

---

### GET /products/:id
Get a single product by UUID.

**Response 200** — product object  
**Response 404**
```json
{ "error": "product not found", "product_id": "<id>" }
```

---

### POST /products
Create a new product.

**Request body**
```json
{
  "name": "Product A",
  "description": "Some description",
  "price": 200.00,
  "category": "prod1",
  "stock": 20,
  "discontinued": false
}
```

| Field | Required | Notes |
|---|---|---|
| `name` | yes | |
| `price` | yes | must be > 0 |
| `stock` | no | default 0, must be >= 0 |
| `discontinued` | no | default false |

**Response 201** — created product object  
**Response 400** — validation error

---

### PUT /products/:id
Update a product (partial update — only provided fields are changed).

**Request body** (all fields optional)
```json
{
  "name": "New Name",
  "price": 250.00,
  "stock": 15,
  "discontinued": true
}
```

**Response 200** — updated product object  
**Response 404** — product not found

---

### GET /products/:id/stock
Get stock level for a product.

**Response 200**
```json
{ "product_id": "<id>", "stock": 20, "available": true }
```
```json
{ "product_id": "<id>", "stock": 0, "available": false }
```

**Response 404** — product not found

---

## Error responses

| Status | Meaning |
|---|---|
| 400 | Bad request / validation error |
| 404 | Product not found |
| 500 | Database error |
