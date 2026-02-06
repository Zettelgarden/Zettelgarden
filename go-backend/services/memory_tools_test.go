package services

import (
	"context"
	"database/sql"
	"testing"
)

// TestMemoryToolsRegistration tests that memory tools are registered correctly
func TestMemoryToolsRegistration(t *testing.T) {
	registry := &ToolRegistry{tools: make(map[string]Tool)}
	registry.RegisterMemoryTools()

	// Check that the tool was registered
	tool, exists := registry.tools[ToolGetUserMemory]
	if !exists {
		t.Errorf("tool %s was not registered", ToolGetUserMemory)
		return
	}

	// Verify the tool definition
	if tool.Definition.Function.Name != ToolGetUserMemory {
		t.Errorf("expected tool name %s, got %s", ToolGetUserMemory, tool.Definition.Function.Name)
	}

	// Verify the description contains expected text
	desc := tool.Definition.Function.Description
	if desc == "" {
		t.Error("tool description is empty")
	}
}

// TestMemoryToolsHandler tests that the handler is set up correctly
func TestMemoryToolsHandler(t *testing.T) {
	registry := &ToolRegistry{tools: make(map[string]Tool)}
	registry.RegisterMemoryTools()

	tool, exists := registry.tools[ToolGetUserMemory]
	if !exists {
		t.Fatal("tool not registered")
	}

	if tool.Handler == nil {
		t.Error("handler is nil")
	}
}

// TestToolRegistryWithMemoryTools tests that the full tool registry includes memory tools
func TestToolRegistryWithMemoryTools(t *testing.T) {
	registry := NewToolRegistry()

	// Verify memory tool is registered
	_, exists := registry.tools[ToolGetUserMemory]
	if !exists {
		t.Error("memory tool not found in registry")
	}
}

// TestMemoryToolHandlerContextValidation tests that the tool context validation works correctly
func TestMemoryToolHandlerContextValidation(t *testing.T) {
	ctx := &ToolContext{
		Context: context.Background(),
	}

	// Test with missing required fields
	err := ctx.Validate()
	if err == nil {
		t.Error("expected error for invalid context, got nil")
	}

	// Test with valid context
	ctx.UserID = 1
	ctx.DB = &sql.DB{} // We don't need a real connection for validation test
	err = ctx.Validate()
	if err != nil {
		t.Errorf("expected valid context, got error: %v", err)
	}
}
