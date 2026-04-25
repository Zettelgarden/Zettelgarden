// Package services provides business logic and tool implementations for Zettelgarden.
//
// Tool Registry Infrastructure:
// - registry.go: Core tool registration and execution
// - context.go: Tool execution context with transaction support
// - params.go: Parameter extraction and validation helpers
// - types.go: Tool type definitions and constants
// - registration.go: Simplified tool registration helpers
//
// Domain-specific tools:
// - card_tools.go: Card CRUD operations and search
// - task_tools.go: Task management and scheduling
// - entity_tools.go: Entity management and linking
// - fact_tools.go: Fact extraction and retrieval
// - template_tools.go: Card template management
// - article_tools.go: Article parsing and creation
// - memory_tools.go: User memory operations
package services

// ToolParam defines a tool parameter with a simplified API
type ToolParam struct {
	Name        string
	Type        string  // "string", "integer", "boolean", "array", "object"
	Required    bool
	Default     interface{}
	Description string
	Desc        string // Short alias for Description (takes precedence if set)
	Enum        []interface{}
	Minimum     *int
	Maximum     *int
	Items       *ToolParam // For array types
}

// ToolHandler is the function signature for tool handlers
type ToolHandler func(args map[string]interface{}, ctx *ToolContext) (map[string]interface{}, error)

// intPtr returns a pointer to an int, useful for Minimum/Maximum values
func intPtr(i int) *int {
	return &i
}

// getDescription returns the description for a parameter, preferring Desc over Description
func (p ToolParam) getDescription() string {
	if p.Desc != "" {
		return p.Desc
	}
	return p.Description
}

// buildParamDefinition converts a ToolParam to its JSON Schema representation
func buildParamDefinition(param ToolParam) map[string]interface{} {
	def := map[string]interface{}{
		"type": param.Type,
	}

	// Add description if provided
	if desc := param.getDescription(); desc != "" {
		def["description"] = desc
	}

	// Add default value if provided
	if param.Default != nil {
		def["default"] = param.Default
	}

	// Add enum values if provided
	if len(param.Enum) > 0 {
		def["enum"] = param.Enum
	}

	// Add minimum/maximum for integer types
	if param.Minimum != nil {
		def["minimum"] = *param.Minimum
	}
	if param.Maximum != nil {
		def["maximum"] = *param.Maximum
	}

	// Handle array types with nested items
	if param.Type == "array" && param.Items != nil {
		def["items"] = buildParamDefinition(*param.Items)
	}

	return def
}

// buildParams constructs an OpenAI-compatible parameter map from ToolParam slice
func buildParams(params []ToolParam) map[string]interface{} {
	properties := make(map[string]interface{})
	required := []string{}

	for _, param := range params {
		properties[param.Name] = buildParamDefinition(param)
		if param.Required {
			required = append(required, param.Name)
		}
	}

	result := map[string]interface{}{
		"type":       "object",
		"properties": properties,
	}

	if len(required) > 0 {
		result["required"] = required
	}

	return result
}

// RegisterTool registers a tool using the simplified API
// This reduces tool registration from ~40 lines to ~5-10 lines
//
// Example usage:
//
//	RegisterTool(registry,
//		"search_cards",
//		"Search for cards in the user's knowledge base",
//		handleSearchCards,
//		ToolParam{Name: "query", Type: "string", Required: true, Desc: "Search query"},
//		ToolParam{Name: "limit", Type: "integer", Required: false, Default: 10, Minimum: intPtr(1), Maximum: intPtr(50)},
//		ToolParam{Name: "search_type", Type: "string", Required: false, Default: "semantic", Enum: []interface{}{"text", "semantic"}},
//	)
func RegisterTool(registry *ToolRegistry, name string, description string, handler ToolHandler, params ...ToolParam) {
	registry.registerTool(name, description, buildParams(params), handler)
}
