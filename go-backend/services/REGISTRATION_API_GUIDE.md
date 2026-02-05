# Tool Registration API Guide

## Overview

This document describes the simplified tool registration API introduced in Phase 1.2 of the AI/Agent refactoring epic. The new API reduces tool registration from ~40 lines to ~5-10 lines per tool.

## Files

- **`registration.go`**: Simplified tool registration helpers
- **`types.go`**: Tool type definitions (Tool, ToolHandler, etc.)
- **`registry.go`**: Core tool registry with `registerTool()` method
- **`context.go`**: ToolContext for tool execution
- **`params.go`**: Parameter extraction helpers

## API Reference

### ToolParam Struct

```go
type ToolParam struct {
    Name        string           // Parameter name (required)
    Type        string           // "string", "integer", "boolean", "array", "object" (required)
    Required    bool             // Whether parameter is required (default: false)
    Default     interface{}      // Default value (optional)
    Description string           // Parameter description (or use Desc for brevity)
    Desc        string           // Short alias for Description (takes precedence if set)
    Enum        []interface{}    // Enum values for string types (optional)
    Minimum     *int             // Minimum value for integer types (optional)
    Maximum     *int             // Maximum value for integer types (optional)
    Items       *ToolParam       // For array types - defines item schema (optional)
}
```

### Helper Functions

#### intPtr(i int) *int
Creates a pointer to an int, useful for Minimum/Maximum values.

```go
ToolParam{Name: "limit", Type: "integer", Minimum: intPtr(1), Maximum: intPtr(50)}
```

### RegisterTool Function

```go
func RegisterTool(registry *ToolRegistry, name string, description string, handler ToolHandler, params ...ToolParam)
```

Registers a tool using the simplified API.

## Usage Examples

### Example 1: Simple String Parameter

```go
RegisterTool(registry,
    "search_cards",
    "Search for cards in the user's knowledge base",
    handleSearchCards,
    ToolParam{Name: "query", Type: "string", Required: true, Desc: "Search query"},
)
```

**Before (~30 lines):**
```go
func (tr *ToolRegistry) registerSearchCards() {
    tr.tools["search_cards"] = Tool{
        Definition: openai.Tool{
            Type: openai.ToolTypeFunction,
            Function: &openai.FunctionDefinition{
                Name:        "search_cards",
                Description: "Search for cards in the user's knowledge base",
                Parameters: map[string]interface{}{
                    "type": "object",
                    "properties": map[string]interface{}{
                        "query": map[string]interface{}{
                            "type":        "string",
                            "description": "Search query",
                        },
                    },
                    "required": []string{"query"},
                },
            },
        },
        Handler: handleSearchCards,
    }
}
```

### Example 2: Multiple Parameters with Different Types

```go
RegisterTool(registry,
    "search_cards",
    "Search for cards in the user's knowledge base using text or semantic similarity",
    handleSearchCards,
    ToolParam{Name: "query", Type: "string", Required: true, Desc: "Search query"},
    ToolParam{Name: "limit", Type: "integer", Required: false, Default: 10, Minimum: intPtr(1), Maximum: intPtr(50)},
    ToolParam{Name: "search_type", Type: "string", Required: false, Default: "semantic", Enum: []interface{}{"text", "semantic"}},
    ToolParam{Name: "verbose", Type: "boolean", Required: false, Default: false},
)
```

### Example 3: Array Parameter

```go
RegisterTool(registry,
    "tag_card",
    "Tag a card with multiple tags",
    handleTagCard,
    ToolParam{Name: "card_id", Type: "integer", Required: true, Desc: "Card ID to tag"},
    ToolParam{
        Name:     "tags",
        Type:     "array",
        Required: true,
        Desc:     "List of tags to apply",
        Items: &ToolParam{
            Type: "string",
        },
    },
)
```

### Example 4: Using Description vs Desc

```go
// Desc takes precedence over Description
ToolParam{Name: "query", Type: "string", Desc: "Short desc"}  // Uses "Short desc"
ToolParam{Name: "query", Type: "string", Description: "Long description"}  // Uses "Long description"
ToolParam{Name: "query", Type: "string", Desc: "Short", Description: "Long"}  // Uses "Short" (Desc wins)
```

## Complete Tool Registration Example

Here's a complete example showing how to migrate an existing registration function to the new API:

### Before (Verbose - ~40 lines)

```go
func (tr *ToolRegistry) registerCreateTask() {
    tr.tools["create_task"] = Tool{
        Definition: openai.Tool{
            Type: openai.ToolTypeFunction,
            Function: &openai.FunctionDefinition{
                Name:        "create_task",
                Description: "Create a new task with a title and optional scheduling",
                Parameters: map[string]interface{}{
                    "type": "object",
                    "properties": map[string]interface{}{
                        "title": map[string]interface{}{
                            "type":        "string",
                            "description": "Title of the task (required)",
                        },
                        "scheduled_date": map[string]interface{}{
                            "type":        "string",
                            "description": "Optional scheduled date in ISO 8601 format",
                        },
                        "priority": map[string]interface{}{
                            "type":        "string",
                            "description": "Optional priority level",
                            "enum":        []string{"high", "medium", "low"},
                        },
                    },
                    "required": []string{"title"},
                },
            },
        },
        Handler: handleCreateTask,
    }
}
```

### After (Simplified - ~8 lines)

```go
func (tr *ToolRegistry) registerCreateTask() {
    RegisterTool(tr,
        "create_task",
        "Create a new task with a title and optional scheduling",
        handleCreateTask,
        ToolParam{Name: "title", Type: "string", Required: true, Desc: "Title of the task"},
        ToolParam{Name: "scheduled_date", Type: "string", Required: false, Desc: "Optional scheduled date in ISO 8601 format"},
        ToolParam{Name: "priority", Type: "string", Required: false, Enum: []interface{}{"high", "medium", "low"}},
    )
}
```

## Migration Guide

To migrate existing tool registrations to the new API:

1. Replace the verbose `tr.tools["tool_name"] = Tool{...}` pattern with a call to `RegisterTool()`
2. Convert parameter definitions from nested maps to `ToolParam` structs
3. Use `intPtr()` for Minimum/Maximum integer constraints
4. Use the `Desc` field instead of `Description` for brevity
5. The `registerTool()` method on ToolRegistry is still available for advanced use cases

## Testing

The registration API includes comprehensive tests in `registration_example_test.go`:

- `TestBuildParams`: Verifies correct JSON Schema generation
- `TestIntPtr`: Tests the intPtr helper
- `TestRegisterTool`: Tests end-to-end tool registration

Run tests with:
```bash
cd go-backend && go test -v ./services/ -run TestBuildParams
```

## Implementation Notes

- The `buildParams()` function constructs an OpenAI-compatible parameter map
- Parameter types are validated at compile time (string, integer, boolean, array, object)
- The `Desc` field takes precedence over `Description` for brevity
- Array types support nested item definitions via the `Items` field
- The `required` array is automatically built from parameters with `Required: true`
