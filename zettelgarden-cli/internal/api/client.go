package api

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"time"

	"github.com/zettelgarden/zettelgarden-cli/internal/config"
)

// Client wraps an HTTP client with authentication and retry logic
type Client struct {
	httpClient  *http.Client
	endpoint    string
	profileName string
	timeout     time.Duration
}

// NewClient creates a new API client for a given profile
func NewClient(cfg *config.Config, profileName string) (*Client, error) {
	profile, err := config.GetProfile(cfg, profileName)
	if err != nil {
		return nil, fmt.Errorf("failed to get profile: %w", err)
	}

	timeout := time.Duration(profile.Timeout) * time.Second
	if timeout == 0 {
		timeout = 30 * time.Second
	}

	return &Client{
		httpClient: &http.Client{
			Timeout: timeout,
		},
		endpoint:    profile.Endpoint,
		profileName: profileName,
		timeout:     timeout,
	}, nil
}

// Request represents an HTTP request to be made
type Request struct {
	Method      string
	Path        string
	Body        interface{}
	Headers     map[string]string
	Authenticated bool
}

// Response represents an HTTP response
type Response struct {
	StatusCode int
	Body       []byte
	Headers    http.Header
}

// Do executes an HTTP request with automatic JWT injection and retry logic
func (c *Client) Do(ctx context.Context, req Request) (*Response, error) {
	// Build full URL
	url := c.endpoint + req.Path

	// Marshal body if provided
	var bodyReader io.Reader
	if req.Body != nil {
		bodyBytes, err := json.Marshal(req.Body)
		if err != nil {
			return nil, fmt.Errorf("failed to marshal request body: %w", err)
		}
		bodyReader = bytes.NewReader(bodyBytes)
	}

	// Create HTTP request
	httpReq, err := http.NewRequestWithContext(ctx, req.Method, url, bodyReader)
	if err != nil {
		return nil, fmt.Errorf("failed to create request: %w", err)
	}

	// Set default headers
	httpReq.Header.Set("Content-Type", "application/json")
	httpReq.Header.Set("Accept", "application/json")

	// Add custom headers
	for key, value := range req.Headers {
		httpReq.Header.Set(key, value)
	}

	// Add authentication if required
	if req.Authenticated {
		token, err := config.GetToken(c.profileName)
		if err != nil {
			return nil, fmt.Errorf("authentication required but no valid token found: %w", err)
		}
		httpReq.Header.Set("Authorization", "Bearer "+token)
	}

	// Execute request with retry logic
	var lastErr error
	maxRetries := 3
	for attempt := 0; attempt < maxRetries; attempt++ {
		if attempt > 0 {
			// Exponential backoff: 1s, 2s, 4s
			backoff := time.Duration(1<<uint(attempt-1)) * time.Second
			time.Sleep(backoff)
		}

		httpResp, err := c.httpClient.Do(httpReq)
		if err != nil {
			lastErr = fmt.Errorf("request failed: %w", err)
			// Retry on network errors
			continue
		}

		// Read response body
		bodyBytes, err := io.ReadAll(httpResp.Body)
		httpResp.Body.Close()
		if err != nil {
			lastErr = fmt.Errorf("failed to read response body: %w", err)
			continue
		}

		resp := &Response{
			StatusCode: httpResp.StatusCode,
			Body:       bodyBytes,
			Headers:    httpResp.Header,
		}

		// Don't retry on successful responses or client errors (4xx)
		if httpResp.StatusCode < 500 {
			return resp, nil
		}

		// Retry on server errors (5xx)
		lastErr = fmt.Errorf("server error: %d", httpResp.StatusCode)
	}

	return nil, fmt.Errorf("request failed after %d attempts: %w", maxRetries, lastErr)
}

// DoJSON executes an HTTP request and unmarshals the JSON response
func (c *Client) DoJSON(ctx context.Context, req Request, result interface{}) error {
	resp, err := c.Do(ctx, req)
	if err != nil {
		return err
	}

	// Check for error status codes
	if resp.StatusCode >= 400 {
		// Try to parse error message from response
		var errResp struct {
			Error   string `json:"error"`
			Message string `json:"message"`
		}
		if err := json.Unmarshal(resp.Body, &errResp); err == nil {
			if errResp.Error != "" {
				return fmt.Errorf("API error (status %d): %s", resp.StatusCode, errResp.Error)
			}
			if errResp.Message != "" {
				return fmt.Errorf("API error (status %d): %s", resp.StatusCode, errResp.Message)
			}
		}
		return fmt.Errorf("API error (status %d): %s", resp.StatusCode, string(resp.Body))
	}

	// Unmarshal response
	if result != nil && len(resp.Body) > 0 {
		if err := json.Unmarshal(resp.Body, result); err != nil {
			return fmt.Errorf("failed to unmarshal response: %w", err)
		}
	}

	return nil
}

// Login authenticates with the API and stores the token
func (c *Client) Login(ctx context.Context, email, password string) error {
	type loginRequest struct {
		Email    string `json:"email"`
		Password string `json:"password"`
	}

	type loginResponse struct {
		AccessToken string `json:"access_token"`
		User        struct {
			ID       int    `json:"id"`
			Email    string `json:"email"`
			Username string `json:"username"`
		} `json:"user"`
		Message string `json:"message"`
	}

	req := Request{
		Method: "POST",
		Path:   "/api/login",
		Body: loginRequest{
			Email:    email,
			Password: password,
		},
		Authenticated: false,
	}

	var resp loginResponse
	if err := c.DoJSON(ctx, req, &resp); err != nil {
		return err
	}

	if resp.AccessToken == "" {
		if resp.Message != "" {
			return fmt.Errorf("login failed: %s", resp.Message)
		}
		return fmt.Errorf("login failed: no token received")
	}

	// Store token with 15 day expiry (matching backend's token generation)
	expiry := time.Now().Add(15 * 24 * time.Hour)
	if err := config.SetToken(c.profileName, c.endpoint, resp.AccessToken, expiry); err != nil {
		return fmt.Errorf("failed to store token: %w", err)
	}

	return nil
}

// CheckToken verifies if the current token is valid
func (c *Client) CheckToken(ctx context.Context) (bool, error) {
	// First check if we have a token stored locally
	if !config.IsTokenValid(c.profileName) {
		return false, nil
	}

	// Verify token with the API
	req := Request{
		Method:        "GET",
		Path:          "/api/auth",
		Authenticated: true,
	}

	var result struct {
		User struct {
			ID       int    `json:"id"`
			Email    string `json:"email"`
			Username string `json:"username"`
		} `json:"user"`
	}

	if err := c.DoJSON(ctx, req, &result); err != nil {
		return false, nil
	}

	return true, nil
}
