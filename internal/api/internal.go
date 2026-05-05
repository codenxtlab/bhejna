package api

import (
	"encoding/json"
	"fmt"
	"net/http"

	"github.com/codenxtlab/bhejna/internal/db"
	"github.com/go-chi/chi/v5"
	"github.com/oklog/ulid/v2"
)

type ProvisionTenantRequest struct {
	WabaID        string `json:"waba_id"`
	PhoneNumberID string `json:"phone_number_id"`
}

type ProvisionTenantResponse struct {
	TenantID    string `json:"tenant_id"`
	AccessToken string `json:"access_token"`
}

// HandleProvisionTenant creates a new tenant and generates an API key.
func HandleProvisionTenant(database *db.DB) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		var req ProvisionTenantRequest
		if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
			http.Error(w, "Bad Request", http.StatusBadRequest)
			return
		}

		tenantID := ulid.Make().String()
		apiKey := fmt.Sprintf("nxt_live_%s", ulid.Make().String())

		tenant := &db.Tenant{
			ID:            tenantID,
			WabaID:        req.WabaID,
			PhoneNumberID: req.PhoneNumberID,
			AccessToken:   apiKey,
		}

		if err := database.InsertTenant(tenant); err != nil {
			http.Error(w, "Internal Server Error", http.StatusInternalServerError)
			return
		}

		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusCreated)
		json.NewEncoder(w).Encode(ProvisionTenantResponse{
			TenantID:    tenantID,
			AccessToken: apiKey,
		})
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
