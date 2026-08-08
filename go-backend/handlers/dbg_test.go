package handlers

import (
	"testing"
	"go-backend/models"
	"go-backend/services"
	"go-backend/tests"
)

func TestDbgParentage(t *testing.T) {
	s := NewHandler()
	defer tests.Teardown()
	router := newSyncRouter(s)

	parent := syncCardChange("c-par-1", "sync-parent", "Parent", 0)
	resp := pushChanges(t, s, router, []models.SyncChange{parent})
	t.Logf("parent push result: %+v", resp.Results[0])

	var cardID string
	var id int
	err := s.GetDB().QueryRow(`SELECT id, card_id FROM cards WHERE sync_uuid='c-par-1'`).Scan(&id, &cardID)
	t.Logf("parent row: id=%d card_id=%q err=%v", id, cardID, err)

	child := syncCardChange("c-chi-1", "sync-parent/sync-child", "Child", 0)
	resp2 := pushChanges(t, s, router, []models.SyncChange{child})
	t.Logf("child push result: %+v", resp2.Results[0])

	var parentID int
	err = s.GetDB().QueryRow(`SELECT parent_id FROM cards WHERE sync_uuid='c-chi-1'`).Scan(&parentID)
	t.Logf("child parent_id=%d err=%v; services.SyncStatusApplied=%s", parentID, err, services.SyncStatusApplied)
}
