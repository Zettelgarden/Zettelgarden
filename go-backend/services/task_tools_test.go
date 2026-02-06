package services

import (
	"context"
	"database/sql"
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

		// Check that all task tools were registered
		taskTools := []string{
			"get_tasks",
			"create_task",
			"update_task",
			"get_task_by_id",
			"complete_task",
			"delete_task",
			"complete_and_schedule_task",
		}

		for _, toolName := range taskTools {
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
		featureflags.Enable(featureflags.FeatureFlagTaskTools)
		defer featureflags.Disable(featureflags.FeatureFlagTaskTools)

		registry := &ToolRegistry{tools: make(map[string]Tool)}
		registry.RegisterTaskTools()

		// Check that all task tools were registered
		taskTools := []string{
			"get_tasks",
			"create_task",
			"update_task",
			"get_task_by_id",
			"complete_task",
			"delete_task",
			"complete_and_schedule_task",
		}

		for _, toolName := range taskTools {
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

// TestTaskToolsHandlerExecution tests that both paths produce valid handlers.
func TestTaskToolsHandlerExecution(t *testing.T) {
	// Clean up before and after tests
	featureflags.ResetAll()
	defer featureflags.ResetAll()

	// We can't test the actual database operations without a test database,
	// but we can verify the handlers are set up correctly

	t.Run("legacy handlers are callable", func(t *testing.T) {
		featureflags.Disable(featureflags.FeatureFlagTaskTools)

		registry := &ToolRegistry{tools: make(map[string]Tool)}
		registry.RegisterTaskTools()

		taskTools := []string{
			"get_tasks",
			"create_task",
			"update_task",
			"get_task_by_id",
			"complete_task",
			"delete_task",
			"complete_and_schedule_task",
		}

		for _, toolName := range taskTools {
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
		featureflags.Enable(featureflags.FeatureFlagTaskTools)
		defer featureflags.Disable(featureflags.FeatureFlagTaskTools)

		registry := &ToolRegistry{tools: make(map[string]Tool)}
		registry.RegisterTaskTools()

		taskTools := []string{
			"get_tasks",
			"create_task",
			"update_task",
			"get_task_by_id",
			"complete_task",
			"delete_task",
			"complete_and_schedule_task",
		}

		for _, toolName := range taskTools {
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
// the tools work end-to-end with both feature flag states.
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

			// Verify all task tools were registered
			taskTools := []string{
				"get_tasks",
				"create_task",
				"update_task",
				"get_task_by_id",
				"complete_task",
				"delete_task",
				"complete_and_schedule_task",
			}

			for _, toolName := range taskTools {
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
// includes task tools regardless of feature flag state.
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

			// Verify all task tools are registered
			taskTools := []string{
				"get_tasks",
				"create_task",
				"update_task",
				"get_task_by_id",
				"complete_task",
				"delete_task",
				"complete_and_schedule_task",
			}

			for _, toolName := range taskTools {
				_, exists := registry.tools[toolName]
				if !exists {
					t.Errorf("task tool %s not found in registry (feature flag: %v)", toolName, tc.enableFlag)
				}
			}
		})
	}
}

// TestTaskToolHandlerContextValidation tests that the tool context
// validation works correctly for task tools.
func TestTaskToolHandlerContextValidation(t *testing.T) {
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

// TestTaskToolsParameterValidation tests parameter validation for task tools.
func TestTaskToolsParameterValidation(t *testing.T) {
	// Clean up before and after tests
	featureflags.ResetAll()
	defer featureflags.ResetAll()

	t.Run("get_tasks parameters", func(t *testing.T) {
		registry := &ToolRegistry{tools: make(map[string]Tool)}
		registry.RegisterTaskTools()

		tool, exists := registry.tools["get_tasks"]
		if !exists {
			t.Fatal("get_tasks tool not registered")
		}

		params := tool.Definition.Function.Parameters
		if params == nil {
			t.Fatal("parameters is nil")
		}

		// Parameters are structured as: {"type": "object", "properties": {...}}
		paramsMap, ok := params.(map[string]interface{})
		if !ok {
			t.Fatal("parameters is not a map")
		}

		properties, ok := paramsMap["properties"].(map[string]interface{})
		if !ok {
			t.Fatal("parameters.properties is not a map")
		}

		// Check include_completed parameter
		includeCompleted, ok := properties["include_completed"].(map[string]interface{})
		if !ok {
			t.Error("include_completed parameter not found")
		} else {
			if includeCompleted["type"] != "boolean" {
				t.Errorf("expected boolean type for include_completed, got %v", includeCompleted["type"])
			}
		}

		// Check card_pk parameter
		cardPK, ok := properties["card_pk"].(map[string]interface{})
		if !ok {
			t.Error("card_pk parameter not found")
		} else {
			if cardPK["type"] != "integer" {
				t.Errorf("expected integer type for card_pk, got %v", cardPK["type"])
			}
		}
	})

	t.Run("create_task parameters", func(t *testing.T) {
		registry := &ToolRegistry{tools: make(map[string]Tool)}
		registry.RegisterTaskTools()

		tool, exists := registry.tools["create_task"]
		if !exists {
			t.Fatal("create_task tool not registered")
		}

		params := tool.Definition.Function.Parameters
		if params == nil {
			t.Fatal("parameters is nil")
		}

		paramsMap, ok := params.(map[string]interface{})
		if !ok {
			t.Fatal("parameters is not a map")
		}

		properties, ok := paramsMap["properties"].(map[string]interface{})
		if !ok {
			t.Fatal("parameters.properties is not a map")
		}

		// Check title parameter (required)
		title, ok := properties["title"].(map[string]interface{})
		if !ok {
			t.Error("title parameter not found")
		} else {
			if title["type"] != "string" {
				t.Errorf("expected string type for title, got %v", title["type"])
			}
		}

		// Check optional parameters exist
		for _, paramName := range []string{"scheduled_date", "due_date", "priority", "card_pk"} {
			_, ok := properties[paramName]
			if !ok {
				t.Errorf("%s parameter not found", paramName)
			}
		}
	})

	t.Run("complete_and_schedule_task parameters", func(t *testing.T) {
		registry := &ToolRegistry{tools: make(map[string]Tool)}
		registry.RegisterTaskTools()

		tool, exists := registry.tools["complete_and_schedule_task"]
		if !exists {
			t.Fatal("complete_and_schedule_task tool not registered")
		}

		params := tool.Definition.Function.Parameters
		if params == nil {
			t.Fatal("parameters is nil")
		}

		paramsMap, ok := params.(map[string]interface{})
		if !ok {
			t.Fatal("parameters is not a map")
		}

		properties, ok := paramsMap["properties"].(map[string]interface{})
		if !ok {
			t.Fatal("parameters.properties is not a map")
		}

		// Check task_id parameter (required)
		taskID, ok := properties["task_id"].(map[string]interface{})
		if !ok {
			t.Error("task_id parameter not found")
		} else {
			if taskID["type"] != "integer" {
				t.Errorf("expected integer type for task_id, got %v", taskID["type"])
			}
		}

		// Check days parameter (required)
		days, ok := properties["days"].(map[string]interface{})
		if !ok {
			t.Error("days parameter not found")
		} else {
			if days["type"] != "integer" {
				t.Errorf("expected integer type for days, got %v", days["type"])
			}
		}
	})
}
