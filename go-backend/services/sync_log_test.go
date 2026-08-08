package services

import (
	"encoding/json"
	"testing"

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
