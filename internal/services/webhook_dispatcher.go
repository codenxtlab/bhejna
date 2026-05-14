package services

import (
	"bytes"
	"context"
	"crypto/hmac"
	"crypto/sha256"
	"encoding/hex"
	"io"
	"log"
	"net/http"
	"time"

	"github.com/codenxtlab/bhejna/internal/db"
)

type WebhookDispatcher struct {
	db     *db.DB
	client *http.Client
}

// NewWebhookDispatcher creates a new dispatcher
// 3. HTTP Client Pooling & Leaks Fix: Ensure http.Client is a single, long-lived instance with timeout
func NewWebhookDispatcher(database *db.DB) *WebhookDispatcher {
	return &WebhookDispatcher{
		db: database,
		client: &http.Client{
			Timeout: 5 * time.Second,
		},
	}
}

func (d *WebhookDispatcher) Dispatch(ctx context.Context, payload []byte, phoneNumberID string) {
	tenant, err := d.db.GetTenantByPhoneNumberID(ctx, phoneNumberID)
	if err != nil {
		log.Printf("[WebhookDispatcher] Error querying tenant for phone_number_id %s: %v", phoneNumberID, err)
		return
	}
	if tenant == nil {
		log.Printf("[WebhookDispatcher] Tenant not found for phone_number_id %s", phoneNumberID)
		return
	}
	if tenant.WebhookURL == nil || *tenant.WebhookURL == "" {
		log.Printf("[WebhookDispatcher] Tenant %s has no WebhookURL configured, dropping message", tenant.ID)
		return
	}

	req, err := http.NewRequestWithContext(ctx, http.MethodPost, *tenant.WebhookURL, bytes.NewReader(payload))
	if err != nil {
		log.Printf("[WebhookDispatcher] Error creating HTTP request for tenant %s: %v", tenant.ID, err)
		return
	}
	req.Header.Set("Content-Type", "application/json")

	if tenant.WebhookSecret != nil && *tenant.WebhookSecret != "" {
		mac := hmac.New(sha256.New, []byte(*tenant.WebhookSecret))
		mac.Write(payload)
		signature := hex.EncodeToString(mac.Sum(nil))
		req.Header.Set("X-Bhejna-Signature", "sha256="+signature)
	}

	resp, err := d.client.Do(req)
	if err != nil {
		log.Printf("[WebhookDispatcher] Error dispatching to tenant %s: %v", tenant.ID, err)
		return
	}
	// 3. HTTP Client Pooling & Leaks Fix: MUST defer resp.Body.Close() immediately after checking errors
	defer resp.Body.Close()

	// 3. HTTP Client Pooling & Leaks Fix: Recycle the TCP connection by reading the body fully
	if _, err := io.Copy(io.Discard, resp.Body); err != nil {
		log.Printf("[WebhookDispatcher] Warning: Error draining response body for tenant %s: %v", tenant.ID, err)
	}

	if resp.StatusCode >= 400 {
		log.Printf("[WebhookDispatcher] Tenant %s webhook returned status: %s", tenant.ID, resp.Status)
	} else {
		log.Printf("[WebhookDispatcher] Successfully dispatched to tenant %s webhook (Status: %s)", tenant.ID, resp.Status)
	}
}
