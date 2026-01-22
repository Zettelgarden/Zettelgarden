package main

import (
	"context"
	"fmt"
	"go-backend/bootstrap"
	"go-backend/handlers"
	"go-backend/mail"
	"go-backend/models"
	"go-backend/pkg/config"
	"go-backend/routes"
	"go-backend/server"
	"go-backend/services"
	"log"
	"net/http"
	"os"
	"os/signal"
	"syscall"
	"time"

	"github.com/gorilla/mux"
	"github.com/rs/cors"
	openai "github.com/sashabaranov/go-openai"
	"github.com/stripe/stripe-go/v82"
)

var s *server.Server
var h *handlers.Handler

// safeGoroutine runs a function in a goroutine with panic recovery
func safeGoroutine(fn func()) {
	go func() {
		defer func() {
			if r := recover(); r != nil {
				log.Printf("panic in goroutine: %v", r)
			}
		}()
		fn()
	}()
}

func configureLogging(cfg config.Config) (*os.File, func(), error) {
	if cfg.Server.DevMode {
		return nil, func() {}, nil
	}

	logPath := cfg.Server.LogLocation
	if logPath == "" {
		return nil, nil, fmt.Errorf("ZETTEL_BACKEND_LOG_LOCATION is empty")
	}

	file, err := handlers.OpenLogFile(logPath)
	if err != nil {
		return nil, nil, err
	}

	previous := log.Writer()
	log.SetOutput(file)

	cleanup := func() {
		// Restore output first so any shutdown logs don't try to write to a closing file.
		log.SetOutput(previous)
		_ = file.Sync()
		_ = file.Close()
	}

	return file, cleanup, nil
}

func run() error {
	log.Printf("Starting Zettelgarden backend server...")

	// Load and validate all configuration from environment variables
	cfg := config.LoadConfig()
	log.Printf("Configuration loaded successfully (dev_mode=%v)", cfg.Server.DevMode)

	_, cleanupLogging, err := configureLogging(cfg)
	if err != nil {
		return err
	}
	defer cleanupLogging()

	// Initialize shared server using bootstrap package
	log.Printf("Initializing database connection (host=%s, port=%s, db=%s)", cfg.Database.Host, cfg.Database.Port, cfg.Database.DatabaseName)
	if s = bootstrap.InitServer(cfg.Database); s == nil {
		log.Fatalf("Failed to initialize server")
	}
	log.Printf("Database connected and migrations completed successfully")
	if s != nil && s.DB != nil {
		defer func() {
			if err := s.DB.Close(); err != nil {
				log.Printf("error closing database: %v", err)
			}
		}()
	}

	h = &handlers.Handler{
		Server:    s,
		DB:        s.DB,
		ToolRetry: services.NewToolCircuitBreaker(),
	}

	// Initialize Stripe
	log.Printf("Initializing Stripe payment processing")
	stripe.Key = cfg.Services.Stripe.SecretKey
	log.Printf("Stripe initialized (billing_url=%s)", cfg.Services.Stripe.BillingURL)

	// Initialize S3 client for file storage
	log.Printf("Initializing S3 client (bucket=%s)", cfg.Services.S3.BucketName)
	s.S3 = h.CreateS3Client()
	log.Printf("S3 client initialized successfully")

	// Initialize mail client
	log.Printf("Initializing mail client (host=%s)", cfg.Services.Mail.Host)
	s.Mail = &mail.MailClient{
		Host:         cfg.Services.Mail.Host,
		Password:     cfg.Services.Mail.Password,
		Queue:        mail.NewEmailQueue(),
		DB:           s.DB,
		ShutdownChan: make(chan struct{}),
	}
	log.Printf("Mail client initialized successfully")

	// Typesense is optional - search will still work without it (slower full-text search only)
	log.Printf("Initializing Typesense search client (host=%s, collection=%s)", cfg.Services.Search.Host, cfg.Services.Search.Collection)
	typesenseClient, err := bootstrap.InitTypesense(cfg.Services.Search)
	if err == nil {
		s.TypesenseClient = typesenseClient
		log.Printf("Typesense search client initialized successfully")
		go safeGoroutine(func() {
			log.Printf("Initializing search collection...")
			h.InitSearchCollection()
		})
	} else {
		log.Printf("WARNING: Typesense initialization failed - search functionality is disabled. Error: %v", err)
		log.Printf("INFO: Searches will use slower full-text search only. Check Typesense configuration and network connectivity.")
	}

	s.JwtSecretKey = []byte(cfg.Server.JwtSecretKey)

	// Initialize LLM client
	log.Printf("Initializing LLM client (endpoint=%s, model=%s)", cfg.Services.LLM.Endpoint, cfg.Services.LLM.DefaultModel)
	llmClient := &models.LLMClient{
		Client:      openai.NewClient(cfg.Services.LLM.APIKey),
		Testing:     cfg.Server.DevMode, // Use server dev mode for testing flag
		Model:       cfg.Services.LLM.DefaultModel,
		UserID:      0, // Will be set per request
		DB:          s.DB,
		RequestType: "chat", // Default request type
	}
	s.LLMClient = llmClient
	log.Printf("LLM client initialized successfully")

	if cfg.Services.LLM.ChunkingEnabled {
		go func() {
			start := time.Now()
			elapsed := time.Since(start)
			fmt.Printf("Operation took %v\n", elapsed)
		}()
	}

	// Create a context for shutdown coordination
	shutdownCtx, shutdownCancel := context.WithCancel(context.Background())
	defer shutdownCancel()

	r := mux.NewRouter()

	// Register all API routes
	routes.RegisterAllRoutes(r, h)
	log.Printf("All routes registered successfully")

	c := cors.New(cors.Options{
		AllowedOrigins:   []string{cfg.Server.URL},
		AllowCredentials: true,
		AllowedHeaders:   []string{"authorization", "content-type"},
		AllowedMethods:   []string{"GET", "POST", "PUT", "PATCH", "DELETE"},
		// Enable Debugging for testing, consider disabling in production
		//Debug: true,
	})

	handler := c.Handler(r)

	port := cfg.Server.Port

	srv := &http.Server{
		Addr:    ":" + port,
		Handler: handler,
	}

	stopCh := make(chan os.Signal, 1)
	signal.Notify(stopCh, syscall.SIGINT, syscall.SIGTERM)
	defer signal.Stop(stopCh)

	go func() {
		<-stopCh
		log.Printf("shutdown signal received")

		// Cancel shutdown context to signal all goroutines to stop
		shutdownCancel()

		// Shutdown mail queue (wait for in-flight emails to complete)
		if err := s.Mail.Shutdown(shutdownCtx); err != nil {
			log.Printf("mail queue shutdown error: %v", err)
		}

		// Shutdown HTTP server with timeout
		ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
		defer cancel()

		if err := srv.Shutdown(ctx); err != nil {
			log.Printf("server shutdown error: %v", err)
		}

		log.Printf("shutdown complete")
	}()

	log.Printf("Initialization complete - starting HTTP server on port %s (cors_origin=%s)", port, cfg.Server.URL)
	if err := srv.ListenAndServe(); err != nil && err != http.ErrServerClosed {
		return fmt.Errorf("server failed to start: %w", err)
	}

	return nil
}

func main() {
	if err := run(); err != nil {
		log.Printf("%v", err)
		os.Exit(1)
	}
}
