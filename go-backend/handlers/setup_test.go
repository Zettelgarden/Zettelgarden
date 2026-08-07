package handlers

import (
	"go-backend/pkg/config"
	"go-backend/services"
	"go-backend/tests"
)

// NewHandler creates a properly configured Handler for testing.
//
// This is the standard way to set up handler tests. It wraps tests.Setup()
// to get the test server and configures the Handler with all necessary
// dependencies (database connection, server reference, S3 client).
//
// Usage:
//
//	s := handlers.NewHandler()
//	defer tests.Teardown()
//
//	// Use s in your tests...
//	rr := makeCardRequestSuccess(s, t, 1)
//
// The Handler returned by this function has:
//   - DB: Set to the test database connection
//   - Server: Set to the test Server (with Testing=true and transaction configured)
//   - Store: A real tempdir-backed storage.LocalStore (set in tests.Setup),
//     so upload/download routes stream real bytes
//
// All database operations through Handler.GetDB() will use the test transaction,
// which is automatically rolled back in tests.Teardown() for test isolation.
func NewHandler() *Handler {
	S := tests.Setup()
	s := &Handler{
		DB:        S.DB,
		Server:    S,
		JobRunner: services.NewJobRunner(S.DB, nil), // nil processor is safe: execute() recovers the panic, so file-upload tests record the llm_jobs audit row without doing real extraction.
		// GitHub OAuth defaults to enabled in tests; individual tests opt out
		// by setting s.GitHubConfig.Enabled = false.
		GitHubConfig: config.GitHubConfig{Enabled: true},
	}

	return s
}
