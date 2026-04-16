# api-payment

**Stack:** .NET 9 + Serilog + MySQL  
**Port:** 8085  
**Base URL:** `http://localhost:8085`

> Uses a **MockPaymentProvider** with configurable rates:  
> 75% success · 15% decline · 10% timeout (via env vars)  
> Max retries: 3 (configurable via `Payment__MaxRetries`)

---

## Endpoints

### GET /health
Returns service status.

**Response 200**
```json
{ "status": "ok", "service": "api-payment" }
```

---

### POST /payments
Process a payment for an order.  
Creates a payment record in `PROCESSING` state, calls the mock provider, then finalises to `SUCCESS` or `FAILED`.

**Request body**
```json
{
  "orderId": "33333333-0001-0000-0000-000000000001",
  "customerId": "22222222-2222-2222-2222-000000000001",
  "amount": 400.00
}
```

**Response 200** — payment succeeded
```json
{
  "id": "44444444-0001-0000-0000-000000000001",
  "orderId": "33333333-0001-0000-0000-000000000001",
  "customerId": "22222222-2222-2222-2222-000000000001",
  "amount": 400.00,
  "status": "SUCCESS",
  "providerRef": "PRV-10001",
  "failureReason": null,
  "retryCount": 0,
  "createdAt": "2026-01-01T00:00:00Z",
  "updatedAt": "2026-01-01T00:00:00Z"
}
```

**Response 402** — payment declined
```json
{ ...payment, "status": "FAILED", "failureReason": "card_declined" }
```

**Response 500** — database error

---

### GET /payments/:id
Get payment status by UUID.

**Response 200** — payment object  
**Response 404**
```json
{ "error": "payment not found", "payment_id": "<id>" }
```

---

### POST /payments/:id/retry
Retry a failed payment. Only `FAILED` payments can be retried.  
Increments `retryCount` on each attempt.

**Response 200** — payment succeeded after retry  
**Response 402** — payment declined again  
**Response 400** — payment is not in FAILED state  
**Response 404** — payment not found  
**Response 429** — max retries exceeded
```json
{
  "error": "max retries exceeded",
  "retry_count": 3,
  "max_retries": 3
}
```

---

### POST /payments/webhook
Receive an external provider callback to update payment status.

**Request body**
```json
{
  "providerRef": "PRV-10001",
  "status": "SUCCESS",
  "failureReason": null
}
```

`status` values: `SUCCESS` or `FAILED`

**Response 200**
```json
{ "message": "webhook processed", "payment_id": "<id>", "status": "SUCCESS" }
```

**Response 404** — no payment found for that `providerRef`

---

## Payment status flow

```
(new) → PROCESSING → SUCCESS
                   → FAILED  → retry → PROCESSING → SUCCESS
                                                   → FAILED (retryCount++)
                                                   → 429 when retryCount >= maxRetries
```

---

## Error responses

| Status | Meaning |
|---|---|
| 400 | Payment not in FAILED state (retry) |
| 402 | Payment declined by provider |
| 404 | Payment not found |
| 429 | Max retries exhausted |
| 500 | Database error |
