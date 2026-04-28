package models

import (
	"database/sql"
	"encoding/json"
	"time"
)

type NullTime struct {
	sql.NullTime
}

// UnmarshalJSON custom unmarshals a NullTime from a JSON string
func (nt *NullTime) UnmarshalJSON(b []byte) error {
	// Define the date format used in your JSON
	const layout = "2006-01-02T15:04:05.999999Z" // Include fractional seconds in layout
	var s string
	if err := json.Unmarshal(b, &s); err != nil {
		return err
	}

	if s == "" || s == "null" {
		nt.Valid = false
		return nil
	}

	parsedTime, err := time.Parse(layout, s)
	if err != nil {
		return err
	}

	nt.Time = parsedTime
	nt.Valid = true
	return nil
}

// MarshalJSON custom marshals a NullTime to a JSON string
func (nt NullTime) MarshalJSON() ([]byte, error) {
	if !nt.Valid {
		return json.Marshal(nil)
	}
	return json.Marshal(nt.Time.Format("2006-01-02T15:04:05.999999Z")) // Include fractional seconds in format

}

type PartialTask struct {
	ID         int    `json:"id"`
	Title      string `json:"title"`
	IsComplete bool   `json:"is_complete"`
	Status     string `json:"status"`
}

type Task struct {
	ID            int           `json:"id"`
	CardPK        int           `json:"card_pk"`
	UserID        int           `json:"user_id"`
	ScheduledDate *time.Time    `json:"scheduled_date"`
	DueDate       *time.Time    `json:"due_date"`
	CreatedAt     time.Time     `json:"created_at"`
	UpdatedAt     time.Time     `json:"updated_at"`
	CompletedAt   *time.Time    `json:"completed_at"`
	Title         string        `json:"title"`
	Description   *string       `json:"description"`
	Priority      *string       `json:"priority"`
	Status        string        `json:"status"`
	IsComplete    bool          `json:"is_complete"`
	IsDeleted     bool          `json:"is_deleted"`
	ReminderTime  *time.Time    `json:"reminder_time"`
	ReminderSent  bool          `json:"reminder_sent"`
	Card          PartialCard   `json:"card"`
	Tags          []Tag         `json:"tags"`
	BlockedBy     []PartialTask `json:"blocked_by"`
	Blocks        []PartialTask `json:"blocks"`
	SortOrder     *int          `json:"sort_order,omitempty"`
	ParentTaskID  *int          `json:"parent_task_id,omitempty"`
	Subtasks      []Task        `json:"subtasks,omitempty"`
	ParentTitle   string        `json:"parent_title,omitempty"`
}

type RecurringTask struct {
	TaskID    int
	Frequency string
	Interval  int
	DayOfWeek int
}

type TasksResponse struct {
	Tasks  []Task `json:"tasks"`
	Total  int    `json:"total"`
	Limit  int    `json:"limit"`
	Offset int    `json:"offset"`
}

type TaskStatus struct {
	ID              int       `json:"id"`
	UserID          int       `json:"user_id"`
	Name            string    `json:"name"`
	DisplayName     string    `json:"display_name"`
	Color           string    `json:"color"`
	Icon            string    `json:"icon"`
	Position        int       `json:"position"`
	IsDefault       bool      `json:"is_default"`
	IsCompleteState bool      `json:"is_complete_state"`
	CreatedAt       time.Time `json:"created_at"`
	UpdatedAt       time.Time `json:"updated_at"`
}

type CreateTaskStatusParams struct {
	Name            string `json:"name"`
	DisplayName     string `json:"display_name"`
	Color           string `json:"color"`
	Icon            string `json:"icon"`
	Position        int    `json:"position"`
	IsDefault       bool   `json:"is_default"`
	IsCompleteState bool   `json:"is_complete_state"`
}

type UpdateTaskStatusParams struct {
	DisplayName     *string `json:"display_name"`
	Color           *string `json:"color"`
	Icon            *string `json:"icon"`
	Position        *int    `json:"position"`
	IsDefault       *bool   `json:"is_default"`
	IsCompleteState *bool   `json:"is_complete_state"`
}

type ReorderTaskStatusesParams struct {
	StatusIDs []int `json:"status_ids"` // Array of status IDs in desired order
}
