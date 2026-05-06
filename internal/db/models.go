package db

import (
	"database/sql"
	"time"
)

type Tenant struct {
	ID             string         `db:"id" json:"id"`
	WabaID         string         `db:"waba_id" json:"waba_id"`
	PhoneNumberID  string         `db:"phone_number_id" json:"phone_number_id"`
	AccessToken    string         `db:"access_token" json:"access_token"`
	MessagingLimit int            `db:"messaging_limit" json:"messaging_limit"`
	QualityRating  string         `db:"quality_rating" json:"quality_rating"`
	IsPaused       bool           `db:"is_paused" json:"is_paused"`
	PausedUntil    sql.NullTime   `db:"paused_until" json:"-"`
	PauseReason    sql.NullString `db:"pause_reason" json:"-"`
	CreatedAt      time.Time      `db:"created_at" json:"created_at"`
	UpdatedAt      time.Time      `db:"updated_at" json:"updated_at"`
}

type Job struct {
	ID             string         `db:"id"`
	TenantID       string         `db:"tenant_id"`
	RecipientPhone string         `db:"recipient_phone"`
	MessageType    string         `db:"message_type"`
	MessagePayload string         `db:"message_payload"`
	Status         string         `db:"status"`
	StatusLevel    int            `db:"status_level"`
	MetaMessageID  sql.NullString `db:"meta_message_id"`
	MetaErrorCode  sql.NullString `db:"meta_error_code"`
	RetryCount     int            `db:"retry_count"`
	NextRetryAt    time.Time      `db:"next_retry_at"`
	Synced         bool           `db:"synced"`
	CreatedAt      time.Time      `db:"created_at"`
	UpdatedAt      time.Time      `db:"updated_at"`
}

type WebhookEvent struct {
	ID             string    `db:"id"`
	IdempotencyKey string    `db:"idempotency_key"`
	WabaID         string    `db:"waba_id"`
	EventType      string    `db:"event_type"`
	RawPayload     string    `db:"raw_payload"`
	IsMatched      bool      `db:"is_matched"`
	CreatedAt      time.Time `db:"created_at"`
}

type ActiveSession struct {
	TenantID       string    `db:"tenant_id"`
	RecipientPhone string    `db:"recipient_phone"`
	ExpiresAt      time.Time `db:"expires_at"`
}
