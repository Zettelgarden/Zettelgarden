package cmd

import (
	"encoding/json"
	"fmt"
	"os"
	"strconv"
	"strings"

	"github.com/spf13/cobra"
	"github.com/nick-zettelgarden/zg/internal/api"
	"github.com/nick-zettelgarden/zg/internal/output"
)

// Fact represents a Zettelgarden fact
type Fact struct {
	ID        int    `json:"id"`
	Fact      string `json:"fact"`
	CreatedAt string `json:"created_at"`
	UpdatedAt string `json:"updated_at"`
}

// FactWithCard represents a fact with its associated card
type FactWithCard struct {
	ID        int         `json:"id"`
	Fact      string      `json:"fact"`
	CreatedAt string      `json:"created_at"`
	UpdatedAt string      `json:"updated_at"`
	Card      PartialCard `json:"card"`
}

// FactWithCardAndScore represents a fact with card and similarity score
type FactWithCardAndScore struct {
	ID        int         `json:"id"`
	Fact      string      `json:"fact"`
	CreatedAt string      `json:"created_at"`
	UpdatedAt string      `json:"updated_at"`
	Card      PartialCard `json:"card"`
	Score     float64     `json:"score"`
}

// PartialCard represents minimal card info
type PartialCard struct {
	ID       int    `json:"id"`
	CardID   string `json:"card_id"`
	Title    string `json:"title"`
	ParentID *int   `json:"parent_id"`
}

var factCmd = &cobra.Command{
	Use:   "fact",
	Short: "Manage facts",
}

var factListCmd = &cobra.Command{
	Use:   "list",
	Short: "List facts",
	RunE:  runFactList,
}

var factGetCmd = &cobra.Command{
	Use:   "get <id>",
	Short: "Get a fact by ID",
	Args:  cobra.ExactArgs(1),
	RunE:  runFactGet,
}

var factUpdateCmd = &cobra.Command{
	Use:   "update <id>",
	Short: "Update a fact's text",
	Args:  cobra.ExactArgs(1),
	RunE:  runFactUpdate,
}

var factDeleteCmd = &cobra.Command{
	Use:   "delete <id>",
	Short: "Delete a fact",
	Args:  cobra.ExactArgs(1),
	RunE:  runFactDelete,
}

var factSimilarCmd = &cobra.Command{
	Use:   "similar <id>",
	Short: "Find facts similar to a given fact",
	Args:  cobra.ExactArgs(1),
	RunE:  runFactSimilar,
}

var factLinkCmd = &cobra.Command{
	Use:   "link <fact-id> <card-id>",
	Short: "Link a fact to a card",
	Args:  cobra.ExactArgs(2),
	RunE:  runFactLink,
}

var factMergeCmd = &cobra.Command{
	Use:   "merge <fact1-id> <fact2-id>",
	Short: "Merge fact2 into fact1 (fact2 is deleted)",
	Args:  cobra.ExactArgs(2),
	RunE:  runFactMerge,
}

var (
	factListSearch string
	factListLimit  int
	factListPage   int
	factListFull   bool

	factText string

	factSimilarLimit int
	factSimilarFull  bool
)

func init() {
	factListCmd.Flags().StringVarP(&factListSearch, "search", "s", "", "Search filter for fact text or card title")
	factListCmd.Flags().IntVarP(&factListLimit, "limit", "l", 20, "Results per page (max 100)")
	factListCmd.Flags().IntVarP(&factListPage, "page", "p", 1, "Page number")
	factListCmd.Flags().BoolVar(&factListFull, "full", false, "Show full fact text (default: truncated to 300 chars)")
	factCmd.AddCommand(factListCmd)

	factCmd.AddCommand(factGetCmd)

	factUpdateCmd.Flags().StringVarP(&factText, "text", "t", "", "New fact text (required)")
	factUpdateCmd.MarkFlagRequired("text")
	factCmd.AddCommand(factUpdateCmd)

	factCmd.AddCommand(factDeleteCmd)

	factSimilarCmd.Flags().IntVarP(&factSimilarLimit, "limit", "l", 10, "Maximum number of similar facts")
	factSimilarCmd.Flags().BoolVar(&factSimilarFull, "full", false, "Show full fact text (default: truncated to 300 chars)")
	factCmd.AddCommand(factSimilarCmd)

	factCmd.AddCommand(factLinkCmd)
	factCmd.AddCommand(factMergeCmd)
}

func runFactList(cmd *cobra.Command, args []string) error {
	cfg, err := loadConfig()
	if err != nil {
		return output.WriteError(os.Stdout, "Config error", err.Error())
	}

	client := api.NewClient(cfg.APIURL, cfg.Token, cfg.TimeoutSeconds)

	url := fmt.Sprintf("/api/facts?page=%d&per_page=%d", factListPage, factListLimit)
	if factListSearch != "" {
		url += fmt.Sprintf("&search=%s", strings.ReplaceAll(factListSearch, " ", "%20"))
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

	var result struct {
		Facts      []FactWithCard `json:"facts"`
		Page       int            `json:"page"`
		PerPage    int            `json:"per_page"`
		Total      int            `json:"total"`
		TotalPages int            `json:"total_pages"`
	}

	if err := json.Unmarshal(body, &result); err != nil {
		return output.WriteError(os.Stdout, "Parse error", err.Error())
	}

	// Truncate fact text if not in full mode
	for i := range result.Facts {
		truncateFactText(&result.Facts[i], factListFull)
	}

	// Return as list with pagination info
	return output.WriteList(os.Stdout, result.Facts, result.Total, result.PerPage, (result.Page-1)*result.PerPage)
}

func runFactGet(cmd *cobra.Command, args []string) error {
	factID, err := strconv.Atoi(args[0])
	if err != nil {
		return output.WriteError(os.Stdout, "Invalid fact ID", "ID must be a number")
	}

	cfg, err := loadConfig()
	if err != nil {
		return output.WriteError(os.Stdout, "Config error", err.Error())
	}

	client := api.NewClient(cfg.APIURL, cfg.Token, cfg.TimeoutSeconds)
	resp, err := client.Get(fmt.Sprintf("/api/facts/%d", factID))
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

	var fact FactWithCard
	if err := json.Unmarshal(body, &fact); err != nil {
		return output.WriteError(os.Stdout, "Parse error", err.Error())
	}

	return output.WriteSuccess(os.Stdout, fact)
}

func runFactUpdate(cmd *cobra.Command, args []string) error {
	factID, err := strconv.Atoi(args[0])
	if err != nil {
		return output.WriteError(os.Stdout, "Invalid fact ID", "ID must be a number")
	}

	cfg, err := loadConfig()
	if err != nil {
		return output.WriteError(os.Stdout, "Config error", err.Error())
	}

	client := api.NewClient(cfg.APIURL, cfg.Token, cfg.TimeoutSeconds)

	reqBody := map[string]string{"fact": factText}
	bodyBytes, err := json.Marshal(reqBody)
	if err != nil {
		return output.WriteError(os.Stdout, "JSON encode error", err.Error())
	}

	resp, err := client.Put(fmt.Sprintf("/api/facts/%d", factID), bodyBytes)
	if err != nil {
		return output.WriteError(os.Stdout, "API request failed", err.Error())
	}

	if resp.StatusCode != 200 {
		body, _ := api.GetBodyString(resp)
		return output.WriteError(os.Stdout, fmt.Sprintf("API error: %d", resp.StatusCode), body)
	}

	return output.WriteMessage(os.Stdout, "Fact updated")
}

func runFactDelete(cmd *cobra.Command, args []string) error {
	factID, err := strconv.Atoi(args[0])
	if err != nil {
		return output.WriteError(os.Stdout, "Invalid fact ID", "ID must be a number")
	}

	cfg, err := loadConfig()
	if err != nil {
		return output.WriteError(os.Stdout, "Config error", err.Error())
	}

	client := api.NewClient(cfg.APIURL, cfg.Token, cfg.TimeoutSeconds)
	resp, err := client.Delete(fmt.Sprintf("/api/facts/%d", factID))
	if err != nil {
		return output.WriteError(os.Stdout, "API request failed", err.Error())
	}

	if resp.StatusCode != 200 {
		body, _ := api.GetBodyString(resp)
		return output.WriteError(os.Stdout, fmt.Sprintf("API error: %d", resp.StatusCode), body)
	}

	return output.WriteMessage(os.Stdout, "Fact deleted")
}

func runFactSimilar(cmd *cobra.Command, args []string) error {
	factID, err := strconv.Atoi(args[0])
	if err != nil {
		return output.WriteError(os.Stdout, "Invalid fact ID", "ID must be a number")
	}

	cfg, err := loadConfig()
	if err != nil {
		return output.WriteError(os.Stdout, "Config error", err.Error())
	}

	client := api.NewClient(cfg.APIURL, cfg.Token, cfg.TimeoutSeconds)
	url := fmt.Sprintf("/api/facts/%d/similar?limit=%d", factID, factSimilarLimit)

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

	var facts []FactWithCardAndScore
	if err := json.Unmarshal(body, &facts); err != nil {
		return output.WriteError(os.Stdout, "Parse error", err.Error())
	}

	// Truncate fact text if not in full mode
	for i := range facts {
		truncateFactWithScoreText(&facts[i], factSimilarFull)
	}

	return output.WriteSuccess(os.Stdout, facts)
}

func runFactLink(cmd *cobra.Command, args []string) error {
	factID, err := strconv.Atoi(args[0])
	if err != nil {
		return output.WriteError(os.Stdout, "Invalid fact ID", "ID must be a number")
	}

	cardID, err := strconv.Atoi(args[1])
	if err != nil {
		return output.WriteError(os.Stdout, "Invalid card ID", "ID must be a number")
	}

	cfg, err := loadConfig()
	if err != nil {
		return output.WriteError(os.Stdout, "Config error", err.Error())
	}

	client := api.NewClient(cfg.APIURL, cfg.Token, cfg.TimeoutSeconds)
	resp, err := client.Post(fmt.Sprintf("/api/facts/%d/link/%d", factID, cardID), nil)
	if err != nil {
		return output.WriteError(os.Stdout, "API request failed", err.Error())
	}

	if resp.StatusCode != 200 {
		body, _ := api.GetBodyString(resp)
		return output.WriteError(os.Stdout, fmt.Sprintf("API error: %d", resp.StatusCode), body)
	}

	return output.WriteMessage(os.Stdout, "Fact linked to card")
}

func runFactMerge(cmd *cobra.Command, args []string) error {
	fact1ID, err := strconv.Atoi(args[0])
	if err != nil {
		return output.WriteError(os.Stdout, "Invalid fact ID", "ID must be a number")
	}

	fact2ID, err := strconv.Atoi(args[1])
	if err != nil {
		return output.WriteError(os.Stdout, "Invalid fact ID", "ID must be a number")
	}

	if fact1ID == fact2ID {
		return output.WriteError(os.Stdout, "Invalid merge", "Cannot merge a fact with itself")
	}

	cfg, err := loadConfig()
	if err != nil {
		return output.WriteError(os.Stdout, "Config error", err.Error())
	}

	client := api.NewClient(cfg.APIURL, cfg.Token, cfg.TimeoutSeconds)

	reqBody := map[string]int{
		"fact1_id": fact1ID,
		"fact2_id": fact2ID,
	}
	bodyBytes, err := json.Marshal(reqBody)
	if err != nil {
		return output.WriteError(os.Stdout, "JSON encode error", err.Error())
	}

	resp, err := client.Post("/api/facts/merge", bodyBytes)
	if err != nil {
		return output.WriteError(os.Stdout, "API request failed", err.Error())
	}

	if resp.StatusCode != 200 {
		body, _ := api.GetBodyString(resp)
		return output.WriteError(os.Stdout, fmt.Sprintf("API error: %d", resp.StatusCode), body)
	}

	return output.WriteMessage(os.Stdout, fmt.Sprintf("Facts merged: %d absorbed %d", fact1ID, fact2ID))
}

// truncateFactText truncates the fact text if not in full mode
func truncateFactText(fact *FactWithCard, fullMode bool) {
	if !fullMode && len(fact.Fact) > maxPreviewLen {
		fact.Fact = truncatePreview(fact.Fact)
	}
}

// truncateFactWithScoreText truncates the fact text if not in full mode
func truncateFactWithScoreText(fact *FactWithCardAndScore, fullMode bool) {
	if !fullMode && len(fact.Fact) > maxPreviewLen {
		fact.Fact = truncatePreview(fact.Fact)
	}
}

// GetFactCmd returns the fact command for registration
func GetFactCmd() *cobra.Command {
	return factCmd
}
