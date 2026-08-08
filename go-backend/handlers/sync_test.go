package handlers

import (
	"bytes"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"strconv"
	"testing"

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
	token, _ := tests.GenerateTestJWT(1)
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
	parent := syncCardChange("c-par-1", "sync-parent", "Parent", 0)
	pushChanges(t, s, router, []models.SyncChange{parent})

	// Child card: card_id encodes the hierarchy; server derives parent_id.
	child := syncCardChange("c-chi-1", "sync-parent/sync-child", "Child", 0)
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

