# Concurrency Model & SQLite Serialization

Bhejna leverages Go's goroutines for high throughput while maintaining data integrity through strict SQLite serialization.

## The Worker Pools

### Message Dispatch Pool
- **Purpose**: Processes the `jobs` table.
- **Worker Count**: Configurable (Default: 5).
- **Behavior**: Continuous polling with recovery on panic. Each worker operates independently.

### Webhook Egress Pool
- **Purpose**: Processes the `client_webhook_queue` table.
- **Worker Count**: Configurable (Default: 10).
- **Behavior**: Handles outbound HTTP calls to third-party servers. Designed to handle slow downstreams without blocking message dispatch.

## SQLite WAL Management

The system uses a **Split-Connection Pool** strategy to handle the "Single-Writer" limitation of SQLite.

| Pool Type | `MaxOpenConns` | Purpose |
| :--- | :--- | :--- |
| **Writer** | **1** | All `INSERT`, `UPDATE`, `DELETE`. Ensures no `SQLITE_BUSY` locks. |
| **Reader** | **10** | All `SELECT` queries. Enables concurrent API lookups. |

### Database Busy Handling
The DSN includes `_busy_timeout=5000`. If the single writer is busy (e.g., during a long transaction), subsequent writes will wait up to 5 seconds before erroring.

## Concurrency Safety Invariants

### 1. Monotonic Leveling
Status updates are gated by levels:
```sql
UPDATE jobs SET status = ?, status_level = ? 
WHERE meta_message_id = ? AND status_level < ?
```
This prevents a late "sent" webhook from overwriting a "delivered" status.

### 2. Job Claiming Atomicity
Workers claim jobs using an atomic CTE-like update in SQLite:
```sql
UPDATE jobs SET status = 'processing' 
WHERE id = (SELECT id FROM jobs WHERE status = 'queued' LIMIT 1)
RETURNING *;
```
This ensures that no two workers can process the same message simultaneously.

### 3. Graceful Shutdown
The system uses `context.WithCancel` for all background tasks. On SIGTERM:
1. The API server stops accepting new requests.
2. The context is cancelled.
3. Workers finish their **in-flight** jobs.
4. The DB connection is closed after workers drain.
