package cmd

import (
	"encoding/json"
	"fmt"
	"os"
	"strconv"

	"github.com/spf13/cobra"
	"github.com/nick-zettelgarden/zg/internal/api"
	"github.com/nick-zettelgarden/zg/internal/output"
)

// CardTemplate represents a Zettelgarden card template
type CardTemplate struct {
	ID        int    `json:"id"`
	UserID    int    `json:"user_id"`
	Name      string `json:"name"`
	Title     string `json:"title"`
	Body      string `json:"body"`
	CreatedAt string `json:"created_at"`
	UpdatedAt string `json:"updated_at"`
}

var templateCmd = &cobra.Command{
	Use:   "template",
	Short: "Manage card templates",
}

var templateGetCmd = &cobra.Command{
	Use:   "get <id>",
	Short: "Get a template by ID",
	Args:  cobra.ExactArgs(1),
	RunE:  runTemplateGet,
}

var templateListCmd = &cobra.Command{
	Use:   "list",
	Short: "List templates",
	RunE:  runTemplateList,
}

func init() {
	templateCmd.AddCommand(templateGetCmd)
	templateCmd.AddCommand(templateListCmd)
}

func runTemplateGet(cmd *cobra.Command, args []string) error {
	templateID, err := strconv.Atoi(args[0])
	if err != nil {
		return output.WriteError(os.Stdout, "Invalid template ID", "ID must be a number")
	}

	cfg, err := loadConfig()
	if err != nil {
		return output.WriteError(os.Stdout, "Config error", err.Error())
	}

	client := api.NewClient(cfg.APIURL, cfg.Token, cfg.TimeoutSeconds)
	resp, err := client.Get(fmt.Sprintf("/api/templates/%d", templateID))
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

	var tmpl CardTemplate
	if err := json.Unmarshal(body, &tmpl); err != nil {
		return output.WriteError(os.Stdout, "Parse error", err.Error())
	}

	return output.WriteSuccess(os.Stdout, tmpl)
}

func runTemplateList(cmd *cobra.Command, args []string) error {
	cfg, err := loadConfig()
	if err != nil {
		return output.WriteError(os.Stdout, "Config error", err.Error())
	}

	client := api.NewClient(cfg.APIURL, cfg.Token, cfg.TimeoutSeconds)
	resp, err := client.Get("/api/templates")
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

	var templates []CardTemplate
	if err := json.Unmarshal(body, &templates); err != nil {
		return output.WriteError(os.Stdout, "Parse error", err.Error())
	}

	return output.WriteSuccess(os.Stdout, templates)
}

// GetTemplateCmd returns the template command for registration
func GetTemplateCmd() *cobra.Command {
	return templateCmd
}
