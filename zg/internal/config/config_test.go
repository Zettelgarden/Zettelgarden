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

func TestResolveAPIURLPrecedence(t *testing.T) {
	const (
		flagURL = "http://flag.example"
		envURL  = "http://env.example"
		cfgURL  = "http://config.example"
	)

	t.Run("flag beats env and config", func(t *testing.T) {
		t.Setenv(EnvAPIURL, envURL)
		cfg := &Config{APIURL: cfgURL}
		got, err := cfg.ResolveAPIURL(flagURL)
		if err != nil || got != flagURL {
			t.Fatalf("got %q err=%v, want %q", got, err, flagURL)
		}
	})

	t.Run("env beats config", func(t *testing.T) {
		t.Setenv(EnvAPIURL, envURL)
		cfg := &Config{APIURL: cfgURL}
		got, err := cfg.ResolveAPIURL("")
		if err != nil || got != envURL {
			t.Fatalf("got %q err=%v, want %q", got, err, envURL)
		}
	})

	t.Run("config file fallback", func(t *testing.T) {
		t.Setenv(EnvAPIURL, "")
		cfg := &Config{APIURL: cfgURL}
		got, err := cfg.ResolveAPIURL("")
		if err != nil || got != cfgURL {
			t.Fatalf("got %q err=%v, want %q", got, err, cfgURL)
		}
	})

	t.Run("env value is trimmed", func(t *testing.T) {
		t.Setenv(EnvAPIURL, "  "+envURL+"  ")
		cfg := &Config{APIURL: cfgURL}
		got, err := cfg.ResolveAPIURL("")
		if err != nil || got != envURL {
			t.Fatalf("got %q err=%v, want trimmed %q", got, err, envURL)
		}
	})

	t.Run("nothing configured", func(t *testing.T) {
		t.Setenv(EnvAPIURL, "")
		cfg := &Config{}
		if _, err := cfg.ResolveAPIURL(""); err == nil {
			t.Fatal("expected error when no URL is configured anywhere")
		}
	})
}

func TestResolveAPIURLValidation(t *testing.T) {
	cases := []struct {
		name string
		url  string
		want bool // true = valid
	}{
		{"http", "http://localhost:8080", true},
		{"https", "https://api.example.com", true},
		{"missing scheme", "localhost:8080", false},
		{"bad scheme", "ftp://example.com", false},
		{"empty host", "http://", false},
		{"garbage", "::not a url::", false},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			t.Setenv(EnvAPIURL, tc.url)
			_, err := (&Config{}).ResolveAPIURL("")
			if (err == nil) != tc.want {
				t.Errorf("ResolveAPIURL(%q) err=%v, want valid=%v", tc.url, err, tc.want)
			}
		})
	}
}
