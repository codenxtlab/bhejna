package api

import (
	"encoding/json"
	"io"
	"net/http"

	"github.com/codenxtlab/bhejna/internal/db"
	"github.com/go-chi/chi/v5"
)

// HandleSyncTenant processes a Supabase Database Webhook to cache a tenant locally.
func HandleSyncTenant(database *db.DB) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		// Read the body once so we can try multiple unmarshal strategies
		bodyBytes, err := io.ReadAll(r.Body)
		if err != nil {
			http.Error(w, "Bad Request", http.StatusBadRequest)
			return
		}

		// Option 1: Supabase Webhook payload format
		var webhookPayload struct {
			Record *db.Tenant `json:"record"`
		}

		var tenant *db.Tenant
		
		if err := json.Unmarshal(bodyBytes, &webhookPayload); err == nil && webhookPayload.Record != nil && webhookPayload.Record.ID != "" {
			tenant = webhookPayload.Record
		} else {
			// Option 2: Direct Tenant payload format
			var directTenant db.Tenant
			if err := json.Unmarshal(bodyBytes, &directTenant); err != nil || directTenant.ID == "" {
				http.Error(w, "Bad Request: invalid payload", http.StatusBadRequest)
				return
			}
			tenant = &directTenant
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
