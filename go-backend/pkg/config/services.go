package config

import (
	"fmt"
	"net/url"
	"os"
)

// ServiceConfig holds configuration for all external services
type ServiceConfig struct {
	LLM     LLMConfig     // Language model/embedding services
	Mail    MailConfig    // Email service
	Stripe  StripeConfig  // Payment processing
	Storage StorageConfig // Local on-disk file storage
	GitHub  GitHubConfig  // OAuth service
	OIDC    OIDCConfig    // Generic OIDC / SSO (e.g. Pocket ID) — opt-in
	Search  SearchConfig  // Search engine
}

// LLMConfig holds language model service configuration
type LLMConfig struct {
	APIKey          string // OpenAI-compatible API key (sensitive)
	Endpoint        string // Base URL for LLM API
	DefaultModel    string // Default model name
	SummarizeModel  string // Model for summarization tasks
	ChunkingEnabled bool   // Enable chunking/embeddings feature
}

// MailConfig holds email service configuration
type MailConfig struct {
	Host     string // Mail server hostname
	Password string // Mail server password (sensitive)
}

// StripeConfig holds payment processing configuration. Enabled defaults to
// true (existing installs keep working unchanged); set STRIPE_ENABLED=false to
// disable billing entirely — the STRIPE_* values become optional, the billing
// routes return 404, and Stripe is never initialized.
type StripeConfig struct {
	Enabled        bool   // If false, billing routes return 404 and Stripe is never initialized
	SecretKey      string // Stripe secret key (sensitive)
	PublishableKey string // Stripe publishable key
	WebhookSecret  string // Webhook signature secret (sensitive)
	MonthPrice     string // Monthly subscription price ID
	YearPrice      string // Yearly subscription price ID
	BillingURL     string // Billing portal URL
}

// StorageConfig holds local on-disk file storage configuration.
// Replaces the former S3Config (Backblaze B2) after the storage migration
// (see docs/plans/2026-07-29-s3-to-local-file-storage-design.md).
type StorageConfig struct {
	Dir string // Path to the file storage root (absolute or relative)
}

// GitHubConfig holds OAuth service configuration. Enabled defaults to true
// (GitHub login is the long-standing default); set GITHUB_AUTH_ENABLED=false
// to disable the routes when another provider (e.g. generic OIDC) replaces it.
type GitHubConfig struct {
	Enabled      bool   // If false, the GitHub OAuth routes return 404
	ClientID     string // GitHub OAuth client ID
	ClientSecret string // GitHub OAuth client secret (sensitive)
	RedirectURI  string // OAuth callback URL
}

// OIDCConfig holds configuration for a generic OpenID Connect provider
// (e.g. Pocket ID). All fields are optional; OIDC is opt-in via OIDC_ENABLED.
// When enabled, Issuer/ClientID/ClientSecret/RedirectURI become required.
type OIDCConfig struct {
	Enabled       bool   // If false, the OIDC routes return 404
	Issuer        string // IdP issuer URL (used for discovery), e.g. https://pocket-id.example
	ClientID      string // OIDC client ID registered at the IdP
	ClientSecret  string // OIDC client secret (sensitive)
	RedirectURI   string // Our callback URL, e.g. https://app/api/auth/oidc/callback
	ProviderLabel string // Stored in users.oidc_provider; defaults to OIDC_ISSUER host
}

// SearchConfig holds search engine configuration
type SearchConfig struct {
	Host       string // Typesense host URL
	Password   string // Typesense API key (sensitive)
	Collection string // Typesense collection name
}

// LoadServiceConfig loads and validates service configuration from environment variables
func loadServiceConfig() ServiceConfig {
	return ServiceConfig{
		LLM:     loadLLMConfig(),
		Mail:    loadMailConfig(),
		Stripe:  loadStripeConfig(),
		Storage: loadStorageConfig(),
		GitHub:  loadGitHubConfig(),
		OIDC:    loadOIDCConfig(),
		Search:  loadSearchConfig(),
	}
}

// loadLLMConfig loads LLM service configuration
func loadLLMConfig() LLMConfig {
	config := LLMConfig{
		APIKey:          requireString("ZETTEL_LLM_KEY"),
		Endpoint:        requireString("ZETTEL_LLM_ENDPOINT"),
		DefaultModel:    requireString("ZETTEL_LLM_DEFAULT_MODEL"),
		SummarizeModel:  requireString("ZETTEL_LLM_SUMMARIZE_MODEL"),
		ChunkingEnabled: optionalBool("ZETTEL_RUN_CHUNKING_EMBEDDING"),
	}

	// Validate API key length (basic security check)
	if len(config.APIKey) < 10 {
		validationErrors = append(validationErrors,
			"ZETTEL_LLM_KEY appears to be too short or invalid")
	}

	validateURL("ZETTEL_LLM_ENDPOINT", config.Endpoint)

	return config
}

// loadMailConfig loads mail service configuration
func loadMailConfig() MailConfig {
	return MailConfig{
		Host:     requireString("MAIL_HOST"),
		Password: requireString("MAIL_PASSWORD"),
	}
}

// loadStripeConfig loads Stripe payment configuration. Billing is enabled by
// default (STRIPE_ENABLED unset or true); when explicitly disabled via
// STRIPE_ENABLED=false the six STRIPE_* values are optional and unvalidated,
// mirroring the opt-in OIDC config (see loadOIDCConfig).
func loadStripeConfig() StripeConfig {
	enabled := true
	if v := os.Getenv("STRIPE_ENABLED"); v != "" {
		enabled = requireBool("STRIPE_ENABLED")
	}

	config := StripeConfig{Enabled: enabled}
	if !enabled {
		return config
	}

	config.SecretKey = requireString("STRIPE_SECRET_KEY")
	config.PublishableKey = requireString("STRIPE_PUBLISHABLE_KEY")
	config.WebhookSecret = requireString("STRIPE_WEBHOOK_SECRET")
	config.MonthPrice = requireString("STRIPE_MONTH_PRICE")
	config.YearPrice = requireString("STRIPE_YEAR_PRICE")
	config.BillingURL = requireString("STRIPE_BILLING_URL")

	// Validate URLs
	validateURL("STRIPE_BILLING_URL", config.BillingURL)

	return config
}

// loadStorageConfig loads local file storage configuration (STORAGE_DIR).
// The storage root is created at load time (mode 0750) so a fresh checkout
// with the default ./data/files works without a manual mkdir, and validation
// fails if the dir can't be created or written to — the one correctness
// property local disk doesn't give us for free (design decision D3).
func loadStorageConfig() StorageConfig {
	config := StorageConfig{
		Dir: requireStringWithDefault("STORAGE_DIR", "./data/files"),
	}

	if err := os.MkdirAll(config.Dir, 0o750); err != nil {
		validationErrors = append(validationErrors,
			fmt.Sprintf("STORAGE_DIR %q cannot be created: %v", config.Dir, err))
	} else if err := checkDirWritable(config.Dir); err != nil {
		validationErrors = append(validationErrors,
			fmt.Sprintf("STORAGE_DIR %q is not writable: %v", config.Dir, err))
	}

	return config
}

// checkDirWritable confirms a directory is writable by creating and removing
// a throwaway temp file inside it.
func checkDirWritable(dir string) error {
	f, err := os.CreateTemp(dir, ".storage-probe-*")
	if err != nil {
		return err
	}
	f.Close()
	return os.Remove(f.Name())
}

// loadGitHubConfig loads GitHub OAuth configuration. GitHub login is enabled by
// default (GITHUB_AUTH_ENABLED defaults to true when unset) so existing
// installs keep working; when explicitly disabled, the three client values are
// no longer required (mirroring the opt-in OIDC config).
func loadGitHubConfig() GitHubConfig {
	enabled := true
	if v := os.Getenv("GITHUB_AUTH_ENABLED"); v != "" {
		enabled = requireBool("GITHUB_AUTH_ENABLED")
	}
	config := GitHubConfig{
		Enabled:      enabled,
		ClientID:     optionalString("GITHUB_CLIENT_ID"),
		ClientSecret: optionalString("GITHUB_CLIENT_SECRET"),
		RedirectURI:  optionalString("GITHUB_REDIRECT_URI"),
	}

	if enabled {
		requireString("GITHUB_CLIENT_ID")
		requireString("GITHUB_CLIENT_SECRET")
		requireString("GITHUB_REDIRECT_URI")
		validateURL("GITHUB_REDIRECT_URI", config.RedirectURI)
	}

	return config
}

// loadOIDCConfig loads optional OIDC / SSO configuration. Unlike the GitHub
// config (which is required), OIDC is opt-in: all values are read with optional
// getters. When OIDC_ENABLED is true, the four provider values become required
// and are validated; otherwise they may be empty.
func loadOIDCConfig() OIDCConfig {
	cfg := OIDCConfig{
		Enabled:      optionalBool("OIDC_ENABLED"),
		Issuer:       optionalString("OIDC_ISSUER"),
		ClientID:     optionalString("OIDC_CLIENT_ID"),
		ClientSecret: optionalString("OIDC_CLIENT_SECRET"),
		RedirectURI:  optionalString("OIDC_REDIRECT_URI"),
	}
	if cfg.Enabled {
		requireString("OIDC_ISSUER")
		requireString("OIDC_CLIENT_ID")
		requireString("OIDC_CLIENT_SECRET")
		requireString("OIDC_REDIRECT_URI")
		validateURL("OIDC_ISSUER", cfg.Issuer)
		validateURL("OIDC_REDIRECT_URI", cfg.RedirectURI)

		// ProviderLabel is stored in users.oidc_provider for stable
		// (provider, sub) lookups. Default to the issuer host so the label
		// tracks the actual IdP without extra config; an explicit
		// OIDC_PROVIDER_LABEL overrides it (e.g. "google", "okta").
		cfg.ProviderLabel = optionalString("OIDC_PROVIDER_LABEL")
		if cfg.ProviderLabel == "" {
			if u, err := url.Parse(cfg.Issuer); err == nil && u.Host != "" {
				cfg.ProviderLabel = u.Host
			} else {
				cfg.ProviderLabel = "oidc"
			}
		}
	}
	return cfg
}

// loadSearchConfig loads search engine configuration
func loadSearchConfig() SearchConfig {
	config := SearchConfig{
		Host:       requireString("TYPESENSE_HOST"),
		Password:   requireString("TYPESENSE_PASSWORD"),
		Collection: requireString("TYPESENSE_COLLECTION"),
	}

	validateURL("TYPESENSE_HOST", config.Host)

	return config
}
