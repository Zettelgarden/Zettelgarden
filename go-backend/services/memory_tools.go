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
	"fmt"
)

// RegisterMemoryTools registers all memory-related tools
func (tr *ToolRegistry) RegisterMemoryTools() {
	RegisterTool(tr, "get_user_memory",
		"Retrieves your memory and observations about the user. This contains important context about the user's preferences, interests, work style, and past interactions. Use this to personalize responses and maintain continuity across conversations.",
		handleGetUserMemory,
	)
}

// Memory tool handlers

func handleGetUserMemory(args map[string]interface{}, ctx *ToolContext) (map[string]interface{}, error) {
	memory, err := GetUserMemory(ctx.DB, ctx.UserID)
	if err != nil {
		return nil, fmt.Errorf("failed to retrieve user memory: %w", err)
	}

	if memory == "" {
		return map[string]interface{}{
			"memory": "",
			"note":   "No memory has been recorded yet for this user.",
		}, nil
	}

	return map[string]interface{}{
		"memory": memory,
	}, nil
}
