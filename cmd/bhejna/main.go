package main

import (
	"context"
	"log"
	"net/http"
	"os"
	"os/signal"
	"sync"
	"syscall"
	"time"

	"github.com/codenxtlab/bhejna/internal/api"
	"github.com/codenxtlab/bhejna/internal/db"
	"github.com/codenxtlab/bhejna/internal/engine"
	"github.com/go-chi/chi/v5"
	"github.com/go-chi/chi/v5/middleware"
	"github.com/joho/godotenv"
)

func main() {
	// 0. Load Environment Variables from file
	envFile := ".env"
	if os.Getenv("GO_ENV") == "production" {
		envFile = ".env.production"
	}
	if err := godotenv.Load(envFile); err != nil {
		log.Printf("Warning: Error loading %s file: %v (relying on system env)", envFile, err)
	}

	// 1. Load configuration from environment
	dbPath := getEnv("DB_PATH", "bhejna.db")
	port := getEnv("PORT", "8080")
	metaAppSecret := getEnv("META_APP_SECRET", "super_secret_app")
	internalSecret := getEnv("INTERNAL_SECRET", "control_plane_secret")
	metaVerifyToken := getEnv("META_VERIFY_TOKEN", "verify_me")
	workerCount := 5 // Default worker count
	supabaseURL := os.Getenv("SUPABASE_URL")
	supabaseServiceKey := os.Getenv("SUPABASE_SERVICE_ROLE_KEY")

	// 2. Initialize Database
	database, err := db.InitDB(dbPath)
	if err != nil {
		log.Fatalf("Failed to initialize database: %v", err)
	}
	defer database.Close()

	// 2.5 Boot-Time Hydration
	if supabaseServiceKey != "" && supabaseURL != "" {
		log.Println("Starting Boot-Time Hydration from Supabase...")
		if err := engine.HydrateTenantsFromSupabase(database, supabaseURL, supabaseServiceKey); err != nil {
			log.Fatalf("Fatal: Boot-Time Hydration failed: %v", err)
		}
	} else {
		log.Println("Warning: SUPABASE_SERVICE_ROLE_KEY or SUPABASE_URL missing, skipping boot-time hydration.")
	}

	// 3. Initialize Engine Components
	limiters := engine.NewLimiterManager()
	metaClient := engine.NewMetaAPIClient()
	pool := engine.NewWorkerPool(database, limiters, metaClient, workerCount)

	// Context for background tasks
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	// 4. Start Worker Pool
	pool.Start(ctx)

	// 4.5 Start Client Webhook Pool
	webhookPool := engine.NewClientWebhookPool(database)
	webhookPool.Start(ctx, 10)

	// 5. Start Janitors — tracked by a WaitGroup so shutdown waits for them
	var bgWg sync.WaitGroup

	bgWg.Add(1)
	go func() {
		defer bgWg.Done()
		engine.StartJanitor(ctx, database)
	}()

	if supabaseServiceKey != "" {
		bgWg.Add(1)
		go func() {
			defer bgWg.Done()
			engine.StartSupabaseSync(ctx, database, supabaseURL, supabaseServiceKey)
		}()
	} else {
		log.Println("Error: SUPABASE_SERVICE_ROLE_KEY is missing. Supabase sync engine is disabled.")
	}

	bgWg.Add(1)
	go func() {
		defer bgWg.Done()
		engine.StartCleanupJanitor(ctx, database)
	}()

	// 6. Set up Chi Router
	r := chi.NewRouter()
	r.Use(middleware.Logger)
	r.Use(middleware.Recoverer)

	// Webhook routes
	r.Get("/webhook", api.HandleWebhookValidation(metaVerifyToken))
	r.With(api.MetaSignatureMiddleware(metaAppSecret)).Post("/webhook", api.HandleWebhookEvent(database))

	// Group 2: Client routes
	r.Group(func(r chi.Router) {
		r.Use(api.APIKeyMiddleware(database))
		r.Post("/v1/messages", api.HandleSendMessage(database))
	})

	// Group 3: Internal routes
	// Note: HandleSyncTenant is outside the strict InternalJWTMiddleware group 
	// because it now handles its own auth (checking both Header and Body).
	r.Post("/v1/internal/tenant", api.HandleSyncTenant(database, internalSecret))
	
	r.Group(func(r chi.Router) {
		r.Use(api.InternalJWTMiddleware(internalSecret))
		r.Put("/v1/internal/tenants/{id}/pause", api.HandlePauseTenant(database))
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

	// Cancel background context — signals all workers and janitors to stop
	cancel()

	// Shutdown HTTP server — stops accepting new connections, waits for in-flight
	shutdownCtx, shutdownCancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer shutdownCancel()

	if err := srv.Shutdown(shutdownCtx); err != nil {
		log.Printf("HTTP server Shutdown: %v", err)
	}

	// Wait for all worker pools to drain their in-flight jobs
	log.Println("Server stopped. Waiting for workers to finish...")
	pool.Stop()
	log.Println("Message workers stopped. Waiting for webhook workers...")
	webhookPool.Stop()
	log.Println("Webhook workers stopped. Waiting for background janitors...")
	bgWg.Wait()
	log.Println("Bhejna shutdown complete.")
}

func getEnv(key, fallback string) string {
	if value, exists := os.LookupEnv(key); exists {
		return value
	}
	return fallback
}
