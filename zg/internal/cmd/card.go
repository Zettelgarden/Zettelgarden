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

// GetCardCmd returns the card command for registration
func GetCardCmd() *cobra.Command {
	return cardCmd
}
