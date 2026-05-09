# Examples Library

This library provides production-ready code examples for integrating with Bhejna.

---

## Sending a Text Message

### cURL
```bash
curl -X POST https://api.bhejna.com/v1/messages \
  -H "X-API-Key: YOUR_API_KEY" \
  -H "Content-Type: application/json" \
  -d '{
    "recipient": "+1234567890",
    "message_type": "text",
    "payload": {
      "body": "Hello World"
    }
  }'
```

### TypeScript
```typescript
async function sendMessage() {
  const res = await fetch('https://api.bhejna.com/v1/messages', {
    method: 'POST',
    headers: {
      'X-API-Key': process.env.BHEJNA_API_KEY,
      'Content-Type': 'application/json',
      'Idempotency-Key': 'unique_key_123'
    },
    body: JSON.stringify({
      recipient: '+1234567890',
      message_type: 'text',
      payload: { body: 'Hello from TS' }
    })
  });
  const data = await res.json();
  console.log(data);
}
```

### Python
```python
import requests

def send_message():
    url = "https://api.bhejna.com/v1/messages"
    headers = {
        "X-API-Key": "YOUR_API_KEY",
        "Content-Type": "application/json"
    }
    payload = {
        "recipient": "+1234567890",
        "message_type": "text",
        "payload": {"body": "Hello from Python"}
    }
    response = requests.post(url, json=payload, headers=headers)
    print(response.json())
```

---

## Sending a Template Message

### Go
```go
package main

import (
	"bytes"
	"encoding/json"
	"net/http"
)

func main() {
	payload := map[string]interface{}{
		"recipient": "+1234567890",
		"message_type": "template",
		"payload": map[string]interface{}{
			"name": "hello_world",
			"language": map[string]string{"code": "en_US"},
		},
	}
	
	body, _ := json.Marshal(payload)
	req, _ := http.NewRequest("POST", "https://api.bhejna.com/v1/messages", bytes.NewBuffer(body))
	req.Header.Set("X-API-Key", "YOUR_API_KEY")
	req.Header.Set("Content-Type", "application/json")

	client := &http.Client{}
	client.Do(req)
}
```

---

## Webhook Receiver (Node.js/Express)

```javascript
const express = require('express');
const crypto = require('crypto');
const app = express();

app.use(express.json());

const WEBHOOK_SECRET = process.env.WEBHOOK_SECRET;

app.post('/webhook', (req, res) => {
  const signature = req.headers['x-bhejna-signature'];
  const body = JSON.stringify(req.body);
  
  const hmac = crypto.createHmac('sha256', WEBHOOK_SECRET);
  const expected = hmac.update(body).digest('hex');

  if (signature !== expected) {
    return res.status(401).send('Invalid signature');
  }

  // Process the event
  console.log('Received event:', req.body);
  
  res.status(200).send('OK');
});

app.listen(3000);
```

---

## Idempotent Retries

When sending a message, always generate a unique key for each attempt to avoid duplicate charges or messages.

```javascript
const idempotencyKey = crypto.randomUUID();

async function sendWithRetry() {
  for (let i = 0; i < 3; i++) {
    try {
      const res = await sendMessage(idempotencyKey);
      if (res.ok) return;
    } catch (e) {
      // Exponential backoff
      await new Promise(r => setTimeout(r, Math.pow(2, i) * 1000));
    }
  }
}
```
