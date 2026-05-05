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

// MetaWebhookPayload represents the structure of Meta's POST webhook.
type MetaWebhookPayload struct {
	Object string `json:"object"`
	Entry  []struct {
		ID      string `json:"id"`
		Changes []struct {
			Value struct {
				Statuses []struct {
					ID     string `json:"id"`
					Status string `json:"status"`
				} `json:"statuses"`
			} `json:"value"`
			Field string `json:"field"`
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

		var payload MetaWebhookPayload
		if err := json.Unmarshal(body, &payload); err != nil {
			w.WriteHeader(http.StatusOK)
			return
		}

		// Fast Path Sweep
		for _, entry := range payload.Entry {
			for _, change := range entry.Changes {
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
					}

					_ = database.InsertWebhookEvent(event)
				}
			}
		}

		w.WriteHeader(http.StatusOK)
	}
}
