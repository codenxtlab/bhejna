package api

import (
	"encoding/json"
	"io"
	"log"
	"net/http"

	"github.com/codenxtlab/bhejna/internal/api/generated"
	"github.com/codenxtlab/bhejna/internal/db"
	"github.com/oklog/ulid/v2"
)

var statusLevels = map[string]int{
	"sent":      3,
	"delivered": 4,
	"read":      5,
	"failed":    3,
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

		var payload generated.WebhookPayload
		if err := json.Unmarshal(body, &payload); err != nil {
			w.WriteHeader(http.StatusOK)
			return
		}

		// Fast Path Sweep
		for _, entry := range payload.Entry {
			for _, change := range entry.Changes {
				// 1. Check for Outbound Status Updates (Delivery Receipts)
				if change.Value.Statuses != nil {
					for _, status := range *change.Value.Statuses {
						if status.Status == nil || status.Id == nil {
							continue
						}

						statusStr := string(*status.Status)
						level, exists := statusLevels[statusStr]
						if !exists {
							continue
						}

						// Insert raw event
						event := &db.WebhookEvent{
							ID:             ulid.Make().String(),
							IdempotencyKey: *status.Id + ":" + statusStr,
							WabaID:         entry.Id,
							EventType:      statusStr,
							RawPayload:     string(body),
							IsMatched:      false,
						}

						matched, err := database.UpdateJobMonotonic(*status.Id, statusStr, level)
						if err != nil {
							log.Printf("[Webhook] ERROR: UpdateJobMonotonic failed for wamid %s: %v", *status.Id, err)
						}
						if matched {
							event.IsMatched = true
							// Enqueue for client egress
							tenant, err := database.GetTenantByWabaID(entry.Id)
							if err == nil && tenant != nil {
								if err := database.EnqueueClientWebhook(tenant.ID, string(body)); err != nil {
									log.Printf("[Webhook] ERROR: EnqueueClientWebhook failed for tenant %s: %v", tenant.ID, err)
								}
							}
						}

						if err := database.InsertWebhookEvent(event); err != nil {
							log.Printf("[Webhook] ERROR: InsertWebhookEvent failed: %v", err)
						}
					}
				}

				// 2. Check for Inbound Messages (User replied)
				if change.Value.Messages != nil {
					for _, msg := range *change.Value.Messages {
						if msg.From == nil || change.Value.Metadata == nil || change.Value.Metadata.PhoneNumberId == nil {
							continue
						}

						phoneID := *change.Value.Metadata.PhoneNumberId
						if phoneID != "" {
							tenant, err := database.GetTenantByPhoneNumberID(phoneID)
							if err == nil && tenant != nil {
								// Open the 24-hour free messaging window
								if err := database.UpsertActiveSession(tenant.ID, *msg.From); err != nil {
									log.Printf("[Webhook] ERROR: UpsertActiveSession failed for tenant %s: %v", tenant.ID, err)
								}

								// Forward the inbound message to the client's webhook
								if err := database.EnqueueClientWebhook(tenant.ID, string(body)); err != nil {
									log.Printf("[Webhook] ERROR: EnqueueClientWebhook failed for inbound message, tenant %s: %v", tenant.ID, err)
								}
							}
						}
					}
				}
			}
		}

		w.WriteHeader(http.StatusOK)
	}
}
