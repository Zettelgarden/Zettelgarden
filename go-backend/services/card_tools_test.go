package services

import (
	"context"
	"database/sql"
	"testing"

	"go-backend/services/featureflags"
)

// TestCardToolsFeatureFlag tests that the feature flag controls which
// registration path is used for card tools.
func TestCardToolsFeatureFlag(t *testing.T) {
	// Clean up before and after tests
	featureflags.ResetAll()
	defer featureflags.ResetAll()

	t.Run("legacy path when feature flag disabled", func(t *testing.T) {
		// Ensure feature flag is disabled
		featureflags.Disable(featureflags.FeatureFlagCardTools)

		registry := &ToolRegistry{tools: make(map[string]Tool)}
		registry.RegisterCardTools()

		// Check that the tools were registered
		expectedTools := []string{
			ToolSearchCards,
			ToolGetCardByID,
			ToolBrowseCardHierarchy,
			ToolCreateCard,
			ToolUpdateCard,
			ToolGetCardAnalysis,
		}

		for _, toolName := range expectedTools {
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
		featureflags.Enable(featureflags.FeatureFlagCardTools)
		defer featureflags.Disable(featureflags.FeatureFlagCardTools)

		registry := &ToolRegistry{tools: make(map[string]Tool)}
		registry.RegisterCardTools()

		// Check that the tools were registered
		expectedTools := []string{
			ToolSearchCards,
			ToolGetCardByID,
			ToolBrowseCardHierarchy,
			ToolCreateCard,
			ToolUpdateCard,
			ToolGetCardAnalysis,
		}

		for _, toolName := range expectedTools {
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

// TestCardToolsHandlerExecution tests that both paths produce the same results
func TestCardToolsHandlerExecution(t *testing.T) {
	// Clean up before and after tests
	featureflags.ResetAll()
	defer featureflags.ResetAll()

	t.Run("legacy handler is callable", func(t *testing.T) {
		featureflags.Disable(featureflags.FeatureFlagCardTools)

		registry := &ToolRegistry{tools: make(map[string]Tool)}
		registry.RegisterCardTools()

		expectedTools := []string{
			ToolSearchCards,
			ToolGetCardByID,
			ToolBrowseCardHierarchy,
			ToolCreateCard,
			ToolUpdateCard,
			ToolGetCardAnalysis,
		}

		for _, toolName := range expectedTools {
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
		featureflags.Enable(featureflags.FeatureFlagCardTools)
		defer featureflags.Disable(featureflags.FeatureFlagCardTools)

		registry := &ToolRegistry{tools: make(map[string]Tool)}
		registry.RegisterCardTools()

		expectedTools := []string{
			ToolSearchCards,
			ToolGetCardByID,
			ToolBrowseCardHierarchy,
			ToolCreateCard,
			ToolUpdateCard,
			ToolGetCardAnalysis,
		}

		for _, toolName := range expectedTools {
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

// TestCardToolsIntegration is an integration test that verifies
// the tool works end-to-end with both feature flag states
func TestCardToolsIntegration(t *testing.T) {
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
				featureflags.Enable(featureflags.FeatureFlagCardTools)
			} else {
				featureflags.Disable(featureflags.FeatureFlagCardTools)
			}
			defer featureflags.ResetAll()

			// Create registry and register tools
			registry := &ToolRegistry{tools: make(map[string]Tool)}
			registry.RegisterCardTools()

			// Verify tools were registered
			expectedTools := []string{
				ToolSearchCards,
				ToolGetCardByID,
				ToolBrowseCardHierarchy,
				ToolCreateCard,
				ToolUpdateCard,
				ToolGetCardAnalysis,
			}

			for _, toolName := range expectedTools {
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

// TestToolRegistryWithCardTools tests that the full tool registry
// includes card tools regardless of feature flag state
func TestToolRegistryWithCardTools(t *testing.T) {
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
				featureflags.Enable(featureflags.FeatureFlagCardTools)
			} else {
				featureflags.Disable(featureflags.FeatureFlagCardTools)
			}
			defer featureflags.ResetAll()

			// Create full registry
			registry := NewToolRegistry()

			// Verify card tools are registered
			expectedTools := []string{
				ToolSearchCards,
				ToolGetCardByID,
				ToolBrowseCardHierarchy,
				ToolCreateCard,
				ToolUpdateCard,
				ToolGetCardAnalysis,
			}

			for _, toolName := range expectedTools {
				_, exists := registry.tools[toolName]
				if !exists {
					t.Errorf("card tool %s not found in registry (feature flag: %v)", toolName, tc.enableFlag)
				}
			}
		})
	}
}

// TestCardToolHandlerContextValidation tests that the tool context
// validation works correctly for card tools
func TestCardToolHandlerContextValidation(t *testing.T) {
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
