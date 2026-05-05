-- Tenants table for multi-tenancy and status tracking
CREATE TABLE IF NOT EXISTS tenants (
    id TEXT PRIMARY KEY,
    waba_id TEXT NOT NULL,
    phone_number_id TEXT UNIQUE NOT NULL,
    access_token TEXT NOT NULL,
    messaging_limit INTEGER DEFAULT 250,
    quality_rating TEXT DEFAULT 'GREEN',
    is_paused BOOLEAN DEFAULT 0,
    paused_until DATETIME,
    pause_reason TEXT,
    created_at DATETIME DEFAULT CURRENT_TIMESTAMP,
    updated_at DATETIME DEFAULT CURRENT_TIMESTAMP
);

-- Jobs table for message dispatching
CREATE TABLE IF NOT EXISTS jobs (
    id TEXT PRIMARY KEY,
    tenant_id TEXT NOT NULL,
    recipient_phone TEXT NOT NULL,
    message_type TEXT NOT NULL,
    message_payload TEXT NOT NULL,
    status TEXT DEFAULT 'queued',
    status_level INTEGER DEFAULT 0,
    meta_message_id TEXT,
    meta_error_code TEXT,
    retry_count INTEGER DEFAULT 0,
    next_retry_at DATETIME DEFAULT CURRENT_TIMESTAMP,
    synced BOOLEAN DEFAULT 0,
    created_at DATETIME DEFAULT CURRENT_TIMESTAMP,
    updated_at DATETIME DEFAULT CURRENT_TIMESTAMP,
    FOREIGN KEY (tenant_id) REFERENCES tenants(id)
);

-- Webhook events for idempotency and matching
CREATE TABLE IF NOT EXISTS webhook_events (
    id TEXT PRIMARY KEY,
    idempotency_key TEXT UNIQUE NOT NULL,
    waba_id TEXT NOT NULL,
    event_type TEXT NOT NULL,
    raw_payload TEXT NOT NULL,
    is_matched BOOLEAN DEFAULT 0,
    created_at DATETIME DEFAULT CURRENT_TIMESTAMP
);

-- Active 24h sessions
CREATE TABLE IF NOT EXISTS active_sessions (
    tenant_id TEXT NOT NULL,
    recipient_phone TEXT NOT NULL,
    expires_at DATETIME NOT NULL,
    PRIMARY KEY (tenant_id, recipient_phone),
    FOREIGN KEY (tenant_id) REFERENCES tenants(id)
);

-- Optimized Indexes
CREATE INDEX IF NOT EXISTS idx_jobs_claim ON jobs(next_retry_at, created_at) 
WHERE status = 'queued';

CREATE INDEX IF NOT EXISTS idx_jobs_wamid ON jobs(meta_message_id) 
WHERE meta_message_id IS NOT NULL;

CREATE INDEX IF NOT EXISTS idx_webhooks_unmatched ON webhook_events(created_at) 
WHERE is_matched = 0;
