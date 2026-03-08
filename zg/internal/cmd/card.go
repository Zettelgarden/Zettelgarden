package cmd

import (
	"encoding/json"
	"fmt"
	"os"
	"strconv"

	"github.com/spf13/cobra"
	"github.com/nick-zettelgarden/zg/internal/api"
	"github.com/nick-zettelgarden/zg/internal/config"
	"github.com/nick-zettelgarden/zg/internal/output"
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

// Global flag values (set by main.go)
var (
	cfgFile  string
	apiURL   string
	apiToken string
)

// SetCfgFile sets the config file path from global flags
func SetCfgFile(v string) { cfgFile = v }

// SetAPIURL sets the API URL from global flags
func SetAPIURL(v string) { apiURL = v }

// SetAPIToken sets the API token from global flags
func SetAPIToken(v string) { apiToken = v }

// Get the flag values for loadConfig to use
func getCfgFile() string  { return cfgFile }
func getAPIURL() string   { return apiURL }
func getAPIToken() string { return apiToken }

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

var cardListCmd = &cobra.Command{
	Use:   "list",
	Short: "List cards",
	RunE:  runCardList,
}

var cardCreateCmd = &cobra.Command{
	Use:   "create",
	Short: "Create a new card",
	RunE:  runCardCreate,
}

var cardUpdateCmd = &cobra.Command{
	Use:   "update <id>",
	Short: "Update a card",
	Args:  cobra.ExactArgs(1),
	RunE:  runCardUpdate,
}

var cardDeleteCmd = &cobra.Command{
	Use:   "delete <id>",
	Short: "Delete a card",
	Args:  cobra.ExactArgs(1),
	RunE:  runCardDelete,
}

var cardSearchCmd = &cobra.Command{
	Use:   "search <query>",
	Short: "Search cards",
	Args:  cobra.ExactArgs(1),
	RunE:  runCardSearch,
}

var cardNextIDCmd = &cobra.Command{
	Use:   "next-id",
	Short: "Get the next root card ID",
	RunE:  runCardNextID,
}

var cardNextChildIDCmd = &cobra.Command{
	Use:   "next-child-id <parent-id>",
	Short: "Get the next child card ID for a parent card",
	Args:  cobra.ExactArgs(1),
	RunE:  runCardNextChildID,
}

// Structured data commands
var cardGetStructuredDataCmd = &cobra.Command{
	Use:   "get-structured-data <id>",
	Short: "Get structured data for a card",
	Args:  cobra.ExactArgs(1),
	RunE:  runCardGetStructuredData,
}

var cardSetStructuredDataCmd = &cobra.Command{
	Use:   "set-structured-data <id>",
	Short: "Set (replace) structured data for a card",
	Args:  cobra.ExactArgs(1),
	RunE:  runCardSetStructuredData,
}

var cardPatchStructuredDataCmd = &cobra.Command{
	Use:   "patch-structured-data <id>",
	Short: "Patch (merge) structured data for a card",
	Args:  cobra.ExactArgs(1),
	RunE:  runCardPatchStructuredData,
}

var cardClearStructuredDataCmd = &cobra.Command{
	Use:   "clear-structured-data <id>",
	Short: "Clear structured data from a card",
	Args:  cobra.ExactArgs(1),
	RunE:  runCardClearStructuredData,
}

var (
	listLimit   int
	listOffset  int
	listStarred bool

	createTitle string
	createBody  string
	createLink  string

	updateTitle string
	updateBody  string
	updateLink  string

	searchFullText bool
	searchLimit    int

	// Structured data flags
	structuredDataSchemaID int
	structuredDataJSON     string
)

func init() {
	cardCmd.AddCommand(cardGetCmd)
	cardListCmd.Flags().IntVarP(&listLimit, "limit", "l", 20, "Limit results")
	cardListCmd.Flags().IntVarP(&listOffset, "offset", "o", 0, "Offset results")
	cardListCmd.Flags().BoolVar(&listStarred, "starred", false, "Show only starred cards")
	cardCmd.AddCommand(cardListCmd)

	cardCreateCmd.Flags().StringVarP(&createTitle, "title", "t", "", "Card title (required)")
	cardCreateCmd.Flags().StringVarP(&createBody, "body", "b", "", "Card body content")
	cardCreateCmd.Flags().StringVarP(&createLink, "link", "l", "", "URL link")
	cardCreateCmd.MarkFlagRequired("title")
	cardCmd.AddCommand(cardCreateCmd)

	cardUpdateCmd.Flags().StringVarP(&updateTitle, "title", "t", "", "New title")
	cardUpdateCmd.Flags().StringVarP(&updateBody, "body", "b", "", "New body")
	cardUpdateCmd.Flags().StringVarP(&updateLink, "link", "l", "", "New link")
	cardCmd.AddCommand(cardUpdateCmd)

	cardCmd.AddCommand(cardDeleteCmd)

	cardSearchCmd.Flags().BoolVarP(&searchFullText, "full-text", "f", false, "Search in body too")
	cardSearchCmd.Flags().IntVarP(&searchLimit, "limit", "l", 20, "Limit results")
	cardCmd.AddCommand(cardSearchCmd)

	cardCmd.AddCommand(cardNextIDCmd)
	cardCmd.AddCommand(cardNextChildIDCmd)

	// Structured data commands
	cardCmd.AddCommand(cardGetStructuredDataCmd)

	cardSetStructuredDataCmd.Flags().IntVarP(&structuredDataSchemaID, "schema-id", "s", 0, "Schema ID (required)")
	cardSetStructuredDataCmd.Flags().StringVarP(&structuredDataJSON, "data", "d", "", "JSON structured data")
	cardSetStructuredDataCmd.MarkFlagRequired("schema-id")
	cardCmd.AddCommand(cardSetStructuredDataCmd)

	cardPatchStructuredDataCmd.Flags().StringVarP(&structuredDataJSON, "data", "d", "", "JSON structured data (required)")
	cardPatchStructuredDataCmd.MarkFlagRequired("data")
	cardCmd.AddCommand(cardPatchStructuredDataCmd)

	cardCmd.AddCommand(cardClearStructuredDataCmd)
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
	resp, err := client.Get(fmt.Sprintf("/api/cards/%d", cardID))
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

func runCardList(cmd *cobra.Command, args []string) error {
	cfg, err := loadConfig()
	if err != nil {
		return output.WriteError(os.Stdout, "Config error", err.Error())
	}

	client := api.NewClient(cfg.APIURL, cfg.Token, cfg.TimeoutSeconds)

	url := fmt.Sprintf("/api/cards?limit=%d&offset=%d", listLimit, listOffset)
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
		Cards  []Card `json:"cards"`
		Total  int    `json:"total"`
		Limit  int    `json:"limit"`
		Offset int    `json:"offset"`
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
	resp, err := client.Post("/api/cards", bodyBytes)
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
		return output.WriteError(os.Stdout, "No updates", "Specify at least one field")
	}

	bodyBytes, err := json.Marshal(requestBody)
	if err != nil {
		return output.WriteError(os.Stdout, "JSON encode error", err.Error())
	}

	client := api.NewClient(cfg.APIURL, cfg.Token, cfg.TimeoutSeconds)
	resp, err := client.Put(fmt.Sprintf("/api/cards/%d", cardID), bodyBytes)
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

	return output.WriteMessage(os.Stdout, "Card updated")
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
	resp, err := client.Delete(fmt.Sprintf("/api/cards/%d", cardID))
	if err != nil {
		return output.WriteError(os.Stdout, "API request failed", err.Error())
	}

	if resp.StatusCode != 204 {
		body, _ := api.GetBodyString(resp)
		return output.WriteError(os.Stdout, fmt.Sprintf("API error: %d", resp.StatusCode), body)
	}

	return output.WriteMessage(os.Stdout, "Card deleted")
}

func runCardSearch(cmd *cobra.Command, args []string) error {
	cfg, err := loadConfig()
	if err != nil {
		return output.WriteError(os.Stdout, "Config error", err.Error())
	}

	query := args[0]
	url := fmt.Sprintf("/api/search?query=%s&limit=%d", query, searchLimit)
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

func runCardNextID(cmd *cobra.Command, args []string) error {
	cfg, err := loadConfig()
	if err != nil {
		return output.WriteError(os.Stdout, "Config error", err.Error())
	}

	client := api.NewClient(cfg.APIURL, cfg.Token, cfg.TimeoutSeconds)
	resp, err := client.Get("/api/cards/next-root-id")
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

	var result struct {
		NextID string `json:"new_id"`
		Error  bool   `json:"error"`
	}
	if err := json.Unmarshal(body, &result); err != nil {
		return output.WriteError(os.Stdout, "Parse error", err.Error())
	}

	return output.WriteSuccess(os.Stdout, result)
}

func runCardNextChildID(cmd *cobra.Command, args []string) error {
	parentID, err := strconv.Atoi(args[0])
	if err != nil {
		return output.WriteError(os.Stdout, "Invalid parent ID", "ID must be a number")
	}

	cfg, err := loadConfig()
	if err != nil {
		return output.WriteError(os.Stdout, "Config error", err.Error())
	}

	client := api.NewClient(cfg.APIURL, cfg.Token, cfg.TimeoutSeconds)
	resp, err := client.Get(fmt.Sprintf("/api/cards/%d/next-child-id", parentID))
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

	var result struct {
		NextID string `json:"new_id"`
		Error  bool   `json:"error"`
	}
	if err := json.Unmarshal(body, &result); err != nil {
		return output.WriteError(os.Stdout, "Parse error", err.Error())
	}

	return output.WriteSuccess(os.Stdout, result)
}

// StructuredDataResponse represents the response for getting structured data
type StructuredDataResponse struct {
	SchemaID       int              `json:"schema_id,omitempty"`
	SchemaName     string           `json:"schema_name,omitempty"`
	SchemaSlug     string           `json:"schema_slug,omitempty"`
	StructuredData *json.RawMessage `json:"structured_data,omitempty"`
}

func runCardGetStructuredData(cmd *cobra.Command, args []string) error {
	cardID, err := strconv.Atoi(args[0])
	if err != nil {
		return output.WriteError(os.Stdout, "Invalid card ID", "ID must be a number")
	}

	cfg, err := loadConfig()
	if err != nil {
		return output.WriteError(os.Stdout, "Config error", err.Error())
	}

	client := api.NewClient(cfg.APIURL, cfg.Token, cfg.TimeoutSeconds)
	resp, err := client.Get(fmt.Sprintf("/api/cards/%d/structured-data", cardID))
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

	var result StructuredDataResponse
	if err := json.Unmarshal(body, &result); err != nil {
		return output.WriteError(os.Stdout, "Parse error", err.Error())
	}

	return output.WriteSuccess(os.Stdout, result)
}

func runCardSetStructuredData(cmd *cobra.Command, args []string) error {
	cardID, err := strconv.Atoi(args[0])
	if err != nil {
		return output.WriteError(os.Stdout, "Invalid card ID", "ID must be a number")
	}

	cfg, err := loadConfig()
	if err != nil {
		return output.WriteError(os.Stdout, "Config error", err.Error())
	}

	requestBody := map[string]any{
		"schema_id": structuredDataSchemaID,
	}

	// Parse and include structured data if provided
	if structuredDataJSON != "" {
		var data json.RawMessage
		if err := json.Unmarshal([]byte(structuredDataJSON), &data); err != nil {
			return output.WriteError(os.Stdout, "Invalid JSON", err.Error())
		}
		requestBody["structured_data"] = data
	}

	bodyBytes, err := json.Marshal(requestBody)
	if err != nil {
		return output.WriteError(os.Stdout, "JSON encode error", err.Error())
	}

	client := api.NewClient(cfg.APIURL, cfg.Token, cfg.TimeoutSeconds)
	resp, err := client.Put(fmt.Sprintf("/api/cards/%d/structured-data", cardID), bodyBytes)
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

	return output.WriteMessage(os.Stdout, "Structured data set successfully")
}

func runCardPatchStructuredData(cmd *cobra.Command, args []string) error {
	cardID, err := strconv.Atoi(args[0])
	if err != nil {
		return output.WriteError(os.Stdout, "Invalid card ID", "ID must be a number")
	}

	cfg, err := loadConfig()
	if err != nil {
		return output.WriteError(os.Stdout, "Config error", err.Error())
	}

	// Parse the JSON data
	var data json.RawMessage
	if err := json.Unmarshal([]byte(structuredDataJSON), &data); err != nil {
		return output.WriteError(os.Stdout, "Invalid JSON", err.Error())
	}

	requestBody := map[string]any{
		"structured_data": data,
	}

	bodyBytes, err := json.Marshal(requestBody)
	if err != nil {
		return output.WriteError(os.Stdout, "JSON encode error", err.Error())
	}

	client := api.NewClient(cfg.APIURL, cfg.Token, cfg.TimeoutSeconds)
	resp, err := client.Patch(fmt.Sprintf("/api/cards/%d/structured-data", cardID), bodyBytes)
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

	return output.WriteMessage(os.Stdout, "Structured data patched successfully")
}

func runCardClearStructuredData(cmd *cobra.Command, args []string) error {
	cardID, err := strconv.Atoi(args[0])
	if err != nil {
		return output.WriteError(os.Stdout, "Invalid card ID", "ID must be a number")
	}

	cfg, err := loadConfig()
	if err != nil {
		return output.WriteError(os.Stdout, "Config error", err.Error())
	}

	client := api.NewClient(cfg.APIURL, cfg.Token, cfg.TimeoutSeconds)
	resp, err := client.Delete(fmt.Sprintf("/api/cards/%d/structured-data", cardID))
	if err != nil {
		return output.WriteError(os.Stdout, "API request failed", err.Error())
	}

	if resp.StatusCode != 200 {
		body, _ := api.GetBodyString(resp)
		return output.WriteError(os.Stdout, fmt.Sprintf("API error: %d", resp.StatusCode), body)
	}

	return output.WriteMessage(os.Stdout, "Structured data cleared")
}

func loadConfig() (*config.Config, error) {
	var configPath string
	var err error

	if getCfgFile() != "" {
		configPath = getCfgFile()
	} else {
		configPath, err = config.GetDefaultConfigPath()
		if err != nil {
			return nil, err
		}
	}

	cfg, err := config.LoadConfig(configPath)
	if err != nil {
		return nil, err
	}

	// Apply command-line overrides
	if getAPIURL() != "" {
		cfg.APIURL = getAPIURL()
	}
	if getAPIToken() != "" {
		cfg.Token = getAPIToken()
	}

	return cfg, nil
}

// GetCardCmd returns the card command for registration
func GetCardCmd() *cobra.Command {
	return cardCmd
}
