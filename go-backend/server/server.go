package server

import (
	"database/sql"
	"go-backend/mail"
	"go-backend/models"

	"github.com/aws/aws-sdk-go-v2/service/s3"
	"github.com/typesense/typesense-go/typesense"
)

type Server struct {
	DB              *sql.DB
	Tx              *sql.Tx // Test transaction for rollback-based test isolation
	S3              *s3.Client
	Testing         bool
	JwtSecretKey    []byte
	StripeKey       string
	Mail            *mail.MailClient
	TestInspector   *TestInspector
	SchemaDir       string
	LLMClient       *models.LLMClient
	TypesenseClient *typesense.Client
}

type TestInspector struct {
	FilesUploaded int
}
