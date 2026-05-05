package engine

import (
	"context"
	"log"
	"time"

	"github.com/codenxtlab/bhejna/internal/db"
)

// StartCleanupJanitor starts the background worker that purges old synced jobs.
func StartCleanupJanitor(ctx context.Context, database *db.DB) {
	// Run every 24 hours
	ticker := time.NewTicker(24 * time.Hour)
	defer ticker.Stop()

	log.Println("Cleanup Janitor started (24h cycle)")

	// Run once on startup to clean up immediately if needed
	performCleanup(database)

	for {
		select {
		case <-ctx.Done():
			log.Println("Cleanup Janitor stopping...")
			return
		case <-ticker.C:
			performCleanup(database)
		}
	}
}

func performCleanup(database *db.DB) {
	rows, err := database.DeleteOldSyncedJobs(7)
	if err != nil {
		log.Printf("Cleanup Janitor Error: %v", err)
		return
	}
	if rows > 0 {
		log.Printf("Cleanup Janitor: purged %d old synced jobs", rows)
	}
}
