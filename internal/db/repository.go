package db

import (
	"context"
	"database/sql"
	"errors"
	"fmt"
	"math/rand"
	"strings"
	"time"

	"github.com/oklog/ulid/v2"
)

var ErrIdempotencyConflict = errors.New("idempotency key already exists")

// ClaimNextJob finds the oldest 'queued' job for an active tenant and marks it 'processing'.
func (db *DB) ClaimNextJob(ctx context.Context) (*Job, error) {
	query := `
		UPDATE jobs 
		SET status = 'processing', status_level = 1, updated_at = CURRENT_TIMESTAMP
		WHERE id = (
			SELECT j.id 
			FROM jobs j
			JOIN tenants t ON j.tenant_id = t.id
			WHERE j.status = 'queued' 
			  AND j.next_retry_at <= CURRENT_TIMESTAMP 
			  AND t.is_paused = 0
			ORDER BY j.next_retry_at ASC, j.created_at ASC
			LIMIT 1
		)
		RETURNING id, tenant_id, recipient_phone, message_type, message_payload, 
		          status, status_level, meta_message_id, meta_error_code, meta_error_message, 
		          retry_count, next_retry_at, synced, created_at, updated_at`

	var j Job
	err := db.Writer.QueryRowContext(ctx, query).Scan(
		&j.ID, &j.TenantID, &j.RecipientPhone, &j.MessageType, &j.MessagePayload,
		&j.Status, &j.StatusLevel, &j.MetaMessageID, &j.MetaErrorCode, &j.MetaErrorMessage,
		&j.RetryCount, &j.NextRetryAt, &j.Synced, &j.CreatedAt, &j.UpdatedAt,
	)

	if err == sql.ErrNoRows {
		return nil, nil
	}
	return &j, err
}

// UpdateJobMonotonic updates status only if the new level is strictly higher.
// This is critical for WhatsApp webhooks because 'delivered' or 'read' events
// may arrive before the 'sent' confirmation from the initial API call.
func (db *DB) UpdateJobMonotonic(ctx context.Context, metaMessageID string, newStatus string, newLevel int) (bool, error) {
	query := `
		UPDATE jobs 
		SET status = ?, status_level = ?, updated_at = CURRENT_TIMESTAMP
		WHERE meta_message_id = ? AND status_level < ?`

	res, err := db.Writer.ExecContext(ctx, query, newStatus, newLevel, metaMessageID, newLevel)
	if err != nil {
		return false, err
	}

	rows, err := res.RowsAffected()
	return rows > 0, err
}

// RequeueWithJitter pushes a job back to 'queued' with a random delay.
func (db *DB) RequeueWithJitter(ctx context.Context, jobID string) error {
	// 3s base + [0, 2]s random jitter
	jitter := time.Duration(3+rand.Intn(3)) * time.Second
	nextRetry := time.Now().UTC().Add(jitter)

	query := `
		UPDATE jobs 
		SET status = 'queued', status_level = 0, next_retry_at = ?, 
		    retry_count = retry_count + 1, updated_at = CURRENT_TIMESTAMP
		WHERE id = ?`

	_, err := db.Writer.ExecContext(ctx, query, nextRetry, jobID)
	return err
}

// GetTenant retrieves tenant details.
func (db *DB) GetTenant(ctx context.Context, id string) (*Tenant, error) {
	query := `SELECT id, waba_id, phone_number_id, access_token, messaging_limit, quality_rating, is_paused, webhook_url, webhook_secret FROM tenants WHERE id = ?`
	var t Tenant
	err := db.Reader.QueryRowContext(ctx, query, id).Scan(&t.ID, &t.WabaID, &t.PhoneNumberID, &t.AccessToken, &t.MessagingLimit, &t.QualityRating, &t.IsPaused, &t.WebhookURL, &t.WebhookSecret)
	if err == sql.ErrNoRows {
		return nil, nil
	}
	if err != nil {
		return nil, err
	}
	return &t, nil
}

// PauseTenant disables a tenant for policy violations.
func (db *DB) PauseTenant(ctx context.Context, id string, reason string) error {
	query := `UPDATE tenants SET is_paused = 1, pause_reason = ?, updated_at = CURRENT_TIMESTAMP WHERE id = ?`
	_, err := db.Writer.ExecContext(ctx, query, reason, id)
	return err
}

// UpdateJobStatus updates the status of a job by ID.
func (db *DB) UpdateJobStatus(ctx context.Context, id string, status string, level int) error {
	query := `UPDATE jobs SET status = ?, status_level = ?, updated_at = CURRENT_TIMESTAMP WHERE id = ?`
	_, err := db.Writer.ExecContext(ctx, query, status, level, id)
	return err
}

// MarkJobFailed records the failure reason and updates status.
func (db *DB) MarkJobFailed(ctx context.Context, id string, errorCode string, errorMessage string) error {
	query := `
		UPDATE jobs 
		SET status = 'failed', 
		    status_level = 6,
		    meta_error_code = ?, 
		    meta_error_message = ?, 
		    updated_at = CURRENT_TIMESTAMP 
		WHERE id = ?`
	_, err := db.Writer.ExecContext(ctx, query, errorCode, errorMessage, id)
	return err
}

// SetJobMetaID binds the Meta WAMID to our internal job record.
func (db *DB) SetJobMetaID(ctx context.Context, id string, metaID string) error {
	query := `UPDATE jobs SET meta_message_id = ?, updated_at = CURRENT_TIMESTAMP WHERE id = ?`
	_, err := db.Writer.ExecContext(ctx, query, metaID, id)
	return err
}

// GetUnmatchedEvents returns events that haven't been reconciled yet.
func (db *DB) GetUnmatchedEvents(ctx context.Context) ([]WebhookEvent, error) {
	query := `SELECT id, raw_payload FROM webhook_events WHERE is_matched = 0 LIMIT 100`
	rows, err := db.Reader.QueryContext(ctx, query)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var events []WebhookEvent
	for rows.Next() {
		var e WebhookEvent
		if err := rows.Scan(&e.ID, &e.RawPayload); err != nil {
			return nil, err
		}
		events = append(events, e)
	}
	return events, rows.Err()
}

// MarkEventMatched marks a webhook event as processed.
func (db *DB) MarkEventMatched(ctx context.Context, id string) error {
	query := `UPDATE webhook_events SET is_matched = 1 WHERE id = ?`
	_, err := db.Writer.ExecContext(ctx, query, id)
	return err
}

// GetStaleJobs finds jobs stuck in 'accepted' status for too long.
func (db *DB) GetStaleJobs(ctx context.Context, threshold time.Duration) ([]Job, error) {
	cutoff := time.Now().UTC().Add(-threshold)
	query := `SELECT id, tenant_id, updated_at FROM jobs WHERE status = 'accepted' AND updated_at < ?`
	rows, err := db.Reader.QueryContext(ctx, query, cutoff)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var jobs []Job
	for rows.Next() {
		var j Job
		if err := rows.Scan(&j.ID, &j.TenantID, &j.UpdatedAt); err != nil {
			return nil, err
		}
		jobs = append(jobs, j)
	}
	return jobs, rows.Err()
}

// GetTenantByAccessToken retrieves a tenant by their API key.
func (db *DB) GetTenantByAccessToken(ctx context.Context, token string) (*Tenant, error) {
	query := `SELECT id, waba_id, phone_number_id, access_token, messaging_limit, quality_rating, is_paused, webhook_url, webhook_secret FROM tenants WHERE access_token = ?`
	var t Tenant
	err := db.Reader.QueryRowContext(ctx, query, token).Scan(&t.ID, &t.WabaID, &t.PhoneNumberID, &t.AccessToken, &t.MessagingLimit, &t.QualityRating, &t.IsPaused, &t.WebhookURL, &t.WebhookSecret)
	if err == sql.ErrNoRows {
		return nil, nil
	}
	return &t, err
}

// InsertJob enqueues a new message job.
func (db *DB) InsertJob(ctx context.Context, j *Job) error {
	query := `INSERT INTO jobs (id, tenant_id, recipient_phone, message_type, message_payload, status, status_level, next_retry_at, idempotency_key, meta_error_code, meta_error_message) 
	          VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?)`
	_, err := db.Writer.ExecContext(ctx, query, j.ID, j.TenantID, j.RecipientPhone, j.MessageType, j.MessagePayload, j.Status, j.StatusLevel, j.NextRetryAt, j.IdempotencyKey, j.MetaErrorCode, j.MetaErrorMessage)
	if err != nil {
		if strings.Contains(err.Error(), "UNIQUE constraint failed") && strings.Contains(err.Error(), "idempotency_key") {
			return ErrIdempotencyConflict
		}
		return err
	}
	return nil
}

// InsertTenant provisions or syncs a tenant.
func (db *DB) InsertTenant(ctx context.Context, t *Tenant) error {
	query := `
		INSERT INTO tenants (
			id, waba_id, phone_number_id, access_token, 
			messaging_limit, quality_rating, is_paused,
			webhook_url, webhook_secret
		) 
		VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?)
		ON CONFLICT(id) DO UPDATE SET 
			waba_id = excluded.waba_id,
			phone_number_id = excluded.phone_number_id,
			access_token = excluded.access_token,
			messaging_limit = excluded.messaging_limit,
			quality_rating = excluded.quality_rating,
			is_paused = excluded.is_paused,
			webhook_url = COALESCE(excluded.webhook_url, tenants.webhook_url),
			webhook_secret = COALESCE(excluded.webhook_secret, tenants.webhook_secret),
			updated_at = CURRENT_TIMESTAMP
	`
	_, err := db.Writer.ExecContext(ctx, query,
		t.ID, t.WabaID, t.PhoneNumberID, t.AccessToken,
		t.MessagingLimit, t.QualityRating, t.IsPaused,
		t.WebhookURL, t.WebhookSecret,
	)
	return err
}

// UpsertTenantByPhone provisions or syncs a tenant based on phone number id.
func (db *DB) UpsertTenantByPhone(ctx context.Context, t *Tenant) error {
	query := `
		INSERT INTO tenants (
			id, waba_id, phone_number_id, access_token, 
			messaging_limit, quality_rating, is_paused,
			webhook_url, webhook_secret
		) 
		VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?)
		ON CONFLICT(phone_number_id) DO UPDATE SET 
			id = excluded.id,
			waba_id = excluded.waba_id,
			access_token = excluded.access_token,
			messaging_limit = excluded.messaging_limit,
			quality_rating = excluded.quality_rating,
			is_paused = excluded.is_paused,
			webhook_url = COALESCE(excluded.webhook_url, tenants.webhook_url),
			webhook_secret = COALESCE(excluded.webhook_secret, tenants.webhook_secret),
			updated_at = CURRENT_TIMESTAMP
	`
	_, err := db.Writer.ExecContext(ctx, query,
		t.ID, t.WabaID, t.PhoneNumberID, t.AccessToken,
		t.MessagingLimit, t.QualityRating, t.IsPaused,
		t.WebhookURL, t.WebhookSecret,
	)
	return err
}

// CountTenantJobsInWindow returns the number of non-failed jobs for a tenant
// created within the last 24 hours. This is the enforcement counter for
// the tenant's messaging_limit quota.
func (db *DB) CountTenantJobsInWindow(ctx context.Context, tenantID string) (int, error) {
	query := `
		SELECT COUNT(*) FROM jobs 
		WHERE tenant_id = ? 
		  AND created_at >= datetime('now', '-24 hours') 
		  AND status != 'failed'`

	var count int
	err := db.Reader.QueryRowContext(ctx, query, tenantID).Scan(&count)
	return count, err
}

// InsertWebhookEvent records a raw Meta event.
func (db *DB) InsertWebhookEvent(ctx context.Context, e *WebhookEvent) error {
	query := `INSERT INTO webhook_events (id, idempotency_key, waba_id, event_type, raw_payload, is_matched) 
	          VALUES (?, ?, ?, ?, ?, ?)`
	_, err := db.Writer.ExecContext(ctx, query, e.ID, e.IdempotencyKey, e.WabaID, e.EventType, e.RawPayload, e.IsMatched)
	return err
}

// GetUnsyncedJobs selects up to limit jobs that are in a terminal state and haven't been synced.
func (db *DB) GetUnsyncedJobs(ctx context.Context, limit int) ([]Job, error) {
	query := `
		SELECT id, tenant_id, recipient_phone, message_type, status, meta_error_code, meta_error_message, created_at 
		FROM jobs 
		WHERE synced = 0 
		  AND status IN ('delivered', 'read', 'failed')
		LIMIT ?`

	rows, err := db.Reader.QueryContext(ctx, query, limit)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var jobs []Job
	for rows.Next() {
		var j Job
		if err := rows.Scan(&j.ID, &j.TenantID, &j.RecipientPhone, &j.MessageType, &j.Status, &j.MetaErrorCode, &j.MetaErrorMessage, &j.CreatedAt); err != nil {
			return nil, err
		}
		jobs = append(jobs, j)
	}
	return jobs, rows.Err()
}

// MarkJobsSynced updates the synced flag to 1 for the given slice of job IDs.
func (db *DB) MarkJobsSynced(ctx context.Context, jobIDs []string) error {
	if len(jobIDs) == 0 {
		return nil
	}

	// SQLite supports up to 999 variables by default, but for simplicity we'll
	// just execute them in a single transaction if needed, or use a single query if small.
	// For this worker, the limit is likely small (e.g. 100).
	query := `UPDATE jobs SET synced = 1 WHERE id = ?`

	tx, err := db.Writer.BeginTx(ctx, nil)
	if err != nil {
		return err
	}
	defer tx.Rollback()

	stmt, err := tx.Prepare(query)
	if err != nil {
		return err
	}
	defer stmt.Close()

	for _, id := range jobIDs {
		if _, err := stmt.ExecContext(ctx, id); err != nil {
			return err
		}
	}

	return tx.Commit()
}

// DeleteOldSyncedJobs purges jobs that are already synced and older than the specified days.
func (db *DB) DeleteOldSyncedJobs(ctx context.Context, days int) (int64, error) {
	query := `DELETE FROM jobs WHERE synced = 1 AND created_at < datetime('now', '-' || ? || ' days')`
	res, err := db.Writer.ExecContext(ctx, query, days)
	if err != nil {
		return 0, err
	}
	return res.RowsAffected()
}

// GetTenantByPhoneNumberID retrieves a tenant by their Meta Phone Number ID.
func (db *DB) GetTenantByPhoneNumberID(ctx context.Context, phoneID string) (*Tenant, error) {
	query := `SELECT id, waba_id, phone_number_id, access_token, messaging_limit, quality_rating, is_paused, webhook_url, webhook_secret FROM tenants WHERE phone_number_id = ?`
	var t Tenant
	err := db.Reader.QueryRowContext(ctx, query, phoneID).Scan(&t.ID, &t.WabaID, &t.PhoneNumberID, &t.AccessToken, &t.MessagingLimit, &t.QualityRating, &t.IsPaused, &t.WebhookURL, &t.WebhookSecret)
	if err == sql.ErrNoRows {
		return nil, nil
	}
	return &t, err
}

// GetTenantByWabaID retrieves a tenant by their Meta WABA ID.
func (db *DB) GetTenantByWabaID(ctx context.Context, wabaID string) (*Tenant, error) {
	query := `SELECT id, waba_id, phone_number_id, access_token, messaging_limit, quality_rating, is_paused, webhook_url, webhook_secret FROM tenants WHERE waba_id = ?`
	var t Tenant
	err := db.Reader.QueryRowContext(ctx, query, wabaID).Scan(&t.ID, &t.WabaID, &t.PhoneNumberID, &t.AccessToken, &t.MessagingLimit, &t.QualityRating, &t.IsPaused, &t.WebhookURL, &t.WebhookSecret)
	if err == sql.ErrNoRows {
		return nil, nil
	}
	return &t, err
}

// UpsertActiveSession updates or inserts a 24-hour active session.
func (db *DB) UpsertActiveSession(ctx context.Context, tenantID, recipientPhone string) error {
	query := `
		INSERT INTO active_sessions (tenant_id, recipient_phone, expires_at) 
		VALUES (?, ?, datetime('now', '+24 hours')) 
		ON CONFLICT(tenant_id, recipient_phone) 
		DO UPDATE SET expires_at = datetime('now', '+24 hours')`
	_, err := db.Writer.ExecContext(ctx, query, tenantID, recipientPhone)
	return err
}

// EnqueueClientWebhook adds a payload to the client egress queue.
func (db *DB) EnqueueClientWebhook(ctx context.Context, tenantID string, payload string) error {
	id := ulid.Make().String()
	query := `INSERT INTO client_webhook_queue (id, tenant_id, payload) VALUES (?, ?, ?)`
	_, err := db.Writer.ExecContext(ctx, query, id, tenantID, payload)
	return err
}

// ClaimClientWebhook grabs the oldest queued webhook job and joins tenant info.
func (db *DB) ClaimClientWebhook(ctx context.Context) (*ClientWebhookJob, error) {
	query := `
		UPDATE client_webhook_queue 
		SET status = 'processing', next_retry_at = datetime('now', '+1 minute')
		WHERE id = (
			SELECT q.id 
			FROM client_webhook_queue q
			WHERE q.status = 'queued' AND q.next_retry_at <= CURRENT_TIMESTAMP
			ORDER BY q.created_at ASC
			LIMIT 1
		)
		RETURNING id, tenant_id, payload, status, retry_count, next_retry_at, created_at`

	var q ClientWebhookJob
	err := db.Writer.QueryRowContext(ctx, query).Scan(
		&q.ID, &q.TenantID, &q.Payload, &q.Status, &q.RetryCount, &q.NextRetryAt, &q.CreatedAt,
	)

	if err == sql.ErrNoRows {
		return nil, nil
	}
	if err != nil {
		return nil, err
	}

	// Join tenant info manually for simplicity or use a CTE in the RETURNING (though RETURNING join is tricky in SQLite)
	tenant, err := db.GetTenant(ctx, q.TenantID)
	if err != nil {
		return nil, err
	}
	if tenant == nil {
		return nil, fmt.Errorf("tenant %s not found for webhook job %s", q.TenantID, q.ID)
	}

	if tenant.WebhookURL != nil {
		q.WebhookURL = *tenant.WebhookURL
	}
	if tenant.WebhookSecret != nil {
		q.WebhookSecret = *tenant.WebhookSecret
	}

	return &q, nil
}

// MarkClientWebhookFailed updates retry count and next retry time.
func (db *DB) MarkClientWebhookFailed(ctx context.Context, id string, retryCount int, nextRetry time.Time) error {
	query := `UPDATE client_webhook_queue SET status = 'queued', retry_count = ?, next_retry_at = ? WHERE id = ?`
	_, err := db.Writer.ExecContext(ctx, query, retryCount, nextRetry, id)
	return err
}

// MarkClientWebhookSuccess marks the job as completed.
func (db *DB) MarkClientWebhookSuccess(ctx context.Context, id string) error {
	query := `UPDATE client_webhook_queue SET status = 'completed' WHERE id = ?`
	_, err := db.Writer.ExecContext(ctx, query, id)
	return err
}
