package models

import (
	"database/sql"
	"encoding/json"
	"fmt"
	"log"
	"time"
)

// JobStatus represents the current status of a job
type JobStatus string

const (
	JobStatusPending   JobStatus = "pending"
	JobStatusRunning   JobStatus = "running"
	JobStatusCompleted JobStatus = "completed"
	JobStatusFailed    JobStatus = "failed"
	JobStatusCancelled JobStatus = "cancelled"
)

// JobType represents the type of LLM job
type JobType string

const (
	JobTypeSummarization        JobType = "summarization"
	JobTypeEntityExtraction     JobType = "entity_extraction"
	JobTypeFactEntityExtraction JobType = "fact_entity_extraction"
	JobTypeChat                 JobType = "chat"
	JobTypeEmail                JobType = "email"
	JobTypeFileTextExtraction   JobType = "file_text_extraction"
)

// LLMJob represents an asynchronous LLM operation in the job queue
type LLMJob struct {
	ID            int                    `json:"id"`
	UserID        int                    `json:"user_id"`
	JobType       JobType                `json:"job_type"`
	Status        JobStatus              `json:"status"`
	Priority      int                    `json:"priority"`
	Payload       map[string]interface{} `json:"payload"`
	Result        map[string]interface{} `json:"result,omitempty"`
	ErrorMessage  string                 `json:"error_message,omitempty"`
	CreatedAt     time.Time              `json:"created_at"`
	StartedAt     *time.Time             `json:"started_at,omitempty"`
	CompletedAt   *time.Time             `json:"completed_at,omitempty"`
	RetryCount    int                    `json:"retry_count"`
	MaxRetries    int                    `json:"max_retries"`
	TimeoutSecs   int                    `json:"timeout_seconds"`
	CorrelationID string                 `json:"correlation_id,omitempty"`
}

// CreateJobParams represents parameters for creating a new job
type CreateJobParams struct {
	UserID        int                    `json:"user_id"`
	JobType       JobType                `json:"job_type"`
	Priority      int                    `json:"priority,omitempty"`
	Payload       map[string]interface{} `json:"payload"`
	MaxRetries    int                    `json:"max_retries,omitempty"`
	TimeoutSecs   int                    `json:"timeout_seconds,omitempty"`
	CorrelationID string                 `json:"correlation_id,omitempty"`
}

// JobListParams represents parameters for listing jobs
type JobListParams struct {
	UserID int       `json:"user_id"`
	Status JobStatus `json:"status,omitempty"`
	Limit  int       `json:"limit,omitempty"`
	Offset int       `json:"offset,omitempty"`
}

// JobStats represents statistics about jobs
type JobStats struct {
	Total     int             `json:"total"`
	Pending   int             `json:"pending"`
	Running   int             `json:"running"`
	Completed int             `json:"completed"`
	Failed    int             `json:"failed"`
	ByType    map[JobType]int `json:"by_type"`
}

// ScanLLMJob scans a single LLMJob from a sql.Row
func ScanLLMJob(row *sql.Row) (*LLMJob, error) {
	var job LLMJob
	var payloadJSON, resultJSON []byte
	var startedAt, completedAt sql.NullTime
	var errorMessage sql.NullString
	var correlationID sql.NullString

	err := row.Scan(
		&job.ID,
		&job.UserID,
		&job.JobType,
		&job.Status,
		&job.Priority,
		&payloadJSON,
		&resultJSON,
		&errorMessage,
		&job.CreatedAt,
		&startedAt,
		&completedAt,
		&job.RetryCount,
		&job.MaxRetries,
		&job.TimeoutSecs,
		&correlationID,
	)
	if err != nil {
		if err == sql.ErrNoRows {
			return nil, nil
		}
		log.Printf("Error scanning LLM job: %v", err)
		return nil, err
	}

	// Handle nullable correlation_id
	if correlationID.Valid {
		job.CorrelationID = correlationID.String
	}

	// Handle nullable error_message
	if errorMessage.Valid {
		job.ErrorMessage = errorMessage.String
	}

	// Unmarshal payload JSONB
	if len(payloadJSON) > 0 && string(payloadJSON) != "null" {
		if err := json.Unmarshal(payloadJSON, &job.Payload); err != nil {
			log.Printf("Error unmarshaling payload JSON: %v", err)
			return nil, err
		}
	} else {
		job.Payload = make(map[string]interface{})
	}

	// Unmarshal result JSONB (may be null)
	if len(resultJSON) > 0 && string(resultJSON) != "null" {
		if err := json.Unmarshal(resultJSON, &job.Result); err != nil {
			log.Printf("Error unmarshaling result JSON: %v", err)
			return nil, err
		}
	}

	// Handle nullable timestamps
	if startedAt.Valid {
		job.StartedAt = &startedAt.Time
	}
	if completedAt.Valid {
		job.CompletedAt = &completedAt.Time
	}

	return &job, nil
}

// ScanLLMJobs scans multiple LLMJobs from sql.Rows
func ScanLLMJobs(rows *sql.Rows) ([]LLMJob, error) {
	var jobs []LLMJob

	defer rows.Close()

	for rows.Next() {
		var job LLMJob
		var payloadJSON, resultJSON []byte
		var startedAt, completedAt sql.NullTime
		var errorMessage sql.NullString
		var correlationID sql.NullString

		if err := rows.Scan(
			&job.ID,
			&job.UserID,
			&job.JobType,
			&job.Status,
			&job.Priority,
			&payloadJSON,
			&resultJSON,
			&errorMessage,
			&job.CreatedAt,
			&startedAt,
			&completedAt,
			&job.RetryCount,
			&job.MaxRetries,
			&job.TimeoutSecs,
			&correlationID,
		); err != nil {
			log.Printf("Error scanning LLM job: %v", err)
			return jobs, err
		}

		// Handle nullable correlation_id
		if correlationID.Valid {
			job.CorrelationID = correlationID.String
		}

		// Handle nullable error_message
		if errorMessage.Valid {
			job.ErrorMessage = errorMessage.String
		}

		// Unmarshal payload JSONB
		if len(payloadJSON) > 0 && string(payloadJSON) != "null" {
			if err := json.Unmarshal(payloadJSON, &job.Payload); err != nil {
				log.Printf("Error unmarshaling payload JSON: %v", err)
				return jobs, err
			}
		} else {
			job.Payload = make(map[string]interface{})
		}

		// Unmarshal result JSONB (may be null)
		if len(resultJSON) > 0 && string(resultJSON) != "null" {
			if err := json.Unmarshal(resultJSON, &job.Result); err != nil {
				log.Printf("Error unmarshaling result JSON: %v", err)
				return jobs, err
			}
		}

		// Handle nullable timestamps
		if startedAt.Valid {
			job.StartedAt = &startedAt.Time
		}
		if completedAt.Valid {
			job.CompletedAt = &completedAt.Time
		}

		jobs = append(jobs, job)
	}

	if err := rows.Err(); err != nil {
		log.Printf("Error iterating LLM jobs: %v", err)
		return jobs, err
	}

	return jobs, nil
}

// CreateJob creates a new job in the database
func CreateJob(db *sql.DB, params CreateJobParams) (*LLMJob, error) {
	payloadJSON, err := json.Marshal(params.Payload)
	if err != nil {
		return nil, fmt.Errorf("error marshaling payload: %w", err)
	}

	// Set defaults
	priority := params.Priority
	if priority == 0 {
		priority = 5
	}
	maxRetries := params.MaxRetries
	if maxRetries == 0 {
		maxRetries = 3
	}
	timeoutSecs := params.TimeoutSecs
	if timeoutSecs == 0 {
		timeoutSecs = 300
	}

	query := `
		INSERT INTO llm_jobs (user_id, job_type, priority, payload, max_retries, timeout_seconds, correlation_id)
		VALUES ($1, $2, $3, $4, $5, $6, $7)
		RETURNING id, user_id, job_type, status, priority, payload, result, error_message,
			created_at, started_at, completed_at, retry_count, max_retries, timeout_seconds, correlation_id
	`

	return ScanLLMJob(db.QueryRow(query,
		params.UserID,
		params.JobType,
		priority,
		payloadJSON,
		maxRetries,
		timeoutSecs,
		params.CorrelationID,
	))
}

// GetJob retrieves a job by ID
func GetJob(db *sql.DB, jobID int) (*LLMJob, error) {
	query := `
		SELECT id, user_id, job_type, status, priority, payload, result, error_message,
			created_at, started_at, completed_at, retry_count, max_retries, timeout_seconds, correlation_id
		FROM llm_jobs
		WHERE id = $1
	`
	return ScanLLMJob(db.QueryRow(query, jobID))
}

// ListJobs retrieves jobs for a user with optional filtering and pagination
func ListJobs(db *sql.DB, params JobListParams) ([]LLMJob, error) {
	query := `
		SELECT id, user_id, job_type, status, priority, payload, result, error_message,
			created_at, started_at, completed_at, retry_count, max_retries, timeout_seconds, correlation_id
		FROM llm_jobs
		WHERE user_id = $1
	`
	args := []interface{}{params.UserID}
	argIdx := 2

	if params.Status != "" {
		query += fmt.Sprintf(" AND status = $%d", argIdx)
		args = append(args, params.Status)
		argIdx++
	}

	query += " ORDER BY created_at DESC"

	if params.Limit > 0 {
		query += fmt.Sprintf(" LIMIT $%d", argIdx)
		args = append(args, params.Limit)
		argIdx++
	}

	if params.Offset > 0 {
		query += fmt.Sprintf(" OFFSET $%d", argIdx)
		args = append(args, params.Offset)
	}

	rows, err := db.Query(query, args...)
	if err != nil {
		return nil, err
	}

	return ScanLLMJobs(rows)
}

// UpdateJobStatus updates the status of a job
func UpdateJobStatus(db *sql.DB, jobID int, status JobStatus) error {
	query := `UPDATE llm_jobs SET status = $1, updated_at = CURRENT_TIMESTAMP WHERE id = $2`
	_, err := db.Exec(query, status, jobID)
	return err
}

// UpdateJobStatusWithResult updates the status and result of a job
func UpdateJobStatusWithResult(db *sql.DB, jobID int, status JobStatus, result map[string]interface{}) error {
	resultJSON, err := json.Marshal(result)
	if err != nil {
		return fmt.Errorf("error marshaling result: %w", err)
	}

	query := `
		UPDATE llm_jobs
		SET status = $1, result = $2, completed_at = CURRENT_TIMESTAMP, updated_at = CURRENT_TIMESTAMP
		WHERE id = $3
	`
	_, err = db.Exec(query, status, resultJSON, jobID)
	return err
}

// UpdateJobStatusWithError updates the status and error message of a job
func UpdateJobStatusWithError(db *sql.DB, jobID int, status JobStatus, errorMsg string) error {
	query := `
		UPDATE llm_jobs
		SET status = $1, error_message = $2, completed_at = CURRENT_TIMESTAMP, updated_at = CURRENT_TIMESTAMP
		WHERE id = $3
	`
	_, err := db.Exec(query, status, errorMsg, jobID)
	return err
}

// MarkJobRunning removed: jobs are marked running directly by JobRunner via
// UpdateJobStatus.

// IncrementJobRetry removed: inline jobs do not auto-retry.
// (Manual re-runs are handled by services.JobRunner.Retry.)

// GetJobStats retrieves statistics about jobs for a user
func GetJobStats(db *sql.DB, userID int) (*JobStats, error) {
	stats := &JobStats{
		ByType: make(map[JobType]int),
	}

	// Get counts by status
	query := `
		SELECT status, COUNT(*)
		FROM llm_jobs
		WHERE user_id = $1
		GROUP BY status
	`
	rows, err := db.Query(query, userID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	for rows.Next() {
		var status string
		var count int
		if err := rows.Scan(&status, &count); err != nil {
			return nil, err
		}
		stats.Total += count
		switch JobStatus(status) {
		case JobStatusPending:
			stats.Pending = count
		case JobStatusRunning:
			stats.Running = count
		case JobStatusCompleted:
			stats.Completed = count
		case JobStatusFailed:
			stats.Failed = count
		}
	}

	// Get counts by job type
	query = `
		SELECT job_type, COUNT(*)
		FROM llm_jobs
		WHERE user_id = $1
		GROUP BY job_type
	`
	rows, err = db.Query(query, userID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	for rows.Next() {
		var jobType string
		var count int
		if err := rows.Scan(&jobType, &count); err != nil {
			return nil, err
		}
		stats.ByType[JobType(jobType)] = count
	}

	return stats, nil
}

// DequeueJob, GetJobForUpdate, MarkJobRunning, IncrementJobRetry, and CancelJob
// have been removed: with inline processing (services.JobRunner) there is no
// claim/dequeue step, no retry counter that needs incrementing, and no
// pending job to cancel.

// CleanupOldJobs deletes jobs that have been completed/failed for longer than the retention period
// Returns the number of jobs deleted
func CleanupOldJobs(db *sql.DB, retentionDays int) (int, error) {
	if retentionDays <= 0 {
		retentionDays = 30 // Default to 30 days
	}

	// App-side time window (SQLite has no INTERVAL): compute the cutoff in Go so
	// the query runs identically on Postgres and SQLite. See migration design P3.
	cutoff := time.Now().UTC().AddDate(0, 0, -retentionDays)
	query := `
		DELETE FROM llm_jobs
		WHERE status IN ('completed', 'failed', 'cancelled')
		  AND completed_at < $1
	`
	result, err := db.Exec(query, cutoff)
	if err != nil {
		return 0, fmt.Errorf("failed to cleanup old jobs: %w", err)
	}

	count, _ := result.RowsAffected()
	return int(count), nil
}
