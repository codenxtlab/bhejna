package db

import (
	"database/sql"
	"math/rand"
	"time"
)

// ClaimNextJob finds the oldest 'queued' job for an active tenant and marks it 'processing'.
func (db *DB) ClaimNextJob() (*Job, error) {
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
		          status, status_level, meta_message_id, meta_error_code, 
		          retry_count, next_retry_at, synced, created_at, updated_at`

	var j Job
	err := db.Writer.QueryRow(query).Scan(
		&j.ID, &j.TenantID, &j.RecipientPhone, &j.MessageType, &j.MessagePayload,
		&j.Status, &j.StatusLevel, &j.MetaMessageID, &j.MetaErrorCode,
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
func (db *DB) UpdateJobMonotonic(metaMessageID string, newStatus string, newLevel int) (bool, error) {
	query := `
		UPDATE jobs 
		SET status = ?, status_level = ?, updated_at = CURRENT_TIMESTAMP
		WHERE meta_message_id = ? AND status_level < ?`

	res, err := db.Writer.Exec(query, newStatus, newLevel, metaMessageID, newLevel)
	if err != nil {
		return false, err
	}

	rows, err := res.RowsAffected()
	return rows > 0, err
}

// RequeueWithJitter pushes a job back to 'queued' with a random delay.
func (db *DB) RequeueWithJitter(jobID string) error {
	// 3s base + [0, 2]s random jitter
	jitter := time.Duration(3+rand.Intn(3)) * time.Second
	nextRetry := time.Now().Add(jitter)

	query := `
		UPDATE jobs 
		SET status = 'queued', status_level = 0, next_retry_at = ?, 
		    retry_count = retry_count + 1, updated_at = CURRENT_TIMESTAMP
		WHERE id = ?`

	_, err := db.Writer.Exec(query, nextRetry, jobID)
	return err
}

// GetTenant retrieves tenant details.
func (db *DB) GetTenant(id string) (*Tenant, error) {
	query := `SELECT id, waba_id, phone_number_id, access_token, messaging_limit, quality_rating, is_paused FROM tenants WHERE id = ?`
	var t Tenant
	err := db.Reader.QueryRow(query, id).Scan(&t.ID, &t.WabaID, &t.PhoneNumberID, &t.AccessToken, &t.MessagingLimit, &t.QualityRating, &t.IsPaused)
	return &t, err
}

// PauseTenant disables a tenant for policy violations.
func (db *DB) PauseTenant(id string, reason string) error {
	query := `UPDATE tenants SET is_paused = 1, pause_reason = ?, updated_at = CURRENT_TIMESTAMP WHERE id = ?`
	_, err := db.Writer.Exec(query, reason, id)
	return err
}

// UpdateJobStatus updates the status of a job by ID.
func (db *DB) UpdateJobStatus(id string, status string, level int) error {
	query := `UPDATE jobs SET status = ?, status_level = ?, updated_at = CURRENT_TIMESTAMP WHERE id = ?`
	_, err := db.Writer.Exec(query, status, level, id)
	return err
}

// SetJobMetaID binds the Meta WAMID to our internal job record.
func (db *DB) SetJobMetaID(id string, metaID string) error {
	query := `UPDATE jobs SET meta_message_id = ?, updated_at = CURRENT_TIMESTAMP WHERE id = ?`
	_, err := db.Writer.Exec(query, metaID, id)
	return err
}

// GetUnmatchedEvents returns events that haven't been reconciled yet.
func (db *DB) GetUnmatchedEvents() ([]WebhookEvent, error) {
	query := `SELECT id, raw_payload FROM webhook_events WHERE is_matched = 0 LIMIT 100`
	rows, err := db.Reader.Query(query)
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
	return events, nil
}

// MarkEventMatched marks a webhook event as processed.
func (db *DB) MarkEventMatched(id string) error {
	query := `UPDATE webhook_events SET is_matched = 1 WHERE id = ?`
	_, err := db.Writer.Exec(query, id)
	return err
}

// GetStaleJobs finds jobs stuck in 'accepted' status for too long.
func (db *DB) GetStaleJobs(threshold time.Duration) ([]Job, error) {
	cutoff := time.Now().Add(-threshold)
	query := `SELECT id, tenant_id, updated_at FROM jobs WHERE status = 'accepted' AND updated_at < ?`
	rows, err := db.Reader.Query(query, cutoff)
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
	return jobs, nil
}

// GetTenantByAccessToken retrieves a tenant by their API key.
func (db *DB) GetTenantByAccessToken(token string) (*Tenant, error) {
	query := `SELECT id, waba_id, phone_number_id, access_token, messaging_limit, quality_rating, is_paused FROM tenants WHERE access_token = ?`
	var t Tenant
	err := db.Reader.QueryRow(query, token).Scan(&t.ID, &t.WabaID, &t.PhoneNumberID, &t.AccessToken, &t.MessagingLimit, &t.QualityRating, &t.IsPaused)
	if err == sql.ErrNoRows {
		return nil, nil
	}
	return &t, err
}

// InsertJob enqueues a new message job.
func (db *DB) InsertJob(j *Job) error {
	query := `INSERT INTO jobs (id, tenant_id, recipient_phone, message_type, message_payload, status, status_level, next_retry_at) 
	          VALUES (?, ?, ?, ?, ?, ?, ?, ?)`
	_, err := db.Writer.Exec(query, j.ID, j.TenantID, j.RecipientPhone, j.MessageType, j.MessagePayload, j.Status, j.StatusLevel, j.NextRetryAt)
	return err
}

// InsertTenant provisions a new tenant.
func (db *DB) InsertTenant(t *Tenant) error {
	query := `INSERT INTO tenants (id, waba_id, phone_number_id, access_token) VALUES (?, ?, ?, ?)`
	_, err := db.Writer.Exec(query, t.ID, t.WabaID, t.PhoneNumberID, t.AccessToken)
	return err
}

// InsertWebhookEvent records a raw Meta event.
func (db *DB) InsertWebhookEvent(e *WebhookEvent) error {
	query := `INSERT INTO webhook_events (id, idempotency_key, waba_id, event_type, raw_payload, is_matched) 
	          VALUES (?, ?, ?, ?, ?, ?)`
	_, err := db.Writer.Exec(query, e.ID, e.IdempotencyKey, e.WabaID, e.EventType, e.RawPayload, e.IsMatched)
	return err
}
// GetUnsyncedJobs selects up to limit jobs that are in a terminal state and haven't been synced.
func (db *DB) GetUnsyncedJobs(limit int) ([]Job, error) {
	query := `
		SELECT id, tenant_id, recipient_phone, message_type, status, meta_error_code, created_at 
		FROM jobs 
		WHERE synced = 0 
		  AND status IN ('delivered', 'read', 'failed')
		LIMIT ?`

	rows, err := db.Reader.Query(query, limit)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var jobs []Job
	for rows.Next() {
		var j Job
		if err := rows.Scan(&j.ID, &j.TenantID, &j.RecipientPhone, &j.MessageType, &j.Status, &j.MetaErrorCode, &j.CreatedAt); err != nil {
			return nil, err
		}
		jobs = append(jobs, j)
	}
	return jobs, nil
}

// MarkJobsSynced updates the synced flag to 1 for the given slice of job IDs.
func (db *DB) MarkJobsSynced(jobIDs []string) error {
	if len(jobIDs) == 0 {
		return nil
	}

	// SQLite supports up to 999 variables by default, but for simplicity we'll 
	// just execute them in a single transaction if needed, or use a single query if small.
	// For this worker, the limit is likely small (e.g. 100).
	query := `UPDATE jobs SET synced = 1 WHERE id = ?`
	
	tx, err := db.Writer.Begin()
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
		if _, err := stmt.Exec(id); err != nil {
			return err
		}
	}

	return tx.Commit()
}

// DeleteOldSyncedJobs purges jobs that are already synced and older than the specified days.
func (db *DB) DeleteOldSyncedJobs(days int) (int64, error) {
	query := `DELETE FROM jobs WHERE synced = 1 AND created_at < datetime('now', '-' || ? || ' days')`
	res, err := db.Writer.Exec(query, days)
	if err != nil {
		return 0, err
	}
	return res.RowsAffected()
}
