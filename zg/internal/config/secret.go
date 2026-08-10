package config

import (
	"encoding/json"
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"time"

	"github.com/99designs/keyring"
)

const (
	// EnvToken overrides the configured token (precedence: flag > env > keyring > config).
	EnvToken = "ZETTELGARDEN_TOKEN"

	// EnvNoKeyring skips the OS keyring entirely, forcing config-file storage
	// (e.g. headless CI where the secret service is unreachable).
	EnvNoKeyring = "ZETTELGARDEN_NO_KEYRING"

	// KeyringService is the OS keyring service name (e.g. "zettelgarden" in
	// gnome-keyring / kwallet).
	KeyringService = "zettelgarden"
	// KeyringItem is the key under which the API token is stored.
	KeyringItem = "api-token"
)

// ErrNoToken is returned when no token could be resolved.
var ErrNoToken = errors.New("no token configured")

// TokenSource identifies where the effective token came from.
type TokenSource string

const (
	TokenFromFlag    TokenSource = "flag"
	TokenFromEnv     TokenSource = "env"
	TokenFromKeyring TokenSource = "keyring"
	TokenFromConfig  TokenSource = "config"
	TokenFromNone    TokenSource = "none"
)

// keyringOperator abstracts the OS keyring so tests can substitute a fake.
type keyringOperator interface {
	Set(token string) error
	Get() (string, error)
	Delete() error
}

// keyringOp is the active keyring implementation. Tests replace it.
var keyringOp keyringOperator = osKeyringOperator{}

// keyringTimeout bounds keyring operations. Some environments (headless
// servers, containers) have a DBUS_SESSION_BUS_ADDRESS pointing at a stale or
// dead bus, which makes keyring.Open block indefinitely. zg is a short-lived
// process, so a timed-out operation just falls back to the config file.
var keyringTimeout = 3 * time.Second

// withKeyringTimeout runs fn, giving up after keyringTimeout and returning an
// error so callers fall back to the config file.
func withKeyringTimeout(fn func() error) error {
	done := make(chan error, 1)
	go func() { done <- fn() }()
	select {
	case err := <-done:
		return err
	case <-time.After(keyringTimeout):
		return fmt.Errorf("keyring unavailable (timed out after %s)", keyringTimeout)
	}
}

// osKeyringOperator stores the API token in the OS keyring (libsecret /
// gnome-keyring on Linux, Keychain on macOS, Credential Manager on Windows).
// Every operation returns an error when no OS keyring is available (e.g. a
// headless server without a reachable secret service daemon).
type osKeyringOperator struct{}

func (osKeyringOperator) open() (keyring.Keyring, error) {
	return keyring.Open(keyring.Config{
		ServiceName: KeyringService,
		// OS backends only; headless environments fall back to the config file.
		AllowedBackends: []keyring.BackendType{
			keyring.SecretServiceBackend,
			keyring.KeychainBackend,
			keyring.WinCredBackend,
		},
	})
}

func (o osKeyringOperator) Set(token string) error {
	ring, err := o.open()
	if err != nil {
		return err
	}
	return ring.Set(keyring.Item{Key: KeyringItem, Data: []byte(token)})
}

func (o osKeyringOperator) Get() (string, error) {
	ring, err := o.open()
	if err != nil {
		return "", err
	}
	item, err := ring.Get(KeyringItem)
	if err != nil {
		return "", err
	}
	return string(item.Data), nil
}

func (o osKeyringOperator) Delete() error {
	ring, err := o.open()
	if err != nil {
		return err
	}
	return ring.Remove(KeyringItem)
}

// ResolveToken returns the effective token and its source using precedence:
// CLI flag > ZETTELGARDEN_TOKEN env var > OS keyring > config file token.
func (c *Config) ResolveToken(flagToken string) (string, TokenSource, error) {
	if flagToken != "" {
		return flagToken, TokenFromFlag, nil
	}
	if envToken := os.Getenv(EnvToken); envToken != "" {
		return envToken, TokenFromEnv, nil
	}
	if token, err := loadKeyringToken(); err == nil {
		return token, TokenFromKeyring, nil
	}
	if c.Token != "" {
		return c.Token, TokenFromConfig, nil
	}
	return "", TokenFromNone, nil
}

// keyringEnabled reports whether OS keyring access is allowed. Set
// ZETTELGARDEN_NO_KEYRING to force config-file token storage.
func keyringEnabled() bool {
	return os.Getenv(EnvNoKeyring) == ""
}

func loadKeyringToken() (string, error) {
	if !keyringEnabled() {
		return "", errors.New("keyring disabled via " + EnvNoKeyring)
	}
	var token string
	err := withKeyringTimeout(func() error {
		var err error
		token, err = keyringOp.Get()
		return err
	})
	if err != nil {
		return "", err
	}
	if token == "" {
		return "", ErrNoToken
	}
	return token, nil
}

// StoreToken persists a token in the OS keyring when available, falling back
// to the config file (0600 perms) otherwise. It returns where the token was
// stored ("keyring" or "config"). When the keyring is used the plaintext token
// is kept out of the config file.
func StoreToken(configPath string, cfg *Config, token string) (string, error) {
	keyringErr := withKeyringTimeout(func() error {
		if !keyringEnabled() {
			return errors.New("keyring disabled via " + EnvNoKeyring)
		}
		return keyringOp.Set(token)
	})
	if keyringErr == nil {
		cfg.Token = "" // secret lives in the keyring, not in config.json
		if err := SaveConfig(configPath, cfg); err != nil {
			return "", fmt.Errorf("save config: %w", err)
		}
		return "keyring", nil
	}
	cfg.Token = token
	if err := SaveConfig(configPath, cfg); err != nil {
		return "", err
	}
	return "config", nil
}

// ClearToken removes the stored token from the keyring (best effort) and the
// config file.
func ClearToken(configPath string, cfg *Config) error {
	_ = withKeyringTimeout(func() error {
		if !keyringEnabled() {
			return nil
		}
		return keyringOp.Delete() // best effort; ignore "not found" / unavailable keyring
	})
	cfg.Token = ""
	return SaveConfig(configPath, cfg)
}

// SaveConfig writes cfg to path with 0600 permissions, creating parent
// directories (0700) as needed. Existing files are tightened to 0600 as well
// so a previously world-readable config gets fixed on the next save.
func SaveConfig(path string, cfg *Config) error {
	if err := os.MkdirAll(filepath.Dir(path), 0700); err != nil {
		return err
	}
	data, err := json.MarshalIndent(cfg, "", "  ")
	if err != nil {
		return err
	}
	if err := os.WriteFile(path, data, 0600); err != nil {
		return err
	}
	return os.Chmod(path, 0600)
}

// LoadOrCreate loads the config file, returning a default config when the file
// does not exist yet (used by `zg auth` before any config has been written).
func LoadOrCreate(path string) (*Config, error) {
	cfg, err := LoadConfig(path)
	if err == nil {
		return cfg, nil
	}
	if errors.Is(err, os.ErrNotExist) {
		return &Config{APIURL: DefaultAPIURL, TimeoutSeconds: DefaultTimeoutSecs}, nil
	}
	return nil, err
}

// IsJWT reports whether token looks like a short-lived JWT (dot-separated
// base64url segments starting with the "eyJ" header prefix). API keys are
// prefixed "zg_live_" and never match.
func IsJWT(token string) bool {
	return strings.HasPrefix(token, "eyJ") && strings.Contains(token, ".")
}

// IsAPIKey reports whether token looks like a Zettelgarden API key.
func IsAPIKey(token string) bool {
	return strings.HasPrefix(token, "zg_live_")
}

// JWTMigrationNotice returns the warning shown when a short-lived JWT is
// configured in place of a durable API key.
func JWTMigrationNotice() string {
	return "configured token looks like a short-lived JWT: session tokens expire after 15 days, so CLI auth will silently break. Create a durable API key in the web UI (Settings → API Keys) or run 'zg auth login'."
}
