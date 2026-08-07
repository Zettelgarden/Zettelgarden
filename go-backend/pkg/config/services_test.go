package config

import (
	"os"
	"testing"
)

const testEnvKey = "STRIPE_TEST_ENV"

// clearStripeEnv unsets every STRIPE_* variable so tests start from a known state.
func clearStripeEnv(t *testing.T) {
	t.Helper()
	for _, k := range []string{
		"STRIPE_ENABLED",
		"STRIPE_SECRET_KEY",
		"STRIPE_PUBLISHABLE_KEY",
		"STRIPE_WEBHOOK_SECRET",
		"STRIPE_MONTH_PRICE",
		"STRIPE_YEAR_PRICE",
		"STRIPE_BILLING_URL",
	} {
		os.Unsetenv(k)
	}
	validationErrors = nil
}

// TestLoadStripeConfigDisabled verifies STRIPE_ENABLED=false makes every
// STRIPE_* value optional: no validation errors are accumulated and the config
// reports disabled, so the server can boot without any Stripe keys.
func TestLoadStripeConfigDisabled(t *testing.T) {
	clearStripeEnv(t)
	t.Setenv("STRIPE_ENABLED", "false")

	cfg := loadStripeConfig()
	if cfg.Enabled {
		t.Fatal("expected Enabled=false when STRIPE_ENABLED=false")
	}
	if len(validationErrors) != 0 {
		t.Fatalf("expected no validation errors when billing disabled, got: %v", validationErrors)
	}
}

// TestLoadStripeConfigEnabledRequiresKeys verifies the default (STRIPE_ENABLED
// unset) keeps existing behavior: billing is enabled and the six values are
// required.
func TestLoadStripeConfigEnabledRequiresKeys(t *testing.T) {
	clearStripeEnv(t)
	os.Unsetenv("STRIPE_ENABLED")

	cfg := loadStripeConfig()
	if !cfg.Enabled {
		t.Fatal("expected Enabled=true by default when STRIPE_ENABLED is unset")
	}
	if len(validationErrors) != 6 {
		t.Fatalf("expected 6 validation errors for missing STRIPE_* keys, got %d: %v", len(validationErrors), validationErrors)
	}
}

// TestLoadStripeConfigEnabledWithKeys verifies a fully-configured install loads
// cleanly with no validation errors.
func TestLoadStripeConfigEnabledWithKeys(t *testing.T) {
	clearStripeEnv(t)
	t.Setenv("STRIPE_SECRET_KEY", "sk_test_123")
	t.Setenv("STRIPE_PUBLISHABLE_KEY", "pk_test_123")
	t.Setenv("STRIPE_WEBHOOK_SECRET", "whsec_123")
	t.Setenv("STRIPE_MONTH_PRICE", "price_month")
	t.Setenv("STRIPE_YEAR_PRICE", "price_year")
	t.Setenv("STRIPE_BILLING_URL", "https://billing.stripe.com/test")

	cfg := loadStripeConfig()
	if !cfg.Enabled {
		t.Fatal("expected Enabled=true when STRIPE_ENABLED is unset and keys present")
	}
	if len(validationErrors) != 0 {
		t.Fatalf("expected no validation errors, got: %v", validationErrors)
	}
}
