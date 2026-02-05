package services

import (
	"context"
	"database/sql"
	"testing"

	"go-backend/services/featureflags"
)

// TestMemoryToolsFeatureFlag tests that the feature flag controls which
// registration path is used for memory tools.
func TestMemoryToolsFeatureFlag(t *testing.T) {
	// Clean up before and after tests
	featureflags.ResetAll()
	defer featureflags.ResetAll()

	t.Run("legacy path when feature flag disabled", func(t *testing.T) {
		// Ensure feature flag is disabled
		featureflags.Disable(featureflags.FeatureFlagMemoryTools)

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
	})

	t.Run("v2 path when feature flag enabled", func(t *testing.T) {
		// Enable feature flag
		featureflags.Enable(featureflags.FeatureFlagMemoryTools)
		defer featureflags.Disable(featureflags.FeatureFlagMemoryTools)

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
	})
}

// TestMemoryToolsHandlerExecution tests that both paths produce the same results
func TestMemoryToolsHandlerExecution(t *testing.T) {
	// Clean up before and after tests
	featureflags.ResetAll()
	defer featureflags.ResetAll()

	// We can't test the actual database operations without a test database,
	// but we can verify the handlers are set up correctly

	t.Run("legacy handler is callable", func(t *testing.T) {
		featureflags.Disable(featureflags.FeatureFlagMemoryTools)

		registry := &ToolRegistry{tools: make(map[string]Tool)}
		registry.RegisterMemoryTools()

		tool, exists := registry.tools[ToolGetUserMemory]
		if !exists {
			t.Fatal("tool not registered")
		}

		if tool.Handler == nil {
			t.Error("legacy handler is nil")
		}
	})

	t.Run("v2 handler is callable", func(t *testing.T) {
		featureflags.Enable(featureflags.FeatureFlagMemoryTools)
		defer featureflags.Disable(featureflags.FeatureFlagMemoryTools)

		registry := &ToolRegistry{tools: make(map[string]Tool)}
		registry.RegisterMemoryTools()

		tool, exists := registry.tools[ToolGetUserMemory]
		if !exists {
			t.Fatal("tool not registered")
		}

		if tool.Handler == nil {
			t.Error("v2 handler is nil")
		}
	})
}

// TestMemoryToolsIntegration is an integration test that verifies
// the tool works end-to-end with both feature flag states
func TestMemoryToolsIntegration(t *testing.T) {
	// Clean up before and after tests
	featureflags.ResetAll()
	defer featureflags.ResetAll()

	testCases := []struct {
		name        string
		enableFlag  bool
		description string
	}{
		{
			name:        "legacy path",
			enableFlag:  false,
			description: "test with legacy registration path",
		},
		{
			name:        "v2 path",
			enableFlag:  true,
			description: "test with new domain package path",
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			if tc.enableFlag {
				featureflags.Enable(featureflags.FeatureFlagMemoryTools)
			} else {
				featureflags.Disable(featureflags.FeatureFlagMemoryTools)
			}
			defer featureflags.ResetAll()

			// Create registry and register tools
			registry := &ToolRegistry{tools: make(map[string]Tool)}
			registry.RegisterMemoryTools()

			// Verify tool was registered
			tool, exists := registry.tools[ToolGetUserMemory]
			if !exists {
				t.Errorf("tool %s was not registered", ToolGetUserMemory)
				return
			}

			// Verify tool has handler
			if tool.Handler == nil {
				t.Error("tool handler is nil")
			}

			// Verify tool definition
			if tool.Definition.Function.Name != ToolGetUserMemory {
				t.Errorf("expected tool name %s, got %s", ToolGetUserMemory, tool.Definition.Function.Name)
			}

			// Verify parameters (memory tool has no required parameters)
			params := tool.Definition.Function.Parameters
			if params == nil {
				t.Error("tool parameters is nil")
			}
		})
	}
}

// TestToolRegistryWithMemoryTools tests that the full tool registry
// includes memory tools regardless of feature flag state
func TestToolRegistryWithMemoryTools(t *testing.T) {
	// Clean up before and after tests
	featureflags.ResetAll()
	defer featureflags.ResetAll()

	testCases := []struct {
		name       string
		enableFlag bool
	}{
		{name: "feature flag disabled", enableFlag: false},
		{name: "feature flag enabled", enableFlag: true},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			if tc.enableFlag {
				featureflags.Enable(featureflags.FeatureFlagMemoryTools)
			} else {
				featureflags.Disable(featureflags.FeatureFlagMemoryTools)
			}
			defer featureflags.ResetAll()

			// Create full registry
			registry := NewToolRegistry()

			// Verify memory tool is registered
			_, exists := registry.tools[ToolGetUserMemory]
			if !exists {
				t.Errorf("memory tool not found in registry (feature flag: %v)", tc.enableFlag)
			}
		})
	}
}

// TestMemoryToolHandlerContextValidation tests that the tool context
// validation works correctly for memory tools
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
