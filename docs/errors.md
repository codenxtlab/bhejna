# Error Reference

Bhejna uses standard HTTP status codes and a consistent error response format to help you debug integration issues.

## Error Response Format

All error responses follow this structure:

```json
{
  "success": false,
  "error": {
    "code": "ERROR_CODE",
    "message": "Human readable explanation",
    "retryable": false
  },
  "request_id": "req_123"
}
```

---

## HTTP Status Codes

| Status | Code | Description | Retryable |
|---|---|---|---|
| 400 | `BAD_REQUEST` | Invalid JSON, missing fields, or validation failed. | No |
| 401 | `UNAUTHORIZED` | Missing or invalid `X-API-Key`. | No |
| 403 | `FORBIDDEN` | Tenant is paused or unauthorized for the action. | No |
| 404 | `NOT_FOUND` | The requested resource does not exist. | No |
| 429 | `RATE_LIMIT_EXCEEDED` | Messaging quota reached for the 24h window. | No |
| 500 | `INTERNAL_SERVER_ERROR` | An unexpected error occurred on our side. | Yes |

---

## Internal Error Codes

### `INVALID_PHONE`
**HTTP Status**: 400
**Description**: The recipient phone number does not match E.164 format.
**Fix**: Ensure the phone number contains only digits and an optional leading `+`.

### `PAYLOAD_TOO_LARGE`
**HTTP Status**: 400
**Description**: The message payload exceeds the 64KB limit.
**Fix**: Reduce the size of your message payload or media metadata.

### `QUOTA_EXCEEDED`
**HTTP Status**: 429
**Description**: Your tenant has reached the maximum allowed messages for the current window.
**Fix**: Wait for the window to reset or contact support to increase your limit.

### `IDEMPOTENCY_CONFLICT`
**HTTP Status**: 202 (Accepted)
**Description**: A request with this `Idempotency-Key` has already been processed.
**Note**: This is not strictly an error but an advisory that we are returning the existing job.

---

## Troubleshooting Guidance

1. **Check the `request_id`**: Always log the `request_id` returned in the response. It is the fastest way for our support team to help you.
2. **Validate Payloads**: Use a JSON validator before sending requests. Bhejna strictly validates the `message_type` against allowed values.
3. **Handle 5xx Errors**: If you receive a 500 error, wait a few seconds and retry with the same `Idempotency-Key`.
