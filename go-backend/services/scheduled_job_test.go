package services

import (
	"context"
	"testing"
)

// Test that a job implementing the interface can be queried
func TestScheduledJobInterface(t *testing.T) {
	// This will fail to compile if ScheduledJob interface doesn't exist
	var job ScheduledJob = &mockScheduledJob{
		name:       "test-job",
		schedule:   "*/5 * * * *",
		maxRetries: 3,
	}

	if job.Name() != "test-job" {
		t.Errorf("expected name 'test-job', got '%s'", job.Name())
	}

	if job.Schedule() != "*/5 * * * *" {
		t.Errorf("expected schedule '*/5 * * * *', got '%s'", job.Schedule())
	}

	if job.MaxRetries() != 3 {
		t.Errorf("expected max retries 3, got %d", job.MaxRetries())
	}
}

// mockScheduledJob implements ScheduledJob for testing
type mockScheduledJob struct {
	name       string
	schedule   string
	maxRetries int
	handlerErr error
}

func (m *mockScheduledJob) Name() string       { return m.name }
func (m *mockScheduledJob) Schedule() string   { return m.schedule }
func (m *mockScheduledJob) MaxRetries() int    { return m.maxRetries }
func (m *mockScheduledJob) Handler(ctx context.Context) error {
	return m.handlerErr
}
