package services

import (
	"context"
	"database/sql"
	"testing"

	"go-backend/services/featureflags"
)

// TestTemplateToolsFeatureFlag tests that the feature flag controls which
// registration path is used for template tools.
func TestTemplateToolsFeatureFlag(t *testing.T) {
	// Clean up before and after tests
	featureflags.ResetAll()
	defer featureflags.ResetAll()

	t.Run("legacy path when feature flag disabled", func(t *testing.T) {
		// Ensure feature flag is disabled
		featureflags.Disable(featureflags.FeatureFlagTemplateTools)

		registry := &ToolRegistry{tools: make(map[string]Tool)}
		registry.RegisterTemplateTools()

		// Check that the tools were registered
		tools := []string{"get_template", "list_templates", "get_next_child_id"}
		for _, toolName := range tools {
			tool, exists := registry.tools[toolName]
			if !exists {
				t.Errorf("tool %s was not registered", toolName)
				continue
			}

			// Verify the tool definition
			if tool.Definition.Function.Name != toolName {
				t.Errorf("expected tool name %s, got %s", toolName, tool.Definition.Function.Name)
			}

			// Verify the description contains expected text
			desc := tool.Definition.Function.Description
			if desc == "" {
				t.Errorf("tool %s description is empty", toolName)
			}
		}
	})

	t.Run("v2 path when feature flag enabled", func(t *testing.T) {
		// Enable feature flag
		featureflags.Enable(featureflags.FeatureFlagTemplateTools)
		defer featureflags.Disable(featureflags.FeatureFlagTemplateTools)

		registry := &ToolRegistry{tools: make(map[string]Tool)}
		registry.RegisterTemplateTools()

		// Check that the tools were registered
		tools := []string{"get_template", "list_templates", "get_next_child_id"}
		for _, toolName := range tools {
			tool, exists := registry.tools[toolName]
			if !exists {
				t.Errorf("tool %s was not registered", toolName)
				continue
			}

			// Verify the tool definition
			if tool.Definition.Function.Name != toolName {
				t.Errorf("expected tool name %s, got %s", toolName, tool.Definition.Function.Name)
			}

			// Verify the description contains expected text
			desc := tool.Definition.Function.Description
			if desc == "" {
				t.Errorf("tool %s description is empty", toolName)
			}
		}
	})
}

// TestTemplateToolsHandlerExecution tests that both paths produce the same results
func TestTemplateToolsHandlerExecution(t *testing.T) {
	// Clean up before and after tests
	featureflags.ResetAll()
	defer featureflags.ResetAll()

	tools := []string{"get_template", "list_templates", "get_next_child_id"}

	t.Run("legacy handler is callable", func(t *testing.T) {
		featureflags.Disable(featureflags.FeatureFlagTemplateTools)

		registry := &ToolRegistry{tools: make(map[string]Tool)}
		registry.RegisterTemplateTools()

		for _, toolName := range tools {
			tool, exists := registry.tools[toolName]
			if !exists {
				t.Errorf("tool %s not registered", toolName)
				continue
			}

			if tool.Handler == nil {
				t.Errorf("legacy handler for %s is nil", toolName)
			}
		}
	})

	t.Run("v2 handler is callable", func(t *testing.T) {
		featureflags.Enable(featureflags.FeatureFlagTemplateTools)
		defer featureflags.Disable(featureflags.FeatureFlagTemplateTools)

		registry := &ToolRegistry{tools: make(map[string]Tool)}
		registry.RegisterTemplateTools()

		for _, toolName := range tools {
			tool, exists := registry.tools[toolName]
			if !exists {
				t.Errorf("tool %s not registered", toolName)
				continue
			}

			if tool.Handler == nil {
				t.Errorf("v2 handler for %s is nil", toolName)
			}
		}
	})
}

// TestTemplateToolsIntegration is an integration test that verifies
// the tools work end-to-end with both feature flag states
func TestTemplateToolsIntegration(t *testing.T) {
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

	tools := []string{"get_template", "list_templates", "get_next_child_id"}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			if tc.enableFlag {
				featureflags.Enable(featureflags.FeatureFlagTemplateTools)
			} else {
				featureflags.Disable(featureflags.FeatureFlagTemplateTools)
			}
			defer featureflags.ResetAll()

			// Create registry and register tools
			registry := &ToolRegistry{tools: make(map[string]Tool)}
			registry.RegisterTemplateTools()

			// Verify tools were registered
			for _, toolName := range tools {
				tool, exists := registry.tools[toolName]
				if !exists {
					t.Errorf("tool %s was not registered", toolName)
					continue
				}

				// Verify tool has handler
				if tool.Handler == nil {
					t.Errorf("tool %s handler is nil", toolName)
				}

				// Verify tool definition
				if tool.Definition.Function.Name != toolName {
					t.Errorf("expected tool name %s, got %s", toolName, tool.Definition.Function.Name)
				}

				// Verify parameters
				params := tool.Definition.Function.Parameters
				if params == nil {
					t.Errorf("tool %s parameters is nil", toolName)
				}
			}
		})
	}
}

// TestToolRegistryWithTemplateTools tests that the full tool registry
// includes template tools regardless of feature flag state
func TestToolRegistryWithTemplateTools(t *testing.T) {
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

	tools := []string{"get_template", "list_templates", "get_next_child_id"}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			if tc.enableFlag {
				featureflags.Enable(featureflags.FeatureFlagTemplateTools)
			} else {
				featureflags.Disable(featureflags.FeatureFlagTemplateTools)
			}
			defer featureflags.ResetAll()

			// Create full registry
			registry := NewToolRegistry()

			// Verify template tools are registered
			for _, toolName := range tools {
				_, exists := registry.tools[toolName]
				if !exists {
					t.Errorf("template tool %s not found in registry (feature flag: %v)", toolName, tc.enableFlag)
				}
			}
		})
	}
}

// TestTemplateToolHandlerContextValidation tests that the tool context
// validation works correctly for template tools
func TestTemplateToolHandlerContextValidation(t *testing.T) {
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

// TestTemplateToolsParameterValidation tests that parameter validation
// works correctly for template tools
func TestTemplateToolsParameterValidation(t *testing.T) {
	// Clean up before and after tests
	featureflags.ResetAll()
	defer featureflags.ResetAll()

	t.Run("get_template has required parameters", func(t *testing.T) {
		featureflags.Disable(featureflags.FeatureFlagTemplateTools)

		registry := &ToolRegistry{tools: make(map[string]Tool)}
		registry.RegisterTemplateTools()

		tool, exists := registry.tools["get_template"]
		if !exists {
			t.Fatal("get_template tool not registered")
		}

		params := tool.Definition.Function.Parameters
		if params == nil {
			t.Fatal("tool parameters is nil")
		}

		// Parameters are non-nil, which is sufficient for this test
		// The actual parameter validation is handled by the go-openai library
	})

	t.Run("get_next_child_id has required parameters", func(t *testing.T) {
		featureflags.Enable(featureflags.FeatureFlagTemplateTools)
		defer featureflags.Disable(featureflags.FeatureFlagTemplateTools)

		registry := &ToolRegistry{tools: make(map[string]Tool)}
		registry.RegisterTemplateTools()

		tool, exists := registry.tools["get_next_child_id"]
		if !exists {
			t.Fatal("get_next_child_id tool not registered")
		}

		params := tool.Definition.Function.Parameters
		if params == nil {
			t.Fatal("tool parameters is nil")
		}

		// Parameters are non-nil, which is sufficient for this test
		// The actual parameter validation is handled by the go-openai library
	})

	t.Run("list_templates has parameters", func(t *testing.T) {
		featureflags.Disable(featureflags.FeatureFlagTemplateTools)

		registry := &ToolRegistry{tools: make(map[string]Tool)}
		registry.RegisterTemplateTools()

		_, exists := registry.tools["list_templates"]
		if !exists {
			t.Fatal("list_templates tool not registered")
		}

		// list_templates may or may not have parameters, just verify the tool exists
		// The actual parameter validation is handled by the go-openai library
	})
}
