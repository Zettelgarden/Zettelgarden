# zg CLI Implementation Plan

> **For Claude:** REQUIRED SUB-SKILL: Use superpowers:executing-plans to implement this plan task-by-task.

**Goal:** Build a standalone Go CLI tool (`zg`) for Zettelgarden card and task CRUD operations, optimized for AI agent usage with JSON output.

**Architecture:** Standalone Go binary using Cobra for CLI framework, communicating with existing Zettelgarden REST API via HTTP client. Configuration via JSON file in `~/.config/zettelgarden/`.

**Tech Stack:** Go 1.23+, Cobra (CLI framework), http.Client (API), standard library JSON handling

---

## Project Setup

### Task 1: Initialize Go module and project structure

**Files:**
- Create: `zg/go.mod`
- Create: `zg/README.md`
- Create: `zg/cmd/zg/main.go`
- Create: `zg/internal/config/config.go`
- Create: `zg/internal/api/client.go`
- Create: `zg/internal/output/writer.go`

**Step 1: Initialize Go module**

Run:
```bash
cd /home/nick/code/Zettelgarden
mkdir -p zg
cd zg
go mod init github.com/nick-zettelgarden/zg
```

**Step 2: Install Cobra dependency**

Run:
```bash
go get -u github.com/spf13/cobra@latest
```

**Step 3: Create main.go with basic Cobra setup**

Create: `zg/cmd/zg/main.go`
```go
package main

import (
	"fmt"
	"os"

	"github.com/spf13/cobra"
)

var rootCmd = &cobra.Command{
	Use:   "zg",
	Short: "Zettelgarden CLI tool",
	Long:  `A standalone CLI tool for Zettelgarden card and task operations.`,
}

func main() {
	if err := rootCmd.Execute(); err != nil {
		fmt.Fprintln(os.Stderr, err)
		os.Exit(1)
	}
}
```

**Step 4: Test basic CLI builds**

Run:
```bash
go build -o zg ./cmd/zg
./zg --help
```

Expected output: Shows help with "A standalone CLI tool for Zettelgarden..."

**Step 5: Commit**

```bash
git add zg/
git commit -m "feat(zg-cli): initialize project with Cobra framework"
```

---

### Task 2: Implement config file loading

**Files:**
- Create: `zg/internal/config/config.go`
- Create: `zg/internal/config/config.go_test.go`

**Step 1: Write failing test for config loading**

Create: `zg/internal/config/config.go_test.go`
```go
package config

import (
	"os"
	"path/filepath"
	"testing"
)

func TestLoadConfig(t *testing.T) {
	// Create temp config dir
	tmpDir := t.TempDir()
	configPath := filepath.Join(tmpDir, "config.json")

	// Write test config
	configContent := `{"api_url": "http://test.local:8080", "token": "test-token"}`
	if err := os.WriteFile(configPath, []byte(configContent), 0644); err != nil {
		t.Fatal(err)
	}

	// Load config
	cfg, err := LoadConfig(configPath)
	if err != nil {
		t.Fatalf("LoadConfig failed: %v", err)
	}

	// Verify values
	if cfg.APIURL != "http://test.local:8080" {
		t.Errorf("Expected APIURL 'http://test.local:8080', got '%s'", cfg.APIURL)
	}
	if cfg.Token != "test-token" {
		t.Errorf("Expected Token 'test-token', got '%s'", cfg.Token)
	}
}

func TestLoadConfigDefaults(t *testing.T) {
	tmpDir := t.TempDir()
	configPath := filepath.Join(tmpDir, "config.json")

	// Write minimal config
	configContent := `{"token": "test-token"}`
	if err := os.WriteFile(configPath, []byte(configContent), 0644); err != nil {
		t.Fatal(err)
	}

	cfg, err := LoadConfig(configPath)
	if err != nil {
		t.Fatalf("LoadConfig failed: %v", err)
	}

	// Verify defaults
	if cfg.APIURL != "http://localhost:8080" {
		t.Errorf("Expected default APIURL 'http://localhost:8080', got '%s'", cfg.APIURL)
	}
	if cfg.TimeoutSeconds != 30 {
		t.Errorf("Expected default TimeoutSeconds 30, got %d", cfg.TimeoutSeconds)
	}
}
```

**Step 2: Run test to verify it fails**

Run:
```bash
cd /home/nick/code/Zettelgarden/zg
go test ./internal/config/...
```

Expected: FAIL with "undefined: LoadConfig"

**Step 3: Implement config loading**

Create: `zg/internal/config/config.go`
```go
package config

import (
	"encoding/json"
	"os"
	"path/filepath"
)

const (
	DefaultAPIURL      = "http://localhost:8080"
	DefaultTimeoutSecs = 30
)

type Config struct {
	APIURL         string `json:"api_url"`
	Token          string `json:"token"`
	TimeoutSeconds int    `json:"timeout_seconds"`
}

// GetDefaultConfigPath returns the default config file location
func GetDefaultConfigPath() (string, error) {
	homeDir, err := os.UserHomeDir()
	if err != nil {
		return "", err
	}
	return filepath.Join(homeDir, ".config", "zettelgarden", "config.json"), nil
}

// LoadConfig loads configuration from the specified path
func LoadConfig(path string) (*Config, error) {
	data, err := os.ReadFile(path)
	if err != nil {
		return nil, err
	}

	var cfg Config
	if err := json.Unmarshal(data, &cfg); err != nil {
		return nil, err
	}

	// Set defaults
	if cfg.APIURL == "" {
		cfg.APIURL = DefaultAPIURL
	}
	if cfg.TimeoutSeconds == 0 {
		cfg.TimeoutSeconds = DefaultTimeoutSecs
	}

	return &cfg, nil
}
```

**Step 4: Run tests to verify they pass**

Run:
```bash
go test ./internal/config/... -v
```

Expected: PASS

**Step 5: Commit**

```bash
git add zg/internal/config/
git commit -m "feat(zg-cli): add config file loading with defaults"
```

---

### Task 3: Implement API client

**Files:**
- Create: `zg/internal/api/client.go`
- Create: `zg/internal/api/client_test.go`

**Step 1: Write failing test for API request**

Create: `zg/internal/api/client_test.go`
```go
package api

import (
	"net/http"
	"net/http/httptest"
	"testing"
)

func TestClient_Get(t *testing.T) {
	// Mock server
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		// Verify auth header
		if r.Header.Get("Authorization") != "Bearer test-token" {
			t.Errorf("Expected 'Bearer test-token', got '%s'", r.Header.Get("Authorization"))
		}
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusOK)
		w.Write([]byte(`{"id": 1, "title": "Test Card"}`))
	}))
	defer server.Close()

	client := NewClient(server.URL, "test-token", 30)
	resp, err := client.Get("/api/user/cards/1")
	if err != nil {
		t.Fatalf("Get failed: %v", err)
	}

	if resp.StatusCode != http.StatusOK {
		t.Errorf("Expected status 200, got %d", resp.StatusCode)
	}
}

func TestClient_Post(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Method != "POST" {
			t.Errorf("Expected POST, got %s", r.Method)
		}
		if r.Header.Get("Authorization") != "Bearer test-token" {
			t.Errorf("Missing auth header")
		}
		w.WriteHeader(http.StatusCreated)
		w.Write([]byte(`{"id": 123}`))
	}))
	defer server.Close()

	client := NewClient(server.URL, "test-token", 30)
	resp, err := client.Post("/api/user/cards", []byte(`{"title":"New"}`))
	if err != nil {
		t.Fatalf("Post failed: %v", err)
	}

	if resp.StatusCode != http.StatusCreated {
		t.Errorf("Expected status 201, got %d", resp.StatusCode)
	}
}
```

**Step 2: Run test to verify it fails**

Run:
```bash
go test ./internal/api/...
```

Expected: FAIL with "undefined: NewClient"

**Step 3: Implement API client**

Create: `zg/internal/api/client.go`
```go
package api

import (
	"bytes"
	"fmt"
	"io"
	"net/http"
	"time"
)

type Client struct {
	baseURL    string
	token      string
	httpClient *http.Client
}

func NewClient(baseURL, token string, timeoutSecs int) *Client {
	return &Client{
		baseURL: baseURL,
		token:   token,
		httpClient: &http.Client{
			Timeout: time.Duration(timeoutSecs) * time.Second,
		},
	}
}

func (c *Client) buildURL(path string) string {
	return c.baseURL + path
}

func (c *Client) addAuth(req *http.Request) {
	req.Header.Set("Authorization", "Bearer "+c.token)
	req.Header.Set("Content-Type", "application/json")
}

func (c *Client) Get(path string) (*http.Response, error) {
	url := c.buildURL(path)
	req, err := http.NewRequest("GET", url, nil)
	if err != nil {
		return nil, err
	}

	c.addAuth(req)
	return c.httpClient.Do(req)
}

func (c *Client) Post(path string, body []byte) (*http.Response, error) {
	url := c.buildURL(path)
	req, err := http.NewRequest("POST", url, bytes.NewReader(body))
	if err != nil {
		return nil, err
	}

	c.addAuth(req)
	return c.httpClient.Do(req)
}

func (c *Client) Put(path string, body []byte) (*http.Response, error) {
	url := c.buildURL(path)
	req, err := http.NewRequest("PUT", url, bytes.NewReader(body))
	if err != nil {
		return nil, err
	}

	c.addAuth(req)
	return c.httpClient.Do(req)
}

func (c *Client) Delete(path string) (*http.Response, error) {
	url := c.buildURL(path)
	req, err := http.NewRequest("DELETE", url, nil)
	if err != nil {
		return nil, err
	}

	c.addAuth(req)
	return c.httpClient.Do(req)
}

// GetBodyBytes reads response body and closes it
func GetBodyBytes(resp *http.Response) ([]byte, error) {
	defer resp.Body.Close()
	return io.ReadAll(resp.Body)
}

// GetBodyString reads response body as string and closes it
func GetBodyString(resp *http.Response) (string, error) {
	body, err := GetBodyBytes(resp)
	if err != nil {
		return "", err
	}
	return string(body), nil
}
```

**Step 4: Run tests to verify they pass**

Run:
```bash
go test ./internal/api/... -v
```

Expected: PASS

**Step 5: Commit**

```bash
git add zg/internal/api/
git commit -m "feat(zg-cli): add HTTP API client with auth"
```

---

### Task 4: Implement JSON output writer

**Files:**
- Create: `zg/internal/output/writer.go`
- Create: `zg/internal/output/writer_test.go`

**Step 1: Write failing test for output formatting**

Create: `zg/internal/output/writer_test.go`
```go
package output

import (
	"bytes"
	"encoding/json"
	"testing"
)

func TestWriteSuccess(t *testing.T) {
	var buf bytes.Buffer
	data := map[string]any{"id": 123, "title": "Test"}

	WriteSuccess(&buf, data)

	result := buf.String()
	var parsed map[string]any
	if err := json.Unmarshal([]byte(result), &parsed); err != nil {
		t.Fatalf("Invalid JSON: %v", err)
	}

	if parsed["success"] != true {
		t.Errorf("Expected success=true, got %v", parsed["success"])
	}
	if parsed["data"] == nil {
		t.Error("Expected data field")
	}
}

func TestWriteError(t *testing.T) {
	var buf bytes.Buffer

	WriteError(&buf, "Something went wrong", "optional details")

	result := buf.String()
	var parsed map[string]any
	if err := json.Unmarshal([]byte(result), &parsed); err != nil {
		t.Fatalf("Invalid JSON: %v", err)
	}

	if parsed["success"] != false {
		t.Errorf("Expected success=false, got %v", parsed["success"])
	}
	if parsed["error"] != "Something went wrong" {
		t.Errorf("Expected error message, got %v", parsed["error"])
	}
}
```

**Step 2: Run test to verify it fails**

Run:
```bash
go test ./internal/output/...
```

Expected: FAIL with "undefined: WriteSuccess"

**Step 3: Implement output writer**

Create: `zg/internal/output/writer.go`
```go
package output

import (
	"encoding/json"
	"io"
	"os"
)

// Response is the standard API response structure
type Response struct {
	Success bool        `json:"success"`
	Data    any         `json:"data,omitempty"`
	Error   string      `json:"error,omitempty"`
	Details string      `json:"details,omitempty"`
	Total   int         `json:"total,omitempty"`
	Limit   int         `json:"limit,omitempty"`
	Offset  int         `json:"offset,omitempty"`
}

// WriteSuccess writes a success response with data
func WriteSuccess(w io.Writer, data any) error {
	return json.NewEncoder(w).Encode(Response{
		Success: true,
		Data:    data,
	})
}

// WriteError writes an error response
func WriteError(w io.Writer, errMsg string, details string) error {
	return json.NewEncoder(w).Encode(Response{
		Success: false,
		Error:   errMsg,
		Details: details,
	})
}

// WriteList writes a list response with pagination
func WriteList(w io.Writer, items any, total, limit, offset int) error {
	return json.NewEncoder(w).Encode(Response{
		Success: true,
		Data:    items,
		Total:   total,
		Limit:   limit,
		Offset:  offset,
	})
}

// IsTTY returns true if output is a terminal (for pretty-printing)
func IsTTY() bool {
 fileInfo, _ := os.Stdout.Stat()
	return (fileInfo.Mode() & os.ModeCharDevice) != 0
}
```

**Step 4: Run tests to verify they pass**

Run:
```bash
go test ./internal/output/... -v
```

Expected: PASS

**Step 5: Commit**

```bash
git add zg/internal/output/
git commit -m "feat(zg-cli): add JSON output writer"
```

---

### Task 5: Implement card get command

**Files:**
- Create: `zg/internal/cmd/card.go`
- Modify: `zg/cmd/zg/main.go`

**Step 1: Define card data models**

Create: `zg/internal/cmd/card.go`
```go
package cmd

import (
	"encoding/json"
	"fmt"
	"io"
	"os"
	"strconv"

	"github.com/spf13/cobra"
	"zg/internal/api"
	"zg/internal/config"
	"zg/internal/output"
)

// Card represents a Zettelgarden card
type Card struct {
	ID        int    `json:"id"`
	CardID    string `json:"card_id"`
	Title     string `json:"title"`
	Body      string `json:"body"`
	Link      string `json:"link"`
	CreatedAt string `json:"created_at"`
	UpdatedAt string `json:"updated_at"`
}

var cardCmd = &cobra.Command{
	Use:   "card",
	Short: "Manage cards",
}

var cardGetCmd = &cobra.Command{
	Use:   "get <id>",
	Short: "Get a card by ID",
	Args:  cobra.ExactArgs(1),
	RunE:  runCardGet,
}

func init() {
	cardCmd.AddCommand(cardGetCmd)
}

func runCardGet(cmd *cobra.Command, args []string) error {
	cardID, err := strconv.Atoi(args[0])
	if err != nil {
		return output.WriteError(os.Stdout, "Invalid card ID", "ID must be a number")
	}

	cfg, err := loadConfig()
	if err != nil {
		return output.WriteError(os.Stdout, "Config error", err.Error())
	}

	client := api.NewClient(cfg.APIURL, cfg.Token, cfg.TimeoutSeconds)
	resp, err := client.Get(fmt.Sprintf("/api/user/cards/%d", cardID))
	if err != nil {
		return output.WriteError(os.Stdout, "API request failed", err.Error())
	}

	body, err := api.GetBodyBytes(resp)
	if err != nil {
		return output.WriteError(os.Stdout, "Reading response failed", err.Error())
	}

	if resp.StatusCode != 200 {
		return output.WriteError(os.Stdout, fmt.Sprintf("API error: %d", resp.StatusCode), string(body))
	}

	var card Card
	if err := json.Unmarshal(body, &card); err != nil {
		return output.WriteError(os.Stdout, "Parse error", err.Error())
	}

	return output.WriteSuccess(os.Stdout, card)
}

func loadConfig() (*config.Config, error) {
	configPath, err := config.GetDefaultConfigPath()
	if err != nil {
		return nil, err
	}
	return config.LoadConfig(configPath)
}
```

**Step 2: Register card command in main.go**

Modify: `zg/cmd/zg/main.go`
```go
package main

import (
	"fmt"
	"os"

	"github.com/spf13/cobra"
	"zg/internal/cmd"
)

var rootCmd = &cobra.Command{
	Use:   "zg",
	Short: "Zettelgarden CLI tool",
	Long:  `A standalone CLI tool for Zettelgarden card and task operations.`,
}

func main() {
	// Add card commands
	rootCmd.AddCommand(cmd.GetCardCmd())

	if err := rootCmd.Execute(); err != nil {
		fmt.Fprintln(os.Stderr, err)
		os.Exit(1)
	}
}
```

**Step 3: Export card command from cmd package**

Modify: `zg/internal/cmd/card.go` - add at end:
```go
// GetCardCmd returns the card command for registration
func GetCardCmd() *cobra.Command {
	return cardCmd
}
```

**Step 4: Test the command**

Run:
```bash
cd /home/nick/code/Zettelgarden/zg
go build -o zg ./cmd/zg
./zg card get 1
```

Expected: JSON output with error about config/auth (since no real backend)

**Step 5: Commit**

```bash
git add zg/
git commit -m "feat(zg-cli): add card get command"
```

---

### Task 6: Implement card list command

**Files:**
- Modify: `zg/internal/cmd/card.go`

**Step 1: Add card list command**

Modify: `zg/internal/cmd/card.go` - add after cardGetCmd:
```go
var cardListCmd = &cobra.Command{
	Use:   "list",
	Short: "List cards",
	RunE:  runCardList,
}

var (
	listLimit  int
	listOffset int
	listStarred bool
)

func init() {
	cardListCmd.Flags().IntVarP(&listLimit, "limit", "l", 20, "Limit results")
	cardListCmd.Flags().IntVarP(&listOffset, "offset", "o", 0, "Offset results")
	cardListCmd.Flags().BoolVar(&listStarred, "starred", false, "Show only starred cards")
	cardCmd.AddCommand(cardListCmd)
}

func runCardList(cmd *cobra.Command, args []string) error {
	cfg, err := loadConfig()
	if err != nil {
		return output.WriteError(os.Stdout, "Config error", err.Error())
	}

	client := api.NewClient(cfg.APIURL, cfg.Token, cfg.TimeoutSeconds)

	url := fmt.Sprintf("/api/user/cards?limit=%d&offset=%d", listLimit, listOffset)
	if listStarred {
		url += "&starred=true"
	}

	resp, err := client.Get(url)
	if err != nil {
		return output.WriteError(os.Stdout, "API request failed", err.Error())
	}

	body, err := api.GetBodyBytes(resp)
	if err != nil {
		return output.WriteError(os.Stdout, "Reading response failed", err.Error())
	}

	if resp.StatusCode != 200 {
		return output.WriteError(os.Stdout, fmt.Sprintf("API error: %d", resp.StatusCode), string(body))
	}

	// Parse paginated response
	var result struct {
		Cards []Card `json:"cards"`
		Total int    `json:"total"`
		Limit int    `json:"limit"`
		Offset int   `json:"offset"`
	}

	if err := json.Unmarshal(body, &result); err != nil {
		// Try direct array parse
		var cards []Card
		if err2 := json.Unmarshal(body, &cards); err2 == nil {
			return output.WriteSuccess(os.Stdout, cards)
		}
		return output.WriteError(os.Stdout, "Parse error", err.Error())
	}

	return output.WriteList(os.Stdout, result.Cards, result.Total, result.Limit, result.Offset)
}
```

**Step 2: Test card list**

Run:
```bash
go build -o zg ./cmd/zg
./zg card list
./zg card list --limit 5 --starred
```

**Step 3: Commit**

```bash
git add zg/
git commit -m "feat(zg-cli): add card list command with filters"
```

---

### Task 7: Implement card create command

**Files:**
- Modify: `zg/internal/cmd/card.go`

**Step 1: Add card create command**

Modify: `zg/internal/cmd/card.go` - add after cardListCmd init:
```go
var cardCreateCmd = &cobra.Command{
	Use:   "create",
	Short: "Create a new card",
	RunE:  runCardCreate,
}

var (
	createTitle string
	createBody  string
	createLink  string
)

func init() {
	cardCreateCmd.Flags().StringVarP(&createTitle, "title", "t", "", "Card title (required)")
	cardCreateCmd.Flags().StringVarP(&createBody, "body", "b", "", "Card body content")
	cardCreateCmd.Flags().StringVarP(&createLink, "link", "l", "", "URL link")
	cardCreateCmd.MarkFlagRequired("title")
	cardCmd.AddCommand(cardCreateCmd)
}

func runCardCreate(cmd *cobra.Command, args []string) error {
	cfg, err := loadConfig()
	if err != nil {
		return output.WriteError(os.Stdout, "Config error", err.Error())
	}

	requestBody := map[string]any{
		"title": createTitle,
	}
	if createBody != "" {
		requestBody["body"] = createBody
	}
	if createLink != "" {
		requestBody["link"] = createLink
	}

	bodyBytes, err := json.Marshal(requestBody)
	if err != nil {
		return output.WriteError(os.Stdout, "JSON encode error", err.Error())
	}

	client := api.NewClient(cfg.APIURL, cfg.Token, cfg.TimeoutSeconds)
	resp, err := client.Post("/api/user/cards", bodyBytes)
	if err != nil {
		return output.WriteError(os.Stdout, "API request failed", err.Error())
	}

	respBody, err := api.GetBodyBytes(resp)
	if err != nil {
		return output.WriteError(os.Stdout, "Reading response failed", err.Error())
	}

	if resp.StatusCode != 201 {
		return output.WriteError(os.Stdout, fmt.Sprintf("API error: %d", resp.StatusCode), string(respBody))
	}

	var card Card
	if err := json.Unmarshal(respBody, &card); err != nil {
		return output.WriteError(os.Stdout, "Parse error", err.Error())
	}

	return output.WriteSuccess(os.Stdout, card)
}
```

**Step 2: Test card create**

Run:
```bash
go build -o zg ./cmd/zg
./zg card create --title "Test Card" --body "Test body"
```

**Step 3: Commit**

```bash
git add zg/
git commit -m "feat(zg-cli): add card create command"
```

---

### Task 8: Implement card update command

**Files:**
- Modify: `zg/internal/cmd/card.go`

**Step 1: Add card update command**

Modify: `zg/internal/cmd/card.go`:
```go
var cardUpdateCmd = &cobra.Command{
	Use:   "update <id>",
	Short: "Update a card",
	Args:  cobra.ExactArgs(1),
	RunE:  runCardUpdate,
}

var (
	updateTitle  string
	updateBody   string
	updateLink   string
)

func init() {
	cardUpdateCmd.Flags().StringVarP(&updateTitle, "title", "t", "", "New title")
	cardUpdateCmd.Flags().StringVarP(&updateBody, "body", "b", "", "New body")
	cardUpdateCmd.Flags().StringVarP(&updateLink, "link", "l", "", "New link")
	cardCmd.AddCommand(cardUpdateCmd)
}

func runCardUpdate(cmd *cobra.Command, args []string) error {
	cardID, err := strconv.Atoi(args[0])
	if err != nil {
		return output.WriteError(os.Stdout, "Invalid card ID", "ID must be a number")
	}

	cfg, err := loadConfig()
	if err != nil {
		return output.WriteError(os.Stdout, "Config error", err.Error())
	}

	requestBody := map[string]any{}
	if updateTitle != "" {
		requestBody["title"] = updateTitle
	}
	if updateBody != "" {
		requestBody["body"] = updateBody
	}
	if updateLink != "" {
		requestBody["link"] = updateLink
	}

	if len(requestBody) == 0 {
		return output.WriteError(os.Stdout, "No updates", "Specify at least one field to update")
	}

	bodyBytes, err := json.Marshal(requestBody)
	if err != nil {
		return output.WriteError(os.Stdout, "JSON encode error", err.Error())
	}

	client := api.NewClient(cfg.APIURL, cfg.Token, cfg.TimeoutSeconds)
	resp, err := client.Put(fmt.Sprintf("/api/user/cards/%d", cardID), bodyBytes)
	if err != nil {
		return output.WriteError(os.Stdout, "API request failed", err.Error())
	}

	respBody, err := api.GetBodyBytes(resp)
	if err != nil {
		return output.WriteError(os.Stdout, "Reading response failed", err.Error())
	}

	if resp.StatusCode != 200 {
		return output.WriteError(os.Stdout, fmt.Sprintf("API error: %d", resp.StatusCode), string(respBody))
	}

	return output.WriteSuccess(os.Stdout, map[string]any{"message": "Card updated"})
}
```

**Step 2: Test card update**

Run:
```bash
go build -o zg ./cmd/zg
./zg card update 1 --title "Updated Title"
```

**Step 3: Commit**

```bash
git add zg/
git commit -m "feat(zg-cli): add card update command"
```

---

### Task 9: Implement card delete command

**Files:**
- Modify: `zg/internal/cmd/card.go`

**Step 1: Add card delete command**

Modify: `zg/internal/cmd/card.go`:
```go
var cardDeleteCmd = &cobra.Command{
	Use:   "delete <id>",
	Short: "Delete a card",
	Args:  cobra.ExactArgs(1),
	RunE:  runCardDelete,
}

func init() {
	cardCmd.AddCommand(cardDeleteCmd)
}

func runCardDelete(cmd *cobra.Command, args []string) error {
	cardID, err := strconv.Atoi(args[0])
	if err != nil {
		return output.WriteError(os.Stdout, "Invalid card ID", "ID must be a number")
	}

	cfg, err := loadConfig()
	if err != nil {
		return output.WriteError(os.Stdout, "Config error", err.Error())
	}

	client := api.NewClient(cfg.APIURL, cfg.Token, cfg.TimeoutSeconds)
	resp, err := client.Delete(fmt.Sprintf("/api/user/cards/%d", cardID))
	if err != nil {
		return output.WriteError(os.Stdout, "API request failed", err.Error())
	}

	if resp.StatusCode != 204 {
		body, _ := api.GetBodyString(resp)
		return output.WriteError(os.Stdout, fmt.Sprintf("API error: %d", resp.StatusCode), body)
	}

	return output.WriteSuccess(os.Stdout, map[string]any{"message": "Card deleted"})
}
```

**Step 2: Test card delete**

Run:
```bash
go build -o zg ./cmd/zg
./zg card delete 1
```

**Step 3: Commit**

```bash
git add zg/
git commit -m "feat(zg-cli): add card delete command"
```

---

### Task 10: Implement card search command

**Files:**
- Modify: `zg/internal/cmd/card.go`

**Step 1: Add card search command**

Modify: `zg/internal/cmd/card.go`:
```go
var cardSearchCmd = &cobra.Command{
	Use:   "search <query>",
	Short: "Search cards",
	Args:  cobra.ExactArgs(1),
	RunE:  runCardSearch,
}

var (
	searchFullText bool
	searchLimit    int
)

func init() {
	cardSearchCmd.Flags().BoolVarP(&searchFullText, "full-text", "f", false, "Search in body too")
	cardSearchCmd.Flags().IntVarP(&searchLimit, "limit", "l", 20, "Limit results")
	cardCmd.AddCommand(cardSearchCmd)
}

func runCardSearch(cmd *cobra.Command, args []string) error {
	cfg, err := loadConfig()
	if err != nil {
		return output.WriteError(os.Stdout, "Config error", err.Error())
	}

	query := args[0]
	url := fmt.Sprintf("/api/user/search?query=%s&limit=%d", query, searchLimit)
	if searchFullText {
		url += "&full_text=true"
	}

	client := api.NewClient(cfg.APIURL, cfg.Token, cfg.TimeoutSeconds)
	resp, err := client.Get(url)
	if err != nil {
		return output.WriteError(os.Stdout, "API request failed", err.Error())
	}

	body, err := api.GetBodyBytes(resp)
	if err != nil {
		return output.WriteError(os.Stdout, "Reading response failed", err.Error())
	}

	if resp.StatusCode != 200 {
		return output.WriteError(os.Stdout, fmt.Sprintf("API error: %d", resp.StatusCode), string(body))
	}

	var cards []Card
	if err := json.Unmarshal(body, &cards); err != nil {
		return output.WriteError(os.Stdout, "Parse error", err.Error())
	}

	return output.WriteSuccess(os.Stdout, cards)
}
```

**Step 2: Test card search**

Run:
```bash
go build -o zg ./cmd/zg
./zg card search "python"
./zg card search "test" --full-text --limit 5
```

**Step 3: Commit**

```bash
git add zg/
git commit -m "feat(zg-cli): add card search command"
```

---

### Task 11: Implement Task data models and get command

**Files:**
- Create: `zg/internal/cmd/task.go`
- Modify: `zg/cmd/zg/main.go`

**Step 1: Create task command file**

Create: `zg/internal/cmd/task.go`
```go
package cmd

import (
	"encoding/json"
	"fmt"
	"os"
	"strconv"

	"github.com/spf13/cobra"
	"zg/internal/api"
	"zg/internal/output"
)

// Task represents a Zettelgarden task
type Task struct {
	ID          int     `json:"id"`
	Title       string  `json:"title"`
	Description string  `json:"description"`
	IsComplete  bool    `json:"is_complete"`
	Priority    *string `json:"priority"`
	Status      *string `json:"status"`
	ScheduledAt *string `json:"scheduled_at"`
	CreatedAt   string  `json:"created_at"`
}

var taskCmd = &cobra.Command{
	Use:   "task",
	Short: "Manage tasks",
}

var taskGetCmd = &cobra.Command{
	Use:   "get <id>",
	Short: "Get a task by ID",
	Args:  cobra.ExactArgs(1),
	RunE:  runTaskGet,
}

func init() {
	taskCmd.AddCommand(taskGetCmd)
}

func runTaskGet(cmd *cobra.Command, args []string) error {
	taskID, err := strconv.Atoi(args[0])
	if err != nil {
		return output.WriteError(os.Stdout, "Invalid task ID", "ID must be a number")
	}

	cfg, err := loadConfig()
	if err != nil {
		return output.WriteError(os.Stdout, "Config error", err.Error())
	}

	client := api.NewClient(cfg.APIURL, cfg.Token, cfg.TimeoutSeconds)
	resp, err := client.Get(fmt.Sprintf("/api/user/tasks/%d", taskID))
	if err != nil {
		return output.WriteError(os.Stdout, "API request failed", err.Error())
	}

	body, err := api.GetBodyBytes(resp)
	if err != nil {
		return output.WriteError(os.Stdout, "Reading response failed", err.Error())
	}

	if resp.StatusCode != 200 {
		return output.WriteError(os.Stdout, fmt.Sprintf("API error: %d", resp.StatusCode), string(body))
	}

	var task Task
	if err := json.Unmarshal(body, &task); err != nil {
		return output.WriteError(os.Stdout, "Parse error", err.Error())
	}

	return output.WriteSuccess(os.Stdout, task)
}

// GetTaskCmd returns the task command for registration
func GetTaskCmd() *cobra.Command {
	return taskCmd
}
```

**Step 2: Register task command in main.go**

Modify: `zg/cmd/zg/main.go`:
```go
package main

import (
	"fmt"
	"os"

	"github.com/spf13/cobra"
	"zg/internal/cmd"
)

var rootCmd = &cobra.Command{
	Use:   "zg",
	Short: "Zettelgarden CLI tool",
	Long:  `A standalone CLI tool for Zettelgarden card and task operations.`,
}

func main() {
	// Add commands
	rootCmd.AddCommand(cmd.GetCardCmd())
	rootCmd.AddCommand(cmd.GetTaskCmd())

	if err := rootCmd.Execute(); err != nil {
		fmt.Fprintln(os.Stderr, err)
		os.Exit(1)
	}
}
```

**Step 3: Test task get**

Run:
```bash
go build -o zg ./cmd/zg
./zg task get 1
```

**Step 4: Commit**

```bash
git add zg/
git commit -m "feat(zg-cli): add task get command"
```

---

### Task 12: Implement task list command

**Files:**
- Modify: `zg/internal/cmd/task.go`

**Step 1: Add task list command**

Modify: `zg/internal/cmd/task.go` - add after taskGetCmd init:
```go
var taskListCmd = &cobra.Command{
	Use:   "list",
	Short: "List tasks",
	RunE:  runTaskList,
}

var (
	taskListLimit       int
	taskListCompleted   *bool
	taskListPriority    string
	taskListScheduled   string
	taskListStatus      string
)

func init() {
	taskListCmd.Flags().IntVarP(&taskListLimit, "limit", "l", 50, "Limit results")

	// Use a pointer for the flag to detect if it was set
	taskListCompletedVal := false
	taskListCompleted = &taskListCompletedVal
	taskListCmd.Flags().BoolVar(taskListCompleted, "completed", false, "Show only completed tasks")
	taskListCmd.Flags().BoolVar(taskListCompleted, "incomplete", false, "Show only incomplete tasks")

	taskListCmd.Flags().StringVarP(&taskListPriority, "priority", "p", "", "Filter by priority (high/medium/low)")
	taskListCmd.Flags().StringVar(&taskListScheduled, "scheduled-date", "", "Filter by scheduled date (YYYY-MM-DD)")
	taskListCmd.Flags().StringVar(&taskListStatus, "status", "", "Filter by status")

	taskCmd.AddCommand(taskListCmd)
}

func runTaskList(cmd *cobra.Command, args []string) error {
	cfg, err := loadConfig()
	if err != nil {
		return output.WriteError(os.Stdout, "Config error", err.Error())
	}

	url := fmt.Sprintf("/api/user/tasks?limit=%d", taskListLimit)

	if cmd.Flags().Changed("completed") || cmd.Flags().Changed("incomplete") {
		url += "&completed=" + strconv.FormatBool(*taskListCompleted)
	}

	if taskListPriority != "" {
		url += "&priority=" + taskListPriority
	}
	if taskListScheduled != "" {
		url += "&scheduled_date=" + taskListScheduled
	}
	if taskListStatus != "" {
		url += "&status=" + taskListStatus
	}

	client := api.NewClient(cfg.APIURL, cfg.Token, cfg.TimeoutSeconds)
	resp, err := client.Get(url)
	if err != nil {
		return output.WriteError(os.Stdout, "API request failed", err.Error())
	}

	body, err := api.GetBodyBytes(resp)
	if err != nil {
		return output.WriteError(os.Stdout, "Reading response failed", err.Error())
	}

	if resp.StatusCode != 200 {
		return output.WriteError(os.Stdout, fmt.Sprintf("API error: %d", resp.StatusCode), string(body))
	}

	// Parse paginated response
	var result struct {
		Tasks  []Task `json:"tasks"`
		Total  int    `json:"total"`
		Limit  int    `json:"limit"`
		Offset int    `json:"offset"`
	}

	if err := json.Unmarshal(body, &result); err != nil {
		// Try direct array parse
		var tasks []Task
		if err2 := json.Unmarshal(body, &tasks); err2 == nil {
			return output.WriteSuccess(os.Stdout, tasks)
		}
		return output.WriteError(os.Stdout, "Parse error", err.Error())
	}

	return output.WriteList(os.Stdout, result.Tasks, result.Total, result.Limit, result.Offset)
}
```

**Step 2: Test task list**

Run:
```bash
go build -o zg ./cmd/zg
./zg task list
./zg task list --completed=false --priority=high
```

**Step 3: Commit**

```bash
git add zg/
git commit -m "feat(zg-cli): add task list command with filters"
```

---

### Task 13: Implement task create command

**Files:**
- Modify: `zg/internal/cmd/task.go`

**Step 1: Add task create command**

Modify: `zg/internal/cmd/task.go` - add after taskListCmd init:
```go
var taskCreateCmd = &cobra.Command{
	Use:   "create",
	Short: "Create a new task",
	RunE:  runTaskCreate,
}

var (
	taskCreateTitle       string
	taskCreateDescription string
	taskCreateScheduled   string
	taskCreatePriority    string
)

func init() {
	taskCreateCmd.Flags().StringVarP(&taskCreateTitle, "title", "t", "", "Task title (required)")
	taskCreateCmd.Flags().StringVarP(&taskCreateDescription, "description", "d", "", "Task description")
	taskCreateCmd.Flags().StringVar(&taskCreateScheduled, "scheduled-date", "", "Scheduled date (YYYY-MM-DD or 'today')")
	taskCreateCmd.Flags().StringVarP(&taskCreatePriority, "priority", "p", "", "Priority (high/medium/low)")
	taskCreateCmd.MarkFlagRequired("title")
	taskCmd.AddCommand(taskCreateCmd)
}

func runTaskCreate(cmd *cobra.Command, args []string) error {
	cfg, err := loadConfig()
	if err != nil {
		return output.WriteError(os.Stdout, "Config error", err.Error())
	}

	requestBody := map[string]any{
		"title": taskCreateTitle,
	}
	if taskCreateDescription != "" {
		requestBody["description"] = taskCreateDescription
	}
	if taskCreateScheduled != "" {
		requestBody["scheduled_date"] = taskCreateScheduled
	}
	if taskCreatePriority != "" {
		requestBody["priority"] = taskCreatePriority
	}

	bodyBytes, err := json.Marshal(requestBody)
	if err != nil {
		return output.WriteError(os.Stdout, "JSON encode error", err.Error())
	}

	client := api.NewClient(cfg.APIURL, cfg.Token, cfg.TimeoutSeconds)
	resp, err := client.Post("/api/user/tasks", bodyBytes)
	if err != nil {
		return output.WriteError(os.Stdout, "API request failed", err.Error())
	}

	respBody, err := api.GetBodyBytes(resp)
	if err != nil {
		return output.WriteError(os.Stdout, "Reading response failed", err.Error())
	}

	if resp.StatusCode != 201 {
		return output.WriteError(os.Stdout, fmt.Sprintf("API error: %d", resp.StatusCode), string(respBody))
	}

	var result map[string]any
	if err := json.Unmarshal(respBody, &result); err != nil {
		return output.WriteError(os.Stdout, "Parse error", err.Error())
	}

	return output.WriteSuccess(os.Stdout, result)
}
```

**Step 2: Test task create**

Run:
```bash
go build -o zg ./cmd/zg
./zg task create --title "Test Task" --priority high
```

**Step 3: Commit**

```bash
git add zg/
git commit -m "feat(zg-cli): add task create command"
```

---

### Task 14: Implement task update command

**Files:**
- Modify: `zg/internal/cmd/task.go`

**Step 1: Add task update command**

Modify: `zg/internal/cmd/task.go`:
```go
var taskUpdateCmd = &cobra.Command{
	Use:   "update <id>",
	Short: "Update a task",
	Args:  cobra.ExactArgs(1),
	RunE:  runTaskUpdate,
}

var (
	taskUpdateTitle       string
	taskUpdateDescription string
	taskUpdateIsComplete  *bool
	taskUpdatePriority    string
	taskUpdateScheduled   string
	taskUpdateStatus      string
)

func init() {
	taskUpdateCmd.Flags().StringVarP(&taskUpdateTitle, "title", "t", "", "New title")
	taskUpdateCmd.Flags().StringVarP(&taskUpdateDescription, "description", "d", "", "New description")
	taskUpdateCmd.Flags().BoolVar(&taskUpdateIsComplete, "complete", false, "Mark as complete")
	taskUpdateCmd.Flags().BoolVar(&taskUpdateIsComplete, "incomplete", false, "Mark as incomplete")
	taskUpdateCmd.Flags().StringVarP(&taskUpdatePriority, "priority", "p", "", "New priority")
	taskUpdateCmd.Flags().StringVar(&taskUpdateScheduled, "scheduled-date", "", "New scheduled date")
	taskUpdateCmd.Flags().StringVar(&taskUpdateStatus, "status", "", "New status")
	taskCmd.AddCommand(taskUpdateCmd)
}

func runTaskUpdate(cmd *cobra.Command, args []string) error {
	taskID, err := strconv.Atoi(args[0])
	if err != nil {
		return output.WriteError(os.Stdout, "Invalid task ID", "ID must be a number")
	}

	cfg, err := loadConfig()
	if err != nil {
		return output.WriteError(os.Stdout, "Config error", err.Error())
	}

	requestBody := map[string]any{}
	if taskUpdateTitle != "" {
		requestBody["title"] = taskUpdateTitle
	}
	if taskUpdateDescription != "" {
		requestBody["description"] = taskUpdateDescription
	}
	if cmd.Flags().Changed("complete") || cmd.Flags().Changed("incomplete") {
		requestBody["is_complete"] = *taskUpdateIsComplete
	}
	if taskUpdatePriority != "" {
		requestBody["priority"] = taskUpdatePriority
	}
	if taskUpdateScheduled != "" {
		requestBody["scheduled_date"] = taskUpdateScheduled
	}
	if taskUpdateStatus != "" {
		requestBody["status"] = taskUpdateStatus
	}

	if len(requestBody) == 0 {
		return output.WriteError(os.Stdout, "No updates", "Specify at least one field to update")
	}

	bodyBytes, err := json.Marshal(requestBody)
	if err != nil {
		return output.WriteError(os.Stdout, "JSON encode error", err.Error())
	}

	client := api.NewClient(cfg.APIURL, cfg.Token, cfg.TimeoutSeconds)
	resp, err := client.Put(fmt.Sprintf("/api/user/tasks/%d", taskID), bodyBytes)
	if err != nil {
		return output.WriteError(os.Stdout, "API request failed", err.Error())
	}

	respBody, err := api.GetBodyBytes(resp)
	if err != nil {
		return output.WriteError(os.Stdout, "Reading response failed", err.Error())
	}

	if resp.StatusCode != 200 {
		return output.WriteError(os.Stdout, fmt.Sprintf("API error: %d", resp.StatusCode), string(respBody))
	}

	return output.WriteSuccess(os.Stdout, map[string]any{"message": "Task updated"})
}
```

**Step 2: Test task update**

Run:
```bash
go build -o zg ./cmd/zg
./zg task update 1 --complete
./zg task update 1 --priority medium
```

**Step 3: Commit**

```bash
git add zg/
git commit -m "feat(zg-cli): add task update command"
```

---

### Task 15: Implement task delete and complete commands

**Files:**
- Modify: `zg/internal/cmd/task.go`

**Step 1: Add task delete command**

Modify: `zg/internal/cmd/task.go`:
```go
var taskDeleteCmd = &cobra.Command{
	Use:   "delete <id>",
	Short: "Delete a task",
	Args:  cobra.ExactArgs(1),
	RunE:  runTaskDelete,
}

var taskCompleteCmd = &cobra.Command{
	Use:   "complete <id>",
	Short: "Mark a task as complete",
	Args:  cobra.ExactArgs(1),
	RunE:  runTaskComplete,
}

func init() {
	taskCmd.AddCommand(taskDeleteCmd)
	taskCmd.AddCommand(taskCompleteCmd)
}

func runTaskDelete(cmd *cobra.Command, args []string) error {
	taskID, err := strconv.Atoi(args[0])
	if err != nil {
		return output.WriteError(os.Stdout, "Invalid task ID", "ID must be a number")
	}

	cfg, err := loadConfig()
	if err != nil {
		return output.WriteError(os.Stdout, "Config error", err.Error())
	}

	client := api.NewClient(cfg.APIURL, cfg.Token, cfg.TimeoutSeconds)
	resp, err := client.Delete(fmt.Sprintf("/api/user/tasks/%d", taskID))
	if err != nil {
		return output.WriteError(os.Stdout, "API request failed", err.Error())
	}

	if resp.StatusCode != 204 {
		body, _ := api.GetBodyString(resp)
		return output.WriteError(os.Stdout, fmt.Sprintf("API error: %d", resp.StatusCode), body)
	}

	return output.WriteSuccess(os.Stdout, map[string]any{"message": "Task deleted"})
}

func runTaskComplete(cmd *cobra.Command, args []string) error {
	return runTaskUpdate(cmd, args)
}
```

**Step 2: Test task delete and complete**

Run:
```bash
go build -o zg ./cmd/zg
./zg task complete 1
./zg task delete 999
```

**Step 3: Commit**

```bash
git add zg/
git commit -m "feat(zg-cli): add task delete and complete commands"
```

---

### Task 16: Add global flags and pretty print support

**Files:**
- Modify: `zg/cmd/zg/main.go`
- Modify: `zg/internal/output/writer.go`

**Step 1: Add global flags**

Modify: `zg/cmd/zg/main.go`:
```go
package main

import (
	"fmt"
	"os"

	"github.com/spf13/cobra"
	"zg/internal/cmd"
	"zg/internal/output"
)

var (
	cfgFile  string
	apiURL   string
	apiToken string
	pretty   bool
)

var rootCmd = &cobra.Command{
	Use:   "zg",
	Short: "Zettelgarden CLI tool",
	Long:  `A standalone CLI tool for Zettelgarden card and task operations.`,
	PersistentPreRun: func(cmd *cobra.Command, args []string) {
		if pretty {
			output.SetPretty(true)
		}
	},
}

func init() {
	rootCmd.PersistentFlags().StringVar(&cfgFile, "config", "", "Config file path")
	rootCmd.PersistentFlags().StringVar(&apiURL, "url", "", "Override API URL")
	rootCmd.PersistentFlags().StringVar(&apiToken, "token", "", "Override auth token")
	rootCmd.PersistentFlags().BoolVar(&pretty, "pretty", false, "Pretty-print JSON output")
}

func main() {
	// Add commands
	rootCmd.AddCommand(cmd.GetCardCmd())
	rootCmd.AddCommand(cmd.GetTaskCmd())

	if err := rootCmd.Execute(); err != nil {
		fmt.Fprintln(os.Stderr, err)
		os.Exit(1)
	}
}
```

**Step 2: Add pretty print support**

Modify: `zg/internal/output/writer.go`:
```go
package output

import (
	"encoding/json"
	"io"
	"os"
	"sync"
)

var (
	prettyPrint bool
	prettyMutex sync.Mutex
)

// SetPretty sets whether to pretty-print JSON output
func SetPretty(v bool) {
	prettyMutex.Lock()
	defer prettyMutex.Unlock()
	prettyPrint = v
}

// isPretty returns the current pretty-print setting
func isPretty() bool {
	prettyMutex.Lock()
	defer prettyMutex.Unlock()
	return prettyPrint
}

// Response is the standard API response structure
type Response struct {
	Success bool        `json:"success"`
	Data    any         `json:"data,omitempty"`
	Error   string      `json:"error,omitempty"`
	Details string      `json:"details,omitempty"`
	Total   int         `json:"total,omitempty"`
	Limit   int         `json:"limit,omitempty"`
	Offset  int         `json:"offset,omitempty"`
}

// encodeJSON encodes the response with or without indentation
func encodeJSON(w io.Writer, v any) error {
	if isPretty() {
		encoder := json.NewEncoder(w)
		encoder.SetIndent("", "  ")
		return encoder.Encode(v)
	}
	return json.NewEncoder(w).Encode(v)
}

// WriteSuccess writes a success response with data
func WriteSuccess(w io.Writer, data any) error {
	return encodeJSON(w, Response{
		Success: true,
		Data:    data,
	})
}

// WriteError writes an error response
func WriteError(w io.Writer, errMsg string, details string) error {
	return encodeJSON(w, Response{
		Success: false,
		Error:   errMsg,
		Details: details,
	})
}

// WriteList writes a list response with pagination
func WriteList(w io.Writer, items any, total, limit, offset int) error {
	return encodeJSON(w, Response{
		Success: true,
		Data:    items,
		Total:   total,
		Limit:   limit,
		Offset:  offset,
	})
}

// IsTTY returns true if output is a terminal (for auto pretty-print in future)
func IsTTY() bool {
	fileInfo, _ := os.Stdout.Stat()
	return (fileInfo.Mode() & os.ModeCharDevice) != 0
}
```

**Step 3: Test global flags**

Run:
```bash
go build -o zg ./cmd/zg
./zg card list --pretty
```

**Step 4: Commit**

```bash
git add zg/
git commit -m "feat(zg-cli): add global flags and pretty print support"
```

---

### Task 17: Add README documentation

**Files:**
- Create: `zg/README.md`

**Step 1: Create README**

Create: `zg/README.md`
```markdown
# zg - Zettelgarden CLI

A standalone CLI tool for Zettelgarden card and task operations.

## Installation

```bash
go build -o zg ./cmd/zg
cp zg /usr/local/bin/
```

## Configuration

Create `~/.config/zettelgarden/config.json`:

```json
{
  "api_url": "http://localhost:8080",
  "token": "your-jwt-token"
}
```

Get your token from the Zettelgarden web UI.

## Usage

### Cards

```bash
zg card get <id>              # Get a card
zg card list [--limit N]      # List cards
zg card create -t "Title"     # Create card
zg card update <id> -t "New"  # Update card
zg card delete <id>           # Delete card
zg card search "query"        # Search cards
```

### Tasks

```bash
zg task get <id>              # Get a task
zg task list [--completed]    # List tasks
zg task create -t "Title"     # Create task
zg task update <id> --complete # Update task
zg task complete <id>         # Mark complete
zg task delete <id>           # Delete task
```

### Global Flags

- `--pretty` - Pretty-print JSON output
- `--config <path>` - Custom config path
- `--url <url>` - Override API URL
- `--token <token>` - Override auth token

## Output Format

All commands return JSON:

```json
{
  "success": true,
  "data": { /* result */ }
}
```

Errors:
```json
{
  "success": false,
  "error": "Error message"
}
```
```

**Step 2: Commit**

```bash
git add zg/README.md
git commit -m "docs(zg-cli): add README documentation"
```

---

### Task 18: Add .gitignore and final cleanup

**Files:**
- Create: `zg/.gitignore`

**Step 1: Create .gitignore**

Create: `zg/.gitignore`
```gitignore
# Binaries
zg
zg-*

# Go
*.exe
*.exe~
*.dll
*.so
*.dylib
*.test
*.out

# Config (for local testing)
config.json
```

**Step 2: Final build test**

Run:
```bash
cd /home/nick/code/Zettelgarden/zg
go mod tidy
go build -o zg ./cmd/zg
./zg --help
```

**Step 3: Commit**

```bash
git add zg/.gitignore
git commit -m "chore(zg-cli): add .gitignore"
```

---

## Summary

This implementation plan creates a complete CLI tool for Zettelgarden with:

1. **Card operations**: get, list, create, update, delete, search
2. **Task operations**: get, list, create, update, delete, complete
3. **JSON output** with structured responses for AI agent consumption
4. **Config file** support at `~/.config/zettelgarden/config.json`
5. **Global flags** for overriding config

Total: 18 bite-sized tasks following TDD principles.
