package featureflags

import (
	"os"
	"strings"
	"testing"
)

func TestIsEnabled(t *testing.T) {
	// Clean up before and after tests
	ResetAll()
	defer ResetAll()

	tests := []struct {
		name     string
		flag     string
		envValue string
		want     bool
	}{
		{
			name:     "flag enabled with true",
			flag:     FeatureFlagMemoryTools,
			envValue: "true",
			want:     true,
		},
		{
			name:     "flag enabled with 1",
			flag:     FeatureFlagMemoryTools,
			envValue: "1",
			want:     true,
		},
		{
			name:     "flag enabled with yes",
			flag:     FeatureFlagMemoryTools,
			envValue: "yes",
			want:     true,
		},
		{
			name:     "flag enabled with YES (uppercase)",
			flag:     FeatureFlagMemoryTools,
			envValue: "YES",
			want:     true,
		},
		{
			name:     "flag disabled with false",
			flag:     FeatureFlagMemoryTools,
			envValue: "false",
			want:     false,
		},
		{
			name:     "flag disabled with 0",
			flag:     FeatureFlagMemoryTools,
			envValue: "0",
			want:     false,
		},
		{
			name:     "flag disabled with empty string",
			flag:     FeatureFlagMemoryTools,
			envValue: "",
			want:     false,
		},
		{
			name:     "flag disabled when not set",
			flag:     FeatureFlagMemoryTools,
			envValue: "",
			want:     false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Reset before each test
			ResetAll()

			// Set environment variable
			if tt.envValue != "" {
				envVar := "ZETTELGARDEN_FEATURE_" + tt.flag
				// Ensure uppercase for env var name
				envVar = "ZETTELGARDEN_FEATURE_" + strings.ToUpper(strings.ReplaceAll(tt.flag, "_v2", "_V2"))
				// Actually, we need to handle the proper conversion
				envVar = "ZETTELGARDEN_FEATURE_" + strings.ToUpper(tt.flag)
				os.Setenv(envVar, tt.envValue)
			}

			// Test
			got := IsEnabled(tt.flag)

			if got != tt.want {
				t.Errorf("IsEnabled(%q) = %v, want %v", tt.flag, got, tt.want)
			}
		})
	}
}

func TestEnableDisable(t *testing.T) {
	ResetAll()
	defer ResetAll()

	flag := "test_flag"

	// Initially disabled
	if IsEnabled(flag) {
		t.Error("flag should be initially disabled")
	}

	// Enable the flag
	Enable(flag)
	if !IsEnabled(flag) {
		t.Error("flag should be enabled after Enable()")
	}

	// Disable the flag
	Disable(flag)
	if IsEnabled(flag) {
		t.Error("flag should be disabled after Disable()")
	}
}

func TestResetAll(t *testing.T) {
	// Enable some flags
	Enable(FeatureFlagMemoryTools)
	Enable("another_flag")

	// Reset all
	ResetAll()

	// Check all are disabled
	if IsEnabled(FeatureFlagMemoryTools) {
		t.Error("memory_tools_v2 should be disabled after ResetAll")
	}
	if IsEnabled("another_flag") {
		t.Error("another_flag should be disabled after ResetAll")
	}
}
