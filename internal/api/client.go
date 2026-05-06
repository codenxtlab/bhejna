package api

import (
	"encoding/json"
	"errors"
	"net/http"
	"time"

	"github.com/codenxtlab/bhejna/internal/db"
	"github.com/oklog/ulid/v2"
)

type SendMessageRequest struct {
	RecipientPhone string          `json:"recipient"`
	MessageType    string          `json:"message_type"`
	Payload        json.RawMessage `json:"payload"`
}

type SendMessageResponse struct {
	JobID  string `json:"job_id"`
	Status string `json:"status"`
}

// HandleSendMessage accepts a message request and enqueues it.
func HandleSendMessage(database *db.DB) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		tenant := GetTenant(r.Context())
		if tenant == nil {
			http.Error(w, "Unauthorized", http.StatusUnauthorized)
			return
		}

		var req SendMessageRequest
		if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
			http.Error(w, "Bad Request: Invalid JSON", http.StatusBadRequest)
			return
		}

		if req.RecipientPhone == "" || req.MessageType == "" {
			http.Error(w, "Bad Request: Missing required fields", http.StatusBadRequest)
			return
		}

		idempotencyKey := r.Header.Get("Idempotency-Key")
		var idempotencyPtr *string
		if idempotencyKey != "" {
			idempotencyPtr = &idempotencyKey
		}

		jobID := ulid.Make().String()
		job := &db.Job{
			ID:             jobID,
			TenantID:       tenant.ID,
			RecipientPhone: req.RecipientPhone,
			MessageType:    req.MessageType,
			MessagePayload: string(req.Payload),
			Status:         "queued",
			StatusLevel:    0,
			NextRetryAt:    time.Now(),
			IdempotencyKey: idempotencyPtr,
		}

		if err := database.InsertJob(job); err != nil {
			if errors.Is(err, db.ErrIdempotencyConflict) {
				w.Header().Set("Content-Type", "application/json")
				w.WriteHeader(http.StatusAccepted)
				json.NewEncoder(w).Encode(SendMessageResponse{
					JobID:  "existing",
					Status: "queued",
				})
				return
			}
			http.Error(w, "Internal Server Error", http.StatusInternalServerError)
			return
		}

		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusAccepted)
		json.NewEncoder(w).Encode(SendMessageResponse{
			JobID:  jobID,
			Status: "queued",
		})
	}
}
