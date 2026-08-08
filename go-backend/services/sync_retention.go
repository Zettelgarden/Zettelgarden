package services

import (
	"database/sql"
	"fmt"
	"time"

	"go-backend/models"
)

// sync_log retention (epic Zettelgarden-v5b, Phase 0a — issue v5b.5).
//
// Policy: sync_log is append-only but NOT unbounded. Rows are pruned when they
// are BOTH:
//   - older than SyncLogRetention (30 days by default), AND
//   - at or below the oldest cursor any ACTIVE client is trailing (a device
//     whose last push was within the retention window).
//
// A device that has not pushed within the retention window no longer blocks
// pruning; when it returns, its stale cursor is older than the pruned boundary
// and the changes feed answers with reset=true, forcing a re-bootstrap
// (snapshot) instead of an impossible incremental catch-up.
//
// Clients report their cursor on every push (SyncPushRequest.Cursor); the push
// handler upserts the sync_clients heartbeat. A read-only client that never
// pushes is not tracked and may occasionally be forced to re-bootstrap after
// pruning — acceptable for the v1 thin-client scope.
const SyncLogRetention = 30 * 24 * time.Hour

// UpsertSyncClient records a device's last-known sync cursor (heartbeat). It
// is called transactionally from the push path, so the cursor a device reports
// is never ahead of what the server has actually applied.
func UpsertSyncClient(db models.Database, userID int, deviceID string, cursor int64) error {
	_, err := db.Exec(
		`INSERT INTO sync_clients (user_id, device_id, cursor, last_seen_at)
		 VALUES ($1, $2, $3, $4)
		 ON CONFLICT (user_id, device_id) DO UPDATE SET cursor = $3, last_seen_at = $4`,
		userID, deviceID, cursor, time.Now().UTC())
	return err
}

// PruneSyncLog applies the retention policy for one user, deleting sync_log
// rows older than `retention` that no active client cursor trails. It returns
// the number of rows deleted. Callers pass the current time so tests can
// backdate deterministically.
func PruneSyncLog(db models.Database, userID int, now time.Time, retention time.Duration) (int, error) {
	// Oldest cursor among devices seen within the retention window.
	var oldest sql.NullInt64
	err := db.QueryRow(
		`SELECT MIN(cursor) FROM sync_clients WHERE user_id = $1 AND last_seen_at >= $2`,
		userID, now.Add(-retention),
	).Scan(&oldest)
	if err != nil {
		return 0, fmt.Errorf("sync_clients min cursor: %w", err)
	}
	if !oldest.Valid {
		return 0, nil // no tracked active client; nothing to prune
	}
	// sync_log.created_at is written by the schema default datetime('now')
	// (UTC, "YYYY-MM-DD HH:MM:SS"); format the threshold identically so the
	// TEXT comparison is lexicographically correct.
	threshold := now.Add(-retention).UTC().Format("2006-01-02 15:04:05")
	res, err := db.Exec(
		`DELETE FROM sync_log WHERE user_id = $1 AND id <= $2 AND created_at < $3`,
		userID, oldest.Int64, threshold,
	)
	if err != nil {
		return 0, fmt.Errorf("prune sync_log: %w", err)
	}
	n, _ := res.RowsAffected()
	return int(n), nil
}
