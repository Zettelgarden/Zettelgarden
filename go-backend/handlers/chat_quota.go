package handlers

import (
	"database/sql"
	"encoding/json"
	"fmt"
	"go-backend/models"
	"log"
	"net/http"
)

// GetUsageQuotaRoute gets user's usage quota status
func (s *Handler) GetUsageQuotaRoute(w http.ResponseWriter, r *http.Request) {
	userID := r.Context().Value("current_user").(int)

	quotas, err := s.GetChatUsageQuotas(userID)
	if err != nil {
		log.Printf("Error getting usage quotas: %v", err)
		http.Error(w, "Failed to get usage quotas", http.StatusInternalServerError)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(quotas)
}

// CheckChatUsageQuota checks if user has exceeded their quota
func (s *Handler) CheckChatUsageQuota(userID int, quotaType string) error {
	quota, err := s.getChatUsageQuota(userID, quotaType)
	if err != nil {
		// If no quota exists, create default quotas
		if err == sql.ErrNoRows {
			err = s.initializeDefaultQuotas(userID)
			if err != nil {
				return err
			}
			quota, err = s.getChatUsageQuota(userID, quotaType)
			if err != nil {
				return err
			}
		} else {
			return err
		}
	}

	// Check if quota is exceeded
	if quota.CurrentUsage >= quota.MaxLimit {
		return fmt.Errorf("quota exceeded for %s", quotaType)
	}

	return nil
}

// IncrementChatUsageQuota increments the usage counter
func (s *Handler) IncrementChatUsageQuota(userID int, quotaType string) error {
	query := `
		INSERT INTO chat_usage_quotas (user_id, quota_type, current_usage, max_limit, reset_date)
		VALUES ($1, $2, 1, $3, CURRENT_DATE)
		ON CONFLICT (user_id, quota_type, reset_date)
		DO UPDATE SET
			current_usage = chat_usage_quotas.current_usage + 1,
			updated_at = NOW()
	`

	maxLimit := s.getDefaultQuotaLimit(quotaType)
	_, err := s.DB.Exec(query, userID, quotaType, maxLimit)
	return err
}

// GetChatUsageQuotas gets all usage quotas for a user
func (s *Handler) GetChatUsageQuotas(userID int) ([]models.ChatUsageQuota, error) {
	query := `
		SELECT id, user_id, quota_type, current_usage, max_limit, reset_date, created_at, updated_at
		FROM chat_usage_quotas
		WHERE user_id = $1 AND reset_date = CURRENT_DATE
		ORDER BY quota_type
	`

	rows, err := s.DB.Query(query, userID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var quotas []models.ChatUsageQuota
	for rows.Next() {
		var quota models.ChatUsageQuota
		err := rows.Scan(
			&quota.ID,
			&quota.UserID,
			&quota.QuotaType,
			&quota.CurrentUsage,
			&quota.MaxLimit,
			&quota.ResetDate,
			&quota.CreatedAt,
			&quota.UpdatedAt,
		)
		if err != nil {
			return nil, err
		}
		quotas = append(quotas, quota)
	}

	return quotas, nil
}

// getChatUsageQuota gets a single usage quota for a user
func (s *Handler) getChatUsageQuota(userID int, quotaType string) (*models.ChatUsageQuota, error) {
	query := `
		SELECT id, user_id, quota_type, current_usage, max_limit, reset_date, created_at, updated_at
		FROM chat_usage_quotas
		WHERE user_id = $1 AND quota_type = $2 AND reset_date = CURRENT_DATE
	`

	var quota models.ChatUsageQuota
	err := s.DB.QueryRow(query, userID, quotaType).Scan(
		&quota.ID,
		&quota.UserID,
		&quota.QuotaType,
		&quota.CurrentUsage,
		&quota.MaxLimit,
		&quota.ResetDate,
		&quota.CreatedAt,
		&quota.UpdatedAt,
	)

	return &quota, err
}

// initializeDefaultQuotas initializes default quotas for a user
func (s *Handler) initializeDefaultQuotas(userID int) error {
	quotas := []struct {
		quotaType string
		maxLimit  int
	}{
		{"messages_per_day", s.getDefaultQuotaLimit("messages_per_day")},
		{"tool_calls_per_day", s.getDefaultQuotaLimit("tool_calls_per_day")},
		{"conversations_per_day", s.getDefaultQuotaLimit("conversations_per_day")},
	}

	for _, quota := range quotas {
		query := `
			INSERT INTO chat_usage_quotas (user_id, quota_type, current_usage, max_limit, reset_date)
			VALUES ($1, $2, 0, $3, CURRENT_DATE)
			ON CONFLICT (user_id, quota_type, reset_date) DO NOTHING
		`
		_, err := s.DB.Exec(query, userID, quota.quotaType, quota.maxLimit)
		if err != nil {
			return err
		}
	}

	return nil
}

// getDefaultQuotaLimit returns the default quota limit for a given quota type
func (s *Handler) getDefaultQuotaLimit(quotaType string) int {
	// TODO: Check user subscription level for PRO limits
	switch quotaType {
	case "messages_per_day":
		return 50 // Free tier limit
	case "tool_calls_per_day":
		return 100 // Free tier limit
	case "conversations_per_day":
		return 10 // Free tier limit
	default:
		return 10
	}
}