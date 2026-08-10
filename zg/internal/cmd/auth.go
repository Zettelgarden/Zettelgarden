package cmd

import (
	"bufio"
	"encoding/json"
	"fmt"
	"net/http"
	"os"
	"strings"

	"github.com/nick-zettelgarden/zg/internal/api"
	"github.com/nick-zettelgarden/zg/internal/config"
	"github.com/nick-zettelgarden/zg/internal/output"
	"github.com/spf13/cobra"
	"golang.org/x/term"
)

// ---------------------------------------------------------------------------
// `zg auth` — durable authentication via long-lived API keys
//
// The web UI issues short-lived JWTs that expire after 15 days. `zg auth`
// mints and stores a durable API key (Settings → API Keys) so CLI auth
// doesn't silently rot. Token storage precedence: CLI flag > ZETTELGARDEN_TOKEN
// env > OS keyring > config file.
// ---------------------------------------------------------------------------

var (
	authEmail    string
	authPassword string
	authKeyName  string
)

var authCmd = &cobra.Command{
	Use:   "auth",
	Short: "Manage authentication (API keys)",
	Long: `Manage durable authentication for the CLI.

The Zettelgarden web UI session tokens expire after 15 days. Use API keys
instead: they never expire and work with the same Bearer header.

  zg auth login             Log in interactively and store an API key
  zg auth set <token>       Store an API key created in the web UI
  zg auth status            Show which token is in use and warn about JWTs
  zg auth revoke            Revoke the stored API key and remove it locally

Token precedence: --token flag > ZETTELGARDEN_TOKEN env > OS keyring > config file.`,
}

var authLoginCmd = &cobra.Command{
	Use:   "login",
	Short: "Log in and store a durable API key",
	Long: `Authenticate with email/password, mint a long-lived API key via the
backend, and store it securely (OS keyring, falling back to the config file
with 0600 permissions).`,
	RunE: runAuthLogin,
}

var authSetCmd = &cobra.Command{
	Use:   "set <token>",
	Short: "Store an API key created in the web UI",
	Args:  cobra.ExactArgs(1),
	RunE:  runAuthSet,
}

var authStatusCmd = &cobra.Command{
	Use:   "status",
	Short: "Show which token is in use and warn about short-lived JWTs",
	RunE:  runAuthStatus,
}

var authRevokeCmd = &cobra.Command{
	Use:   "revoke",
	Short: "Revoke the stored API key and remove it locally",
	RunE:  runAuthRevoke,
}

// authStatus is the machine-readable output of `zg auth status`.
type authStatus struct {
	APIURL      string `json:"api_url"`
	TokenSource string `json:"token_source"` // flag | env | keyring | config | none
	APIKeyName  string `json:"api_key_name,omitempty"`
	APIKeyID    int    `json:"api_key_id,omitempty"`
	TokenValid  *bool  `json:"token_valid,omitempty"`
	Warning     string `json:"warning,omitempty"`
}

// FormatHuman implements output.HumanFormatter for `--pretty` mode.
func (s authStatus) FormatHuman() string {
	var b strings.Builder
	fmt.Fprintf(&b, "API URL:     %s\n", s.APIURL)
	fmt.Fprintf(&b, "Token source: %s\n", s.TokenSource)
	if s.APIKeyName != "" {
		fmt.Fprintf(&b, "API key:      %s", s.APIKeyName)
		if s.APIKeyID > 0 {
			fmt.Fprintf(&b, " (id %d)", s.APIKeyID)
		}
		b.WriteString("\n")
	}
	if s.TokenValid != nil {
		fmt.Fprintf(&b, "Token valid:  %v\n", *s.TokenValid)
	}
	if s.Warning != "" {
		fmt.Fprintf(&b, "Warning:      %s\n", s.Warning)
	}
	return strings.TrimRight(b.String(), "\n")
}

func init() {
	authLoginCmd.Flags().StringVar(&authEmail, "email", "", "Account email (prompted if omitted)")
	authLoginCmd.Flags().StringVar(&authPassword, "password", "", "Account password (prompted securely if omitted)")
	authLoginCmd.Flags().StringVar(&authKeyName, "name", "zg-cli", "API key name")
	authSetCmd.Flags().StringVar(&authKeyName, "name", "", "API key name (for status/revoke bookkeeping)")

	authCmd.AddCommand(authLoginCmd)
	authCmd.AddCommand(authSetCmd)
	authCmd.AddCommand(authStatusCmd)
	authCmd.AddCommand(authRevokeCmd)
}

// GetAuthCmd returns the auth command for registration in main.
func GetAuthCmd() *cobra.Command {
	return authCmd
}

func runAuthLogin(cmd *cobra.Command, args []string) error {
	configPath, err := getConfigPath()
	if err != nil {
		return output.WriteError(os.Stdout, "Config error", err.Error())
	}
	cfg, err := config.LoadOrCreate(configPath)
	if err != nil {
		return output.WriteError(os.Stdout, "Config error", err.Error())
	}
	if getAPIURL() != "" {
		cfg.APIURL = getAPIURL()
	}

	email := authEmail
	if email == "" {
		email, err = promptText("Email: ")
		if err != nil {
			return output.WriteError(os.Stdout, "Read error", err.Error())
		}
	}
	password := authPassword
	if password == "" {
		password, err = readSecret("Password: ")
		if err != nil {
			return output.WriteError(os.Stdout, "Read error", err.Error())
		}
	}
	if email == "" || password == "" {
		return output.WriteError(os.Stdout, "Email and password are required", "")
	}

	// 1. Exchange credentials for a session JWT (used once, below).
	client := api.NewClient(cfg.APIURL, "", cfg.TimeoutSeconds)
	loginBody, _ := json.Marshal(map[string]string{"email": email, "password": password})
	resp, err := client.PostNoAuth("/api/login", loginBody)
	if err != nil {
		return output.WriteError(os.Stdout, "API request failed", err.Error())
	}
	body, err := api.GetBodyBytes(resp)
	if err != nil {
		return output.WriteError(os.Stdout, "Reading response failed", err.Error())
	}
	if resp.StatusCode != http.StatusOK {
		return output.WriteError(os.Stdout, fmt.Sprintf("Login failed (%d)", resp.StatusCode), loginErrorMessage(body))
	}
	var loginResp struct {
		AccessToken string `json:"access_token"`
	}
	if err := json.Unmarshal(body, &loginResp); err != nil || loginResp.AccessToken == "" {
		return output.WriteError(os.Stdout, "Login failed", "no access token in response")
	}

	// 2. Mint a durable API key with the session token.
	authed := api.NewClient(cfg.APIURL, loginResp.AccessToken, cfg.TimeoutSeconds)
	createBody, _ := json.Marshal(map[string]string{
		"name":        authKeyName,
		"description": "Created by zg CLI",
	})
	resp, err = authed.Post("/api/api-keys", createBody)
	if err != nil {
		return output.WriteError(os.Stdout, "API request failed", err.Error())
	}
	body, err = api.GetBodyBytes(resp)
	if err != nil {
		return output.WriteError(os.Stdout, "Reading response failed", err.Error())
	}
	switch resp.StatusCode {
	case http.StatusConflict:
		return output.WriteError(os.Stdout, "API key name already in use",
			fmt.Sprintf("An active API key named %q exists. Revoke it (Settings → API Keys or `zg auth revoke`) or pick another with --name.", authKeyName))
	case http.StatusUnauthorized:
		return output.WriteError(os.Stdout, "Not authorized", "the session token was rejected; check your credentials")
	case http.StatusCreated, http.StatusOK:
		// expected
	default:
		return output.WriteError(os.Stdout, fmt.Sprintf("Create API key failed (%d)", resp.StatusCode), string(body))
	}
	var keyResp struct {
		ID   int    `json:"id"`
		Name string `json:"name"`
		Key  string `json:"key"`
	}
	if err := json.Unmarshal(body, &keyResp); err != nil || keyResp.Key == "" {
		return output.WriteError(os.Stdout, "Create API key failed", "unexpected response")
	}

	// 3. Store the durable key (keyring preferred; config file fallback).
	cfg.APIKeyName = keyResp.Name
	cfg.APIKeyID = keyResp.ID
	stored, err := config.StoreToken(configPath, cfg, keyResp.Key)
	if err != nil {
		return output.WriteError(os.Stdout, "Storing API key failed", err.Error())
	}

	return output.WriteMessage(os.Stdout,
		fmt.Sprintf("API key %q (id %d) created and stored in %s. CLI auth now uses a durable key that does not expire.", keyResp.Name, keyResp.ID, stored))
}

func runAuthSet(cmd *cobra.Command, args []string) error {
	token := strings.TrimSpace(args[0])
	if token == "" {
		return output.WriteError(os.Stdout, "Token is required", "")
	}
	configPath, err := getConfigPath()
	if err != nil {
		return output.WriteError(os.Stdout, "Config error", err.Error())
	}
	cfg, err := config.LoadOrCreate(configPath)
	if err != nil {
		return output.WriteError(os.Stdout, "Config error", err.Error())
	}
	if getAPIURL() != "" {
		cfg.APIURL = getAPIURL()
	}

	if config.IsJWT(token) {
		fmt.Fprintln(os.Stderr, "warning: "+config.JWTMigrationNotice())
	}
	if config.IsAPIKey(token) && authKeyName != "" {
		cfg.APIKeyName = authKeyName
	}

	stored, err := config.StoreToken(configPath, cfg, token)
	if err != nil {
		return output.WriteError(os.Stdout, "Storing token failed", err.Error())
	}
	return output.WriteMessage(os.Stdout, fmt.Sprintf("API token stored in %s", stored))
}

func runAuthStatus(cmd *cobra.Command, args []string) error {
	cfg, err := loadConfig()
	if err != nil {
		return output.WriteError(os.Stdout, "Config error", err.Error())
	}
	token, source, err := cfg.ResolveToken(getAPIToken())
	if err != nil {
		return output.WriteError(os.Stdout, "Config error", err.Error())
	}

	status := authStatus{
		APIURL:      cfg.APIURL,
		TokenSource: string(source),
		APIKeyName:  cfg.APIKeyName,
		APIKeyID:    cfg.APIKeyID,
	}
	if token != "" && config.IsJWT(token) {
		status.Warning = config.JWTMigrationNotice()
	}

	// Warn when the config file is world-readable (the token may be in it).
	if configPath, err := getConfigPath(); err == nil {
		if info, err := os.Stat(configPath); err == nil && info.Mode().Perm()&0o077 != 0 {
			permWarning := fmt.Sprintf("config file is world-readable; run 'chmod 600 %s'", configPath)
			if status.Warning != "" {
				status.Warning += " " + permWarning
			} else {
				status.Warning = permWarning
			}
		}
	}

	if token != "" {
		client := api.NewClient(cfg.APIURL, token, cfg.TimeoutSeconds)
		resp, err := client.Get("/api/api-keys")
		if err == nil {
			body, readErr := api.GetBodyBytes(resp)
			valid := resp.StatusCode == http.StatusOK &&
				strings.Contains(resp.Header.Get("Content-Type"), "application/json") &&
				readErr == nil && json.Valid(body)
			status.TokenValid = &valid
		}
	}

	return output.WriteSuccess(os.Stdout, status)
}

func runAuthRevoke(cmd *cobra.Command, args []string) error {
	cfg, err := loadConfig()
	if err != nil {
		return output.WriteError(os.Stdout, "Config error", err.Error())
	}
	token, _, err := cfg.ResolveToken(getAPIToken())
	if err != nil {
		return output.WriteError(os.Stdout, "Config error", err.Error())
	}
	if token == "" {
		return output.WriteError(os.Stdout, "No token configured", "Run `zg auth login` or `zg auth set <token>` first")
	}

	if cfg.APIKeyID > 0 {
		client := api.NewClient(cfg.APIURL, token, cfg.TimeoutSeconds)
		resp, err := client.Delete(fmt.Sprintf("/api/api-keys/%d", cfg.APIKeyID))
		if err != nil {
			fmt.Fprintf(os.Stderr, "warning: failed to revoke API key on server: %v\n", err)
		} else {
			api.GetBodyBytes(resp)
			switch resp.StatusCode {
			case http.StatusNoContent:
				// revoked
			case http.StatusNotFound:
				fmt.Fprintln(os.Stderr, "note: API key was already revoked")
			default:
				fmt.Fprintf(os.Stderr, "warning: server returned %d while revoking API key\n", resp.StatusCode)
			}
		}
	}

	configPath, err := getConfigPath()
	if err != nil {
		return output.WriteError(os.Stdout, "Config error", err.Error())
	}
	if err := config.ClearToken(configPath, cfg); err != nil {
		return output.WriteError(os.Stdout, "Clearing token failed", err.Error())
	}
	return output.WriteMessage(os.Stdout, "API key revoked and removed from local storage")
}

// loginErrorMessage extracts a human-readable message from a failed /api/login
// response body.
func loginErrorMessage(body []byte) string {
	var resp struct {
		Message string `json:"message"`
		Error   string `json:"error"`
	}
	if err := json.Unmarshal(body, &resp); err == nil {
		if resp.Message != "" {
			return resp.Message
		}
		if resp.Error != "" {
			return resp.Error
		}
	}
	return string(body)
}

// promptText reads a line from stdin (works with a TTY or piped input).
func promptText(prompt string) (string, error) {
	fmt.Fprint(os.Stderr, prompt)
	line, err := bufio.NewReader(os.Stdin).ReadString('\n')
	return strings.TrimSpace(line), err
}

// readSecret reads a secret from stdin without echoing it back when stdin is
// a terminal; falls back to reading a plain line when piped.
func readSecret(prompt string) (string, error) {
	fmt.Fprint(os.Stderr, prompt)
	if term.IsTerminal(int(os.Stdin.Fd())) {
		value, err := term.ReadPassword(int(os.Stdin.Fd()))
		fmt.Fprintln(os.Stderr)
		return strings.TrimSpace(string(value)), err
	}
	line, err := bufio.NewReader(os.Stdin).ReadString('\n')
	return strings.TrimSpace(line), err
}
