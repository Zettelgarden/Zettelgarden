// Package settings provides admin-managed, file-backed application settings.
//
// The design (Zettelgarden-6er.15) is the Gitea/Matomo model: non-secret
// admin settings live in ONE YAML file (default ./data/config.yaml, next to
// the SQLite DB) which is the single source of truth. Env vars only SEED the
// file on first boot (or when a new registry key appears); afterwards the
// file wins and env is ignored for these keys. Secrets and boot-time infra
// (SECRET_KEY, MAIL_PASSWORD, STRIPE_*, TYPESENSE_PASSWORD, ZETTEL_PORT,
// ZETTEL_URL, SQLITE_PATH, STORAGE_DIR, ...) stay env-only and never enter
// the file.
//
// Hot reload: the manager caches the parsed file and checks the file mtime on
// every read; hand-edits are picked up without a restart. Writes from the app
// (admin UI) update the cache directly and persist atomically (temp file +
// rename), so they apply immediately.
package settings

import (
	"fmt"
	"log"
	"os"
	"path/filepath"
	"strconv"
	"strings"
	"sync"
	"time"

	"gopkg.in/yaml.v3"
)

// Kind is the value type of a setting key.
type Kind int

const (
	// KindString is a free-form string value.
	KindString Kind = iota
	// KindBool is a boolean value stored as "true"/"false".
	KindBool
)

// Key describes one registry entry: how it is seeded from env and typed.
// The registry is an allowlist — only keys listed here are read, written, or
// exposed. It is deliberately non-secret: nothing sensitive belongs here.
type Key struct {
	Name    string // YAML key in the settings file
	Env     string // env var that seeds the value on first boot
	Kind    Kind
	Default string // fallback seed when Env is unset
}

// Registry is the ordered allowlist of admin-managed settings. Order defines
// the file layout. Keys coordinate with sibling beads:
//   - admin_email: ZETTEL_ADMIN_EMAIL (notification recipient; 6er.15)
//   - signups_enabled / mail_enabled / email_auto_validate: 6er.10 / 6er.6
//   - support_email: 6er.7
var Registry = []Key{
	{Name: "admin_email", Env: "ZETTEL_ADMIN_EMAIL", Kind: KindString},
	{Name: "site_name", Env: "ZETTEL_SITE_NAME", Kind: KindString, Default: "Zettelgarden"},
	{Name: "signups_enabled", Env: "SIGNUPS_ENABLED", Kind: KindBool, Default: "true"},
	{Name: "mail_enabled", Env: "MAIL_ENABLED", Kind: KindBool},
	{Name: "email_auto_validate", Env: "EMAIL_AUTO_VALIDATE", Kind: KindBool, Default: "true"},
	{Name: "support_email", Env: "ZETTEL_SUPPORT_EMAIL", Kind: KindString},
}

// registryByName indexes Registry for O(1) lookups.
var registryByName = func() map[string]Key {
	m := make(map[string]Key, len(Registry))
	for _, k := range Registry {
		m[k.Name] = k
	}
	return m
}()

// DefaultPath returns the default settings file path, derived from the SQLite
// path so the file sits next to the database (./data/config.yaml for
// ./data/zettelgarden.db). ZETTEL_CONFIG_FILE overrides it.
func DefaultPath(sqlitePath string) string {
	if v := os.Getenv("ZETTEL_CONFIG_FILE"); v != "" {
		return v
	}
	return filepath.Join(filepath.Dir(sqlitePath), "config.yaml")
}

// Manager reads, validates, caches, and persists the settings file.
type Manager struct {
	path string
	mu   sync.RWMutex

	values map[string]string // key -> canonical string value
	mtime  time.Time         // mtime of the file as last loaded
}

// New loads (or creates) the settings file at path and returns a Manager.
// If the file is missing, it is written seeded from env defaults. If it
// exists but lacks a registry key (e.g. after an upgrade), missing keys are
// appended. A file that fails to parse or validate is a hard error so a
// hand-broken config fails loudly at boot instead of being overwritten.
func New(path string) (*Manager, error) {
	m := &Manager{path: path}

	if err := m.ensureSeeded(); err != nil {
		return nil, err
	}
	if err := m.reload(); err != nil {
		return nil, err
	}
	return m, nil
}

// ensureSeeded creates the file when missing and appends registry keys that
// are absent (so upgrades add new settings to existing installs without
// clobbering values). It never overwrites existing values.
func (m *Manager) ensureSeeded() error {
	m.mu.Lock()
	defer m.mu.Unlock()

	if _, err := os.Stat(m.path); err == nil {
		// File exists: append any registry keys the file is missing.
		values, err := readFile(m.path)
		if err != nil {
			return err
		}
		var added bool
		for _, k := range Registry {
			if _, ok := values[k.Name]; !ok {
				values[k.Name] = seedValue(k)
				added = true
			}
		}
		if !added {
			return nil
		}
		m.values = values
		return m.writeLocked()
	}

	// File missing: seed every key from env and write the file.
	m.values = make(map[string]string, len(Registry))
	for _, k := range Registry {
		m.values[k.Name] = seedValue(k)
	}
	return m.writeLocked()
}

// seedValue returns the env-derived seed for a key, falling back to the
// registry default. mail_enabled has a special auto-detect: enabled when a
// MAIL_HOST is configured (matches MAIL_ENABLED semantics from 6er.6).
func seedValue(k Key) string {
	if v := os.Getenv(k.Env); v != "" {
		return v
	}
	if k.Name == "mail_enabled" {
		if os.Getenv("MAIL_HOST") != "" {
			return "true"
		}
		return "false"
	}
	return k.Default
}

// reload parses the file into m.values and records its mtime. Callers must
// hold m.mu (write lock). A parse/validation failure is returned so boot can
// fail loudly; at runtime (stale check) failures keep serving cached values.
func (m *Manager) reload() error {
	values, err := readFile(m.path)
	if err != nil {
		return err
	}
	if err := validateValues(values); err != nil {
		return err
	}
	m.values = values

	fi, err := os.Stat(m.path)
	if err != nil {
		return fmt.Errorf("stat settings file %s: %w", m.path, err)
	}
	m.mtime = fi.ModTime()
	return nil
}

// readFile parses the YAML file into a map of canonical string values.
func readFile(path string) (map[string]string, error) {
	data, err := os.ReadFile(path)
	if err != nil {
		return nil, fmt.Errorf("read settings file %s: %w", path, err)
	}

	var raw map[string]any
	if err := yaml.Unmarshal(data, &raw); err != nil {
		return nil, fmt.Errorf("parse settings file %s: %w", path, err)
	}

	values := make(map[string]string, len(raw))
	for name, v := range raw {
		k, ok := registryByName[name]
		if !ok {
			// Unknown keys are ignored (forward compatibility: an older
			// binary reading a file written by a newer one).
			continue
		}
		// Type-check the RAW value before coercion so a hand-edit like
		// `signups_enabled: maybe` fails loudly instead of silently
		// becoming "false".
		if err := checkRaw(k, v); err != nil {
			return nil, err
		}
		values[name] = coerce(k, v)
	}
	return values, nil
}

// checkRaw verifies a parsed YAML value is acceptable for the key's kind.
func checkRaw(k Key, v any) error {
	switch k.Kind {
	case KindBool:
		switch t := v.(type) {
		case bool, int, float64:
			return nil
		case string:
			if _, err := strconv.ParseBool(strings.TrimSpace(t)); err != nil {
				return fmt.Errorf("settings: %s must be true or false, got %q", k.Name, t)
			}
			return nil
		default:
			return fmt.Errorf("settings: %s must be true or false, got %v", k.Name, v)
		}
	}
	return nil
}

// coerce converts a parsed YAML value to the key's canonical string form.
func coerce(k Key, v any) string {
	switch k.Kind {
	case KindBool:
		return strconv.FormatBool(toBool(v))
	default:
		switch t := v.(type) {
		case string:
			return t
		case bool:
			return strconv.FormatBool(t)
		case int:
			return strconv.Itoa(t)
		case float64:
			return strconv.FormatFloat(t, 'f', -1, 64)
		default:
			return fmt.Sprintf("%v", t)
		}
	}
}

// toBool parses a YAML value as a boolean (handles the scalar types yaml.v3
// may produce). Unparseable values fall back to false; validateValues reports
// them loudly before this is ever relied on.
func toBool(v any) bool {
	switch t := v.(type) {
	case bool:
		return t
	case string:
		b, err := strconv.ParseBool(strings.TrimSpace(t))
		if err != nil {
			return false
		}
		return b
	case int:
		return t != 0
	case float64:
		return t != 0
	default:
		return false
	}
}

// validateValues type-checks every registry key present in values.
func validateValues(values map[string]string) error {
	for _, k := range Registry {
		v, ok := values[k.Name]
		if !ok {
			continue // missing keys are handled by ensureSeeded
		}
		switch k.Kind {
		case KindBool:
			if _, err := strconv.ParseBool(strings.TrimSpace(v)); err != nil {
				return fmt.Errorf("settings: %s must be true or false, got %q", k.Name, v)
			}
		}
	}
	if email := values["admin_email"]; email != "" && !strings.Contains(email, "@") {
		log.Printf("WARNING: settings admin_email %q is not a valid email address; notifications will fail to deliver", email)
	}
	return nil
}

// refreshIfStale re-reads the file when its mtime changed. Runtime failures
// (hand-edit broke the YAML) keep serving cached values and log once.
func (m *Manager) refreshIfStale() {
	m.mu.RLock()
	mtime := m.mtime
	m.mu.RUnlock()

	fi, err := os.Stat(m.path)
	if err != nil {
		return // file vanished mid-run; keep serving cache
	}
	if !fi.ModTime().After(mtime) {
		return
	}

	m.mu.Lock()
	defer m.mu.Unlock()
	// Re-check under the lock: another reader may have already reloaded.
	if fi2, err := os.Stat(m.path); err == nil && !fi2.ModTime().After(m.mtime) {
		return
	}
	if err := m.reload(); err != nil {
		log.Printf("WARNING: settings file %s changed but failed to reload (%v); keeping previous values", m.path, err)
	}
}

// Get returns the current value for a registry key ("" when unset).
func (m *Manager) Get(key string) string {
	if _, ok := registryByName[key]; !ok {
		return ""
	}
	m.refreshIfStale()
	m.mu.RLock()
	defer m.mu.RUnlock()
	return m.values[key]
}

// GetBool returns the current value for a bool registry key. Missing or
// unparseable values resolve to false (validateValues prevents the latter
// from persisting).
func (m *Manager) GetBool(key string) bool {
	v := m.Get(key)
	b, err := strconv.ParseBool(strings.TrimSpace(v))
	if err != nil {
		return false
	}
	return b
}

// All returns a copy of all registry values (non-secret by construction).
func (m *Manager) All() map[string]string {
	m.refreshIfStale()
	m.mu.RLock()
	defer m.mu.RUnlock()
	out := make(map[string]string, len(m.values))
	for k, v := range m.values {
		out[k] = v
	}
	return out
}

// Set validates and persists a value for a registry key, then updates the
// cache so the change applies immediately. It refuses to write when the
// current file is broken (never clobber a hand-edit that failed to parse).
func (m *Manager) Set(key, value string) error {
	k, ok := registryByName[key]
	if !ok {
		return fmt.Errorf("settings: unknown key %q", key)
	}

	switch k.Kind {
	case KindBool:
		if _, err := strconv.ParseBool(strings.TrimSpace(value)); err != nil {
			return fmt.Errorf("settings: %s must be true or false, got %q", key, value)
		}
	case KindString:
		if key == "admin_email" && value != "" && !strings.Contains(value, "@") {
			return fmt.Errorf("settings: admin_email %q is not a valid email address", value)
		}
	}

	m.mu.Lock()
	defer m.mu.Unlock()

	// Refuse to write over a file we can't parse.
	if _, err := readFile(m.path); err != nil {
		return fmt.Errorf("settings: refusing to overwrite unreadable file %s: %w", m.path, err)
	}

	// Update the in-memory cache and merge into existing file values so other
	// keys survive the rewrite.
	values, err := readFile(m.path)
	if err != nil {
		return err
	}
	values[key] = value
	m.values = values

	return m.writeLocked()
}

// writeLocked persists m.values to the file atomically (temp file + rename).
// Callers must hold m.mu (write lock).
func (m *Manager) writeLocked() error {
	dir := filepath.Dir(m.path)
	if err := os.MkdirAll(dir, 0o750); err != nil {
		return fmt.Errorf("settings: create dir %s: %w", dir, err)
	}

	tmp, err := os.CreateTemp(dir, ".config-*.yaml")
	if err != nil {
		return fmt.Errorf("settings: create temp file: %w", err)
	}
	tmpName := tmp.Name()
	defer os.Remove(tmpName) // no-op after successful rename

	if _, err := tmp.WriteString(m.render()); err != nil {
		tmp.Close()
		return fmt.Errorf("settings: write temp file: %w", err)
	}
	if err := tmp.Chmod(0o644); err != nil {
		tmp.Close()
		return fmt.Errorf("settings: chmod temp file: %w", err)
	}
	if err := tmp.Close(); err != nil {
		return fmt.Errorf("settings: close temp file: %w", err)
	}
	if err := os.Rename(tmpName, m.path); err != nil {
		return fmt.Errorf("settings: rename temp file: %w", err)
	}

	// Keep the cache mtime in sync so refreshIfStale doesn't immediately
	// reload what we just wrote.
	fi, err := os.Stat(m.path)
	if err == nil {
		m.mtime = fi.ModTime()
	}
	return nil
}

// render builds the YAML file content with a header comment and stable
// registry ordering.
func (m *Manager) render() string {
	var b strings.Builder
	b.WriteString("# Zettelgarden admin settings.\n")
	b.WriteString("# Managed via the admin UI; edit by hand at your own risk.\n")
	b.WriteString("# Env vars seed these on first boot; this file is the source of truth afterwards.\n")
	for _, k := range Registry {
		b.WriteString(k.Name)
		b.WriteString(": ")
		b.WriteString(quoteYAML(m.values[k.Name]))
		b.WriteString("\n")
	}
	return b.String()
}

// quoteYAML quotes string values so hand-edits and re-reads round-trip
// exactly (avoids YAML scalar surprises like `yes`/`no` or values containing
// `: `).
func quoteYAML(v string) string {
	return strconv.Quote(v)
}
