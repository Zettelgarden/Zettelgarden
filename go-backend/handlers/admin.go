package handlers

import (
	"database/sql"
	"encoding/json"
	"fmt"
	"log"
	"net"
	"net/http"
	"strings"

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

	// Clean up IP address: remove port if present (e.g., "127.0.0.1:1234" -> "127.0.0.1")
	if strings.Contains(ipAddress, ":") {
		host, _, err := net.SplitHostPort(ipAddress)
		if err == nil {
			ipAddress = host
		}
	}
	// If IP address is empty, use a placeholder
	if ipAddress == "" {
		ipAddress = "0.0.0.0"
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
	_, err = s.GetDB().Exec(query, userID, action, targetType, targetID, detailsJSON, ipAddress, r.UserAgent())
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

	query += whereClause + " ORDER BY created_at DESC, id DESC LIMIT $" + fmt.Sprint(len(args)+1) + " OFFSET $" + fmt.Sprint(len(args)+2)
	args = append(args, limit, offset)

	rows, err := s.GetDB().Query(query, args...)
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
