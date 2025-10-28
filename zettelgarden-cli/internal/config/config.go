package config

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"

	"github.com/spf13/viper"
)

// Profile represents a configuration profile for a Zettelgarden instance
type Profile struct {
	Endpoint string `json:"endpoint" mapstructure:"endpoint"`
	Timeout  int    `json:"timeout" mapstructure:"timeout"` // timeout in seconds
}

// OutputConfig represents output formatting preferences
type OutputConfig struct {
	Format  string `json:"format" mapstructure:"format"`   // json, compact-json
	Compact bool   `json:"compact" mapstructure:"compact"` // compact JSON output
	Color   bool   `json:"color" mapstructure:"color"`     // colored output (for future use)
}

// Config represents the complete CLI configuration
type Config struct {
	DefaultProfile string             `json:"default_profile" mapstructure:"default_profile"`
	Profiles       map[string]Profile `json:"profiles" mapstructure:"profiles"`
	Output         OutputConfig       `json:"output" mapstructure:"output"`
}

var defaultConfig = Config{
	DefaultProfile: "default",
	Profiles: map[string]Profile{
		"default": {
			Endpoint: "https://zettelgarden.com",
			Timeout:  30,
		},
	},
	Output: OutputConfig{
		Format:  "json",
		Compact: false,
		Color:   false,
	},
}

// GetConfigDir returns the configuration directory path
func GetConfigDir() (string, error) {
	home, err := os.UserHomeDir()
	if err != nil {
		return "", fmt.Errorf("failed to get home directory: %w", err)
	}
	return filepath.Join(home, ".config", "zettelgarden"), nil
}

// GetConfigPath returns the full path to the config file
func GetConfigPath() (string, error) {
	dir, err := GetConfigDir()
	if err != nil {
		return "", err
	}
	return filepath.Join(dir, "config.json"), nil
}

// LoadConfig loads the configuration from file or returns default config
func LoadConfig() (*Config, error) {
	configPath, err := GetConfigPath()
	if err != nil {
		return nil, err
	}

	// If config file doesn't exist, return default config
	if _, err := os.Stat(configPath); os.IsNotExist(err) {
		return &defaultConfig, nil
	}

	// Read config file
	data, err := os.ReadFile(configPath)
	if err != nil {
		return nil, fmt.Errorf("failed to read config file: %w", err)
	}

	// Parse JSON
	var cfg Config
	if err := json.Unmarshal(data, &cfg); err != nil {
		return nil, fmt.Errorf("failed to parse config file: %w", err)
	}

	return &cfg, nil
}

// SaveConfig saves the configuration to file
func SaveConfig(cfg *Config) error {
	configDir, err := GetConfigDir()
	if err != nil {
		return err
	}

	// Create config directory if it doesn't exist
	if err := os.MkdirAll(configDir, 0700); err != nil {
		return fmt.Errorf("failed to create config directory: %w", err)
	}

	configPath := filepath.Join(configDir, "config.json")

	// Marshal to JSON with indentation
	data, err := json.MarshalIndent(cfg, "", "  ")
	if err != nil {
		return fmt.Errorf("failed to marshal config: %w", err)
	}

	// Write to file with restrictive permissions
	if err := os.WriteFile(configPath, data, 0600); err != nil {
		return fmt.Errorf("failed to write config file: %w", err)
	}

	return nil
}

// GetProfile returns the specified profile from config, with environment variable overrides
func GetProfile(cfg *Config, profileName string) (*Profile, error) {
	// If no profile name specified, use default
	if profileName == "" {
		profileName = cfg.DefaultProfile
	}

	// Check if profile exists
	profile, exists := cfg.Profiles[profileName]
	if !exists {
		return nil, fmt.Errorf("profile '%s' not found in config", profileName)
	}

	// Apply environment variable overrides
	if endpoint := viper.GetString("endpoint"); endpoint != "" {
		profile.Endpoint = endpoint
	}

	if timeout := viper.GetInt("timeout"); timeout > 0 {
		profile.Timeout = timeout
	}

	return &profile, nil
}

// GetActiveProfileName returns the active profile name from viper (respecting --profile flag and ZG_PROFILE env var)
func GetActiveProfileName(cfg *Config) string {
	// Check viper for profile (from --profile flag or ZG_PROFILE env var)
	if profile := viper.GetString("profile"); profile != "" {
		return profile
	}
	// Fall back to default profile from config
	return cfg.DefaultProfile
}

// InitDefaultConfig creates a default config file if none exists
func InitDefaultConfig() error {
	configPath, err := GetConfigPath()
	if err != nil {
		return err
	}

	// If config file already exists, don't overwrite
	if _, err := os.Stat(configPath); err == nil {
		return nil
	}

	return SaveConfig(&defaultConfig)
}
