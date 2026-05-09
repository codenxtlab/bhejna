# Bhejna Public API Reference

Welcome to the Bhejna API documentation. Bhejna is a robust, high-throughput proxy for the WhatsApp Business Platform, designed for reliability, multi-tenancy, and developer experience.

## Overview

Bhejna acts as a delivery engine between your application and Meta's WhatsApp API. It handles:
- **Queueing**: Safe message buffering.
- **Retries**: Automatic handling of transient provider failures.
- **Rate Limiting**: Per-tenant quota enforcement.
- **Webhooks**: Consistent delivery receipts and inbound message forwarding.

---

## Quick Start

### 1. Obtain an API Key
Your API key is provided during tenant provisioning. Keep it secure.

### 2. Send Your First Message
```bash
curl -X POST https://api.bhejna.com/v1/messages \
  -H "X-API-Key: YOUR_API_KEY" \
  -H "Content-Type: application/json" \
  -d '{
    "recipient": "+1234567890",
    "message_type": "text",
    "payload": {
      "body": "Hello from Bhejna!"
    }
  }'
```

### 3. Track Status
Listen for webhooks at your configured `webhook_url` to receive delivery receipts (`sent`, `delivered`, `read`).

---

## Authentication

All client requests must include the `X-API-Key` header.

| Header | Value | Description |
|---|---|---|
| `X-API-Key` | `string` | Your unique tenant API key |

---

## API Conventions

### JSON Standards
All requests and responses use standard JSON. Timestamps are returned in ISO 8601 (UTC).

### Request IDs
Every response includes a `request_id`. Include this ID when contacting support.

### Idempotency
Bhejna supports idempotent requests via the `Idempotency-Key` header. If a request fails or times out, you can safely retry with the same key.

---

## Endpoints

### POST `/v1/messages`

**Purpose**: Enqueue a message for delivery.

**Authentication**: Required (`X-API-Key`)

**Headers**:
- `Idempotency-Key`: (Optional) Unique string to prevent duplicate processing.

**Request Body**:

| Field | Type | Required | Description | Validation |
|---|---|---|---|---|
| `recipient` | `string` | Yes | Recipient phone number | E.164 format |
| `message_type` | `string` | Yes | Type of message | See Enums |
| `payload` | `object` | Yes | Meta-compatible payload | Max 64KB |

**Message Types (Enums)**:
`text`, `template`, `image`, `document`, `audio`, `video`, `sticker`, `location`, `contacts`, `interactive`

**Example Request (TypeScript)**:
```typescript
const response = await fetch('https://api.bhejna.com/v1/messages', {
  method: 'POST',
  headers: {
    'X-API-Key': 'YOUR_API_KEY',
    'Content-Type': 'application/json'
  },
  body: JSON.stringify({
    recipient: '+1234567890',
    message_type: 'text',
    payload: { body: 'Hello!' }
  })
});
```

**Example Response (Success)**:
```json
{
  "success": true,
  "data": {
    "job_id": "01ARZ3NDEKTSV4RRFFQ69G5FAV",
    "status": "queued"
  },
  "request_id": "req_abc123"
}
```

---

## Lifecycle Semantics

Messages move through the following states:

1. **`queued`**: Message accepted by Bhejna and waiting in the worker pool.
2. **`processing`**: Message being sent to Meta.
3. **`sent`**: Meta has accepted the message.
4. **`delivered`**: The message has reached the recipient's device.
5. **`read`**: The recipient has opened the message.
6. **`failed`**: Permanent failure (e.g., invalid number, provider error).

---

## Retry Semantics

- **Transient Errors**: Bhejna automatically retries 5xx errors from Meta using exponential backoff.
- **Validation Errors**: 4xx errors are considered non-retryable and marked as `failed`.
- **Idempotency**: Requests with the same `Idempotency-Key` within 24 hours will return the existing `job_id` instead of creating a new one.
