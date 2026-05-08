# Bhejna Backend: WhatsApp Cloud API Proxy

Bhejna is a high-performance Go-based proxy for the WhatsApp Cloud API (Meta). It features a local SQLite queue for edge reliability and an asynchronous synchronization engine for Supabase.

## Architecture Overview

The system follows a "Queue-First" architecture to ensure zero message loss even during Meta API outages or rate-limiting events.

1.  **SvelteKit -> Go Router**: The frontend (Dashboard) or external clients send requests to the Go backend.
2.  **Go Router -> SQLite Queue**: Incoming messages are validated and immediately persisted into the local SQLite `jobs` table.
3.  **SQLite Queue -> Go Worker**: A pool of background workers (Meta Dispatchers) continuously poll the SQLite database for new jobs.
4.  **Go Worker -> Meta API**: The worker reconstructs the Meta JSON envelope and dispatches the request to the WhatsApp Cloud API using a System User Token.
5.  **Supabase Sync Worker**: A separate background process periodically syncs terminal job states (Success/Failure) and tenant updates back to the central Supabase database.

### Database & Schema
- **SQLite with WAL Mode**: The backend uses SQLite in Write-Ahead Logging (WAL) mode. This allows for concurrent readers while maintaining a single serial writer, preventing "database is locked" errors under load.
- **Embedded Schema**: The database schema is embedded directly into the Go binary using `go:embed`. It is automatically applied on startup, ensuring the edge database is always in the correct state.

## Environment Variables

Create a `.env` file in the root directory with the following variables:

| Variable | Description | Example |
| :--- | :--- | :--- |
| `PORT` | Port the server listens on | `8080` |
| `DB_PATH` | Path to the SQLite database file | `bhejna.db` |
| `INTERNAL_SECRET` | Secret used for internal control-plane routes | `your_high_entropy_secret` |
| `SUPABASE_URL` | Your Supabase project URL | `https://xyz.supabase.co` |
| `SUPABASE_SERVICE_ROLE_KEY` | Supabase Service Role Key (Required for sync) | `eyJhbGci...` |
| `META_APP_SECRET` | Meta App Secret (for webhook verification) | `abcdef123...` |
| `META_VERIFY_TOKEN` | Meta Webhook Verify Token | `bhejna_verify_2026` |
| `META_SYSTEM_USER_TOKEN` | Global Meta System User Token (Optional fallback) | `EAAWq...` |

## Internal Routing

### `POST /v1/internal/tenant`
Used to synchronize tenant provisioning (API Keys, Phone IDs) from the control plane (SvelteKit/Supabase) to the local edge database.

**Authentication:**
The endpoint accepts authentication via two methods:
1.  **Header**: `Authorization: Bearer <INTERNAL_SECRET>`
2.  **Body**: `{ "system_token": "<INTERNAL_SECRET>" }`

**Payload Examples:**
- **SvelteKit Direct**: `{ "tenant_id": "uuid", "api_key": "sb_...", "phone_number_id": "123" }`
- **Supabase Webhook**: `{ "record": { "id": "uuid", "api_key": "sb_...", ... } }`

## Worker Processes

### 1. Meta Dispatcher
The primary worker responsible for message delivery.
- **Envelope Construction**: It wraps the internal payload into the official Meta format (adding `messaging_product: "whatsapp"`, `to`, and `type`).
- **Error Handling**: 
    - **Transient (5xx)**: Automatically requeues the job with a randomized jitter.
    - **Policy (400/Limit)**: Fails the job and automatically **pauses the tenant** to prevent further account risk.
- **Status Monotonicity**: Ensures statuses like `accepted`, `delivered`, and `read` are updated in order.

### 2. Supabase Sync Worker
An asynchronous task that reconciles the edge state with the central database.
- **Schema**: Maps the SQLite `jobs` table to the Supabase `jobs_analytics` table.
- **Frequency**: Runs every 5 minutes (default) to batch-upload terminal states.
- **Requirement**: Requires `SUPABASE_SERVICE_ROLE_KEY` to bypass RLS for analytical writes.

## Running Locally

To run the Bhejna server in development mode:

```bash
# Using go run
go run cmd/bhejna/main.go

# Or build and run
go build -o bhejna cmd/bhejna/main.go
./bhejna
```

The server will automatically load variables from the `.env` file in the current working directory.
