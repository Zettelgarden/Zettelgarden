package server

import (
	"database/sql"
	"go-backend/mail"
	"go-backend/models"
	"go-backend/services/storage"
	"go-backend/settings"

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
	LLMClient       *models.LLMClient
	TypesenseClient *typesense.Client

	// Settings is the file-backed admin settings manager (config.yaml next
	// to the SQLite DB). Non-secret admin settings; env seeds it on first
	// boot. See Zettelgarden-6er.15.
	Settings *settings.Manager
}
