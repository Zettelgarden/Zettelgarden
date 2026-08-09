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
	"go-backend/services/jobs"
	"go-backend/services/storage"
	"go-backend/settings"

	"log"
	"net/http"
	"os"
	"os/signal"
	"strconv"
	"syscall"
	"time"

	"github.com/gorilla/mux"
	"github.com/rs/cors"
	openai "github.com/sashabaranov/go-openai"
	"github.com/stripe/stripe-go/v82"
)

var s *server.Server
var h *handlers.Handler

// getEnvInt gets an integer environment variable with a fallback
func getEnvInt(key string, defaultValue int) int {
	if val := os.Getenv(key); val != "" {
		if intVal, err := strconv.Atoi(val); err == nil {
			return intVal
		}
	}
	return defaultValue
}

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
	log.Printf("Initializing SQLite database (path=%s)", cfg.Database.SQLitePath)
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
		Server: s,
		DB:     s.DB,
	}

	// Initialize the file-backed admin settings manager (config.yaml next to
	// the SQLite DB). Env seeds it on first boot; the file is the source of
	// truth afterwards (hot-reloaded on change). See Zettelgarden-6er.15.
	settingsPath := settings.DefaultPath(cfg.Database.SQLitePath)
	sm, err := settings.New(settingsPath)
	if err != nil {
		log.Fatalf("Failed to load settings file %s: %v", settingsPath, err)
	}
	s.Settings = sm
	h.Settings = sm
	log.Printf("Settings loaded from %s", settingsPath)
	h.GitHubConfig = cfg.Services.GitHub
	if h.GitHubConfig.Enabled {
		log.Printf("GitHub OAuth enabled")
	} else {
		log.Printf("GitHub OAuth disabled (set GITHUB_AUTH_ENABLED=true to enable)")
	}
	h.OIDCConfig = cfg.Services.OIDC
	if h.OIDCConfig.Enabled {
		log.Printf("OIDC SSO enabled (issuer=%s)", h.OIDCConfig.Issuer)
	} else {
		log.Printf("OIDC SSO disabled (set OIDC_ENABLED=true to enable)")
	}
	h.StripeConfig = cfg.Services.Stripe

	// Initialize Stripe (billing is enabled by default; set STRIPE_ENABLED
	// =false to run an instance without payment processing)
	if cfg.Services.Stripe.Enabled {
		log.Printf("Initializing Stripe payment processing")
		stripe.Key = cfg.Services.Stripe.SecretKey
		log.Printf("Stripe initialized (billing_url=%s)", cfg.Services.Stripe.BillingURL)
	} else {
		log.Printf("Stripe billing disabled (set STRIPE_ENABLED=true to enable)")
	}

	// Initialize local file storage for uploaded files (replaces B2/S3)
	log.Printf("Initializing file storage (dir=%s)", cfg.Services.Storage.Dir)
	fileStore, err := storage.NewLocalStore(cfg.Services.Storage.Dir)
	if err != nil {
		log.Fatalf("Failed to initialize file storage: %v", err)
	}
	s.Store = fileStore
	log.Printf("File storage initialized successfully")

	// Initialize mail client for transactional emails (password resets,
	// reminders). Mail is optional (6er.6): when no SMTP relay is configured
	// the client is a Disabled no-op so callers degrade gracefully. Delivery
	// is direct SMTP (the python-mail service was retired, 6er.12).
	//
	// Hot reload (6er.16): Disabled reflects SMTP *infrastructure* only
	// (host + from configured); the operator toggle mail_enabled is checked
	// per-send via EnabledFn so an admin UI toggle applies immediately,
	// without a restart.
	smtpConfigured := cfg.Services.Mail.SMTPHost != "" && cfg.Services.Mail.SMTPFrom != ""
	if sm.GetBool("mail_enabled") && smtpConfigured {
		log.Printf("Initializing mail client (smtp=%s:%d, from=%s)", cfg.Services.Mail.SMTPHost, cfg.Services.Mail.SMTPPort, cfg.Services.Mail.SMTPFrom)
	} else if !smtpConfigured {
		log.Printf("Mail disabled (no SMTP_HOST/SMTP_FROM configured; set them in the environment to enable)")
	} else {
		log.Printf("Mail enabled but disabled via mail_enabled=false in config.yaml")
	}
	s.Mail = &mail.MailClient{
		SMTPHost:     cfg.Services.Mail.SMTPHost,
		SMTPPort:     cfg.Services.Mail.SMTPPort,
		SMTPUsername: cfg.Services.Mail.SMTPUsername,
		SMTPPassword: cfg.Services.Mail.SMTPPassword,
		SMTPFrom:     cfg.Services.Mail.SMTPFrom,
		StartTLS:     cfg.Services.Mail.StartTLS,
		Queue:        mail.NewEmailQueue(),
		DB:           s.DB,
		ShutdownChan: make(chan struct{}),
		Disabled:     !smtpConfigured,
		EnabledFn:    func() bool { return sm.GetBool("mail_enabled") },
	}
	if smtpConfigured {
		log.Printf("Mail client initialized successfully")
	}

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

	// Initialize LLM job runner (inline processing + audit log)
	log.Printf("Initializing LLM job runner")
	llmJobProcessor := services.NewLLMJobProcessor(s.DB)
	h.JobRunner = services.NewJobRunner(s.DB, llmJobProcessor)

	// Clean up jobs orphaned by a previous run (process crashed/restarted
	// mid-job). Anything still marked "running" is now stale.
	count, err := h.JobRunner.CleanupStale(context.Background())
	if err != nil {
		log.Printf("Failed to cleanup stale jobs: %v", err)
	} else if count > 0 {
		log.Printf("Cleaned up %d stale jobs", count)
	}
	log.Printf("LLM job runner initialized successfully")

	// Initialize and start the scheduled job runner
	log.Printf("Initializing scheduled job runner")
	scheduler := services.NewScheduler(s.DB)

	// Register scheduled jobs with their required dependencies
	scheduler.Register(jobs.NewCleanupJob(s.DB, sm))
	scheduler.Register(jobs.NewTaskRemindersJob(s.DB, s.Mail))
	scheduler.Register(jobs.NewUptimeKumaPingJob())
	scheduler.Register(jobs.NewRSSFetchJob(s.DB))
	scheduler.Register(jobs.NewRSSArticleCleanupJob(s.DB, sm))

	scheduler.Start()
	defer scheduler.Stop()
	log.Printf("[main] scheduled job runner started")

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
	routes.RegisterAllRoutes(r, h, scheduler)
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

		// Shutdown scheduler
		log.Printf("[main] shutting down scheduler...")
		scheduler.Stop()

		// Shutdown legacy mail queue (wait for in-flight emails to complete)
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
