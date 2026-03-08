package cmd

import (
	"encoding/json"
	"fmt"
	"os"

	"github.com/spf13/cobra"
	"github.com/nick-zettelgarden/zg/internal/api"
	"github.com/nick-zettelgarden/zg/internal/output"
)

var articleCmd = &cobra.Command{
	Use:   "article",
	Short: "Manage article cards from URLs",
}

var articleCreateCmd = &cobra.Command{
	Use:   "create <url>",
	Short: "Create an article card from a URL",
	Long: `Create an article card from a URL by parsing the web content.

The article content will be extracted, converted to markdown, and stored
as a new card with #to-read #reference tags by default.`,
	Args: cobra.ExactArgs(1),
	RunE: runArticleCreate,
}

var (
	articleCardID string
	articleTags   string
)

func init() {
	articleCreateCmd.Flags().StringVarP(&articleCardID, "card-id", "c", "", "Optional card ID (e.g., '1a')")
	articleCreateCmd.Flags().StringVarP(&articleTags, "tags", "t", "", "Custom tags (default: '#to-read #reference')")
	articleCmd.AddCommand(articleCreateCmd)
}

func runArticleCreate(cmd *cobra.Command, args []string) error {
	url := args[0]

	cfg, err := loadConfig()
	if err != nil {
		return output.WriteError(os.Stdout, "Config error", err.Error())
	}

	requestBody := map[string]any{
		"url": url,
	}
	if articleCardID != "" {
		requestBody["card_id"] = articleCardID
	}
	if articleTags != "" {
		requestBody["tags"] = articleTags
	}

	bodyBytes, err := json.Marshal(requestBody)
	if err != nil {
		return output.WriteError(os.Stdout, "JSON encode error", err.Error())
	}

	client := api.NewClient(cfg.APIURL, cfg.Token, cfg.TimeoutSeconds)
	resp, err := client.Post("/api/articles", bodyBytes)
	if err != nil {
		return output.WriteError(os.Stdout, "API request failed", err.Error())
	}

	respBody, err := api.GetBodyBytes(resp)
	if err != nil {
		return output.WriteError(os.Stdout, "Reading response failed", err.Error())
	}

	if resp.StatusCode != 200 && resp.StatusCode != 201 {
		return output.WriteError(os.Stdout, fmt.Sprintf("API error: %d", resp.StatusCode), string(respBody))
	}

	var card Card
	if err := json.Unmarshal(respBody, &card); err != nil {
		return output.WriteError(os.Stdout, "Parse error", err.Error())
	}

	return output.WriteSuccess(os.Stdout, card)
}

// GetArticleCmd returns the article command for registration
func GetArticleCmd() *cobra.Command {
	return articleCmd
}
