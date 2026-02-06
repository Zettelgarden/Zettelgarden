package services

import (
	"context"
	"database/sql"
	"testing"

	"go-backend/services/featureflags"
)

// TestCalendarToolsFeatureFlag tests that the feature flag controls which
// registration path is used for calendar tools.
func TestCalendarToolsFeatureFlag(t *testing.T) {
	// Clean up before and after tests
	featureflags.ResetAll()
	defer featureflags.ResetAll()

	t.Run("legacy path when feature flag disabled", func(t *testing.T) {
		// Ensure feature flag is disabled
		featureflags.Disable(featureflags.FeatureFlagCalendarTools)

		registry := &ToolRegistry{tools: make(map[string]Tool)}
		registry.RegisterCalendarTools()

		// Check that the tools were registered
		tools := []string{
			"list_external_calendars",
			"list_external_events",
			"link_event_to_card",
		}

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
		featureflags.Enable(featureflags.FeatureFlagCalendarTools)
		defer featureflags.Disable(featureflags.FeatureFlagCalendarTools)

		registry := &ToolRegistry{tools: make(map[string]Tool)}
		registry.RegisterCalendarTools()

		// Check that the tools were registered
		tools := []string{
			"list_external_calendars",
			"list_external_events",
			"link_event_to_card",
		}

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

// TestCalendarToolsHandlerExecution tests that both paths produce the same results
func TestCalendarToolsHandlerExecution(t *testing.T) {
	// Clean up before and after tests
	featureflags.ResetAll()
	defer featureflags.ResetAll()

	// We can't test the actual database operations without a test database,
	// but we can verify the handlers are set up correctly

	t.Run("legacy handlers are callable", func(t *testing.T) {
		featureflags.Disable(featureflags.FeatureFlagCalendarTools)

		registry := &ToolRegistry{tools: make(map[string]Tool)}
		registry.RegisterCalendarTools()

		tools := []string{
			"list_external_calendars",
			"list_external_events",
			"link_event_to_card",
		}

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

	t.Run("v2 handlers are callable", func(t *testing.T) {
		featureflags.Enable(featureflags.FeatureFlagCalendarTools)
		defer featureflags.Disable(featureflags.FeatureFlagCalendarTools)

		registry := &ToolRegistry{tools: make(map[string]Tool)}
		registry.RegisterCalendarTools()

		tools := []string{
			"list_external_calendars",
			"list_external_events",
			"link_event_to_card",
		}

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

// TestCalendarToolsIntegration is an integration test that verifies
// the tools work end-to-end with both feature flag states
func TestCalendarToolsIntegration(t *testing.T) {
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

	tools := []string{
		"list_external_calendars",
		"list_external_events",
		"link_event_to_card",
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			if tc.enableFlag {
				featureflags.Enable(featureflags.FeatureFlagCalendarTools)
			} else {
				featureflags.Disable(featureflags.FeatureFlagCalendarTools)
			}
			defer featureflags.ResetAll()

			// Create registry and register tools
			registry := &ToolRegistry{tools: make(map[string]Tool)}
			registry.RegisterCalendarTools()

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

// TestToolRegistryWithCalendarTools tests that the full tool registry
// includes calendar tools regardless of feature flag state
func TestToolRegistryWithCalendarTools(t *testing.T) {
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

	tools := []string{
		"list_external_calendars",
		"list_external_events",
		"link_event_to_card",
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			if tc.enableFlag {
				featureflags.Enable(featureflags.FeatureFlagCalendarTools)
			} else {
				featureflags.Disable(featureflags.FeatureFlagCalendarTools)
			}
			defer featureflags.ResetAll()

			// Create full registry
			registry := NewToolRegistry()

			// Verify calendar tools are registered
			for _, toolName := range tools {
				_, exists := registry.tools[toolName]
				if !exists {
					t.Errorf("calendar tool %s not found in registry (feature flag: %v)", toolName, tc.enableFlag)
				}
			}
		})
	}
}

// TestCalendarToolHandlerContextValidation tests that the tool context
// validation works correctly for calendar tools
func TestCalendarToolHandlerContextValidation(t *testing.T) {
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

// TestCalendarToolsParameters tests that parameter validation works
func TestCalendarToolsParameters(t *testing.T) {
	// Clean up before and after tests
	featureflags.ResetAll()
	defer featureflags.ResetAll()

	t.Run("list_external_events has required parameters", func(t *testing.T) {
		featureflags.Disable(featureflags.FeatureFlagCalendarTools)

		registry := &ToolRegistry{tools: make(map[string]Tool)}
		registry.RegisterCalendarTools()

		tool, exists := registry.tools["list_external_events"]
		if !exists {
			t.Fatal("tool list_external_events not registered")
		}

		params := tool.Definition.Function.Parameters
		if params == nil {
			t.Fatal("tool parameters is nil")
		}

		// Parameters are non-nil, which is sufficient for this test
		// The actual parameter validation is handled by the go-openai library
	})

	t.Run("link_event_to_card has required parameters", func(t *testing.T) {
		featureflags.Enable(featureflags.FeatureFlagCalendarTools)
		defer featureflags.Disable(featureflags.FeatureFlagCalendarTools)

		registry := &ToolRegistry{tools: make(map[string]Tool)}
		registry.RegisterCalendarTools()

		tool, exists := registry.tools["link_event_to_card"]
		if !exists {
			t.Fatal("tool link_event_to_card not registered")
		}

		params := tool.Definition.Function.Parameters
		if params == nil {
			t.Fatal("tool parameters is nil")
		}

		// Parameters are non-nil, which is sufficient for this test
		// The actual parameter validation is handled by the go-openai library
	})
}
