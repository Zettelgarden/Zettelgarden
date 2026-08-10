package cmd

import (
	"encoding/json"
	"fmt"
	"os"
	"strconv"

	"github.com/nick-zettelgarden/zg/internal/api"
	"github.com/nick-zettelgarden/zg/internal/output"
	"github.com/spf13/cobra"
)

// SchemaDefinition mirrors the backend SchemaDefinition model
// (go-backend/models/schema.go).
type SchemaDefinition struct {
	ID        int    `json:"id"`
	Name      string `json:"name"`
	Slug      string `json:"slug"`
	OwnerID   int    `json:"owner_id"`
	Fields    []any  `json:"fields"`
	CreatedAt string `json:"created_at"`
	UpdatedAt string `json:"updated_at"`
	IsDeleted bool   `json:"is_deleted"`
	CardCount int    `json:"card_count,omitempty"`
}

var schemaCmd = &cobra.Command{
	Use:   "schema",
	Short: "Manage structured-data schemas",
	Long: `Manage structured-data schemas. Use ` + "`zg schema list`" + ` to find a
schema id for ` + "`zg card set-structured-data --schema-id`" + `.`,
}

var schemaListCmd = &cobra.Command{
	Use:   "list",
	Short: "List schemas",
	RunE:  runSchemaList,
}

var schemaGetCmd = &cobra.Command{
	Use:   "get <id>",
	Short: "Get a schema by ID",
	Args:  cobra.ExactArgs(1),
	RunE:  runSchemaGet,
}

func init() {
	schemaCmd.AddCommand(schemaListCmd)
	schemaCmd.AddCommand(schemaGetCmd)
}

// GetSchemaCmd returns the schema command for registration in main.
func GetSchemaCmd() *cobra.Command {
	return schemaCmd
}

func runSchemaList(cmd *cobra.Command, args []string) error {
	cfg, err := loadConfig()
	if err != nil {
		return output.WriteError(os.Stdout, "Config error", err.Error())
	}

	client := api.NewClient(cfg.APIURL, cfg.Token, cfg.TimeoutSeconds)
	resp, err := client.Get("/api/schemas")
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

	var schemas []SchemaDefinition
	if err := json.Unmarshal(body, &schemas); err != nil {
		return output.WriteError(os.Stdout, "Parse error", err.Error())
	}

	return output.WriteSuccess(os.Stdout, schemas)
}

func runSchemaGet(cmd *cobra.Command, args []string) error {
	schemaID, err := strconv.Atoi(args[0])
	if err != nil {
		return output.WriteError(os.Stdout, "Invalid schema ID", "ID must be a number")
	}

	cfg, err := loadConfig()
	if err != nil {
		return output.WriteError(os.Stdout, "Config error", err.Error())
	}

	client := api.NewClient(cfg.APIURL, cfg.Token, cfg.TimeoutSeconds)
	resp, err := client.Get(fmt.Sprintf("/api/schemas/%d", schemaID))
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

	var schema SchemaDefinition
	if err := json.Unmarshal(body, &schema); err != nil {
		return output.WriteError(os.Stdout, "Parse error", err.Error())
	}

	return output.WriteSuccess(os.Stdout, schema)
}
