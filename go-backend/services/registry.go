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
	"encoding/json"
	"fmt"
	"log"
	"time"

	openai "github.com/sashabaranov/go-openai"
)

// ToolRegistry holds all available tools
type ToolRegistry struct {
	tools map[string]Tool
}

// NewToolRegistry creates a new tool registry with all available tools
func NewToolRegistry() *ToolRegistry {
	registry := &ToolRegistry{
		tools: make(map[string]Tool),
	}

	// Register all tools
	registry.RegisterCardTools()
	registry.RegisterTaskTools()
	registry.RegisterEntityTools()
	registry.RegisterFactTools()
	registry.RegisterTemplateTools()
	registry.RegisterCalendarTools()
	registry.RegisterArticleTools()
	registry.RegisterMemoryTools()

	return registry
}

// GetToolDefinitions returns all tool definitions for OpenAI API
func (tr *ToolRegistry) GetToolDefinitions() []openai.Tool {
	var tools []openai.Tool
	for _, tool := range tr.tools {
		tools = append(tools, tool.Definition)
	}
	return tools
}

// ExecuteTool executes a tool by name with given arguments
func (tr *ToolRegistry) ExecuteTool(name string, args map[string]interface{}, ctx *ToolContext) (map[string]interface{}, error) {
	// Validate the tool context before execution
	if err := ctx.Validate(); err != nil {
		return nil, fmt.Errorf("invalid tool context: %w", err)
	}

	tool, exists := tr.tools[name]
	if !exists {
		return nil, fmt.Errorf("tool %s not found", name)
	}

	start := time.Now()
	result, execErr := tool.Handler(args, ctx)
	executionTime := time.Since(start)

	// Use structured logging if available
	ctx.LogToolExecution(name, args, executionTime, execErr)

	// Log tool execution for analytics (with timeout to prevent goroutine leaks)
	go func() {
		// Create a context with timeout for logging
		logCtx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
		defer cancel()

		logErr := logToolExecution(ctx.DB, ctx.UserID, name, args, result, int(executionTime.Milliseconds()), execErr, ctx.ConversationID, ctx.MessageID)
		if logErr != nil {
			if logCtx.Err() == context.DeadlineExceeded {
				log.Printf("Timeout logging tool execution for %s", name)
			} else {
				log.Printf("Error logging tool execution: %v", logErr)
			}
		}
	}()

	return result, execErr
}

// registerTool is a helper for registering tools with less boilerplate
func (tr *ToolRegistry) registerTool(name, description string, params map[string]interface{}, handler func(map[string]interface{}, *ToolContext) (map[string]interface{}, error)) {
	tr.tools[name] = Tool{
		Definition: openai.Tool{
			Type: openai.ToolTypeFunction,
			Function: &openai.FunctionDefinition{
				Name:        name,
				Description: description,
				Parameters:  params,
			},
		},
		Handler: handler,
	}
}

// logToolExecution logs tool execution to the database
func logToolExecution(db *sql.DB, userID int, toolName string, args map[string]interface{}, result map[string]interface{}, executionTimeMs int, execErr error, conversationID, messageID *string) error {
	argsJSON, _ := json.Marshal(args)
	resultJSON, _ := json.Marshal(result)

	query := `
		INSERT INTO chat_tool_calls (id, user_id, conversation_id, message_id, tool_name, tool_arguments, tool_result, execution_time_ms, created_at)
		VALUES (gen_random_uuid(), $1, $2, $3, $4, $5, $6, $7, NOW())
	`
	fmt.Printf("Tool call: %v\n", toolName)

	_, err := db.Exec(query, userID, conversationID, messageID, toolName, argsJSON, resultJSON, executionTimeMs)
	return err
}
