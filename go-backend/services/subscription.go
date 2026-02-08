package services

import (
	"database/sql"
	"go-backend/models"
	"log"
)

// UserHasSubscription checks if a user has an active subscription
func UserHasSubscription(db models.Database, userID int) bool {
	var stripeSubscriptionStatus string
	err := db.QueryRow("SELECT stripe_subscription_status FROM users WHERE id = $1", userID).Scan(&stripeSubscriptionStatus)
	if err != nil {
		if err != sql.ErrNoRows {
			log.Printf("Error checking subscription for user %d: %v", userID, err)
		}
		return false
	}
	return stripeSubscriptionStatus == "active" || stripeSubscriptionStatus == "trialing"
}
