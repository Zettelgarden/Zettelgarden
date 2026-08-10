package config

import (
	"encoding/json"
	"errors"
	"fmt"
	"net/url"
	"os"
	"path/filepath"
	"strings"
)

const (
	DefaultAPIURL      = "http://localhost:8080"
	DefaultTimeoutSecs = 30

	// EnvAPIURL overrides the configured API URL
	// (precedence: --url flag > env > config file).
	EnvAPIURL = "ZETTELGARDEN_API_URL"
)

type Config struct {
	APIURL         string `json:"api_url"`
	Token          string `json:"token"`
	TimeoutSeconds int    `json:"timeout_seconds"`
	// APIKeyName/APIKeyID record the API key minted by `zg auth login` so
	// `zg auth status`/`revoke` can identify and manage it. The token itself
	// lives in the OS keyring (or config.Token as a fallback).
	APIKeyName string `json:"api_key_name,omitempty"`
	APIKeyID   int    `json:"api_key_id,omitempty"`
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

// ResolveAPIURL returns the effective API URL using precedence:
// --url flag > ZETTELGARDEN_API_URL env var > config file value. The result
// must be a valid http(s) URL so a malformed env var fails loudly instead of
// producing confusing "connection refused" errors on every command.
func (c *Config) ResolveAPIURL(flagURL string) (string, error) {
	candidate := flagURL
	if candidate == "" {
		candidate = strings.TrimSpace(os.Getenv(EnvAPIURL))
	}
	if candidate == "" {
		candidate = c.APIURL
	}
	if candidate == "" {
		return "", errors.New("no API URL configured (set api_url in the config file, ZETTELGARDEN_API_URL, or --url)")
	}
	u, err := url.Parse(candidate)
	if err != nil || (u.Scheme != "http" && u.Scheme != "https") || u.Host == "" {
		return "", errors.New("invalid API URL " + fmt.Sprintf("%q", candidate) + " (must be an http(s) URL)")
	}
	return candidate, nil
}
