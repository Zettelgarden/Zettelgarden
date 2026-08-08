package services

import (
	"database/sql"

	"go-backend/models"
)

// tablesNotCascaded lists user-owned tables whose rows are NOT removed by the
// users(id) ON DELETE CASCADE (their user_id columns predate the FK or have no
// ON DELETE action). These must be cleaned up explicitly before the user row
// itself is deleted:
//
//   - task_saved_searches, starred_cards, inactive_cards: user_id has no FK
//   - keywords: card_pk has a NO ACTION FK to cards and user_id has no FK —
//     without pre-clearing, the cascade delete of cards violates the FK
//   - entity_card_junction: entity_id has a NO ACTION FK to entities
//   - files: user_id has no FK (rows are normally removed via created_by, but
//     a file uploaded by another user on behalf of this one would survive)
var tablesNotCascaded = []string{
	"task_saved_searches",
	"starred_cards",
	"inactive_cards",
	"keywords",
	"entity_card_junction",
	"files",
}

// CountAdmins returns the number of admin users.
func CountAdmins(db models.Database) (int, error) {
	var n int
	err := db.QueryRow(`SELECT COUNT(*) FROM users WHERE is_admin = 1`).Scan(&n)
	return n, err
}

// DeleteUserData deletes all database rows owned by userID within the given
// database handle (a transaction in production, the test tx in tests):
//
//   - file metadata rows are collected first (their blobs live on disk and are
//     removed by the caller after commit);
//   - rows in tablesNotCascaded are deleted explicitly (no ON DELETE CASCADE);
//   - the users row is deleted last, letting SQLite cascade the remaining
//     ~40 user-owned tables (all of which declare ON DELETE CASCADE).
//
// It returns the storage keys of the user's file blobs so the caller can
// remove the bytes from disk. The caller owns transaction lifecycle.
func DeleteUserData(db models.Database, userID int) ([]string, error) {
	rows, err := db.Query(`SELECT path, thumbnail_path FROM files WHERE user_id = $1`, userID)
	if err != nil {
		return nil, err
	}
	var keys []string
	for rows.Next() {
		var path string
		var thumb sql.NullString
		if err := rows.Scan(&path, &thumb); err != nil {
			rows.Close()
			return nil, err
		}
		if path != "" {
			keys = append(keys, path)
		}
		if thumb.Valid && thumb.String != "" {
			keys = append(keys, thumb.String)
		}
	}
	rows.Close()
	if err := rows.Err(); err != nil {
		return nil, err
	}

	for _, table := range tablesNotCascaded {
		if _, err := db.Exec(`DELETE FROM `+table+` WHERE user_id = $1`, userID); err != nil {
			return nil, err
		}
	}

	res, err := db.Exec(`DELETE FROM users WHERE id = $1`, userID)
	if err != nil {
		return nil, err
	}
	n, err := res.RowsAffected()
	if err != nil {
		return nil, err
	}
	if n == 0 {
		return nil, sql.ErrNoRows
	}
	return keys, nil
}
