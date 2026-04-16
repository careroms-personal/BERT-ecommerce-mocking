# api-search

**Stack:** Rust + axum + tracing + Elasticsearch  
**Port:** 8086  
**Base URL:** `http://localhost:8086`

> Backed by Elasticsearch index `products`.  
> On startup, waits up to 10 attempts (3s apart) for ES to be reachable, then starts regardless.  
> All search/suggest/index endpoints return `503` if ES is down.

---

## Endpoints

### GET /health
Returns service and Elasticsearch status.

**Response 200** — ES is up
```json
{ "status": "ok", "service": "api-search", "elasticsearch": "up" }
```

**Response 503** — ES is unreachable
```json
{ "status": "degraded", "elasticsearch": "down" }
```

---

### GET /search
Full-text search across the products index.

**Query params**
| Param | Type | Required | Description |
|---|---|---|---|
| `q` | string | yes | Search query |
| `category` | string | no | Filter by category |
| `page` | int | no | Page number (default 1) |

Page size is fixed at **10** results.

**Response 200**
```json
{
  "data": [
    {
      "id": "11111111-1111-1111-1111-000000000001",
      "name": "Product A",
      "description": "I have stock",
      "price": 200.00,
      "category": "prod1"
    }
  ],
  "total": 2,
  "page": 1,
  "took_ms": 4
}
```

**Response 400** — missing `q` param  
**Response 503** — Elasticsearch unavailable

---

### GET /search/suggest
Prefix autocomplete suggestions. Minimum 2 characters required.

**Query params**
| Param | Type | Required | Description |
|---|---|---|---|
| `q` | string | yes | Prefix to match (min 2 chars) |

**Response 200**
```json
{
  "suggestions": [
    { "id": "...", "name": "Product A", "category": "prod1" }
  ]
}
```

**Response 400** — `q` is less than 2 characters  
**Response 503** — Elasticsearch unavailable

---

### POST /search/index
Manually index a product document into Elasticsearch.  
Used by **api-product** or a data transfer service to keep the search index in sync.

**Request body** — any JSON object with an `id` field
```json
{
  "id": "11111111-1111-1111-1111-000000000001",
  "name": "Product A",
  "description": "I have stock",
  "price": 200.00,
  "category": "prod1",
  "stock": 20,
  "discontinued": false
}
```

**Response 200**
```json
{ "indexed": true, "id": "11111111-1111-1111-1111-000000000001", "result": "created" }
```

`result` can be `created` or `updated` depending on whether the document already existed.

**Response 503** — Elasticsearch unavailable

---

## Error responses

| Status | Meaning |
|---|---|
| 400 | Missing or too-short query param |
| 503 | Elasticsearch is down |
