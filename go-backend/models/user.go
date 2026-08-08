package models

import "time"

type User struct {
	ID                          int        `json:"id"`
	Username                    string     `json:"username"`
	Email                       string     `json:"email"`
	Password                    string     `json:"password"`
	CreatedAt                   time.Time  `json:"created_at"`
	UpdatedAt                   time.Time  `json:"updated_at"`
	IsAdmin                     bool       `json:"is_admin"`
	LastLogin                   *time.Time `json:"last_login"`
	LastSeen                    *time.Time `json:"last_seen"`
	EmailValidated              bool       `json:"email_validated"`
	CanUploadFiles              bool       `json:"can_upload_files"`
	MaxFileStorage              int        `json:"max_file_storage"`
	StripeCustomerID            string     `json:"stripe_customer_id"`
	StripeSubscriptionID        string     `json:"stripe_subscription_id"`
	StripeSubscriptionStatus    string     `json:"stripe_subscription_status"`
	StripeSubscriptionFrequency string     `json:"stripe_subscription_frequency"`
	StripeCurrentPlan           string     `json:"stripe_current_plan"`
	IsActive                    bool       `json:"is_active"`
	DashboardCardPK             int        `json:"dashboard_card_pk"`
	CardCount                   int        `json:"card_count"`
	TaskCount                   int        `json:"task_count"`
	FileCount                   int        `json:"file_count"`
	ChatMessageCount            int        `json:"chat_message_count"`
	LLMCost                     float64    `json:"llm_cost"`
	Revenue                     float64    `json:"revenue"`
	HasSeenGettingStarted       bool       `json:"has_seen_getting_started"`
	Timezone                    string     `json:"timezone"`
	ShowTasks                   bool       `json:"show_tasks"`
	ShowRss                     bool       `json:"show_rss"`

	// OIDC / SSO identity. Populated only by OIDC-specific queries; the
	// standard QueryUser* scan intentionally does not read these columns.
	OIDCProvider string `json:"oidc_provider,omitempty"`
	OIDCSub      string `json:"oidc_sub,omitempty"`
}

type UserSubscription struct {
	ID                          int    `json:"id"`
	StripeCustomerID            string `json:"stripe_customer_id"`
	StripeSubscriptionID        string `json:"stripe_subscription_id"`
	StripeSubscriptionStatus    string `json:"stripe_subscription_status"`
	StripeSubscriptionFrequency string `json:"stripe_subscription_frequency"`
	StripeCurrentPlan           string `json:"stripe_current_plan"`
	IsActive                    bool   `json:"is_active"`
}

type EditUserParams struct {
	Username              string `json:"username"`
	Email                 string `json:"email"`
	IsAdmin               bool   `json:"is_admin"`
	DashboardCardPK       int    `json:"dashboard_card_pk"`
	HasSeenGettingStarted bool   `json:"has_seen_getting_started"`
	Timezone              string `json:"timezone"`
	ShowTasks             bool   `json:"show_tasks"`
	ShowRss               bool   `json:"show_rss"`
}

type CreateUserParams struct {
	Username        string `json:"username"`
	Email           string `json:"email"`
	Password        string `json:"password"`
	ConfirmPassword string `json:"confirmPassword"`
}

type CreateUserResponse struct {
	Error   bool   `json:"error"`
	Message string `json:"message"`
	NewID   int    `json:"new_id"`
	User    User   `json:"user"`
}
