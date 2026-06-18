package models

import "time"

// TaskSavedSearch is a user-saved task search (filter + sort + view mode)
// that is synced across devices.
type TaskSavedSearch struct {
	ID            int       `json:"id"`
	UserID        int       `json:"user_id"`
	Name          string    `json:"name"`
	FilterString  string    `json:"filter_string"`
	SortField     string    `json:"sort_field"`
	SortDirection string    `json:"sort_direction"`
	ViewMode      string    `json:"view_mode"`
	CreatedAt     time.Time `json:"created_at"`
	UpdatedAt     time.Time `json:"updated_at"`
}

// CreateTaskSavedSearchParams is the request body for creating a saved search.
type CreateTaskSavedSearchParams struct {
	Name          string `json:"name"`
	FilterString  string `json:"filter_string"`
	SortField     string `json:"sort_field"`
	SortDirection string `json:"sort_direction"`
	ViewMode      string `json:"view_mode"`
}

// UpdateTaskSavedSearchParams is the request body for updating a saved search.
// All fields are optional; only provided fields are updated.
type UpdateTaskSavedSearchParams struct {
	Name          *string `json:"name"`
	FilterString  *string `json:"filter_string"`
	SortField     *string `json:"sort_field"`
	SortDirection *string `json:"sort_direction"`
	ViewMode      *string `json:"view_mode"`
}
