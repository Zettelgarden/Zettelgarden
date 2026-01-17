package main

import (
	"context"
	"fmt"
	"go-backend/bootstrap"
	"go-backend/handlers"
	"go-backend/mail"
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

func configureLogging() (*os.File, func(), error) {
	if os.Getenv("ZETTEL_DEV") == "true" {
		return nil, func() {}, nil
	}

	logPath := os.Getenv("ZETTEL_BACKEND_LOG_LOCATION")
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
	_, cleanupLogging, err := configureLogging()
	if err != nil {
		return err
	}
	defer cleanupLogging()

	// Initialize shared server using bootstrap package
	s = bootstrap.InitServer()
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
	stripe.Key = os.Getenv("STRIPE_SECRET_KEY")

	s.S3 = h.CreateS3Client()

	s.Mail = &mail.MailClient{
		Host:     os.Getenv("MAIL_HOST"),
		Password: os.Getenv("MAIL_PASSWORD"),
		Queue:    mail.NewEmailQueue(),
		DB:       s.DB,
	}

	typesenseClient, err := bootstrap.InitTypesense()
	if err == nil {
		s.TypesenseClient = typesenseClient
		go func() {
			log.Printf("updating typesense")
			h.InitSearchCollection()
		}()
	}

	log.Printf("email server initialized (host=%q)", s.Mail.Host)
	s.JwtSecretKey = []byte(os.Getenv("SECRET_KEY"))
	config := openai.DefaultConfig(os.Getenv("ZETTEL_LLM_KEY"))
	config.BaseURL = os.Getenv("ZETTEL_LLM_ENDPOINT")

	if os.Getenv("ZETTEL_RUN_CHUNKING_EMBEDDING") == "true" {
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
		AllowedOrigins:   []string{os.Getenv("ZETTEL_URL")},
		AllowCredentials: true,
		AllowedHeaders:   []string{"authorization", "content-type"},
		AllowedMethods:   []string{"GET", "POST", "PUT", "PATCH", "DELETE"},
		// Enable Debugging for testing, consider disabling in production
		//Debug: true,
	})

	handler := c.Handler(r)

	port := os.Getenv("ZETTEL_PORT")
	if port == "" {
		port = "8080"
	}

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
