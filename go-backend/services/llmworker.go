package services

import (
	"context"
	"fmt"
	"go-backend/models"
	"log"
	"os"
	"sync"
	"sync/atomic"
	"time"
)

// WorkerStats tracks statistics for a worker or worker pool
type WorkerStats struct {
	JobsProcessed int64 `json:"jobs_processed"`
	JobsSucceeded int64 `json:"jobs_succeeded"`
	JobsFailed    int64 `json:"jobs_failed"`
	JobsRetried   int64 `json:"jobs_retried"`
}

// Worker represents a single background job processor
type Worker struct {
	ID         string
	Queue      JobQueue
	Processor  JobProcessor
	Stats      WorkerStats
	stopChan   chan struct{}
	wg         sync.WaitGroup
	paused     *atomic.Bool // Shared pause state with worker pool
}

// WorkerPool manages a pool of workers processing jobs from the queue
type WorkerPool struct {
	workers    []*Worker
	queue      JobQueue
	processor  JobProcessor
	stats      WorkerStats
	stopChan   chan struct{}
	wg         sync.WaitGroup
	running    atomic.Bool
	shutdown   atomic.Bool
	paused     atomic.Bool // Indicates if job processing is paused
	mu         sync.RWMutex

	// Monitoring goroutine fields
	monitorStop chan struct{}
	monitorWg   sync.WaitGroup

	// Configuration
	workerCount      int
	pollInterval     time.Duration
	shutdownTimeout  time.Duration
}

// WorkerConfig holds configuration for the worker pool
type WorkerConfig struct {
	WorkerCount      int           // Number of worker goroutines (default: 5)
	PollInterval     time.Duration // How often to poll for jobs (default: 1s)
	ShutdownTimeout  time.Duration // Max time to wait for graceful shutdown (default: 30s)
}

// DefaultWorkerConfig returns default worker configuration
func DefaultWorkerConfig() WorkerConfig {
	return WorkerConfig{
		WorkerCount:      getEnvInt("LLM_WORKERS", 5),
		PollInterval:     1 * time.Second,
		ShutdownTimeout:  30 * time.Second,
	}
}

// JobProcessor defines the interface for processing jobs
// Concrete implementations will be provided by the LLM processor
type JobProcessor interface {
	// ProcessJob executes a job and returns the result or error
	ProcessJob(ctx context.Context, job *models.LLMJob) (map[string]interface{}, error)
}

// NewWorker creates a new worker with the given ID, queue, processor, and pause state
func NewWorker(id string, queue JobQueue, processor JobProcessor, paused *atomic.Bool) *Worker {
	return &Worker{
		ID:        id,
		Queue:     queue,
		Processor: processor,
		stopChan:  make(chan struct{}),
		paused:    paused,
	}
}

// Start begins the worker's main loop
func (w *Worker) Start() {
	w.wg.Add(1)
	go func() {
		defer w.wg.Done()
		log.Printf("[Worker %s] Started", w.ID)
		defer log.Printf("[Worker %s] Stopped", w.ID)

		for {
			select {
			case <-w.stopChan:
				return
			default:
				// Check if paused, if so wait before trying again
				if w.paused != nil && w.paused.Load() {
					time.Sleep(500 * time.Millisecond)
					continue
				}

				if err := w.processNextJob(); err != nil {
					log.Printf("[Worker %s] Error processing job: %v", w.ID, err)
					// Don't sleep on error, try again immediately
				}
			}
		}
	}()
}

// Stop signals the worker to stop gracefully
func (w *Worker) Stop() {
	log.Printf("[Worker %s] Stopping...", w.ID)
	close(w.stopChan)
	w.wg.Wait()
	log.Printf("[Worker %s] Stopped (final stats: processed=%d, succeeded=%d, failed=%d)",
		w.ID, w.Stats.JobsProcessed, w.Stats.JobsSucceeded, w.Stats.JobsFailed)
}

// processNextJob attempts to dequeue and process a single job
func (w *Worker) processNextJob() error {
	ctx := context.Background()

	// Try to dequeue a job
	job, err := w.Queue.Dequeue(ctx)
	if err != nil {
		return fmt.Errorf("dequeue error: %w", err)
	}

	// No job available, sleep briefly
	if job == nil {
		time.Sleep(100 * time.Millisecond)
		return nil
	}

	// Process the job
	w.Stats.JobsProcessed++
	return w.processJobWithRetry(ctx, job)
}

// processJobWithRetry processes a job with retry logic
func (w *Worker) processJobWithRetry(ctx context.Context, job *models.LLMJob) error {
	// Get job timeout
	timeout := GetJobTimeout(job)

	// Create context with timeout
	jobCtx, cancel := context.WithTimeout(ctx, timeout)
	defer cancel()

	// Start heartbeat goroutine
	heartbeatStop := make(chan struct{})
	done := make(chan struct{})
	go func() {
		defer close(done)
		ticker := time.NewTicker(30 * time.Second)
		defer ticker.Stop()
		for {
			select {
			case <-heartbeatStop:
				return
			case <-ticker.C:
				if err := w.Queue.UpdateHeartbeat(jobCtx, job.ID); err != nil {
					log.Printf("[Worker %s] Failed to update heartbeat for job %d: %v", w.ID, job.ID, err)
				}
			}
		}
	}()
	defer func() {
		close(heartbeatStop)
		<-done // Wait for heartbeat goroutine to exit
	}()

	// Process the job
	result, err := w.Processor.ProcessJob(jobCtx, job)

	if err != nil {
		return w.handleJobError(ctx, job, err)
	}

	// Success - update with result
	if err := w.Queue.UpdateStatusWithResult(ctx, job.ID, models.JobStatusCompleted, result); err != nil {
		log.Printf("[Worker %s] Failed to update job %d as completed: %v", w.ID, job.ID, err)
		return err
	}

	w.Stats.JobsSucceeded++
	log.Printf("[Worker %s] Job %d (type: %s) completed successfully", w.ID, job.ID, job.JobType)
	return nil
}

// handleJobError handles job processing errors with retry logic
func (w *Worker) handleJobError(ctx context.Context, job *models.LLMJob, jobErr error) error {
	w.Stats.JobsFailed++

	// Check if context was cancelled (timeout or shutdown)
	if ctx.Err() == context.DeadlineExceeded {
		errMsg := fmt.Sprintf("Job timed out after %ds", job.TimeoutSecs)
		log.Printf("[Worker %s] Job %d timed out", w.ID, job.ID)

		// Check if we should retry
		if ShouldRetryJob(job) {
			w.Stats.JobsRetried++
			backoff := CalculateBackoff(job.RetryCount)
			log.Printf("[Worker %s] Retrying job %d after %v (retry %d/%d)",
				w.ID, job.ID, backoff, job.RetryCount+1, job.MaxRetries)

			time.Sleep(backoff)
			if err := w.Queue.IncrementRetry(ctx, job.ID); err != nil {
				log.Printf("[Worker %s] Failed to increment retry for job %d: %v", w.ID, job.ID, err)
			}
			return nil
		}

		// Max retries exceeded
		if err := w.Queue.UpdateStatusWithError(ctx, job.ID, models.JobStatusFailed, errMsg); err != nil {
			log.Printf("[Worker %s] Failed to update job %d as failed: %v", w.ID, job.ID, err)
		}
		return fmt.Errorf("job timed out: %w", jobErr)
	}

	if ctx.Err() == context.Canceled {
		errMsg := "Job cancelled due to shutdown"
		log.Printf("[Worker %s] Job %d cancelled", w.ID, job.ID)
		if err := w.Queue.UpdateStatusWithError(ctx, job.ID, models.JobStatusFailed, errMsg); err != nil {
			log.Printf("[Worker %s] Failed to update job %d as failed: %v", w.ID, job.ID, err)
		}
		return fmt.Errorf("job cancelled: %w", jobErr)
	}

	// Other errors
	log.Printf("[Worker %s] Job %d failed: %v", w.ID, job.ID, jobErr)

	if ShouldRetryJob(job) {
		w.Stats.JobsRetried++
		backoff := CalculateBackoff(job.RetryCount)
		log.Printf("[Worker %s] Retrying job %d after %v (retry %d/%d)",
			w.ID, job.ID, backoff, job.RetryCount+1, job.MaxRetries)

		time.Sleep(backoff)
		if err := w.Queue.IncrementRetry(ctx, job.ID); err != nil {
			log.Printf("[Worker %s] Failed to increment retry for job %d: %v", w.ID, job.ID, err)
		}
		return nil
	}

	// Max retries exceeded - mark as permanently failed
	errMsg := fmt.Sprintf("Failed after %d retries: %v", job.RetryCount, jobErr)
	if err := w.Queue.UpdateStatusWithError(ctx, job.ID, models.JobStatusFailed, errMsg); err != nil {
		log.Printf("[Worker %s] Failed to update job %d as failed: %v", w.ID, job.ID, err)
	}

	return fmt.Errorf("job failed permanently: %w", jobErr)
}

// NewWorkerPool creates a new worker pool with the given configuration
func NewWorkerPool(queue JobQueue, processor JobProcessor, config WorkerConfig) *WorkerPool {
	return &WorkerPool{
		queue:           queue,
		processor:       processor,
		workerCount:     config.WorkerCount,
		pollInterval:    config.PollInterval,
		shutdownTimeout: config.ShutdownTimeout,
		stopChan:        make(chan struct{}),
	}
}

// Start starts all workers in the pool
func (p *WorkerPool) Start() error {
	p.mu.Lock()
	defer p.mu.Unlock()

	if p.running.Load() {
		return fmt.Errorf("worker pool already running")
	}

	p.workers = make([]*Worker, p.workerCount)
	for i := 0; i < p.workerCount; i++ {
		workerID := fmt.Sprintf("worker-%d", i+1)
		p.workers[i] = NewWorker(workerID, p.queue, p.processor, &p.paused)
		p.workers[i].Start()
	}

	// Start monitoring goroutine for stuck job detection
	p.monitorStop = make(chan struct{})
	p.monitorWg.Add(1)
	go p.monitorRunningJobs()

	p.running.Store(true)
	log.Printf("[WorkerPool] Started with %d workers", p.workerCount)
	return nil
}

// Stop gracefully shuts down all workers in the pool
func (p *WorkerPool) Stop() error {
	p.mu.Lock()
	defer p.mu.Unlock()

	if !p.running.Load() {
		return fmt.Errorf("worker pool not running")
	}

	if p.shutdown.Load() {
		return fmt.Errorf("shutdown already in progress")
	}

	p.shutdown.Store(true)
	log.Printf("[WorkerPool] Shutting down...")

	// Stop monitoring goroutine first
	if p.monitorStop != nil {
		close(p.monitorStop)
		p.monitorWg.Wait()
	}

	// Signal all workers to stop
	for _, worker := range p.workers {
		worker.Stop()
	}

	// Wait for all workers to finish or timeout
	done := make(chan struct{})
	go func() {
		p.wg.Wait()
		close(done)
	}()

	select {
	case <-done:
		log.Printf("[WorkerPool] All workers stopped gracefully")
	case <-time.After(p.shutdownTimeout):
		log.Printf("[WorkerPool] Shutdown timed out after %v", p.shutdownTimeout)
	}

	p.running.Store(false)
	p.shutdown.Store(false)

	// Log final stats
	totalProcessed := atomic.LoadInt64(&p.stats.JobsProcessed)
	totalSucceeded := atomic.LoadInt64(&p.stats.JobsSucceeded)
	totalFailed := atomic.LoadInt64(&p.stats.JobsFailed)
	log.Printf("[WorkerPool] Final stats: processed=%d, succeeded=%d, failed=%d",
		totalProcessed, totalSucceeded, totalFailed)

	return nil
}

// monitorRunningJobs periodically checks for and marks stuck jobs as failed
func (p *WorkerPool) monitorRunningJobs() {
	defer p.monitorWg.Done()
	ticker := time.NewTicker(60 * time.Second)
	defer ticker.Stop()

	log.Printf("[WorkerPool] Started monitoring goroutine for stuck job detection")

	for {
		select {
		case <-p.monitorStop:
			log.Printf("[WorkerPool] Stopping monitoring goroutine")
			return
		case <-ticker.C:
			count, err := p.queue.MarkStuckJobsFailed(context.Background())
			if err != nil {
				log.Printf("[WorkerPool] Failed to mark stuck jobs: %v", err)
			} else if count > 0 {
				log.Printf("[WorkerPool] Marked %d stuck jobs as failed", count)
			}
		}
	}
}

// IsRunning returns true if the worker pool is running
func (p *WorkerPool) IsRunning() bool {
	return p.running.Load()
}

// Stats returns the current statistics for the worker pool
func (p *WorkerPool) Stats() WorkerStats {
	// Aggregate stats from all workers
	stats := WorkerStats{}

	p.mu.RLock()
	defer p.mu.RUnlock()

	for _, worker := range p.workers {
		stats.JobsProcessed += atomic.LoadInt64(&worker.Stats.JobsProcessed)
		stats.JobsSucceeded += atomic.LoadInt64(&worker.Stats.JobsSucceeded)
		stats.JobsFailed += atomic.LoadInt64(&worker.Stats.JobsFailed)
		stats.JobsRetried += atomic.LoadInt64(&worker.Stats.JobsRetried)
	}

	return stats
}

// WorkerCount returns the number of workers in the pool
func (p *WorkerPool) WorkerCount() int {
	p.mu.RLock()
	defer p.mu.RUnlock()
	return len(p.workers)
}

// Pause pauses job processing in the worker pool
// Workers will stop picking up new jobs but will finish processing current jobs
func (p *WorkerPool) Pause() error {
	p.mu.Lock()
	defer p.mu.Unlock()

	if !p.running.Load() {
		return fmt.Errorf("worker pool not running")
	}

	if p.paused.Load() {
		return fmt.Errorf("worker pool already paused")
	}

	p.paused.Store(true)
	log.Printf("[WorkerPool] Paused job processing")
	return nil
}

// Resume resumes job processing in the worker pool
func (p *WorkerPool) Resume() error {
	p.mu.Lock()
	defer p.mu.Unlock()

	if !p.running.Load() {
		return fmt.Errorf("worker pool not running")
	}

	if !p.paused.Load() {
		return fmt.Errorf("worker pool not paused")
	}

	p.paused.Store(false)
	log.Printf("[WorkerPool] Resumed job processing")
	return nil
}

// IsPaused returns true if the worker pool is paused
func (p *WorkerPool) IsPaused() bool {
	return p.paused.Load()
}

// GetQueueDepth returns the number of pending jobs in the queue
func (p *WorkerPool) GetQueueDepth() (int, error) {
	return p.queue.GetQueueDepth(context.Background())
}

// getEnvInt gets an integer environment variable with a fallback
func getEnvInt(key string, defaultValue int) int {
	if val := os.Getenv(key); val != "" {
		if intVal, err := parseIntSafe(val); err == nil {
			return intVal
		}
	}
	return defaultValue
}

// parseIntSafe parses an integer safely
func parseIntSafe(s string) (int, error) {
	var i int
	_, err := fmt.Sscanf(s, "%d", &i)
	return i, err
}
