package main

import (
	//	"bytes"
	//"encoding/json"
	"fmt"
	"go-backend/bootstrap"
	"go-backend/handlers"
	"go-backend/mail"
	"go-backend/routes"
	"go-backend/server"
	"log"
	"net/http"
	"os"
	"time"

	"github.com/gorilla/mux"
	"github.com/rs/cors"
	openai "github.com/sashabaranov/go-openai"
	"github.com/stripe/stripe-go/v82"
)

var s *server.Server
var h *handlers.Handler




func main() {
	// Set up logging based on environment
	if os.Getenv("ZETTEL_DEV") != "true" {
		file, err := handlers.OpenLogFile(os.Getenv("ZETTEL_BACKEND_LOG_LOCATION"))
		if err != nil {
			log.Fatal(err)
		}
		log.SetOutput(file)
	}

	// Initialize shared server using bootstrap package
	s = bootstrap.InitServer()

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
			//			migrations.RunEmbeddings(h)
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
	log.Printf("Starting server on port %s", port)
	if err := http.ListenAndServe(":"+port, handler); err != nil {
		log.Fatal("Server failed to start:", err)
	}
}
