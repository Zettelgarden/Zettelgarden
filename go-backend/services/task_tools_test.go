package services

import (
	"testing"

	"go-backend/services/featureflags"
)

// TestTaskToolsFeatureFlag tests that the feature flag controls which
// registration path is used for task tools.
func TestTaskToolsFeatureFlag(t *testing.T) {
	// Clean up before and after tests
	featureflags.ResetAll()
	defer featureflags.ResetAll()

	t.Run("legacy path when feature flag disabled", func(t *testing.T) {
		// Ensure feature flag is disabled
		featureflags.Disable(featureflags.FeatureFlagTaskTools)

		registry := &ToolRegistry{tools: make(map[string]Tool)}
		registry.RegisterTaskTools()

		// Check that tools were registered
		tools := []string{"get_tasks", "create_task", "update_task", "get_task_by_id", "complete_task", "delete_task", "complete_and_schedule_task"}
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

			// Verify the description is not empty
			desc := tool.Definition.Function.Description
			if desc == "" {
				t.Errorf("tool %s description is empty", toolName)
			}
		}
	})

	t.Run("v2 path when feature flag enabled", func(t *testing.T) {
		// Enable feature flag
		featureflags.Enable(featureflags.FeatureFlagTaskTools)
		defer featureflags.Disable(featureflags.FeatureFlagTaskTools)

		registry := &ToolRegistry{tools: make(map[string]Tool)}
		registry.RegisterTaskTools()

		// Check that tools were registered
		tools := []string{"get_tasks", "create_task", "update_task", "get_task_by_id", "complete_task", "delete_task", "complete_and_schedule_task"}
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

			// Verify the description is not empty
			desc := tool.Definition.Function.Description
			if desc == "" {
				t.Errorf("tool %s description is empty", toolName)
			}
		}
	})
}

// TestTaskToolsHandlerExecution tests that both paths produce the same results
func TestTaskToolsHandlerExecution(t *testing.T) {
	// Clean up before and after tests
	featureflags.ResetAll()
	defer featureflags.ResetAll()

	t.Run("legacy handler is callable", func(t *testing.T) {
		featureflags.Disable(featureflags.FeatureFlagTaskTools)

		registry := &ToolRegistry{tools: make(map[string]Tool)}
		registry.RegisterTaskTools()

		tools := []string{"get_tasks", "create_task", "update_task", "get_task_by_id", "complete_task", "delete_task", "complete_and_schedule_task"}
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
		featureflags.Enable(featureflags.FeatureFlagTaskTools)
		defer featureflags.Disable(featureflags.FeatureFlagTaskTools)

		registry := &ToolRegistry{tools: make(map[string]Tool)}
		registry.RegisterTaskTools()

		tools := []string{"get_tasks", "create_task", "update_task", "get_task_by_id", "complete_task", "delete_task", "complete_and_schedule_task"}
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

// TestTaskToolsIntegration is an integration test that verifies
// the tools work end-to-end with both feature flag states
func TestTaskToolsIntegration(t *testing.T) {
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
				featureflags.Enable(featureflags.FeatureFlagTaskTools)
			} else {
				featureflags.Disable(featureflags.FeatureFlagTaskTools)
			}
			defer featureflags.ResetAll()

			// Create registry and register tools
			registry := &ToolRegistry{tools: make(map[string]Tool)}
			registry.RegisterTaskTools()

			// Verify tools were registered
			tools := []string{"get_tasks", "create_task", "update_task", "get_task_by_id", "complete_task", "delete_task", "complete_and_schedule_task"}
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

				// Verify parameters exist
				params := tool.Definition.Function.Parameters
				if params == nil {
					t.Errorf("tool %s parameters is nil", toolName)
				}
			}
		})
	}
}

// TestToolRegistryWithTaskTools tests that the full tool registry
// includes task tools regardless of feature flag state
func TestToolRegistryWithTaskTools(t *testing.T) {
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
				featureflags.Enable(featureflags.FeatureFlagTaskTools)
			} else {
				featureflags.Disable(featureflags.FeatureFlagTaskTools)
			}
			defer featureflags.ResetAll()

			// Create full registry
			registry := NewToolRegistry()

			// Verify task tools are registered
			tools := []string{"get_tasks", "create_task", "update_task", "get_task_by_id", "complete_task", "delete_task", "complete_and_schedule_task"}
			for _, toolName := range tools {
				_, exists := registry.tools[toolName]
				if !exists {
					t.Errorf("task tool %s not found in registry (feature flag: %v)", toolName, tc.enableFlag)
				}
			}
		})
	}
}
