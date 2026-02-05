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
//
// PHASE 3: Domain Package Migration with Feature Flags
// ---------------------------------------------------
// This file now supports both legacy and new domain package registration
// controlled by the FeatureFlagMemoryTools feature flag.
//
// - Feature flag DISABLED (default): Uses legacy registration in this file
// - Feature flag ENABLED: Uses services/tools/memory package
//
// To enable the new domain package:
//   export ZETTELGARDEN_FEATURE_MEMORY_TOOLS_V2=true
//
// Migration Status: memory_tools is the proof of concept for domain packages.
package services

import (
	"fmt"

	"go-backend/services/featureflags"
	"go-backend/services/tools/memory"
)

// RegisterMemoryTools registers all memory-related tools.
//
// This method supports two registration paths controlled by feature flag:
// 1. Legacy path (default): Registers tools directly from this file
// 2. New path (feature flag): Delegates to services/tools/memory package
func (tr *ToolRegistry) RegisterMemoryTools() {
	if featureflags.IsEnabled(featureflags.FeatureFlagMemoryTools) {
		// NEW: Use the domain package (imported to trigger side-effect registration)
		// The memory package will handle registration via its init function
		// or we call RegisterMemoryTools directly
		tr.registerMemoryToolsV2()
	} else {
		// LEGACY: Use the original registration in this file
		tr.registerMemoryToolsLegacy()
	}
}

// registerMemoryToolsV2 uses the new memory domain package
func (tr *ToolRegistry) registerMemoryToolsV2() {
	RegisterTool(tr, ToolGetUserMemory,
		"Retrieves your memory and observations about the user. This contains important context about the user's preferences, interests, work style, and past interactions. Use this to personalize responses and maintain continuity across conversations.",
		handleGetUserMemoryV2,
	)
}

// registerMemoryToolsLegacy is the original memory tools registration.
// Kept for backward compatibility and as a fallback.
func (tr *ToolRegistry) registerMemoryToolsLegacy() {
	RegisterTool(tr, ToolGetUserMemory,
		"Retrieves your memory and observations about the user. This contains important context about the user's preferences, interests, work style, and past interactions. Use this to personalize responses and maintain continuity across conversations.",
		handleGetUserMemoryLegacy,
	)
}

// V2 memory tool handler (uses domain package logic)

func handleGetUserMemoryV2(args map[string]interface{}, ctx *ToolContext) (map[string]interface{}, error) {
	// Call the domain package function directly
	userMemory, err := memory.GetUserMemory(ctx.DB, ctx.UserID)
	if err != nil {
		return nil, fmt.Errorf("failed to retrieve user memory: %w", err)
	}

	if userMemory == "" {
		return map[string]interface{}{
			"memory": "",
			"note":   "No memory has been recorded yet for this user.",
		}, nil
	}

	return map[string]interface{}{
		"memory": userMemory,
	}, nil
}

// Legacy memory tool handler (kept for backward compatibility)

func handleGetUserMemoryLegacy(args map[string]interface{}, ctx *ToolContext) (map[string]interface{}, error) {
	// Also use domain package - the legacy refers to the registration pattern,
	// not the implementation
	userMemory, err := memory.GetUserMemory(ctx.DB, ctx.UserID)
	if err != nil {
		return nil, fmt.Errorf("failed to retrieve user memory: %w", err)
	}

	if userMemory == "" {
		return map[string]interface{}{
			"memory": "",
			"note":   "No memory has been recorded yet for this user.",
		}, nil
	}

	return map[string]interface{}{
		"memory": userMemory,
	}, nil
}
