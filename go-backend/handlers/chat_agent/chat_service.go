package chat_agent

import (
	"database/sql"
	"sync"

	"go-backend/server"
	"go-backend/services"
)

// ChatService handles all chat-related operations with LLM integration.
// It provides dependency injection for better testability and separation of concerns.
//
// During the migration phase, some methods still need access to the full Handler
// for database operations. This will be resolved as the refactoring progresses.
type ChatService struct {
	DB             *sql.DB
	Server         *server.Server
	messageMutexes sync.Map // map[string]*sync.Mutex - per-message mutexes

	// ToolRegistry for tool execution (lazy loaded)
	toolRegistry *services.ToolRegistry
}

// NewChatService creates a new ChatService with the given dependencies.
func NewChatService(db *sql.DB, srv *server.Server) *ChatService {
	return &ChatService{
		DB:             db,
		Server:         srv,
		messageMutexes: sync.Map{},
	}
}

// getMessageMutex gets or creates a mutex for a specific message ID
func (s *ChatService) getMessageMutex(messageID string) *sync.Mutex {
	mu, _ := s.messageMutexes.LoadOrStore(messageID, &sync.Mutex{})
	return mu.(*sync.Mutex)
}

// cleanupMessageMutex removes a mutex after a message is completed/failed
func (s *ChatService) cleanupMessageMutex(messageID string) {
	s.messageMutexes.Delete(messageID)
}

// getToolRegistry returns the tool registry (lazy initialization)
func (s *ChatService) getToolRegistry() *services.ToolRegistry {
	if s.toolRegistry == nil {
		s.toolRegistry = services.NewToolRegistry()
	}
	return s.toolRegistry
}
