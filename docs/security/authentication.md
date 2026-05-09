# Security & Authentication

The Bhejna architecture enforces security at multiple levels of the stack.

## Authentication Layers

### 1. Client Authentication
Clients must provide a `Bearer` token in the `Authorization` header.
- **Lookup**: Tokens are resolved in the `tenants` table.
- **Status Check**: If the `is_paused` flag is set, all requests are rejected with `403 Forbidden`.

### 2. Meta Webhook Verification
Meta sends delivery receipts via POST requests.
- **HMAC Verification**: Bhejna calculates the HMAC-SHA256 signature of the raw body using the `META_APP_SECRET`.
- **Integrity**: If the calculated signature doesn't match the `X-Hub-Signature-256` header, the request is rejected.

### 3. Internal Control Plane
Internal routes are protected by a shared `INTERNAL_SECRET`.
- **Simplicity**: High-trust communication between the frontend server and the Go backend.

## Security Invariants (Do Not Break)

### I. Body Size Limiting
All ingress points MUST limit request bodies:
- **Client API**: 256KB Max.
- **Webhooks**: 1MB Max (to accommodate large batch events).
- Failure to enforce this leads to OOM (Out of Memory) vulnerabilities.

### II. Body Exhaustion Prevention
In Go, reading the request body exhausts the `io.Reader`. Middleware that verifies signatures (like `MetaSignatureMiddleware`) MUST restore the body using `io.NopCloser` to ensure subsequent handlers can process the JSON.

### III. PII Sanitization
- Logs should NOT contain `access_token` or full message payloads.
- Log job IDs and metadata (phone ID, message type) only.

## Secret Management
Secrets should be provided via environment variables:
- `META_APP_SECRET`
- `INTERNAL_SECRET`
- `SUPABASE_SERVICE_ROLE_KEY`
- `META_VERIFY_TOKEN`
