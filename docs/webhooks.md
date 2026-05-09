# Webhook Documentation

Webhooks allow Bhejna to notify your application in real-time about delivery status updates and inbound messages.

## Delivery Guarantees

- **At-least-once delivery**: Bhejna ensures webhooks are delivered at least once.
- **Ordering**: Events are generally sent in order, but we recommend using timestamps for strictly ordered processing.
- **Retries**: If your server returns anything other than a `2xx` status, Bhejna will retry up to 5 times with exponential backoff.

---

## Security & Verification

Bhejna signs every webhook payload so you can verify it originated from our servers.

### `X-Bhejna-Signature`
This header contains an HMAC SHA256 signature of the raw request body, using your `webhook_secret` as the key.

#### Go Verification Example
```go
func VerifySignature(body []byte, signature string, secret string) bool {
    h := hmac.New(sha256.New, []byte(secret))
    h.Write(body)
    expected := hex.EncodeToString(h.Sum(nil))
    return hmac.Equal([]byte(expected), []byte(signature))
}
```

#### Node.js Verification Example
```javascript
const crypto = require('crypto');

function verifySignature(body, signature, secret) {
  const hmac = crypto.createHmac('sha256', secret);
  const expected = hmac.update(body).digest('hex');
  return expected === signature;
}
```

---

## Event Payloads

Bhejna forwards the raw Meta webhook structure for maximum compatibility.

### Delivery Update Example
```json
{
  "object": "whatsapp_business_account",
  "entry": [
    {
      "id": "WABA_ID",
      "changes": [
        {
          "field": "messages",
          "value": {
            "messaging_product": "whatsapp",
            "statuses": [
              {
                "id": "wamid.HBgLM...",
                "status": "delivered",
                "timestamp": "1622112345",
                "recipient_id": "1234567890"
              }
            ]
          }
        }
      ]
    }
  ]
}
```

### Inbound Message Example
```json
{
  "object": "whatsapp_business_account",
  "entry": [
    {
      "id": "WABA_ID",
      "changes": [
        {
          "field": "messages",
          "value": {
            "messaging_product": "whatsapp",
            "messages": [
              {
                "from": "1234567890",
                "id": "wamid.HBgLM...",
                "timestamp": "1622112345",
                "type": "text",
                "text": {
                  "body": "Hello! I have a question."
                }
              }
            ]
          }
        }
      ]
    }
  ]
}
```

---

## Best Practices

1. **Respond Quickly**: Return a `200 OK` as soon as you receive the payload. Process the event asynchronously.
2. **Idempotency**: Use the `id` (wamid) in the payload to detect duplicate events.
3. **HTTPS**: Always use an HTTPS endpoint for your `webhook_url`.
