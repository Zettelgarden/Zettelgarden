package models

import (
	"database/sql"
)

type FactCardJunction struct {
	FactID int64 `db:"fact_id" json:"fact_id"`
	CardID int64 `db:"card_pk" json:"card_pk"`
}

func LinkFactToCard(db *sql.DB, factID, cardID int64, userID int) error {
	_, err := db.Exec(`
		INSERT INTO fact_card_junction(fact_id, card_pk, user_id, is_origin, created_at, updated_at)
		VALUES ($1, $2, $3, FALSE, CURRENT_TIMESTAMP, CURRENT_TIMESTAMP)
		ON CONFLICT (fact_id, card_pk) DO UPDATE SET updated_at = CURRENT_TIMESTAMP
	`, factID, cardID, userID)
	return err
}

func UnlinkFactFromCard(db *sql.DB, factID, cardID int64, userID int) error {
	_, err := db.Exec(`
		DELETE FROM fact_card_junction
		WHERE fact_id = $1 AND card_pk = $2 AND user_id = $3
	`, factID, cardID, userID)
	return err
}
