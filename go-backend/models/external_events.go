package models

import "time"

// ExternalCalendar represents a subscription to an external iCal feed
type ExternalCalendar struct {
	ID                int        `json:"id"`
	UserID            int        `json:"user_id"`
	Name              string     `json:"name"`
	URL               string     `json:"url"`
	SyncEnabled       bool       `json:"sync_enabled"`
	SyncIntervalHours int        `json:"sync_interval_hours"`
	Color             string     `json:"color"`
	LastSyncedAt      *time.Time `json:"last_synced_at,omitempty"`
	LastError         *string    `json:"last_error,omitempty"`
	CreatedAt         time.Time  `json:"created_at"`
	UpdatedAt         time.Time  `json:"updated_at"`
}

// CreateExternalCalendarRequest is used to create a new calendar subscription
type CreateExternalCalendarRequest struct {
	Name  string `json:"name"`
	URL   string `json:"url"`
	Color string `json:"color"`
}

// UpdateExternalCalendarRequest is used to update an existing calendar subscription
type UpdateExternalCalendarRequest struct {
	Name              *string `json:"name,omitempty"`
	URL               *string `json:"url,omitempty"`
	Color             *string `json:"color,omitempty"`
	SyncEnabled       *bool   `json:"sync_enabled,omitempty"`
	SyncIntervalHours *int    `json:"sync_interval_hours,omitempty"`
}

// ExternalEvent represents an imported calendar event from an external feed
type ExternalEvent struct {
	ID                 int        `json:"id"`
	UserID             int        `json:"user_id"`
	ExternalCalendarID *int       `json:"external_calendar_id,omitempty"`
	Title              string     `json:"title"`
	Description        *string    `json:"description,omitempty"`
	StartTime          time.Time  `json:"start_time"`
	EndTime            time.Time  `json:"end_time"`
	AllDay             bool       `json:"all_day"`
	Location           *string    `json:"location,omitempty"`
	ExternalUID        *string    `json:"external_uid,omitempty"`
	ExternalURL        *string    `json:"external_url,omitempty"`
	RecurrenceRule     *string    `json:"recurrence_rule,omitempty"`
	Color              *string    `json:"color,omitempty"`
	CreatedAt          time.Time  `json:"created_at"`
	UpdatedAt          time.Time  `json:"updated_at"`
	LastSyncedAt       *time.Time `json:"last_synced_at,omitempty"`
}

// ExternalEventsResponse is the API response for listing external events
type ExternalEventsResponse struct {
	Events []ExternalEvent `json:"events"`
	Total  int             `json:"total"`
}
