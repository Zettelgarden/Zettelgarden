package tools

import (
	"reflect"
	"testing"

	"go-backend/models"
)

func TestWrapToolSuccess(t *testing.T) {
	t.Run("basic success wrapper", func(t *testing.T) {
		data := map[string]interface{}{
			"id":   123,
			"title": "Test Card",
		}

		result := WrapToolSuccess(data)

		// Verify structure
		if result["success"] != true {
			t.Errorf("expected success to be true, got %v", result["success"])
		}

		resultData, ok := result["data"]
		if !ok {
			t.Error("expected 'data' key in result")
			return
		}

		// Verify data contains our input
		dataMap, ok := resultData.(map[string]interface{})
		if !ok {
			t.Errorf("expected data to be map[string]interface{}, got %T", resultData)
			return
		}

		if dataMap["id"] != 123 {
			t.Errorf("expected id 123, got %v", dataMap["id"])
		}
		if dataMap["title"] != "Test Card" {
			t.Errorf("expected title 'Test Card', got %v", dataMap["title"])
		}
	})

	t.Run("success with struct data", func(t *testing.T) {
		type TestData struct {
			Name string `json:"name"`
			Count int    `json:"count"`
		}
		data := TestData{Name: "test", Count: 5}

		result := WrapToolSuccess(data)

		if result["success"] != true {
			t.Errorf("expected success to be true, got %v", result["success"])
		}

		// In Go, structs are not automatically converted to maps
		// The wrapper just passes through the data as-is
		resultData := result["data"]
		dataStruct, ok := resultData.(TestData)
		if !ok {
			t.Errorf("expected data to be TestData struct, got %T", resultData)
			return
		}

		if dataStruct.Name != "test" {
			t.Errorf("expected name 'test', got %v", dataStruct.Name)
		}
		if dataStruct.Count != 5 {
			t.Errorf("expected count 5, got %v", dataStruct.Count)
		}
	})
}

func TestWrapToolSuccessWithMetadata(t *testing.T) {
	t.Run("success with metadata", func(t *testing.T) {
		data := map[string]interface{}{
			"id": 456,
		}
		metadata := NewMetadata(
			WithTotal(100),
			WithOperation("search"),
			WithTool("search_cards"),
		)

		result := WrapToolSuccessWithMetadata(data, metadata)

		if result["success"] != true {
			t.Errorf("expected success to be true, got %v", result["success"])
		}

		resultData := result["data"]
		if resultData == nil {
			t.Error("expected 'data' key in result")
			return
		}

		resultMetadata := result["metadata"]
		if resultMetadata == nil {
			t.Error("expected 'metadata' key in result")
			return
		}

		metadataMap, ok := resultMetadata.(map[string]interface{})
		if !ok {
			t.Errorf("expected metadata to be map[string]interface{}, got %T", resultMetadata)
			return
		}

		if metadataMap["total"] != 100 {
			t.Errorf("expected total 100, got %v", metadataMap["total"])
		}
		if metadataMap["operation"] != "search" {
			t.Errorf("expected operation 'search', got %v", metadataMap["operation"])
		}
		if metadataMap["tool"] != "search_cards" {
			t.Errorf("expected tool 'search_cards', got %v", metadataMap["tool"])
		}
	})

	t.Run("success with nil metadata should not include metadata key", func(t *testing.T) {
		data := map[string]interface{}{
			"id": 789,
		}

		result := WrapToolSuccessWithMetadata(data, nil)

		if result["success"] != true {
			t.Errorf("expected success to be true, got %v", result["success"])
		}

		// Verify metadata key is not present when nil
		if _, exists := result["metadata"]; exists {
			t.Error("expected 'metadata' key to not exist when metadata is nil")
		}

		// Verify data is present
		if result["data"] == nil {
			t.Error("expected 'data' key in result")
		}
	})

	t.Run("success with empty metadata should include metadata key", func(t *testing.T) {
		data := map[string]interface{}{
			"id": 999,
		}
		metadata := NewMetadata() // Empty metadata

		result := WrapToolSuccessWithMetadata(data, metadata)

		if result["success"] != true {
			t.Errorf("expected success to be true, got %v", result["success"])
		}

		// Verify metadata key exists when empty map is provided
		if _, exists := result["metadata"]; !exists {
			t.Error("expected 'metadata' key to exist even when empty")
		}

		metadataMap := result["metadata"].(map[string]interface{})
		if len(metadataMap) != 0 {
			t.Errorf("expected empty metadata map, got %v", metadataMap)
		}
	})
}

func TestWrapToolSuccessWithList(t *testing.T) {
	t.Run("list wrapper with items", func(t *testing.T) {
		items := []map[string]interface{}{
			{"id": 1, "name": "Item 1"},
			{"id": 2, "name": "Item 2"},
			{"id": 3, "name": "Item 3"},
		}

		result := WrapToolSuccessWithList(items)

		if result["success"] != true {
			t.Errorf("expected success to be true, got %v", result["success"])
		}

		// Verify data structure with nested items
		resultData := result["data"]
		dataMap, ok := resultData.(map[string]interface{})
		if !ok {
			t.Errorf("expected data to be map[string]interface{}, got %T", resultData)
			return
		}

		resultItems, ok := dataMap["items"]
		if !ok {
			t.Error("expected 'items' key in data")
			return
		}

		itemsSlice, ok := resultItems.([]map[string]interface{})
		if !ok {
			t.Errorf("expected items to be []map[string]interface{}, got %T", resultItems)
			return
		}

		if len(itemsSlice) != 3 {
			t.Errorf("expected 3 items, got %d", len(itemsSlice))
		}

		// Verify metadata total
		resultMetadata := result["metadata"]
		metadataMap, ok := resultMetadata.(map[string]interface{})
		if !ok {
			t.Errorf("expected metadata to be map[string]interface{}, got %T", resultMetadata)
			return
		}

		if metadataMap["total"] != 3 {
			t.Errorf("expected total 3, got %v", metadataMap["total"])
		}
	})

	t.Run("list wrapper with empty items", func(t *testing.T) {
		items := []map[string]interface{}{}

		result := WrapToolSuccessWithList(items)

		if result["success"] != true {
			t.Errorf("expected success to be true, got %v", result["success"])
		}

		dataMap := result["data"].(map[string]interface{})
		itemsSlice := dataMap["items"].([]map[string]interface{})

		if len(itemsSlice) != 0 {
			t.Errorf("expected 0 items, got %d", len(itemsSlice))
		}

		metadataMap := result["metadata"].(map[string]interface{})
		if metadataMap["total"] != 0 {
			t.Errorf("expected total 0, got %v", metadataMap["total"])
		}
	})

	t.Run("list wrapper with nil items", func(t *testing.T) {
		var items []map[string]interface{} = nil

		result := WrapToolSuccessWithList(items)

		if result["success"] != true {
			t.Errorf("expected success to be true, got %v", result["success"])
		}

		dataMap := result["data"].(map[string]interface{})
		itemsSlice := dataMap["items"].([]map[string]interface{})

		if len(itemsSlice) != 0 {
			t.Errorf("expected 0 items for nil input, got %d", len(itemsSlice))
		}

		metadataMap := result["metadata"].(map[string]interface{})
		if metadataMap["total"] != 0 {
			t.Errorf("expected total 0, got %v", metadataMap["total"])
		}
	})
}

func TestWrapToolError(t *testing.T) {
	t.Run("error wrapper", func(t *testing.T) {
		toolErr := &models.ToolError{
			Type:       models.ToolErrorTypeValidation,
			Message:     "Invalid parameter: card_id is required",
			Retryable:   false,
			ToolName:    "get_card",
			Arguments:   map[string]interface{}{"card_id": ""},
			Suggestion:  "Please provide a valid card_id",
		}

		result := WrapToolError(toolErr)

		if result["success"] != false {
			t.Errorf("expected success to be false, got %v", result["success"])
		}

		// Verify error structure
		resultError := result["error"]
		if resultError == nil {
			t.Error("expected 'error' key in result")
			return
		}

		errorMap, ok := resultError.(map[string]interface{})
		if !ok {
			t.Errorf("expected error to be map[string]interface{}, got %T", resultError)
			return
		}

		// Verify error fields match ToolError.ToMap() format
		if errorMap["type"] != models.ToolErrorTypeValidation {
			t.Errorf("expected error type %s, got %v", models.ToolErrorTypeValidation, errorMap["type"])
		}
		if errorMap["message"] != "Invalid parameter: card_id is required" {
			t.Errorf("expected error message 'Invalid parameter: card_id is required', got %v", errorMap["message"])
		}
		if errorMap["retryable"] != false {
			t.Errorf("expected retryable false, got %v", errorMap["retryable"])
		}
		if errorMap["tool_name"] != "get_card" {
			t.Errorf("expected tool_name 'get_card', got %v", errorMap["tool_name"])
		}
	})

	t.Run("error wrapper with nil arguments", func(t *testing.T) {
		toolErr := &models.ToolError{
			Type:      models.ToolErrorTypeNotFound,
			Message:    "Card not found",
			Retryable:  false,
			ToolName:   "get_card",
		}

		result := WrapToolError(toolErr)

		errorMap := result["error"].(map[string]interface{})
		// ToolError.ToMap() creates a map with arguments key
		// The arguments field itself will be nil in the ToolError
		// but ToMap() creates a key for it
		arguments, hasArguments := errorMap["arguments"]
		if !hasArguments {
			t.Error("expected 'arguments' key in error map")
		}
		// Check if arguments is nil (the value, not the key)
		if arguments != nil && len(arguments.(map[string]interface{})) != 0 {
			t.Errorf("expected arguments to be empty map, got %v", arguments)
		}
	})
}

func TestMetadataHelpers(t *testing.T) {
	t.Run("WithTotal adds total field", func(t *testing.T) {
		metadata := NewMetadata(WithTotal(42))

		if metadata["total"] != 42 {
			t.Errorf("expected total 42, got %v", metadata["total"])
		}
	})

	t.Run("WithOperation adds operation field", func(t *testing.T) {
		metadata := NewMetadata(WithOperation("create"))

		if metadata["operation"] != "create" {
			t.Errorf("expected operation 'create', got %v", metadata["operation"])
		}
	})

	t.Run("WithTool adds tool name field", func(t *testing.T) {
		metadata := NewMetadata(WithTool("get_card"))

		if metadata["tool"] != "get_card" {
			t.Errorf("expected tool 'get_card', got %v", metadata["tool"])
		}
	})

	t.Run("multiple metadata helpers combined", func(t *testing.T) {
		metadata := NewMetadata(
			WithTotal(100),
			WithOperation("search"),
			WithTool("search_cards"),
		)

		if metadata["total"] != 100 {
			t.Errorf("expected total 100, got %v", metadata["total"])
		}
		if metadata["operation"] != "search" {
			t.Errorf("expected operation 'search', got %v", metadata["operation"])
		}
		if metadata["tool"] != "search_cards" {
			t.Errorf("expected tool 'search_cards', got %v", metadata["tool"])
		}
	})

	t.Run("empty NewMetadata", func(t *testing.T) {
		metadata := NewMetadata()

		if len(metadata) != 0 {
			t.Errorf("expected empty metadata map, got %v", metadata)
		}
	})
}

func TestWrapToolSuccessWithEmptyData(t *testing.T) {
	t.Run("nil data", func(t *testing.T) {
		result := WrapToolSuccess(nil)

		if result["success"] != true {
			t.Errorf("expected success to be true, got %v", result["success"])
		}

		if result["data"] != nil {
			t.Errorf("expected data to be nil, got %v", result["data"])
		}
	})

	t.Run("empty map data", func(t *testing.T) {
		data := map[string]interface{}{}
		result := WrapToolSuccess(data)

		if result["success"] != true {
			t.Errorf("expected success to be true, got %v", result["success"])
		}

		dataMap := result["data"].(map[string]interface{})
		if len(dataMap) != 0 {
			t.Errorf("expected empty data map, got %v", dataMap)
		}
	})

	t.Run("empty string data", func(t *testing.T) {
		result := WrapToolSuccess("")

		if result["success"] != true {
			t.Errorf("expected success to be true, got %v", result["success"])
		}

		if result["data"] != "" {
			t.Errorf("expected empty string data, got %v", result["data"])
		}
	})

	t.Run("empty slice data", func(t *testing.T) {
		result := WrapToolSuccess([]string{})

		if result["success"] != true {
			t.Errorf("expected success to be true, got %v", result["success"])
		}

		dataSlice := result["data"].([]string)
		if len(dataSlice) != 0 {
			t.Errorf("expected empty slice data, got %v", dataSlice)
		}
	})
}

func TestWrapToolSuccessWithNilMetadata(t *testing.T) {
	t.Run("WrapToolSuccessWithMetadata with nil metadata", func(t *testing.T) {
		data := map[string]interface{}{
			"id": 123,
		}

		result := WrapToolSuccessWithMetadata(data, nil)

		// Verify success flag
		if result["success"] != true {
			t.Errorf("expected success to be true, got %v", result["success"])
		}

		// Verify data is present
		if result["data"] == nil {
			t.Error("expected 'data' key in result")
			return
		}

		dataMap := result["data"].(map[string]interface{})
		if dataMap["id"] != 123 {
			t.Errorf("expected id 123, got %v", dataMap["id"])
		}

		// Verify metadata key is NOT present when nil
		if _, exists := result["metadata"]; exists {
			t.Error("expected 'metadata' key to not exist when metadata is nil")
		}
	})

	t.Run("WrapToolSuccessWithMetadata with data and no metadata", func(t *testing.T) {
		data := "simple string result"

		result := WrapToolSuccessWithMetadata(data, nil)

		if result["success"] != true {
			t.Errorf("expected success to be true, got %v", result["success"])
		}

		if result["data"] != "simple string result" {
			t.Errorf("expected data 'simple string result', got %v", result["data"])
		}

		if _, exists := result["metadata"]; exists {
			t.Error("expected 'metadata' key to not exist when metadata is nil")
		}
	})
}

// TestStructToMapConversion verifies that structs are handled correctly
func TestStructToMapConversion(t *testing.T) {
	t.Run("struct data is passed through", func(t *testing.T) {
		type TestStruct struct {
			ID       int    `json:"id"`
			Name      string `json:"name"`
			Optional  string `json:"optional,omitempty"`
			Ignored  string `json:"-"`
		}
		data := TestStruct{
			ID:      1,
			Name:    "Test",
			Optional: "",
			Ignored: "should not appear",
		}

		result := WrapToolSuccess(data)

		// In Go, the wrapper doesn't auto-convert structs
		// Structs are passed through as-is
		dataStruct, ok := result["data"].(TestStruct)
		if !ok {
			t.Errorf("expected data to be TestStruct, got %T", result["data"])
			return
		}

		if dataStruct.ID != 1 {
			t.Errorf("expected id 1, got %v", dataStruct.ID)
		}
		if dataStruct.Name != "Test" {
			t.Errorf("expected name 'Test', got %v", dataStruct.Name)
		}
	})
}

// BenchmarkWrapToolSuccess benchmarks the success wrapper performance
func BenchmarkWrapToolSuccess(b *testing.B) {
	data := map[string]interface{}{
		"id":      123,
		"title":    "Test Card",
		"body":     "This is a test card body",
		"tags":     []string{"test", "benchmark"},
		"created": "2024-01-01T00:00:00Z",
	}

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		_ = WrapToolSuccess(data)
	}
}

// BenchmarkWrapToolSuccessWithMetadata benchmarks the metadata wrapper performance
func BenchmarkWrapToolSuccessWithMetadata(b *testing.B) {
	data := map[string]interface{}{"id": 123}
	metadata := NewMetadata(WithTotal(100), WithOperation("search"))

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		_ = WrapToolSuccessWithMetadata(data, metadata)
	}
}

// BenchmarkWrapToolSuccessWithList benchmarks the list wrapper performance
func BenchmarkWrapToolSuccessWithList(b *testing.B) {
	items := make([]map[string]interface{}, 10)
	for i := 0; i < 10; i++ {
		items[i] = map[string]interface{}{
			"id":    i,
			"title": "Item Title",
		}
	}

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		_ = WrapToolSuccessWithList(items)
	}
}

// Helper function to check if two maps are deeply equal (for tests that need it)
func mapsEqual(a, b map[string]interface{}) bool {
	return reflect.DeepEqual(a, b)
}
