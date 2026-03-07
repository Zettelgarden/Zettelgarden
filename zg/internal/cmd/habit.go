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

// Habit represents a Zettelgarden habit
type Habit struct {
	ID          int     `json:"id"`
	Title       string  `json:"title"`
	Description *string `json:"description"`
	Frequency   string  `json:"frequency"`
	CustomDays  []int   `json:"custom_days,omitempty"`
	Icon        *string `json:"icon"`
	Color       *string `json:"color"`
	Position    int     `json:"position"`
	CreatedAt   string  `json:"created_at"`
}

// HabitWithCheckin includes today's check-in status
type HabitWithCheckin struct {
	Habit
	IsDueToday     bool  `json:"is_due_today"`
	CheckedInToday bool  `json:"checked_in_today"`
	TodayLogID     *int  `json:"today_log_id"`
}

// HabitStats represents habit statistics
type HabitStats struct {
	CurrentStreak    int     `json:"current_streak"`
	LongestStreak    int     `json:"longest_streak"`
	TotalCompletions int     `json:"total_completions"`
	CompletionRate7d float64 `json:"completion_rate_7d"`
	CompletionRate30d float64 `json:"completion_rate_30d"`
	LastCompletedAt  *string `json:"last_completed_at"`
}

// HabitLog represents a check-in log entry
type HabitLog struct {
	ID          int     `json:"id"`
	HabitID     int     `json:"habit_id"`
	CompletedAt string  `json:"completed_at"`
	Notes       *string `json:"notes"`
}

var habitCmd = &cobra.Command{
	Use:   "habit",
	Short: "Manage habits",
}

var habitGetCmd = &cobra.Command{
	Use:   "get <id>",
	Short: "Get a habit by ID",
	Args:  cobra.ExactArgs(1),
	RunE:  runHabitGet,
}

var habitListCmd = &cobra.Command{
	Use:   "list",
	Short: "List all habits",
	RunE:  runHabitList,
}

var habitTodayCmd = &cobra.Command{
	Use:   "today",
	Short: "List habits due today",
	RunE:  runHabitToday,
}

var habitCreateCmd = &cobra.Command{
	Use:   "create",
	Short: "Create a new habit",
	RunE:  runHabitCreate,
}

var habitUpdateCmd = &cobra.Command{
	Use:   "update <id>",
	Short: "Update a habit",
	Args:  cobra.ExactArgs(1),
	RunE:  runHabitUpdate,
}

var habitDeleteCmd = &cobra.Command{
	Use:   "delete <id>",
	Short: "Delete a habit",
	Args:  cobra.ExactArgs(1),
	RunE:  runHabitDelete,
}

var habitCheckinCmd = &cobra.Command{
	Use:   "checkin <id>",
	Short: "Check in to a habit for today",
	Args:  cobra.ExactArgs(1),
	RunE:  runHabitCheckin,
}

var habitUndoCmd = &cobra.Command{
	Use:   "undo <id> <log-id>",
	Short: "Undo a habit check-in",
	Args:  cobra.ExactArgs(2),
	RunE:  runHabitUndo,
}

var habitStatsCmd = &cobra.Command{
	Use:   "stats <id>",
	Short: "Show habit statistics",
	Args:  cobra.ExactArgs(1),
	RunE:  runHabitStats,
}

var habitLogsCmd = &cobra.Command{
	Use:   "logs <id>",
	Short: "Show habit check-in history",
	Args:  cobra.ExactArgs(1),
	RunE:  runHabitLogs,
}

var (
	habitCreateTitle       string
	habitCreateDescription string
	habitCreateFrequency   string
	habitCreateIcon        string
	habitCreateColor       string

	habitUpdateTitle       string
	habitUpdateDescription string
	habitUpdateFrequency   string
	habitUpdateIcon        string
	habitUpdateColor       string

	habitCheckinNote string

	habitLogsLimit int
)

func init() {
	// Habit get command
	habitCmd.AddCommand(habitGetCmd)

	// Habit list command
	habitCmd.AddCommand(habitListCmd)

	// Habit today command
	habitCmd.AddCommand(habitTodayCmd)

	// Habit create command with flags
	habitCreateCmd.Flags().StringVarP(&habitCreateTitle, "title", "t", "", "Habit title (required)")
	habitCreateCmd.Flags().StringVarP(&habitCreateDescription, "description", "d", "", "Habit description")
	habitCreateCmd.Flags().StringVarP(&habitCreateFrequency, "frequency", "f", "daily", "Frequency (daily/weekly)")
	habitCreateCmd.Flags().StringVar(&habitCreateIcon, "icon", "", "Emoji icon")
	habitCreateCmd.Flags().StringVar(&habitCreateColor, "color", "", "Hex color (e.g., #10b981)")
	habitCreateCmd.MarkFlagRequired("title")
	habitCmd.AddCommand(habitCreateCmd)

	// Habit update command with flags
	habitUpdateCmd.Flags().StringVarP(&habitUpdateTitle, "title", "t", "", "New title")
	habitUpdateCmd.Flags().StringVarP(&habitUpdateDescription, "description", "d", "", "New description")
	habitUpdateCmd.Flags().StringVarP(&habitUpdateFrequency, "frequency", "f", "", "New frequency (daily/weekly)")
	habitUpdateCmd.Flags().StringVar(&habitUpdateIcon, "icon", "", "New icon")
	habitUpdateCmd.Flags().StringVar(&habitUpdateColor, "color", "", "New color")
	habitCmd.AddCommand(habitUpdateCmd)

	// Habit delete command
	habitCmd.AddCommand(habitDeleteCmd)

	// Habit checkin command with flags
	habitCheckinCmd.Flags().StringVarP(&habitCheckinNote, "note", "n", "", "Optional note for this check-in")
	habitCmd.AddCommand(habitCheckinCmd)

	// Habit undo command
	habitCmd.AddCommand(habitUndoCmd)

	// Habit stats command
	habitCmd.AddCommand(habitStatsCmd)

	// Habit logs command with flags
	habitLogsCmd.Flags().IntVarP(&habitLogsLimit, "limit", "l", 30, "Limit results")
	habitCmd.AddCommand(habitLogsCmd)
}

func runHabitGet(cmd *cobra.Command, args []string) error {
	habitID, err := strconv.Atoi(args[0])
	if err != nil {
		return output.WriteError(os.Stdout, "Invalid habit ID", "ID must be a number")
	}

	cfg, err := loadConfig()
	if err != nil {
		return output.WriteError(os.Stdout, "Config error", err.Error())
	}

	client := api.NewClient(cfg.APIURL, cfg.Token, cfg.TimeoutSeconds)
	resp, err := client.Get(fmt.Sprintf("/api/habits/%d", habitID))
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

	var habit Habit
	if err := json.Unmarshal(body, &habit); err != nil {
		return output.WriteError(os.Stdout, "Parse error", err.Error())
	}

	return output.WriteSuccess(os.Stdout, habit)
}

func runHabitList(cmd *cobra.Command, args []string) error {
	cfg, err := loadConfig()
	if err != nil {
		return output.WriteError(os.Stdout, "Config error", err.Error())
	}

	client := api.NewClient(cfg.APIURL, cfg.Token, cfg.TimeoutSeconds)
	resp, err := client.Get("/api/habits")
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

	var habits []Habit
	if err := json.Unmarshal(body, &habits); err != nil {
		return output.WriteError(os.Stdout, "Parse error", err.Error())
	}

	return output.WriteSuccess(os.Stdout, habits)
}

func runHabitToday(cmd *cobra.Command, args []string) error {
	cfg, err := loadConfig()
	if err != nil {
		return output.WriteError(os.Stdout, "Config error", err.Error())
	}

	client := api.NewClient(cfg.APIURL, cfg.Token, cfg.TimeoutSeconds)
	resp, err := client.Get("/api/habits/today")
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

	var habits []HabitWithCheckin
	if err := json.Unmarshal(body, &habits); err != nil {
		return output.WriteError(os.Stdout, "Parse error", err.Error())
	}

	return output.WriteSuccess(os.Stdout, habits)
}

func runHabitCreate(cmd *cobra.Command, args []string) error {
	cfg, err := loadConfig()
	if err != nil {
		return output.WriteError(os.Stdout, "Config error", err.Error())
	}

	requestBody := map[string]any{
		"title":     habitCreateTitle,
		"frequency": habitCreateFrequency,
	}
	if habitCreateDescription != "" {
		requestBody["description"] = habitCreateDescription
	}
	if habitCreateIcon != "" {
		requestBody["icon"] = habitCreateIcon
	}
	if habitCreateColor != "" {
		requestBody["color"] = habitCreateColor
	}

	bodyBytes, err := json.Marshal(requestBody)
	if err != nil {
		return output.WriteError(os.Stdout, "JSON encode error", err.Error())
	}

	client := api.NewClient(cfg.APIURL, cfg.Token, cfg.TimeoutSeconds)
	resp, err := client.Post("/api/habits", bodyBytes)
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

func runHabitUpdate(cmd *cobra.Command, args []string) error {
	habitID, err := strconv.Atoi(args[0])
	if err != nil {
		return output.WriteError(os.Stdout, "Invalid habit ID", "ID must be a number")
	}

	cfg, err := loadConfig()
	if err != nil {
		return output.WriteError(os.Stdout, "Config error", err.Error())
	}

	requestBody := map[string]any{}
	if habitUpdateTitle != "" {
		requestBody["title"] = habitUpdateTitle
	}
	if habitUpdateDescription != "" {
		requestBody["description"] = habitUpdateDescription
	}
	if habitUpdateFrequency != "" {
		requestBody["frequency"] = habitUpdateFrequency
	}
	if habitUpdateIcon != "" {
		requestBody["icon"] = habitUpdateIcon
	}
	if habitUpdateColor != "" {
		requestBody["color"] = habitUpdateColor
	}

	if len(requestBody) == 0 {
		return output.WriteError(os.Stdout, "No updates", "Specify at least one field")
	}

	bodyBytes, err := json.Marshal(requestBody)
	if err != nil {
		return output.WriteError(os.Stdout, "JSON encode error", err.Error())
	}

	client := api.NewClient(cfg.APIURL, cfg.Token, cfg.TimeoutSeconds)
	resp, err := client.Put(fmt.Sprintf("/api/habits/%d", habitID), bodyBytes)
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

	return output.WriteSuccess(os.Stdout, map[string]any{"message": "Habit updated"})
}

func runHabitDelete(cmd *cobra.Command, args []string) error {
	habitID, err := strconv.Atoi(args[0])
	if err != nil {
		return output.WriteError(os.Stdout, "Invalid habit ID", "ID must be a number")
	}

	cfg, err := loadConfig()
	if err != nil {
		return output.WriteError(os.Stdout, "Config error", err.Error())
	}

	client := api.NewClient(cfg.APIURL, cfg.Token, cfg.TimeoutSeconds)
	resp, err := client.Delete(fmt.Sprintf("/api/habits/%d", habitID))
	if err != nil {
		return output.WriteError(os.Stdout, "API request failed", err.Error())
	}

	if resp.StatusCode != 204 && resp.StatusCode != 200 {
		body, _ := api.GetBodyString(resp)
		return output.WriteError(os.Stdout, fmt.Sprintf("API error: %d", resp.StatusCode), body)
	}

	return output.WriteSuccess(os.Stdout, map[string]any{"message": "Habit deleted"})
}

func runHabitCheckin(cmd *cobra.Command, args []string) error {
	habitID, err := strconv.Atoi(args[0])
	if err != nil {
		return output.WriteError(os.Stdout, "Invalid habit ID", "ID must be a number")
	}

	cfg, err := loadConfig()
	if err != nil {
		return output.WriteError(os.Stdout, "Config error", err.Error())
	}

	requestBody := map[string]any{}
	if habitCheckinNote != "" {
		requestBody["notes"] = habitCheckinNote
	}

	bodyBytes, err := json.Marshal(requestBody)
	if err != nil {
		return output.WriteError(os.Stdout, "JSON encode error", err.Error())
	}

	client := api.NewClient(cfg.APIURL, cfg.Token, cfg.TimeoutSeconds)
	resp, err := client.Post(fmt.Sprintf("/api/habits/%d/checkin", habitID), bodyBytes)
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

	var result map[string]any
	if err := json.Unmarshal(respBody, &result); err != nil {
		return output.WriteError(os.Stdout, "Parse error", err.Error())
	}

	return output.WriteSuccess(os.Stdout, result)
}

func runHabitUndo(cmd *cobra.Command, args []string) error {
	habitID, err := strconv.Atoi(args[0])
	if err != nil {
		return output.WriteError(os.Stdout, "Invalid habit ID", "ID must be a number")
	}

	logID, err := strconv.Atoi(args[1])
	if err != nil {
		return output.WriteError(os.Stdout, "Invalid log ID", "ID must be a number")
	}

	cfg, err := loadConfig()
	if err != nil {
		return output.WriteError(os.Stdout, "Config error", err.Error())
	}

	client := api.NewClient(cfg.APIURL, cfg.Token, cfg.TimeoutSeconds)
	resp, err := client.Delete(fmt.Sprintf("/api/habits/%d/checkin/%d", habitID, logID))
	if err != nil {
		return output.WriteError(os.Stdout, "API request failed", err.Error())
	}

	if resp.StatusCode != 204 && resp.StatusCode != 200 {
		body, _ := api.GetBodyString(resp)
		return output.WriteError(os.Stdout, fmt.Sprintf("API error: %d", resp.StatusCode), body)
	}

	return output.WriteSuccess(os.Stdout, map[string]any{"message": "Check-in undone"})
}

func runHabitStats(cmd *cobra.Command, args []string) error {
	habitID, err := strconv.Atoi(args[0])
	if err != nil {
		return output.WriteError(os.Stdout, "Invalid habit ID", "ID must be a number")
	}

	cfg, err := loadConfig()
	if err != nil {
		return output.WriteError(os.Stdout, "Config error", err.Error())
	}

	client := api.NewClient(cfg.APIURL, cfg.Token, cfg.TimeoutSeconds)
	resp, err := client.Get(fmt.Sprintf("/api/habits/%d/stats", habitID))
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

	var stats HabitStats
	if err := json.Unmarshal(body, &stats); err != nil {
		return output.WriteError(os.Stdout, "Parse error", err.Error())
	}

	return output.WriteSuccess(os.Stdout, stats)
}

func runHabitLogs(cmd *cobra.Command, args []string) error {
	habitID, err := strconv.Atoi(args[0])
	if err != nil {
		return output.WriteError(os.Stdout, "Invalid habit ID", "ID must be a number")
	}

	cfg, err := loadConfig()
	if err != nil {
		return output.WriteError(os.Stdout, "Config error", err.Error())
	}

	url := fmt.Sprintf("/api/habits/%d/logs?limit=%d", habitID, habitLogsLimit)

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

	var logs []HabitLog
	if err := json.Unmarshal(body, &logs); err != nil {
		return output.WriteError(os.Stdout, "Parse error", err.Error())
	}

	return output.WriteSuccess(os.Stdout, logs)
}

// GetHabitCmd returns the habit command for registration
func GetHabitCmd() *cobra.Command {
	return habitCmd
}
