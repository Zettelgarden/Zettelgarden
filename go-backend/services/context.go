// Package services provides business logic and tool implementations for Zettelgarden.
//
// Tool Registry Infrastructure:
// - registry.go: Core tool registration and execution
// - context.go: Tool execution context with transaction support
// - params.go: Parameter extraction and validation helpers
// - types.go: Tool type definitions and constants
//
// Domain-specific tools:
// - card_tools.go: Card CRUD operations and search
// - task_tools.go: Task management and scheduling
// - entity_tools.go: Entity management and linking
// - fact_tools.go: Fact extraction and retrieval
// - template_tools.go: Card template management
// - calendar_tools.go: Calendar integration
// - article_tools.go: Article parsing and creation
// - memory_tools.go: User memory operations
package services

import (
	"context"
	"database/sql"
	"fmt"
	"log/slog"
	"time"

	"github.com/typesense/typesense-go/typesense"
)

// ToolContext contains all the context needed for tool execution
type ToolContext struct {
	UserID          int
	DB              *sql.DB
	TypesenseClient *typesense.Client
	ConversationID  *string
	MessageID       *string
	Model           string
	Context         context.Context // Request context for timeout/cancellation
	// Observability fields
	RequestID string   // Request ID for tracing
	Logger    *slog.Logger // Structured logger
	// Transaction support
	Tx *sql.Tx // Optional transaction for operations
}

// BeginTransaction starts a new database transaction
func (c *ToolContext) BeginTransaction() error {
	if c.DB == nil {
		return fmt.Errorf("database connection is nil")
	}

	tx, err := c.DB.BeginTx(c.Context, nil)
	if err != nil {
		return fmt.Errorf("failed to begin transaction: %w", err)
	}
	c.Tx = tx
	return nil
}

// CommitTransaction commits the current transaction
func (c *ToolContext) CommitTransaction() error {
	if c.Tx == nil {
		return nil // No transaction to commit
	}
	err := c.Tx.Commit()
	c.Tx = nil // Clear the transaction reference
	return err
}

// RollbackTransaction rolls back the current transaction
func (c *ToolContext) RollbackTransaction() error {
	if c.Tx == nil {
		return nil // No transaction to rollback
	}
	err := c.Tx.Rollback()
	c.Tx = nil // Clear the transaction reference
	return err
}

// Validate checks if the context has all required fields
func (c *ToolContext) Validate() error {
	if c.UserID == 0 {
		return fmt.Errorf("UserID is required")
	}
	if c.DB == nil {
		return fmt.Errorf("DB connection is required")
	}
	if c.Context == nil {
		return fmt.Errorf("context is required")
	}
	return nil
}

// LogToolExecution logs tool execution with structured logging
func (c *ToolContext) LogToolExecution(toolName string, args map[string]interface{}, duration time.Duration, err error) {
	if c.Logger == nil {
		return // No logger configured
	}

	logArgs := []any{
		"request_id", c.RequestID,
		"tool", toolName,
		"duration_ms", duration.Milliseconds(),
		"user_id", c.UserID,
	}

	if err != nil {
		c.Logger.Error("tool_execution_failed", append(logArgs, "error", err)...)
	} else {
		c.Logger.Info("tool_execution_success", logArgs...)
	}
}

// WithTx returns a new ToolContext with the given transaction
func (c *ToolContext) WithTx(tx *sql.Tx) *ToolContext {
	newCtx := *c
	newCtx.Tx = tx
	return &newCtx
}

// WithRequestID returns a new ToolContext with the given request ID
func (c *ToolContext) WithRequestID(requestID string) *ToolContext {
	newCtx := *c
	newCtx.RequestID = requestID
	return &newCtx
}

// WithLogger returns a new ToolContext with the given logger
func (c *ToolContext) WithLogger(logger *slog.Logger) *ToolContext {
	newCtx := *c
	newCtx.Logger = logger
	return &newCtx
}
