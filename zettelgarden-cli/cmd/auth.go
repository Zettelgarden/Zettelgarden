package cmd

import (
	"context"
	"encoding/json"
	"fmt"
	"os"
	"syscall"
	"time"

	"github.com/spf13/cobra"
	"github.com/zettelgarden/zettelgarden-cli/internal/api"
	"github.com/zettelgarden/zettelgarden-cli/internal/config"
	"golang.org/x/term"
)

var authCmd = &cobra.Command{
	Use:   "auth",
	Short: "Manage authentication",
	Long:  `Authenticate with Zettelgarden API and manage login status.`,
}

var authLoginCmd = &cobra.Command{
	Use:   "login [email]",
	Short: "Login to Zettelgarden",
	Long: `Authenticate with Zettelgarden API using email and password.
The authentication token will be stored securely for subsequent requests.

Examples:
  zg auth login user@example.com
  zg auth login --profile prod user@example.com`,
	Args: cobra.MaximumNArgs(1),
	RunE: func(cmd *cobra.Command, args []string) error {
		// Load config
		cfg, err := config.LoadConfig()
		if err != nil {
			return fmt.Errorf("failed to load config: %w", err)
		}

		// Get active profile name
		profileName := config.GetActiveProfileName(cfg)

		// Prompt for email if not provided
		var email string
		if len(args) > 0 {
			email = args[0]
		} else {
			fmt.Fprint(os.Stderr, "Email: ")
			fmt.Scanln(&email)
		}

		if email == "" {
			return fmt.Errorf("email is required")
		}

		// Prompt for password (hidden input)
		fmt.Fprint(os.Stderr, "Password: ")
		passwordBytes, err := term.ReadPassword(int(syscall.Stdin))
		fmt.Fprintln(os.Stderr) // New line after password input
		if err != nil {
			return fmt.Errorf("failed to read password: %w", err)
		}

		password := string(passwordBytes)
		if password == "" {
			return fmt.Errorf("password is required")
		}

		// Create API client
		client, err := api.NewClient(cfg, profileName)
		if err != nil {
			return fmt.Errorf("failed to create API client: %w", err)
		}

		// Attempt login
		ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
		defer cancel()

		if err := client.Login(ctx, email, password); err != nil {
			return fmt.Errorf("login failed: %w", err)
		}

		// Success - output JSON response
		response := map[string]interface{}{
			"success": true,
			"profile": profileName,
			"email":   email,
			"message": "Login successful",
		}

		data, err := json.MarshalIndent(response, "", "  ")
		if err != nil {
			return fmt.Errorf("failed to format response: %w", err)
		}

		fmt.Println(string(data))
		return nil
	},
}

var authLogoutCmd = &cobra.Command{
	Use:   "logout",
	Short: "Logout from Zettelgarden",
	Long: `Remove stored authentication token for the current profile.

Examples:
  zg auth logout
  zg auth logout --profile prod`,
	RunE: func(cmd *cobra.Command, args []string) error {
		// Load config
		cfg, err := config.LoadConfig()
		if err != nil {
			return fmt.Errorf("failed to load config: %w", err)
		}

		// Get active profile name
		profileName := config.GetActiveProfileName(cfg)

		// Check if already logged out
		if !config.IsTokenValid(profileName) {
			response := map[string]interface{}{
				"success": true,
				"profile": profileName,
				"message": "Already logged out",
			}

			data, err := json.MarshalIndent(response, "", "  ")
			if err != nil {
				return fmt.Errorf("failed to format response: %w", err)
			}

			fmt.Println(string(data))
			return nil
		}

		// Clear token
		if err := config.ClearToken(profileName); err != nil {
			return fmt.Errorf("failed to clear token: %w", err)
		}

		// Success response
		response := map[string]interface{}{
			"success": true,
			"profile": profileName,
			"message": "Logout successful",
		}

		data, err := json.MarshalIndent(response, "", "  ")
		if err != nil {
			return fmt.Errorf("failed to format response: %w", err)
		}

		fmt.Println(string(data))
		return nil
	},
}

var authStatusCmd = &cobra.Command{
	Use:   "status",
	Short: "Check authentication status",
	Long: `Check if you are currently authenticated with Zettelgarden API.

Examples:
  zg auth status
  zg auth status --profile prod`,
	RunE: func(cmd *cobra.Command, args []string) error {
		// Load config
		cfg, err := config.LoadConfig()
		if err != nil {
			return fmt.Errorf("failed to load config: %w", err)
		}

		// Get active profile name
		profileName := config.GetActiveProfileName(cfg)

		// Check if token exists and is not expired locally
		isValid := config.IsTokenValid(profileName)

		response := map[string]interface{}{
			"profile":        profileName,
			"authenticated":  isValid,
		}

		if isValid {
			// Get token details
			creds, err := config.LoadCredentials()
			if err == nil {
				if profileCred, exists := creds.Profiles[profileName]; exists {
					response["endpoint"] = profileCred.Endpoint
					response["token_expiry"] = profileCred.TokenExpiry.Format(time.RFC3339)

					// Optionally verify with API
					client, err := api.NewClient(cfg, profileName)
					if err == nil {
						ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
						defer cancel()

						valid, err := client.CheckToken(ctx)
						if err == nil {
							response["token_valid_with_api"] = valid
							if !valid {
								response["authenticated"] = false
								response["message"] = "Token is invalid or expired on server"
							}
						}
					}
				}
			}
		} else {
			response["message"] = "Not authenticated"
		}

		data, err := json.MarshalIndent(response, "", "  ")
		if err != nil {
			return fmt.Errorf("failed to format response: %w", err)
		}

		fmt.Println(string(data))
		return nil
	},
}

func init() {
	rootCmd.AddCommand(authCmd)
	authCmd.AddCommand(authLoginCmd)
	authCmd.AddCommand(authLogoutCmd)
	authCmd.AddCommand(authStatusCmd)
}
