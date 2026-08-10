package cmd

import (
	"encoding/json"
	"fmt"
	"os"

	"github.com/nick-zettelgarden/zg/internal/api"
	"github.com/nick-zettelgarden/zg/internal/output"
	"github.com/spf13/cobra"
)

// ParseResult mirrors the backend ParseResult (go-backend/handlers/cards.go).
type ParseResult struct {
	Title    string `json:"title"`
	Content  string `json:"content"`
	URL      string `json:"url,omitempty"`
	Author   string `json:"author,omitempty"`
	Excerpt  string `json:"excerpt,omitempty"`
	SiteName string `json:"site_name,omitempty"`
}

var parseURLCmd = &cobra.Command{
	Use:   "parse-url <url>",
	Short: "Extract article content from a URL as markdown",
	Long: `Fetch a URL and return the article content as markdown, using the same
readability extraction as the app. Pipe the result into
` + "`zg article create`" + ` for a ready-to-save article card.`,
	Args: cobra.ExactArgs(1),
	RunE: runParseURL,
}

// GetParseURLCmd returns the parse-url command for registration in main.
func GetParseURLCmd() *cobra.Command {
	return parseURLCmd
}

func runParseURL(cmd *cobra.Command, args []string) error {
	cfg, err := loadConfig()
	if err != nil {
		return output.WriteError(os.Stdout, "Config error", err.Error())
	}

	requestBody, _ := json.Marshal(map[string]string{"url": args[0]})

	client := api.NewClient(cfg.APIURL, cfg.Token, cfg.TimeoutSeconds)
	resp, err := client.Post("/api/url/parse", requestBody)
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

	var result ParseResult
	if err := json.Unmarshal(body, &result); err != nil {
		return output.WriteError(os.Stdout, "Parse error", err.Error())
	}

	return output.WriteSuccess(os.Stdout, result)
}
