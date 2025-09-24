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

type Task struct {
	ID            int         `json:"id"`
	CardPK        int         `json:"card_pk"`
	UserID        int         `json:"user_id"`
	ScheduledDate *time.Time  `json:"scheduled_date"`
	DueDate       *time.Time  `json:"due_date"`
	CreatedAt     time.Time   `json:"created_at"`
	UpdatedAt     time.Time   `json:"updated_at"`
	CompletedAt   *time.Time  `json:"completed_at"`
	Title         string      `json:"title"`
	Priority      *string     `json:"priority"`
	IsComplete    bool        `json:"is_complete"`
	IsDeleted     bool        `json:"is_deleted"`
	Card          PartialCard `json:"card"`
	Tags          []Tag       `json:"tags"`
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
