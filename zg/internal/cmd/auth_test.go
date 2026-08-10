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
	t.Setenv(config.EnvToken, "")  // neutralize any ambient ZETTELGARDEN_TOKEN
	t.Setenv(config.EnvAPIURL, "") // neutralize any ambient ZETTELGARDEN_API_URL
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

func TestLoadConfigEnvURLBeatsConfig(t *testing.T) {
	writeConfigContent(t, `{"api_url": "http://config.example", "token": ""}`)
	t.Setenv(config.EnvAPIURL, "http://env.example")

	cfg, err := loadConfig()
	if err != nil {
		t.Fatalf("loadConfig: %v", err)
	}
	if cfg.APIURL != "http://env.example" {
		t.Errorf("api_url = %q, want env override http://env.example", cfg.APIURL)
	}
}

func TestLoadConfigFlagBeatsEnvURL(t *testing.T) {
	writeConfigContent(t, `{"api_url": "http://config.example", "token": ""}`)
	t.Setenv(config.EnvAPIURL, "http://env.example")
	SetAPIURL("http://flag.example")
	t.Cleanup(func() { SetAPIURL("") })

	cfg, err := loadConfig()
	if err != nil {
		t.Fatalf("loadConfig: %v", err)
	}
	if cfg.APIURL != "http://flag.example" {
		t.Errorf("api_url = %q, want flag override http://flag.example", cfg.APIURL)
	}
}

func TestLoadConfigRejectsInvalidEnvURL(t *testing.T) {
	writeConfigContent(t, `{"api_url": "http://config.example", "token": ""}`)
	t.Setenv(config.EnvAPIURL, "not-a-url")

	if _, err := loadConfig(); err == nil {
		t.Fatal("expected error for invalid ZETTELGARDEN_API_URL")
	}
}

// newAuthServer mocks the backend /api/login and /api/api-keys endpoints used
// by `zg auth login`. It records the created key name, the email/password
// received at login, and the Authorization header sent to each endpoint so
// tests can assert the CLI doesn't send the raw password as a bearer token.
func newAuthServer(t *testing.T, loginErr string) (*httptest.Server, *string, *map[string]string, *string) {
	t.Helper()
	var createdName string
	loginReceived := map[string]string{}
	var apiKeysAuth string
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		switch r.URL.Path {
		case "/api/login":
			if loginErr != "" {
				w.WriteHeader(http.StatusUnauthorized)
				json.NewEncoder(w).Encode(map[string]string{"message": loginErr})
				return
			}
			var body map[string]string
			json.NewDecoder(r.Body).Decode(&body)
			loginReceived["email"] = body["email"]
			loginReceived["password"] = body["password"]
			json.NewEncoder(w).Encode(map[string]string{"access_token": "jwt-token"})
		case "/api/api-keys":
			apiKeysAuth = r.Header.Get("Authorization")
			if r.Method != http.MethodPost {
				w.WriteHeader(http.StatusMethodNotAllowed)
				return
			}
			var body map[string]string
			json.NewDecoder(r.Body).Decode(&body)
			createdName = body["name"]
			if body["name"] == "taken" {
				w.WriteHeader(http.StatusConflict)
				json.NewEncoder(w).Encode(map[string]string{"message": "API key with this name already exists"})
				return
			}
			w.WriteHeader(http.StatusCreated)
			json.NewEncoder(w).Encode(map[string]any{"id": 7, "name": body["name"], "key": "zg_live_testkey"})
		default:
			w.WriteHeader(http.StatusNotFound)
		}
	}))
	t.Cleanup(server.Close)
	return server, &createdName, &loginReceived, &apiKeysAuth
}

func TestRunAuthLogin(t *testing.T) {
	t.Setenv(config.EnvNoKeyring, "1") // config-file storage for determinism
	server, createdName, loginReceived, apiKeysAuth := newAuthServer(t, "")

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
	// Credentials must reach /api/login and the session JWT (never the raw
	// password) must be the bearer token on /api/api-keys.
	if (*loginReceived)["email"] != "test@example.com" || (*loginReceived)["password"] != "secret" {
		t.Errorf("login received %v, want email+password", *loginReceived)
	}
	if *apiKeysAuth != "Bearer jwt-token" {
		t.Errorf("api-keys Authorization = %q, want Bearer jwt-token", *apiKeysAuth)
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
	server, _, _, _ := newAuthServer(t, "Invalid credentials")

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
	server, _, _, _ := newAuthServer(t, "")

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

// newAuthStatusServer mocks GET /api/api-keys and DELETE /api/api-keys/{id}.
func newAuthStatusServer(t *testing.T, listStatus int, listBody string, deleteCalls *[]string) *httptest.Server {
	t.Helper()
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		switch {
		case r.Method == http.MethodGet && r.URL.Path == "/api/api-keys":
			w.Header().Set("Content-Type", "application/json")
			w.WriteHeader(listStatus)
			w.Write([]byte(listBody))
		case r.Method == http.MethodDelete && strings.HasPrefix(r.URL.Path, "/api/api-keys/"):
			*deleteCalls = append(*deleteCalls, r.URL.Path)
			w.WriteHeader(http.StatusNoContent)
		default:
			w.WriteHeader(http.StatusNotFound)
		}
	}))
	t.Cleanup(server.Close)
	return server
}

func TestRunAuthSet(t *testing.T) {
	t.Setenv(config.EnvNoKeyring, "1")
	writeConfigContent(t, `{"api_url": "http://example.com", "token": ""}`)
	authKeyName = "my-key"
	t.Cleanup(func() { authKeyName = "" })

	out := captureStdout(t, func() {
		if err := runAuthSet(authSetCmd, []string{"zg_live_manualkey"}); err != nil {
			t.Fatalf("runAuthSet: %v", err)
		}
	})
	if !strings.Contains(out, "stored in config") {
		t.Errorf("expected success message, got %q", out)
	}
	cfg, err := loadConfig()
	if err != nil {
		t.Fatalf("loadConfig: %v", err)
	}
	if cfg.Token != "zg_live_manualkey" || cfg.APIKeyName != "my-key" {
		t.Errorf("cfg = %q/%q, want zg_live_manualkey/my-key", cfg.Token, cfg.APIKeyName)
	}
}

func TestRunAuthSetWarnsOnJWT(t *testing.T) {
	t.Setenv(config.EnvNoKeyring, "1")
	writeConfigContent(t, `{"api_url": "http://example.com", "token": ""}`)

	stderr := captureStderr(t, func() {
		if err := runAuthSet(authSetCmd, []string{jwtLike}); err != nil {
			t.Fatalf("runAuthSet: %v", err)
		}
	})
	if !strings.Contains(stderr, "short-lived JWT") {
		t.Errorf("expected JWT warning on stderr, got %q", stderr)
	}
}

func TestRunAuthRevokeWithServerID(t *testing.T) {
	t.Setenv(config.EnvNoKeyring, "1")
	var deleteCalls []string
	server := newAuthStatusServer(t, 200, `{"api_keys":[]}`, &deleteCalls)

	writeConfigContent(t, `{"api_url": "`+server.URL+`", "token": "zg_live_testkey", "api_key_name": "zg-cli", "api_key_id": 7}`)
	SetAPIURL(server.URL)
	t.Cleanup(func() { SetAPIURL("") })

	out := captureStdout(t, func() {
		if err := runAuthRevoke(authRevokeCmd, nil); err != nil {
			t.Fatalf("runAuthRevoke: %v", err)
		}
	})
	if len(deleteCalls) != 1 || deleteCalls[0] != "/api/api-keys/7" {
		t.Errorf("delete calls = %v, want [/api/api-keys/7]", deleteCalls)
	}
	if !strings.Contains(out, "revoked on server") {
		t.Errorf("expected server-revoke success message, got %q", out)
	}

	// Local storage must be fully cleared, including the key metadata.
	cfg, err := loadConfig()
	if err != nil {
		t.Fatalf("loadConfig: %v", err)
	}
	if cfg.Token != "" || cfg.APIKeyName != "" || cfg.APIKeyID != 0 {
		t.Errorf("config not cleared after revoke: %+v", cfg)
	}
}

func TestRunAuthRevokeWithoutServerID(t *testing.T) {
	t.Setenv(config.EnvNoKeyring, "1")
	var deleteCalls []string
	server := newAuthStatusServer(t, 200, `{"api_keys":[]}`, &deleteCalls)

	// A key stored via `zg auth set` has no recorded server-side id.
	writeConfigContent(t, `{"api_url": "`+server.URL+`", "token": "zg_live_testkey"}`)
	SetAPIURL(server.URL)
	t.Cleanup(func() { SetAPIURL("") })

	out := captureStdout(t, func() {
		if err := runAuthRevoke(authRevokeCmd, nil); err != nil {
			t.Fatalf("runAuthRevoke: %v", err)
		}
	})
	if len(deleteCalls) != 0 {
		t.Errorf("server revoke attempted without an id: %v", deleteCalls)
	}
	if !strings.Contains(out, "no API key id recorded") {
		t.Errorf("expected skip note in message, got %q", out)
	}
}

func TestRunAuthStatusJWTWarningAndValidity(t *testing.T) {
	t.Setenv(config.EnvNoKeyring, "1")
	t.Setenv(config.EnvToken, "")
	server := newAuthStatusServer(t, 200, `{"api_keys":[{"id":7,"name":"zg-cli"}]}`, &[]string{})

	// Write the config with world-readable perms to also exercise the perm warning.
	tmpDir := t.TempDir()
	configPath := filepath.Join(tmpDir, "config.json")
	content := `{"api_url": "` + server.URL + `", "token": "` + jwtLike + `"}`
	if err := os.WriteFile(configPath, []byte(content), 0644); err != nil {
		t.Fatal(err)
	}
	SetCfgFile(configPath)
	SetAPIURL(server.URL)
	t.Cleanup(func() { SetCfgFile(""); SetAPIURL(""); SetAPIToken("") })

	out := captureStdout(t, func() {
		if err := runAuthStatus(authStatusCmd, nil); err != nil {
			t.Fatalf("runAuthStatus: %v", err)
		}
	})
	if !strings.Contains(out, `"token_source":"config"`) {
		t.Errorf("expected config token source, got %q", out)
	}
	if !strings.Contains(out, "short-lived JWT") {
		t.Errorf("expected JWT warning in status output, got %q", out)
	}
	if !strings.Contains(out, "world-readable") {
		t.Errorf("expected perm warning in status output, got %q", out)
	}
	if !strings.Contains(out, `"token_valid":true`) {
		t.Errorf("expected token_valid true for a real key-list response, got %q", out)
	}
}

func TestRunAuthStatusRejectsHTMLFallback(t *testing.T) {
	t.Setenv(config.EnvNoKeyring, "1")
	t.Setenv(config.EnvToken, "")
	// A dev-server SPA fallback answers 200 with HTML; that must NOT count as valid.
	server := newAuthStatusServer(t, 200, `<!doctype html><html></html>`, &[]string{})

	writeConfigContent(t, `{"api_url": "`+server.URL+`", "token": "zg_live_testkey"}`)
	SetAPIURL(server.URL)
	t.Cleanup(func() { SetAPIURL("") })

	out := captureStdout(t, func() {
		if err := runAuthStatus(authStatusCmd, nil); err != nil {
			t.Fatalf("runAuthStatus: %v", err)
		}
	})
	if !strings.Contains(out, `"token_valid":false`) {
		t.Errorf("expected token_valid false for HTML response, got %q", out)
	}
}
