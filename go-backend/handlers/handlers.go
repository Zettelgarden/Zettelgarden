package handlers

import (
	"database/sql"
	"go-backend/server"
	"go-backend/services"
	"sync"
)

type Handler struct {
	DB             *sql.DB
	Server         *server.Server
	ToolRetry      *services.ToolCircuitBreaker
	messageMutexes sync.Map // map[string]*sync.Mutex - per-message mutexes
}

// getMessageMutex gets or creates a mutex for a specific message
func (s *Handler) getMessageMutex(messageID string) *sync.Mutex {
	mu, _ := s.messageMutexes.LoadOrStore(messageID, &sync.Mutex{})
	return mu.(*sync.Mutex)
}

// cleanupMessageMutex removes a mutex after a message is completed/failed
func (s *Handler) cleanupMessageMutex(messageID string) {
	s.messageMutexes.Delete(messageID)
}
