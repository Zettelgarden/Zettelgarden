package tools

import (
	"go-backend/models"
)

// WrapToolSuccess creates a standardized success response for tool results.
// It wraps the provided data in a map with a success: true flag.
//
// Parameters:
//   data: The result data from the tool (can be any type: map, struct, slice, etc.)
//
// Returns:
//   A map with structure: {success: true, data: data}
//
// Example:
//
//	result := WrapToolSuccess(map[string]interface{}{
//	    "id": 123,
//	    "title": "My Card",
//	})
//	// Returns: {success: true, data: {id: 123, title: "My Card"}}
func WrapToolSuccess(data interface{}) map[string]interface{} {
	return map[string]interface{}{
		"success": true,
		"data":    data,
	}
}

// WrapToolSuccessWithMetadata creates a standardized success response with optional metadata.
// It includes metadata only if the metadata parameter is not nil.
//
// Parameters:
//   data: The result data from the tool
//   metadata: Optional metadata map with additional information (total, operation, tool, etc.)
//
// Returns:
//   A map with structure:
//   - With metadata: {success: true, data: data, metadata: metadata}
//   - Without metadata: {success: true, data: data}
//
// Example:
//
//	result := WrapToolSuccessWithMetadata(
//	    map[string]interface{}{"id": 123},
//	    NewMetadata(WithTotal(100), WithOperation("search")),
//	)
//	// Returns: {success: true, data: {id: 123}, metadata: {total: 100, operation: "search"}}
func WrapToolSuccessWithMetadata(data interface{}, metadata map[string]interface{}) map[string]interface{} {
	result := map[string]interface{}{
		"success": true,
		"data":    data,
	}

	// Only include metadata if it's not nil
	if metadata != nil {
		result["metadata"] = metadata
	}

	return result
}

// WrapToolSuccessWithList creates a standardized success response for list results.
// It wraps items in a nested structure with automatic total count.
//
// Parameters:
//   items: A slice of item maps (or nil/empty for no results)
//
// Returns:
//   A map with structure:
//   {success: true, data: {items: items}, metadata: {total: len(items)}}
//
// Example:
//
//	items := []map[string]interface{}{
//	    {"id": 1, "name": "First"},
//	    {"id": 2, "name": "Second"},
//	}
//	result := WrapToolSuccessWithList(items)
//	// Returns: {success: true, data: {items: [...]}, metadata: {total: 2}}
//
// This function handles nil items gracefully, returning an empty slice and total: 0.
func WrapToolSuccessWithList(items []map[string]interface{}) map[string]interface{} {
	// Handle nil items by converting to empty slice
	if items == nil {
		items = []map[string]interface{}{}
	}

	total := len(items)

	return map[string]interface{}{
		"success": true,
		"data": map[string]interface{}{
			"items": items,
		},
		"metadata": map[string]interface{}{
			"total": total,
		},
	}
}

// WrapToolError creates a standardized error response for tool failures.
// It wraps the ToolError in the standardized error format.
//
// Parameters:
//   toolErr: A ToolError struct with error details
//
// Returns:
//   A map with structure: {success: false, error: toolErr.ToMap()["error"]}
//
// Example:
//
//	err := &models.ToolError{
//	    Type:      models.ToolErrorTypeValidation,
//	    Message:    "card_id is required",
//	    Retryable:  false,
//	    ToolName:   "get_card",
//	}
//	result := WrapToolError(err)
//	// Returns: {success: false, error: {type: "validation", message: "card_id is required", ...}}
func WrapToolError(toolErr *models.ToolError) map[string]interface{} {
	return map[string]interface{}{
		"success": false,
		"error":   toolErr.ToMap()["error"],
	}
}

// MetadataOption is a functional option for building metadata maps.
// It takes a metadata map and returns it with additional fields.
type MetadataOption func(map[string]interface{}) map[string]interface{}

// NewMetadata creates a metadata map by applying zero or more functional options.
// This uses the functional options pattern for flexible, readable metadata construction.
//
// Parameters:
//   pairs: Zero or more MetadataOption functions that modify the metadata map
//
// Returns:
//   A map[string]interface{} with all applied options
//
// Example:
//
//	metadata := NewMetadata(
//	    WithTotal(100),
//	    WithOperation("search"),
//	    WithTool("search_cards"),
//	)
//	// Returns: {total: 100, operation: "search", tool: "search_cards"}
//
// Example with no options (empty metadata):
//
//	metadata := NewMetadata()
//	// Returns: {}
func NewMetadata(pairs ...MetadataOption) map[string]interface{} {
	metadata := make(map[string]interface{})

	for _, option := range pairs {
		metadata = option(metadata)
	}

	return metadata
}

// WithTotal adds a total count field to metadata.
// This is typically used for list results to indicate the total number of items.
//
// Parameters:
//   count: The total count to include in metadata
//
// Example:
//
//	metadata := NewMetadata(WithTotal(42))
//	// Returns: {total: 42}
func WithTotal(count int) MetadataOption {
	return func(metadata map[string]interface{}) map[string]interface{} {
		metadata["total"] = count
		return metadata
	}
}

// WithOperation adds an operation name field to metadata.
// This indicates what type of operation was performed (create, update, search, etc.).
//
// Parameters:
//   operation: The operation name to include in metadata
//
// Example:
//
//	metadata := NewMetadata(WithOperation("create"))
//	// Returns: {operation: "create"}
func WithOperation(operation string) MetadataOption {
	return func(metadata map[string]interface{}) map[string]interface{} {
		metadata["operation"] = operation
		return metadata
	}
}

// WithTool adds a tool name field to metadata.
// This indicates which tool generated the result.
//
// Parameters:
//   toolName: The name of the tool to include in metadata
//
// Example:
//
//	metadata := NewMetadata(WithTool("get_card"))
//	// Returns: {tool: "get_card"}
func WithTool(toolName string) MetadataOption {
	return func(metadata map[string]interface{}) map[string]interface{} {
		metadata["tool"] = toolName
		return metadata
	}
}
