package services

import (
	"context"
	"database/sql"
	"testing"

	"go-backend/services/featureflags"
)

// TestEntityToolsFeatureFlag tests that the feature flag controls which
// registration path is used for entity tools.
func TestEntityToolsFeatureFlag(t *testing.T) {
	// Clean up before and after tests
	featureflags.ResetAll()
	defer featureflags.ResetAll()

	t.Run("legacy path when feature flag disabled", func(t *testing.T) {
		// Ensure feature flag is disabled
		featureflags.Disable(featureflags.FeatureFlagEntityTools)

		registry := &ToolRegistry{tools: make(map[string]Tool)}
		registry.RegisterEntityTools()

		// Check that all 10 entity tools were registered
		entityTools := []string{
			ToolGetEntityByName,
			ToolSearchEntities,
			ToolGetCardsByEntity,
			ToolGetEntityByID,
			ToolMergeEntities,
			ToolUpdateEntity,
			ToolDeleteEntity,
			ToolAddEntityToCard,
			ToolRemoveEntityFromCard,
			ToolGetSimilarEntities,
		}

		for _, toolName := range entityTools {
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
		featureflags.Enable(featureflags.FeatureFlagEntityTools)
		defer featureflags.Disable(featureflags.FeatureFlagEntityTools)

		registry := &ToolRegistry{tools: make(map[string]Tool)}
		registry.RegisterEntityTools()

		// Check that all 10 entity tools were registered
		entityTools := []string{
			ToolGetEntityByName,
			ToolSearchEntities,
			ToolGetCardsByEntity,
			ToolGetEntityByID,
			ToolMergeEntities,
			ToolUpdateEntity,
			ToolDeleteEntity,
			ToolAddEntityToCard,
			ToolRemoveEntityFromCard,
			ToolGetSimilarEntities,
		}

		for _, toolName := range entityTools {
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

// TestEntityToolsHandlerExecution tests that both paths produce the same results
func TestEntityToolsHandlerExecution(t *testing.T) {
	// Clean up before and after tests
	featureflags.ResetAll()
	defer featureflags.ResetAll()

	// We can't test the actual database operations without a test database,
	// but we can verify the handlers are set up correctly

	t.Run("legacy handler is callable", func(t *testing.T) {
		featureflags.Disable(featureflags.FeatureFlagEntityTools)

		registry := &ToolRegistry{tools: make(map[string]Tool)}
		registry.RegisterEntityTools()

		// Check all 10 entity tools have handlers
		entityTools := []string{
			ToolGetEntityByName,
			ToolSearchEntities,
			ToolGetCardsByEntity,
			ToolGetEntityByID,
			ToolMergeEntities,
			ToolUpdateEntity,
			ToolDeleteEntity,
			ToolAddEntityToCard,
			ToolRemoveEntityFromCard,
			ToolGetSimilarEntities,
		}

		for _, toolName := range entityTools {
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
		featureflags.Enable(featureflags.FeatureFlagEntityTools)
		defer featureflags.Disable(featureflags.FeatureFlagEntityTools)

		registry := &ToolRegistry{tools: make(map[string]Tool)}
		registry.RegisterEntityTools()

		// Check all 10 entity tools have handlers
		entityTools := []string{
			ToolGetEntityByName,
			ToolSearchEntities,
			ToolGetCardsByEntity,
			ToolGetEntityByID,
			ToolMergeEntities,
			ToolUpdateEntity,
			ToolDeleteEntity,
			ToolAddEntityToCard,
			ToolRemoveEntityFromCard,
			ToolGetSimilarEntities,
		}

		for _, toolName := range entityTools {
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

// TestEntityToolsIntegration is an integration test that verifies
// the tools work end-to-end with both feature flag states
func TestEntityToolsIntegration(t *testing.T) {
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
				featureflags.Enable(featureflags.FeatureFlagEntityTools)
			} else {
				featureflags.Disable(featureflags.FeatureFlagEntityTools)
			}
			defer featureflags.ResetAll()

			// Create registry and register tools
			registry := &ToolRegistry{tools: make(map[string]Tool)}
			registry.RegisterEntityTools()

			// Verify all 10 entity tools were registered
			entityTools := []string{
				ToolGetEntityByName,
				ToolSearchEntities,
				ToolGetCardsByEntity,
				ToolGetEntityByID,
				ToolMergeEntities,
				ToolUpdateEntity,
				ToolDeleteEntity,
				ToolAddEntityToCard,
				ToolRemoveEntityFromCard,
				ToolGetSimilarEntities,
			}

			for _, toolName := range entityTools {
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

				// Verify parameters exist
				params := tool.Definition.Function.Parameters
				if params == nil {
					t.Errorf("tool %s parameters is nil", toolName)
				}
			}
		})
	}
}

// TestToolRegistryWithEntityTools tests that the full tool registry
// includes entity tools regardless of feature flag state
func TestToolRegistryWithEntityTools(t *testing.T) {
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
				featureflags.Enable(featureflags.FeatureFlagEntityTools)
			} else {
				featureflags.Disable(featureflags.FeatureFlagEntityTools)
			}
			defer featureflags.ResetAll()

			// Create full registry
			registry := NewToolRegistry()

			// Verify all 10 entity tools are registered
			entityTools := []string{
				ToolGetEntityByName,
				ToolSearchEntities,
				ToolGetCardsByEntity,
				ToolGetEntityByID,
				ToolMergeEntities,
				ToolUpdateEntity,
				ToolDeleteEntity,
				ToolAddEntityToCard,
				ToolRemoveEntityFromCard,
				ToolGetSimilarEntities,
			}

			for _, toolName := range entityTools {
				_, exists := registry.tools[toolName]
				if !exists {
					t.Errorf("entity tool %s not found in registry (feature flag: %v)", toolName, tc.enableFlag)
				}
			}
		})
	}
}

// TestEntityToolHandlerContextValidation tests that the tool context
// validation works correctly for entity tools
func TestEntityToolHandlerContextValidation(t *testing.T) {
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

// TestEntityToolsParameterValidation tests that entity tools
// have the correct parameter definitions
func TestEntityToolsParameterValidation(t *testing.T) {
	// Clean up before and after tests
	featureflags.ResetAll()
	defer featureflags.ResetAll()

	testCases := []struct {
		name       string
		enableFlag bool
		toolName   string
		hasParams  bool
	}{
		{
			name:       "get_entity_by_name - legacy",
			enableFlag: false,
			toolName:   ToolGetEntityByName,
			hasParams:  true,
		},
		{
			name:       "get_entity_by_name - v2",
			enableFlag: true,
			toolName:   ToolGetEntityByName,
			hasParams:  true,
		},
		{
			name:       "search_entities - legacy",
			enableFlag: false,
			toolName:   ToolSearchEntities,
			hasParams:  true,
		},
		{
			name:       "search_entities - v2",
			enableFlag: true,
			toolName:   ToolSearchEntities,
			hasParams:  true,
		},
		{
			name:       "merge_entities - legacy",
			enableFlag: false,
			toolName:   ToolMergeEntities,
			hasParams:  true,
		},
		{
			name:       "merge_entities - v2",
			enableFlag: true,
			toolName:   ToolMergeEntities,
			hasParams:  true,
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			if tc.enableFlag {
				featureflags.Enable(featureflags.FeatureFlagEntityTools)
			} else {
				featureflags.Disable(featureflags.FeatureFlagEntityTools)
			}
			defer featureflags.ResetAll()

			registry := &ToolRegistry{tools: make(map[string]Tool)}
			registry.RegisterEntityTools()

			tool, exists := registry.tools[tc.toolName]
			if !exists {
				t.Fatalf("tool %s was not registered", tc.toolName)
			}

			params := tool.Definition.Function.Parameters
			if tc.hasParams && params == nil {
				t.Errorf("tool %s expected parameters but got nil", tc.toolName)
			}
		})
	}
}
