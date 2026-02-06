// Package services provides business logic and tool implementations for Zettelgarden.
//
// Tool Registry Infrastructure:
// - registry.go: Core tool registration and execution
// - context.go: Tool execution context with transaction support
// - params.go: Parameter extraction and validation helpers
// - types.go: Tool type definitions and constants
//
// Domain-specific tools:
// - card_tools.go: Card CRUD operations and search
// - task_tools.go: Task management and scheduling
// - entity_tools.go: Entity management and linking
// - fact_tools.go: Fact extraction and retrieval
// - template_tools.go: Card template management
// - calendar_tools.go: Calendar integration
// - article_tools.go: Article parsing and creation
// - memory_tools.go: User memory operations
//
// PHASE 3: Domain Package Migration with Feature Flags
// ---------------------------------------------------
// This file now supports both legacy and new domain package registration
// controlled by the FeatureFlagTaskTools feature flag.
//
// - Feature flag DISABLED (default): Uses legacy registration in this file
// - Feature flag ENABLED: Uses services/tools/task package
//
// To enable the new domain package:
//   export ZETTELGARDEN_FEATURE_TASK_TOOLS_V2=true
package services

import (
	"fmt"
	"go-backend/models"
	"time"

	"go-backend/services/featureflags"
	"go-backend/services/tools/task"
)

// RegisterTaskTools registers all task-related tools.
//
// This method supports two registration paths controlled by feature flag:
// 1. Legacy path (default): Registers tools directly from this file
// 2. New path (feature flag): Delegates to services/tools/task package
func (tr *ToolRegistry) RegisterTaskTools() {
	if featureflags.IsEnabled(featureflags.FeatureFlagTaskTools) {
		// NEW: Use the domain package
		tr.registerTaskToolsV2()
	} else {
		// LEGACY: Use the original registration in this file
		tr.registerTaskToolsLegacy()
	}
}

// registerTaskToolsV2 uses the new task domain package
func (tr *ToolRegistry) registerTaskToolsV2() {
	// Get tasks
	RegisterTool(tr,
		"get_tasks",
		"Retrieve a list of tasks for the user. Can optionally filter to include completed tasks or get tasks for a specific card.",
		handleGetTasksV2,
		ToolParam{Name: "include_completed", Type: "boolean", Required: false, Default: false, Desc: "Whether to include completed tasks in the results (default: false)"},
		ToolParam{Name: "card_pk", Type: "integer", Required: false, Desc: "Optional card primary key to filter tasks by card (returns only tasks linked to this card)"},
	)

	// Create task
	RegisterTool(tr,
		"create_task",
		"Create a new task with a title and optional scheduling, priority, and card linkage.",
		handleCreateTaskV2,
		ToolParam{Name: "title", Type: "string", Required: true, Desc: "Title of the task (required)"},
		ToolParam{Name: "scheduled_date", Type: "string", Required: false, Desc: "Optional scheduled date in ISO 8601 format (e.g., 2024-01-15T10:30:00Z)"},
		ToolParam{Name: "due_date", Type: "string", Required: false, Desc: "Optional due date in ISO 8601 format (e.g., 2024-01-15T10:30:00Z)"},
		ToolParam{Name: "priority", Type: "string", Required: false, Desc: "Optional priority level (e.g., 'high', 'medium', 'low')"},
		ToolParam{Name: "card_pk", Type: "integer", Required: false, Desc: "Optional card primary key to link the task to a specific card"},
	)

	// Update task
	RegisterTool(tr,
		"update_task",
		"Update an existing task's properties. Only provided fields will be updated.",
		handleUpdateTaskV2,
		ToolParam{Name: "task_id", Type: "integer", Required: true, Desc: "The ID of the task to update (required)"},
		ToolParam{Name: "title", Type: "string", Required: false, Desc: "Updated title for the task (optional)"},
		ToolParam{Name: "scheduled_date", Type: "string", Required: false, Desc: "Updated scheduled date in ISO 8601 format (optional)"},
		ToolParam{Name: "due_date", Type: "string", Required: false, Desc: "Updated due date in ISO 8601 format (optional)"},
		ToolParam{Name: "priority", Type: "string", Required: false, Desc: "Updated priority level (optional)"},
		ToolParam{Name: "is_complete", Type: "boolean", Required: false, Desc: "Whether the task is complete (optional)"},
		ToolParam{Name: "card_pk", Type: "integer", Required: false, Desc: "Updated card primary key to link the task to (optional)"},
	)

	// Get task by ID
	RegisterTool(tr,
		"get_task_by_id",
		"Retrieve a specific task by its ID.",
		handleGetTaskByIDV2,
		ToolParam{Name: "task_id", Type: "integer", Required: true, Desc: "The ID of the task to retrieve"},
	)

	// Complete task
	RegisterTool(tr,
		"complete_task",
		"Mark a task as complete. This is a convenience wrapper for updating a task's completion status.",
		handleCompleteTaskV2,
		ToolParam{Name: "task_id", Type: "integer", Required: true, Desc: "The ID of the task to mark as complete"},
	)

	// Delete task
	RegisterTool(tr,
		"delete_task",
		"Delete a task by its ID. This action cannot be undone.",
		handleDeleteTaskV2,
		ToolParam{Name: "task_id", Type: "integer", Required: true, Desc: "The ID of the task to delete"},
	)

	// Complete and schedule task
	RegisterTool(tr,
		"complete_and_schedule_task",
		"Complete a recurring task and create a new one scheduled for a specified number of days later. Useful for managing recurring tasks.",
		handleCompleteAndScheduleTaskV2,
		ToolParam{Name: "task_id", Type: "integer", Required: true, Desc: "The ID of the task to complete"},
		ToolParam{Name: "days", Type: "integer", Required: true, Desc: "Number of days to schedule the new task in the future (must be greater than 0)"},
	)
}

// registerTaskToolsLegacy is the original task tools registration.
// Kept for backward compatibility and as a fallback.
func (tr *ToolRegistry) registerTaskToolsLegacy() {
	// Get tasks
	RegisterTool(tr,
		"get_tasks",
		"Retrieve a list of tasks for the user. Can optionally filter to include completed tasks or get tasks for a specific card.",
		handleGetTasksLegacy,
		ToolParam{Name: "include_completed", Type: "boolean", Required: false, Default: false, Desc: "Whether to include completed tasks in the results (default: false)"},
		ToolParam{Name: "card_pk", Type: "integer", Required: false, Desc: "Optional card primary key to filter tasks by card (returns only tasks linked to this card)"},
	)

	// Create task
	RegisterTool(tr,
		"create_task",
		"Create a new task with a title and optional scheduling, priority, and card linkage.",
		handleCreateTaskLegacy,
		ToolParam{Name: "title", Type: "string", Required: true, Desc: "Title of the task (required)"},
		ToolParam{Name: "scheduled_date", Type: "string", Required: false, Desc: "Optional scheduled date in ISO 8601 format (e.g., 2024-01-15T10:30:00Z)"},
		ToolParam{Name: "due_date", Type: "string", Required: false, Desc: "Optional due date in ISO 8601 format (e.g., 2024-01-15T10:30:00Z)"},
		ToolParam{Name: "priority", Type: "string", Required: false, Desc: "Optional priority level (e.g., 'high', 'medium', 'low')"},
		ToolParam{Name: "card_pk", Type: "integer", Required: false, Desc: "Optional card primary key to link the task to a specific card"},
	)

	// Update task
	RegisterTool(tr,
		"update_task",
		"Update an existing task's properties. Only provided fields will be updated.",
		handleUpdateTaskLegacy,
		ToolParam{Name: "task_id", Type: "integer", Required: true, Desc: "The ID of the task to update (required)"},
		ToolParam{Name: "title", Type: "string", Required: false, Desc: "Updated title for the task (optional)"},
		ToolParam{Name: "scheduled_date", Type: "string", Required: false, Desc: "Updated scheduled date in ISO 8601 format (optional)"},
		ToolParam{Name: "due_date", Type: "string", Required: false, Desc: "Updated due date in ISO 8601 format (optional)"},
		ToolParam{Name: "priority", Type: "string", Required: false, Desc: "Updated priority level (optional)"},
		ToolParam{Name: "is_complete", Type: "boolean", Required: false, Desc: "Whether the task is complete (optional)"},
		ToolParam{Name: "card_pk", Type: "integer", Required: false, Desc: "Updated card primary key to link the task to (optional)"},
	)

	// Get task by ID
	RegisterTool(tr,
		"get_task_by_id",
		"Retrieve a specific task by its ID.",
		handleGetTaskByIDLegacy,
		ToolParam{Name: "task_id", Type: "integer", Required: true, Desc: "The ID of the task to retrieve"},
	)

	// Complete task
	RegisterTool(tr,
		"complete_task",
		"Mark a task as complete. This is a convenience wrapper for updating a task's completion status.",
		handleCompleteTaskLegacy,
		ToolParam{Name: "task_id", Type: "integer", Required: true, Desc: "The ID of the task to mark as complete"},
	)

	// Delete task
	RegisterTool(tr,
		"delete_task",
		"Delete a task by its ID. This action cannot be undone.",
		handleDeleteTaskLegacy,
		ToolParam{Name: "task_id", Type: "integer", Required: true, Desc: "The ID of the task to delete"},
	)

	// Complete and schedule task
	RegisterTool(tr,
		"complete_and_schedule_task",
		"Complete a recurring task and create a new one scheduled for a specified number of days later. Useful for managing recurring tasks.",
		handleCompleteAndScheduleTaskLegacy,
		ToolParam{Name: "task_id", Type: "integer", Required: true, Desc: "The ID of the task to complete"},
		ToolParam{Name: "days", Type: "integer", Required: true, Desc: "Number of days to schedule the new task in the future (must be greater than 0)"},
	)
}

// V2 task tool handlers (use domain package logic)

func handleGetTasksV2(args map[string]interface{}, ctx *ToolContext) (map[string]interface{}, error) {
	includeCompleted := getOptionalBoolParam(args, "include_completed")

	var tasks []models.Task
	var err error

	if cardPKFloat, ok := args["card_pk"].(float64); ok {
		cardPK := int(cardPKFloat)
		tasks, err = task.GetTasksByCard(ctx.DB, ctx.UserID, cardPK)
	} else {
		tasks, err = task.GetTasks(ctx.DB, ctx.UserID, includeCompleted, "UTC")
	}

	if err != nil {
		return nil, fmt.Errorf("failed to get tasks: %v", err)
	}

	var results []map[string]interface{}
	for _, t := range tasks {
		results = append(results, StructToMap(t))
	}

	return map[string]interface{}{
		"tasks": results,
		"total": len(tasks),
	}, nil
}

func handleCreateTaskV2(args map[string]interface{}, ctx *ToolContext) (map[string]interface{}, error) {
	title, err := getStringParam(args, "title")
	if err != nil {
		return nil, err
	}

	newTaskModel := models.Task{
		UserID:     ctx.UserID,
		Title:      title,
		IsComplete: false,
	}

	if scheduledDateStr, ok := getOptionalStringParam(args, "scheduled_date"); ok {
		scheduledDate, perr := time.Parse(time.RFC3339, scheduledDateStr)
		if perr != nil {
			return nil, fmt.Errorf("invalid scheduled_date format: %v", perr)
		}
		newTaskModel.ScheduledDate = &scheduledDate
	}

	if dueDateStr, ok := getOptionalStringParam(args, "due_date"); ok {
		dueDate, perr := time.Parse(time.RFC3339, dueDateStr)
		if perr != nil {
			return nil, fmt.Errorf("invalid due_date format: %v", perr)
		}
		newTaskModel.DueDate = &dueDate
	} else {
		now := time.Now()
		newTaskModel.DueDate = &now
	}

	if priority, ok := getOptionalStringParam(args, "priority"); ok {
		newTaskModel.Priority = &priority
	}

	if cardPK, ok, _ := getOptionalIntParam(args, "card_pk"); ok {
		newTaskModel.CardPK = cardPK
	}

	taskID, err := task.CreateTask(ctx.DB, newTaskModel)
	if err != nil {
		return nil, fmt.Errorf("failed to create task: %v", err)
	}

	newTask, err := task.GetTask(ctx.DB, ctx.UserID, taskID)
	if err != nil {
		return nil, fmt.Errorf("failed to retrieve created task: %v", err)
	}

	return StructToMap(newTask), nil
}

func handleUpdateTaskV2(args map[string]interface{}, ctx *ToolContext) (map[string]interface{}, error) {
	taskID, err := getIntParam(args, "task_id")
	if err != nil {
		return nil, err
	}

	currentTask, lerr := task.GetTask(ctx.DB, ctx.UserID, taskID)
	if lerr != nil {
		return nil, fmt.Errorf("failed to get task: %v", lerr)
	}

	if title, ok := getOptionalStringParam(args, "title"); ok {
		currentTask.Title = title
	}

	if scheduledDateStr, ok := getOptionalStringParam(args, "scheduled_date"); ok {
		scheduledDate, perr := time.Parse(time.RFC3339, scheduledDateStr)
		if perr != nil {
			return nil,	fmt.Errorf("invalid scheduled_date format: %v", perr)
		}
		currentTask.ScheduledDate = &scheduledDate
	}

	if dueDateStr, ok := getOptionalStringParam(args, "due_date"); ok {
		dueDate, perr := time.Parse(time.RFC3339, dueDateStr)
		if perr != nil {
			return nil, fmt.Errorf("invalid due_date format: %v", perr)
		}
		currentTask.DueDate = &dueDate
	}

	if priority, ok := getOptionalStringParam(args, "priority"); ok {
		currentTask.Priority = &priority
	}

	if isComplete, ok := args["is_complete"].(bool); ok {
		currentTask.IsComplete = isComplete
	}

	if cardPK, ok, _ := getOptionalIntParam(args, "card_pk"); ok {
		currentTask.CardPK = cardPK
	}

	_, uerr := task.UpdateTask(ctx.DB, ctx.UserID, taskID, currentTask)
	if uerr != nil {
		return nil, fmt.Errorf("failed to update task: %v", uerr)
	}

	updatedTask, uerr := task.GetTask(ctx.DB, ctx.UserID, taskID)
	if uerr != nil {
		return nil, fmt.Errorf("failed to retrieve updated task: %v", uerr)
	}

	return StructToMap(updatedTask), nil
}

func handleGetTaskByIDV2(args map[string]interface{}, ctx *ToolContext) (map[string]interface{}, error) {
	taskID, err := getIntParam(args, "task_id")
	if err != nil {
		return nil, err
	}

	task, lerr := task.GetTask(ctx.DB, ctx.UserID, taskID)
	if lerr != nil {
		return nil, fmt.Errorf("failed to get task: %v", lerr)
	}

	return StructToMap(task), nil
}

func handleCompleteTaskV2(args map[string]interface{}, ctx *ToolContext) (map[string]interface{}, error) {
	taskID, err := getIntParam(args, "task_id")
	if err != nil {
		return nil, err
	}

	currentTask, lerr := task.GetTask(ctx.DB, ctx.UserID, taskID)
	if lerr != nil {
		return nil, fmt.Errorf("failed to get task: %v", lerr)
	}

	currentTask.IsComplete = true

	_, uerr := task.UpdateTask(ctx.DB, ctx.UserID, taskID, currentTask)
	if uerr != nil {
		return nil, fmt.Errorf("failed to complete task: %v", uerr)
	}

	updatedTask, uerr := task.GetTask(ctx.DB, ctx.UserID, taskID)
	if uerr != nil {
		return nil, fmt.Errorf("failed to retrieve updated task: %v", uerr)
	}

	return StructToMap(updatedTask), nil
}

func handleDeleteTaskV2(args map[string]interface{}, ctx *ToolContext) (map[string]interface{}, error) {
	taskID, err := getIntParam(args, "task_id")
	if err != nil {
		return nil, err
	}

	err = task.DeleteTask(ctx.DB, ctx.UserID, taskID)
	if err != nil {
		return nil, fmt.Errorf("failed to delete task: %v", err)
	}

	return map[string]interface{}{
		"status":  "deleted",
		"task_id": taskID,
	}, nil
}

func handleCompleteAndScheduleTaskV2(args map[string]interface{}, ctx *ToolContext) (map[string]interface{}, error) {
	taskID, err := getIntParam(args, "task_id")
	if err != nil {
		return nil, err
	}

	days, err := getIntParam(args, "days")
	if err != nil {
		return nil, err
	}

	if days <= 0 {
		return nil, fmt.Errorf("days must be greater than 0")
	}

	// Get the complete and default status names
	completeStatus, lerr := task.GetCompleteTaskStatus(ctx.DB, ctx.UserID)
	if lerr != nil {
		return nil, fmt.Errorf("failed to get complete status: %v", lerr)
	}

	defaultStatus, lerr := task.GetDefaultTaskStatus(ctx.DB, ctx.UserID)
	if lerr != nil {
		return nil, fmt.Errorf("failed to get default status: %v", lerr)
	}

	newTaskID, lerr := task.CompleteAndScheduleTask(ctx.DB, ctx.UserID, taskID, days, completeStatus.Name, defaultStatus.Name)
	if lerr != nil {
		return nil, fmt.Errorf("failed to complete and schedule task: %v", lerr)
	}

	return map[string]interface{}{
		"status":        "completed_and_scheduled",
		"task_id":       taskID,
		"new_task_id":   newTaskID,
		"scheduled_in":  fmt.Sprintf("%d days", days),
	}, nil
}

// Legacy task tool handlers (kept for backward compatibility)

func handleGetTasksLegacy(args map[string]interface{}, ctx *ToolContext) (map[string]interface{}, error) {
	includeCompleted := getOptionalBoolParam(args, "include_completed")

	var tasks []models.Task
	var err error

	if cardPKFloat, ok := args["card_pk"].(float64); ok {
		cardPK := int(cardPKFloat)
		tasks, err = task.GetTasksByCard(ctx.DB, ctx.UserID, cardPK)
	} else {
		tasks, err = task.GetTasks(ctx.DB, ctx.UserID, includeCompleted, "UTC")
	}

	if err != nil {
		return nil, fmt.Errorf("failed to get tasks: %v", err)
	}

	var results []map[string]interface{}
	for _, t := range tasks {
		results = append(results, StructToMap(t))
	}

	return map[string]interface{}{
		"tasks": results,
		"total": len(tasks),
	}, nil
}

func handleCreateTaskLegacy(args map[string]interface{}, ctx *ToolContext) (map[string]interface{}, error) {
	title, err := getStringParam(args, "title")
	if err != nil {
		return nil, err
	}

	newTaskModel := models.Task{
		UserID:     ctx.UserID,
		Title:      title,
		IsComplete: false,
	}

	if scheduledDateStr, ok := getOptionalStringParam(args, "scheduled_date"); ok {
		scheduledDate, perr := time.Parse(time.RFC3339, scheduledDateStr)
		if perr != nil {
			return nil, fmt.Errorf("invalid scheduled_date format: %v", perr)
		}
		newTaskModel.ScheduledDate = &scheduledDate
	}

	if dueDateStr, ok := getOptionalStringParam(args, "due_date"); ok {
		dueDate, perr := time.Parse(time.RFC3339, dueDateStr)
		if perr != nil {
			return nil, fmt.Errorf("invalid due_date format: %v", perr)
		}
		newTaskModel.DueDate = &dueDate
	} else {
		now := time.Now()
		newTaskModel.DueDate = &now
	}

	if priority, ok := getOptionalStringParam(args, "priority"); ok {
		newTaskModel.Priority = &priority
	}

	if cardPK, ok, _ := getOptionalIntParam(args, "card_pk"); ok {
		newTaskModel.CardPK = cardPK
	}

	taskID, err := task.CreateTask(ctx.DB, newTaskModel)
	if err != nil {
		return nil, fmt.Errorf("failed to create task: %v", err)
	}

	newTask, err := task.GetTask(ctx.DB, ctx.UserID, taskID)
	if err != nil {
		return nil, fmt.Errorf("failed to retrieve created task: %v", err)
	}

	return StructToMap(newTask), nil
}

func handleUpdateTaskLegacy(args map[string]interface{}, ctx *ToolContext) (map[string]interface{}, error) {
	taskID, err := getIntParam(args, "task_id")
	if err != nil {
		return nil, err
	}

	currentTask, lerr := task.GetTask(ctx.DB, ctx.UserID, taskID)
	if lerr != nil {
		return nil, fmt.Errorf("failed to get task: %v", lerr)
	}

	if title, ok := getOptionalStringParam(args, "title"); ok {
		currentTask.Title = title
	}

	if scheduledDateStr, ok := getOptionalStringParam(args, "scheduled_date"); ok {
		scheduledDate, perr := time.Parse(time.RFC3339, scheduledDateStr)
		if perr != nil {
			return nil, fmt.Errorf("invalid scheduled_date format: %v", perr)
		}
		currentTask.ScheduledDate = &scheduledDate
	}

	if dueDateStr, ok := getOptionalStringParam(args, "due_date"); ok {
		dueDate, perr := time.Parse(time.RFC3339, dueDateStr)
		if perr != nil {
			return nil, fmt.Errorf("invalid due_date format: %v", perr)
		}
		currentTask.DueDate = &dueDate
	}

	if priority, ok := getOptionalStringParam(args, "priority"); ok {
		currentTask.Priority = &priority
	}

	if isComplete, ok := args["is_complete"].(bool); ok {
		currentTask.IsComplete = isComplete
	}

	if cardPK, ok, _ := getOptionalIntParam(args, "card_pk"); ok {
		currentTask.CardPK = cardPK
	}

	_, uerr := task.UpdateTask(ctx.DB, ctx.UserID, taskID, currentTask)
	if uerr != nil {
		return nil, fmt.Errorf("failed to update task: %v", uerr)
	}

	updatedTask, uerr := task.GetTask(ctx.DB, ctx.UserID, taskID)
	if uerr != nil {
		return nil, fmt.Errorf("failed to retrieve updated task: %v", uerr)
	}

	return StructToMap(updatedTask), nil
}

func handleGetTaskByIDLegacy(args map[string]interface{}, ctx *ToolContext) (map[string]interface{}, error) {
	taskID, err := getIntParam(args, "task_id")
	if err != nil {
		return nil, err
	}

	task, lerr := task.GetTask(ctx.DB, ctx.UserID, taskID)
	if lerr != nil {
		return nil, fmt.Errorf("failed to get task: %v", lerr)
	}

	return StructToMap(task), nil
}

func handleCompleteTaskLegacy(args map[string]interface{}, ctx *ToolContext) (map[string]interface{}, error) {
	taskID, err := getIntParam(args, "task_id")
	if err != nil {
		return nil, err
	}

	currentTask, lerr := task.GetTask(ctx.DB, ctx.UserID, taskID)
	if lerr != nil {
		return nil, fmt.Errorf("failed to get task: %v", lerr)
	}

	currentTask.IsComplete = true

	_, uerr := task.UpdateTask(ctx.DB, ctx.UserID, taskID, currentTask)
	if uerr != nil {
		return nil, fmt.Errorf("failed to complete task: %v", uerr)
	}

	updatedTask, uerr := task.GetTask(ctx.DB, ctx.UserID, taskID)
	if uerr != nil {
		return nil, fmt.Errorf("failed to retrieve updated task: %v", uerr)
	}

	return StructToMap(updatedTask), nil
}

func handleDeleteTaskLegacy(args map[string]interface{}, ctx *ToolContext) (map[string]interface{}, error) {
	taskID, err := getIntParam(args, "task_id")
	if err != nil {
		return nil, err
	}

	err = task.DeleteTask(ctx.DB, ctx.UserID, taskID)
	if err != nil {
		return nil, fmt.Errorf("failed to delete task: %v", err)
	}

	return map[string]interface{}{
		"status":  "deleted",
		"task_id": taskID,
	}, nil
}

func handleCompleteAndScheduleTaskLegacy(args map[string]interface{}, ctx *ToolContext) (map[string]interface{}, error) {
	taskID, err := getIntParam(args, "task_id")
	if err != nil {
		return nil, err
	}

	days, err := getIntParam(args, "days")
	if err != nil {
		return nil, err
	}

	if days <= 0 {
		return nil, fmt.Errorf("days must be greater than 0")
	}

	// Get the complete and default status names
	completeStatus, lerr := task.GetCompleteTaskStatus(ctx.DB, ctx.UserID)
	if lerr != nil {
		return nil, fmt.Errorf("failed to get complete status: %v", lerr)
	}

	defaultStatus, lerr := task.GetDefaultTaskStatus(ctx.DB, ctx.UserID)
	if lerr != nil {
		return nil, fmt.Errorf("failed to get default status: %v", lerr)
	}

	newTaskID, lerr := task.CompleteAndScheduleTask(ctx.DB, ctx.UserID, taskID, days, completeStatus.Name, defaultStatus.Name)
	if lerr != nil {
		return nil, fmt.Errorf("failed to complete and schedule task: %v", lerr)
	}

	return map[string]interface{}{
		"status":        "completed_and_scheduled",
		"task_id":       taskID,
		"new_task_id":   newTaskID,
		"scheduled_in":  fmt.Sprintf("%d days", days),
	}, nil
}
