package settings

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"
)

func testEnv(t *testing.T) {
	t.Helper()
	t.Setenv("ZETTEL_ADMIN_EMAIL", "admin@test.com")
	t.Setenv("SMTP_HOST", "smtp.example.com")
	t.Setenv("ZETTEL_SITE_NAME", "")
}

// bumpMtime forces the file mtime forward so the mtime-checked hot reload
// reliably notices an external change.
func bumpMtime(t *testing.T, path string) {
	t.Helper()
	future := time.Now().Add(2 * time.Second)
	if err := os.Chtimes(path, future, future); err != nil {
		t.Fatalf("chtimes: %v", err)
	}
}

func TestNewSeedsFileFromEnv(t *testing.T) {
	testEnv(t)
	dir := t.TempDir()
	path := filepath.Join(dir, "config.yaml")

	m, err := New(path)
	if err != nil {
		t.Fatalf("New: %v", err)
	}

	if got := m.Get("admin_email"); got != "admin@test.com" {
		t.Errorf("admin_email = %q, want admin@test.com", got)
	}
	if got := m.Get("site_name"); got != "Zettelgarden" {
		t.Errorf("site_name = %q, want default Zettelgarden", got)
	}
	if got := m.GetBool("signups_enabled"); !got {
		t.Error("signups_enabled should default to true")
	}
	// mail_enabled auto-detects from SMTP_HOST (set in testEnv).
	if got := m.GetBool("mail_enabled"); !got {
		t.Error("mail_enabled should auto-detect true when SMTP_HOST is set")
	}
	// Cleanup retention keys default to 30 days.
	if got := m.GetInt("job_retention_days", -1); got != 30 {
		t.Errorf("job_retention_days = %d, want default 30", got)
	}
	if got := m.GetInt("rss_article_retention_days", -1); got != 30 {
		t.Errorf("rss_article_retention_days = %d, want default 30", got)
	}

	// The file should exist on disk with the seeded values.
	data, err := os.ReadFile(path)
	if err != nil {
		t.Fatalf("read seeded file: %v", err)
	}
	if !strings.Contains(string(data), "admin_email: \"admin@test.com\"") {
		t.Errorf("seeded file missing admin_email:\n%s", data)
	}
}

func TestNewExistingFileNotOverwritten(t *testing.T) {
	testEnv(t)
	dir := t.TempDir()
	path := filepath.Join(dir, "config.yaml")

	m, err := New(path)
	if err != nil {
		t.Fatalf("New: %v", err)
	}
	// Change a value through the API, then re-open: the edited value must
	// survive (file is the source of truth; env is only a seed).
	if err := m.Set("admin_email", "edited@test.com"); err != nil {
		t.Fatalf("Set: %v", err)
	}

	m2, err := New(path)
	if err != nil {
		t.Fatalf("New (reopen): %v", err)
	}
	if got := m2.Get("admin_email"); got != "edited@test.com" {
		t.Errorf("admin_email after reopen = %q, want edited@test.com (env must not re-seed)", got)
	}
}

func TestHotReloadPicksUpExternalEdit(t *testing.T) {
	testEnv(t)
	dir := t.TempDir()
	path := filepath.Join(dir, "config.yaml")

	m, err := New(path)
	if err != nil {
		t.Fatalf("New: %v", err)
	}
	if got := m.Get("admin_email"); got != "admin@test.com" {
		t.Fatalf("seed admin_email = %q", got)
	}

	// Simulate a hand-edit: rewrite the file directly and bump mtime.
	data, err := os.ReadFile(path)
	if err != nil {
		t.Fatalf("read: %v", err)
	}
	edited := strings.Replace(string(data), "admin@test.com", "hand@edited.com", 1)
	if err := os.WriteFile(path, []byte(edited), 0o644); err != nil {
		t.Fatalf("write: %v", err)
	}
	bumpMtime(t, path)

	if got := m.Get("admin_email"); got != "hand@edited.com" {
		t.Errorf("admin_email after hand-edit = %q, want hand@edited.com (hot reload)", got)
	}
}

func TestSetPersistsAndAppliesImmediately(t *testing.T) {
	testEnv(t)
	dir := t.TempDir()
	path := filepath.Join(dir, "config.yaml")

	m, err := New(path)
	if err != nil {
		t.Fatalf("New: %v", err)
	}

	if err := m.Set("signups_enabled", "false"); err != nil {
		t.Fatalf("Set: %v", err)
	}
	if got := m.GetBool("signups_enabled"); got {
		t.Error("signups_enabled should be false after Set")
	}

	// Persisted: a fresh manager reads it back.
	m2, err := New(path)
	if err != nil {
		t.Fatalf("New (reopen): %v", err)
	}
	if got := m2.GetBool("signups_enabled"); got {
		t.Error("signups_enabled should persist across reopen")
	}

	// Set preserves other keys.
	if got := m2.Get("admin_email"); got != "admin@test.com" {
		t.Errorf("admin_email clobbered by Set: %q", got)
	}
}

func TestSetValidation(t *testing.T) {
	testEnv(t)
	dir := t.TempDir()
	path := filepath.Join(dir, "config.yaml")

	m, err := New(path)
	if err != nil {
		t.Fatalf("New: %v", err)
	}

	if err := m.Set("signups_enabled", "not-a-bool"); err == nil {
		t.Error("Set of invalid bool should fail")
	}
	if err := m.Set("admin_email", "not-an-email"); err == nil {
		t.Error("Set of invalid admin_email should fail")
	}
	if err := m.Set("nonexistent_key", "x"); err == nil {
		t.Error("Set of unknown key should fail")
	}
}

func TestBootFailsLoudlyOnBrokenYAML(t *testing.T) {
	testEnv(t)
	dir := t.TempDir()
	path := filepath.Join(dir, "config.yaml")

	if _, err := New(path); err != nil {
		t.Fatalf("New: %v", err)
	}

	// Hand-edit the file into invalid YAML.
	if err := os.WriteFile(path, []byte("admin_email: [unclosed"), 0o644); err != nil {
		t.Fatalf("write: %v", err)
	}
	if _, err := New(path); err == nil {
		t.Fatal("New on broken YAML should fail loudly at boot")
	}
}

func TestBootFailsLoudlyOnInvalidBool(t *testing.T) {
	testEnv(t)
	dir := t.TempDir()
	path := filepath.Join(dir, "config.yaml")

	if _, err := New(path); err != nil {
		t.Fatalf("New: %v", err)
	}
	if err := os.WriteFile(path, []byte("signups_enabled: maybe\nadmin_email: \"a@b.c\"\n"), 0o644); err != nil {
		t.Fatalf("write: %v", err)
	}
	if _, err := New(path); err == nil {
		t.Fatal("New with invalid bool value should fail loudly")
	}
}

func TestSetRefusesToClobberBrokenFile(t *testing.T) {
	testEnv(t)
	dir := t.TempDir()
	path := filepath.Join(dir, "config.yaml")

	m, err := New(path)
	if err != nil {
		t.Fatalf("New: %v", err)
	}
	if err := os.WriteFile(path, []byte("admin_email: [unclosed"), 0o644); err != nil {
		t.Fatalf("write: %v", err)
	}
	if err := m.Set("signups_enabled", "false"); err == nil {
		t.Fatal("Set over a broken file should refuse to write")
	}
}

func TestRetentionKeysSeedFromEnv(t *testing.T) {
	t.Setenv("ZETTEL_ADMIN_EMAIL", "admin@test.com")
	t.Setenv("SMTP_HOST", "smtp.example.com")
	t.Setenv("JOB_RETENTION_DAYS", "14")
	t.Setenv("RSS_ARTICLE_RETENTION_DAYS", "7")
	dir := t.TempDir()

	m, err := New(filepath.Join(dir, "config.yaml"))
	if err != nil {
		t.Fatalf("New: %v", err)
	}
	if got := m.GetInt("job_retention_days", -1); got != 14 {
		t.Errorf("job_retention_days = %d, want env-seeded 14", got)
	}
	if got := m.GetInt("rss_article_retention_days", -1); got != 7 {
		t.Errorf("rss_article_retention_days = %d, want env-seeded 7", got)
	}
}

func TestSetIntValidationAndPersistence(t *testing.T) {
	testEnv(t)
	dir := t.TempDir()
	path := filepath.Join(dir, "config.yaml")

	m, err := New(path)
	if err != nil {
		t.Fatalf("New: %v", err)
	}

	if err := m.Set("job_retention_days", "not-a-number"); err == nil {
		t.Error("Set of invalid int should fail")
	}
	if err := m.Set("job_retention_days", "90"); err != nil {
		t.Fatalf("Set job_retention_days: %v", err)
	}
	if got := m.GetInt("job_retention_days", -1); got != 90 {
		t.Errorf("job_retention_days after Set = %d, want 90", got)
	}

	// Persisted across reopen; GetInt without the key falls back to def.
	m2, err := New(path)
	if err != nil {
		t.Fatalf("New (reopen): %v", err)
	}
	if got := m2.GetInt("job_retention_days", -1); got != 90 {
		t.Errorf("job_retention_days persisted = %d, want 90", got)
	}
	if got := m2.GetInt("nonexistent_key", 42); got != 42 {
		t.Errorf("GetInt unknown key = %d, want fallback 42", got)
	}
}

func TestBootFailsLoudlyOnInvalidInt(t *testing.T) {
	testEnv(t)
	dir := t.TempDir()
	path := filepath.Join(dir, "config.yaml")

	if _, err := New(path); err != nil {
		t.Fatalf("New: %v", err)
	}
	if err := os.WriteFile(path, []byte("job_retention_days: \"thirty\"\nadmin_email: \"a@b.c\"\n"), 0o644); err != nil {
		t.Fatalf("write: %v", err)
	}
	if _, err := New(path); err == nil {
		t.Fatal("New with invalid int value should fail loudly")
	}

	// A fractional number must also fail loudly rather than silently truncate.
	if err := os.WriteFile(path, []byte("job_retention_days: 30.5\nadmin_email: \"a@b.c\"\n"), 0o644); err != nil {
		t.Fatalf("write: %v", err)
	}
	if _, err := New(path); err == nil {
		t.Fatal("New with fractional int value should fail loudly")
	}
}

func TestUpgradeAppendsMissingKeys(t *testing.T) {
	testEnv(t)
	dir := t.TempDir()
	path := filepath.Join(dir, "config.yaml")

	// Simulate an old install whose file predates site_name/signups_enabled.
	if err := os.WriteFile(path, []byte("admin_email: \"old@test.com\"\n"), 0o644); err != nil {
		t.Fatalf("write: %v", err)
	}

	m, err := New(path)
	if err != nil {
		t.Fatalf("New: %v", err)
	}
	if got := m.Get("admin_email"); got != "old@test.com" {
		t.Errorf("admin_email = %q, want preserved old@test.com", got)
	}
	if got := m.Get("site_name"); got != "Zettelgarden" {
		t.Errorf("site_name = %q, want appended default", got)
	}
	if got := m.GetBool("signups_enabled"); !got {
		t.Error("signups_enabled should be appended as true")
	}
}

func TestUnknownKeysIgnored(t *testing.T) {
	testEnv(t)
	dir := t.TempDir()
	path := filepath.Join(dir, "config.yaml")

	if err := os.WriteFile(path, []byte("admin_email: \"a@b.c\"\nsome_future_key: 42\n"), 0o644); err != nil {
		t.Fatalf("write: %v", err)
	}
	m, err := New(path)
	if err != nil {
		t.Fatalf("New: %v", err)
	}
	if got := m.Get("some_future_key"); got != "" {
		t.Errorf("unknown key should not be readable, got %q", got)
	}
}

func TestDefaultPath(t *testing.T) {
	t.Setenv("ZETTEL_CONFIG_FILE", "")
	got := DefaultPath("./data/zettelgarden.db")
	if got != filepath.Join("data", "config.yaml") {
		t.Errorf("DefaultPath = %q, want data/config.yaml", got)
	}

	t.Setenv("ZETTEL_CONFIG_FILE", "/etc/zg/settings.yaml")
	if got := DefaultPath("./data/zettelgarden.db"); got != "/etc/zg/settings.yaml" {
		t.Errorf("DefaultPath with override = %q", got)
	}
}

func TestAllReturnsCopy(t *testing.T) {
	testEnv(t)
	dir := t.TempDir()
	path := filepath.Join(dir, "config.yaml")

	m, err := New(path)
	if err != nil {
		t.Fatalf("New: %v", err)
	}
	all := m.All()
	if len(all) != len(Registry) {
		t.Errorf("All() has %d keys, want %d", len(all), len(Registry))
	}
	if all["admin_email"] != "admin@test.com" {
		t.Errorf("All() admin_email = %q", all["admin_email"])
	}
	// Mutating the returned map must not affect the manager.
	all["admin_email"] = "mutated@test.com"
	if got := m.Get("admin_email"); got != "admin@test.com" {
		t.Errorf("Get after mutating All() result = %q", got)
	}
}
