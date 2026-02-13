package services

import (
	"fmt"
	"strings"
	"time"

	"github.com/teambition/rrule-go"
)

const (
	// MaxRecurrenceExpansion is the maximum number of occurrences to generate for a recurring event
	MaxRecurrenceExpansion = 365
	// RecurrenceExpansionMonths is how many months into the future to expand recurring events
	RecurrenceExpansionMonths = 12
)

// RecurrenceOccurrence represents a single occurrence of a recurring event
type RecurrenceOccurrence struct {
	StartTime time.Time
	EndTime   time.Time
	Index     int // The occurrence index (0 for the first occurrence)
}

// ExpandRecurrence expands an RRULE string into individual occurrences
// Returns occurrences from the event start time up to the expansion window
func ExpandRecurrence(rruleStr string, startTime, endTime time.Time, allDay bool) ([]RecurrenceOccurrence, error) {
	if rruleStr == "" {
		return nil, fmt.Errorf("empty recurrence rule")
	}

	// Parse the RRULE string into an RRule object
	rule, err := parseRRule(rruleStr, startTime, allDay)
	if err != nil {
		return nil, fmt.Errorf("failed to parse RRULE: %w", err)
	}

	// Calculate the end of the expansion window
	// Use the later of time.Now() or startTime as the base for expansion
	// This ensures tests with future dates work correctly
	baseTime := time.Now()
	if startTime.After(baseTime) {
		baseTime = startTime
	}
	expansionEnd := baseTime.AddDate(0, RecurrenceExpansionMonths, 0)

	// Get all occurrences using All() method, then filter
	// Between() can have issues with boundary conditions
	allOccurrences := rule.All()

	// Filter to occurrences within our window
	// IMPORTANT: Always include event's startTime to ensure first occurrence is included
	// Then filter out any occurrences that ended more than a week ago
	oneWeekAgo := time.Now().AddDate(0, 0, -7).UTC()
	occurrences := make([]time.Time, 0)
	for _, occ := range allOccurrences {
		// Stop if we've gone past our expansion window
		if occ.After(expansionEnd) {
			break
		}
		// Skip occurrences that ended more than a week ago
		// But always include the event's start time
		occEnd := occ.Add(endTime.Sub(startTime))
		if occEnd.Before(oneWeekAgo) && !occ.Equal(startTime) {
			continue
		}
		occurrences = append(occurrences, occ)
	}

	// Convert to our RecurrenceOccurrence format
	result := make([]RecurrenceOccurrence, 0, len(occurrences))
	duration := endTime.Sub(startTime)

	for i, occ := range occurrences {
		// Calculate the end time for this occurrence
		occEnd := occ.Add(duration)

		result = append(result, RecurrenceOccurrence{
			StartTime: occ,
			EndTime:   occEnd,
			Index:     i,
		})

		// Safety limit to prevent excessive expansions
		if len(result) >= MaxRecurrenceExpansion {
			break
		}
	}

	if len(result) == 0 {
		// At minimum, include the first occurrence
		result = append(result, RecurrenceOccurrence{
			StartTime: startTime,
			EndTime:   endTime,
			Index:     0,
		})
	}

	return result, nil
}

// parseRRule parses an iCal RRULE string into an rrule.RRule object
// Handles conversion between iCal format and the format expected by rrule-go
func parseRRule(rruleStr string, startTime time.Time, allDay bool) (*rrule.RRule, error) {
	// Parse the RRULE string using StrToRRule which handles both the RRULE string
	// and uses the start time if provided in the RRULE
	rule, err := rrule.StrToRRule(rruleStr)
	if err != nil {
		return nil, fmt.Errorf("invalid RRULE format: %w", err)
	}

	// Set the start time for the recurrence
	// We need to use the DTStart method to set the start time
	rule.DTStart(startTime)

	// For all-day events, ensure we're working in UTC
	if allDay {
		startTime = startTime.UTC()
		rule.DTStart(startTime)
	}

	return rule, nil
}

// GetRecurrenceID generates a unique identifier for a recurring event series
// This is used to group all instances of the same recurring event
func GetRecurrenceID(externalUID string) string {
	return externalUID
}

// GetInstanceUID generates a unique identifier for a specific instance of a recurring event
// Format: base_uid#recurrence_id_instance
func GetInstanceUID(baseUID string, index int) string {
	return fmt.Sprintf("%s#%d", baseUID, index)
}

// ParseInstanceUID parses an instance UID to extract the base UID and instance index
// Returns the base UID and instance index, or an error if not an instance UID
func ParseInstanceUID(instanceUID string) (string, int, error) {
	parts := strings.Split(instanceUID, "#")
	if len(parts) != 2 {
		return "", 0, fmt.Errorf("not an instance UID")
	}

	var index int
	_, err := fmt.Sscanf(parts[1], "%d", &index)
	if err != nil {
		return "", 0, fmt.Errorf("invalid instance index: %w", err)
	}

	return parts[0], index, nil
}

// IsInstanceUID checks if a UID is an instance UID (contains #)
func IsInstanceUID(uid string) bool {
	return strings.Contains(uid, "#")
}

// GetBaseUID extracts the base UID from an instance UID
// If the UID is not an instance UID, returns it as-is
func GetBaseUID(uid string) string {
	if IsInstanceUID(uid) {
		baseUID, _, _ := ParseInstanceUID(uid)
		return baseUID
	}
	return uid
}
