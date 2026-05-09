# Operations & Deployment

Bhejna is container-native and designed to run as a single-node engine per region.

## Deployment Model

### Docker
The application is containerized using the provided `Dockerfile`. 
- **Volumes**: The SQLite database file (`bhejna.db`) MUST be stored on a persistent volume to survive container restarts.
- **Reverse Proxy**: It is recommended to run Bhejna behind Caddy or Nginx for SSL termination.

### Scaling Assumption
Bhejna is **vertically scalable**. A single instance can handle thousands of messages per second due to the non-blocking nature of the worker pools. Horizontal scaling requires LiteFS or moving to a centralized PostgreSQL.

## Failure Scenarios

### Database Outage (Rare for SQLite)
- **Impact**: All API calls will 500. Workers will stop.
- **Recovery**: Check file permissions on the DB volume. Ensure `bhejna.db` is not corrupted.

### Redis Outage
- **Impact**: Currently, Bhejna does NOT use Redis. Rate limits are in-memory.
- **Benefit**: Zero-dependency runtime.
- **Trade-off**: Restarts reset rate limit counters.

---

# Onboarding: Local Setup

## 1. Prerequisites
- Go 1.21+
- SQLite 3
- (Optional) Docker

## 2. Environment Setup
Create a `.env` file in the root:
```env
DB_PATH=bhejna.db
PORT=8080
META_APP_SECRET=your_secret
INTERNAL_SECRET=your_secret
META_VERIFY_TOKEN=verify_me
SUPABASE_URL=...
SUPABASE_SERVICE_ROLE_KEY=...
```

## 3. Running the Server
```bash
go run cmd/bhejna/main.go
```
The database and schema will be automatically initialized on the first boot.

## 4. Testing Endpoints
Use `curl` to test the status:
```bash
curl http://localhost:8080/webhook?hub.mode=subscribe&hub.challenge=123&hub.verify_token=verify_me
```
Expected: `123`
