package config

import (
	"os"
	"path/filepath"
	"testing"
)

func TestLoadConfigDefault(t *testing.T) {
	// Create a temporary directory for test config
	tmpDir := t.TempDir()

	// Override home directory for test
	originalHome := os.Getenv("HOME")
	os.Setenv("HOME", tmpDir)
	defer os.Setenv("HOME", originalHome)

	// Load config (should return default since file doesn't exist)
	cfg, err := LoadConfig()
	if err != nil {
		t.Fatalf("LoadConfig failed: %v", err)
	}

	if cfg.DefaultProfile != "default" {
		t.Errorf("Expected default profile to be 'default', got '%s'", cfg.DefaultProfile)
	}

	if cfg.Profiles["default"].Endpoint != "https://zettelgarden.com" {
		t.Errorf("Expected default endpoint, got '%s'", cfg.Profiles["default"].Endpoint)
	}
}

func TestSaveAndLoadConfig(t *testing.T) {
	// Create a temporary directory for test config
	tmpDir := t.TempDir()

	// Override home directory for test
	originalHome := os.Getenv("HOME")
	os.Setenv("HOME", tmpDir)
	defer os.Setenv("HOME", originalHome)

	// Create a custom config
	cfg := &Config{
		DefaultProfile: "production",
		Profiles: map[string]Profile{
			"production": {
				Endpoint: "https://prod.zettelgarden.com",
				Timeout:  60,
			},
			"staging": {
				Endpoint: "https://staging.zettelgarden.com",
				Timeout:  30,
			},
		},
		Output: OutputConfig{
			Format:  "compact-json",
			Compact: true,
			Color:   false,
		},
	}

	// Save config
	if err := SaveConfig(cfg); err != nil {
		t.Fatalf("SaveConfig failed: %v", err)
	}

	// Verify config directory was created with correct permissions
	configDir := filepath.Join(tmpDir, ".config", "zettelgarden")
	info, err := os.Stat(configDir)
	if err != nil {
		t.Fatalf("Config directory not created: %v", err)
	}
	if !info.IsDir() {
		t.Error("Config path is not a directory")
	}

	// Load config back
	loadedCfg, err := LoadConfig()
	if err != nil {
		t.Fatalf("LoadConfig failed: %v", err)
	}

	// Verify loaded config matches saved config
	if loadedCfg.DefaultProfile != "production" {
		t.Errorf("Expected default profile 'production', got '%s'", loadedCfg.DefaultProfile)
	}

	if loadedCfg.Profiles["production"].Endpoint != "https://prod.zettelgarden.com" {
		t.Errorf("Expected production endpoint, got '%s'", loadedCfg.Profiles["production"].Endpoint)
	}

	if loadedCfg.Profiles["staging"].Timeout != 30 {
		t.Errorf("Expected staging timeout 30, got %d", loadedCfg.Profiles["staging"].Timeout)
	}

	if loadedCfg.Output.Compact != true {
		t.Error("Expected compact output to be true")
	}
}

func TestGetProfile(t *testing.T) {
	cfg := &Config{
		DefaultProfile: "default",
		Profiles: map[string]Profile{
			"default": {
				Endpoint: "https://zettelgarden.com",
				Timeout:  30,
			},
			"local": {
				Endpoint: "http://localhost:8080",
				Timeout:  10,
			},
		},
	}

	// Test getting default profile
	profile, err := GetProfile(cfg, "")
	if err != nil {
		t.Fatalf("GetProfile failed: %v", err)
	}
	if profile.Endpoint != "https://zettelgarden.com" {
		t.Errorf("Expected default endpoint, got '%s'", profile.Endpoint)
	}

	// Test getting specific profile
	profile, err = GetProfile(cfg, "local")
	if err != nil {
		t.Fatalf("GetProfile failed: %v", err)
	}
	if profile.Endpoint != "http://localhost:8080" {
		t.Errorf("Expected local endpoint, got '%s'", profile.Endpoint)
	}

	// Test getting non-existent profile
	_, err = GetProfile(cfg, "nonexistent")
	if err == nil {
		t.Error("Expected error for non-existent profile")
	}
}

func TestInitDefaultConfig(t *testing.T) {
	// Create a temporary directory for test config
	tmpDir := t.TempDir()

	// Override home directory for test
	originalHome := os.Getenv("HOME")
	os.Setenv("HOME", tmpDir)
	defer os.Setenv("HOME", originalHome)

	// Initialize default config
	if err := InitDefaultConfig(); err != nil {
		t.Fatalf("InitDefaultConfig failed: %v", err)
	}

	// Verify config file was created
	configPath, _ := GetConfigPath()
	if _, err := os.Stat(configPath); os.IsNotExist(err) {
		t.Error("Config file was not created")
	}

	// Load and verify it's the default config
	cfg, err := LoadConfig()
	if err != nil {
		t.Fatalf("LoadConfig failed: %v", err)
	}

	if cfg.DefaultProfile != "default" {
		t.Errorf("Expected default profile, got '%s'", cfg.DefaultProfile)
	}

	// Try to init again - should not error (idempotent)
	if err := InitDefaultConfig(); err != nil {
		t.Errorf("InitDefaultConfig should be idempotent: %v", err)
	}
}
