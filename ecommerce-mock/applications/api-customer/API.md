# api-customer

**Stack:** .NET 9 + Serilog + PostgreSQL  
**Port:** 8082  
**Base URL:** `http://localhost:8082`

---

## Endpoints

### GET /health
Returns service status.

**Response 200**
```json
{ "status": "ok", "service": "api-customer" }
```

---

### POST /customers/register
Register a new customer account.

**Request body**
```json
{
  "email": "customer_1@mail.com",
  "password": "customer_1@mail.com_pass",
  "firstName": "customer",
  "lastName": "1"
}
```

**Response 201** — customer profile (no password hash)
```json
{
  "id": "22222222-2222-2222-2222-000000000001",
  "email": "customer_1@mail.com",
  "firstName": "customer",
  "lastName": "1",
  "createdAt": "2026-01-01T00:00:00Z"
}
```

**Response 409** — email already registered

---

### GET /customers/:id
Get customer profile by UUID.

**Response 200** — customer profile  
**Response 404**
```json
{ "error": "customer not found", "customer_id": "<id>" }
```

---

### PUT /customers/:id
Update customer profile (partial — only provided fields change).

**Request body** (all fields optional)
```json
{
  "email": "new@mail.com",
  "firstName": "New",
  "lastName": "Name"
}
```

**Response 200** — updated customer profile  
**Response 404** — customer not found  
**Response 409** — new email already in use

---

### POST /auth/login
Authenticate and receive JWT tokens.

**Request body**
```json
{
  "email": "customer_1@mail.com",
  "password": "customer_1@mail.com_pass"
}
```

**Response 200**
```json
{
  "accessToken": "<jwt>",
  "refreshToken": "<jwt>",
  "expiresAt": "2026-01-01T01:00:00Z",
  "customer": { "id": "...", "email": "..." }
}
```

**Response 401** — invalid credentials

---

### POST /auth/logout
Revoke the current access token.

**Headers**
```
Authorization: Bearer <access_token>
```

**Response 200**
```json
{ "message": "logged out" }
```

**Response 401** — invalid or expired token

---

### POST /auth/refresh
Exchange a refresh token for new access + refresh tokens.

**Headers**
```
Authorization: Bearer <refresh_token>
```

**Response 200** — new token pair (same shape as login)  
**Response 401** — expired, revoked, or invalid token

---

## Error responses

| Status | Meaning |
|---|---|
| 400 | Validation error |
| 401 | Authentication failed / invalid token |
| 404 | Customer not found |
| 409 | Email conflict |
| 500 | Database error |
