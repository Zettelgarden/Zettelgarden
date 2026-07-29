package server

import (
	"database/sql"
	"go-backend/mail"
	"go-backend/models"
	"go-backend/services/storage"

	"github.com/typesense/typesense-go/typesense"
)

type Server struct {
	DB              *sql.DB
	Tx              *sql.Tx // Test transaction for rollback-based test isolation
	Store           storage.Store
	Testing         bool
	JwtSecretKey    []byte
	StripeKey       string
	Mail            *mail.MailClient
	SchemaDir       string
	Driver          string // "postgres" (default/empty) or "sqlite"
	LLMClient       *models.LLMClient
	TypesenseClient *typesense.Client
}
