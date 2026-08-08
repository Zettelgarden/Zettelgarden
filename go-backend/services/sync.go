package services

import (
	"database/sql"
	"fmt"
	"log"

	"go-backend/models"

	"github.com/google/uuid"
)

// Sync collections (epic Zettelgarden-v5b, Phase 0a). These are the pinned
// offline-writable tables. Junction tables (card_tags, task_tags) are
// server-derived from card body / task title and deliberately do NOT emit —
// see AddTagsFromCard/AddTagsFromTask.
const (
	SyncCollectionCards = "cards"
	SyncCollectionTasks = "tasks"
	SyncCollectionTags  = "tags"
)

// Sync ops recorded in sync_log.
const (
	SyncOpUpsert = "upsert"
	SyncOpDelete = "delete"
)

// EmitChange records one entry in the append-only sync_log change feed. It
// MUST be called on the same Database handle as the mutation it describes (the
// service layer's convention; the handler test harness passes a *sql.Tx, so
// tests get true atomicity — in production the handle is the pool, matching
// how CreateAuditEvent and the user_stats counters already work).
//
// sync_log is the incremental feed for the local-first sync engine: the sync
// API (Phase 0b) joins these entries with the current row state by row_uuid to
// serve the changes feed. The log is never pruned while a client cursor trails
// it; a failed emit means the mutation is invisible to sync, so callers must
// propagate the error.
func EmitChange(db models.Database, userID int, collection, rowUUID, op string, version int) error {
	_, err := db.Exec(
		`INSERT INTO sync_log (user_id, collection, row_uuid, op, version) VALUES ($1, $2, $3, $4, $5)`,
		userID, collection, rowUUID, op, version,
	)
	if err != nil {
		log.Printf("sync_log emit failed (user %d, %s %s %s v%d): %v", userID, collection, op, rowUUID, version, err)
	}
	return err
}

// ensureSyncIdentity returns the (sync_uuid, version) for a row, lazily
// assigning a canonical UUID to rows that still have NULL sync_uuid (legacy
// rows that predate the Phase 0a back-fill, or test fixtures inserted with raw
// SQL). It is the single seam where the service layer reads the sync identity
// after a mutation; the version is the row's CURRENT version, which the
// mutation's UPDATE already bumped (callers add `version = version + 1`).
func ensureSyncIdentity(db models.Database, table string, id int) (string, int, error) {
	var syncUUID sql.NullString
	var version int
	err := db.QueryRow(
		fmt.Sprintf(`SELECT sync_uuid, version FROM %s WHERE id = $1`, table), id,
	).Scan(&syncUUID, &version)
	if err != nil {
		return "", 0, fmt.Errorf("ensureSyncIdentity %s %d: %w", table, id, err)
	}
	if syncUUID.Valid {
		return syncUUID.String, version, nil
	}
	assigned := uuid.New().String()
	if _, err := db.Exec(
		fmt.Sprintf(`UPDATE %s SET sync_uuid = $1 WHERE id = $2`, table), assigned, id,
	); err != nil {
		return "", 0, fmt.Errorf("assign sync_uuid %s %d: %w", table, id, err)
	}
	return assigned, version, nil
}

// emitRowChange reads the sync identity for a row and writes its sync_log
// entry. Callers use it after any mutation that bumps `version`.
func emitRowChange(db models.Database, userID int, table string, id int, op string) error {
	rowUUID, version, err := ensureSyncIdentity(db, table, id)
	if err != nil {
		return err
	}
	return EmitChange(db, userID, table, rowUUID, op, version)
}
