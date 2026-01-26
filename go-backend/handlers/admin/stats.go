package admin

import (
	"database/sql"
	"encoding/json"
	"go-backend/handlers"
	"log"
	"net/http"
)

// GetAdminStatsRoute returns summary statistics for the admin dashboard
// admin protected (via middleware)
func GetAdminStatsRoute(h *handlers.Handler, w http.ResponseWriter, r *http.Request) {
	// Get user statistics
	userStats, err := getUserStats(h)
	if err != nil {
		log.Printf("Error getting user stats: %v", err)
		http.Error(w, "Error retrieving statistics", http.StatusInternalServerError)
		return
	}

	// Get subscription statistics
	subStats, err := getSubscriptionStats(h)
	if err != nil {
		log.Printf("Error getting subscription stats: %v", err)
		http.Error(w, "Error retrieving statistics", http.StatusInternalServerError)
		return
	}

	// Get revenue statistics
	revStats, err := getRevenueStats(h)
	if err != nil {
		log.Printf("Error getting revenue stats: %v", err)
		http.Error(w, "Error retrieving statistics", http.StatusInternalServerError)
		return
	}

	// Get content statistics
	contentStats, err := getContentStats(h)
	if err != nil {
		log.Printf("Error getting content stats: %v", err)
		http.Error(w, "Error retrieving statistics", http.StatusInternalServerError)
		return
	}

	// Combine all stats
	stats := map[string]interface{}{
		"users":         userStats,
		"subscriptions": subStats,
		"revenue":       revStats,
		"content":       contentStats,
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(stats)
}

// getUserStats returns user-related statistics
func getUserStats(h *handlers.Handler) (map[string]interface{}, error) {
	stats := make(map[string]interface{})

	// Total users
	var totalUsers int
	err := h.DB.QueryRow(`SELECT COUNT(*) FROM users`).Scan(&totalUsers)
	if err != nil {
		return nil, err
	}
	stats["total"] = totalUsers

	// Active users (last 7 days)
	var activeWeek int
	err = h.DB.QueryRow(`
		SELECT COUNT(*) FROM users
		WHERE last_seen > NOW() - INTERVAL '7 days'
	`).Scan(&activeWeek)
	if err != nil {
		return nil, err
	}
	stats["active_this_week"] = activeWeek

	// Active users (last 30 days)
	var activeMonth int
	err = h.DB.QueryRow(`
		SELECT COUNT(*) FROM users
		WHERE last_seen > NOW() - INTERVAL '30 days'
	`).Scan(&activeMonth)
	if err != nil {
		return nil, err
	}
	stats["active_this_month"] = activeMonth

	// Total admins
	var totalAdmins int
	err = h.DB.QueryRow(`SELECT COUNT(*) FROM users WHERE is_admin = true`).Scan(&totalAdmins)
	if err != nil {
		return nil, err
	}
	stats["total_admins"] = totalAdmins

	// New users this week
	var newUsersWeek int
	err = h.DB.QueryRow(`
		SELECT COUNT(*) FROM users
		WHERE created_at > NOW() - INTERVAL '7 days'
	`).Scan(&newUsersWeek)
	if err != nil {
		return nil, err
	}
	stats["new_this_week"] = newUsersWeek

	// New users this month
	var newUsersMonth int
	err = h.DB.QueryRow(`
		SELECT COUNT(*) FROM users
		WHERE created_at > NOW() - INTERVAL '30 days'
	`).Scan(&newUsersMonth)
	if err != nil {
		return nil, err
	}
	stats["new_this_month"] = newUsersMonth

	return stats, nil
}

// getSubscriptionStats returns subscription-related statistics
func getSubscriptionStats(h *handlers.Handler) (map[string]interface{}, error) {
	stats := make(map[string]interface{})

	// Count by subscription status
	rows, err := h.DB.Query(`
		SELECT
			stripe_subscription_status,
			COUNT(*) as count
		FROM users
		GROUP BY stripe_subscription_status
		ORDER BY count DESC
	`)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	subscriptionCounts := make(map[string]int)
	total := 0
	for rows.Next() {
		var status sql.NullString
		var count int
		if err := rows.Scan(&status, &count); err != nil {
			return nil, err
		}
		statusStr := status.String
		if statusStr == "" {
			statusStr = "free"
		}
		subscriptionCounts[statusStr] = count
		total += count
	}

	stats["by_status"] = subscriptionCounts
	stats["total"] = total

	// Calculate totals for each category
	stats["active"] = subscriptionCounts["active"] + subscriptionCounts["trialing"]
	stats["free"] = subscriptionCounts["free"]
	stats["past_due"] = subscriptionCounts["past_due"]

	return stats, nil
}

// getRevenueStats returns revenue-related statistics
func getRevenueStats(h *handlers.Handler) (map[string]interface{}, error) {
	stats := make(map[string]interface{})

	// Total revenue
	var totalRevenue sql.NullFloat64
	err := h.DB.QueryRow(`
		SELECT COALESCE(SUM(amount_cents), 0) / 100.0
		FROM revenue
	`).Scan(&totalRevenue)
	if err != nil {
		return nil, err
	}
	stats["total_revenue"] = totalRevenue.Float64

	// Revenue this month
	var revenueThisMonth sql.NullFloat64
	err = h.DB.QueryRow(`
		SELECT COALESCE(SUM(amount_cents), 0) / 100.0
		FROM revenue
		WHERE created_at > NOW() - INTERVAL '30 days'
	`).Scan(&revenueThisMonth)
	if err != nil {
		return nil, err
	}
	stats["revenue_this_month"] = revenueThisMonth.Float64

	// Estimate MRR (monthly recurring revenue)
	// For simplicity, we'll sum all active/trialing user's last payment amount
	// In production, you'd want to track this more accurately
	var mrr sql.NullFloat64
	err = h.DB.QueryRow(`
		SELECT COALESCE(SUM(
			CASE
				WHEN stripe_subscription_status = 'active' THEN
					-- Get the monthly amount from their plan
					COALESCE(
						(SELECT amount_cents FROM revenue WHERE user_id = u.id
						  ORDER BY created_at DESC LIMIT 1),
						0
					) / 100.0
				WHEN stripe_subscription_status = 'trialing' THEN 0
				ELSE 0
			END
		), 0) as mrr
		FROM users u
		WHERE stripe_subscription_status IN ('active', 'trialing')
	`).Scan(&mrr)
	if err != nil {
		return nil, err
	}
	stats["monthly_recurring_revenue"] = mrr.Float64

	return stats, nil
}

// getContentStats returns content-related statistics
func getContentStats(h *handlers.Handler) (map[string]interface{}, error) {
	stats := make(map[string]interface{})

	// Card counts
	var totalCards int
	err := h.DB.QueryRow(`SELECT COUNT(*) FROM cards`).Scan(&totalCards)
	if err != nil {
		return nil, err
	}
	stats["total_cards"] = totalCards

	// Task counts
	var totalTasks int
	err = h.DB.QueryRow(`SELECT COUNT(*) FROM tasks`).Scan(&totalTasks)
	if err != nil {
		return nil, err
	}
	stats["total_tasks"] = totalTasks

	// File counts
	var totalFiles int
	err = h.DB.QueryRow(`SELECT COUNT(*) FROM files`).Scan(&totalFiles)
	if err != nil {
		return nil, err
	}
	stats["total_files"] = totalFiles

	// Chat message counts
	var totalMessages int
	err = h.DB.QueryRow(`
		SELECT COUNT(*) FROM chat_messages
	`).Scan(&totalMessages)
	if err != nil {
		return nil, err
	}
	stats["total_chat_messages"] = totalMessages

	// Entity counts
	var totalEntities int
	err = h.DB.QueryRow(`SELECT COUNT(*) FROM entities`).Scan(&totalEntities)
	if err != nil {
		return nil, err
	}
	stats["total_entities"] = totalEntities

	// Fact counts
	var totalFacts int
	err = h.DB.QueryRow(`SELECT COUNT(*) FROM facts`).Scan(&totalFacts)
	if err != nil {
		return nil, err
	}
	stats["total_facts"] = totalFacts

	return stats, nil
}
