package api

import (
	"encoding/json"
	"errors"
	"log"
	"net/http"
	"regexp"
	"strings"
	"time"

	"github.com/codenxtlab/bhejna/internal/api/generated"
	"github.com/codenxtlab/bhejna/internal/db"
	"github.com/oklog/ulid/v2"
)

// phoneRegex matches E.164-ish format: optional leading +, then 7-15 digits only.
var phoneRegex = regexp.MustCompile(`^\+?\d{7,15}$`)

// HandleSendMessage accepts a message request and enqueues it.
func HandleSendMessage(database *db.DB) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		// Cap client request bodies at 256KB to prevent OOM from oversized payloads
		r.Body = http.MaxBytesReader(w, r.Body, 256<<10)

		tenant := GetTenant(r.Context())
		if tenant == nil {
			http.Error(w, "Unauthorized", http.StatusUnauthorized)
			return
		}

		var req generated.SendMessageRequest
		if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
			http.Error(w, "Bad Request: Invalid JSON", http.StatusBadRequest)
			return
		}

		// --- Input Sanitization ---

		// 1. Strip all whitespace from phone number
		req.Recipient = strings.ReplaceAll(req.Recipient, " ", "")
		req.Recipient = strings.TrimSpace(req.Recipient)

		// 2. Validate phone number format (E.164: optional +, 7-15 digits)
		if !phoneRegex.MatchString(req.Recipient) {
			http.Error(w, "Bad Request: Invalid recipient phone number format", http.StatusBadRequest)
			return
		}

		// 3. Validate message type against generated enum
		if !req.MessageType.Valid() {
			http.Error(w, "Bad Request: Invalid message_type", http.StatusBadRequest)
			return
		}

		// --- Quota Enforcement ---
		if tenant.MessagingLimit > 0 {
			count, err := database.CountTenantJobsInWindow(r.Context(), tenant.ID)
			if err != nil {
				log.Printf("Quota check error for tenant %s: %v", tenant.ID, err)
				http.Error(w, "Internal Server Error", http.StatusInternalServerError)
				return
			}
			if count >= tenant.MessagingLimit {
				http.Error(w, "Rate Limit Exceeded: 24h messaging quota reached", http.StatusTooManyRequests)
				return
			}
		}

		idempotencyKey := r.Header.Get("Idempotency-Key")
		var idempotencyPtr *string
		if idempotencyKey != "" {
			idempotencyPtr = &idempotencyKey
		}

		payloadBytes, err := json.Marshal(req.Payload)
		if err != nil {
			log.Printf("Error marshaling payload: %v", err)
			http.Error(w, "Internal Server Error", http.StatusInternalServerError)
			return
		}
		// 4. Cap payload size (64KB max)
		if len(payloadBytes) > 65536 {
			http.Error(w, "Bad Request: Payload too large", http.StatusBadRequest)
			return
		}

		jobID := ulid.Make().String()
		job := &db.Job{
			ID:             jobID,
			TenantID:       tenant.ID,
			RecipientPhone: req.Recipient,
			MessageType:    string(req.MessageType),
			MessagePayload: string(payloadBytes),
			Status:         "queued",
			StatusLevel:    0,
			NextRetryAt:    time.Now().UTC(),
			IdempotencyKey: idempotencyPtr,
		}

		if err := database.InsertJob(r.Context(), job); err != nil {
			if errors.Is(err, db.ErrIdempotencyConflict) {
				w.Header().Set("Content-Type", "application/json")
				w.WriteHeader(http.StatusAccepted)
				json.NewEncoder(w).Encode(generated.SendMessageResponse{
					JobId:  "existing",
					Status: generated.Queued,
				})
				return
			}
			http.Error(w, "Internal Server Error", http.StatusInternalServerError)
			return
		}

		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusAccepted)
		json.NewEncoder(w).Encode(generated.SendMessageResponse{
			JobId:  jobID,
			Status: generated.Queued,
		})
	}
}
