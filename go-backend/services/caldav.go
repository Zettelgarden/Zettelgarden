package services

import (
	"database/sql"
	"fmt"
	"io"
	"log"
	"net/http"
	"strings"
	"time"

	"go-backend/models"
)

// CalDAV sync service for syncing tasks with external CalDAV servers
// This service provides functionality to:
// 1. Export tasks as VTODO/iCalendar events to a CalDAV server
// 2. Import events from a CalDAV server and create/update tasks
// 3. Bi-directional sync between Zettelgarden tasks and CalDAV events

const (
	// iCalendar/VTODO component constants
	ICalBeginVCalendar    = "BEGIN:VCALENDAR"
	ICalEndVCalendar      = "END:VCALENDAR"
	ICalBeginVTodo        = "BEGIN:VTODO"
	ICalEndVTodo          = "END:VTODO"
	ICalVersion           = "VERSION:2.0"
	ICalProdID            = "PRODID:-//Zettelgarden//Task Calendar//EN"
	ICalCalScale          = "CALSCALE:GREGORIAN"
	ICalMethod            = "METHOD:PUBLISH"
	ICalUID               = "UID"
	ICalDTStamp            = "DTSTAMP"
	ICalCreated            = "CREATED"
	ICalLastModified       = "LAST-MODIFIED"
	ICalDue                = "DUE;VALUE=DATE"
	ICalStartDate          = "DTSTART;VALUE=DATE"
	ICalSummary            = "SUMMARY"
	ICalDescription        = "DESCRIPTION"
	ICalPriority           = "PRIORITY"
	ICalStatus             = "STATUS"
	ICalPercentComplete    = "PERCENT-COMPLETE"
	ICalCategories         = "CATEGORIES"
)

// TaskStatus values
const (
	StatusNeedsAction = "NEEDS-ACTION"
	StatusInProgress  = "IN-PROCESS"
	StatusCompleted   = "COMPLETED"
	StatusCancelled   = "CANCELLED"
)

// Priority mapping (1-9, where 1 is highest priority)
// Zettelgarden uses A-D, mapping to CalDAV priorities:
// A -> 1 (high), B -> 5 (medium), C -> 7 (low), D -> 9 (none)
func mapPriorityToCalDAV(priority string) string {
	switch strings.ToUpper(priority) {
	case "A":
		return "1"
	case "B":
		return "5"
	case "C":
		return "7"
	case "D":
		return "9"
	default:
		return "0" // undefined
	}
}

func mapPriorityFromCalDAV(priority string) string {
	switch priority {
	case "1", "2", "3":
		return "A"
	case "4", "5", "6":
		return "B"
	case "7":
		return "C"
	case "8", "9":
		return "D"
	default:
		return ""
	}
}

func mapStatusToCalDAV(status string, isComplete bool) string {
	if isComplete {
		return StatusCompleted
	}
	switch status {
	case "todo":
		return StatusNeedsAction
	case "in_progress":
		return StatusInProgress
	case "blocked":
		return StatusNeedsAction
	case "done":
		return StatusCompleted
	default:
		return StatusNeedsAction
	}
}

func mapStatusFromCalDAV(calDAVStatus string) (string, bool) {
	switch calDAVStatus {
	case StatusNeedsAction:
		return "todo", false
	case StatusInProgress:
		return "in_progress", false
	case StatusCompleted:
		return "done", true
	case StatusCancelled:
		return "todo", false
	default:
		return "todo", false
	}
}

// formatICalDate formats a time.Time to iCalendar DATE format (YYYYMMDD)
func formatICalDate(t time.Time) string {
	return t.Format("20060102")
}

// formatICalDateTime formats a time.Time to iCalendar DATETIME format (YYYYMMDDTHHmmssZ)
func formatICalDateTime(t time.Time) string {
	return t.UTC().Format("20060102T150405Z")
}

// CalDAVService handles CalDAV operations
type CalDAVService struct {
	db *sql.DB
}

// NewCalDAVService creates a new CalDAV service instance
func NewCalDAVService(db *sql.DB) *CalDAVService {
	return &CalDAVService{db: db}
}

// ExportTasksToICalendar exports tasks as iCalendar VTODO format
func (s *CalDAVService) ExportTasksToICalendar(tasks []models.Task) (string, error) {
	var lines []string

	lines = append(lines, ICalBeginVCalendar)
	lines = append(lines, ICalVersion)
	lines = append(lines, ICalProdID)
	lines = append(lines, ICalCalScale)
	lines = append(lines, ICalMethod)

	for _, task := range tasks {
		vtodo, err := s.taskToVTodo(task)
		if err != nil {
			log.Printf("Error converting task %d to VTODO: %v", task.ID, err)
			continue
		}
		lines = append(lines, vtodo...)
	}

	lines = append(lines, ICalEndVCalendar)

	return strings.Join(lines, "\r\n"), nil
}

// taskToVTodo converts a task to VTODO iCalendar format
func (s *CalDAVService) taskToVTodo(task models.Task) ([]string, error) {
	var lines []string

	lines = append(lines, ICalBeginVTodo)

	// UID - use task ID with domain prefix
	lines = append(lines, fmt.Sprintf("%s:zettelgarden-task-%d@example.com", ICalUID, task.ID))

	// DTSTAMP - current timestamp
	lines = append(lines, fmt.Sprintf("%s:%s", ICalDTStamp, formatICalDateTime(time.Now())))

	// Created
	if !task.CreatedAt.IsZero() {
		lines = append(lines, fmt.Sprintf("%s:%s", ICalCreated, formatICalDateTime(task.CreatedAt)))
	}

	// Last Modified
	if !task.UpdatedAt.IsZero() {
		lines = append(lines, fmt.Sprintf("%s:%s", ICalLastModified, formatICalDateTime(task.UpdatedAt)))
	}

	// Summary (title)
	if task.Title != "" {
		lines = append(lines, fmt.Sprintf("%s:%s", ICalSummary, escapeICalText(task.Title)))
	}

	// Description
	if task.Description != nil && *task.Description != "" {
		lines = append(lines, fmt.Sprintf("%s:%s", ICalDescription, escapeICalText(*task.Description)))
	}

	// Priority
	if task.Priority != nil && *task.Priority != "" {
		lines = append(lines, fmt.Sprintf("%s:%s", ICalPriority, mapPriorityToCalDAV(*task.Priority)))
	}

	// Status
	status := mapStatusToCalDAV(task.Status, task.IsComplete)
	lines = append(lines, fmt.Sprintf("%s:%s", ICalStatus, status))

	// Percent Complete
	if task.IsComplete {
		lines = append(lines, fmt.Sprintf("%s:100", ICalPercentComplete))
	} else {
		lines = append(lines, fmt.Sprintf("%s:0", ICalPercentComplete))
	}

	// Due Date
	if task.DueDate != nil {
		dueDate := time.Time(*task.DueDate)
		lines = append(lines, fmt.Sprintf("%s:%s", ICalDue, formatICalDate(dueDate)))
	}

	// Start Date (use scheduled date if available)
	if task.ScheduledDate != nil {
		startDate := time.Time(*task.ScheduledDate)
		lines = append(lines, fmt.Sprintf("%s:%s", ICalStartDate, formatICalDate(startDate)))
	}

	// Categories (tags)
	if len(task.Tags) > 0 {
		var tags []string
		for _, tag := range task.Tags {
			tags = append(tags, tag.Name)
		}
		lines = append(lines, fmt.Sprintf("%s:%s", ICalCategories, strings.Join(tags, ",")))
	}

	lines = append(lines, ICalEndVTodo)

	return lines, nil
}

// escapeICalText escapes special characters in iCalendar text values
func escapeICalText(text string) string {
	text = strings.ReplaceAll(text, "\\", "\\\\")
	text = strings.ReplaceAll(text, ";", "\\;")
	text = strings.ReplaceAll(text, ",", "\\,")
	text = strings.ReplaceAll(text, "\n", "\\n")
	return text
}

// unescapeICalText unescapes special characters in iCalendar text values
func unescapeICalText(text string) string {
	text = strings.ReplaceAll(text, "\\n", "\n")
	text = strings.ReplaceAll(text, "\\,", ",")
	text = strings.ReplaceAll(text, "\\;", ";")
	text = strings.ReplaceAll(text, "\\\\", "\\")
	return text
}

// SyncToCalDAVServer syncs tasks to a CalDAV server
// This is a placeholder for future implementation
// CalDAV sync requires authentication and WebDAV operations (PUT, GET, DELETE, REPORT)
func (s *CalDAVService) SyncToCalDAVServer(caldavURL string, tasks []models.Task) error {
	// Generate iCalendar content
	icalContent, err := s.ExportTasksToICalendar(tasks)
	if err != nil {
		return fmt.Errorf("failed to generate iCalendar: %w", err)
	}

	// Create HTTP request for CalDAV PUT
	// Note: Full CalDAV sync implementation would require:
	// 1. Authentication (Basic auth, OAuth, etc.)
	// 2. WebDAV methods (MKCOL, PUT, GET, DELETE, REPORT)
	// 3. ETag/If-Match header handling for conflict resolution
	// 4. Sync-token support for incremental sync
	// 5. VTODO component filtering

	log.Printf("CalDAV sync to %s: %d bytes of iCalendar data", caldavURL, len(icalContent))

	// Placeholder: This would require full WebDAV client implementation
	// Consider using a Go CalDAV client library like:
	// - github.com/emersion/go-webdav
	// - github.com/lukeshay/caldav-go

	return nil
}

// FetchFromCalDAVServer fetches VTODO items from a CalDAV server
// This is a placeholder for future implementation
func (s *CalDAVService) FetchFromCalDAVServer(caldavURL string) ([]models.Task, error) {
	// Placeholder implementation
	// Full implementation would:
	// 1. Authenticate to CalDAV server
	// 2. Perform WebDAV REPORT request to list VTODO items
	// 3. Parse iCalendar VTODO components
	// 4. Convert VTODO to Task models
	// 5. Handle sync tokens and conflict resolution

	_ = caldavURL // Use the variable

	return []models.Task{}, nil
}

// MakeCalDAVRequest makes an authenticated HTTP request to a CalDAV server
func (s *CalDAVService) MakeCalDAVRequest(method, url string, body io.Reader, username, password string) (*http.Response, error) {
	req, err := http.NewRequest(method, url, body)
	if err != nil {
		return nil, err
	}

	// Set headers for CalDAV/WebDAV
	req.Header.Set("Content-Type", "text/calendar; charset=utf-8")
	req.Header.Set("Depth", "1") // For WebDAV operations

	// Basic authentication
	if username != "" && password != "" {
		req.SetBasicAuth(username, password)
	}

	client := &http.Client{
		Timeout: 30 * time.Second,
	}

	return client.Do(req)
}

// ValidateCalDAVURL validates if a CalDAV URL is accessible
func (s *CalDAVService) ValidateCalDAVURL(caldavURL string) error {
	if caldavURL == "" {
		return fmt.Errorf("CalDAV URL is empty")
	}

	// Basic URL format validation
	if !strings.HasPrefix(caldavURL, "http://") && !strings.HasPrefix(caldavURL, "https://") {
		return fmt.Errorf("CalDAV URL must start with http:// or https://")
	}

	// Try to make a simple GET request to verify accessibility
	resp, err := s.MakeCalDAVRequest("GET", caldavURL, nil, "", "")
	if err != nil {
		return fmt.Errorf("failed to connect to CalDAV server: %w", err)
	}
	defer resp.Body.Close()

	// CalDAV servers should respond with 207 Multi-Status or 200 OK
	if resp.StatusCode != 200 && resp.StatusCode != 207 {
		return fmt.Errorf("CalDAV server returned status %d", resp.StatusCode)
	}

	return nil
}
