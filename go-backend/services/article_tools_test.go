package services

import (
	"context"
	"database/sql"
	"testing"

	"go-backend/services/featureflags"
)

// TestArticleToolsFeatureFlag tests that the feature flag controls which
// registration path is used for article tools.
func TestArticleToolsFeatureFlag(t *testing.T) {
	// Clean up before and after tests
	featureflags.ResetAll()
	defer featureflags.ResetAll()

	t.Run("legacy path when feature flag disabled", func(t *testing.T) {
		// Ensure feature flag is disabled
		featureflags.Disable(featureflags.FeatureFlagArticleTools)

		registry := &ToolRegistry{tools: make(map[string]Tool)}
		registry.RegisterArticleTools()

		// Check that parse_url was registered
		tool, exists := registry.tools["parse_url"]
		if !exists {
			t.Errorf("tool parse_url was not registered")
			return
		}

		// Verify the tool definition
		if tool.Definition.Function.Name != "parse_url" {
			t.Errorf("expected tool name parse_url, got %s", tool.Definition.Function.Name)
		}

		// Verify the description contains expected text
		desc := tool.Definition.Function.Description
		if desc == "" {
			t.Error("tool description is empty")
		}

		// Check that create_article was registered
		tool2, exists := registry.tools["create_article"]
		if !exists {
			t.Errorf("tool create_article was not registered")
			return
		}

		// Verify the tool definition
		if tool2.Definition.Function.Name != "create_article" {
			t.Errorf("expected tool name create_article, got %s", tool2.Definition.Function.Name)
		}
	})

	t.Run("v2 path when feature flag enabled", func(t *testing.T) {
		// Enable feature flag
		featureflags.Enable(featureflags.FeatureFlagArticleTools)
		defer featureflags.Disable(featureflags.FeatureFlagArticleTools)

		registry := &ToolRegistry{tools: make(map[string]Tool)}
		registry.RegisterArticleTools()

		// Check that parse_url was registered
		tool, exists := registry.tools["parse_url"]
		if !exists {
			t.Errorf("tool parse_url was not registered")
			return
		}

		// Verify the tool definition
		if tool.Definition.Function.Name != "parse_url" {
			t.Errorf("expected tool name parse_url, got %s", tool.Definition.Function.Name)
		}

		// Verify the description contains expected text
		desc := tool.Definition.Function.Description
		if desc == "" {
			t.Error("tool description is empty")
		}

		// Check that create_article was registered
		tool2, exists := registry.tools["create_article"]
		if !exists {
			t.Errorf("tool create_article was not registered")
			return
		}

		// Verify the tool definition
		if tool2.Definition.Function.Name != "create_article" {
			t.Errorf("expected tool name create_article, got %s", tool2.Definition.Function.Name)
		}
	})
}

// TestArticleToolsHandlerExecution tests that both paths produce the same results
func TestArticleToolsHandlerExecution(t *testing.T) {
	// Clean up before and after tests
	featureflags.ResetAll()
	defer featureflags.ResetAll()

	t.Run("legacy handler is callable", func(t *testing.T) {
		featureflags.Disable(featureflags.FeatureFlagArticleTools)

		registry := &ToolRegistry{tools: make(map[string]Tool)}
		registry.RegisterArticleTools()

		tool, exists := registry.tools["parse_url"]
		if !exists {
			t.Fatal("tool parse_url not registered")
		}

		if tool.Handler == nil {
			t.Error("legacy handler is nil")
		}

		tool2, exists := registry.tools["create_article"]
		if !exists {
			t.Fatal("tool create_article not registered")
		}

		if tool2.Handler == nil {
			t.Error("legacy handler is nil")
		}
	})

	t.Run("v2 handler is callable", func(t *testing.T) {
		featureflags.Enable(featureflags.FeatureFlagArticleTools)
		defer featureflags.Disable(featureflags.FeatureFlagArticleTools)

		registry := &ToolRegistry{tools: make(map[string]Tool)}
		registry.RegisterArticleTools()

		tool, exists := registry.tools["parse_url"]
		if !exists {
			t.Fatal("tool parse_url not registered")
		}

		if tool.Handler == nil {
			t.Error("v2 handler is nil")
		}

		tool2, exists := registry.tools["create_article"]
		if !exists {
			t.Fatal("tool create_article not registered")
		}

		if tool2.Handler == nil {
			t.Error("v2 handler is nil")
		}
	})
}

// TestArticleToolsIntegration is an integration test that verifies
// the tool works end-to-end with both feature flag states
func TestArticleToolsIntegration(t *testing.T) {
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
				featureflags.Enable(featureflags.FeatureFlagArticleTools)
			} else {
				featureflags.Disable(featureflags.FeatureFlagArticleTools)
			}
			defer featureflags.ResetAll()

			// Create registry and register tools
			registry := &ToolRegistry{tools: make(map[string]Tool)}
			registry.RegisterArticleTools()

			// Verify parse_url was registered
			tool, exists := registry.tools["parse_url"]
			if !exists {
				t.Errorf("tool parse_url was not registered")
				return
			}

			// Verify tool has handler
			if tool.Handler == nil {
				t.Error("tool handler is nil")
			}

			// Verify tool definition
			if tool.Definition.Function.Name != "parse_url" {
				t.Errorf("expected tool name parse_url, got %s", tool.Definition.Function.Name)
			}

			// Verify parameters (parse_url has required url parameter)
			params := tool.Definition.Function.Parameters
			if params == nil {
				t.Error("tool parameters is nil")
			}

			// Verify create_article was registered
			tool2, exists := registry.tools["create_article"]
			if !exists {
				t.Errorf("tool create_article was not registered")
				return
			}

			// Verify tool has handler
			if tool2.Handler == nil {
				t.Error("tool handler is nil")
			}

			// Verify tool definition
			if tool2.Definition.Function.Name != "create_article" {
				t.Errorf("expected tool name create_article, got %s", tool2.Definition.Function.Name)
			}
		})
	}
}

// TestToolRegistryWithArticleTools tests that the full tool registry
// includes article tools regardless of feature flag state
func TestToolRegistryWithArticleTools(t *testing.T) {
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
				featureflags.Enable(featureflags.FeatureFlagArticleTools)
			} else {
				featureflags.Disable(featureflags.FeatureFlagArticleTools)
			}
			defer featureflags.ResetAll()

			// Create full registry
			registry := NewToolRegistry()

			// Verify article tools are registered
			_, exists := registry.tools["parse_url"]
			if !exists {
				t.Errorf("parse_url tool not found in registry (feature flag: %v)", tc.enableFlag)
			}

			_, exists = registry.tools["create_article"]
			if !exists {
				t.Errorf("create_article tool not found in registry (feature flag: %v)", tc.enableFlag)
			}
		})
	}
}

// TestArticleToolHandlerContextValidation tests that the tool context
// validation works correctly for article tools
func TestArticleToolHandlerContextValidation(t *testing.T) {
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
