package handlers

import (
	"context"
	"encoding/json"
	"log"

	"github.com/codenxtlab/bhejna/internal/api/generated"
	"github.com/codenxtlab/bhejna/internal/db"
	"github.com/codenxtlab/bhejna/internal/services"
	"github.com/oklog/ulid/v2"
)

var statusLevels = map[string]int{
	"sent":      3,
	"delivered": 4,
	"read":      5,
	"failed":    3,
}

type MetaWebhook struct {
	db              *db.DB
	dispatcher      *services.WebhookDispatcher
	metaVerifyToken string
}

func NewMetaWebhook(database *db.DB, dispatcher *services.WebhookDispatcher, metaVerifyToken string) *MetaWebhook {
	return &MetaWebhook{
		db:              database,
		dispatcher:      dispatcher,
		metaVerifyToken: metaVerifyToken,
	}
}

// extractPhoneNumberID safely extracts the phone number ID without panicking on nil pointers
func extractPhoneNumberID(payload *generated.WebhookPayload) string {
	if payload == nil || len(payload.Entry) == 0 {
		return ""
	}
	entry := payload.Entry[0]
	if len(entry.Changes) == 0 {
		return ""
	}
	change := entry.Changes[0]
	if change.Value.Metadata == nil || change.Value.Metadata.PhoneNumberId == nil {
		return ""
	}
	return *change.Value.Metadata.PhoneNumberId
}

// GetV1MetaWebhook implements the strict interface method for verifying the webhook
func (h *MetaWebhook) GetV1MetaWebhook(ctx context.Context, request generated.GetV1MetaWebhookRequestObject) (generated.GetV1MetaWebhookResponseObject, error) {
	if request.Params.HubMode == "subscribe" && request.Params.HubVerifyToken == h.metaVerifyToken {
		return generated.GetV1MetaWebhook200TextResponse(request.Params.HubChallenge), nil
	}

	return generated.GetV1MetaWebhook403Response{}, nil
}

// PostV1MetaWebhook implements the strict interface method for receiving Meta payloads
func (h *MetaWebhook) PostV1MetaWebhook(ctx context.Context, request generated.PostV1MetaWebhookRequestObject) (generated.PostV1MetaWebhookResponseObject, error) {
	if request.Body == nil {
		return generated.PostV1MetaWebhook200Response{}, nil
	}

	// 1. Context Cancellation Fix: Detach context to prevent cancellation when HTTP response completes
	bgCtx := context.WithoutCancel(ctx)

	// Re-marshal to get raw payload for dispatcher since strict server consumed it
	rawPayload, err := json.Marshal(request.Body)
	if err != nil {
		log.Printf("[MetaWebhook] Failed to marshal strict payload: %v", err)
		return generated.PostV1MetaWebhook200Response{}, nil
	}

	// 2. Goroutine Variable Capture Fix: Explicitly pass variables to avoid race conditions
	go func(payload *generated.WebhookPayload, raw []byte, c context.Context) {
		// Safe extraction to prevent Deep Nil-Pointer Panics
		phoneID := extractPhoneNumberID(payload)
		if phoneID != "" {
			// Fire Egress Dispatcher
			go func(p []byte, id string, childCtx context.Context) {
				h.dispatcher.Dispatch(childCtx, p, id)
			}(raw, phoneID, c)
		}

		// Fire existing internal logic for DB updates
		h.processInternalLogic(c, raw, *payload)
	}(request.Body, rawPayload, bgCtx)

	// Immediately return 200 OK so Meta doesn't retry
	return generated.PostV1MetaWebhook200Response{}, nil
}

func (h *MetaWebhook) processInternalLogic(ctx context.Context, body []byte, payload generated.WebhookPayload) {
	for _, entry := range payload.Entry {
		for _, change := range entry.Changes {
			// Status Updates (Delivery Receipts)
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

					event := &db.WebhookEvent{
						ID:             ulid.Make().String(),
						IdempotencyKey: *status.Id + ":" + statusStr,
						WabaID:         entry.Id,
						EventType:      statusStr,
						RawPayload:     string(body),
						IsMatched:      false,
					}

					matched, err := h.db.UpdateJobMonotonic(ctx, *status.Id, statusStr, level)
					if err != nil {
						log.Printf("[Webhook] ERROR: UpdateJobMonotonic failed for wamid %s: %v", *status.Id, err)
					}
					if matched {
						event.IsMatched = true
					}

					if err := h.db.InsertWebhookEvent(ctx, event); err != nil {
						log.Printf("[Webhook] ERROR: InsertWebhookEvent failed: %v", err)
					}
				}
			}

			// Inbound Messages (User replied)
			if change.Value.Messages != nil {
				for _, msg := range *change.Value.Messages {
					if msg.From == nil {
						continue
					}

					// Safely extract metadata using pointer checks
					if change.Value.Metadata == nil || change.Value.Metadata.PhoneNumberId == nil {
						continue
					}

					phoneID := *change.Value.Metadata.PhoneNumberId
					if phoneID != "" {
						tenant, err := h.db.GetTenantByPhoneNumberID(ctx, phoneID)
						if err == nil && tenant != nil {
							// Open the 24-hour free messaging window
							if err := h.db.UpsertActiveSession(ctx, tenant.ID, *msg.From); err != nil {
								log.Printf("[Webhook] ERROR: UpsertActiveSession failed for tenant %s: %v", tenant.ID, err)
							}
						}
					}
				}
			}
		}
	}
}
