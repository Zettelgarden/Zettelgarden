package handlers

import (
	"database/sql"
	"encoding/json"
	"fmt"
	"log"
	"net/http"
	"strconv"

	"github.com/gorilla/mux"
)

// AdminAuditLog represents an audit log entry for admin actions
type AdminAuditLog struct {
	ID          int                    `json:"id"`
	AdminUserID int                    `json:"admin_user_id"`
	Action      string                 `json:"action"`
	TargetType  string                 `json:"target_type"`
	TargetID    sql.NullInt32          `json:"target_id"`
	Details     map[string]interface{} `json:"details"`
	IPAddress   sql.NullString         `json:"ip_address"`
	UserAgent   sql.NullString         `json:"user_agent"`
	CreatedAt   string                 `json:"created_at"`
}

// LogAdminAction logs an admin action to the audit log.
// This should be called after successful completion of admin operations.
//
// Parameters:
//   - r: The HTTP request (used to get admin user from context and extract IP/user-agent)
//   - action: The action performed (e.g., "user.update", "mailing_list.unsubscribe")
//   - targetType: Type of entity affected (e.g., "user", "mailing_list")
//   - targetID: ID of the affected entity (0 if no specific target)
//   - details: Additional context about the action (before/after values, reasons, etc.)
func (s *Handler) LogAdminAction(r *http.Request, action string, targetType string, targetID int, details map[string]interface{}) {
	userID, ok := r.Context().Value("current_user").(int)
	if !ok {
		log.Printf("LogAdminAction: current_user not found in context")
		return
	}

	// Extract IP address
	ipAddress := r.RemoteAddr
	// Check for X-Forwarded-For header (proxy/load balancer)
	if xff := r.Header.Get("X-Forwarded-For"); xff != "" {
		ipAddress = xff
	}
	// Check for X-Real-IP header
	if xri := r.Header.Get("X-Real-IP"); xri != "" {
		ipAddress = xri
	}

	// Convert details to JSONB
	detailsJSON, err := json.Marshal(details)
	if err != nil {
		log.Printf("LogAdminAction: error marshaling details: %v", err)
		detailsJSON = []byte("{}")
	}

	// Insert audit log entry
	query := `
		INSERT INTO admin_audit_log (admin_user_id, action, target_type, target_id, details, ip_address, user_agent)
		VALUES ($1, $2, $3, $4, $5, $6, $7)
	`
	_, err = s.DB.Exec(query, userID, action, targetType, targetID, detailsJSON, ipAddress, r.UserAgent())
	if err != nil {
		log.Printf("LogAdminAction: error inserting audit log: %v", err)
	}
}

// LogAdminActionAsync logs an admin action asynchronously.
// Use this for performance-critical paths where audit logging shouldn't block.
func (s *Handler) LogAdminActionAsync(r *http.Request, action string, targetType string, targetID int, details map[string]interface{}) {
	go func() {
		s.LogAdminAction(r, action, targetType, targetID, details)
	}()
}

// GetAdminAuditLogs retrieves audit logs with optional filtering.
// Admin only.
func (s *Handler) GetAdminAuditLogs(limit int, offset int, actionFilter string, targetTypeFilter string) ([]AdminAuditLog, error) {
	query := `
		SELECT id, admin_user_id, action, target_type, target_id, details, ip_address, user_agent, created_at
		FROM admin_audit_log
	`
	args := []interface{}{}
	whereClause := ""

	if actionFilter != "" {
		whereClause += " WHERE action = $" + fmt.Sprint(len(args)+1)
		args = append(args, actionFilter)
	}
	if targetTypeFilter != "" {
		if whereClause == "" {
			whereClause += " WHERE"
		} else {
			whereClause += " AND"
		}
		whereClause += " target_type = $" + fmt.Sprint(len(args)+1)
		args = append(args, targetTypeFilter)
	}

	query += whereClause + " ORDER BY created_at DESC LIMIT $" + fmt.Sprint(len(args)+1) + " OFFSET $" + fmt.Sprint(len(args)+2)
	args = append(args, limit, offset)

	rows, err := s.DB.Query(query, args...)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var logs []AdminAuditLog
	for rows.Next() {
		var auditLog AdminAuditLog
		var detailsJSON []byte
		err := rows.Scan(
			&auditLog.ID,
			&auditLog.AdminUserID,
			&auditLog.Action,
			&auditLog.TargetType,
			&auditLog.TargetID,
			&detailsJSON,
			&auditLog.IPAddress,
			&auditLog.UserAgent,
			&auditLog.CreatedAt,
		)
		if err != nil {
			return nil, err
		}
		if err := json.Unmarshal(detailsJSON, &auditLog.Details); err != nil {
			log.Printf("GetAdminAuditLogs: error unmarshaling details: %v", err)
			auditLog.Details = make(map[string]interface{})
		}
		logs = append(logs, auditLog)
	}

	return logs, nil
}

// AdminMiddleware checks if the current user is an admin.
// Returns 403 Forbidden if the user is not an admin.
// Must be used after APIKeyOrJWTMiddleware which sets current_user in context.
func (s *Handler) AdminMiddleware(next http.HandlerFunc) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		userID, ok := r.Context().Value("current_user").(int)
		if !ok {
			log.Printf("AdminMiddleware: current_user not found in context")
			http.Error(w, "Unauthorized", http.StatusUnauthorized)
			return
		}

		user, err := s.QueryUser(userID)
		if err != nil {
			log.Printf("AdminMiddleware: error querying user %d: %v", userID, err)
			http.Error(w, "Error verifying permissions", http.StatusInternalServerError)
			return
		}

		if !user.IsAdmin {
			log.Printf("AdminMiddleware: user %d (%s) attempted to access admin route %s", userID, user.Username, r.URL.Path)
			http.Error(w, "Access denied. Admin privileges required.", http.StatusForbidden)
			return
		}

		next(w, r)
	}
}

// AdminOrSelfMiddleware allows access if:
// 1. The current user is an admin, OR
// 2. The current user is accessing their own resource (id matches)
//
// The idParam should be the URL variable name for the user ID (e.g., "id" for /api/users/{id})
// Must be used after APIKeyOrJWTMiddleware which sets current_user in context.
func (s *Handler) AdminOrSelfMiddleware(idParam string) func(http.HandlerFunc) http.HandlerFunc {
	return func(next http.HandlerFunc) http.HandlerFunc {
		return func(w http.ResponseWriter, r *http.Request) {
			userID, ok := r.Context().Value("current_user").(int)
			if !ok {
				log.Printf("AdminOrSelfMiddleware: current_user not found in context")
				http.Error(w, "Unauthorized", http.StatusUnauthorized)
				return
			}

			user, err := s.QueryUser(userID)
			if err != nil {
				log.Printf("AdminOrSelfMiddleware: error querying user %d: %v", userID, err)
				http.Error(w, "Error verifying permissions", http.StatusInternalServerError)
				return
			}

			// Admins can access any resource
			if user.IsAdmin {
				next(w, r)
				return
			}

			// Non-admins can only access their own resources
			vars := mux.Vars(r)
			requestedIDStr, ok := vars[idParam]
			if !ok {
				log.Printf("AdminOrSelfMiddleware: id param '%s' not found in URL", idParam)
				http.Error(w, "Invalid request", http.StatusBadRequest)
				return
			}

			var requestedID int
			_, err = fmt.Sscanf(requestedIDStr, "%d", &requestedID)
			if err != nil {
				log.Printf("AdminOrSelfMiddleware: invalid id format: %s", requestedIDStr)
				http.Error(w, "Invalid ID format", http.StatusBadRequest)
				return
			}

			if user.ID != requestedID {
				log.Printf("AdminOrSelfMiddleware: user %d attempted to access user %d's resource", userID, requestedID)
				http.Error(w, "Access denied. You can only access your own resources.", http.StatusForbidden)
				return
			}

			next(w, r)
		}
	}
}

// EmailQueueStats represents statistics about the email queue
type EmailQueueStats struct {
	PendingEmails   int `json:"pending_emails"`
	RunningEmails   int `json:"running_emails"`
	FailedEmails    int `json:"failed_emails"`
	CompletedEmails int `json:"completed_emails"`
	TotalEmails     int `json:"total_emails"`
}

// FailedEmail represents a failed email job with details
type FailedEmail struct {
	ID           int                    `json:"id"`
	UserID       int                    `json:"user_id"`
	Subject      string                 `json:"subject"`
	Recipient    string                 `json:"recipient"`
	ErrorMessage string                 `json:"error_message"`
	CreatedAt    string                 `json:"created_at"`
	CompletedAt  string                 `json:"completed_at"`
	RetryCount   int                    `json:"retry_count"`
	Payload      map[string]interface{} `json:"payload"`
}

// GetEmailQueueStatsRoute retrieves statistics about the email queue
// Admin only
func (s *Handler) GetEmailQueueStatsRoute(w http.ResponseWriter, r *http.Request) {
	stats := EmailQueueStats{}

	// Get counts by status for email jobs
	query := `
		SELECT status, COUNT(*)
		FROM llm_jobs
		WHERE job_type = 'email'
		GROUP BY status
	`
	rows, err := s.DB.Query(query)
	if err != nil {
		log.Printf("GetEmailQueueStatsRoute: error querying email stats: %v", err)
		http.Error(w, "Failed to get email queue stats", http.StatusInternalServerError)
		return
	}
	defer rows.Close()

	for rows.Next() {
		var status string
		var count int
		if err := rows.Scan(&status, &count); err != nil {
			log.Printf("GetEmailQueueStatsRoute: error scanning row: %v", err)
			continue
		}
		stats.TotalEmails += count
		switch status {
		case "pending":
			stats.PendingEmails = count
		case "running":
			stats.RunningEmails = count
		case "failed":
			stats.FailedEmails = count
		case "completed":
			stats.CompletedEmails = count
		}
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(stats)
}

// GetFailedEmailsRoute retrieves failed emails with optional filtering and pagination
// Admin only
func (s *Handler) GetFailedEmailsRoute(w http.ResponseWriter, r *http.Request) {
	// Parse query parameters
	page, _ := strconv.Atoi(r.URL.Query().Get("page"))
	perPage, _ := strconv.Atoi(r.URL.Query().Get("per_page"))
	userIDFilter := r.URL.Query().Get("user_id")

	// Set defaults
	if page < 1 {
		page = 1
	}
	if perPage < 1 || perPage > 100 {
		perPage = 50
	}

	offset := (page - 1) * perPage

	// Build query
	query := `
		SELECT id, user_id, payload, error_message, created_at, completed_at, retry_count
		FROM llm_jobs
		WHERE job_type = 'email' AND status = 'failed'
	`
	args := []interface{}{}
	argIdx := 1

	if userIDFilter != "" {
		userID, err := strconv.Atoi(userIDFilter)
		if err == nil {
			query += fmt.Sprintf(" AND user_id = $%d", argIdx)
			args = append(args, userID)
			argIdx++
		}
	}

	query += " ORDER BY created_at DESC"

	// Get total count
	countQuery := "SELECT COUNT(*) FROM (" + query + ") AS subq"
	var total int
	err := s.DB.QueryRow(countQuery, args...).Scan(&total)
	if err != nil {
		log.Printf("GetFailedEmailsRoute: error getting count: %v", err)
		http.Error(w, "Failed to get failed emails", http.StatusInternalServerError)
		return
	}

	// Add pagination
	query += fmt.Sprintf(" LIMIT $%d OFFSET $%d", argIdx, argIdx+1)
	args = append(args, perPage, offset)

	rows, err := s.DB.Query(query, args...)
	if err != nil {
		log.Printf("GetFailedEmailsRoute: error querying failed emails: %v", err)
		http.Error(w, "Failed to get failed emails", http.StatusInternalServerError)
		return
	}
	defer rows.Close()

	emails := []FailedEmail{}
	for rows.Next() {
		var e FailedEmail
		var payloadJSON []byte
		var completedAt sql.NullTime

		err := rows.Scan(
			&e.ID,
			&e.UserID,
			&payloadJSON,
			&e.ErrorMessage,
			&e.CreatedAt,
			&completedAt,
			&e.RetryCount,
		)
		if err != nil {
			log.Printf("GetFailedEmailsRoute: error scanning row: %v", err)
			continue
		}

		// Unmarshal payload
		if err := json.Unmarshal(payloadJSON, &e.Payload); err != nil {
			log.Printf("GetFailedEmailsRoute: error unmarshaling payload: %v", err)
			e.Payload = make(map[string]interface{})
		}

		// Extract subject and recipient from payload
		if subject, ok := e.Payload["subject"].(string); ok {
			e.Subject = subject
		}
		if recipient, ok := e.Payload["recipient"].(string); ok {
			e.Recipient = recipient
		}

		if completedAt.Valid {
			e.CompletedAt = completedAt.Time.Format("2006-01-02T15:04:05Z07:00")
		}

		emails = append(emails, e)
	}

	// Parse response
	response := struct {
		Emails     []FailedEmail `json:"emails"`
		Total      int           `json:"total"`
		Page       int           `json:"page"`
		PerPage    int           `json:"per_page"`
		TotalPages int           `json:"total_pages"`
	}{
		Emails:     emails,
		Total:      total,
		Page:       page,
		PerPage:    perPage,
		TotalPages: (total + perPage - 1) / perPage,
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(response)
}

// RetryFailedEmailRoute retries a failed email job
// Admin only
func (s *Handler) RetryFailedEmailRoute(w http.ResponseWriter, r *http.Request) {
	vars := mux.Vars(r)
	jobIDStr := vars["id"]

	jobID, err := strconv.Atoi(jobIDStr)
	if err != nil {
		http.Error(w, "Invalid job ID", http.StatusBadRequest)
		return
	}

	// Check if job exists and is a failed email
	var exists bool
	err = s.DB.QueryRow("SELECT EXISTS(SELECT 1 FROM llm_jobs WHERE id = $1 AND job_type = 'email' AND status = 'failed')", jobID).Scan(&exists)
	if err != nil {
		log.Printf("RetryFailedEmailRoute: error checking job: %v", err)
		http.Error(w, "Failed to retry email", http.StatusInternalServerError)
		return
	}

	if !exists {
		http.Error(w, "Failed email not found", http.StatusNotFound)
		return
	}

	// Reset job to pending and clear error
	query := `
		UPDATE llm_jobs
		SET status = 'pending',
		    error_message = '',
		    retry_count = 0,
		    updated_at = NOW()
		WHERE id = $1
	`
	_, err = s.DB.Exec(query, jobID)
	if err != nil {
		log.Printf("RetryFailedEmailRoute: error retrying email: %v", err)
		http.Error(w, "Failed to retry email", http.StatusInternalServerError)
		return
	}

	// Log admin action
	s.LogAdminAction(r, "email.retry", "email", jobID, map[string]interface{}{
		"action": "retried_failed_email",
	})

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(map[string]string{
		"message": "Email queued for retry",
	})
}

// DeleteFailedEmailRoute permanently deletes a failed email job
// Admin only
func (s *Handler) DeleteFailedEmailRoute(w http.ResponseWriter, r *http.Request) {
	vars := mux.Vars(r)
	jobIDStr := vars["id"]

	jobID, err := strconv.Atoi(jobIDStr)
	if err != nil {
		http.Error(w, "Invalid job ID", http.StatusBadRequest)
		return
	}

	// Check if job exists and is a failed email, get details for logging
	var exists bool
	var userID int
	var payloadJSON []byte
	err = s.DB.QueryRow("SELECT EXISTS(SELECT 1 FROM llm_jobs WHERE id = $1 AND job_type = 'email' AND status = 'failed'), user_id, payload", jobID).Scan(&exists, &userID, &payloadJSON)
	if err != nil && err != sql.ErrNoRows {
		log.Printf("DeleteFailedEmailRoute: error checking job: %v", err)
		http.Error(w, "Failed to delete email", http.StatusInternalServerError)
		return
	}

	if !exists {
		http.Error(w, "Failed email not found", http.StatusNotFound)
		return
	}

	// Delete the job
	_, err = s.DB.Exec("DELETE FROM llm_jobs WHERE id = $1", jobID)
	if err != nil {
		log.Printf("DeleteFailedEmailRoute: error deleting email: %v", err)
		http.Error(w, "Failed to delete email", http.StatusInternalServerError)
		return
	}

	// Log admin action
	s.LogAdminAction(r, "email.delete", "email", jobID, map[string]interface{}{
		"action":  "deleted_failed_email",
		"user_id": userID,
	})

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(map[string]string{
		"message": "Failed email deleted",
	})
}