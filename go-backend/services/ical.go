package services

import (
	"bufio"
	"fmt"
	"io"
	"net/http"
	"strings"
	"time"
)

const (
	// Max events to import from a single feed
	MaxEventsPerFeed = 2000
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

// ParseICalendar parses an iCal feed and returns VEVENT components
func ParseICalendar(r io.Reader) ([]ICalEvent, error) {
	scanner := bufio.NewScanner(r)
	var events []ICalEvent
	var currentEvent *ICalEvent
	var inVEVENT bool

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
					return events, fmt.Errorf("exceeded maximum events per feed (%d)", MaxEventsPerFeed)
				}
				events = append(events, *currentEvent)
			}
			inVEVENT = false
			currentEvent = nil

		case inVEVENT && currentEvent != nil:
			parseEventProperty(currentEvent, line)
		}
	}

	return events, scanner.Err()
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

// FetchICalURL fetches an iCal feed from a URL
// Returns the parsed events from the feed
func FetchICalURL(url string) ([]ICalEvent, error) {
	client := &http.Client{
		Timeout: 30 * time.Second,
	}

	resp, err := client.Get(url)
	if err != nil {
		return nil, fmt.Errorf("failed to fetch iCal feed: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("iCal feed returned status %d", resp.StatusCode)
	}

	// Check content type
	contentType := resp.Header.Get("Content-Type")
	if !strings.Contains(contentType, "text/calendar") && !strings.Contains(contentType, "text/plain") {
		// Don't fail on wrong content type, just log it
		// Some servers return incorrect content types
	}

	events, err := ParseICalendar(resp.Body)
	if err != nil {
		return nil, fmt.Errorf("failed to parse iCal feed: %w", err)
	}

	return events, nil
}

// ValidateICalURL checks if a URL is a valid iCal feed
// Attempts to fetch and parse the feed to verify it works
func ValidateICalURL(url string) error {
	if url == "" {
		return fmt.Errorf("URL is empty")
	}

	if !strings.HasPrefix(url, "http://") && !strings.HasPrefix(url, "https://") {
		return fmt.Errorf("URL must start with http:// or https://")
	}

	// Try to fetch and parse
	events, err := FetchICalURL(url)
	if err != nil {
		return err
	}

	// Should have at least one event to be considered valid
	if len(events) == 0 {
		return fmt.Errorf("feed contains no events")
	}

	return nil
}
