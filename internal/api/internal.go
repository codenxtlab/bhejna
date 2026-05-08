package api

import (
	"encoding/json"
	"io"
	"net/http"
	"strings"

	"github.com/codenxtlab/bhejna/internal/db"
	"github.com/go-chi/chi/v5"
)

// HandleSyncTenant processes a provisioning request from SvelteKit or a Supabase Webhook.
func HandleSyncTenant(database *db.DB, internalSecret string) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		// Read the body once so we can try multiple unmarshal strategies
		bodyBytes, err := io.ReadAll(r.Body)
		if err != nil {
			http.Error(w, "Bad Request", http.StatusBadRequest)
			return
		}

		// Check for system_token in body if not already authorized by middleware
		// (Though middleware usually runs first, we'll check here to be sure)
		var authCheck struct {
			SystemToken string `json:"system_token"`
		}
		json.Unmarshal(bodyBytes, &authCheck)

		// Verification logic: check header first (via middleware usually), then body
		authHeader := r.Header.Get("Authorization")
		token := ""
		if strings.HasPrefix(authHeader, "Bearer ") {
			token = strings.TrimPrefix(authHeader, "Bearer ")
		} else if authCheck.SystemToken != "" {
			token = authCheck.SystemToken
		}

		if token != internalSecret {
			http.Error(w, "Unauthorized: Invalid or missing token", http.StatusUnauthorized)
			return
		}

		// Option 1: Supabase Webhook payload format
		var webhookPayload struct {
			Record *db.Tenant `json:"record"`
		}

		var tenant *db.Tenant

		if err := json.Unmarshal(bodyBytes, &webhookPayload); err == nil && webhookPayload.Record != nil && (webhookPayload.Record.ID != "") {
			tenant = webhookPayload.Record
		} else {
			// Option 2: Direct Tenant payload format (SvelteKit)
			var directReq struct {
				db.Tenant
				TenantID string `json:"tenant_id"`
			}
			if err := json.Unmarshal(bodyBytes, &directReq); err != nil {
				http.Error(w, "Bad Request: invalid payload", http.StatusBadRequest)
				return
			}
			tenant = &directReq.Tenant
			// Map tenant_id to ID if ID is empty
			if tenant.ID == "" {
				tenant.ID = directReq.TenantID
			}

			if tenant.ID == "" {
				http.Error(w, "Bad Request: missing tenant identification", http.StatusBadRequest)
				return
			}
		}

		// UPSERT the tenant into local SQLite
		if err := database.InsertTenant(tenant); err != nil {
			http.Error(w, "Internal Server Error", http.StatusInternalServerError)
			return
		}

		w.WriteHeader(http.StatusOK)
	}
}

// HandlePauseTenant manually pauses a tenant.
func HandlePauseTenant(database *db.DB) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		tenantID := chi.URLParam(r, "id")
		if tenantID == "" {
			http.Error(w, "Bad Request: Missing ID", http.StatusBadRequest)
			return
		}

		if err := database.PauseTenant(tenantID, "MANUAL_PAUSE"); err != nil {
			http.Error(w, "Internal Server Error", http.StatusInternalServerError)
			return
		}

		w.WriteHeader(http.StatusNoContent)
	}
}
