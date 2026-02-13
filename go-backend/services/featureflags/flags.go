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
	// FeatureFlagMemoryTools enables the new memory_tools domain package
	FeatureFlagMemoryTools = "memory_tools_v2"
	// FeatureFlagTemplateTools enables the new template_tools domain package
	FeatureFlagTemplateTools = "template_tools_v2"
	// FeatureFlagCalendarTools enables the new calendar_tools domain package
	FeatureFlagCalendarTools = "calendar_tools_v2"
	// FeatureFlagArticleTools enables the new article_tools domain package
	FeatureFlagArticleTools = "article_tools_v2"
	// FeatureFlagFactTools enables the new fact_tools domain package
	FeatureFlagFactTools = "fact_tools_v2"
	// FeatureFlagTaskTools enables the new task_tools domain package
	FeatureFlagTaskTools = "task_tools_v2"
	// FeatureFlagEntityTools enables the new entity_tools domain package
	FeatureFlagEntityTools = "entity_tools_v2"
	// FeatureFlagCardTools enables the new card_tools domain package
	FeatureFlagCardTools = "card_tools_v2"
	// FeatureFlagChatAgentV2 enables the refactored ChatService for chat operations
	FeatureFlagChatAgentV2 = "chat_agent_v2"
)

// defaultEnabledFlags are flags that are enabled by default
var defaultEnabledFlags = map[string]bool{
	FeatureFlagChatAgentV2: true, // New chat agent is now the default
}

// IsEnabled checks if a feature flag is enabled via environment variable.
// The environment variable format is: ZETTELGARDEN_FEATURE_{FLAG_NAME}
//
// Example for "memory_tools_v2":
//   ZETTELGARDEN_FEATURE_MEMORY_TOOLS_V2=true
//
// Flags can be explicitly disabled by setting them to "false", "0", or "no".
// Default-enabled flags can be disabled with:
//   ZETTELGARDEN_FEATURE_CHAT_AGENT_V2=false
func IsEnabled(flag string) bool {
	envVar := "ZETTELGARDEN_FEATURE_" + strings.ToUpper(flag)
	val := os.Getenv(envVar)

	// If env var is explicitly set, respect it (including "false" to disable defaults)
	if val != "" {
		lowerVal := strings.ToLower(val)
		// Explicit false values
		if lowerVal == "false" || lowerVal == "0" || lowerVal == "no" {
			return false
		}
		// Explicit true values
		return lowerVal == "true" || lowerVal == "1" || lowerVal == "yes"
	}

	// No env var set, check if flag is enabled by default
	return defaultEnabledFlags[flag]
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
