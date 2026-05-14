package api

import (
	"encoding/json"
	"io"
	"log"
	"net/http"
	"strings"

	"github.com/codenxtlab/bhejna/internal/api/generated"
	"github.com/codenxtlab/bhejna/internal/db"
	"github.com/go-chi/chi/v5"
)

// HandleSyncTenant processes a provisioning request from SvelteKit or a Supabase Webhook.
func HandleSyncTenant(database *db.DB, internalSecret string) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		// Cap internal payloads at 1MB to prevent OOM from oversized bodies
		r.Body = http.MaxBytesReader(w, r.Body, 1<<20)

		// Read the body once so we can try multiple unmarshal strategies
		bodyBytes, err := io.ReadAll(r.Body)
		if err != nil {
			http.Error(w, "Bad Request", http.StatusBadRequest)
			return
		}

		// Check for system_token in body if not already authorized by middleware
		var authCheck struct {
			SystemToken string `json:"system_token"`
		}
		if err := json.Unmarshal(bodyBytes, &authCheck); err != nil {
			log.Printf("Warning: Failed to unmarshal authCheck token: %v", err)
		}

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

		var syncBody generated.SyncTenantJSONBody
		if err := json.Unmarshal(bodyBytes, &syncBody); err != nil {
			http.Error(w, "Bad Request: invalid payload structure", http.StatusBadRequest)
			return
		}

		var genTenant generated.Tenant
		// Try AsSyncTenantJSONBody1 first (the wrapped {record: ...} format)
		if wrapped, err := syncBody.AsSyncTenantJSONBody1(); err == nil && wrapped.Record.Id != "" {
			genTenant = wrapped.Record
		} else if direct, err := syncBody.AsTenant(); err == nil && direct.Id != "" {
			genTenant = direct
		} else {
			http.Error(w, "Bad Request: invalid tenant data", http.StatusBadRequest)
			return
		}

		// Convert generated.Tenant to db.Tenant
		tenant := &db.Tenant{
			ID:            genTenant.Id,
			WabaID:        genTenant.WabaId,
			PhoneNumberID: genTenant.PhoneNumberId,
			IsPaused:      false,
		}

		if genTenant.ApiKey != nil {
			tenant.AccessToken = *genTenant.ApiKey
		}
		if genTenant.MessagingLimit != nil {
			tenant.MessagingLimit = *genTenant.MessagingLimit
		}
		if genTenant.QualityRating != nil {
			tenant.QualityRating = *genTenant.QualityRating
		}
		if genTenant.IsPaused != nil {
			tenant.IsPaused = *genTenant.IsPaused
		}
		tenant.WebhookURL = genTenant.WebhookUrl
		tenant.WebhookSecret = genTenant.WebhookSecret

		// UPSERT the tenant into local SQLite
		if err := database.InsertTenant(r.Context(), tenant); err != nil {
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

		if err := database.PauseTenant(r.Context(), tenantID, "MANUAL_PAUSE"); err != nil {
			http.Error(w, "Internal Server Error", http.StatusInternalServerError)
			return
		}

		w.WriteHeader(http.StatusNoContent)
	}
}
