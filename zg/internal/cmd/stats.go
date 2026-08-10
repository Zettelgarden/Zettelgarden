package cmd

import (
	"encoding/json"
	"fmt"
	"net/url"
	"os"
	"time"

	"github.com/nick-zettelgarden/zg/internal/api"
	"github.com/nick-zettelgarden/zg/internal/output"
	"github.com/spf13/cobra"
)

// DailyStats mirrors go-backend/models/stats.go.
type DailyStats struct {
	Date           time.Time `json:"date"`
	CardsCreated   int       `json:"cards_created"`
	TasksCreated   int       `json:"tasks_created"`
	TasksCompleted int       `json:"tasks_completed"`
}

// DailyStatsResponse mirrors the /api/stats/daily response.
type DailyStatsResponse struct {
	Stats []DailyStats `json:"stats"`
	Total struct {
		CardsCreated   int `json:"cards_created"`
		TasksCreated   int `json:"tasks_created"`
		TasksCompleted int `json:"tasks_completed"`
	} `json:"total"`
}

var statsCmd = &cobra.Command{
	Use:   "stats",
	Short: "Show daily activity statistics",
	Long: `Show cards/tasks created and tasks completed per day for a date range.
Defaults to the last 30 days.`,
	RunE: runStats,
}

var statsDays int

func init() {
	statsCmd.Flags().IntVarP(&statsDays, "days", "d", 30, "Number of days to report (max 365)")
}

// GetStatsCmd returns the stats command for registration in main.
func GetStatsCmd() *cobra.Command {
	return statsCmd
}

func runStats(cmd *cobra.Command, args []string) error {
	if statsDays <= 0 || statsDays > 365 {
		return output.WriteError(os.Stdout, "Invalid --days", "Must be between 1 and 365")
	}

	cfg, err := loadConfig()
	if err != nil {
		return output.WriteError(os.Stdout, "Config error", err.Error())
	}

	end := time.Now()
	start := end.AddDate(0, 0, -(statsDays - 1))
	query := url.Values{}
	query.Set("start_date", start.Format("2006-01-02"))
	query.Set("end_date", end.Format("2006-01-02"))

	client := api.NewClient(cfg.APIURL, cfg.Token, cfg.TimeoutSeconds)
	resp, err := client.Get("/api/stats/daily?" + query.Encode())
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

	var result DailyStatsResponse
	if err := json.Unmarshal(body, &result); err != nil {
		return output.WriteError(os.Stdout, "Parse error", err.Error())
	}

	return output.WriteSuccess(os.Stdout, result)
}
