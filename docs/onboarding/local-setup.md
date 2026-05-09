# Local Setup Guide

Follow these steps to get Bhejna running on your local machine for development.

## 1. Prerequisites
- **Go**: Version 1.21 or higher.
- **SQLite**: Version 3.35 or higher (required for `RETURNING` clause support).
- **Meta Developer Account**: To test with real WhatsApp webhooks (or use ngrok).

## 2. Environment Variables
Bhejna reads configuration from environment variables or a `.env` file.

| Variable | Description |
| :--- | :--- |
| `PORT` | The port to listen on (default: 8080). |
| `DB_PATH` | Path to the SQLite file. |
| `META_APP_SECRET` | Your Facebook App Secret. |
| `INTERNAL_SECRET` | Shared secret for internal API calls. |
| `META_VERIFY_TOKEN` | The token you set in Meta's Webhook configuration. |

## 3. Installation
```bash
# Clone the repository
git clone <repo_url>
cd bhejna

# Install dependencies
go mod download
```

## 4. Running the Application
```bash
# Start in development mode
go run cmd/bhejna/main.go
```

The application will:
1. Create `bhejna.db` if it doesn't exist.
2. Execute `schema.sql` to set up tables.
3. Start the worker pools and the HTTP server.

## 5. Development Workflow
- **Adding Handlers**: Add routes in `main.go` and logic in `internal/api/`.
- **Adding DB Methods**: Update `internal/db/repository.go` and the corresponding struct in `models.go`.
- **Adding Workers**: Define worker logic in `internal/engine/`.

## 6. Testing with Webhooks
To receive webhooks from Meta on your local machine:
1. Start ngrok: `ngrok http 8080`.
2. Copy the HTTPS URL provided by ngrok.
3. Go to the Meta Developer Portal -> WhatsApp -> Configuration.
4. Set the Callback URL to `<ngrok_url>/webhook`.
5. Set the Verify Token to your `META_VERIFY_TOKEN`.
