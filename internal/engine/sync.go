package engine

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"log"
	"net/http"
	"strconv"
	"time"

	"github.com/codenxtlab/bhejna/internal/db"
)

// SupabaseJob matches the schema of the Supabase jobs_analytics table.
type SupabaseJob struct {
	ID             string `json:"id"`
	TenantID       string `json:"tenant_id"`
	RecipientPhone string `json:"recipient_phone"`
	MessageType    string `json:"message_type"`
	Status         string `json:"status"`
	MetaErrorCode  int    `json:"meta_error_code"`
	CreatedAt      string `json:"created_at"`
}

// StartSupabaseSync starts the worker that syncs terminal job states to Supabase.
func StartSupabaseSync(ctx context.Context, database *db.DB, supabaseURL, supabaseServiceKey string) {
	if supabaseURL == "" || supabaseServiceKey == "" {
		log.Println("Supabase sync disabled: missing URL or Service Role Key")
		return
	}

	ticker := time.NewTicker(5 * time.Minute)
	defer ticker.Stop()

	client := &http.Client{
		Timeout: 10 * time.Second,
	}

	log.Println("Supabase Sync Worker started")

	for {
		select {
		case <-ctx.Done():
			log.Println("Supabase Sync Worker stopping...")
			return
		case <-ticker.C:
			if err := syncJobs(database, client, supabaseURL, supabaseServiceKey); err != nil {
				log.Printf("Supabase Sync Error: %v", err)
			}
		}
	}
}

func syncJobs(database *db.DB, client *http.Client, url, key string) error {
	// Fetch up to 100 unsynced jobs
	jobs, err := database.GetUnsyncedJobs(100)
	if err != nil {
		return fmt.Errorf("failed to fetch unsynced jobs: %w", err)
	}

	if len(jobs) == 0 {
		return nil
	}

	// Map to Supabase structure
	var payload []SupabaseJob
	var jobIDs []string

	for _, j := range jobs {
		sj := SupabaseJob{
			ID:             j.ID,
			TenantID:       j.TenantID,
			RecipientPhone: j.RecipientPhone,
			MessageType:    j.MessageType,
			Status:         j.Status,
			CreatedAt:      j.CreatedAt.Format(time.RFC3339),
		}
		if j.MetaErrorCode.Valid {
			code, _ := strconv.Atoi(j.MetaErrorCode.String)
			sj.MetaErrorCode = code
		}
		payload = append(payload, sj)
		jobIDs = append(jobIDs, j.ID)
	}

	// Prepare request
	body, err := json.Marshal(payload)
	if err != nil {
		return fmt.Errorf("failed to marshal payload: %w", err)
	}

	reqURL := fmt.Sprintf("%s/rest/v1/jobs_analytics", url)
	req, err := http.NewRequest("POST", reqURL, bytes.NewBuffer(body))
	if err != nil {
		return fmt.Errorf("failed to create request: %w", err)
	}

	req.Header.Set("apikey", key)
	req.Header.Set("Authorization", "Bearer "+key)
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("Prefer", "return=minimal")

	resp, err := client.Do(req)
	if err != nil {
		return fmt.Errorf("failed to execute request: %w", err)
	}
	defer resp.Body.Close()
	_, _ = io.Copy(io.Discard, resp.Body)

	if resp.StatusCode >= 200 && resp.StatusCode < 300 {
		if err := database.MarkJobsSynced(jobIDs); err != nil {
			return fmt.Errorf("failed to mark jobs as synced: %w", err)
		}
		log.Printf("Successfully synced %d jobs to Supabase", len(jobIDs))
	} else {
		return fmt.Errorf("supabase returned non-2xx status: %d", resp.StatusCode)
	}

	return nil
}

// HydrateTenantsFromSupabase fetches all tenants from Supabase and upserts them locally.
func HydrateTenantsFromSupabase(database *db.DB, url, key string) error {
	reqURL := fmt.Sprintf("%s/rest/v1/tenants?select=*", url)
	req, err := http.NewRequest("GET", reqURL, nil)
	if err != nil {
		return fmt.Errorf("failed to create hydration request: %w", err)
	}

	req.Header.Set("apikey", key)
	req.Header.Set("Authorization", "Bearer "+key)

	client := &http.Client{Timeout: 10 * time.Second}
	resp, err := client.Do(req)
	if err != nil {
		return fmt.Errorf("failed to execute hydration request: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != 200 {
		return fmt.Errorf("supabase returned non-200 status during hydration: %d", resp.StatusCode)
	}

	var tenants []db.Tenant
	if err := json.NewDecoder(resp.Body).Decode(&tenants); err != nil {
		return fmt.Errorf("failed to decode hydration response: %w", err)
	}

	for _, t := range tenants {
		if err := database.UpsertTenantByPhone(&t); err != nil {
			return fmt.Errorf("failed to upsert tenant %s during hydration: %w", t.PhoneNumberID, err)
		}
	}

	log.Printf("Boot-Time Hydration complete: successfully synced %d tenants from Supabase", len(tenants))
	return nil
}
