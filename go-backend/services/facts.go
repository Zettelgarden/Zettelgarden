package services

import (
	"database/sql"
	"go-backend/models"
)

func GetCardFacts(db *sql.DB, userID int, cardPK int) ([]models.Fact, error) {
	rows, err := db.Query(`
		SELECT f.id, f.user_id, f.card_pk, f.fact, f.created_at, f.updated_at
		FROM facts f
		JOIN fact_card_junction fcj ON f.id = fcj.fact_id
		WHERE fcj.card_pk = $1 AND fcj.user_id = $2
		ORDER BY f.created_at DESC
	`, cardPK, userID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var facts []models.Fact
	for rows.Next() {
		var f models.Fact
		if err := rows.Scan(&f.ID, &f.UserID, &f.CardPK, &f.Fact, &f.CreatedAt, &f.UpdatedAt); err != nil {
			return nil, err
		}
		facts = append(facts, f)
	}
	return facts, nil
}

func GetEntityFacts(db *sql.DB, userID int, entityID int) ([]models.Fact, error) {
	rows, err := db.Query(`
		SELECT f.id, f.user_id, f.card_pk, f.fact, f.created_at, f.updated_at
		FROM facts f
		JOIN entity_fact_junction efj ON f.id = efj.fact_id
		WHERE efj.entity_id = $1 AND efj.user_id = $2
		ORDER BY f.created_at DESC
	`, entityID, userID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var facts []models.Fact
	for rows.Next() {
		var f models.Fact
		if err := rows.Scan(&f.ID, &f.UserID, &f.CardPK, &f.Fact, &f.CreatedAt, &f.UpdatedAt); err != nil {
			return nil, err
		}
		facts = append(facts, f)
	}
	return facts, nil
}

func GetFactCards(db *sql.DB, userID int, factID int) ([]models.PartialCard, error) {
	rows, err := db.Query(`
		SELECT c.id, c.card_id, c.user_id, c.title, c.parent_id, c.created_at, c.updated_at
		FROM cards c
		JOIN fact_card_junction fcj ON c.id = fcj.card_pk
		WHERE fcj.fact_id = $1 AND fcj.user_id = $2
	`, factID, userID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var cards []models.PartialCard
	for rows.Next() {
		var c models.PartialCard
		if err := rows.Scan(&c.ID, &c.CardID, &c.UserID, &c.Title, &c.ParentID, &c.CreatedAt, &c.UpdatedAt); err != nil {
			return nil, err
		}
		cards = append(cards, c)
	}
	return cards, nil
}