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
	// Load and validate all configuration from environment variables
	cfg := config.LoadConfig()

	_, cleanupLogging, err := configureLogging(cfg)
	if err != nil {
		return err
	}
	defer cleanupLogging()

	// Initialize shared server using bootstrap package
	if s = bootstrap.InitServer(cfg.Database); s == nil {
		log.Fatalf("Failed to initialize server")
	}
	if s != nil && s.DB != nil {
		defer func() {
			if err := s.DB.Close(); err != nil {
				log.Printf("error closing database: %v", err)
			}
		}()
	}

	h = &handlers.Handler{
		Server: s,
		DB:     s.DB,
	}

	// Initialize Stripe
	stripe.Key = cfg.Services.Stripe.SecretKey

	s.S3 = h.CreateS3Client()

	s.Mail = &mail.MailClient{
		Host:     cfg.Services.Mail.Host,
		Password: cfg.Services.Mail.Password,
		Queue:    mail.NewEmailQueue(),
		DB:       s.DB,
	}

	// Typesense is optional - search will still work without it (slower full-text search only)
	typesenseClient, err := bootstrap.InitTypesense(cfg.Services.Search)
	if err == nil {
		s.TypesenseClient = typesenseClient
		go func() {
			log.Printf("updating typesense")
			h.InitSearchCollection()
		}()
	} else {
		log.Printf("WARNING: Typesense initialization failed - search functionality is disabled. Error: %v", err)
		log.Printf("INFO: Searches will use slower full-text search only. Check Typesense configuration and network connectivity.")
	}

	log.Printf("email server initialized (host=%q)", s.Mail.Host)
	s.JwtSecretKey = []byte(cfg.Server.JwtSecretKey)

	// Initialize LLM client
	llmClient := &models.LLMClient{
		Client:      openai.NewClient(cfg.Services.LLM.APIKey),
		Testing:     cfg.Server.DevMode, // Use server dev mode for testing flag
		Model:       cfg.Services.LLM.DefaultModel,
		UserID:      0, // Will be set per request
		DB:          s.DB,
		RequestType: "chat", // Default request type
	}
	s.LLMClient = llmClient

	if cfg.Services.LLM.ChunkingEnabled {
		go func() {
			start := time.Now()
			elapsed := time.Since(start)
			fmt.Printf("Operation took %v\n", elapsed)
		}()
	}

	r := mux.NewRouter()

	// Register all API routes
	routes.RegisterAllRoutes(r, h)

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

		ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
		defer cancel()

		if err := srv.Shutdown(ctx); err != nil {
			log.Printf("server shutdown error: %v", err)
		}
	}()

	log.Printf("Starting server on port %s", port)
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
