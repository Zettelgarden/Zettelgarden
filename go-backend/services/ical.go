package services

import (
	"bufio"
	"fmt"
	"io"
	"net"
	"net/http"
	"net/url"
	"strings"
	"time"
)

const (
	// Max events to import from a single feed
	MaxEventsPerFeed = 2000
	// Max todos to import from a single feed
	MaxTodosPerFeed = 500
)

// ICalEvent represents a parsed VEVENT from iCal format
type ICalEvent struct {
	UID            string
	DTStart        time.Time
	DTEnd          time.Time
	Summary        string
	Description    string
	Location       string
	AllDay         bool
	RecurrenceRule string
	URL            string
	TZID           string // Timezone ID from the event
}

// ICalTodo represents a parsed VTODO from iCal format
type ICalTodo struct {
	UID             string
	Summary         string
	Description     string
	Due             time.Time
	DTStart         time.Time // Scheduled date (DTSTART maps to scheduled_date)
	Status          string    // NEEDS-ACTION, IN-PROCESS, COMPLETED, CANCELLED
	Priority        int       // 0-9 (0=undefined, 1=highest, 9=lowest)
	PercentComplete int       // 0-100
	Completed       time.Time // COMPLETED timestamp
}

// ParseICalendar parses an iCal feed and returns VEVENT components
func ParseICalendar(r io.Reader) ([]ICalEvent, error) {
	events, _, err := ParseICalendarWithTodos(r)
	return events, err
}

// ParseICalendarWithTodos parses an iCal feed and returns both VEVENT and VTODO components
func ParseICalendarWithTodos(r io.Reader) ([]ICalEvent, []ICalTodo, error) {
	scanner := bufio.NewScanner(r)
	var events []ICalEvent
	var todos []ICalTodo
	var currentEvent *ICalEvent
	var currentTodo *ICalTodo
	var inVEVENT bool
	var inVTODO bool

	// Increase buffer size for long lines
	scanner.Buffer(make([]byte, 0, 256*1024), 1024*1024)

	for scanner.Scan() {
		line := scanner.Text()
		line = strings.TrimSpace(line)

		switch {
		case line == "BEGIN:VEVENT":
			inVEVENT = true
			currentEvent = &ICalEvent{}

		case line == "END:VEVENT":
			if currentEvent != nil && currentEvent.UID != "" {
				if len(events) >= MaxEventsPerFeed {
					return events, todos, fmt.Errorf("exceeded maximum events per feed (%d)", MaxEventsPerFeed)
				}
				events = append(events, *currentEvent)
			}
			inVEVENT = false
			currentEvent = nil

		case inVEVENT && currentEvent != nil:
			parseEventProperty(currentEvent, line)

		case line == "BEGIN:VTODO":
			inVTODO = true
			currentTodo = &ICalTodo{}

		case line == "END:VTODO":
			if currentTodo != nil && currentTodo.UID != "" {
				if len(todos) >= MaxTodosPerFeed {
					return events, todos, fmt.Errorf("exceeded maximum todos per feed (%d)", MaxTodosPerFeed)
				}
				todos = append(todos, *currentTodo)
			}
			inVTODO = false
			currentTodo = nil

		case inVTODO && currentTodo != nil:
			parseTodoProperty(currentTodo, line)
		}
	}

	return events, todos, scanner.Err()
}

// parseEventProperty parses a single iCal property line into an event
func parseEventProperty(event *ICalEvent, line string) {
	// Handle line continuations (lines starting with space/tabs continue previous property)
	for strings.HasPrefix(line, " ") || strings.HasPrefix(line, "\t") {
		line = line[1:]
	}

	// Split on first colon to separate key from value
	parts := strings.SplitN(line, ":", 2)
	if len(parts) < 2 {
		return
	}

	key := parts[0]
	value := parts[1]

	// Handle parameters (e.g., DTSTART;VALUE=DATE:20230101)
	keyParts := strings.Split(key, ";")
	key = keyParts[0]

	switch key {
	case "UID":
		event.UID = value
	case "DTSTART":
		event.DTStart = parseICalDateTime(value, keyParts)
	case "DTEND":
		event.DTEnd = parseICalDateTime(value, keyParts)
	case "SUMMARY":
		event.Summary = unescapeICalText(value)
	case "DESCRIPTION":
		event.Description = unescapeICalText(value)
	case "LOCATION":
		event.Location = unescapeICalText(value)
	case "RRULE":
		event.RecurrenceRule = value
	case "URL":
		event.URL = value
	case "TZID":
		// TZID may come as a separate property in some feeds
		event.TZID = value
	}
}

// parseTodoProperty parses a single iCal property line into a todo
func parseTodoProperty(todo *ICalTodo, line string) {
	// Handle line continuations (lines starting with space/tabs continue previous property)
	for strings.HasPrefix(line, " ") || strings.HasPrefix(line, "\t") {
		line = line[1:]
	}

	// Split on first colon to separate key from value
	parts := strings.SplitN(line, ":", 2)
	if len(parts) < 2 {
		return
	}

	key := parts[0]
	value := parts[1]

	// Handle parameters (e.g., DTSTART;VALUE=DATE:20230101)
	keyParts := strings.Split(key, ";")
	key = keyParts[0]

	switch key {
	case "UID":
		todo.UID = value
	case "SUMMARY":
		todo.Summary = unescapeICalText(value)
	case "DESCRIPTION":
		todo.Description = unescapeICalText(value)
	case "DTSTART":
		todo.DTStart = parseICalDateTime(value, keyParts)
	case "DUE":
		todo.Due = parseICalDateTime(value, keyParts)
	case "STATUS":
		todo.Status = value
	case "PRIORITY":
		// PRIORITY is 0-9 (0=undefined, 1=highest, 9=lowest)
		if pri := parseICalIntSafe(value, 0); pri >= 0 && pri <= 9 {
			todo.Priority = pri
		}
	case "PERCENT-COMPLETE":
		todo.PercentComplete = parseICalIntSafe(value, 0)
	case "COMPLETED":
		// COMPLETED is a datetime stamp
		todo.Completed = parseICalDateTime(value, keyParts)
	}
}

// parseICalDateTime parses an iCal datetime string into time.Time
// Handles both DATE (all-day) and DATETIME formats with timezone support
func parseICalDateTime(value string, params []string) time.Time {
	// Check if it's a DATE (all-day) or DATETIME
	isDate := false
	var tzid string

	for _, p := range params {
		if strings.Contains(p, "VALUE=DATE") {
			isDate = true
		}
		if tz, ok := strings.CutPrefix(p, "TZID="); ok {
			tzid = tz
		}
	}

	// Parse based on format
	if isDate || len(value) == 8 {
		// DATE format: 20230101
		t, err := time.Parse("20060102", value)
		if err == nil {
			return t.UTC()
		}
	} else {
		// DATETIME format: 20230101T120000Z or 20230101T120000
		layouts := []string{
			"20060102T150405Z", // UTC time with Z suffix
			"20060102T150405",  // Local time (no Z)
		}

		for _, layout := range layouts {
			if t, err := time.Parse(layout, value); err == nil {
				// If we have a TZID, parse in that timezone
				if tzid != "" {
					loc, err := time.LoadLocation(tzid)
					if err == nil {
						// Re-parse with timezone location
						if t, err := time.ParseInLocation(layout, value, loc); err == nil {
							return t.UTC()
						}
					}
					// If TZID lookup fails, fall through to UTC
				}
				return t.UTC()
			}
		}
	}

	// Return zero time if parsing failed
	return time.Time{}
}

// parseICalIntSafe parses an integer from an iCal value string, returning defaultValue on failure
func parseICalIntSafe(s string, defaultValue int) int {
	var result int
	_, err := fmt.Sscanf(s, "%d", &result)
	if err != nil {
		return defaultValue
	}
	return result
}

// FetchICalURL fetches an iCal feed from a URL with optional Basic Authentication
// Returns the parsed events from the feed
func FetchICalURL(feedURL, username, password string) ([]ICalEvent, error) {
	events, _, err := FetchICalURLWithTodos(feedURL, username, password)
	return events, err
}

// FetchICalURLWithTodos fetches an iCal feed from a URL with optional Basic Authentication
// Returns both parsed VEVENT and VTODO components from the feed
func FetchICalURLWithTodos(feedURL, username, password string) ([]ICalEvent, []ICalTodo, error) {
	client := &http.Client{
		Timeout: 30 * time.Second,
	}

	req, err := http.NewRequest("GET", feedURL, nil)
	if err != nil {
		return nil, nil, fmt.Errorf("failed to create request: %w", err)
	}

	// Add Basic Auth if credentials provided
	if username != "" && password != "" {
		req.SetBasicAuth(username, password)
	}

	resp, err := client.Do(req)
	if err != nil {
		return nil, nil, fmt.Errorf("failed to fetch iCal feed: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return nil, nil, fmt.Errorf("iCal feed returned status %d", resp.StatusCode)
	}

	// Check content-length header to prevent memory exhaustion
	if resp.ContentLength > MaxICalFeedSizeBytes {
		return nil, nil, fmt.Errorf("iCal feed too large (%d bytes, max %d bytes)", resp.ContentLength, MaxICalFeedSizeBytes)
	}

	// Check content type
	contentType := resp.Header.Get("Content-Type")
	if !strings.Contains(contentType, "text/calendar") && !strings.Contains(contentType, "text/plain") {
		// Don't fail on wrong content type, just log it
		// Some servers return incorrect content types
	}

	// Limit the reader to prevent memory exhaustion for feeds without content-length header
	limitedReader := io.LimitReader(resp.Body, MaxICalFeedSizeBytes)

	events, todos, err := ParseICalendarWithTodos(limitedReader)
	if err != nil {
		return nil, nil, fmt.Errorf("failed to parse iCal feed: %w", err)
	}

	return events, todos, nil
}

// ValidatePublicURL validates that a URL is safe to fetch (SSRF protection)
// Blocks private/internal IP addresses and localhost
// Exported for use by other services (e.g., CalDAV)
func ValidatePublicURL(rawURL string) error {
	parsedURL, err := url.Parse(rawURL)
	if err != nil {
		return fmt.Errorf("invalid URL: %w", err)
	}

	// Block non-http/https schemes (should already be checked, but be safe)
	if parsedURL.Scheme != "http" && parsedURL.Scheme != "https" {
		return fmt.Errorf("unsupported URL scheme")
	}

	host := parsedURL.Hostname()
	if host == "" {
		return fmt.Errorf("invalid URL host")
	}

	// Check if host is an IP address
	ip := net.ParseIP(host)
	if ip != nil {
		// Block private IPs (RFC1918)
		if ip.IsPrivate() {
			return fmt.Errorf("private IP addresses are not allowed")
		}
		// Block loopback addresses
		if ip.IsLoopback() {
			return fmt.Errorf("loopback addresses are not allowed")
		}
		// Block link-local addresses
		if ip.IsLinkLocalUnicast() {
			return fmt.Errorf("link-local addresses are not allowed")
		}
		// Block reserved addresses
		if ip.IsInterfaceLocalMulticast() || ip.IsLinkLocalMulticast() {
			return fmt.Errorf("reserved addresses are not allowed")
		}
	} else {
		// For domain names, check for common internal/internal-like domains
		lowerHost := strings.ToLower(host)
		internalDomains := []string{
			"localhost",
			"local",
			"localhost.localdomain",
			".local", // mDNS/Bonjour
			"0.0.0.0",
		}
		for _, internal := range internalDomains {
			if lowerHost == internal || strings.HasSuffix(lowerHost, "."+internal) {
				return fmt.Errorf("internal domain names are not allowed")
			}
		}
	}

	return nil
}

// ValidateICalURL checks if a URL is a valid iCal feed with optional Basic Authentication
// Attempts to fetch and parse the feed to verify it works
func ValidateICalURL(rawURL, username, password string) error {
	if rawURL == "" {
		return fmt.Errorf("URL is empty")
	}

	if !strings.HasPrefix(rawURL, "http://") && !strings.HasPrefix(rawURL, "https://") {
		return fmt.Errorf("URL must start with http:// or https://")
	}

	// Validate URL to prevent SSRF attacks
	if err := ValidatePublicURL(rawURL); err != nil {
		return err
	}

	// Try to fetch and parse
	events, todos, err := FetchICalURLWithTodos(rawURL, username, password)
	if err != nil {
		return err
	}

	// Should have at least one event or todo to be considered valid
	if len(events) == 0 && len(todos) == 0 {
		return fmt.Errorf("feed contains no events or todos")
	}

	return nil
}
