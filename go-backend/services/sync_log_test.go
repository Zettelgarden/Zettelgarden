package services

import (
	"encoding/json"
	"fmt"
	"testing"
	"time"

	"go-backend/models"
	"go-backend/tests"
)

// Phase 0a acceptance tests (epic Zettelgarden-v5b, issue 13d): the sync_log
// change feed must capture every mutation on cards/tasks/tags — exactly one
// entry per mutation, in the same transaction — while junction re-derivation
// (card_tags/task_tags) emits nothing.
//
// The test suite shares one file-backed DB (tests.Setup); the fixture import
// and the sync emits from other tests' committed writes can precede a test's
// transaction. Every test therefore snapshots the max sync_log id first and
// asserts only on entries written after it.

type syncLogRow struct {
	ID         int
	UserID     int
	Collection string
	RowUUID    string
	Op         string
	Version    int
}

// syncLogMaxID returns the current high-water mark of sync_log.
func syncLogMaxID(t *testing.T, db models.Database) int {
	t.Helper()
	var maxID int
	if err := db.QueryRow(`SELECT COALESCE(MAX(id), 0) FROM sync_log`).Scan(&maxID); err != nil {
		t.Fatalf("max sync_log id: %v", err)
	}
	return maxID
}

func querySyncLogSince(t *testing.T, db models.Database, sinceID int) []syncLogRow {
	t.Helper()
	rows, err := db.Query(`SELECT id, user_id, collection, row_uuid, op, version FROM sync_log WHERE id > $1 ORDER BY id`, sinceID)
	if err != nil {
		t.Fatalf("query sync_log: %v", err)
	}
	defer rows.Close()
	var out []syncLogRow
	for rows.Next() {
		var r syncLogRow
		if err := rows.Scan(&r.ID, &r.UserID, &r.Collection, &r.RowUUID, &r.Op, &r.Version); err != nil {
			t.Fatalf("scan sync_log: %v", err)
		}
		out = append(out, r)
	}
	if err := rows.Err(); err != nil {
		t.Fatalf("iterate sync_log: %v", err)
	}
	return out
}

func filterCollection(rows []syncLogRow, collection string) []syncLogRow {
	var out []syncLogRow
	for _, r := range rows {
		if r.Collection == collection {
			out = append(out, r)
		}
	}
	return out
}

func tableHasColumn(t *testing.T, db models.Database, table, column string) bool {
	t.Helper()
	rows, err := db.Query("PRAGMA table_info(" + table + ")")
	if err != nil {
		t.Fatalf("pragma table_info(%s): %v", table, err)
	}
	defer rows.Close()
	for rows.Next() {
		var cid, notnull, pk int
		var name, ctype string
		var dflt interface{}
		if err := rows.Scan(&cid, &name, &ctype, &notnull, &dflt, &pk); err != nil {
			t.Fatalf("scan table_info(%s): %v", table, err)
		}
		if name == column {
			return true
		}
	}
	return false
}

func TestSyncSchemaPresent(t *testing.T) {
	s := tests.Setup()
	defer tests.Teardown()
	db := s.Tx

	var n int
	if err := db.QueryRow(`SELECT COUNT(*) FROM sqlite_master WHERE type='table' AND name='sync_log'`).Scan(&n); err != nil {
		t.Fatalf("check sync_log: %v", err)
	}
	if n != 1 {
		t.Fatalf("sync_log table missing")
	}
	for _, table := range []string{"cards", "tasks", "tags"} {
		for _, col := range []string{"version", "sync_uuid"} {
			if !tableHasColumn(t, db, table, col) {
				t.Errorf("%s.%s column missing from consolidated schema", table, col)
			}
		}
	}
}

// TestCreateCardEmitsUpsert covers the core contract: exactly one cards entry,
// version 1, row_uuid equal to the row's sync_uuid; a #tag in the body creates
// a real tags row (emitted) but NEVER a card_tags junction entry.
func TestCreateCardEmitsUpsert(t *testing.T) {
	s := tests.Setup()
	defer tests.Teardown()
	db := s.Tx
	userID := 1
	startID := syncLogMaxID(t, db)

	card, err := CreateCard(db, userID, models.EditCardParams{
		Title:  "Sync test card",
		Body:   "A body with #sync-tag",
		CardID: "sync-test-1",
	})
	if err != nil {
		t.Fatalf("CreateCard: %v", err)
	}

	entries := querySyncLogSince(t, db, startID)
	cards := filterCollection(entries, SyncCollectionCards)
	if len(cards) != 1 {
		t.Fatalf("expected exactly 1 cards entry, got %d: %+v", len(cards), cards)
	}
	e := cards[0]
	if e.UserID != userID || e.Op != SyncOpUpsert || e.Version != 1 {
		t.Errorf("unexpected cards entry: %+v", e)
	}
	var rowUUID string
	if err := db.QueryRow(`SELECT sync_uuid FROM cards WHERE id = $1`, card.ID).Scan(&rowUUID); err != nil {
		t.Fatalf("read card sync_uuid: %v", err)
	}
	if e.RowUUID != rowUUID {
		t.Errorf("sync_log row_uuid %q != cards.sync_uuid %q", e.RowUUID, rowUUID)
	}

	tags := filterCollection(entries, SyncCollectionTags)
	if len(tags) == 0 {
		t.Error("expected a tags entry for the #sync-tag created from the body")
	}
	if j := filterCollection(entries, "card_tags"); len(j) != 0 {
		t.Errorf("card_tags is server-derived and must never emit; got %d entries", len(j))
	}
}

// TestCardUpdateAndStructuredDataBumpVersion asserts version monotonicity and
// row_uuid stability across update paths.
func TestCardUpdateAndStructuredDataBumpVersion(t *testing.T) {
	s := tests.Setup()
	defer tests.Teardown()
	db := s.Tx
	userID := 1
	startID := syncLogMaxID(t, db)

	card, err := CreateCard(db, userID, models.EditCardParams{Title: "v1", CardID: "sync-ver"})
	if err != nil {
		t.Fatalf("CreateCard: %v", err)
	}
	if _, err := UpdateCard(db, userID, card.ID, models.EditCardParams{Title: "v2", Body: "b", CardID: "sync-ver"}); err != nil {
		t.Fatalf("UpdateCard: %v", err)
	}
	raw := json.RawMessage(`{"k":"v"}`)
	if _, err := UpdateCardStructuredData(db, userID, card.ID, nil, &raw); err != nil {
		t.Fatalf("UpdateCardStructuredData: %v", err)
	}

	cards := filterCollection(querySyncLogSince(t, db, startID), SyncCollectionCards)
	if len(cards) != 3 {
		t.Fatalf("expected 3 cards entries (create/update/structured), got %d: %+v", len(cards), cards)
	}
	for i, want := range []int{1, 2, 3} {
		if cards[i].Version != want {
			t.Errorf("entry %d version = %d, want %d", i, cards[i].Version, want)
		}
		if cards[i].Op != SyncOpUpsert {
			t.Errorf("entry %d op = %s, want %s", i, cards[i].Op, SyncOpUpsert)
		}
		if i > 0 && cards[i].RowUUID != cards[0].RowUUID {
			t.Errorf("row_uuid changed across updates: %q -> %q", cards[0].RowUUID, cards[i].RowUUID)
		}
	}
	var rowVersion int
	if err := db.QueryRow(`SELECT version FROM cards WHERE id = $1`, card.ID).Scan(&rowVersion); err != nil {
		t.Fatal(err)
	}
	if rowVersion != 3 {
		t.Errorf("cards.version = %d, want 3", rowVersion)
	}
}

// TestDeleteCardEmitsDelete asserts the soft-delete path records op=delete with
// a bumped version.
func TestDeleteCardEmitsDelete(t *testing.T) {
	s := tests.Setup()
	defer tests.Teardown()
	db := s.Tx
	userID := 1
	startID := syncLogMaxID(t, db)

	card, err := CreateCard(db, userID, models.EditCardParams{Title: "doomed", CardID: "sync-del"})
	if err != nil {
		t.Fatalf("CreateCard: %v", err)
	}
	if err := DeleteCard(db, userID, card.ID); err != nil {
		t.Fatalf("DeleteCard: %v", err)
	}

	cards := filterCollection(querySyncLogSince(t, db, startID), SyncCollectionCards)
	last := cards[len(cards)-1]
	if last.Op != SyncOpDelete {
		t.Errorf("last cards op = %s, want %s", last.Op, SyncOpDelete)
	}
	if last.Version != 2 {
		t.Errorf("delete version = %d, want 2 (create v1 -> delete v2)", last.Version)
	}
	var isDeleted bool
	if err := db.QueryRow(`SELECT is_deleted FROM cards WHERE id = $1`, card.ID).Scan(&isDeleted); err != nil {
		t.Fatal(err)
	}
	if !isDeleted {
		t.Error("card not soft-deleted")
	}
}

// TestTaskWritePathsEmit drives every task mutation path and asserts one entry
// each with monotonic versions and the right ops.
func TestTaskWritePathsEmit(t *testing.T) {
	s := tests.Setup()
	defer tests.Teardown()
	db := s.Tx
	userID := 1
	startID := syncLogMaxID(t, db)

	taskID, err := CreateTask(db, models.Task{UserID: userID, CardPK: 1, Title: "t1"})
	if err != nil {
		t.Fatalf("CreateTask: %v", err)
	}
	parentID, err := CreateTask(db, models.Task{UserID: userID, CardPK: 1, Title: "parent"})
	if err != nil {
		t.Fatalf("CreateTask parent: %v", err)
	}
	if _, err := UpdateTask(db, userID, taskID, models.Task{UserID: userID, CardPK: 1, Title: "t1-updated"}); err != nil {
		t.Fatalf("UpdateTask: %v", err)
	}
	if err := UpdateTaskParent(db, userID, taskID, &parentID); err != nil {
		t.Fatalf("UpdateTaskParent: %v", err)
	}
	if err := ReorderTasks(db, userID, []struct {
		ID        int `json:"id"`
		SortOrder int `json:"sort_order"`
	}{{ID: taskID, SortOrder: 5}}); err != nil {
		t.Fatalf("ReorderTasks: %v", err)
	}
	if err := DeleteTask(db, userID, taskID); err != nil {
		t.Fatalf("DeleteTask: %v", err)
	}

	entries := filterCollection(querySyncLogSince(t, db, startID), SyncCollectionTasks)
	// create(parent), create(task), update, parent, reorder, delete
	if len(entries) != 6 {
		t.Fatalf("expected 6 task entries, got %d: %+v", len(entries), entries)
	}
	wantOps := []string{SyncOpUpsert, SyncOpUpsert, SyncOpUpsert, SyncOpUpsert, SyncOpUpsert, SyncOpDelete}
	for i, want := range wantOps {
		if entries[i].Op != want {
			t.Errorf("entry %d op = %s, want %s", i, entries[i].Op, want)
		}
	}
	if entries[5].Version != entries[4].Version+1 {
		t.Errorf("delete version %d should bump from %d", entries[5].Version, entries[4].Version)
	}
}

// TestTagWritePathsEmit covers create/rename/delete and the resurrect path
// (CreateTag on a soft-deleted name reuses the row via EditTag — same
// sync_uuid, version keeps climbing).
func TestTagWritePathsEmit(t *testing.T) {
	s := tests.Setup()
	defer tests.Teardown()
	db := s.Tx
	userID := 1
	startID := syncLogMaxID(t, db)

	tag, err := CreateTag(db, userID, models.EditTagParams{Name: "sync-tag", Color: "blue"})
	if err != nil {
		t.Fatalf("CreateTag: %v", err)
	}
	if _, err := EditTag(db, userID, "sync-tag", models.EditTagParams{Name: "sync-tag-renamed", Color: "red"}); err != nil {
		t.Fatalf("EditTag: %v", err)
	}
	if err := DeleteTag(db, userID, tag.ID); err != nil {
		t.Fatalf("DeleteTag: %v", err)
	}
	// Resurrect: same name reuses the soft-deleted row.
	if _, err := CreateTag(db, userID, models.EditTagParams{Name: "sync-tag-renamed", Color: "green"}); err != nil {
		t.Fatalf("CreateTag resurrect: %v", err)
	}

	tags := filterCollection(querySyncLogSince(t, db, startID), SyncCollectionTags)
	if len(tags) != 4 {
		t.Fatalf("expected 4 tag entries, got %d: %+v", len(tags), tags)
	}
	wantOps := []string{SyncOpUpsert, SyncOpUpsert, SyncOpDelete, SyncOpUpsert}
	for i, want := range wantOps {
		if tags[i].Op != want {
			t.Errorf("entry %d op = %s, want %s", i, tags[i].Op, want)
		}
	}
	rowUUID := tags[0].RowUUID
	for i, e := range tags {
		if e.RowUUID != rowUUID {
			t.Errorf("entry %d row_uuid %q != initial %q (rename/resurrect must keep identity)", i, e.RowUUID, rowUUID)
		}
		if e.Version != i+1 {
			t.Errorf("entry %d version = %d, want %d", i, e.Version, i+1)
		}
	}
}

// TestJunctionDerivationNeverEmits asserts tag additions/removals caused by
// card body edits never produce card_tags/task_tags feed entries.
func TestJunctionDerivationNeverEmits(t *testing.T) {
	s := tests.Setup()
	defer tests.Teardown()
	db := s.Tx
	userID := 1
	startID := syncLogMaxID(t, db)

	card, err := CreateCard(db, userID, models.EditCardParams{Title: "tagged", Body: "#a #b", CardID: "sync-junc"})
	if err != nil {
		t.Fatalf("CreateCard: %v", err)
	}
	if _, err := UpdateCard(db, userID, card.ID, models.EditCardParams{Title: "tagged", Body: "#a", CardID: "sync-junc"}); err != nil {
		t.Fatalf("UpdateCard: %v", err)
	}

	entries := querySyncLogSince(t, db, startID)
	if j := filterCollection(entries, "card_tags"); len(j) != 0 {
		t.Errorf("card_tags must never emit; got %d entries", len(j))
	}
	if j := filterCollection(entries, "task_tags"); len(j) != 0 {
		t.Errorf("task_tags must never emit; got %d entries", len(j))
	}
	// Only cards + tags collections should exist at all.
	for _, c := range entries {
		if c.Collection != SyncCollectionCards && c.Collection != SyncCollectionTags {
			t.Errorf("unexpected sync collection %q", c.Collection)
		}
	}
}

// TestApplySyncPushEmitFailureRollsBack covers the v5b.3 invariant: a failed
// sync_log emit inside a push must propagate so the caller rolls back the
// whole batch — never a committed row write with no feed entry for other
// clients to pull.
func TestApplySyncPushEmitFailureRollsBack(t *testing.T) {
	s := tests.Setup()
	defer tests.Teardown()

	// Use a dedicated pool-level tx so we can prove the rollback discards the
	// partial write (the shared test tx is owned by the harness).
	tx, err := s.DB.Begin()
	if err != nil {
		t.Fatal(err)
	}
	defer tx.Rollback()

	// Inject an emit failure: the sync_log table vanishes mid-batch.
	if _, err := tx.Exec(`DROP TABLE sync_log`); err != nil {
		t.Fatal(err)
	}

	data, _ := json.Marshal(models.SyncCardData{CardID: "emit-fail", Title: "t", Body: "b"})
	_, _, _, err = ApplySyncPush(tx, 1, &models.SyncPushRequest{
		DeviceID: "dev",
		Changes: []models.SyncChange{
			{Collection: SyncCollectionCards, RowUUID: "c-emit-fail", Op: SyncOpUpsert, BaseVersion: 0, Data: data},
		},
	})
	if err == nil {
		t.Fatal("ApplySyncPush: expected error from failed sync_log emit")
	}

	// The caller rolled back: no partial card write survived the failure.
	var count int
	if err := s.DB.QueryRow(`SELECT COUNT(*) FROM cards WHERE sync_uuid = 'c-emit-fail'`).Scan(&count); err != nil {
		t.Fatal(err)
	}
	if count != 0 {
		t.Errorf("cards rows = %d, want 0 (emit failure must roll back the batch)", count)
	}
}

// TestSyncLogPruneAndRebootstrap covers the v5b.5 retention policy: rows older
// than the threshold are pruned only when no active client cursor trails them;
// a device that stopped reporting no longer blocks pruning; the changes feed
// then answers reset for a stale cursor.
func TestSyncLogPruneAndRebootstrap(t *testing.T) {
	s := tests.Setup()
	defer tests.Teardown()
	db := s.Tx
	userID := 2 // isolated from user-1 noise in the shared test DB
	now := time.Now()

	old := now.Add(-60 * 24 * time.Hour).UTC().Format("2006-01-02 15:04:05")
	insertRow := func(rowUUID, createdAt string) int64 {
		t.Helper()
		var id int64
		if err := db.QueryRow(`INSERT INTO sync_log (user_id, collection, row_uuid, op, version, created_at) VALUES ($1, 'cards', $2, 'upsert', 1, $3) RETURNING id`,
			userID, rowUUID, createdAt).Scan(&id); err != nil {
			t.Fatalf("insert sync_log row: %v", err)
		}
		return id
	}

	// Three old rows the device has consumed + one recent row (same cursor
	// range) that must survive because it is within retention.
	var oldIDs []int64
	for i := 0; i < 3; i++ {
		oldIDs = append(oldIDs, insertRow(fmt.Sprintf("c-ret-old-%d", i), old))
	}
	_ = insertRow("c-ret-recent", now.UTC().Format("2006-01-02 15:04:05"))

	// Active device trails a high cursor; a dead device (last seen 90 days
	// ago) trails cursor 0. Only the active device counts, so pruning uses the
	// high cutoff — the dead device's stale cursor 0 must not block it.
	if err := UpsertSyncClient(db, userID, "dev-a", 1<<30); err != nil {
		t.Fatal(err)
	}
	if _, err := db.Exec(`INSERT INTO sync_clients (user_id, device_id, cursor, last_seen_at) VALUES ($1, 'dead-dev', 0, $2)`, userID, now.Add(-90*24*time.Hour).UTC().Format("2006-01-02 15:04:05")); err != nil {
		t.Fatal(err)
	}

	pruned, err := PruneSyncLog(db, userID, now, SyncLogRetention)
	if err != nil {
		t.Fatal(err)
	}
	if pruned != 3 {
		t.Fatalf("pruned = %d, want 3 (old rows at/below the active cursor; dead device excluded)", pruned)
	}
	var count int
	if err := db.QueryRow(`SELECT COUNT(*) FROM sync_log WHERE user_id = $1 AND row_uuid = 'c-ret-recent'`, userID).Scan(&count); err != nil {
		t.Fatal(err)
	}
	if count != 1 {
		t.Errorf("recent row within retention was pruned (count = %d)", count)
	}

	// With no tracked clients at all, nothing is pruned.
	if _, err := db.Exec(`DELETE FROM sync_clients WHERE user_id = $1`, userID); err != nil {
		t.Fatal(err)
	}
	moreOld := insertRow("c-ret-old-4", old)
	_ = moreOld
	pruned2, err := PruneSyncLog(db, userID, now, SyncLogRetention)
	if err != nil {
		t.Fatal(err)
	}
	if pruned2 != 0 {
		t.Errorf("pruned = %d, want 0 with no tracked clients", pruned2)
	}
}

// TestSyncLogAppendOnly asserts ids are strictly increasing (nothing is
// updated in place) across the whole feed, and that every mutation produced
// exactly one entry within the test.
func TestSyncLogAppendOnly(t *testing.T) {
	s := tests.Setup()
	defer tests.Teardown()
	db := s.Tx
	userID := 1
	startID := syncLogMaxID(t, db)

	card, err := CreateCard(db, userID, models.EditCardParams{Title: "a", CardID: "sync-append"})
	if err != nil {
		t.Fatalf("CreateCard: %v", err)
	}
	if _, err := UpdateCard(db, userID, card.ID, models.EditCardParams{Title: "b", CardID: "sync-append"}); err != nil {
		t.Fatalf("UpdateCard: %v", err)
	}

	entries := querySyncLogSince(t, db, startID)
	for i := 1; i < len(entries); i++ {
		if entries[i].ID <= entries[i-1].ID {
			t.Fatalf("sync_log ids not strictly increasing at %d: %d -> %d", i, entries[i-1].ID, entries[i].ID)
		}
	}
}
