# API Overview

Bhejna provides two distinct API surfaces: the **External Client API** and the **Internal Control Plane**.

## 1. External Client API
Base URL: `/v1`

### POST /messages
Dispatches a WhatsApp message.
- **Auth**: `Authorization: Bearer <tenant_token>`
- **Headers**: `Idempotency-Key` (Optional, ULID recommended)
- **Body**:
```json
{
    "recipient": "+1234567890",
    "message_type": "template",
    "payload": {
        "name": "hello_world",
        "language": { "code": "en_US" }
    }
}
```
- **Responses**:
    - `202 Accepted`: Job enqueued.
    - `401 Unauthorized`: Invalid token.
    - `403 Forbidden`: Tenant is paused.
    - `429 Too Many Requests`: 24h quota exceeded.

## 2. Webhook Ingress
Endpoint: `/webhook`

### GET /webhook
Handles Meta's verification challenge.
- **Query Params**: `hub.mode`, `hub.verify_token`, `hub.challenge`.

### POST /webhook
Processes delivery receipts and inbound messages.
- **Security**: Validates `X-Hub-Signature-256`.
- **Response**: Always returns `200 OK` to Meta to prevent retry loops.

## 3. Internal Control Plane
Base URL: `/v1/internal`

### POST /internal/tenant
Synchronizes tenant metadata from the management system.
- **Auth**: `INTERNAL_SECRET` check.
- **Action**: Upserts tenant WABA IDs and credentials.

### PUT /internal/tenants/{id}/pause
Manually pauses a tenant.
- **Auth**: `INTERNAL_SECRET` check.
