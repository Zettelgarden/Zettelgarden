package handlers

import (
	"go-backend/handlers/chat_agent"
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
// - DB: Set to the test database connection
// - Server: Set to the test Server (with Testing=true and transaction configured)
// - S3: Configured with a mock S3 client for testing
//
// All database operations through Handler.GetDB() will use the test transaction,
// which is automatically rolled back in tests.Teardown() for test isolation.
func NewHandler() *Handler {
	S := tests.Setup()
	s := &Handler{
		DB:     S.DB,
		Server: S,
	}

	S.S3 = s.CreateS3Client()
	s.ChatService = chat_agent.NewChatService(s.DB, s.Server)
	return s
}
