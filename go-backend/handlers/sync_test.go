package handlers

import (
	"bytes"
	"encoding/json"
	"fmt"
	"net/http"
	"net/http/httptest"
	"strconv"
	"strings"
	"testing"
	"time"

	"go-backend/models"
	"go-backend/services"
	"go-backend/tests"

	"github.com/gorilla/mux"
)

// Sync API integration tests (epic Zettelgarden-v5b, Phase 0b — issue tsv).
// These exercise the exit-criteria flow end-to-end: snapshot bootstrap,
// incremental feed, batch push with offline-created linked rows, tag
// name-merge, idempotent retry, optimistic-concurrency conflicts, delete
// propagation, and server-authoritative parentage.

func newSyncRouter(s *Handler) *mux.Router {
	r := mux.NewRouter()
	r.HandleFunc("/api/sync/snapshot", s.JwtMiddleware(s.SnapshotRoute)).Methods("GET")
	r.HandleFunc("/api/sync/changes", s.JwtMiddleware(s.ChangesRoute)).Methods("GET")
	r.HandleFunc("/api/sync/push", s.JwtMiddleware(s.PushRoute)).Methods("POST")
	return r
}

func syncRequest(t *testing.T, s *Handler, router *mux.Router, method, path string, body any) *httptest.ResponseRecorder {
	return syncRequestAs(t, s, router, 1, method, path, body)
}

// syncRequestAs is syncRequest for an explicit user id (multi-user tests).
func syncRequestAs(t *testing.T, s *Handler, router *mux.Router, userID int, method, path string, body any) *httptest.ResponseRecorder {
	t.Helper()
	var buf bytes.Buffer
	if body != nil {
		if err := json.NewEncoder(&buf).Encode(body); err != nil {
			t.Fatalf("encode body: %v", err)
		}
	}
	req, err := http.NewRequest(method, path, &buf)
	if err != nil {
		t.Fatalf("new request: %v", err)
	}
	token, _ := tests.GenerateTestJWT(userID)
	req.Header.Set("Authorization", "Bearer "+token)
	if body != nil {
		req.Header.Set("Content-Type", "application/json")
	}
	rr := httptest.NewRecorder()
	router.ServeHTTP(rr, req)
	return rr
}

func decodeSyncResp[T any](t *testing.T, rr *httptest.ResponseRecorder) T {
	t.Helper()
	if rr.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d: %s", rr.Code, rr.Body.String())
	}
	var out T
	if err := json.NewDecoder(rr.Body).Decode(&out); err != nil {
		t.Fatalf("decode response: %v (body=%s)", err, rr.Body.String())
	}
	return out
}

func pushChanges(t *testing.T, s *Handler, router *mux.Router, changes []models.SyncChange) models.SyncPushResponse {
	t.Helper()
	rr := syncRequest(t, s, router, "POST", "/api/sync/push", models.SyncPushRequest{Changes: changes, DeviceID: "test-device"})
	return decodeSyncResp[models.SyncPushResponse](t, rr)
}

func syncCardChange(rowUUID, cardID, title string, baseVersion int) models.SyncChange {
	data, _ := json.Marshal(models.SyncCardData{CardID: cardID, Title: title, Body: title})
	return models.SyncChange{Collection: services.SyncCollectionCards, RowUUID: rowUUID, Op: services.SyncOpUpsert, BaseVersion: baseVersion, Data: data}
}

func syncTagChange(rowUUID, name, color string, baseVersion int) models.SyncChange {
	data, _ := json.Marshal(models.SyncTagData{Name: name, Color: color})
	return models.SyncChange{Collection: services.SyncCollectionTags, RowUUID: rowUUID, Op: services.SyncOpUpsert, BaseVersion: baseVersion, Data: data}
}

func TestSyncSnapshotBootstrap(t *testing.T) {
	s := NewHandler()
	defer tests.Teardown()
	router := newSyncRouter(s)

	rr := syncRequest(t, s, router, "GET", "/api/sync/snapshot", nil)
	resp := decodeSyncResp[models.SyncSnapshotResponse](t, rr)

	if resp.Cursor < 0 {
		t.Fatalf("cursor must be >= 0, got %d", resp.Cursor)
	}
	for _, c := range []string{"cards", "tasks", "tags"} {
		if _, ok := resp.Collections[c]; !ok {
			t.Errorf("snapshot missing collection %q", c)
		}
	}
	// Every row must carry a non-empty sync_uuid and version >= 1.
	for coll, rows := range resp.Collections {
		for _, row := range rows {
			if row.RowUUID == "" {
				t.Errorf("%s row missing row_uuid", coll)
			}
			if row.Version < 1 {
				t.Errorf("%s row %s version %d < 1", coll, row.RowUUID, row.Version)
			}
			if len(row.Data) == 0 {
				t.Errorf("%s row %s missing data", coll, row.RowUUID)
			}
		}
	}
}

func TestSyncPushCreateThenFeed(t *testing.T) {
	s := NewHandler()
	defer tests.Teardown()
	router := newSyncRouter(s)

	// Bootstrap to get a cursor, then mutate.
	boot := decodeSyncResp[models.SyncSnapshotResponse](t, syncRequest(t, s, router, "GET", "/api/sync/snapshot", nil))

	resp := pushChanges(t, s, router, []models.SyncChange{syncCardChange("c-push-1", "sync-push-card", "Pushed card", 0)})
	if resp.Results[0].Status != services.SyncStatusApplied {
		t.Fatalf("expected applied, got %+v", resp.Results[0])
	}
	if resp.Results[0].ServerID == nil || *resp.Results[0].ServerID == 0 {
		t.Fatalf("expected server_id, got %+v", resp.Results[0])
	}
	if resp.Results[0].ServerVersion != 1 {
		t.Fatalf("expected version 1, got %d", resp.Results[0].ServerVersion)
	}

	// Feed since the snapshot cursor must contain the pushed card.
	feed := decodeSyncResp[models.SyncChangesResponse](t, syncRequest(t, s, router, "GET",
		"/api/sync/changes?since="+strconv.FormatInt(boot.Cursor, 10), nil))

	found := false
	for _, row := range feed.Rows {
		if row.RowUUID == "c-push-1" {
			found = true
			if row.Op != services.SyncOpUpsert {
				t.Errorf("feed op = %s, want upsert", row.Op)
			}
			if row.Version != 1 {
				t.Errorf("feed version = %d, want 1", row.Version)
			}
			var data map[string]any
			if err := json.Unmarshal(row.Data, &data); err != nil {
				t.Fatalf("feed data: %v", err)
			}
			if data["card_id"] != "sync-push-card" {
				t.Errorf("feed card_id = %v", data["card_id"])
			}
		}
	}
	if !found {
		t.Fatalf("pushed card %q missing from feed; rows=%d", "c-push-1", len(feed.Rows))
	}
	if feed.Cursor < boot.Cursor {
		t.Errorf("feed cursor %d regressed below snapshot cursor %d", feed.Cursor, boot.Cursor)
	}
}

// TestSyncPushOfflineLinkedRows is the core exit-criteria case: an
// offline-created card and an offline-created task linked to it via
// card_pk_uuid in the same batch must both land, with the task's card_pk
// resolved to the card's server id.
func TestSyncPushOfflineLinkedRows(t *testing.T) {
	s := NewHandler()
	defer tests.Teardown()
	router := newSyncRouter(s)

	cardData, _ := json.Marshal(models.SyncCardData{CardID: "sync-offline-parent", Title: "Offline parent", Body: "p"})
	taskData, _ := json.Marshal(models.SyncTaskData{
		Title:      "Offline child task",
		CardPKUUID: "c-offline-1",
		Status:     "todo",
	})
	resp := pushChanges(t, s, router, []models.SyncChange{
		{Collection: services.SyncCollectionCards, RowUUID: "c-offline-1", Op: services.SyncOpUpsert, BaseVersion: 0, Data: cardData},
		{Collection: services.SyncCollectionTasks, RowUUID: "t-offline-1", Op: services.SyncOpUpsert, BaseVersion: 0, Data: taskData},
	})

	if resp.Results[0].Status != services.SyncStatusApplied {
		t.Fatalf("card result: %+v", resp.Results[0])
	}
	if resp.Results[1].Status != services.SyncStatusApplied {
		t.Fatalf("task result: %+v", resp.Results[1])
	}

	var cardPK int
	if err := s.GetDB().QueryRow(`SELECT card_pk FROM tasks WHERE sync_uuid = 't-offline-1'`).Scan(&cardPK); err != nil {
		t.Fatalf("read task card_pk: %v", err)
	}
	cardID := *resp.Results[0].ServerID
	if cardPK != cardID {
		t.Errorf("task.card_pk = %d, want resolved to card id %d", cardPK, cardID)
	}
}

// TestSyncPushTagNameMerge covers the second-pass review's critical case:
// the same-named tag pushed from two devices must merge into ONE server row,
// with the second device's uuid remapped to the first.
func TestSyncPushTagNameMerge(t *testing.T) {
	s := NewHandler()
	defer tests.Teardown()
	router := newSyncRouter(s)

	name := "sync-merge-tag-xyz"
	devA, _ := json.Marshal(models.SyncTagData{Name: name, Color: "black"})
	devB, _ := json.Marshal(models.SyncTagData{Name: name, Color: "red"})

	respA := pushChanges(t, s, router, []models.SyncChange{
		{Collection: services.SyncCollectionTags, RowUUID: "tag-a-1", Op: services.SyncOpUpsert, BaseVersion: 0, Data: devA},
	})
	if respA.Results[0].Status != services.SyncStatusApplied {
		t.Fatalf("device A tag: %+v", respA.Results[0])
	}

	respB := pushChanges(t, s, router, []models.SyncChange{
		{Collection: services.SyncCollectionTags, RowUUID: "tag-b-1", Op: services.SyncOpUpsert, BaseVersion: 0, Data: devB},
	})
	if respB.Results[0].Status != services.SyncStatusMerged {
		t.Fatalf("device B tag: expected merged, got %+v", respB.Results[0])
	}
	if respB.Results[0].MappedToRowUUID != "tag-a-1" {
		t.Errorf("device B mapped to %q, want tag-a-1", respB.Results[0].MappedToRowUUID)
	}

	// Exactly one server tag with that name.
	var count int
	if err := s.GetDB().QueryRow(`SELECT COUNT(*) FROM tags WHERE user_id = 1 AND name = $1`, name).Scan(&count); err != nil {
		t.Fatalf("count tags: %v", err)
	}
	if count != 1 {
		t.Errorf("expected 1 merged tag, got %d", count)
	}
}

// TestSyncPushTagRenameTombstone covers the 8g0 policy: a rename keeps a
// soft-deleted tombstone for the renamed-away name (fresh uuid, never
// emitted), so a later offline create of the old name RESURRECTS it instead
// of making a fresh row — stable identity across the rename+recreate cycle.
func TestSyncPushTagRenameTombstone(t *testing.T) {
	s := NewHandler()
	defer tests.Teardown()
	router := newSyncRouter(s)

	// A creates 'work'.
	create := syncTagChange("tag-a", "work", "blue", 0)
	if r := pushChanges(t, s, router, []models.SyncChange{create}); r.Results[0].Status != services.SyncStatusApplied {
		t.Fatalf("create work: %+v", r.Results[0])
	}

	// A renames 'work' -> 'tasks' (same row, uuid tag-a).
	rename := syncTagChange("tag-a", "tasks", "blue", 1)
	if r := pushChanges(t, s, router, []models.SyncChange{rename}); r.Results[0].Status != services.SyncStatusApplied {
		t.Fatalf("rename to tasks: %+v", r.Results[0])
	}

	// A tombstone row for the renamed-away name exists: soft-deleted, FRESH
	// uuid (tag-a now belongs to 'tasks').
	var tombUUID string
	if err := s.GetDB().QueryRow(`SELECT sync_uuid FROM tags WHERE user_id = 1 AND name = 'work' AND is_deleted = TRUE`).Scan(&tombUUID); err != nil {
		t.Fatalf("tombstone for 'work' missing after rename: %v", err)
	}
	if tombUUID == "tag-a" {
		t.Fatalf("tombstone reused the renamed row's uuid, want a fresh one")
	}

	// B creates 'work' offline: the create RESURRECTS the tombstone (merged,
	// mapped to the tombstone uuid) instead of making a fresh row.
	bCreate := syncTagChange("tag-b-fresh", "work", "red", 0)
	resp := pushChanges(t, s, router, []models.SyncChange{bCreate})
	if resp.Results[0].Status != services.SyncStatusMerged {
		t.Fatalf("B's create of the renamed-away name: expected merged, got %+v", resp.Results[0])
	}
	if resp.Results[0].MappedToRowUUID != tombUUID {
		t.Errorf("B mapped to %q, want the tombstone uuid %q", resp.Results[0].MappedToRowUUID, tombUUID)
	}

	// Exactly one live 'work' (the tombstone row, resurrected) and one live
	// 'tasks' (tag-a); B's fresh uuid never landed server-side.
	var workCount int
	if err := s.GetDB().QueryRow(`SELECT COUNT(*) FROM tags WHERE user_id = 1 AND name = 'work' AND is_deleted = FALSE`).Scan(&workCount); err != nil {
		t.Fatal(err)
	}
	if workCount != 1 {
		t.Errorf("live 'work' count = %d, want 1", workCount)
	}
	var tasksCount int
	if err := s.GetDB().QueryRow(`SELECT COUNT(*) FROM tags WHERE user_id = 1 AND name = 'tasks' AND is_deleted = FALSE`).Scan(&tasksCount); err != nil {
		t.Fatal(err)
	}
	if tasksCount != 1 {
		t.Errorf("live 'tasks' count = %d, want 1", tasksCount)
	}
	var freshCount int
	if err := s.GetDB().QueryRow(`SELECT COUNT(*) FROM tags WHERE user_id = 1 AND sync_uuid = 'tag-b-fresh'`).Scan(&freshCount); err != nil {
		t.Fatal(err)
	}
	if freshCount != 0 {
		t.Errorf("B's fresh uuid survived server-side (%d rows)", freshCount)
	}
}

func TestSyncPushIdempotentRetry(t *testing.T) {
	s := NewHandler()
	defer tests.Teardown()
	router := newSyncRouter(s)

	ch := syncCardChange("c-retry-1", "sync-retry", "Retry me", 0)
	resp1 := pushChanges(t, s, router, []models.SyncChange{ch})
	if resp1.Results[0].Status != services.SyncStatusApplied {
		t.Fatalf("first push: %+v", resp1.Results[0])
	}
	resp2 := pushChanges(t, s, router, []models.SyncChange{ch})
	if resp2.Results[0].Status != services.SyncStatusApplied {
		t.Fatalf("retry must be idempotent-applied, got %+v", resp2.Results[0])
	}

	var count int
	if err := s.GetDB().QueryRow(`SELECT COUNT(*) FROM cards WHERE sync_uuid = 'c-retry-1'`).Scan(&count); err != nil {
		t.Fatal(err)
	}
	if count != 1 {
		t.Fatalf("idempotent retry created %d rows, want 1", count)
	}
	if resp2.LostEdits != 0 {
		t.Errorf("idempotent retry counted %d lost edits", resp2.LostEdits)
	}
}

// TestSyncPushUpdateRetryIdempotent covers the dropped-response case: the
// server applies an update, the response is lost, and the client re-pushes the
// SAME outbox entry (base_version unchanged). The retry must be reported
// applied with no lost edit, and must not emit a duplicate feed entry.
func TestSyncPushUpdateRetryIdempotent(t *testing.T) {
	s := NewHandler()
	defer tests.Teardown()
	router := newSyncRouter(s)

	boot := decodeSyncResp[models.SyncSnapshotResponse](t, syncRequest(t, s, router, "GET", "/api/sync/snapshot", nil))

	create := syncCardChange("c-ur-1", "sync-update-retry", "v1 title", 0)
	resp1 := pushChanges(t, s, router, []models.SyncChange{create})
	if resp1.Results[0].Status != services.SyncStatusApplied {
		t.Fatalf("create: %+v", resp1.Results[0])
	}

	// Update applies server-side (v2) but the response never reaches the
	// client, so it retries the identical outbox entry.
	update := syncCardChange("c-ur-1", "sync-update-retry", "v2 title", 1)
	resp2 := pushChanges(t, s, router, []models.SyncChange{update})
	if resp2.Results[0].Status != services.SyncStatusApplied {
		t.Fatalf("first update: %+v", resp2.Results[0])
	}

	retry := pushChanges(t, s, router, []models.SyncChange{update})
	if retry.Results[0].Status != services.SyncStatusApplied {
		t.Fatalf("update retry: expected applied (idempotent), got %+v", retry.Results[0])
	}
	if retry.LostEdits != 0 {
		t.Errorf("update retry counted %d lost edits, want 0", retry.LostEdits)
	}
	if retry.Results[0].ServerVersion != 2 {
		t.Errorf("retry server_version = %d, want 2 (current server version)", retry.Results[0].ServerVersion)
	}

	var title string
	if err := s.GetDB().QueryRow(`SELECT title FROM cards WHERE sync_uuid = 'c-ur-1'`).Scan(&title); err != nil {
		t.Fatal(err)
	}
	if title != "v2 title" {
		t.Errorf("server title = %q, want v2 title", title)
	}
	var count int
	if err := s.GetDB().QueryRow(`SELECT COUNT(*) FROM cards WHERE sync_uuid = 'c-ur-1'`).Scan(&count); err != nil {
		t.Fatal(err)
	}
	if count != 1 {
		t.Errorf("row count = %d, want 1 (no duplicate from retry)", count)
	}

	// Exactly one upsert entry for the update (v2) in the feed: the retry
	// must not emit a second entry.
	feed := decodeSyncResp[models.SyncChangesResponse](t, syncRequest(t, s, router, "GET",
		"/api/sync/changes?since="+strconv.FormatInt(boot.Cursor, 10), nil))
	upserts := 0
	for _, row := range feed.Rows {
		if row.RowUUID == "c-ur-1" && row.Op == services.SyncOpUpsert {
			upserts++
		}
	}
	if upserts != 2 { // create v1 + update v2
		t.Errorf("feed upsert entries for c-ur-1 = %d, want 2 (create + update, no retry duplicate)", upserts)
	}
}

// TestSyncPushDeleteRetryIdempotent is the delete sibling of the dropped-
// response retry: the delete applies, the response is lost, and the re-push of
// the same delete entry must be applied with no lost edit.
func TestSyncPushDeleteRetryIdempotent(t *testing.T) {
	s := NewHandler()
	defer tests.Teardown()
	router := newSyncRouter(s)

	create := syncCardChange("c-dr-1", "sync-delete-retry", "doomed", 0)
	resp1 := pushChanges(t, s, router, []models.SyncChange{create})
	if resp1.Results[0].Status != services.SyncStatusApplied {
		t.Fatalf("create: %+v", resp1.Results[0])
	}

	delData, _ := json.Marshal(models.SyncCardData{})
	del := models.SyncChange{Collection: services.SyncCollectionCards, RowUUID: "c-dr-1", Op: services.SyncOpDelete, BaseVersion: 1, Data: delData}
	resp2 := pushChanges(t, s, router, []models.SyncChange{del})
	if resp2.Results[0].Status != services.SyncStatusApplied {
		t.Fatalf("delete: %+v", resp2.Results[0])
	}

	retry := pushChanges(t, s, router, []models.SyncChange{del})
	if retry.Results[0].Status != services.SyncStatusApplied {
		t.Fatalf("delete retry: expected applied (idempotent), got %+v", retry.Results[0])
	}
	if retry.LostEdits != 0 {
		t.Errorf("delete retry counted %d lost edits, want 0", retry.LostEdits)
	}

	var isDeleted bool
	if err := s.GetDB().QueryRow(`SELECT is_deleted FROM cards WHERE sync_uuid = 'c-dr-1'`).Scan(&isDeleted); err != nil {
		t.Fatal(err)
	}
	if !isDeleted {
		t.Error("card not soft-deleted after retry")
	}
}

// TestSyncPushTaskUpdateRetryIdempotent exercises the task retry matcher,
// which must resolve uuid FK references and compare timestamp fields when
// deciding an idempotent retry from a genuinely stale concurrent edit.
func TestSyncPushTaskUpdateRetryIdempotent(t *testing.T) {
	s := NewHandler()
	defer tests.Teardown()
	router := newSyncRouter(s)

	cardData, _ := json.Marshal(models.SyncCardData{CardID: "sync-task-retry", Title: "Parent", Body: "p"})
	respCard := pushChanges(t, s, router, []models.SyncChange{
		{Collection: services.SyncCollectionCards, RowUUID: "c-tr-1", Op: services.SyncOpUpsert, BaseVersion: 0, Data: cardData},
	})
	if respCard.Results[0].Status != services.SyncStatusApplied {
		t.Fatalf("card: %+v", respCard.Results[0])
	}

	due := time.Date(2026, 9, 1, 12, 30, 0, 0, time.UTC)
	taskData, _ := json.Marshal(models.SyncTaskData{
		Title:        "Retry me",
		Status:       "todo",
		CardPKUUID:   "c-tr-1",
		DueDate:      &due,
		ReminderTime: &due,
		ReminderSent: true,
		Priority:     strPtr("high"),
		Description:  strPtr("note"),
		SortOrder:    intPtr(3),
	})
	taskChange := models.SyncChange{Collection: services.SyncCollectionTasks, RowUUID: "t-tr-1", Op: services.SyncOpUpsert, BaseVersion: 0, Data: taskData}
	resp1 := pushChanges(t, s, router, []models.SyncChange{taskChange})
	if resp1.Results[0].Status != services.SyncStatusApplied {
		t.Fatalf("create task: %+v", resp1.Results[0])
	}

	// Edit the task (v2), then retry the same outbox entry.
	v2Data, _ := json.Marshal(models.SyncTaskData{
		Title:        "Retry me v2",
		Status:       "todo",
		CardPKUUID:   "c-tr-1",
		DueDate:      &due,
		ReminderTime: &due,
		ReminderSent: true,
		Priority:     strPtr("high"),
		Description:  strPtr("note"),
		SortOrder:    intPtr(3),
	})
	update := models.SyncChange{Collection: services.SyncCollectionTasks, RowUUID: "t-tr-1", Op: services.SyncOpUpsert, BaseVersion: 1, Data: v2Data}
	resp2 := pushChanges(t, s, router, []models.SyncChange{update})
	if resp2.Results[0].Status != services.SyncStatusApplied {
		t.Fatalf("update task: %+v", resp2.Results[0])
	}
	retry := pushChanges(t, s, router, []models.SyncChange{update})
	if retry.Results[0].Status != services.SyncStatusApplied {
		t.Fatalf("task update retry: expected applied, got %+v", retry.Results[0])
	}
	if retry.LostEdits != 0 {
		t.Errorf("task update retry counted %d lost edits, want 0", retry.LostEdits)
	}

	var title string
	var cardPK int
	if err := s.GetDB().QueryRow(`SELECT title, card_pk FROM tasks WHERE sync_uuid = 't-tr-1'`).Scan(&title, &cardPK); err != nil {
		t.Fatal(err)
	}
	if title != "Retry me v2" {
		t.Errorf("task title = %q, want Retry me v2", title)
	}
	if cardPK != *respCard.Results[0].ServerID {
		t.Errorf("task.card_pk = %d, want %d", cardPK, *respCard.Results[0].ServerID)
	}
}

// TestSyncPushStaleUpdateVsDelete pins the discriminator between an idempotent
// delete retry and a genuinely stale concurrent edit: a different edit landing
// at version == base+1 must still conflict with a lost edit, even though the
// row is already soft-deleted (so a naive "row deleted => delete retry"
// shortcut would wrongly swallow it).
func TestSyncPushStaleUpdateVsDelete(t *testing.T) {
	s := NewHandler()
	defer tests.Teardown()
	router := newSyncRouter(s)

	create := syncCardChange("c-sd-1", "sync-stale-del", "v1 title", 0)
	resp1 := pushChanges(t, s, router, []models.SyncChange{create})
	if resp1.Results[0].Status != services.SyncStatusApplied {
		t.Fatalf("create: %+v", resp1.Results[0])
	}

	// Device A deletes from v1 -> server v2 (soft-deleted).
	delData, _ := json.Marshal(models.SyncCardData{})
	delA := models.SyncChange{Collection: services.SyncCollectionCards, RowUUID: "c-sd-1", Op: services.SyncOpDelete, BaseVersion: 1, Data: delData}
	resp2 := pushChanges(t, s, router, []models.SyncChange{delA})
	if resp2.Results[0].Status != services.SyncStatusApplied {
		t.Fatalf("delete A: %+v", resp2.Results[0])
	}

	// Device B never saw the delete; it edits from v1 with DIFFERENT content.
	// version == base+1 and the row is deleted, but the payload doesn't match
	// the row -> genuine conflict, not an idempotent delete retry.
	stale := syncCardChange("c-sd-1", "sync-stale-del", "B's v2 title", 1)
	resp3 := pushChanges(t, s, router, []models.SyncChange{stale})
	if resp3.Results[0].Status != services.SyncStatusConflict {
		t.Fatalf("stale update after delete: expected conflict, got %+v", resp3.Results[0])
	}
	if resp3.LostEdits != 1 {
		t.Errorf("lost_edits = %d, want 1", resp3.LostEdits)
	}
}

// TestSyncPushDeleteThenRecreateBatch: a batch containing BOTH a delete and an
// upsert of the same row_uuid must end with the row active, carrying the
// recreate's data — the same-batch upsert resurrects the row instead of being
// rejected as a stale conflict or misread as a create-retry.
func TestSyncPushDeleteThenRecreateBatch(t *testing.T) {
	s := NewHandler()
	defer tests.Teardown()
	router := newSyncRouter(s)

	create := syncCardChange("c-rc-1", "sync-recreate", "v1 title", 0)
	resp1 := pushChanges(t, s, router, []models.SyncChange{create})
	if resp1.Results[0].Status != services.SyncStatusApplied {
		t.Fatalf("create: %+v", resp1.Results[0])
	}

	// One batch: delete from v1, then re-create the same uuid from v1.
	delData, _ := json.Marshal(models.SyncCardData{})
	resp := pushChanges(t, s, router, []models.SyncChange{
		{Collection: services.SyncCollectionCards, RowUUID: "c-rc-1", Op: services.SyncOpDelete, BaseVersion: 1, Data: delData},
		syncCardChange("c-rc-1", "sync-recreate", "recreated title", 1),
	})
	for i, r := range resp.Results {
		if r.Status != services.SyncStatusApplied {
			t.Fatalf("result %d (%s): expected applied, got %+v", i, r.RowUUID, r)
		}
	}
	if resp.LostEdits != 0 {
		t.Errorf("lost_edits = %d, want 0 (the recreate must not be a lost edit)", resp.LostEdits)
	}

	var title string
	var isDeleted bool
	if err := s.GetDB().QueryRow(`SELECT title, is_deleted FROM cards WHERE sync_uuid = 'c-rc-1'`).Scan(&title, &isDeleted); err != nil {
		t.Fatal(err)
	}
	if isDeleted {
		t.Error("card still soft-deleted after same-batch recreate")
	}
	if title != "recreated title" {
		t.Errorf("server title = %q, want recreated title", title)
	}

	// Exactly one active row with that uuid.
	var count int
	if err := s.GetDB().QueryRow(`SELECT COUNT(*) FROM cards WHERE sync_uuid = 'c-rc-1' AND is_deleted = FALSE`).Scan(&count); err != nil {
		t.Fatal(err)
	}
	if count != 1 {
		t.Errorf("active row count = %d, want 1", count)
	}
}

// TestSyncPushTagMergeLostEdit pins the v5b.6 policy: a tag name-merge that
// discards a differing offline edit counts a lost edit.
func TestSyncPushTagMergeLostEdit(t *testing.T) {
	s := NewHandler()
	defer tests.Teardown()
	router := newSyncRouter(s)

	name := "sync-merge-loss-tag"
	devA, _ := json.Marshal(models.SyncTagData{Name: name, Color: "black"})
	devB, _ := json.Marshal(models.SyncTagData{Name: name, Color: "red"})

	respA := pushChanges(t, s, router, []models.SyncChange{
		{Collection: services.SyncCollectionTags, RowUUID: "tag-loss-a", Op: services.SyncOpUpsert, BaseVersion: 0, Data: devA},
	})
	if respA.Results[0].Status != services.SyncStatusApplied {
		t.Fatalf("device A tag: %+v", respA.Results[0])
	}

	// Device B's edit differs (red vs black): the merge discards it and must
	// count the lost edit.
	respB := pushChanges(t, s, router, []models.SyncChange{
		{Collection: services.SyncCollectionTags, RowUUID: "tag-loss-b", Op: services.SyncOpUpsert, BaseVersion: 0, Data: devB},
	})
	if respB.Results[0].Status != services.SyncStatusMerged {
		t.Fatalf("device B tag: expected merged, got %+v", respB.Results[0])
	}
	if respB.LostEdits != 1 {
		t.Errorf("lost_edits = %d, want 1 (differing color discarded by merge)", respB.LostEdits)
	}
	// The surviving row is authoritative; the client must adopt its data.
	var color string
	if err := s.GetDB().QueryRow(`SELECT color FROM tags WHERE sync_uuid = 'tag-loss-a'`).Scan(&color); err != nil {
		t.Fatal(err)
	}
	if color != "black" {
		t.Errorf("surviving tag color = %q, want black", color)
	}
}

// TestSyncPushTagMergeIdenticalNoLoss: a same-named tag pushed with data
// identical to the surviving row reports no lost edit.
func TestSyncPushTagMergeIdenticalNoLoss(t *testing.T) {
	s := NewHandler()
	defer tests.Teardown()
	router := newSyncRouter(s)

	name := "sync-merge-same-tag"
	same, _ := json.Marshal(models.SyncTagData{Name: name, Color: "blue"})

	respA := pushChanges(t, s, router, []models.SyncChange{
		{Collection: services.SyncCollectionTags, RowUUID: "tag-same-a", Op: services.SyncOpUpsert, BaseVersion: 0, Data: same},
	})
	if respA.Results[0].Status != services.SyncStatusApplied {
		t.Fatalf("device A tag: %+v", respA.Results[0])
	}
	respB := pushChanges(t, s, router, []models.SyncChange{
		{Collection: services.SyncCollectionTags, RowUUID: "tag-same-b", Op: services.SyncOpUpsert, BaseVersion: 0, Data: same},
	})
	if respB.Results[0].Status != services.SyncStatusMerged {
		t.Fatalf("device B tag: expected merged, got %+v", respB.Results[0])
	}
	if respB.LostEdits != 0 {
		t.Errorf("identical merge lost_edits = %d, want 0", respB.LostEdits)
	}
}

// TestSyncPushDeleteRefusedWhenChildren: the sync delete path must mirror
// DeleteCard's guard — a card with children cannot be deleted. The push is
// conflict-rejected and the card stays active.
func TestSyncPushDeleteRefusedWhenChildren(t *testing.T) {
	s := NewHandler()
	defer tests.Teardown()
	router := newSyncRouter(s)

	parent := syncCardChange("c-guard-par", "syncguardparent", "Parent", 0)
	child := syncCardChange("c-guard-chi", "syncguardparent/syncguardchild", "Child", 0)
	pushChanges(t, s, router, []models.SyncChange{parent, child})

	delData, _ := json.Marshal(models.SyncCardData{})
	del := models.SyncChange{Collection: services.SyncCollectionCards, RowUUID: "c-guard-par", Op: services.SyncOpDelete, BaseVersion: 1, Data: delData}
	resp := pushChanges(t, s, router, []models.SyncChange{del})
	if resp.Results[0].Status != services.SyncStatusConflict {
		t.Fatalf("delete with children: expected conflict, got %+v", resp.Results[0])
	}

	var isDeleted bool
	if err := s.GetDB().QueryRow(`SELECT is_deleted FROM cards WHERE sync_uuid = 'c-guard-par'`).Scan(&isDeleted); err != nil {
		t.Fatal(err)
	}
	if isDeleted {
		t.Error("parent card must survive a refused delete")
	}
}

// TestSyncPushDeleteRefusedWhenBacklinks: same guard for backlinks.
func TestSyncPushDeleteRefusedWhenBacklinks(t *testing.T) {
	s := NewHandler()
	defer tests.Teardown()
	router := newSyncRouter(s)

	target := syncCardChange("c-guard-tgt", "syncguardtarget", "Target", 0)
	source := syncCardChange("c-guard-src", "syncguardsource", "Source", 0)
	resp := pushChanges(t, s, router, []models.SyncChange{target, source})
	if resp.Results[0].Status != services.SyncStatusApplied {
		t.Fatalf("create cards: %+v", resp.Results[0])
	}
	targetID := *resp.Results[0].ServerID
	sourceID := *resp.Results[1].ServerID

	// A backlink from the source card to the target (matches GetBacklinks).
	if _, err := s.GetDB().Exec(`INSERT INTO backlinks (source_id, target_id, source_id_int, target_id_int, created_at, updated_at) VALUES ('syncguardsource', 'syncguardtarget', $1, $2, CURRENT_TIMESTAMP, CURRENT_TIMESTAMP)`, sourceID, targetID); err != nil {
		t.Fatal(err)
	}

	delData, _ := json.Marshal(models.SyncCardData{})
	del := models.SyncChange{Collection: services.SyncCollectionCards, RowUUID: "c-guard-tgt", Op: services.SyncOpDelete, BaseVersion: 1, Data: delData}
	respDel := pushChanges(t, s, router, []models.SyncChange{del})
	if respDel.Results[0].Status != services.SyncStatusConflict {
		t.Fatalf("delete with backlinks: expected conflict, got %+v", respDel.Results[0])
	}

	var isDeleted bool
	if err := s.GetDB().QueryRow(`SELECT is_deleted FROM cards WHERE sync_uuid = 'c-guard-tgt'`).Scan(&isDeleted); err != nil {
		t.Fatal(err)
	}
	if isDeleted {
		t.Error("target card must survive a refused delete")
	}
}

// TestSyncPushDeleteCleansFacts: deleting a normal card via push runs the same
// fact/junction cleanup as DeleteCard.
func TestSyncPushDeleteCleansFacts(t *testing.T) {
	s := NewHandler()
	defer tests.Teardown()
	router := newSyncRouter(s)

	create := syncCardChange("c-clean-1", "sync-cleanup", "Doomed", 0)
	resp := pushChanges(t, s, router, []models.SyncChange{create})
	if resp.Results[0].Status != services.SyncStatusApplied {
		t.Fatalf("create: %+v", resp.Results[0])
	}
	cardID := *resp.Results[0].ServerID

	// Fact originating from the card + its junction + an entity junction.
	var factID int
	if err := s.GetDB().QueryRow(`INSERT INTO facts (card_pk, user_id, fact, created_at, updated_at) VALUES ($1, 1, 'fact', CURRENT_TIMESTAMP, CURRENT_TIMESTAMP) RETURNING id`, cardID).Scan(&factID); err != nil {
		t.Fatal(err)
	}
	if _, err := s.GetDB().Exec(`INSERT INTO fact_card_junction (fact_id, card_pk, user_id, is_origin) VALUES ($1, $2, 1, TRUE)`, factID, cardID); err != nil {
		t.Fatal(err)
	}
	if _, err := s.GetDB().Exec(`INSERT INTO entity_card_junction (user_id, entity_id, card_pk) VALUES (1, 1, $1)`, cardID); err != nil {
		t.Fatal(err)
	}

	delData, _ := json.Marshal(models.SyncCardData{})
	del := models.SyncChange{Collection: services.SyncCollectionCards, RowUUID: "c-clean-1", Op: services.SyncOpDelete, BaseVersion: 1, Data: delData}
	respDel := pushChanges(t, s, router, []models.SyncChange{del})
	if respDel.Results[0].Status != services.SyncStatusApplied {
		t.Fatalf("delete: %+v", respDel.Results[0])
	}

	var factCount, junctionCount, entityCount int
	if err := s.GetDB().QueryRow(`SELECT COUNT(*) FROM facts WHERE card_pk = $1 AND user_id = 1`, cardID).Scan(&factCount); err != nil {
		t.Fatal(err)
	}
	if err := s.GetDB().QueryRow(`SELECT COUNT(*) FROM fact_card_junction WHERE fact_id = $1`, factID).Scan(&junctionCount); err != nil {
		t.Fatal(err)
	}
	if err := s.GetDB().QueryRow(`SELECT COUNT(*) FROM entity_card_junction WHERE card_pk = $1 AND user_id = 1`, cardID).Scan(&entityCount); err != nil {
		t.Fatal(err)
	}
	if factCount != 0 || junctionCount != 0 || entityCount != 0 {
		t.Errorf("derived data survived delete: facts=%d junctions=%d entities=%d, want all 0", factCount, junctionCount, entityCount)
	}

	var isDeleted bool
	if err := s.GetDB().QueryRow(`SELECT is_deleted FROM cards WHERE sync_uuid = 'c-clean-1'`).Scan(&isDeleted); err != nil {
		t.Fatal(err)
	}
	if !isDeleted {
		t.Error("card not soft-deleted")
	}
}

// TestSyncMultiUserIsolation: user A's snapshot cursor and changes feed must
// never expose or skip user B's rows — sync_log is a global max cursor but the
// feed and snapshot filter by user_id.
func TestSyncMultiUserIsolation(t *testing.T) {
	s := NewHandler()
	defer tests.Teardown()
	router := newSyncRouter(s)

	// Second user for the isolation check (fixture data only has user 1).
	if _, err := s.GetDB().Exec(`INSERT INTO users (username, email, password, created_at, updated_at) VALUES ('user2', 'user2@test.local', 'x', CURRENT_TIMESTAMP, CURRENT_TIMESTAMP)`); err != nil {
		t.Fatal(err)
	}

	// User 1 pushes a card; user 2 must never see it.
	resp1 := pushChanges(t, s, router, []models.SyncChange{syncCardChange("c-iso-1", "sync-isolation", "user1 card", 0)})
	if resp1.Results[0].Status != services.SyncStatusApplied {
		t.Fatalf("user1 push: %+v", resp1.Results[0])
	}

	// User 2's snapshot has no user-1 rows.
	snap2 := decodeSyncResp[models.SyncSnapshotResponse](t, syncRequestAs(t, s, router, 2, "GET", "/api/sync/snapshot", nil))
	for _, row := range snap2.Collections["cards"] {
		if row.RowUUID == "c-iso-1" {
			t.Error("user2 snapshot leaks user1's card")
		}
	}

	// User 2's changes feed is free of user-1 entries.
	feed2 := decodeSyncResp[models.SyncChangesResponse](t, syncRequestAs(t, s, router, 2, "GET", "/api/sync/changes?since=0", nil))
	for _, row := range feed2.Rows {
		if row.RowUUID == "c-iso-1" {
			t.Error("user2 feed leaks user1's change")
		}
	}

	// User 2 pushes its own card; user 1's feed must not show it, and must
	// still show user 1's own card (nothing skipped by the shared cursor).
	resp2 := pushChangesAs(t, s, router, 2, []models.SyncChange{syncCardChange("c-iso-2", "sync-isolation-2", "user2 card", 0)})
	if resp2.Results[0].Status != services.SyncStatusApplied {
		t.Fatalf("user2 push: %+v", resp2.Results[0])
	}
	feed1 := decodeSyncResp[models.SyncChangesResponse](t, syncRequest(t, s, router, "GET", "/api/sync/changes?since=0", nil))
	foundOwn := false
	for _, row := range feed1.Rows {
		if row.RowUUID == "c-iso-2" {
			t.Error("user1 feed leaks user2's change")
		}
		if row.RowUUID == "c-iso-1" {
			foundOwn = true
		}
	}
	if !foundOwn {
		t.Error("user1 feed missing own card (global cursor must not skip rows)")
	}
}

// TestSyncPushPerUserSyncUUIDSharing (Zettelgarden-xre): sync_uuid identity
// is per-user — two accounts may legitimately create rows with the same
// sync_uuid offline (each has its own namespace). The sync_uuid unique index
// must be scoped (user_id, sync_uuid): with the pre-xre GLOBAL index, user 2's
// create was silently ignored (the harness's rename-vs-create divergence).
func TestSyncPushPerUserSyncUUIDSharing(t *testing.T) {
	s := NewHandler()
	defer tests.Teardown()
	router := newSyncRouter(s)

	if _, err := s.GetDB().Exec(`INSERT INTO users (username, email, password, created_at, updated_at) VALUES ('user2', 'shared-uuid@test.local', 'x', CURRENT_TIMESTAMP, CURRENT_TIMESTAMP)`); err != nil {
		t.Fatal(err)
	}

	// User 1 creates a tag with sync_uuid "tag-w".
	resp1 := pushChanges(t, s, router, []models.SyncChange{
		syncTagChange("tag-w", "work", "blue", 0),
	})
	if resp1.Results[0].Status != services.SyncStatusApplied {
		t.Fatalf("user1 tag push: %+v", resp1.Results[0])
	}

	// User 2 creates a tag with the SAME sync_uuid — must apply, not be
	// silently ignored by a global uniqueness constraint.
	resp2 := pushChangesAs(t, s, router, 2, []models.SyncChange{
		syncTagChange("tag-w", "work", "red", 0),
	})
	if resp2.Results[0].Status != services.SyncStatusApplied {
		t.Fatalf("user2 tag push with shared sync_uuid: %+v (want applied, not ignored)", resp2.Results[0])
	}
}

// TestSyncFeedPagination: the changes feed pages at syncFeedPageSize with a
// stable hasMore + cursor contract past 500 rows.
func TestSyncFeedPagination(t *testing.T) {
	s := NewHandler()
	defer tests.Teardown()
	router := newSyncRouter(s)

	// Other tests may have committed sync_log entries through the pool; start
	// the pagination from the current high-water mark.
	var sinceID int64
	if err := s.GetDB().QueryRow(`SELECT COALESCE(MAX(id), 0) FROM sync_log`).Scan(&sinceID); err != nil {
		t.Fatal(err)
	}

	total := syncFeedPageSize + 1 // 501
	changes := make([]models.SyncChange, 0, total)
	for i := 0; i < total; i++ {
		changes = append(changes, syncCardChange(fmt.Sprintf("c-page-%03d", i), fmt.Sprintf("sync-page-%03d", i), fmt.Sprintf("Page card %03d", i), 0))
	}
	resp := pushChanges(t, s, router, changes)
	if resp.LostEdits != 0 {
		t.Fatalf("lost_edits = %d, want 0", resp.LostEdits)
	}

	page1 := decodeSyncResp[models.SyncChangesResponse](t, syncRequest(t, s, router, "GET", "/api/sync/changes?since="+strconv.FormatInt(sinceID, 10), nil))
	if len(page1.Rows) != syncFeedPageSize {
		t.Fatalf("page1 rows = %d, want %d", len(page1.Rows), syncFeedPageSize)
	}
	if !page1.HasMore {
		t.Error("page1 must report hasMore")
	}

	page2 := decodeSyncResp[models.SyncChangesResponse](t, syncRequest(t, s, router, "GET", "/api/sync/changes?since="+strconv.FormatInt(page1.Cursor, 10), nil))
	if len(page2.Rows) != 1 {
		t.Fatalf("page2 rows = %d, want 1", len(page2.Rows))
	}
	if page2.HasMore {
		t.Error("page2 must not report hasMore")
	}
	if page2.Cursor < page1.Cursor {
		t.Errorf("page2 cursor = %d, want >= %d (cursor must not regress)", page2.Cursor, page1.Cursor)
	}
}

// pushChangesAs pushes a batch as an explicit user (multi-user tests).
func pushChangesAs(t *testing.T, s *Handler, router *mux.Router, userID int, changes []models.SyncChange) models.SyncPushResponse {
	t.Helper()
	rr := syncRequestAs(t, s, router, userID, "POST", "/api/sync/push", models.SyncPushRequest{Changes: changes, DeviceID: "test-device"})
	return decodeSyncResp[models.SyncPushResponse](t, rr)
}

// TestSyncChangesFeedResetAfterPrune proves the handler side of retention: a
// client whose since cursor predates the pruned boundary gets reset=true (and
// must re-bootstrap); a cursor at the boundary still fetches incrementally.
func TestSyncChangesFeedResetAfterPrune(t *testing.T) {
	s := NewHandler()
	defer tests.Teardown()
	router := newSyncRouter(s)
	userID := 2
	now := time.Now()

	old := now.Add(-60 * 24 * time.Hour).UTC().Format("2006-01-02 15:04:05")
	var oldIDs []int64
	for i := 0; i < 3; i++ {
		var id int64
		if err := s.GetDB().QueryRow(`INSERT INTO sync_log (user_id, collection, row_uuid, op, version, created_at) VALUES ($1, 'cards', $2, 'upsert', 1, $3) RETURNING id`,
			userID, fmt.Sprintf("c-feed-old-%d", i), old).Scan(&id); err != nil {
			t.Fatal(err)
		}
		oldIDs = append(oldIDs, id)
	}
	var recentID int64
	if err := s.GetDB().QueryRow(`INSERT INTO sync_log (user_id, collection, row_uuid, op, version) VALUES ($1, 'cards', 'c-feed-recent', 'upsert', 1) RETURNING id`, userID).Scan(&recentID); err != nil {
		t.Fatal(err)
	}
	if err := services.UpsertSyncClient(s.GetDB(), userID, "dev-a", oldIDs[len(oldIDs)-1]); err != nil {
		t.Fatal(err)
	}
	pruned, err := services.PruneSyncLog(s.GetDB(), userID, now, services.SyncLogRetention)
	if err != nil {
		t.Fatal(err)
	}
	if pruned != 3 {
		t.Fatalf("pruned = %d, want 3", pruned)
	}

	// Stale cursor (0) predates the boundary -> reset.
	feed := decodeSyncResp[models.SyncChangesResponse](t, syncRequestAs(t, s, router, userID, "GET", "/api/sync/changes?since=0", nil))
	if !feed.Reset {
		t.Fatalf("stale cursor: expected reset, got %+v", feed)
	}
	// Cursor at the boundary (minID-1) -> normal incremental feed.
	feed2 := decodeSyncResp[models.SyncChangesResponse](t, syncRequestAs(t, s, router, userID, "GET", "/api/sync/changes?since="+strconv.FormatInt(recentID-1, 10), nil))
	if feed2.Reset {
		t.Fatalf("boundary cursor: unexpected reset, got %+v", feed2)
	}
	if len(feed2.Rows) != 1 || feed2.Rows[0].RowUUID != "c-feed-recent" {
		t.Errorf("boundary feed rows = %+v, want the surviving recent row", feed2.Rows)
	}
}

// TestSyncPushCreateRetryWithEdit: a dropped create response followed by a
// local edit must NOT be silently swallowed by the base-0 create-retry
// shortcut — LWW reports a visible conflict + lost edit (review P1-1).
func TestSyncPushCreateRetryWithEdit(t *testing.T) {
	s := NewHandler()
	defer tests.Teardown()
	router := newSyncRouter(s)

	create := syncCardChange("c-edit-1", "sync-edit-retry", "v1 title", 0)
	resp1 := pushChanges(t, s, router, []models.SyncChange{create})
	if resp1.Results[0].Status != services.SyncStatusApplied {
		t.Fatalf("create: %+v", resp1.Results[0])
	}

	// Retry of the same entry, but the user edited before retrying: payload
	// differs from the server row. Must conflict (visible), never a silent
	// applied-with-no-write that clobbers the edit on the next pull.
	edited := syncCardChange("c-edit-1", "sync-edit-retry", "edited title", 0)
	resp2 := pushChanges(t, s, router, []models.SyncChange{edited})
	if resp2.Results[0].Status != services.SyncStatusConflict {
		t.Fatalf("edited create retry: expected conflict, got %+v", resp2.Results[0])
	}
	if resp2.LostEdits != 1 {
		t.Errorf("lost_edits = %d, want 1", resp2.LostEdits)
	}
	var title string
	if err := s.GetDB().QueryRow(`SELECT title FROM cards WHERE sync_uuid = 'c-edit-1'`).Scan(&title); err != nil {
		t.Fatal(err)
	}
	if title != "v1 title" {
		t.Errorf("server title = %q, want v1 title (LWW kept the server row)", title)
	}
}

// TestSyncPushCrossBatchDeleteRecreate: the delete syncs (outbox drained),
// THEN the client re-creates the same row_uuid. The mirror row is gone so the
// recreate pushes base 0 — the server must resurrect the soft-deleted row
// instead of LWW-conflicting the new content away (review P1-2).
func TestSyncPushCrossBatchDeleteRecreate(t *testing.T) {
	s := NewHandler()
	defer tests.Teardown()
	router := newSyncRouter(s)

	create := syncCardChange("c-xb-1", "sync-xbatch", "v1 title", 0)
	resp1 := pushChanges(t, s, router, []models.SyncChange{create})
	if resp1.Results[0].Status != services.SyncStatusApplied {
		t.Fatalf("create: %+v", resp1.Results[0])
	}
	delData, _ := json.Marshal(models.SyncCardData{})
	del := models.SyncChange{Collection: services.SyncCollectionCards, RowUUID: "c-xb-1", Op: services.SyncOpDelete, BaseVersion: 1, Data: delData}
	respDel := pushChanges(t, s, router, []models.SyncChange{del})
	if respDel.Results[0].Status != services.SyncStatusApplied {
		t.Fatalf("delete: %+v", respDel.Results[0])
	}

	// Recreate in a LATER batch with base 0: resurrect, not conflict.
	recreate := syncCardChange("c-xb-1", "sync-xbatch", "recreated title", 0)
	resp3 := pushChanges(t, s, router, []models.SyncChange{recreate})
	if resp3.Results[0].Status != services.SyncStatusApplied {
		t.Fatalf("cross-batch recreate: expected applied (resurrect), got %+v", resp3.Results[0])
	}
	if resp3.LostEdits != 0 {
		t.Errorf("lost_edits = %d, want 0", resp3.LostEdits)
	}
	var title string
	var isDeleted bool
	if err := s.GetDB().QueryRow(`SELECT title, is_deleted FROM cards WHERE sync_uuid = 'c-xb-1'`).Scan(&title, &isDeleted); err != nil {
		t.Fatal(err)
	}
	if isDeleted {
		t.Error("card still soft-deleted after cross-batch recreate")
	}
	if title != "recreated title" {
		t.Errorf("title = %q, want recreated title", title)
	}
}

// TestSyncPushTagDeleteFallbackResultUUID: when a tag delete falls back to a
// name-keyed lookup (the client uuid is not on the server), the push RESULT
// must carry the client's uuid so the engine drops and reconciles the right
// outbox entry — otherwise the delete re-pushes forever (review P1-3).
func TestSyncPushTagDeleteFallbackResultUUID(t *testing.T) {
	s := NewHandler()
	defer tests.Teardown()
	router := newSyncRouter(s)

	name := "sync-fallback-tag"
	devA, _ := json.Marshal(models.SyncTagData{Name: name, Color: "black"})
	respA := pushChanges(t, s, router, []models.SyncChange{
		{Collection: services.SyncCollectionTags, RowUUID: "tag-fb-a", Op: services.SyncOpUpsert, BaseVersion: 0, Data: devA},
	})
	if respA.Results[0].Status != services.SyncStatusApplied {
		t.Fatalf("create: %+v", respA.Results[0])
	}

	// Device B holds a pre-merge uuid "tag-fb-b" for the same name and deletes.
	delData, _ := json.Marshal(models.SyncTagData{Name: name, Color: "black"})
	del := models.SyncChange{Collection: services.SyncCollectionTags, RowUUID: "tag-fb-b", Op: services.SyncOpDelete, BaseVersion: 1, Data: delData}
	respDel := pushChanges(t, s, router, []models.SyncChange{del})
	if respDel.Results[0].Status != services.SyncStatusApplied {
		t.Fatalf("delete: %+v", respDel.Results[0])
	}
	if respDel.Results[0].RowUUID != "tag-fb-b" {
		t.Errorf("result row_uuid = %q, want tag-fb-b (client uuid, so the engine can drop the outbox entry)", respDel.Results[0].RowUUID)
	}
	var isDeleted bool
	if err := s.GetDB().QueryRow(`SELECT is_deleted FROM tags WHERE sync_uuid = 'tag-fb-a'`).Scan(&isDeleted); err != nil {
		t.Fatal(err)
	}
	if !isDeleted {
		t.Error("tag not soft-deleted")
	}
}

// TestSyncPushTagRecreateResurrect: re-creating a deleted tag NAME with a
// fresh uuid resurrects the soft-deleted row (REST CreateTag semantics)
// instead of merging onto a tombstone (review P2-4).
func TestSyncPushTagRecreateResurrect(t *testing.T) {
	s := NewHandler()
	defer tests.Teardown()
	router := newSyncRouter(s)

	name := "sync-resurrect-tag"
	tagData, _ := json.Marshal(models.SyncTagData{Name: name, Color: "black"})
	resp1 := pushChanges(t, s, router, []models.SyncChange{
		{Collection: services.SyncCollectionTags, RowUUID: "tag-rs-1", Op: services.SyncOpUpsert, BaseVersion: 0, Data: tagData},
	})
	if resp1.Results[0].Status != services.SyncStatusApplied {
		t.Fatalf("create: %+v", resp1.Results[0])
	}
	delData, _ := json.Marshal(models.SyncTagData{Name: name})
	del := models.SyncChange{Collection: services.SyncCollectionTags, RowUUID: "tag-rs-1", Op: services.SyncOpDelete, BaseVersion: 1, Data: delData}
	if resp := pushChanges(t, s, router, []models.SyncChange{del}); resp.Results[0].Status != services.SyncStatusApplied {
		t.Fatalf("delete: %+v", resp.Results[0])
	}

	// A fresh uuid re-creating the deleted name must resurrect the row.
	recreate, _ := json.Marshal(models.SyncTagData{Name: name, Color: "red"})
	resp2 := pushChanges(t, s, router, []models.SyncChange{
		{Collection: services.SyncCollectionTags, RowUUID: "tag-rs-2", Op: services.SyncOpUpsert, BaseVersion: 0, Data: recreate},
	})
	if resp2.Results[0].Status != services.SyncStatusMerged {
		t.Fatalf("recreate: expected merged (resurrected), got %+v", resp2.Results[0])
	}
	if resp2.LostEdits != 0 {
		t.Errorf("lost_edits = %d, want 0 (resurrection is not a loss)", resp2.LostEdits)
	}
	var color string
	var isDeleted bool
	if err := s.GetDB().QueryRow(`SELECT color, is_deleted FROM tags WHERE sync_uuid = 'tag-rs-1'`).Scan(&color, &isDeleted); err != nil {
		t.Fatal(err)
	}
	if isDeleted {
		t.Error("tag not resurrected")
	}
	if color != "red" {
		t.Errorf("color = %q, want red (recreate content applied)", color)
	}
}

// TestSyncPushRejectsInvalidOp: the push route must reject unknown ops instead
// of silently treating them as upserts (review P3-5).
func TestSyncPushRejectsInvalidOp(t *testing.T) {
	s := NewHandler()
	defer tests.Teardown()
	router := newSyncRouter(s)

	data, _ := json.Marshal(models.SyncCardData{Title: "x"})
	rr := syncRequest(t, s, router, "POST", "/api/sync/push", models.SyncPushRequest{Changes: []models.SyncChange{
		{Collection: services.SyncCollectionCards, RowUUID: "c-bad-op", Op: "frobnicate", BaseVersion: 0, Data: data},
	}})
	if rr.Code != http.StatusBadRequest {
		t.Fatalf("expected 400 for invalid op, got %d", rr.Code)
	}
}

// strPtr/intPtr are tiny helpers for pointer-typed task fields in the retry
// tests above.
func strPtr(s string) *string { return &s }

func intPtr(i int) *int { return &i }

func TestSyncPushStaleBaseConflict(t *testing.T) {
	s := NewHandler()
	defer tests.Teardown()
	router := newSyncRouter(s)

	// Create v1.
	create := syncCardChange("c-conf-1", "sync-conflict", "v1 title", 0)
	resp1 := pushChanges(t, s, router, []models.SyncChange{create})
	if resp1.Results[0].Status != services.SyncStatusApplied {
		t.Fatalf("create: %+v", resp1.Results[0])
	}

	// Legit update from v1 -> v2.
	update := syncCardChange("c-conf-1", "sync-conflict", "v2 title", 1)
	resp2 := pushChanges(t, s, router, []models.SyncChange{update})
	if resp2.Results[0].Status != services.SyncStatusApplied {
		t.Fatalf("update: %+v", resp2.Results[0])
	}

	// Stale update from v1 (server is now v2): LWW keeps the server row.
	stale := syncCardChange("c-conf-1", "sync-conflict", "stale title", 1)
	resp3 := pushChanges(t, s, router, []models.SyncChange{stale})
	if resp3.Results[0].Status != services.SyncStatusConflict {
		t.Fatalf("stale push: expected conflict, got %+v", resp3.Results[0])
	}
	if resp3.LostEdits != 1 {
		t.Errorf("lost_edits = %d, want 1", resp3.LostEdits)
	}
	if resp3.Results[0].ServerVersion != 2 {
		t.Errorf("conflict server_version = %d, want 2", resp3.Results[0].ServerVersion)
	}
	// Server row unchanged.
	var title string
	if err := s.GetDB().QueryRow(`SELECT title FROM cards WHERE sync_uuid = 'c-conf-1'`).Scan(&title); err != nil {
		t.Fatal(err)
	}
	if title != "v2 title" {
		t.Errorf("server title = %q, want v2 title (LWW kept newer)", title)
	}
}

// TestSyncPushIdenticalContentStaleBaseNoConflict is the dsd decision: a stale
// push whose payload is byte-identical to the current row is NOT a conflict,
// even when the row advanced by more than one version because another device
// happened to set the same value. The old guard required version == base+1 and
// counted a spurious lost edit for an edit whose intent is fully reflected.
func TestSyncPushIdenticalContentStaleBaseNoConflict(t *testing.T) {
	s := NewHandler()
	defer tests.Teardown()
	router := newSyncRouter(s)

	// Create v1.
	create := syncCardChange("c-dsd-1", "sync-dsd", "v1 title", 0)
	resp1 := pushChanges(t, s, router, []models.SyncChange{create})
	if resp1.Results[0].Status != services.SyncStatusApplied {
		t.Fatalf("create: %+v", resp1.Results[0])
	}

	// Device A sets title "same" (v2). Device B — having adopted v2 — sets the
	// IDENTICAL title from base 2 (v3), advancing the row two versions past A's
	// base before A's retry lands.
	updA := syncCardChange("c-dsd-1", "sync-dsd", "same title", 1)
	if r := pushChanges(t, s, router, []models.SyncChange{updA}); r.Results[0].Status != services.SyncStatusApplied {
		t.Fatalf("update A: %+v", r.Results[0])
	}
	updB := syncCardChange("c-dsd-1", "sync-dsd", "same title", 2)
	if r := pushChanges(t, s, router, []models.SyncChange{updB}); r.Results[0].Status != services.SyncStatusApplied {
		t.Fatalf("update B: %+v", r.Results[0])
	}

	// A's dropped-response retry arrives with base 1 but the row is now v3
	// (not base+1). Identical content -> applied, zero lost edits.
	retry := pushChanges(t, s, router, []models.SyncChange{updA})
	if retry.Results[0].Status != services.SyncStatusApplied {
		t.Fatalf("identical-content stale retry: expected applied, got %+v", retry.Results[0])
	}
	if retry.LostEdits != 0 {
		t.Errorf("identical-content stale retry counted %d lost edits, want 0", retry.LostEdits)
	}
	if retry.Results[0].ServerVersion != 3 {
		t.Errorf("server_version = %d, want 3", retry.Results[0].ServerVersion)
	}
}

// TestSyncPushRejectsDuplicateCardID is the idp decision: the sync push path
// enforces card_id uniqueness exactly like REST (checkIsCardIDUnique), so two
// offline devices creating the same card_id cannot both land — the loser gets
// a conflict + lost edit and the server stays free of duplicates that would
// corrupt parent/backlink/root-id lookups.
func TestSyncPushRejectsDuplicateCardID(t *testing.T) {
	s := NewHandler()
	defer tests.Teardown()
	router := newSyncRouter(s)

	// Device A creates card_id DUP1 (uuid c-idp-a).
	if r := pushChanges(t, s, router, []models.SyncChange{syncCardChange("c-idp-a", "DUP1", "winner", 0)}); r.Results[0].Status != services.SyncStatusApplied {
		t.Fatalf("create A: %+v", r.Results[0])
	}

	// Device B creates the SAME card_id (uuid c-idp-b): the loser is MERGED
	// onto the winner (adopt the winner's uuid, like the tag name-merge) so no
	// ghost row lingers to re-conflict on every edit. Different content -> the
	// losing edit is counted.
	loser := syncCardChange("c-idp-b", "DUP1", "loser", 0)
	resp := pushChanges(t, s, router, []models.SyncChange{loser})
	if resp.Results[0].Status != services.SyncStatusMerged {
		t.Fatalf("duplicate card_id create: expected merged, got %+v", resp.Results[0])
	}
	if resp.Results[0].MappedToRowUUID == "" {
		t.Errorf("merged result missing mapped_to_row_uuid: %+v", resp.Results[0])
	}
	if resp.LostEdits != 1 {
		t.Errorf("lost_edits = %d, want 1 (loser's different content discarded)", resp.LostEdits)
	}
	// The merge payload is the winner's row (server-authoritative). Note:
	// currentRow emits SQLite-native 0/1 for is_deleted (the engine's
	// rowIsDeleted handles the truthiness), so assert via a loose map.
	var mergeData map[string]interface{}
	if err := json.Unmarshal(resp.Results[0].Data, &mergeData); err != nil {
		t.Fatalf("merge data: %v", err)
	}
	if mergeData["card_id"] != "DUP1" || mergeData["title"] != "winner" {
		t.Errorf("merge data = %v, want winner's row", mergeData)
	}

	// Server has exactly ONE live row with DUP1 (the winner).
	var count int
	if err := s.GetDB().QueryRow(`SELECT COUNT(*) FROM cards WHERE card_id = 'DUP1' AND is_deleted = FALSE`).Scan(&count); err != nil {
		t.Fatal(err)
	}
	if count != 1 {
		t.Fatalf("live rows with card_id DUP1 = %d, want 1", count)
	}

	// Rename collision: B's card (uuid c-idp-b2, card_id DUP2) renames to
	// DUP1 -> conflict, keeps DUP2 server-side.
	if r := pushChanges(t, s, router, []models.SyncChange{syncCardChange("c-idp-b2", "DUP2", "b card", 0)}); r.Results[0].Status != services.SyncStatusApplied {
		t.Fatalf("create B2: %+v", r.Results[0])
	}
	rename := syncCardChange("c-idp-b2", "DUP1", "b card renamed", 1)
	respR := pushChanges(t, s, router, []models.SyncChange{rename})
	if respR.Results[0].Status != services.SyncStatusConflict {
		t.Fatalf("rename to owned card_id: expected conflict, got %+v", respR.Results[0])
	}
	var serverCardID string
	if err := s.GetDB().QueryRow(`SELECT card_id FROM cards WHERE sync_uuid = 'c-idp-b2'`).Scan(&serverCardID); err != nil {
		t.Fatal(err)
	}
	if serverCardID != "DUP2" {
		t.Errorf("server card_id after refused rename = %q, want DUP2", serverCardID)
	}

	// Empty card_id is exempt (REST allows empty).
	if r := pushChanges(t, s, router, []models.SyncChange{syncCardChange("c-idp-empty", "", "no id", 0)}); r.Results[0].Status != services.SyncStatusApplied {
		t.Fatalf("create with empty card_id: expected applied, got %+v", r.Results[0])
	}
}

func TestSyncPushDeletePropagates(t *testing.T) {
	s := NewHandler()
	defer tests.Teardown()
	router := newSyncRouter(s)

	boot := decodeSyncResp[models.SyncSnapshotResponse](t, syncRequest(t, s, router, "GET", "/api/sync/snapshot", nil))
	create := syncCardChange("c-del-1", "sync-delete", "doomed", 0)
	pushChanges(t, s, router, []models.SyncChange{create})

	delData, _ := json.Marshal(models.SyncCardData{})
	rr := syncRequest(t, s, router, "POST", "/api/sync/push", models.SyncPushRequest{Changes: []models.SyncChange{
		{Collection: services.SyncCollectionCards, RowUUID: "c-del-1", Op: services.SyncOpDelete, BaseVersion: 1, Data: delData},
	}})
	resp := decodeSyncResp[models.SyncPushResponse](t, rr)
	if resp.Results[0].Status != services.SyncStatusApplied {
		t.Fatalf("delete: %+v", resp.Results[0])
	}

	var isDeleted bool
	if err := s.GetDB().QueryRow(`SELECT is_deleted FROM cards WHERE sync_uuid = 'c-del-1'`).Scan(&isDeleted); err != nil {
		t.Fatal(err)
	}
	if !isDeleted {
		t.Error("card not soft-deleted")
	}

	// The feed must carry the delete tombstone.
	feed := decodeSyncResp[models.SyncChangesResponse](t, syncRequest(t, s, router, "GET",
		"/api/sync/changes?since="+strconv.FormatInt(boot.Cursor, 10), nil))
	seen := false
	for _, row := range feed.Rows {
		if row.RowUUID == "c-del-1" && row.Op == services.SyncOpDelete {
			seen = true
		}
	}
	if !seen {
		t.Error("delete tombstone missing from feed")
	}

	// Snapshot must exclude the deleted row.
	snap := decodeSyncResp[models.SyncSnapshotResponse](t, syncRequest(t, s, router, "GET", "/api/sync/snapshot", nil))
	for _, row := range snap.Collections["cards"] {
		if row.RowUUID == "c-del-1" {
			t.Error("deleted card still in snapshot")
		}
	}
}

func TestSyncPushServerAuthoritativeParentage(t *testing.T) {
	s := NewHandler()
	defer tests.Teardown()
	router := newSyncRouter(s)

	// Parent exists on the server already (created via push, root card).
	parent := syncCardChange("c-par-1", "syncparent", "Parent", 0)
	pushChanges(t, s, router, []models.SyncChange{parent})

	// Child card: card_id encodes the hierarchy; server derives parent_id.
	child := syncCardChange("c-chi-1", "syncparent/syncchild", "Child", 0)
	resp := pushChanges(t, s, router, []models.SyncChange{child})
	if resp.Results[0].Status != services.SyncStatusApplied {
		t.Fatalf("child: %+v", resp.Results[0])
	}

	var parentID int
	if err := s.GetDB().QueryRow(`SELECT parent_id FROM cards WHERE sync_uuid = 'c-chi-1'`).Scan(&parentID); err != nil {
		t.Fatal(err)
	}
	if parentID == 0 || parentID == *resp.Results[0].ServerID {
		t.Errorf("child parent_id = %d, want the parent card's id", parentID)
	}
}

// TestSyncPushBatchOutOfOrderFKResolution proves the apply loop's topological
// ordering: a task sent BEFORE its card in the same batch still resolves
// card_pk_uuid (cards are applied before tasks regardless of send order).
func TestSyncPushBatchOutOfOrderFKResolution(t *testing.T) {
	s := NewHandler()
	defer tests.Teardown()
	router := newSyncRouter(s)

	cardData, _ := json.Marshal(models.SyncCardData{CardID: "sync-oob", Title: "Card", Body: "c"})
	taskData, _ := json.Marshal(models.SyncTaskData{
		Title:      "Out-of-order task",
		CardPKUUID: "c-oob-1",
		Status:     "todo",
	})
	resp := pushChanges(t, s, router, []models.SyncChange{
		{Collection: services.SyncCollectionTasks, RowUUID: "t-oob-1", Op: services.SyncOpUpsert, BaseVersion: 0, Data: taskData},
		{Collection: services.SyncCollectionCards, RowUUID: "c-oob-1", Op: services.SyncOpUpsert, BaseVersion: 0, Data: cardData},
	})
	var cardResult *models.SyncPushResult
	var taskResult *models.SyncPushResult
	for i := range resp.Results {
		switch resp.Results[i].RowUUID {
		case "c-oob-1":
			cardResult = &resp.Results[i]
		case "t-oob-1":
			taskResult = &resp.Results[i]
		}
	}
	if cardResult == nil || taskResult == nil {
		t.Fatalf("missing results: %+v", resp.Results)
	}
	if cardResult.Status != services.SyncStatusApplied || taskResult.Status != services.SyncStatusApplied {
		t.Fatalf("expected both applied: %+v", resp.Results)
	}
	var cardPK int
	if err := s.GetDB().QueryRow(`SELECT card_pk FROM tasks WHERE sync_uuid = 't-oob-1'`).Scan(&cardPK); err != nil {
		t.Fatalf("read task card_pk: %v", err)
	}
	if cardPK != *cardResult.ServerID {
		t.Errorf("task.card_pk = %d, want card id %d (topological resolution)", cardPK, *cardResult.ServerID)
	}
}

// TestSyncPushTaskTitleTagDerivation asserts a task title's #tag creates a
// real tag (emitted) and task_tags junction rows, while task_tags never
// appears in the feed.
func TestSyncPushTaskTitleTagDerivation(t *testing.T) {
	s := NewHandler()
	defer tests.Teardown()
	router := newSyncRouter(s)

	taskData, _ := json.Marshal(models.SyncTaskData{Title: "Buy milk #grocery", Status: "todo"})
	resp := pushChanges(t, s, router, []models.SyncChange{
		{Collection: services.SyncCollectionTasks, RowUUID: "t-tag-1", Op: services.SyncOpUpsert, BaseVersion: 0, Data: taskData},
	})
	if resp.Results[0].Status != services.SyncStatusApplied {
		t.Fatalf("task: %+v", resp.Results[0])
	}

	var junctionCount int
	if err := s.GetDB().QueryRow(`SELECT COUNT(*) FROM task_tags tt JOIN tasks t ON tt.task_pk = t.id WHERE t.sync_uuid = 't-tag-1'`).Scan(&junctionCount); err != nil {
		t.Fatalf("count task_tags: %v", err)
	}
	if junctionCount != 1 {
		t.Errorf("expected 1 derived task_tags row, got %d", junctionCount)
	}

	var tagCount int
	if err := s.GetDB().QueryRow(`SELECT COUNT(*) FROM tags WHERE user_id = 1 AND name = 'grocery' AND is_deleted = FALSE`).Scan(&tagCount); err != nil {
		t.Fatal(err)
	}
	if tagCount != 1 {
		t.Errorf("expected derived tag 'grocery', got %d", tagCount)
	}
}

// TestSyncPushSchemaValidationRejectsInvalidStructuredData verifies the sync
// push path enforces required schema fields exactly like the REST save path
// (bead Zettelgarden-s2l): a pushed card with a schema but missing/empty
// required structured_data is refused with 400 + message, and a valid push
// lands.
func TestSyncPushSchemaValidationRejectsInvalidStructuredData(t *testing.T) {
	s := NewHandler()
	defer tests.Teardown()
	router := newSyncRouter(s)

	// Create a schema with one required text field (same helper as cards tests).
	fields := []models.FieldDefinition{
		{Name: "required_field", Type: "text", Required: true},
	}
	schemaID := createTestSchema(s, t, 1, "Sync Required Schema", fields)

	validSD, _ := json.Marshal(map[string]interface{}{"required_field": "filled"})
	valid := models.SyncChange{
		Collection: services.SyncCollectionCards, RowUUID: "sync-schema-valid",
		Op: services.SyncOpUpsert, BaseVersion: 0,
		Data: mustJSON(t, models.SyncCardData{
			CardID: "SYNCSCHEMA1", Title: "Valid", Body: "ok",
			CardSchemaID: &schemaID, StructuredData: ptrRaw(validSD),
		}),
	}
	resp := decodeSyncResp[models.SyncPushResponse](t, syncRequest(t, s, router, "POST", "/api/sync/push", models.SyncPushRequest{Changes: []models.SyncChange{valid}, DeviceID: "test-device"}))
	if len(resp.Results) != 1 || resp.Results[0].Status != services.SyncStatusApplied {
		t.Fatalf("expected applied for valid schema card, got %+v", resp.Results)
	}

	// Now push a card missing the required field -> 400 with a message.
	missing := models.SyncChange{
		Collection: services.SyncCollectionCards, RowUUID: "sync-schema-missing",
		Op: services.SyncOpUpsert, BaseVersion: 0,
		Data: mustJSON(t, models.SyncCardData{
			CardID: "SYNCSCHEMA2", Title: "Missing", Body: "bad",
			CardSchemaID: &schemaID, StructuredData: ptrRaw(json.RawMessage(`{"other":"x"}`)),
		}),
	}
	rr := syncRequest(t, s, router, "POST", "/api/sync/push", models.SyncPushRequest{Changes: []models.SyncChange{missing}, DeviceID: "test-device"})
	if rr.Code != http.StatusBadRequest {
		t.Fatalf("expected 400 for missing required field via sync, got %d: %s", rr.Code, rr.Body.String())
	}
	if !strings.Contains(rr.Body.String(), "required field 'required_field'") {
		t.Errorf("expected message naming the field, got: %s", rr.Body.String())
	}

	// Empty (whitespace-only) required value is refused too.
	empty := models.SyncChange{
		Collection: services.SyncCollectionCards, RowUUID: "sync-schema-empty",
		Op: services.SyncOpUpsert, BaseVersion: 0,
		Data: mustJSON(t, models.SyncCardData{
			CardID: "SYNCSCHEMA3", Title: "Empty", Body: "bad",
			CardSchemaID: &schemaID, StructuredData: ptrRaw(json.RawMessage(`{"required_field":"   "}`)),
		}),
	}
	rr = syncRequest(t, s, router, "POST", "/api/sync/push", models.SyncPushRequest{Changes: []models.SyncChange{empty}, DeviceID: "test-device"})
	if rr.Code != http.StatusBadRequest {
		t.Fatalf("expected 400 for empty required field via sync, got %d: %s", rr.Code, rr.Body.String())
	}
}

// TestSyncPushRejectsStructuredDataWithoutSchema mirrors the REST guard
// "structured_data requires schema_id to be specified" (handlers/cards.go) on
// the sync push path (bead Zettelgarden-a1u): same user intent must produce the
// same outcome by transport.
func TestSyncPushRejectsStructuredDataWithoutSchema(t *testing.T) {
	s := NewHandler()
	defer tests.Teardown()
	router := newSyncRouter(s)

	// NOTE: the success case runs FIRST — the handler rolls back the shared
	// test transaction when a push is rejected, so anything after a 400 would
	// hit "transaction has already been committed or rolled back".

	// Empty structured_data with no schema_id stays valid (matches REST).
	emptyData := models.SyncChange{
		Collection: services.SyncCollectionCards, RowUUID: "sync-noschema-2",
		Op: services.SyncOpUpsert, BaseVersion: 0,
		Data: mustJSON(t, models.SyncCardData{
			CardID: "NOSCHEMA2", Title: "No schema empty", Body: "x",
			StructuredData: ptrRaw(json.RawMessage(`{}`)),
		}),
	}
	resp := decodeSyncResp[models.SyncPushResponse](t, syncRequest(t, s, router, "POST", "/api/sync/push", models.SyncPushRequest{Changes: []models.SyncChange{emptyData}, DeviceID: "test-device"}))
	if len(resp.Results) != 1 || resp.Results[0].Status != services.SyncStatusApplied {
		t.Fatalf("expected applied for empty structured_data without schema, got %+v", resp.Results)
	}

	// Non-empty structured_data with no schema_id -> 400 + message.
	noSchema := models.SyncChange{
		Collection: services.SyncCollectionCards, RowUUID: "sync-noschema-1",
		Op: services.SyncOpUpsert, BaseVersion: 0,
		Data: mustJSON(t, models.SyncCardData{
			CardID: "NOSCHEMA1", Title: "No schema", Body: "x",
			StructuredData: ptrRaw(json.RawMessage(`{"a":"b"}`)),
		}),
	}
	rr := syncRequest(t, s, router, "POST", "/api/sync/push", models.SyncPushRequest{Changes: []models.SyncChange{noSchema}, DeviceID: "test-device"})
	if rr.Code != http.StatusBadRequest {
		t.Fatalf("expected 400 for structured_data without schema_id via sync, got %d: %s", rr.Code, rr.Body.String())
	}
	if !strings.Contains(rr.Body.String(), "structured_data requires schema_id") {
		t.Errorf("expected REST-mirroring message, got: %s", rr.Body.String())
	}
}

func mustJSON[T any](t *testing.T, v T) json.RawMessage {
	t.Helper()
	b, err := json.Marshal(v)
	if err != nil {
		t.Fatalf("marshal: %v", err)
	}
	return b
}

func ptrRaw(b json.RawMessage) *json.RawMessage { return &b }
