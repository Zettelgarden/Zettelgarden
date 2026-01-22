package handlers

import (
	"database/sql"
	"go-backend/server"
	"go-backend/services"
)

type Handler struct {
	DB             *sql.DB
	Server         *server.Server
	ToolRetry      *services.ToolCircuitBreaker
}
