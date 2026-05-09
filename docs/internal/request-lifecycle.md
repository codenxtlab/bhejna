# Request Lifecycle & Flow

This document traces the path of a message from the initial client request to final reconciliation.

## 1. Dispatch Phase (Ingress)
1. **HTTP Handshake**: Client sends `POST /v1/messages` with a `Bearer` token and `Idempotency-Key`.
2. **Middleware Chain**:
    - `Logger` & `Recoverer`.
    - `APIKeyMiddleware`: Resolves `tenant_id` from the DB and checks if `is_paused`.
3. **Payload Sanitization**: 
    - Body is capped at **256KB**.
    - Phone number is stripped of spaces and validated against E.164.
4. **Quota Guard**: `CountTenantJobsInWindow` queries SQLite to ensure the 24h limit isn't exceeded.
5. **Enqueuing**: A job record is created with status `queued` (Level 0).
6. **Response**: Client receives `202 Accepted` with a `job_id`.

## 2. Processing Phase (Execution)
1. **Claiming**: A worker calls `ClaimNextJob()`, moving the status to `processing` (Level 1).
2. **Dispatch**: The `MetaAPIClient` constructs the WhatsApp envelope.
3. **External I/O**: The request is sent to Meta's Graph API.
4. **Response Handling**:
    - **Success**: Meta returns a `wamid`. Internal job is updated to `accepted` (Level 2).
    - **Transient Error (5xx)**: Job is requeued with jittered backoff.
    - **Policy Error (4xx)**: Job is marked `failed` (Level 6); Tenant is paused.

## 3. Reconciliation Phase (Webhook)
1. **Callback**: Meta sends a POST to `/webhook`.
2. **Signature Verification**: Middleware validates `X-Hub-Signature-256` using `META_APP_SECRET`.
3. **Monotonic Update**: `UpdateJobMonotonic` is called.
    - If status is "sent", update to Level 3.
    - If status is "delivered", update to Level 4.
    - If status is "read", update to Level 5.
4. **Egress Enqueue**: If the status update was successful, the event is added to `client_webhook_queue`.

## 4. Egress Phase (Callback)
1. **Poll**: Webhook workers claim the egress job.
2. **Callback**: The payload (including Meta's raw event) is POSTed to the client's `webhook_url`.
3. **Signature**: `X-Bhejna-Signature` is added if the tenant has a `webhook_secret`.
4. **Finality**: Job is marked `completed` in the egress queue.
