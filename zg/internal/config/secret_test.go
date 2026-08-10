package config

import (
	"errors"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"
)

// fakeKeyring is an in-memory keyringOperator for tests.
type fakeKeyring struct {
	token string
	err   error
}

func (f *fakeKeyring) Set(token string) error {
	if f.err != nil {
		return f.err
	}
	f.token = token
	return nil
}

func (f *fakeKeyring) Get() (string, error) {
	if f.err != nil {
		return "", f.err
	}
	if f.token == "" {
		return "", errors.New("no item")
	}
	return f.token, nil
}

func (f *fakeKeyring) Delete() error {
	f.token = ""
	return nil
}

// failingKeyring always errors, simulating a headless system with no OS keyring.
type failingKeyring struct{}

func (failingKeyring) Set(string) error { return errors.New("no keyring provider") }
func (failingKeyring) Get() (string, error) {
	return "", errors.New("no keyring provider")
}
func (failingKeyring) Delete() error { return errors.New("no keyring provider") }

// hangingKeyring blocks forever, simulating a stale DBus session bus.
type hangingKeyring struct{}

func (hangingKeyring) Set(string) error     { select {} }
func (hangingKeyring) Get() (string, error) { select {} }
func (hangingKeyring) Delete() error        { select {} }

func TestResolveTokenKeyringTimeoutFallsBack(t *testing.T) {
	t.Setenv(EnvToken, "")
	keyringTimeout = 50 * time.Millisecond
	defer func() { keyringTimeout = 3 * time.Second }()
	keyringOp = hangingKeyring{}

	cfg := &Config{Token: "config-token"}
	start := time.Now()
	token, source, err := cfg.ResolveToken("")
	if err != nil {
		t.Fatalf("ResolveToken: %v", err)
	}
	if source != TokenFromConfig || token != "config-token" {
		t.Fatalf("got token=%q source=%q, want config fallback", token, source)
	}
	if elapsed := time.Since(start); elapsed > 2*time.Second {
		t.Errorf("fallback took %v; keyring timeout not applied", elapsed)
	}
}

func TestStoreTokenKeyringTimeoutFallsBack(t *testing.T) {
	t.Setenv(EnvToken, "")
	keyringTimeout = 50 * time.Millisecond
	defer func() { keyringTimeout = 3 * time.Second }()
	keyringOp = hangingKeyring{}

	dir := t.TempDir()
	path := filepath.Join(dir, "config.json")
	cfg := &Config{APIURL: "http://example.com"}
	stored, err := StoreToken(path, cfg, "zg_live_secret")
	if err != nil {
		t.Fatalf("StoreToken: %v", err)
	}
	if stored != "config" {
		t.Errorf("stored in %q, want config fallback", stored)
	}
}

func TestResolveTokenPrecedence(t *testing.T) {
	const (
		flagToken = "flag-token"
		envToken  = "env-token"
		keyToken  = "keyring-token"
		cfgToken  = "config-token"
	)

	t.Run("flag beats everything", func(t *testing.T) {
		t.Setenv(EnvToken, envToken)
		keyringOp = &fakeKeyring{token: keyToken}
		cfg := &Config{Token: cfgToken}
		token, source, err := cfg.ResolveToken(flagToken)
		if err != nil || token != flagToken || source != TokenFromFlag {
			t.Fatalf("got token=%q source=%q err=%v, want %q/%q", token, source, err, flagToken, TokenFromFlag)
		}
	})

	t.Run("env beats keyring and config", func(t *testing.T) {
		t.Setenv(EnvToken, envToken)
		keyringOp = &fakeKeyring{token: keyToken}
		cfg := &Config{Token: cfgToken}
		token, source, err := cfg.ResolveToken("")
		if err != nil || token != envToken || source != TokenFromEnv {
			t.Fatalf("got token=%q source=%q err=%v, want %q/%q", token, source, err, envToken, TokenFromEnv)
		}
	})

	t.Run("keyring beats config", func(t *testing.T) {
		t.Setenv(EnvToken, "")
		keyringOp = &fakeKeyring{token: keyToken}
		cfg := &Config{Token: cfgToken}
		token, source, err := cfg.ResolveToken("")
		if err != nil || token != keyToken || source != TokenFromKeyring {
			t.Fatalf("got token=%q source=%q err=%v, want %q/%q", token, source, err, keyToken, TokenFromKeyring)
		}
	})

	t.Run("config file fallback", func(t *testing.T) {
		t.Setenv(EnvToken, "")
		keyringOp = failingKeyring{}
		cfg := &Config{Token: cfgToken}
		token, source, err := cfg.ResolveToken("")
		if err != nil || token != cfgToken || source != TokenFromConfig {
			t.Fatalf("got token=%q source=%q err=%v, want %q/%q", token, source, err, cfgToken, TokenFromConfig)
		}
	})

	t.Run("no token anywhere", func(t *testing.T) {
		t.Setenv(EnvToken, "")
		keyringOp = failingKeyring{}
		cfg := &Config{}
		token, source, err := cfg.ResolveToken("")
		if err != nil || token != "" || source != TokenFromNone {
			t.Fatalf("got token=%q source=%q err=%v, want empty/none", token, source, err)
		}
	})
}

func TestIsJWT(t *testing.T) {
	cases := []struct {
		name  string
		token string
		want  bool
	}{
		{"jwt", "eyJhbGciOiJIUzI1NiJ9.eyJzdWIiOiIxIn0.signature", true},
		{"jwt short", "eyJhbGciOiJIUzI1NiJ9.eyJzdWIiOiIxIn0", true},
		{"api key", "zg_live_0123456789abcdef0123456789abcdef0123456789abcdef0123456789abcdef", false},
		{"garbage", "hello-world", false},
		{"empty", "", false},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			if got := IsJWT(tc.token); got != tc.want {
				t.Errorf("IsJWT(%q) = %v, want %v", tc.token, got, tc.want)
			}
		})
	}
}

func TestIsAPIKey(t *testing.T) {
	if !IsAPIKey("zg_live_abc") {
		t.Error("expected zg_live_ prefix to be an API key")
	}
	if IsAPIKey("eyJhbGciOiJIUzI1NiJ9.x.y") {
		t.Error("JWT should not be an API key")
	}
}

func TestSaveConfigPermissions(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "nested", "config.json") // exercise MkdirAll
	cfg := &Config{APIURL: "http://example.com", TimeoutSeconds: 30, APIKeyName: "zg-cli"}

	if err := SaveConfig(path, cfg); err != nil {
		t.Fatalf("SaveConfig: %v", err)
	}

	info, err := os.Stat(path)
	if err != nil {
		t.Fatalf("stat: %v", err)
	}
	if perm := info.Mode().Perm(); perm != 0600 {
		t.Errorf("config file perms = %v, want 0600", perm)
	}
	dirInfo, err := os.Stat(filepath.Dir(path))
	if err != nil {
		t.Fatalf("stat dir: %v", err)
	}
	if perm := dirInfo.Mode().Perm(); perm != 0700 {
		t.Errorf("config dir perms = %v, want 0700", perm)
	}
}

func TestStoreTokenKeyringPreferred(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "config.json")
	keyringOp = &fakeKeyring{}

	cfg := &Config{APIURL: "http://example.com"}
	stored, err := StoreToken(path, cfg, "zg_live_secret")
	if err != nil {
		t.Fatalf("StoreToken: %v", err)
	}
	if stored != "keyring" {
		t.Errorf("stored in %q, want keyring", stored)
	}
	// The plaintext must not be written to the config file when the keyring works.
	data, err := os.ReadFile(path)
	if err != nil {
		t.Fatalf("read config: %v", err)
	}
	if contains := strings.Contains(string(data), "zg_live_secret"); contains {
		t.Errorf("config file leaked plaintext token: %s", data)
	}
	// Resolution should come back from the keyring.
	t.Setenv(EnvToken, "")
	token, source, err := cfg.ResolveToken("")
	if err != nil || token != "zg_live_secret" || source != TokenFromKeyring {
		t.Fatalf("resolve: token=%q source=%q err=%v", token, source, err)
	}
}

func TestStoreTokenConfigFallback(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "config.json")
	keyringOp = failingKeyring{}

	cfg := &Config{APIURL: "http://example.com", TimeoutSeconds: 30}
	stored, err := StoreToken(path, cfg, "zg_live_secret")
	if err != nil {
		t.Fatalf("StoreToken: %v", err)
	}
	if stored != "config" {
		t.Errorf("stored in %q, want config", stored)
	}
	info, err := os.Stat(path)
	if err != nil {
		t.Fatalf("stat: %v", err)
	}
	if perm := info.Mode().Perm(); perm != 0600 {
		t.Errorf("config file perms = %v, want 0600", perm)
	}
	// Resolution falls back to the config file token.
	t.Setenv(EnvToken, "")
	token, source, err := cfg.ResolveToken("")
	if err != nil || token != "zg_live_secret" || source != TokenFromConfig {
		t.Fatalf("resolve: token=%q source=%q err=%v", token, source, err)
	}
}

func TestClearToken(t *testing.T) {
	t.Setenv(EnvToken, "") // neutralize any ambient ZETTELGARDEN_TOKEN
	dir := t.TempDir()
	path := filepath.Join(dir, "config.json")
	keyringOp = &fakeKeyring{}

	cfg := &Config{APIURL: "http://example.com"}
	if _, err := StoreToken(path, cfg, "zg_live_secret"); err != nil {
		t.Fatalf("StoreToken: %v", err)
	}
	if err := ClearToken(path, cfg); err != nil {
		t.Fatalf("ClearToken: %v", err)
	}
	if cfg.Token != "" {
		t.Errorf("cfg.Token = %q after clear", cfg.Token)
	}
	token, source, err := cfg.ResolveToken("")
	if err != nil || token != "" || source != TokenFromNone {
		t.Fatalf("resolve after clear: token=%q source=%q err=%v", token, source, err)
	}
}

func TestLoadOrCreate(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "config.json")

	cfg, err := LoadOrCreate(path)
	if err != nil {
		t.Fatalf("LoadOrCreate (missing): %v", err)
	}
	if cfg.APIURL != DefaultAPIURL || cfg.TimeoutSeconds != DefaultTimeoutSecs {
		t.Errorf("defaults not applied: %+v", cfg)
	}

	if err := SaveConfig(path, &Config{APIURL: "http://custom", TimeoutSeconds: 5}); err != nil {
		t.Fatalf("SaveConfig: %v", err)
	}
	cfg, err = LoadOrCreate(path)
	if err != nil {
		t.Fatalf("LoadOrCreate (exists): %v", err)
	}
	if cfg.APIURL != "http://custom" || cfg.TimeoutSeconds != 5 {
		t.Errorf("existing config not loaded: %+v", cfg)
	}
}
