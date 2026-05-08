# Bhejna Public API Documentation

Welcome to the Bhejna Messaging API. This documentation provides the technical specifications for integrating our high-reliability WhatsApp Cloud API infrastructure into your application.

## Base URL

All API requests should be made to:
`https://api.bhejna.codenxtlab.tech`

## Authentication

Bhejna uses API keys to authenticate requests. You can view and manage your API keys in the Bhejna Dashboard.

Your API key must be passed in the `Authorization` header as a Bearer token. All production keys start with the prefix `nxt_live_`.

**Header:**
`Authorization: Bearer nxt_live_xxxxxxxxxxxxxxxx`

---

## Send Message

Dispatches a message to a recipient's WhatsApp number. This endpoint is asynchronous; a successful response indicates that the message has been validated and queued for delivery.

### Endpoint
`POST /v1/messages`

### Request Headers
| Header | Required | Description |
| :--- | :--- | :--- |
| `Authorization` | Yes | `Bearer <YOUR_API_KEY>` |
| `Content-Type` | Yes | `application/json` |
| `Idempotency-Key` | No | A unique string to prevent duplicate processing of the same request. |

### Request Body
The request body must be a JSON object with the following fields:

| Field | Type | Description |
| :--- | :--- | :--- |
| `recipient` | string | The recipient's phone number in E.164 format (e.g., `15551234567`). |
| `message_type` | string | The type of message being sent. Supported: `template`, `text`. |
| `payload` | object | The message content object. This follows the standard WhatsApp Cloud API structure for the specified `message_type`. |

#### Example: Template Message
To send a template message, set `message_type` to `template` and provide the template details in the `payload`.

```json
{
  "recipient": "15551234567",
  "message_type": "template",
  "payload": {
    "name": "hello_world",
    "language": {
      "code": "en_US"
    },
    "components": [
      {
        "type": "body",
        "parameters": [
          {
            "type": "text",
            "text": "BarberBase"
          }
        ]
      }
    ]
  }
}
```

### Response Codes

| Status Code | Description |
| :--- | :--- |
| `202 Accepted` | Message successfully queued. Returns a `job_id` for tracking. |
| `400 Bad Request` | Invalid JSON payload or missing required fields. |
| `401 Unauthorized` | Invalid or missing API key. |
| `403 Forbidden` | Account is paused due to policy violations or limits. |
| `429 Too Many Requests` | Rate limit exceeded for the sender. |

### Example Response (`202 Accepted`)
```json
{
  "job_id": "01H2X3Y4Z5W6V7U8T9S0R1Q2P3",
  "status": "queued"
}
```

---

## Integration Snippet (cURL)

```bash
curl -X POST https://api.bhejna.codenxtlab.tech/v1/messages \
  -H "Authorization: Bearer nxt_live_your_api_key_here" \
  -H "Content-Type: application/json" \
  -H "Idempotency-Key: unique_request_id_123" \
  -d '{
    "recipient": "15551234567",
    "message_type": "template",
    "payload": {
      "name": "sample_template",
      "language": { "code": "en_US" }
    }
  }'
```
