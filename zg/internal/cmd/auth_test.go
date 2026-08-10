package cmd

import (
	"encoding/json"
	"io"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/nick-zettelgarden/zg/internal/config"
)

// writeConfigContent writes the given JSON to a temp config file and points
// the CLI at it, resetting all global overrides on cleanup. The OS keyring is
// disabled so token resolution is deterministic across machines.
func writeConfigContent(t *testing.T, content string) {
	t.Helper()
	t.Setenv(config.EnvNoKeyring, "1")
	t.Setenv(config.EnvToken, "") // neutralize any ambient ZETTELGARDEN_TOKEN
	tmpDir := t.TempDir()
	configPath := filepath.Join(tmpDir, "config.json")
	if err := os.WriteFile(configPath, []byte(content), 0600); err != nil {
		t.Fatal(err)
	}
	SetCfgFile(configPath)
	t.Cleanup(func() {
		SetCfgFile("")
		SetAPIURL("")
		SetAPIToken("")
	})
}

// captureStderr runs fn with os.Stderr redirected to a pipe and returns
// everything written to it.
func captureStderr(t *testing.T, fn func()) string {
	t.Helper()
	oldStderr := os.Stderr
	r, w, err := os.Pipe()
	if err != nil {
		t.Fatal(err)
	}
	os.Stderr = w
	defer func() { os.Stderr = oldStderr }()

	fn()

	if err := w.Close(); err != nil {
		t.Fatal(err)
	}
	out, err := io.ReadAll(r)
	if err != nil {
		t.Fatal(err)
	}
	return string(out)
}

// jwtLike mimics a short-lived session JWT (header.payload.signature).
const jwtLike = "eyJhbGciOiJIUzI1NiJ9.eyJzdWIiOiIxIn0.signature"

func TestLoadConfigWarnsOnJWTTokens(t *testing.T) {
	t.Setenv(config.EnvToken, "")
	writeConfigContent(t, `{"api_url": "http://example.com", "token": "`+jwtLike+`"}`)

	var cfg *config.Config
	var err error
	stderr := captureStderr(t, func() {
		cfg, err = loadConfig()
	})
	if err != nil {
		t.Fatalf("loadConfig: %v", err)
	}
	if cfg.Token != jwtLike {
		t.Errorf("token = %q, want %q", cfg.Token, jwtLike)
	}
	if !strings.Contains(stderr, "short-lived JWT") {
		t.Errorf("expected JWT warning on stderr, got %q", stderr)
	}
}

func TestLoadConfigNoWarningForAPIKey(t *testing.T) {
	t.Setenv(config.EnvToken, "")
	writeConfigContent(t, `{"api_url": "http://example.com", "token": "zg_live_abc"}`)

	stderr := captureStderr(t, func() {
		if _, err := loadConfig(); err != nil {
			t.Fatalf("loadConfig: %v", err)
		}
	})
	if strings.Contains(stderr, "warning") {
		t.Errorf("unexpected warning for API key: %q", stderr)
	}
}

func TestLoadConfigFlagBeatsEnvAndConfig(t *testing.T) {
	t.Setenv(config.EnvToken, "env-token")
	writeConfigContent(t, `{"api_url": "http://example.com", "token": "config-token"}`)
	SetAPIToken("flag-token")
	t.Cleanup(func() { SetAPIToken("") })

	cfg, err := loadConfig()
	if err != nil {
		t.Fatalf("loadConfig: %v", err)
	}
	if cfg.Token != "flag-token" {
		t.Errorf("token = %q, want flag-token", cfg.Token)
	}
}

func TestLoadConfigEnvBeatsConfig(t *testing.T) {
	writeConfigContent(t, `{"api_url": "http://example.com", "token": "config-token"}`)
	t.Setenv(config.EnvToken, "env-token")

	cfg, err := loadConfig()
	if err != nil {
		t.Fatalf("loadConfig: %v", err)
	}
	if cfg.Token != "env-token" {
		t.Errorf("token = %q, want env-token", cfg.Token)
	}
}

// newAuthServer mocks the backend /api/login and /api/api-keys endpoints used
// by `zg auth login`. It records the created key name.
func newAuthServer(t *testing.T, loginErr string) (*httptest.Server, *string) {
	t.Helper()
	var createdName string
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		switch r.URL.Path {
		case "/api/login":
			if loginErr != "" {
				w.WriteHeader(http.StatusUnauthorized)
				json.NewEncoder(w).Encode(map[string]string{"message": loginErr})
				return
			}
			json.NewEncoder(w).Encode(map[string]string{"access_token": "jwt-token"})
		case "/api/api-keys":
			if r.Method != http.MethodPost {
				w.WriteHeader(http.StatusMethodNotAllowed)
				return
			}
			var body map[string]string
			json.NewDecoder(r.Body).Decode(&body)
			createdName = body["name"]
			if body["name"] == "taken" {
				w.WriteHeader(http.StatusConflict)
				return
			}
			w.WriteHeader(http.StatusCreated)
			json.NewEncoder(w).Encode(map[string]any{"id": 7, "name": body["name"], "key": "zg_live_testkey"})
		default:
			w.WriteHeader(http.StatusNotFound)
		}
	}))
	t.Cleanup(server.Close)
	return server, &createdName
}

func TestRunAuthLogin(t *testing.T) {
	t.Setenv(config.EnvNoKeyring, "1") // config-file storage for determinism
	server, createdName := newAuthServer(t, "")

	writeConfigContent(t, `{"api_url": "`+server.URL+`", "token": ""}`)
	SetAPIURL(server.URL)
	t.Cleanup(func() { SetAPIURL("") })

	authEmail = "test@example.com"
	authPassword = "secret"
	authKeyName = "zg-cli"
	t.Cleanup(func() { authEmail, authPassword, authKeyName = "", "", "" })

	out := captureStdout(t, func() {
		if err := runAuthLogin(authLoginCmd, nil); err != nil {
			t.Fatalf("runAuthLogin: %v", err)
		}
	})

	if *createdName != "zg-cli" {
		t.Errorf("api key created with name %q, want zg-cli", *createdName)
	}
	if !strings.Contains(out, "stored in config") {
		t.Errorf("expected success message, got %q", out)
	}

	// The durable key and its metadata must be stored and resolvable.
	cfg, err := loadConfig()
	if err != nil {
		t.Fatalf("loadConfig: %v", err)
	}
	if cfg.Token != "zg_live_testkey" {
		t.Errorf("cfg.Token = %q, want zg_live_testkey", cfg.Token)
	}
	if cfg.APIKeyName != "zg-cli" || cfg.APIKeyID != 7 {
		t.Errorf("api key metadata = %q/%d, want zg-cli/7", cfg.APIKeyName, cfg.APIKeyID)
	}
}

func TestRunAuthLoginBadCredentials(t *testing.T) {
	t.Setenv(config.EnvNoKeyring, "1")
	server, _ := newAuthServer(t, "Invalid credentials")

	writeConfigContent(t, `{"api_url": "`+server.URL+`", "token": ""}`)
	SetAPIURL(server.URL)
	t.Cleanup(func() { SetAPIURL("") })
	authEmail = "test@example.com"
	authPassword = "wrong"
	authKeyName = "zg-cli"
	t.Cleanup(func() { authEmail, authPassword, authKeyName = "", "", "" })

	out := captureStdout(t, func() {
		if err := runAuthLogin(authLoginCmd, nil); err != nil {
			t.Fatalf("runAuthLogin: %v", err)
		}
	})
	if !strings.Contains(out, "Invalid credentials") {
		t.Errorf("expected credential error in output, got %q", out)
	}
}

func TestRunAuthLoginNameConflict(t *testing.T) {
	t.Setenv(config.EnvNoKeyring, "1")
	server, _ := newAuthServer(t, "")

	writeConfigContent(t, `{"api_url": "`+server.URL+`", "token": ""}`)
	SetAPIURL(server.URL)
	t.Cleanup(func() { SetAPIURL("") })
	authEmail = "test@example.com"
	authPassword = "secret"
	authKeyName = "taken"
	t.Cleanup(func() { authEmail, authPassword, authKeyName = "", "", "" })

	out := captureStdout(t, func() {
		if err := runAuthLogin(authLoginCmd, nil); err != nil {
			t.Fatalf("runAuthLogin: %v", err)
		}
	})
	if !strings.Contains(out, "already in use") {
		t.Errorf("expected name-conflict error, got %q", out)
	}
}
