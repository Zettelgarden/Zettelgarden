package services

import (
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
)

func TestExpandRecurrence_Daily(t *testing.T) {
	// Test a simple daily recurring event
	startTime := time.Date(2026, 2, 1, 10, 0, 0, 0, time.UTC)
	endTime := time.Date(2026, 2, 1, 11, 0, 0, 0, time.UTC)
	rrule := "FREQ=DAILY;COUNT=5"

	occurrences, err := ExpandRecurrence(rrule, startTime, endTime, false)
	assert.NoError(t, err)
	assert.NotEmpty(t, occurrences)

	// Should have generated at least some occurrences
	assert.GreaterOrEqual(t, len(occurrences), 1)

	// First occurrence should be the start time
	assert.Equal(t, startTime, occurrences[0].StartTime)
	assert.Equal(t, endTime, occurrences[0].EndTime)

	// Check duration is preserved for all occurrences
	duration := endTime.Sub(startTime)
	for _, occ := range occurrences {
		assert.Equal(t, duration, occ.EndTime.Sub(occ.StartTime))
	}
}

func TestExpandRecurrence_Weekly(t *testing.T) {
	// Test a weekly recurring event
	startTime := time.Date(2026, 2, 1, 10, 0, 0, 0, time.UTC)
	endTime := time.Date(2026, 2, 1, 11, 0, 0, 0, time.UTC)
	rrule := "FREQ=WEEKLY;COUNT=3"

	occurrences, err := ExpandRecurrence(rrule, startTime, endTime, false)
	assert.NoError(t, err)
	assert.NotEmpty(t, occurrences)

	// Check that occurrences are weekly (7 days apart)
	for i := 1; i < len(occurrences); i++ {
		diff := occurrences[i].StartTime.Sub(occurrences[i-1].StartTime)
		expectedDiff := 7 * 24 * time.Hour
		// Allow for some tolerance due to expansion window
		assert.Equal(t, expectedDiff, diff)
	}
}

func TestExpandRecurrence_Monthly(t *testing.T) {
	// Test a monthly recurring event
	startTime := time.Date(2026, 2, 1, 10, 0, 0, 0, time.UTC)
	endTime := time.Date(2026, 2, 1, 11, 0, 0, 0, time.UTC)
	rrule := "FREQ=MONTHLY;COUNT=3"

	occurrences, err := ExpandRecurrence(rrule, startTime, endTime, false)
	assert.NoError(t, err)
	assert.NotEmpty(t, occurrences)

	// First occurrence should be the start time
	assert.Equal(t, startTime, occurrences[0].StartTime)
}

func TestExpandRecurrence_AllDay(t *testing.T) {
	// Test an all-day recurring event
	startTime := time.Date(2026, 2, 1, 0, 0, 0, 0, time.UTC)
	endTime := time.Date(2026, 2, 2, 0, 0, 0, 0, time.UTC)
	rrule := "FREQ=DAILY;COUNT=5"

	occurrences, err := ExpandRecurrence(rrule, startTime, endTime, true)
	assert.NoError(t, err)
	assert.NotEmpty(t, occurrences)

	// Check that all occurrences are all-day (24 hours)
	for _, occ := range occurrences {
		duration := occ.EndTime.Sub(occ.StartTime)
		assert.Equal(t, 24*time.Hour, duration)
	}
}

func TestGetInstanceUID(t *testing.T) {
	baseUID := "event123"

	// Test generating instance UIDs
	uid0 := GetInstanceUID(baseUID, 0)
	assert.Equal(t, "event123#0", uid0)

	uid1 := GetInstanceUID(baseUID, 1)
	assert.Equal(t, "event123#1", uid1)

	uid5 := GetInstanceUID(baseUID, 5)
	assert.Equal(t, "event123#5", uid5)
}

func TestParseInstanceUID(t *testing.T) {
	tests := []struct {
		name          string
		instanceUID   string
		expectedBase  string
		expectedIndex int
		expectError   bool
	}{
		{
			name:          "valid instance UID",
			instanceUID:   "event123#0",
			expectedBase:  "event123",
			expectedIndex: 0,
			expectError:   false,
		},
		{
			name:          "valid instance UID with higher index",
			instanceUID:   "event456#42",
			expectedBase:  "event456",
			expectedIndex: 42,
			expectError:   false,
		},
		{
			name:        "invalid instance UID - no hash",
			instanceUID: "event123",
			expectError: true,
		},
		{
			name:        "invalid instance UID - multiple hashes",
			instanceUID: "event123#0#1",
			expectError: true,
		},
		{
			name:        "invalid instance UID - non-numeric index",
			instanceUID: "event123#abc",
			expectError: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			base, index, err := ParseInstanceUID(tt.instanceUID)
			if tt.expectError {
				assert.Error(t, err)
			} else {
				assert.NoError(t, err)
				assert.Equal(t, tt.expectedBase, base)
				assert.Equal(t, tt.expectedIndex, index)
			}
		})
	}
}

func TestIsInstanceUID(t *testing.T) {
	assert.True(t, IsInstanceUID("event123#0"))
	assert.True(t, IsInstanceUID("event456#42"))
	assert.False(t, IsInstanceUID("event123"))
	assert.False(t, IsInstanceUID(""))
	// Note: "event#" technically contains a hash, so IsInstanceUID returns true
	// but ParseInstanceUID would fail to parse the index
	assert.True(t, IsInstanceUID("event#"))
}

func TestGetBaseUID(t *testing.T) {
	assert.Equal(t, "event123", GetBaseUID("event123#0"))
	assert.Equal(t, "event456", GetBaseUID("event456#42"))
	assert.Equal(t, "event789", GetBaseUID("event789")) // Non-instance UID returned as-is
}

func TestGetRecurrenceID(t *testing.T) {
	uid := "event123@calendar.com"
	recurrenceID := GetRecurrenceID(uid)
	assert.Equal(t, uid, recurrenceID)
}

func TestExpandRecurrence_EmptyRRULE(t *testing.T) {
	startTime := time.Date(2026, 2, 1, 10, 0, 0, 0, time.UTC)
	endTime := time.Date(2026, 2, 1, 11, 0, 0, 0, time.UTC)

	_, err := ExpandRecurrence("", startTime, endTime, false)
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "empty recurrence rule")
}

func TestExpandRecurrence_InvalidRRULE(t *testing.T) {
	startTime := time.Date(2026, 2, 1, 10, 0, 0, 0, time.UTC)
	endTime := time.Date(2026, 2, 1, 11, 0, 0, 0, time.UTC)

	_, err := ExpandRecurrence("INVALID_RRULE", startTime, endTime, false)
	assert.Error(t, err)
}
