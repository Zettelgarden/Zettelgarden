package config

import (
	"os"
	"strings"
	"testing"
)

// stripeEnvKeys lists the STRIPE_* variables that become optional when
// STRIPE_ENABLED=false (see loadStripeConfig).
var stripeEnvKeys = []string{
	"STRIPE_SECRET_KEY",
	"STRIPE_PUBLISHABLE_KEY",
	"STRIPE_WEBHOOK_SECRET",
	"STRIPE_MONTH_PRICE",
	"STRIPE_YEAR_PRICE",
	"STRIPE_BILLING_URL",
}

// githubClientEnvKeys lists the three client variables that become optional
// when GITHUB_AUTH_ENABLED=false (see loadGitHubConfig).
var githubClientEnvKeys = []string{
	"GITHUB_CLIENT_ID",
	"GITHUB_CLIENT_SECRET",
	"GITHUB_REDIRECT_URI",
}

// smtpEnvKeys lists the SMTP_* variables that become optional when mail is
// disabled (SMTP_HOST unset with MAIL_ENABLED unset, or MAIL_ENABLED=false).
var smtpEnvKeys = []string{
	"SMTP_HOST",
	"SMTP_PORT",
	"SMTP_USERNAME",
	"SMTP_PASSWORD",
	"SMTP_FROM",
	"SMTP_STARTTLS",
}

// setFullValidEnv populates every variable LoadConfig reads with valid values,
// optionally forcing dev mode. t.Setenv registers an automatic restore of the
// previous value (or unset state), so tests that os.Unsetenv individual
// variables afterwards still get a clean environment back.
func setFullValidEnv(t *testing.T, devMode bool) {
	t.Helper()
	if devMode {
		t.Setenv("ZETTEL_DEV", "true")
	} else {
		t.Setenv("ZETTEL_DEV", "false")
	}
	t.Setenv("ZETTEL_PORT", "8080")
	t.Setenv("ZETTEL_URL", "http://localhost:8080")
	t.Setenv("ZETTEL_ADMIN_EMAIL", "admin@test.com")
	t.Setenv("SECRET_KEY", "test-secret-key-for-jwt-signing-32-chars-minimum")
	t.Setenv("ZETTEL_LLM_KEY", "test-zai-api-key")
	t.Setenv("ZETTEL_LLM_ENDPOINT", "https://api.z.ai/api/coding/paas/v4")
	t.Setenv("ZETTEL_LLM_DEFAULT_MODEL", "glm-5.1")
	t.Setenv("ZETTEL_LLM_SUMMARIZE_MODEL", "glm-5.1")
	t.Setenv("SMTP_HOST", "smtp.gmail.com")
	t.Setenv("SMTP_PORT", "587")
	t.Setenv("SMTP_USERNAME", "test-smtp-user")
	t.Setenv("SMTP_PASSWORD", "test-mail-password")
	t.Setenv("SMTP_FROM", "noreply@test.com")
	for _, k := range stripeEnvKeys {
		t.Setenv(k, "test-value")
	}
	t.Setenv("STRIPE_BILLING_URL", "https://billing.stripe.com/test")
	t.Setenv("STORAGE_DIR", t.TempDir())
	t.Setenv("GITHUB_AUTH_ENABLED", "true")
	t.Setenv("GITHUB_CLIENT_ID", "test-client-id")
	t.Setenv("GITHUB_CLIENT_SECRET", "test-client-secret")
	t.Setenv("GITHUB_REDIRECT_URI", "http://localhost:8080/auth/github/callback")
	t.Setenv("TYPESENSE_HOST", "http://localhost:8108")
	t.Setenv("TYPESENSE_PASSWORD", "test-typesense-password")
	t.Setenv("TYPESENSE_COLLECTION", "zettelgarden_test")
}

// prodRequiredEnvKeys lists every variable LoadConfig requires in production
// mode (dev mode tolerates all of them missing). STRIPE_*, the GitHub/OIDC
// client vars, and the SMTP_* vars are excluded because their opt-outs
// (STRIPE_ENABLED=false, GITHUB_AUTH_ENABLED=false, and mail-off
// auto-detect/MAIL_ENABLED=false) are covered separately.
var prodRequiredEnvKeys = []string{
	"ZETTEL_URL",
	"ZETTEL_ADMIN_EMAIL",
	"SECRET_KEY",
	"ZETTEL_LLM_KEY",
	"ZETTEL_LLM_ENDPOINT",
	"ZETTEL_LLM_DEFAULT_MODEL",
	"ZETTEL_LLM_SUMMARIZE_MODEL",
	"TYPESENSE_HOST",
	"TYPESENSE_PASSWORD",
	"TYPESENSE_COLLECTION",
}

// assertPanics runs fn and fails the test if it does not panic, returning the
// recovered value so callers can inspect the panic message.
func assertPanics(t *testing.T, fn func()) (recovered interface{}) {
	t.Helper()
	defer func() {
		recovered = recover()
	}()
	fn()
	t.Fatal("expected panic, but the function returned normally")
	return nil
}

// TestLoadConfigProdModePanicsOnMissingRequiredVar verifies that in production
// mode a missing required variable fails fast with a readable panic instead of
// silently booting with an empty string.
func TestLoadConfigProdModePanicsOnMissingRequiredVar(t *testing.T) {
	setFullValidEnv(t, false)
	os.Unsetenv("SECRET_KEY")

	recovered := assertPanics(t, func() { LoadConfig() })
	msg, ok := recovered.(string)
	if !ok {
		t.Fatalf("expected panic value to be a string, got %T", recovered)
	}
	if !strings.Contains(msg, "Configuration validation failed") {
		t.Errorf("expected panic message to mention 'Configuration validation failed', got: %q", msg)
	}
	if !strings.Contains(msg, "SECRET_KEY") {
		t.Errorf("expected panic message to mention SECRET_KEY, got: %q", msg)
	}
}

// TestLoadConfigProdModeStripeDisabledAllowsMissingStripeKeys verifies the
// STRIPE_ENABLED=false opt-out runs before the final enforcement: with billing
// disabled, missing STRIPE_* values must not panic LoadConfig.
func TestLoadConfigProdModeStripeDisabledAllowsMissingStripeKeys(t *testing.T) {
	setFullValidEnv(t, false)
	t.Setenv("STRIPE_ENABLED", "false")
	for _, k := range stripeEnvKeys {
		os.Unsetenv(k)
	}

	cfg := LoadConfig() // must not panic with every STRIPE_* value missing
	if cfg.Services.Stripe.Enabled {
		t.Fatal("expected Stripe.Enabled=false when STRIPE_ENABLED=false")
	}
}

// TestLoadConfigProdModeStripeDisabledStillRequiresCoreVars verifies the
// STRIPE_ENABLED=false opt-out only relaxes STRIPE_*: a missing truly-required
// variable (SECRET_KEY) must still panic.
func TestLoadConfigProdModeStripeDisabledStillRequiresCoreVars(t *testing.T) {
	setFullValidEnv(t, false)
	t.Setenv("STRIPE_ENABLED", "false")
	for _, k := range stripeEnvKeys {
		os.Unsetenv(k)
	}
	os.Unsetenv("SECRET_KEY")

	recovered := assertPanics(t, func() { LoadConfig() })
	msg, ok := recovered.(string)
	if !ok {
		t.Fatalf("expected panic value to be a string, got %T", recovered)
	}
	if !strings.Contains(msg, "SECRET_KEY") {
		t.Errorf("expected panic message to mention SECRET_KEY, got: %q", msg)
	}
}

// TestLoadConfigProdModeGitHubDisabledAllowsMissingClientVars verifies the
// GITHUB_AUTH_ENABLED=false opt-out runs before the final enforcement: with
// GitHub auth disabled, the three client values are not required.
func TestLoadConfigProdModeGitHubDisabledAllowsMissingClientVars(t *testing.T) {
	setFullValidEnv(t, false)
	t.Setenv("GITHUB_AUTH_ENABLED", "false")
	for _, k := range githubClientEnvKeys {
		os.Unsetenv(k)
	}

	cfg := LoadConfig() // must not panic with the GitHub client values missing
	if cfg.Services.GitHub.Enabled {
		t.Fatal("expected GitHub.Enabled=false when GITHUB_AUTH_ENABLED=false")
	}
}

// TestLoadConfigProdModeMailDisabledAllowsMissingMailKeys verifies the
// MAIL_ENABLED=false opt-out: with mail explicitly off, the SMTP_* values are
// not required (6er.6).
func TestLoadConfigProdModeMailDisabledAllowsMissingMailKeys(t *testing.T) {
	setFullValidEnv(t, false)
	t.Setenv("MAIL_ENABLED", "false")
	for _, k := range smtpEnvKeys {
		os.Unsetenv(k)
	}

	cfg := LoadConfig() // must not panic with every SMTP_* value missing
	if cfg.Services.Mail.SMTPHost != "" {
		t.Errorf("expected Mail.SMTPHost empty when mail disabled, got %q", cfg.Services.Mail.SMTPHost)
	}
	if cfg.Services.Mail.SMTPFrom != "" {
		t.Errorf("expected Mail.SMTPFrom empty when mail disabled, got %q", cfg.Services.Mail.SMTPFrom)
	}
}

// TestLoadConfigProdModeMailOptionalWithoutConfig verifies the auto-detect
// path: with no SMTP_HOST and no MAIL_ENABLED at all, mail is treated as
// disabled and LoadConfig does not require the SMTP_* values — a self-hoster
// with no SMTP at all boots cleanly (6er.6).
func TestLoadConfigProdModeMailOptionalWithoutConfig(t *testing.T) {
	setFullValidEnv(t, false)
	for _, k := range smtpEnvKeys {
		os.Unsetenv(k)
	}
	os.Unsetenv("MAIL_ENABLED")

	cfg := LoadConfig() // must not panic
	if cfg.Services.Mail.SMTPHost != "" {
		t.Errorf("expected Mail.SMTPHost empty without mail config, got %q", cfg.Services.Mail.SMTPHost)
	}
	if !cfg.Services.Mail.StartTLS {
		t.Error("expected Mail.StartTLS to default to true")
	}
	if cfg.Services.Mail.SMTPPort != 587 {
		t.Errorf("expected Mail.SMTPPort to default to 587, got %d", cfg.Services.Mail.SMTPPort)
	}
}

// TestLoadConfigProdModeMailHostSetRequiresFrom verifies the auto-detect
// enable path: with SMTP_HOST set (and MAIL_ENABLED unset), mail is enabled
// and SMTP_FROM becomes required (SMTP_USERNAME/PASSWORD stay optional for
// local relays).
func TestLoadConfigProdModeMailHostSetRequiresFrom(t *testing.T) {
	setFullValidEnv(t, false)
	os.Unsetenv("SMTP_FROM")
	os.Unsetenv("SMTP_USERNAME")
	os.Unsetenv("SMTP_PASSWORD")

	recovered := assertPanics(t, func() { LoadConfig() })
	msg, ok := recovered.(string)
	if !ok {
		t.Fatalf("expected panic value to be a string, got %T", recovered)
	}
	if !strings.Contains(msg, "SMTP_FROM") {
		t.Errorf("expected panic message to mention SMTP_FROM, got: %q", msg)
	}
}

// TestLoadConfigInvalidSMTPPort verifies SMTP_PORT must parse as a positive
// integer when set; a bogus value is a validation error even when the rest of
// the mail config is valid.
func TestLoadConfigInvalidSMTPPort(t *testing.T) {
	setFullValidEnv(t, false)
	t.Setenv("SMTP_PORT", "not-a-port")

	recovered := assertPanics(t, func() { LoadConfig() })
	msg, ok := recovered.(string)
	if !ok {
		t.Fatalf("expected panic value to be a string, got %T", recovered)
	}
	if !strings.Contains(msg, "SMTP_PORT") {
		t.Errorf("expected panic message to mention SMTP_PORT, got: %q", msg)
	}
}

// TestLoadConfigSMTPFromRequiresAtSign verifies SMTP_FROM is validated as an
// email address when set.
func TestLoadConfigSMTPFromRequiresAtSign(t *testing.T) {
	setFullValidEnv(t, false)
	t.Setenv("SMTP_FROM", "not-an-email")

	recovered := assertPanics(t, func() { LoadConfig() })
	msg, ok := recovered.(string)
	if !ok {
		t.Fatalf("expected panic value to be a string, got %T", recovered)
	}
	if !strings.Contains(msg, "SMTP_FROM") {
		t.Errorf("expected panic message to mention SMTP_FROM, got: %q", msg)
	}
}

// TestLoadConfigSMTPPortOverridesDefault verifies SMTP_PORT is parsed into
// MailConfig when set, and SMTP_STARTTLS=false turns off STARTTLS.
func TestLoadConfigSMTPPortOverridesDefault(t *testing.T) {
	setFullValidEnv(t, false)
	t.Setenv("SMTP_PORT", "2525")
	t.Setenv("SMTP_STARTTLS", "false")

	cfg := LoadConfig()
	if cfg.Services.Mail.SMTPPort != 2525 {
		t.Errorf("expected Mail.SMTPPort=2525, got %d", cfg.Services.Mail.SMTPPort)
	}
	if cfg.Services.Mail.StartTLS {
		t.Error("expected Mail.StartTLS=false when SMTP_STARTTLS=false")
	}
	if cfg.Services.Mail.SMTPHost != "smtp.gmail.com" {
		t.Errorf("expected Mail.SMTPHost=smtp.gmail.com, got %q", cfg.Services.Mail.SMTPHost)
	}
}

// TestLoadConfigDevModeAllowsMissingVars verifies dev mode (ZETTEL_DEV=true)
// stays lenient: LoadConfig does not panic even when every production-required
// variable is missing.
func TestLoadConfigDevModeAllowsMissingVars(t *testing.T) {
	setFullValidEnv(t, true)
	for _, k := range prodRequiredEnvKeys {
		os.Unsetenv(k)
	}
	for _, k := range stripeEnvKeys {
		os.Unsetenv(k)
	}
	for _, k := range githubClientEnvKeys {
		os.Unsetenv(k)
	}
	for _, k := range smtpEnvKeys {
		os.Unsetenv(k)
	}
	os.Unsetenv("MAIL_ENABLED")

	cfg := LoadConfig() // must not panic with all required vars missing
	if !cfg.Server.DevMode {
		t.Fatal("expected Server.DevMode=true")
	}
}
