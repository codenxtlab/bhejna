package api

import (
	"encoding/json"
	"io"
	"net/http"

	"github.com/codenxtlab/bhejna/internal/db"
	"github.com/oklog/ulid/v2"
)

var statusLevels = map[string]int{
	"sent":      3,
	"delivered": 4,
	"read":      5,
	"failed":    0,
}

// WebhookPayload represents the root JSON sent by Meta.
type WebhookPayload struct {
	Object string `json:"object"` // Usually "whatsapp_business_account"
	Entry  []struct {
		ID      string `json:"id"` // WABA ID
		Changes []struct {
			Field string `json:"field"` // Usually "messages"
			Value struct {
				MessagingProduct string `json:"messaging_product"`
				Metadata         struct {
					DisplayPhoneNumber string `json:"display_phone_number"`
					PhoneNumberID      string `json:"phone_number_id"`
				} `json:"metadata"`
                
				// Statuses represents delivery receipts (sent, delivered, read, failed)
				Statuses []struct {
					ID          string `json:"id"`     // The wamid
					Status      string `json:"status"` 
					Timestamp   string `json:"timestamp"`
					RecipientID string `json:"recipient_id"`
					Errors      []struct {
						Code  int    `json:"code"`
						Title string `json:"title"`
					} `json:"errors,omitempty"`
				} `json:"statuses,omitempty"`
                
				// Messages represents actual inbound text/media from the user
				Messages []struct {
					From      string `json:"from"`
					ID        string `json:"id"` // Inbound wamid
					Timestamp string `json:"timestamp"`
					Type      string `json:"type"` // "text", "image", etc.
					Text      struct {
						Body string `json:"body"`
					} `json:"text,omitempty"`
				} `json:"messages,omitempty"`
                
			} `json:"value"`
		} `json:"changes"`
	} `json:"entry"`
}

// HandleWebhookValidation handles Meta's verification challenge.
func HandleWebhookValidation(verifyToken string) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		mode := r.URL.Query().Get("hub.mode")
		token := r.URL.Query().Get("hub.verify_token")
		challenge := r.URL.Query().Get("hub.challenge")

		if mode == "subscribe" && token == verifyToken {
			w.WriteHeader(http.StatusOK)
			w.Write([]byte(challenge))
			return
		}

		http.Error(w, "Forbidden", http.StatusForbidden)
	}
}

// HandleWebhookEvent processes status updates from Meta.
func HandleWebhookEvent(database *db.DB) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		body, err := io.ReadAll(r.Body)
		if err != nil {
			// Always return 200 to Meta to avoid retries on bad payloads
			w.WriteHeader(http.StatusOK)
			return
		}

		var payload WebhookPayload
		if err := json.Unmarshal(body, &payload); err != nil {
			w.WriteHeader(http.StatusOK)
			return
		}

		// Fast Path Sweep
		for _, entry := range payload.Entry {
			for _, change := range entry.Changes {
				// 1. Check for Outbound Status Updates (Delivery Receipts)
				for _, status := range change.Value.Statuses {
					level, exists := statusLevels[status.Status]
					if !exists {
						continue
					}

					// Insert raw event
					event := &db.WebhookEvent{
						ID:             ulid.Make().String(),
						IdempotencyKey: status.ID + ":" + status.Status,
						WabaID:         entry.ID,
						EventType:      status.Status,
						RawPayload:     string(body),
						IsMatched:      false,
					}

					matched, _ := database.UpdateJobMonotonic(status.ID, status.Status, level)
					if matched {
						event.IsMatched = true
						// Enqueue for client egress
						tenant, err := database.GetTenantByWabaID(entry.ID)
						if err == nil && tenant != nil {
							_ = database.EnqueueClientWebhook(tenant.ID, string(body))
						}
					}

					_ = database.InsertWebhookEvent(event)
				}

				// 2. Check for Inbound Messages (User replied)
				for _, msg := range change.Value.Messages {
					phoneID := change.Value.Metadata.PhoneNumberID
					if phoneID != "" {
						tenant, err := database.GetTenantByPhoneNumberID(phoneID)
						if err == nil && tenant != nil {
							_ = database.UpsertActiveSession(tenant.ID, msg.From)
						}
					}
				}
			}
		}

		w.WriteHeader(http.StatusOK)
	}
}
