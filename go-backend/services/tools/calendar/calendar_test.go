package calendar

import (
	"testing"
	"time"
)

// TODO: Add unit tests for calendar domain functions
// These tests will require a test database setup

// TestParseISODate tests the ISO date parsing function
func TestParseISODate(t *testing.T) {
	tests := []struct {
		name    string
		input   string
		wantErr bool
	}{
		{
			name:    "valid RFC3339 date",
			input:   "2026-01-01T00:00:00Z",
			wantErr: false,
		},
		{
			name:    "valid RFC3339 with offset",
			input:   "2026-12-31T23:59:59+08:00",
			wantErr: false,
		},
		{
			name:    "invalid date format",
			input:   "not-a-date",
			wantErr: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			_, err := ParseISODate(tt.input)
			if (err != nil) != tt.wantErr {
				t.Errorf("ParseISODate() error = %v, wantErr %v", err, tt.wantErr)
			}
		})
	}
}

// TestValidateDateRange tests the date range validation function
func TestValidateDateRange(t *testing.T) {
	start := time.Date(2026, 1, 1, 0, 0, 0, 0, time.UTC)
	end := time.Date(2026, 12, 31, 23, 59, 59, 0, time.UTC)
	invalidEnd := time.Date(2025, 1, 1, 0, 0, 0, 0, time.UTC)

	tests := []struct {
		name    string
		start   time.Time
		end     time.Time
		wantErr bool
	}{
		{
			name:    "valid date range",
			start:   start,
			end:     end,
			wantErr: false,
		},
		{
			name:    "same start and end",
			start:   start,
			end:     start,
			wantErr: false,
		},
		{
			name:    "end before start",
			start:   start,
			end:     invalidEnd,
			wantErr: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := ValidateDateRange(tt.start, tt.end)
			if (err != nil) != tt.wantErr {
				t.Errorf("ValidateDateRange() error = %v, wantErr %v", err, tt.wantErr)
			}
		})
	}
}
