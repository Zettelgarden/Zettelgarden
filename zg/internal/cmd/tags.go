package cmd

import (
	"encoding/json"
	"fmt"
	"os"
	"regexp"
	"strconv"
	"strings"

	"github.com/nick-zettelgarden/zg/internal/api"
	"github.com/nick-zettelgarden/zg/internal/output"
	"github.com/spf13/cobra"
)

// Tag mirrors the backend Tag model (go-backend/models/tags.go).
type Tag struct {
	ID        int    `json:"id"`
	Name      string `json:"name"`
	Color     string `json:"color"`
	UserID    int    `json:"user_id"`
	IsDeleted bool   `json:"is_deleted"`
	CreatedAt string `json:"created_at"`
	UpdatedAt string `json:"updated_at"`
}

var tagCmd = &cobra.Command{
	Use:   "tag",
	Short: "Manage tags",
	Long: `Manage tags. Tags are derived from #hashtags in card bodies, so
` + "`zg tag add`" + ` edits the card body to add one.`,
}

var tagListCmd = &cobra.Command{
	Use:   "list",
	Short: "List all tags",
	RunE:  runTagList,
}

var tagCreateCmd = &cobra.Command{
	Use:   "create <name>",
	Short: "Create a tag",
	Args:  cobra.ExactArgs(1),
	RunE:  runTagCreate,
}

var tagDeleteCmd = &cobra.Command{
	Use:   "delete <id>",
	Short: "Delete a tag by ID",
	Args:  cobra.ExactArgs(1),
	RunE:  runTagDelete,
}

var tagCardsCmd = &cobra.Command{
	Use:   "cards <card-id>",
	Short: "List tags on a card",
	Args:  cobra.ExactArgs(1),
	RunE:  runTagCards,
}

var tagAddCmd = &cobra.Command{
	Use:   "add <card-id> <tag>",
	Short: "Tag a card (adds #tag to its body)",
	Long: `Tag a card by adding the hashtag to its body. Tags in Zettelgarden are
derived from #hashtags in card bodies, so this fetches the card, appends
` + "`#tag`" + ` to the body if missing, and saves it back.`,
	Args: cobra.ExactArgs(2),
	RunE: runTagAdd,
}

var tagColor string

// tagTokenRegex matches the backend's tag tokenization
// (go-backend/services/tags.go ParseTagsFromCardBody: `#` + [\w-]+ after a line
// start or whitespace).
var tagTokenRegex = regexp.MustCompile(`(?:^|\s)(#[\w-]+)`)

func init() {
	tagCreateCmd.Flags().StringVar(&tagColor, "color", "black", "Tag color")
	tagCmd.AddCommand(tagListCmd)
	tagCmd.AddCommand(tagCreateCmd)
	tagCmd.AddCommand(tagDeleteCmd)
	tagCmd.AddCommand(tagCardsCmd)
	tagCmd.AddCommand(tagAddCmd)
}

// GetTagCmd returns the tag command for registration in main.
func GetTagCmd() *cobra.Command {
	return tagCmd
}

func runTagList(cmd *cobra.Command, args []string) error {
	cfg, err := loadConfig()
	if err != nil {
		return output.WriteError(os.Stdout, "Config error", err.Error())
	}

	client := api.NewClient(cfg.APIURL, cfg.Token, cfg.TimeoutSeconds)
	resp, err := client.Get("/api/tags")
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

	var tags []Tag
	if err := json.Unmarshal(body, &tags); err != nil {
		return output.WriteError(os.Stdout, "Parse error", err.Error())
	}
	if tags == nil {
		tags = []Tag{}
	}

	return output.WriteSuccess(os.Stdout, tags)
}

func runTagCreate(cmd *cobra.Command, args []string) error {
	tagName := strings.TrimPrefix(strings.TrimSpace(args[0]), "#")
	if !validTagName(tagName) {
		return output.WriteError(os.Stdout, "Invalid tag name", fmt.Sprintf("%q cannot be a #hashtag (use letters, digits, underscores and hyphens only)", tagName))
	}

	cfg, err := loadConfig()
	if err != nil {
		return output.WriteError(os.Stdout, "Config error", err.Error())
	}

	requestBody, _ := json.Marshal(map[string]string{
		"name":  tagName,
		"color": tagColor,
	})

	client := api.NewClient(cfg.APIURL, cfg.Token, cfg.TimeoutSeconds)
	resp, err := client.Post("/api/tags", requestBody)
	if err != nil {
		return output.WriteError(os.Stdout, "API request failed", err.Error())
	}

	body, err := api.GetBodyBytes(resp)
	if err != nil {
		return output.WriteError(os.Stdout, "Reading response failed", err.Error())
	}
	if resp.StatusCode != 200 && resp.StatusCode != 201 {
		return output.WriteError(os.Stdout, fmt.Sprintf("API error: %d", resp.StatusCode), string(body))
	}

	var tag Tag
	if err := json.Unmarshal(body, &tag); err != nil {
		return output.WriteError(os.Stdout, "Parse error", err.Error())
	}

	return output.WriteSuccess(os.Stdout, tag)
}

func runTagDelete(cmd *cobra.Command, args []string) error {
	tagID, err := strconv.Atoi(args[0])
	if err != nil {
		return output.WriteError(os.Stdout, "Invalid tag ID", "ID must be a number")
	}

	cfg, err := loadConfig()
	if err != nil {
		return output.WriteError(os.Stdout, "Config error", err.Error())
	}

	client := api.NewClient(cfg.APIURL, cfg.Token, cfg.TimeoutSeconds)
	resp, err := client.Delete(fmt.Sprintf("/api/tags/id/%d", tagID))
	if err != nil {
		return output.WriteError(os.Stdout, "API request failed", err.Error())
	}

	body, err := api.GetBodyBytes(resp)
	if err != nil {
		return output.WriteError(os.Stdout, "Reading response failed", err.Error())
	}
	if resp.StatusCode != 200 && resp.StatusCode != 204 {
		return output.WriteError(os.Stdout, fmt.Sprintf("API error: %d", resp.StatusCode), string(body))
	}

	return output.WriteMessage(os.Stdout, fmt.Sprintf("Tag %d deleted", tagID))
}

func runTagCards(cmd *cobra.Command, args []string) error {
	cardID, err := strconv.Atoi(args[0])
	if err != nil {
		return output.WriteError(os.Stdout, "Invalid card ID", "ID must be a number")
	}

	cfg, err := loadConfig()
	if err != nil {
		return output.WriteError(os.Stdout, "Config error", err.Error())
	}

	client := api.NewClient(cfg.APIURL, cfg.Token, cfg.TimeoutSeconds)
	resp, err := client.Get(fmt.Sprintf("/api/cards/%d/tags", cardID))
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

	var tags []Tag
	if err := json.Unmarshal(body, &tags); err != nil {
		return output.WriteError(os.Stdout, "Parse error", err.Error())
	}
	if tags == nil {
		tags = []Tag{}
	}

	return output.WriteSuccess(os.Stdout, tags)
}

// tagInBody reports whether body already contains the #tag hashtag, using the
// same tokenization as the backend (ParseTagsFromCardBody: `#` + [\w-]+ after a
// line start or whitespace). #alpha-beta does not count as tag "alpha".
func tagInBody(body, tagName string) bool {
	for _, m := range tagTokenRegex.FindAllStringSubmatch(body, -1) {
		if len(m) == 2 && m[1] == "#"+tagName {
			return true
		}
	}
	return false
}

// validTagName reports whether name can be represented as a #hashtag in a card
// body (mirrors the backend tokenizer charset).
func validTagName(name string) bool {
	matched, _ := regexp.MatchString(`^[\w-]+$`, name)
	return matched
}

func runTagAdd(cmd *cobra.Command, args []string) error {
	cardID, err := strconv.Atoi(args[0])
	if err != nil {
		return output.WriteError(os.Stdout, "Invalid card ID", "ID must be a number")
	}
	tagName := strings.TrimPrefix(strings.TrimSpace(args[1]), "#")
	if !validTagName(tagName) {
		return output.WriteError(os.Stdout, "Invalid tag name", fmt.Sprintf("%q cannot be a #hashtag (use letters, digits, underscores and hyphens only)", tagName))
	}

	cfg, err := loadConfig()
	if err != nil {
		return output.WriteError(os.Stdout, "Config error", err.Error())
	}

	client := api.NewClient(cfg.APIURL, cfg.Token, cfg.TimeoutSeconds)

	// Fetch the current card so we don't clobber other fields.
	getResp, err := client.Get(fmt.Sprintf("/api/cards/%d", cardID))
	if err != nil {
		return output.WriteError(os.Stdout, "API request failed", err.Error())
	}
	getBody, err := api.GetBodyBytes(getResp)
	if err != nil {
		return output.WriteError(os.Stdout, "Reading response failed", err.Error())
	}
	if getResp.StatusCode != 200 {
		return output.WriteError(os.Stdout, fmt.Sprintf("API error: %d", getResp.StatusCode), string(getBody))
	}

	var current Card
	if err := json.Unmarshal(getBody, &current); err != nil {
		return output.WriteError(os.Stdout, "Parse error", err.Error())
	}

	if tagInBody(current.Body, tagName) {
		return output.WriteMessage(os.Stdout, fmt.Sprintf("Card %d already tagged #%s", cardID, tagName))
	}

	newBody := current.Body
	if strings.TrimSpace(newBody) != "" && !strings.HasSuffix(newBody, "\n") {
		newBody += "\n"
	}
	newBody += fmt.Sprintf("#%s", tagName)

	requestBody, _ := json.Marshal(map[string]any{
		"title":   current.Title,
		"body":    newBody,
		"link":    current.Link,
		"card_id": current.CardID,
	})

	putResp, err := client.Put(fmt.Sprintf("/api/cards/%d", cardID), requestBody)
	if err != nil {
		return output.WriteError(os.Stdout, "API request failed", err.Error())
	}
	putBody, err := api.GetBodyBytes(putResp)
	if err != nil {
		return output.WriteError(os.Stdout, "Reading response failed", err.Error())
	}
	if putResp.StatusCode != 200 {
		return output.WriteError(os.Stdout, fmt.Sprintf("API error: %d", putResp.StatusCode), string(putBody))
	}

	return output.WriteMessage(os.Stdout, fmt.Sprintf("Card %d tagged #%s", cardID, tagName))
}
