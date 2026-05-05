package main

import (
	"context"
	"log"
	"net/http"
	"os"
	"os/signal"
	"syscall"
	"time"

	"github.com/codenxtlab/bhejna/internal/api"
	"github.com/codenxtlab/bhejna/internal/db"
	"github.com/codenxtlab/bhejna/internal/engine"
	"github.com/go-chi/chi/v5"
	"github.com/go-chi/chi/v5/middleware"
)

func main() {
	// 1. Load configuration from environment
	dbPath := getEnv("DB_PATH", "bhejna.db")
	port := getEnv("PORT", "8080")
	metaAppSecret := getEnv("META_APP_SECRET", "super_secret_app")
	internalSecret := getEnv("INTERNAL_SECRET", "control_plane_secret")
	metaVerifyToken := getEnv("META_VERIFY_TOKEN", "verify_me")
	workerCount := 5 // Default worker count

	// 2. Initialize Database
	database, err := db.InitDB(dbPath)
	if err != nil {
		log.Fatalf("Failed to initialize database: %v", err)
	}
	defer database.Close()

	// 3. Initialize Engine Components
	limiters := engine.NewLimiterManager()
	metaClient := engine.NewMetaAPIClient()
	pool := engine.NewWorkerPool(database, limiters, metaClient, workerCount)

	// Context for background tasks
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	// 4. Start Worker Pool
	pool.Start(ctx)

	// 5. Start Janitor
	go engine.StartJanitor(ctx, database)

	// 6. Set up Chi Router
	r := chi.NewRouter()
	r.Use(middleware.Logger)
	r.Use(middleware.Recoverer)

	// Group 1: Webhook routes
	r.Group(func(r chi.Router) {
		r.Use(api.MetaSignatureMiddleware(metaAppSecret))
		r.Get("/webhook", api.HandleWebhookValidation(metaVerifyToken))
		r.Post("/webhook", api.HandleWebhookEvent(database))
	})

	// Group 2: Client routes
	r.Group(func(r chi.Router) {
		r.Use(api.APIKeyMiddleware(database))
		r.Post("/v1/messages", api.HandleSendMessage(database))
	})

	// Group 3: Internal routes
	r.Group(func(r chi.Router) {
		r.Use(api.InternalJWTMiddleware(internalSecret))
		r.Post("/api/internal/tenants", api.HandleProvisionTenant(database))
		r.Put("/api/internal/tenants/{id}/pause", api.HandlePauseTenant(database))
	})

	// 7. Start HTTP Server
	srv := &http.Server{
		Addr:    ":" + port,
		Handler: r,
	}

	go func() {
		log.Printf("Bhejna API started on port %s", port)
		if err := srv.ListenAndServe(); err != nil && err != http.ErrServerClosed {
			log.Fatalf("Listen: %v", err)
		}
	}()

	// 8. Graceful Shutdown
	stop := make(chan os.Signal, 1)
	signal.Notify(stop, os.Interrupt, syscall.SIGINT, syscall.SIGTERM)

	<-stop
	log.Println("Shutting down gracefully...")

	// Cancel background context
	cancel()

	// Shutdown HTTP server
	shutdownCtx, shutdownCancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer shutdownCancel()

	if err := srv.Shutdown(shutdownCtx); err != nil {
		log.Printf("HTTP server Shutdown: %v", err)
	}

	log.Println("Server stopped. Waiting for workers to finish...")
	pool.Stop()
	log.Println("Bhejna shutdown complete.")
}

func getEnv(key, fallback string) string {
	if value, exists := os.LookupEnv(key); exists {
		return value
	}
	return fallback
}
