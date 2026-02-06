package services

import (
	"database/sql"

	"go-backend/models"
	"go-backend/services/tools/fact"

	"github.com/typesense/typesense-go/typesense"
)

// PHASE 3: Domain Package Migration
// ----------------------------------
// Fact-related functions are now implemented in the fact domain package.
// These are re-exports for backward compatibility during the migration period.

// GetCardFacts retrieves all facts associated with a specific card.
// Re-exported from services/tools/fact package.
func GetCardFacts(db *sql.DB, userID int, cardPK int) ([]models.Fact, error) {
	return fact.GetCardFacts(db, userID, cardPK)
}

// GetEntityFacts retrieves all facts linked to a specific entity.
// Re-exported from services/tools/fact package.
func GetEntityFacts(db *sql.DB, userID int, entityID int) ([]models.Fact, error) {
	return fact.GetEntityFacts(db, userID, entityID)
}

// GetFactCards retrieves all cards that are linked to a specific fact.
// Re-exported from services/tools/fact package.
func GetFactCards(db *sql.DB, userID int, factID int) ([]models.PartialCard, error) {
	cards, err := fact.GetFactCards(db, userID, factID)
	if err != nil {
		return nil, err
	}

	// Convert fact.PartialCard to models.PartialCard
	result := make([]models.PartialCard, len(cards))
	for i, c := range cards {
		result[i] = models.PartialCard{
			ID:        c.ID,
			CardID:    c.CardID,
			UserID:    c.UserID,
			Title:     c.Title,
			ParentID:  c.ParentID,
			CreatedAt: c.CreatedAt,
			UpdatedAt: c.UpdatedAt,
		}
	}
	return result, nil
}

// ExecuteFactTextSearch performs text-based search for facts.
// Re-exported from services/tools/fact package for backward compatibility.
func ExecuteFactTextSearch(db *sql.DB, userID int, query string, limit int, typesenseClient *typesense.Client) ([]map[string]interface{}, error) {
	return fact.ExecuteFactTextSearch(db, userID, query, limit, typesenseClient)
}

// ExecuteFactSemanticSearch performs semantic similarity search for facts.
// Re-exported from services/tools/fact package for backward compatibility.
func ExecuteFactSemanticSearch(db *sql.DB, userID int, query string, limit int, typesenseClient *typesense.Client) ([]map[string]interface{}, error) {
	return fact.ExecuteFactSemanticSearch(db, userID, query, limit, typesenseClient)
}