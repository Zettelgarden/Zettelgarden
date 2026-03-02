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

// Task represents a Zettelgarden task
type Task struct {
	ID          int     `json:"id"`
	Title       string  `json:"title"`
	Description string  `json:"description"`
	IsComplete  bool    `json:"is_complete"`
	Priority    *string `json:"priority"`
	Status      *string `json:"status"`
	ScheduledAt *string `json:"scheduled_at"`
	CreatedAt   string  `json:"created_at"`
}

var taskCmd = &cobra.Command{
	Use:   "task",
	Short: "Manage tasks",
}

var taskGetCmd = &cobra.Command{
	Use:   "get <id>",
	Short: "Get a task by ID",
	Args:  cobra.ExactArgs(1),
	RunE:  runTaskGet,
}

var taskListCmd = &cobra.Command{
	Use:   "list",
	Short: "List tasks",
	RunE:  runTaskList,
}

var taskCreateCmd = &cobra.Command{
	Use:   "create",
	Short: "Create a new task",
	RunE:  runTaskCreate,
}

var taskUpdateCmd = &cobra.Command{
	Use:   "update <id>",
	Short: "Update a task",
	Args:  cobra.ExactArgs(1),
	RunE:  runTaskUpdate,
}

var taskDeleteCmd = &cobra.Command{
	Use:   "delete <id>",
	Short: "Delete a task",
	Args:  cobra.ExactArgs(1),
	RunE:  runTaskDelete,
}

var taskCompleteCmd = &cobra.Command{
	Use:   "complete <id>",
	Short: "Mark a task as complete",
	Args:  cobra.ExactArgs(1),
	RunE:  runTaskComplete,
}

var (
	taskListLimit       int
	taskListCompleted   *bool
	taskListPriority    string
	taskListScheduled   string
	taskListStatus      string

	taskCreateTitle       string
	taskCreateDescription string
	taskCreateScheduled   string
	taskCreatePriority    string

	taskUpdateTitle       string
	taskUpdateDescription string
	taskUpdateIsComplete  *bool
	taskUpdatePriority    string
	taskUpdateScheduled   string
	taskUpdateStatus      string
)

func init() {
	// Task get command
	taskCmd.AddCommand(taskGetCmd)

	// Task list command with flags
	taskListCompletedVal := false
	taskListCompleted = &taskListCompletedVal
	taskListCmd.Flags().IntVarP(&taskListLimit, "limit", "l", 50, "Limit results")
	taskListCmd.Flags().BoolVar(taskListCompleted, "completed", false, "Show only completed tasks")
	taskListCmd.Flags().BoolVar(taskListCompleted, "incomplete", false, "Show only incomplete tasks")
	taskListCmd.Flags().StringVarP(&taskListPriority, "priority", "p", "", "Filter by priority (high/medium/low)")
	taskListCmd.Flags().StringVar(&taskListScheduled, "scheduled-date", "", "Filter by scheduled date (YYYY-MM-DD)")
	taskListCmd.Flags().StringVar(&taskListStatus, "status", "", "Filter by status")
	taskCmd.AddCommand(taskListCmd)

	// Task create command with flags
	taskCreateCmd.Flags().StringVarP(&taskCreateTitle, "title", "t", "", "Task title (required)")
	taskCreateCmd.Flags().StringVarP(&taskCreateDescription, "description", "d", "", "Task description")
	taskCreateCmd.Flags().StringVar(&taskCreateScheduled, "scheduled-date", "", "Scheduled date (YYYY-MM-DD or 'today')")
	taskCreateCmd.Flags().StringVarP(&taskCreatePriority, "priority", "p", "", "Priority (high/medium/low)")
	taskCreateCmd.MarkFlagRequired("title")
	taskCmd.AddCommand(taskCreateCmd)

	// Task update command with flags
	taskUpdateCompleteVal := false
	taskUpdateIsComplete = &taskUpdateCompleteVal
	taskUpdateCmd.Flags().StringVarP(&taskUpdateTitle, "title", "t", "", "New title")
	taskUpdateCmd.Flags().StringVarP(&taskUpdateDescription, "description", "d", "", "New description")
	taskUpdateCmd.Flags().BoolVar(taskUpdateIsComplete, "complete", false, "Mark as complete")
	taskUpdateCmd.Flags().BoolVar(taskUpdateIsComplete, "incomplete", false, "Mark as incomplete")
	taskUpdateCmd.Flags().StringVarP(&taskUpdatePriority, "priority", "p", "", "New priority")
	taskUpdateCmd.Flags().StringVar(&taskUpdateScheduled, "scheduled-date", "", "New scheduled date")
	taskUpdateCmd.Flags().StringVar(&taskUpdateStatus, "status", "", "New status")
	taskCmd.AddCommand(taskUpdateCmd)

	// Task delete command
	taskCmd.AddCommand(taskDeleteCmd)

	// Task complete command
	taskCmd.AddCommand(taskCompleteCmd)
}

func runTaskGet(cmd *cobra.Command, args []string) error {
	taskID, err := strconv.Atoi(args[0])
	if err != nil {
		return output.WriteError(os.Stdout, "Invalid task ID", "ID must be a number")
	}

	cfg, err := loadConfig()
	if err != nil {
		return output.WriteError(os.Stdout, "Config error", err.Error())
	}

	client := api.NewClient(cfg.APIURL, cfg.Token, cfg.TimeoutSeconds)
	resp, err := client.Get(fmt.Sprintf("/api/user/tasks/%d", taskID))
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

	var task Task
	if err := json.Unmarshal(body, &task); err != nil {
		return output.WriteError(os.Stdout, "Parse error", err.Error())
	}

	return output.WriteSuccess(os.Stdout, task)
}

func runTaskList(cmd *cobra.Command, args []string) error {
	cfg, err := loadConfig()
	if err != nil {
		return output.WriteError(os.Stdout, "Config error", err.Error())
	}

	url := fmt.Sprintf("/api/user/tasks?limit=%d", taskListLimit)
	if cmd.Flags().Changed("completed") || cmd.Flags().Changed("incomplete") {
		url += "&completed=" + strconv.FormatBool(*taskListCompleted)
	}
	if taskListPriority != "" {
		url += "&priority=" + taskListPriority
	}
	if taskListScheduled != "" {
		url += "&scheduled_date=" + taskListScheduled
	}
	if taskListStatus != "" {
		url += "&status=" + taskListStatus
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

	var result struct {
		Tasks  []Task `json:"tasks"`
		Total  int    `json:"total"`
		Limit  int    `json:"limit"`
		Offset int    `json:"offset"`
	}

	if err := json.Unmarshal(body, &result); err != nil {
		var tasks []Task
		if err2 := json.Unmarshal(body, &tasks); err2 == nil {
			return output.WriteSuccess(os.Stdout, tasks)
		}
		return output.WriteError(os.Stdout, "Parse error", err.Error())
	}

	return output.WriteList(os.Stdout, result.Tasks, result.Total, result.Limit, result.Offset)
}

func runTaskCreate(cmd *cobra.Command, args []string) error {
	cfg, err := loadConfig()
	if err != nil {
		return output.WriteError(os.Stdout, "Config error", err.Error())
	}

	requestBody := map[string]any{
		"title": taskCreateTitle,
	}
	if taskCreateDescription != "" {
		requestBody["description"] = taskCreateDescription
	}
	if taskCreateScheduled != "" {
		requestBody["scheduled_date"] = taskCreateScheduled
	}
	if taskCreatePriority != "" {
		requestBody["priority"] = taskCreatePriority
	}

	bodyBytes, err := json.Marshal(requestBody)
	if err != nil {
		return output.WriteError(os.Stdout, "JSON encode error", err.Error())
	}

	client := api.NewClient(cfg.APIURL, cfg.Token, cfg.TimeoutSeconds)
	resp, err := client.Post("/api/user/tasks", bodyBytes)
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

func runTaskUpdate(cmd *cobra.Command, args []string) error {
	taskID, err := strconv.Atoi(args[0])
	if err != nil {
		return output.WriteError(os.Stdout, "Invalid task ID", "ID must be a number")
	}

	cfg, err := loadConfig()
	if err != nil {
		return output.WriteError(os.Stdout, "Config error", err.Error())
	}

	requestBody := map[string]any{}
	if taskUpdateTitle != "" {
		requestBody["title"] = taskUpdateTitle
	}
	if taskUpdateDescription != "" {
		requestBody["description"] = taskUpdateDescription
	}
	if cmd.Flags().Changed("complete") || cmd.Flags().Changed("incomplete") {
		requestBody["is_complete"] = *taskUpdateIsComplete
	}
	if taskUpdatePriority != "" {
		requestBody["priority"] = taskUpdatePriority
	}
	if taskUpdateScheduled != "" {
		requestBody["scheduled_date"] = taskUpdateScheduled
	}
	if taskUpdateStatus != "" {
		requestBody["status"] = taskUpdateStatus
	}

	if len(requestBody) == 0 {
		return output.WriteError(os.Stdout, "No updates", "Specify at least one field")
	}

	bodyBytes, err := json.Marshal(requestBody)
	if err != nil {
		return output.WriteError(os.Stdout, "JSON encode error", err.Error())
	}

	client := api.NewClient(cfg.APIURL, cfg.Token, cfg.TimeoutSeconds)
	resp, err := client.Put(fmt.Sprintf("/api/user/tasks/%d", taskID), bodyBytes)
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

	return output.WriteSuccess(os.Stdout, map[string]any{"message": "Task updated"})
}

func runTaskDelete(cmd *cobra.Command, args []string) error {
	taskID, err := strconv.Atoi(args[0])
	if err != nil {
		return output.WriteError(os.Stdout, "Invalid task ID", "ID must be a number")
	}

	cfg, err := loadConfig()
	if err != nil {
		return output.WriteError(os.Stdout, "Config error", err.Error())
	}

	client := api.NewClient(cfg.APIURL, cfg.Token, cfg.TimeoutSeconds)
	resp, err := client.Delete(fmt.Sprintf("/api/user/tasks/%d", taskID))
	if err != nil {
		return output.WriteError(os.Stdout, "API request failed", err.Error())
	}

	if resp.StatusCode != 204 {
		body, _ := api.GetBodyString(resp)
		return output.WriteError(os.Stdout, fmt.Sprintf("API error: %d", resp.StatusCode), body)
	}

	return output.WriteSuccess(os.Stdout, map[string]any{"message": "Task deleted"})
}

func runTaskComplete(cmd *cobra.Command, args []string) error {
	return runTaskUpdate(cmd, args)
}

// GetTaskCmd returns the task command for registration
func GetTaskCmd() *cobra.Command {
	return taskCmd
}
