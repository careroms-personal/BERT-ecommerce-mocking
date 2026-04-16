# api-cart

**Stack:** Node.js + TypeScript + Fastify + Pino + MongoDB  
**Port:** 8083  
**Base URL:** `http://localhost:8083`

> Cart items are validated against **api-product** stock before being added.  
> Carts expire after **24 hours** of inactivity (TTL index on `updatedAt`).

---

## Endpoints

### GET /health
Returns service status.

**Response 200**
```json
{ "status": "ok", "service": "api-cart" }
```

---

### GET /cart/:userId
Get a user's cart.

**Response 200**
```json
{
  "userId": "22222222-2222-2222-2222-000000000001",
  "items": [
    {
      "itemId": "uuid",
      "productId": "11111111-1111-1111-1111-000000000001",
      "name": "Product A",
      "price": 200.00,
      "quantity": 2,
      "addedAt": "2026-01-01T00:00:00Z"
    }
  ],
  "createdAt": "2026-01-01T00:00:00Z",
  "updatedAt": "2026-01-01T00:00:00Z"
}
```

**Response 404** — cart not found  
**Response 410** — cart expired (deleted automatically)

---

### POST /cart/:userId/items
Add an item to the cart. Checks stock via **api-product** first.  
If the product is already in the cart, quantity is incremented.

**Request body**
```json
{
  "productId": "11111111-1111-1111-1111-000000000001",
  "name": "Product A",
  "price": 200.00,
  "quantity": 1
}
```

| Field | Required | Notes |
|---|---|---|
| `productId` | yes | |
| `name` | yes | |
| `price` | yes | |
| `quantity` | no | default 1 |

**Response 201** — updated cart  
**Response 404** — product not found in api-product  
**Response 409** — product out of stock or insufficient stock  
**Response 400** — missing required fields

---

### DELETE /cart/:userId/items/:itemId
Remove a specific item from the cart by `itemId`.

**Response 200** — updated cart  
**Response 404** — cart not found or item not found

---

### DELETE /cart/:userId
Clear the entire cart (delete all items).

**Response 200**
```json
{ "message": "cart cleared", "user_id": "<userId>" }
```

---

### POST /cart/:userId/sync
Sync cart against current stock levels from **api-product**.  
Removes out-of-stock items and adjusts quantities if stock is lower than requested.

**Response 200**
```json
{
  "synced": true,
  "mismatches": [
    { "itemId": "...", "productId": "...", "reason": "out_of_stock" },
    { "itemId": "...", "productId": "...", "reason": "quantity_adjusted", "from": 3, "to": 1 },
    { "itemId": "...", "productId": "...", "reason": "product_not_found" }
  ],
  "cart": { ... }
}
```

**Response 404** — cart not found

---

## Error responses

| Status | Meaning |
|---|---|
| 400 | Missing required fields |
| 404 | Cart or item or product not found |
| 409 | Out of stock / insufficient stock |
| 410 | Cart expired |
| 500 | Database error |
