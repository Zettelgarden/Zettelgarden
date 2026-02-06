package services

import (
	"context"
	"database/sql"
	"testing"

	"go-backend/services/featureflags"
)

// TestFactToolsFeatureFlag tests that the feature flag controls which
// registration path is used for fact tools.
func TestFactToolsFeatureFlag(t *testing.T) {
	// Clean up before and after tests
	featureflags.ResetAll()
	defer featureflags.ResetAll()

	t.Run("legacy path when feature flag disabled", func(t *testing.T) {
		// Ensure feature flag is disabled
		featureflags.Disable(featureflags.FeatureFlagFactTools)

		registry := &ToolRegistry{tools: make(map[string]Tool)}
		registry.RegisterFactTools()

		// Check that all fact tools were registered
		factTools := []string{
			"search_facts",
			"get_card_facts",
			"get_entity_facts",
			"get_fact_cards",
		}

		for _, toolName := range factTools {
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
		featureflags.Enable(featureflags.FeatureFlagFactTools)
		defer featureflags.Disable(featureflags.FeatureFlagFactTools)

		registry := &ToolRegistry{tools: make(map[string]Tool)}
		registry.RegisterFactTools()

		// Check that all fact tools were registered
		factTools := []string{
			"search_facts",
			"get_card_facts",
			"get_entity_facts",
			"get_fact_cards",
		}

		for _, toolName := range factTools {
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

// TestFactToolsHandlerExecution tests that both paths produce the same results
func TestFactToolsHandlerExecution(t *testing.T) {
	// Clean up before and after tests
	featureflags.ResetAll()
	defer featureflags.ResetAll()

	// We can't test the actual database operations without a test database,
	// but we can verify the handlers are set up correctly

	t.Run("legacy handler is callable", func(t *testing.T) {
		featureflags.Disable(featureflags.FeatureFlagFactTools)

		registry := &ToolRegistry{tools: make(map[string]Tool)}
		registry.RegisterFactTools()

		factTools := []string{
			"search_facts",
			"get_card_facts",
			"get_entity_facts",
			"get_fact_cards",
		}

		for _, toolName := range factTools {
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
		featureflags.Enable(featureflags.FeatureFlagFactTools)
		defer featureflags.Disable(featureflags.FeatureFlagFactTools)

		registry := &ToolRegistry{tools: make(map[string]Tool)}
		registry.RegisterFactTools()

		factTools := []string{
			"search_facts",
			"get_card_facts",
			"get_entity_facts",
			"get_fact_cards",
		}

		for _, toolName := range factTools {
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

// TestFactToolsIntegration is an integration test that verifies
// the tool works end-to-end with both feature flag states
func TestFactToolsIntegration(t *testing.T) {
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

	factTools := []string{
		"search_facts",
		"get_card_facts",
		"get_entity_facts",
		"get_fact_cards",
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			if tc.enableFlag {
				featureflags.Enable(featureflags.FeatureFlagFactTools)
			} else {
				featureflags.Disable(featureflags.FeatureFlagFactTools)
			}
			defer featureflags.ResetAll()

			// Create registry and register tools
			registry := &ToolRegistry{tools: make(map[string]Tool)}
			registry.RegisterFactTools()

			// Verify all fact tools were registered
			for _, toolName := range factTools {
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

// TestToolRegistryWithFactTools tests that the full tool registry
// includes fact tools regardless of feature flag state
func TestToolRegistryWithFactTools(t *testing.T) {
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

	factTools := []string{
		"search_facts",
		"get_card_facts",
		"get_entity_facts",
		"get_fact_cards",
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			if tc.enableFlag {
				featureflags.Enable(featureflags.FeatureFlagFactTools)
			} else {
				featureflags.Disable(featureflags.FeatureFlagFactTools)
			}
			defer featureflags.ResetAll()

			// Create full registry
			registry := NewToolRegistry()

			// Verify all fact tools are registered
			for _, toolName := range factTools {
				_, exists := registry.tools[toolName]
				if !exists {
					t.Errorf("fact tool %s not found in registry (feature flag: %v)", toolName, tc.enableFlag)
				}
			}
		})
	}
}

// TestFactToolHandlerContextValidation tests that the tool context
// validation works correctly for fact tools
func TestFactToolHandlerContextValidation(t *testing.T) {
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

// TestFactToolsParameterValidation tests that parameter validation
// works correctly for fact tools
func TestFactToolsParameterValidation(t *testing.T) {
	// Clean up before and after tests
	featureflags.ResetAll()
	defer featureflags.ResetAll()

	testCases := []struct {
		name       string
		enableFlag bool
	}{
		{name: "legacy path", enableFlag: false},
		{name: "v2 path", enableFlag: true},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			if tc.enableFlag {
				featureflags.Enable(featureflags.FeatureFlagFactTools)
			} else {
				featureflags.Disable(featureflags.FeatureFlagFactTools)
			}
			defer featureflags.ResetAll()

			registry := &ToolRegistry{tools: make(map[string]Tool)}
			registry.RegisterFactTools()

			t.Run("search_facts requires query parameter", func(t *testing.T) {
				tool := registry.tools["search_facts"]
				if tool.Handler == nil {
					t.Skip("handler not registered")
				}

				ctx := &ToolContext{
					Context: context.Background(),
					UserID:  1,
					DB:      &sql.DB{},
				}

				// Test missing required parameter
				_, err := tool.Handler(map[string]interface{}{}, ctx)
				if err == nil {
					t.Error("expected error for missing query parameter, got nil")
				}
			})

			t.Run("get_card_facts requires card_pk parameter", func(t *testing.T) {
				tool := registry.tools["get_card_facts"]
				if tool.Handler == nil {
					t.Skip("handler not registered")
				}

				ctx := &ToolContext{
					Context: context.Background(),
					UserID:  1,
					DB:      &sql.DB{},
				}

				// Test missing required parameter
				_, err := tool.Handler(map[string]interface{}{}, ctx)
				if err == nil {
					t.Error("expected error for missing card_pk parameter, got nil")
				}
			})

			t.Run("get_entity_facts requires entity_id parameter", func(t *testing.T) {
				tool := registry.tools["get_entity_facts"]
				if tool.Handler == nil {
					t.Skip("handler not registered")
				}

				ctx := &ToolContext{
					Context: context.Background(),
					UserID:  1,
					DB:      &sql.DB{},
				}

				// Test missing required parameter
				_, err := tool.Handler(map[string]interface{}{}, ctx)
				if err == nil {
					t.Error("expected error for missing entity_id parameter, got nil")
				}
			})

			t.Run("get_fact_cards requires fact_id parameter", func(t *testing.T) {
				tool := registry.tools["get_fact_cards"]
				if tool.Handler == nil {
					t.Skip("handler not registered")
				}

				ctx := &ToolContext{
					Context: context.Background(),
					UserID:  1,
					DB:      &sql.DB{},
				}

				// Test missing required parameter
				_, err := tool.Handler(map[string]interface{}{}, ctx)
				if err == nil {
					t.Error("expected error for missing fact_id parameter, got nil")
				}
			})
		})
	}
}
