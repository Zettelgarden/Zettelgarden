package config

import (
	"encoding/json"
	"os"
	"path/filepath"
)

const (
	DefaultAPIURL      = "http://localhost:8080"
	DefaultTimeoutSecs = 30
)

type Config struct {
	APIURL         string `json:"api_url"`
	Token          string `json:"token"`
	TimeoutSeconds int    `json:"timeout_seconds"`
}

// GetDefaultConfigPath returns the default config file location
func GetDefaultConfigPath() (string, error) {
	homeDir, err := os.UserHomeDir()
	if err != nil {
		return "", err
	}
	return filepath.Join(homeDir, ".config", "zettelgarden", "config.json"), nil
}

// LoadConfig loads configuration from the specified path
func LoadConfig(path string) (*Config, error) {
	data, err := os.ReadFile(path)
	if err != nil {
		return nil, err
	}

	var cfg Config
	if err := json.Unmarshal(data, &cfg); err != nil {
		return nil, err
	}

	// Set defaults
	if cfg.APIURL == "" {
		cfg.APIURL = DefaultAPIURL
	}
	if cfg.TimeoutSeconds <= 0 {
		cfg.TimeoutSeconds = DefaultTimeoutSecs
	}

	return &cfg, nil
}
