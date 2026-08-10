package handlers

import (
	"database/sql"
	"encoding/json"
	"errors"
	"log"
	"net/http"
	"strconv"
	"strings"
	"time"

	"go-backend/models"
	"go-backend/server"
	"go-backend/services"
)

// Sync API handlers (epic Zettelgarden-v5b, Phase 0b — issue tsv): snapshot
// bootstrap, incremental changes feed, and transactional batch push. These are
// the server-as-sync-hub surface consumed by the local-first clients; the
// existing REST surface is untouched.

// Column sets for sync row payloads (mirror of the raw table columns).
var syncCardColumns = `id, card_id, title, body, link, is_deleted, parent_id, created_at, updated_at, card_schema_id, structured_data, version, sync_uuid`
var syncTaskColumns = `id, card_pk, user_id, scheduled_date, due_date, created_at, updated_at, completed_at, title, description, priority, status, is_complete, is_deleted, reminder_time, reminder_sent, parent_task_id, sort_order, version, sync_uuid`
var syncTagColumns = `id, name, color, user_id, is_deleted, created_at, updated_at, version, sync_uuid`

const syncFeedPageSize = 500
const syncPushMaxBytes = 10 << 20 // 10 MiB per push batch

// parseCollectionsParam parses ?collections=cards,tasks,tags (default: all).
func parseCollectionsParam(r *http.Request) map[string]bool {
	all := map[string]bool{
		services.SyncCollectionCards: true,
		services.SyncCollectionTasks: true,
		services.SyncCollectionTags:  true,
	}
	raw := r.URL.Query().Get("collections")
	if raw == "" {
		return all
	}
	out := map[string]bool{}
	for _, c := range strings.Split(raw, ",") {
		c = strings.TrimSpace(c)
		if all[c] {
			out[c] = true
		}
	}
	return out
}

func validSyncCollection(c string) bool {
	return c == services.SyncCollectionCards || c == services.SyncCollectionTasks || c == services.SyncCollectionTags
}

func writeJSON(w http.ResponseWriter, status int, v any) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(status)
	if err := json.NewEncoder(w).Encode(v); err != nil {
		log.Printf("sync: write response: %v", err)
	}
}

// syncRowsFromQuery materializes a collection query (which must select the
// syncCardColumns/syncTaskColumns/syncTagColumns column sets) into SyncRows.
func syncRowsFromQuery(rows *sql.Rows, err error) ([]models.SyncRow, error) {
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	maps, err := services.RowsToJSON(rows)
	if err != nil {
		return nil, err
	}
	out := make([]models.SyncRow, 0, len(maps))
	for _, m := range maps {
		uuid, _ := m["sync_uuid"].(string)
		version, _ := m["version"].(int64)
		data, err := json.Marshal(m)
		if err != nil {
			return nil, err
		}
		out = append(out, models.SyncRow{
			RowUUID: uuid, Version: int(version), Op: services.SyncOpUpsert, Data: data,
		})
	}
	return out, nil
}

// SnapshotRoute serves the full current state of the requested collections,
// with a cursor captured in the same read transaction. A fresh client stores
// the cursor and pulls changes since it from there; reinstalls re-bootstrap
// the same way.
func (s *Handler) SnapshotRoute(w http.ResponseWriter, r *http.Request) {
	userID := r.Context().Value("current_user").(int)
	collections := parseCollectionsParam(r)

	tx, err := s.BeginReadTx()
	if err != nil {
		http.Error(w, "unable to start read transaction", http.StatusInternalServerError)
		return
	}
	// Read-only: release the tx in production. In tests BeginTx returns the
	// shared per-test transaction, which the framework owns (rolled back at
	// Teardown) — rolling it back here would kill the test's transaction for
	// every subsequent request.
	if s.ShouldCommitTx() {
		defer tx.Rollback()
	}

	var cursor int64
	if err := tx.QueryRow(`SELECT COALESCE(MAX(id), 0) FROM sync_log`).Scan(&cursor); err != nil {
		http.Error(w, "unable to read sync cursor", http.StatusInternalServerError)
		return
	}

	resp := models.SyncSnapshotResponse{
		Cursor:      cursor,
		Collections: map[string][]models.SyncRow{},
	}
	if collections[services.SyncCollectionCards] {
		if resp.Collections[services.SyncCollectionCards], err = syncRowsFromQuery(tx.Query(
			`SELECT `+syncCardColumns+` FROM cards WHERE user_id = $1 AND is_deleted = FALSE`, userID)); err != nil {
			http.Error(w, "unable to read cards", http.StatusInternalServerError)
			return
		}
	}
	if collections[services.SyncCollectionTasks] {
		if resp.Collections[services.SyncCollectionTasks], err = syncRowsFromQuery(tx.Query(
			`SELECT `+syncTaskColumns+` FROM tasks WHERE user_id = $1 AND is_deleted = FALSE`, userID)); err != nil {
			http.Error(w, "unable to read tasks", http.StatusInternalServerError)
			return
		}
	}
	if collections[services.SyncCollectionTags] {
		if resp.Collections[services.SyncCollectionTags], err = syncRowsFromQuery(tx.Query(
			`SELECT `+syncTagColumns+` FROM tags WHERE user_id = $1 AND is_deleted = FALSE`, userID)); err != nil {
			http.Error(w, "unable to read tags", http.StatusInternalServerError)
			return
		}
	}
	writeJSON(w, http.StatusOK, resp)
}

// ChangesRoute serves the incremental feed since a cursor, ordered by
// sync_log.id. Each entry carries the CURRENT row state (attached by
// row_uuid); the op comes from the log entry so deletes tombstone cleanly.
func (s *Handler) ChangesRoute(w http.ResponseWriter, r *http.Request) {
	userID := r.Context().Value("current_user").(int)
	since, err := strconv.ParseInt(r.URL.Query().Get("since"), 10, 64)
	if err != nil || since < 0 {
		http.Error(w, "invalid or missing since cursor", http.StatusBadRequest)
		return
	}
	collections := parseCollectionsParam(r)

	// Retention boundary + feed are read in ONE read transaction so a prune
	// committed between two statements can't let a stale client advance past
	// the pruned range without ever seeing reset (TOCTOU, review P2-2).
	tx, err := s.BeginReadTx()
	if err != nil {
		http.Error(w, "unable to start read transaction", http.StatusInternalServerError)
		return
	}
	if s.ShouldCommitTx() {
		defer tx.Rollback() // read-only in production; test tx owned by harness
	}

	// Retention boundary: if the client's cursor is older than the pruned
	// sync_log range, it can no longer catch up incrementally — answer with
	// reset so it re-bootstraps (snapshot). minID is the first remaining row;
	// the client is fine when since >= minID-1 (nothing pruned between).
	var minID sql.NullInt64
	if err := tx.QueryRow(`SELECT MIN(id) FROM sync_log WHERE user_id = $1`, userID).Scan(&minID); err != nil {
		http.Error(w, "unable to read changes", http.StatusInternalServerError)
		return
	}
	if minID.Valid && since < minID.Int64-1 {
		writeJSON(w, http.StatusOK, models.SyncChangesResponse{Cursor: since, Reset: true})
		return
	}

	// Build the WHERE clause. When all three collections are requested (the
	// default) no IN filter is needed; otherwise constrain to the requested set.
	query := `SELECT id, collection, row_uuid, op, version FROM sync_log WHERE user_id = $1 AND id > $2`
	args := []any{userID, since}
	if len(collections) < 3 {
		keys := make([]string, 0, 3)
		for c := range collections {
			keys = append(keys, c)
		}
		query += ` AND collection IN (`
		for i, c := range keys {
			if i > 0 {
				query += `, `
			}
			query += `$` + strconv.Itoa(len(args)+1)
			args = append(args, c)
		}
		query += `)`
	}
	query += ` ORDER BY id LIMIT $` + strconv.Itoa(len(args)+1)
	args = append(args, syncFeedPageSize+1) // +1 to detect has_more

	rows, err := tx.Query(query, args...)
	if err != nil {
		http.Error(w, "unable to read changes", http.StatusInternalServerError)
		return
	}
	defer rows.Close()

	type feedEntry struct {
		ID         int64
		Collection string
		RowUUID    string
		Op         string
		Version    int
	}
	var entries []feedEntry
	for rows.Next() {
		var e feedEntry
		if err := rows.Scan(&e.ID, &e.Collection, &e.RowUUID, &e.Op, &e.Version); err != nil {
			http.Error(w, "unable to scan changes", http.StatusInternalServerError)
			return
		}
		entries = append(entries, e)
	}
	hasMore := len(entries) > syncFeedPageSize
	if hasMore {
		entries = entries[:syncFeedPageSize]
	}

	// Attach current row state per collection (one batched query per
	// collection). A row_uuid that no longer resolves (e.g. a tag merged onto
	// another row) yields a tombstone with no data.
	dataByUUID := map[string][]byte{}
	byCollection := map[string][]string{}
	for _, e := range entries {
		byCollection[e.Collection] = append(byCollection[e.Collection], e.RowUUID)
	}
	for collection, uuids := range byCollection {
		var cols string
		switch collection {
		case services.SyncCollectionCards:
			cols = syncCardColumns
		case services.SyncCollectionTasks:
			cols = syncTaskColumns
		case services.SyncCollectionTags:
			cols = syncTagColumns
		default:
			continue
		}
		placeholders := make([]string, len(uuids))
		a := []any{userID}
		for i, u := range uuids {
			placeholders[i] = "$" + strconv.Itoa(i+2)
			a = append(a, u)
		}
		rrows, err := tx.Query(
			`SELECT `+cols+` FROM `+collection+` WHERE user_id = $1 AND sync_uuid IN (`+strings.Join(placeholders, ", ")+`)`, a...)
		if err != nil {
			continue
		}
		maps, err := services.RowsToJSON(rrows)
		rrows.Close()
		if err != nil {
			continue
		}
		for _, m := range maps {
			uuid, _ := m["sync_uuid"].(string)
			if b, err := json.Marshal(m); err == nil {
				dataByUUID[uuid] = b
			}
		}
	}

	out := make([]models.SyncRow, 0, len(entries))
	cursor := since
	for _, e := range entries {
		if e.ID > cursor {
			cursor = e.ID
		}
		out = append(out, models.SyncRow{
			RowUUID: e.RowUUID, Version: e.Version, Op: e.Op, Collection: e.Collection, Data: dataByUUID[e.RowUUID],
		})
	}
	writeJSON(w, http.StatusOK, models.SyncChangesResponse{
		Cursor: cursor, Rows: out, HasMore: hasMore,
	})
}

// PushRoute applies an outbox batch transactionally.
func (s *Handler) PushRoute(w http.ResponseWriter, r *http.Request) {
	userID := r.Context().Value("current_user").(int)
	r.Body = http.MaxBytesReader(w, r.Body, syncPushMaxBytes)

	var req models.SyncPushRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		http.Error(w, "invalid push payload", http.StatusBadRequest)
		return
	}
	for _, ch := range req.Changes {
		if ch.RowUUID == "" || !validSyncCollection(ch.Collection) {
			http.Error(w, "invalid change: row_uuid and collection are required", http.StatusBadRequest)
			return
		}
		if ch.Op != services.SyncOpUpsert && ch.Op != services.SyncOpDelete {
			http.Error(w, "invalid change: op must be upsert or delete", http.StatusBadRequest)
			return
		}
	}

	// Apply the batch inside ONE transaction, retrying on SQLITE_BUSY. Write
	// transactions BEGIN IMMEDIATE (_txlock=immediate, server.OpenSQLite), so
	// busy surfaces at BeginTx when a concurrent writer (background job
	// runner, another push) holds the write lock; the apply itself reads
	// before it writes. IsSQLiteBusy matches the extended busy codes too — 517
	// SQLITE_BUSY_SNAPSHOT surfaces instead of 5 because modernc enables
	// extended result codes, which was the production "database is locked"
	// failure (the old == SQLITE_BUSY check missed it and the push never
	// retried). A retry is safe — a busy failure rolled the whole batch back,
	// so nothing partial was committed.
	const pushBusyRetries = 6
	var (
		results []models.SyncPushResult
		cursor  int64
		lost    int
		tx      *sql.Tx
	)
	for attempt := 0; ; attempt++ {
		var err error
		if tx, err = s.BeginTx(); err != nil {
			if server.IsSQLiteBusy(err) && attempt < pushBusyRetries {
				log.Printf("sync push begin busy (attempt %d/%d): %v", attempt+1, pushBusyRetries, err)
				time.Sleep(time.Duration(25*(1<<attempt)) * time.Millisecond)
				continue
			}
			log.Printf("sync push begin: %v", err)
			http.Error(w, "unable to start transaction", http.StatusInternalServerError)
			return
		}
		results, cursor, lost, err = services.ApplySyncPush(tx, userID, &req)
		if err == nil {
			break
		}
		tx.Rollback()
		if !server.IsSQLiteBusy(err) || attempt >= pushBusyRetries {
			log.Printf("sync push apply: %v", err)
			// Schema/validation rejections (bead s2l) are client errors:
			// surface the message so the client knows why the batch was
			// refused instead of a generic 500.
			var vErr *services.ValidationError
			if errors.As(err, &vErr) {
				http.Error(w, vErr.Msg, http.StatusBadRequest)
				return
			}
			http.Error(w, "unable to apply push", http.StatusInternalServerError)
			return
		}
		log.Printf("sync push busy (attempt %d/%d): %v", attempt+1, pushBusyRetries, err)
		time.Sleep(time.Duration(25*(1<<attempt)) * time.Millisecond)
	}
	// Retention heartbeat + opportunistic prune, inside the same tx so the
	// reported cursor is never ahead of the applied batch.
	if req.Cursor != nil {
		if err := services.UpsertSyncClient(tx, userID, req.DeviceID, *req.Cursor); err != nil {
			tx.Rollback()
			log.Printf("sync push heartbeat: %v", err)
			http.Error(w, "unable to record sync cursor", http.StatusInternalServerError)
			return
		}
	}
	if _, err := services.PruneSyncLog(tx, userID, time.Now(), services.SyncLogRetention); err != nil {
		log.Printf("sync push prune: %v", err)
	}
	if s.ShouldCommitTx() {
		if err := tx.Commit(); err != nil {
			log.Printf("sync push commit: %v", err)
			http.Error(w, "unable to commit push", http.StatusInternalServerError)
			return
		}
	}
	writeJSON(w, http.StatusOK, models.SyncPushResponse{
		Results: results, Cursor: cursor, LostEdits: lost,
	})
}
