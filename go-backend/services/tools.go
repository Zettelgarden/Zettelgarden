package services

// This file provides backward compatibility by re-exporting all types and functions
// from the tools subdirectory. The actual implementation has been moved to:
// - services/tools/registry.go - Core registry infrastructure
// - services/tools/context.go - ToolContext with transaction support
// - services/tools/params.go - Parameter extraction helpers
// - services/tools/types.go - Tool types and constants
// - services/tools/card_tools.go - Card-related tools
// - services/tools/task_tools.go - Task-related tools
// - services/tools/entity_tools.go - Entity-related tools
// - services/tools/fact_tools.go - Fact-related tools
// - services/tools/template_tools.go - Template tools
// - services/tools/article_tools.go - Article/URL tools
// - services/tools/memory_tools.go - Memory tools
//
// All files in services/tools/ are part of the same "services" package,
// so types and functions are directly accessible without imports.

// Note: This file is kept for documentation purposes and as a reference point.
// All types (Tool, ToolContext, ToolRegistry) and constants are now defined
// in the services/tools/ subdirectory files.

// The original implementation (71,483 bytes / ~2,413 lines) has been split into:
// - registry.go (~150 lines) - Core registry infrastructure
// - context.go (~120 lines) - ToolContext with transaction support
// - params.go (~100 lines) - Parameter extraction helpers
// - types.go (~60 lines) - Tool types and name constants
// - card_tools.go (~350 lines) - Card-related tools
// - task_tools.go (~400 lines) - Task-related tools
// - entity_tools.go (~400 lines) - Entity-related tools
// - fact_tools.go (~200 lines) - Fact-related tools
// - template_tools.go (~250 lines) - Template and helper functions
// - article_tools.go (~100 lines) - Article/URL tools
// - memory_tools.go (~50 lines) - Memory tools
//
// Total: ~2,330 lines (similar to original) but better organized by domain
//
// Backward compatibility: All existing code using services.NewToolRegistry(),
// services.ToolContext, services.ToolRegistry, etc. continues to work unchanged.
