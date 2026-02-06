// Package featureflags provides simple feature flag support for incremental rollout.
//
// Flags can be controlled via environment variables in the format:
// ZETTELGARDEN_FEATURE_{FLAG_NAME}=true
//
// Example:
//   ZETTELGARDEN_FEATURE_TEMPLATE_TOOLS_V2=true ./go-backend
package featureflags

import (
	"os"
	"strings"
)

// Feature flag constants
const (
	// FeatureFlagTemplateTools enables the new template_tools domain package
	FeatureFlagTemplateTools = "template_tools_v2"
	// FeatureFlagCalendarTools enables the new calendar_tools domain package
	FeatureFlagCalendarTools = "calendar_tools_v2"
	// FeatureFlagArticleTools enables the new article_tools domain package
	FeatureFlagArticleTools = "article_tools_v2"

	// Future flags for other domains (to be added):
	// FeatureFlagFactTools = "fact_tools_v2"
	// FeatureFlagTaskTools = "task_tools_v2"
	// FeatureFlagEntityTools = "entity_tools_v2"
	// FeatureFlagCardTools = "card_tools_v2"
)

// IsEnabled checks if a feature flag is enabled via environment variable.
// The environment variable format is: ZETTELGARDEN_FEATURE_{FLAG_NAME}
//
// Example for "memory_tools_v2":
//   ZETTELGARDEN_FEATURE_MEMORY_TOOLS_V2=true
func IsEnabled(flag string) bool {
	envVar := "ZETTELGARDEN_FEATURE_" + strings.ToUpper(flag)
	val := os.Getenv(envVar)
	return strings.ToLower(val) == "true" || strings.ToLower(val) == "1" || strings.ToLower(val) == "yes"
}

// Enable is used for testing to programmatically enable a feature flag.
// In production, use environment variables instead.
func Enable(flag string) {
	envVar := "ZETTELGARDEN_FEATURE_" + strings.ToUpper(flag)
	os.Setenv(envVar, "true")
}

// Disable is used for testing to programmatically disable a feature flag.
func Disable(flag string) {
	envVar := "ZETTELGARDEN_FEATURE_" + strings.ToUpper(flag)
	os.Setenv(envVar, "false")
}

// ResetAll clears all feature flags (useful for testing).
func ResetAll() {
	prefix := "ZETTELGARDEN_FEATURE_"
	for _, env := range os.Environ() {
		if strings.HasPrefix(env, prefix) {
			parts := strings.SplitN(env, "=", 2)
			if len(parts) == 2 {
				os.Unsetenv(parts[0])
			}
		}
	}
}
