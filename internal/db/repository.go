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
