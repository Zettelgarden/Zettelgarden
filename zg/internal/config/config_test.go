package config

import (
	"os"
	"path/filepath"
	"testing"
)

func TestLoadConfig(t *testing.T) {
	// Create temp config dir
	tmpDir := t.TempDir()
	configPath := filepath.Join(tmpDir, "config.json")

	// Write test config
	configContent := `{"api_url": "http://test.local:8080", "token": "test-token"}`
	if err := os.WriteFile(configPath, []byte(configContent), 0600); err != nil {
		t.Fatal(err)
	}

	// Load config
	cfg, err := LoadConfig(configPath)
	if err != nil {
		t.Fatalf("LoadConfig failed: %v", err)
	}

	// Verify values
	if cfg.APIURL != "http://test.local:8080" {
		t.Errorf("Expected APIURL 'http://test.local:8080', got '%s'", cfg.APIURL)
	}
	if cfg.Token != "test-token" {
		t.Errorf("Expected Token 'test-token', got '%s'", cfg.Token)
	}
}

func TestLoadConfigDefaults(t *testing.T) {
	tmpDir := t.TempDir()
	configPath := filepath.Join(tmpDir, "config.json")

	// Write minimal config
	configContent := `{"token": "test-token"}`
	if err := os.WriteFile(configPath, []byte(configContent), 0600); err != nil {
		t.Fatal(err)
	}

	cfg, err := LoadConfig(configPath)
	if err != nil {
		t.Fatalf("LoadConfig failed: %v", err)
	}

	// Verify defaults
	if cfg.APIURL != "http://localhost:8080" {
		t.Errorf("Expected default APIURL 'http://localhost:8080', got '%s'", cfg.APIURL)
	}
	if cfg.TimeoutSeconds != 30 {
		t.Errorf("Expected default TimeoutSeconds 30, got %d", cfg.TimeoutSeconds)
	}
}

func TestLoadConfigNegativeTimeout(t *testing.T) {
	tmpDir := t.TempDir()
	configPath := filepath.Join(tmpDir, "config.json")

	// Write config with negative timeout
	configContent := `{"token": "test-token", "timeout_seconds": -5}`
	if err := os.WriteFile(configPath, []byte(configContent), 0600); err != nil {
		t.Fatal(err)
	}

	cfg, err := LoadConfig(configPath)
	if err != nil {
		t.Fatalf("LoadConfig failed: %v", err)
	}

	// Verify negative timeout is replaced with default
	if cfg.TimeoutSeconds != 30 {
		t.Errorf("Expected default TimeoutSeconds 30 for negative input, got %d", cfg.TimeoutSeconds)
	}
}
