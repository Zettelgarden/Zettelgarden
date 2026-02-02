package jobs

import (
	"context"
	"net/http"
	"net/http/httptest"
	"os"
	"testing"
	"time"

	"go-backend/services"
)

// TestUptimeKumaPingJobImplementsInterface verifies that UptimeKumaPingJob implements
// the ScheduledJob interface with correct values.
func TestUptimeKumaPingJobImplementsInterface(t *testing.T) {
	job := NewUptimeKumaPingJob()

	// Verify Name returns expected value
	if got, want := job.Name(), "uptime-kuma-ping"; got != want {
		t.Errorf("Name() = %q, want %q", got, want)
	}

	// Verify Schedule returns expected cron expression (every minute)
	if got, want := job.Schedule(), "0 * * * * *"; got != want {
		t.Errorf("Schedule() = %q, want %q", got, want)
	}

	// Verify MaxRetries returns expected value
	if got, want := job.MaxRetries(), 3; got != want {
		t.Errorf("MaxRetries() = %d, want %d", got, want)
	}

	// Verify the job implements ScheduledJob interface
	var _ services.ScheduledJob = job
}

// TestUptimeKumaPingJobHandler_MissingURL verifies behavior when UPTIME_KUMA_PUSH_URL is not set
func TestUptimeKumaPingJobHandler_MissingURL(t *testing.T) {
	// Unset the environment variable
	os.Unsetenv("UPTIME_KUMA_PUSH_URL")

	job := NewUptimeKumaPingJob()
	ctx := context.Background()

	// Handler should succeed when URL is not configured (graceful degradation)
	if err := job.Handler(ctx); err != nil {
		t.Errorf("Handler() with missing URL should succeed, got error: %v", err)
	}
}

// TestUptimeKumaPingJobHandler_Success verifies successful POST to Uptime Kuma
func TestUptimeKumaPingJobHandler_Success(t *testing.T) {
	// Create a test server that acts like Uptime Kuma
	receivedRequest := make(chan struct{})
	testServer := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		close(receivedRequest)
		if r.Method != http.MethodPost {
			t.Errorf("Expected POST request, got %s", r.Method)
		}
		w.WriteHeader(http.StatusOK)
	}))
	defer testServer.Close()

	// Set the environment variable to point to our test server
	os.Setenv("UPTIME_KUMA_PUSH_URL", testServer.URL)
	defer os.Unsetenv("UPTIME_KUMA_PUSH_URL")

	job := NewUptimeKumaPingJob()
	ctx := context.Background()

	done := make(chan error, 1)
	go func() {
		done <- job.Handler(ctx)
	}()

	select {
	case <-receivedRequest:
		// Request was received
	case err := <-done:
		t.Fatalf("Handler() completed before request was received: %v", err)
	case <-time.After(5 * time.Second):
		t.Fatal("Handler did not send HTTP request within timeout")
	}

	if err := <-done; err != nil {
		t.Errorf("Handler() should succeed, got error: %v", err)
	}
}
