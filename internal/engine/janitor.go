package engine

import (
	"context"
	"log"
	"time"

	"github.com/codenxtlab/bhejna/internal/db"
)

// StartJanitor runs background maintenance tasks.
func StartJanitor(ctx context.Context, database *db.DB) {
	ticker1 := time.NewTicker(1 * time.Minute)
	ticker2 := time.NewTicker(15 * time.Minute)

	defer ticker1.Stop()
	defer ticker2.Stop()

	log.Println("Janitor: started background reconciliation")

	for {
		select {
		case <-ctx.Done():
			log.Println("Janitor: stopping")
			return
		case <-ticker1.C:
			// Ticker 1: Park & Sweep (Every 1 min)
			// Selects unmatched webhook events and tries to reconcile them with jobs.
			parkAndSweep(database)
		case <-ticker2.C:
			// Ticker 2: Stale Detector (Every 15 mins)
			// Alerts for jobs stuck in 'accepted' status for too long.
			staleDetector(database)
		}
	}
}

func parkAndSweep(database *db.DB) {
	events, err := database.GetUnmatchedEvents()
	if err != nil {
		log.Printf("Janitor: error fetching unmatched events: %v", err)
		return
	}

	for _, event := range events {
		// Attempt to reconcile the event with a job
		// Note: In Phase 3, we'll implement a proper webhook parser.
		wamid, status, level := parseWebhookDummy(event.RawPayload)
		if wamid == "" {
			continue
		}

		success, err := database.UpdateJobMonotonic(wamid, status, level)
		if err != nil {
			log.Printf("Janitor: error updating job %s: %v", wamid, err)
			continue
		}

		if success {
			_ = database.MarkEventMatched(event.ID)
		}
	}
}

func staleDetector(database *db.DB) {
	staleJobs, err := database.GetStaleJobs(15 * time.Minute)
	if err != nil {
		log.Printf("Janitor: error checking stale jobs: %v", err)
		return
	}

	for _, job := range staleJobs {
		log.Printf("ALERT: Stale job detected! ID: %s, Tenant: %s, Last Update: %v", 
			job.ID, job.TenantID, job.UpdatedAt)
	}
}

func parseWebhookDummy(_ string) (string, string, int) {
	// Placeholder for real webhook parsing logic
	return "", "", 0
}
