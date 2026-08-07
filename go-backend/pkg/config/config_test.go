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
	t.Setenv("MAIL_HOST", "smtp.gmail.com")
	t.Setenv("MAIL_PASSWORD", "test-mail-password")
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
// mode (dev mode tolerates all of them missing). STRIPE_* and the GitHub/OIDC
// client vars are excluded because their opt-outs are covered separately.
var prodRequiredEnvKeys = []string{
	"ZETTEL_URL",
	"ZETTEL_ADMIN_EMAIL",
	"SECRET_KEY",
	"ZETTEL_LLM_KEY",
	"ZETTEL_LLM_ENDPOINT",
	"ZETTEL_LLM_DEFAULT_MODEL",
	"ZETTEL_LLM_SUMMARIZE_MODEL",
	"MAIL_HOST",
	"MAIL_PASSWORD",
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

	cfg := LoadConfig() // must not panic with all required vars missing
	if !cfg.Server.DevMode {
		t.Fatal("expected Server.DevMode=true")
	}
}
