package config

// ServiceConfig holds configuration for all external services
type ServiceConfig struct {
	LLM     LLMConfig     // Language model/embedding services
	Mail    MailConfig    // Email service
	Stripe  StripeConfig  // Payment processing
	S3      S3Config      // Object storage
	GitHub  GitHubConfig  // OAuth service
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

// StripeConfig holds payment processing configuration
type StripeConfig struct {
	SecretKey       string // Stripe secret key (sensitive)
	PublishableKey  string // Stripe publishable key
	WebhookSecret   string // Webhook signature secret (sensitive)
	MonthPrice      string // Monthly subscription price ID
	YearPrice       string // Yearly subscription price ID
	BillingURL      string // Billing portal URL
}

// S3Config holds object storage configuration
type S3Config struct {
	AccessKeyID     string // S3/B2 access key ID (sensitive)
	SecretAccessKey string // S3/B2 secret access key (sensitive)
	BucketName      string // Bucket name
}

// GitHubConfig holds OAuth service configuration
type GitHubConfig struct {
	ClientID     string // GitHub OAuth client ID
	ClientSecret string // GitHub OAuth client secret (sensitive)
	RedirectURI  string // OAuth callback URL
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
		S3:      loadS3Config(),
		GitHub:  loadGitHubConfig(),
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

// loadStripeConfig loads Stripe payment configuration
func loadStripeConfig() StripeConfig {
	config := StripeConfig{
		SecretKey:      requireString("STRIPE_SECRET_KEY"),
		PublishableKey: requireString("STRIPE_PUBLISHABLE_KEY"),
		WebhookSecret:  requireString("STRIPE_WEBHOOK_SECRET"),
		MonthPrice:     requireString("STRIPE_MONTH_PRICE"),
		YearPrice:      requireString("STRIPE_YEAR_PRICE"),
		BillingURL:     requireString("STRIPE_BILLING_URL"),
	}

	// Validate URLs
	validateURL("STRIPE_BILLING_URL", config.BillingURL)

	return config
}

// loadS3Config loads S3/B2 object storage configuration
func loadS3Config() S3Config {
	config := S3Config{
		AccessKeyID:     requireString("B2_ACCESS_KEY_ID"),
		SecretAccessKey: requireString("B2_SECRET_ACCESS_KEY"),
		BucketName:      requireString("B2_BUCKET_NAME"),
	}

	// Basic validation that all S3 credentials are provided together
	if (config.AccessKeyID != "" || config.SecretAccessKey != "" || config.BucketName != "") &&
		(config.AccessKeyID == "" || config.SecretAccessKey == "" || config.BucketName == "") {
		validationErrors = append(validationErrors,
			"B2_ACCESS_KEY_ID, B2_SECRET_ACCESS_KEY, and B2_BUCKET_NAME must all be provided together or all be empty")
	}

	return config
}

// loadGitHubConfig loads GitHub OAuth configuration
func loadGitHubConfig() GitHubConfig {
	config := GitHubConfig{
		ClientID:     requireString("GITHUB_CLIENT_ID"),
		ClientSecret: requireString("GITHUB_CLIENT_SECRET"),
		RedirectURI:  requireString("GITHUB_REDIRECT_URI"),
	}

	validateURL("GITHUB_REDIRECT_URI", config.RedirectURI)

	return config
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