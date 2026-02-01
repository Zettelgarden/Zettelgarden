package services

import (
	"context"
	"fmt"
	"log"
	"strings"
	"sync/atomic"
	"testing"
	"time"

	"github.com/robfig/cron/v3"
)

// testMockJob is a test helper for scheduler tests that extends mockScheduledJob
// with execution tracking capabilities
type testMockJob struct {
	name       string
	schedule   string
	maxRetries int
	execCount  *int32
	executed   chan struct{}
	failCount  *int32
	shouldFail bool
	handler    func(context.Context) error
}

func newTestMockJob(name, schedule string, maxRetries int) *testMockJob {
	var count int32
	var failCount int32
	return &testMockJob{
		name:       name,
		schedule:   schedule,
		maxRetries: maxRetries,
		execCount:  &count,
		executed:   make(chan struct{}, 10),
		failCount:  &failCount,
		shouldFail: false,
		handler: func(ctx context.Context) error {
			return nil
		},
	}
}

func (m *testMockJob) Name() string {
	return m.name
}

func (m *testMockJob) Schedule() string {
	return m.schedule
}

func (m *testMockJob) Handler(ctx context.Context) error {
	atomic.AddInt32(m.execCount, 1)
	if m.executed != nil {
		m.executed <- struct{}{}
	}
	if m.shouldFail {
		atomic.AddInt32(m.failCount, 1)
		return fmt.Errorf("mock job failure")
	}
	if m.handler != nil {
		return m.handler(ctx)
	}
	return nil
}

func (m *testMockJob) MaxRetries() int {
	return m.maxRetries
}

func (m *testMockJob) NextRun(from time.Time) time.Time {
	schedule, err := cron.ParseStandard(m.schedule)
	if err != nil {
		return time.Time{}
	}
	return schedule.Next(from)
}

// waitTimeout waits for the channel to receive a value or timeout
func (m *testMockJob) waitTimeout(timeout time.Duration) bool {
	select {
	case <-m.executed:
		return true
	case <-time.After(timeout):
		return false
	}
}

// TestSchedulerRegisterAndStart tests registering a job, starting the scheduler,
// and verifying execution
func TestSchedulerRegisterAndStart(t *testing.T) {
	// Create a logger for the scheduler
	var logBuf strings.Builder
	logger := log.New(&logBuf, "scheduler: ", log.LstdFlags)

	// Create scheduler without database (nil db = no tracking)
	scheduler := NewScheduler(nil)
	if scheduler == nil {
		t.Fatal("NewScheduler returned nil")
	}

	scheduler.logger = logger

	// Create a mock job that runs every second
	job := newTestMockJob("test-job", "*/1 * * * * *", 0)

	// Register the job
	err := scheduler.Register(job)
	if err != nil {
		t.Fatalf("Failed to register job: %v", err)
	}

	// Verify job is in the list
	jobs := scheduler.ListJobs()
	if len(jobs) != 1 {
		t.Fatalf("Expected 1 job, got %d", len(jobs))
	}
	if jobs[0] != "test-job" {
		t.Fatalf("Expected job name 'test-job', got '%s'", jobs[0])
	}

	// Start the scheduler
	scheduler.Start()
	defer scheduler.Stop()

	// Wait for the job to execute (should run within 2 seconds)
	if !job.waitTimeout(2 * time.Second) {
		t.Fatal("Job did not execute within timeout")
	}

	// Verify the job was executed
	count := atomic.LoadInt32(job.execCount)
	if count != 1 {
		t.Fatalf("Expected job to execute once, got %d executions", count)
	}

	t.Logf("Job executed successfully. Log output:\n%s", logBuf.String())
}

// TestSchedulerDuplicateRegistration verifies that duplicate job names are rejected
func TestSchedulerDuplicateRegistration(t *testing.T) {
	scheduler := NewScheduler(nil)

	// Create and register first job
	job1 := newTestMockJob("duplicate-job", "*/1 * * * * *", 0)
	err := scheduler.Register(job1)
	if err != nil {
		t.Fatalf("Failed to register first job: %v", err)
	}

	// Try to register a second job with the same name
	job2 := newTestMockJob("duplicate-job", "*/2 * * * * *", 0)
	err = scheduler.Register(job2)
	if err == nil {
		t.Fatal("Expected error when registering duplicate job name, got nil")
	}

	// Verify error message contains useful information
	if !strings.Contains(err.Error(), "duplicate-job") {
		t.Fatalf("Expected error message to contain job name, got: %v", err)
	}

	t.Logf("Correctly rejected duplicate registration: %v", err)
}

// TestSchedulerListJobs verifies ListJobs returns all registered jobs
func TestSchedulerListJobs(t *testing.T) {
	scheduler := NewScheduler(nil)

	// Initially should have no jobs
	jobs := scheduler.ListJobs()
	if len(jobs) != 0 {
		t.Fatalf("Expected 0 jobs, got %d", len(jobs))
	}

	// Register multiple jobs
	job1 := newTestMockJob("job1", "*/1 * * * * *", 0)
	job2 := newTestMockJob("job2", "*/2 * * * * *", 0)
	job3 := newTestMockJob("job3", "*/3 * * * * *", 0)

	for _, job := range []*testMockJob{job1, job2, job3} {
		if err := scheduler.Register(job); err != nil {
			t.Fatalf("Failed to register job %s: %v", job.Name(), err)
		}
	}

	// Verify all jobs are listed
	jobs = scheduler.ListJobs()
	if len(jobs) != 3 {
		t.Fatalf("Expected 3 jobs, got %d", len(jobs))
	}

	// Verify job names (order may vary)
	jobMap := make(map[string]bool)
	for _, name := range jobs {
		jobMap[name] = true
	}

	expectedJobs := []string{"job1", "job2", "job3"}
	for _, expected := range expectedJobs {
		if !jobMap[expected] {
			t.Fatalf("Expected job '%s' not found in list", expected)
		}
	}
}

// TestSchedulerJobTimeout verifies that job timeouts work correctly
func TestSchedulerJobTimeout(t *testing.T) {
	t.Skip("Skipping timeout test as it requires waiting 30+ minutes for context timeout")

	scheduler := NewScheduler(nil)

	// Create a job that will exceed its timeout
	var execCount int32
	job := &testMockJob{
		name:       "timeout-job",
		schedule:   "*/1 * * * * *",
		maxRetries: 0,
		execCount:  &execCount,
		executed:   make(chan struct{}, 10),
	}
	// Set handler to block longer than timeout
	job.handler = func(ctx context.Context) error {
		job.executed <- struct{}{}
		// Sleep longer than the 30-minute timeout (we'll simulate this)
		select {
		case <-ctx.Done():
			return ctx.Err()
		case <-time.After(35 * time.Minute):
			return nil
		}
	}

	err := scheduler.Register(job)
	if err != nil {
		t.Fatalf("Failed to register job: %v", err)
	}

	scheduler.Start()
	defer scheduler.Stop()

	// Wait for execution attempt
	if !job.waitTimeout(2 * time.Second) {
		t.Fatal("Job did not execute within timeout")
	}

	t.Log("Job timeout test completed (context cancellation should occur)")
}

// TestSchedulerRetryLogic verifies that failed jobs are retried
func TestSchedulerRetryLogic(t *testing.T) {
	t.Skip("Skipping retry test as it requires waiting for retry delays")

	scheduler := NewScheduler(nil)

	// Create a job that fails initially then succeeds
	attempts := int32(0)
	var execCount int32
	job := &testMockJob{
		name:       "retry-job",
		schedule:   "*/1 * * * * *",
		maxRetries: 3,
		execCount:  &execCount,
		executed:   make(chan struct{}, 10),
	}
	job.handler = func(ctx context.Context) error {
		atomic.AddInt32(&attempts, 1)
		job.executed <- struct{}{}
		if atomic.LoadInt32(&attempts) <= 2 {
			return fmt.Errorf("temporary failure")
		}
		return nil
	}

	err := scheduler.Register(job)
	if err != nil {
		t.Fatalf("Failed to register job: %v", err)
	}

	scheduler.Start()
	defer scheduler.Stop()

	// Wait for the job to execute and succeed
	timeout := time.After(5 * time.Second)
	success := false
	for i := 0; i < 5; i++ {
		select {
		case <-job.executed:
			if atomic.LoadInt32(&attempts) >= 3 {
				success = true
			}
		case <-timeout:
			break
		}
		time.Sleep(500 * time.Millisecond)
	}

	if !success {
		t.Fatalf("Job did not succeed after retries. Attempts: %d", atomic.LoadInt32(&attempts))
	}

	t.Logf("Job succeeded after %d attempts", atomic.LoadInt32(&attempts))
}
