package models

import (
	"encoding/json"
	"time"
)

// Sync API types (epic Zettelgarden-v5b, Phase 0b — issue tsv). The sync API
// is the server-as-sync-hub surface for the local-first clients: a snapshot
// bootstrap, an incremental changes feed over sync_log, and a transactional
// batch push with optimistic concurrency (last-write-wins on version).

// SyncRow is one row in a snapshot or changes-feed payload. Data is the raw
// row columns (see the per-collection column lists in handlers/sync.go);
// Op is "upsert" or "delete" (delete rows may carry no Data).
type SyncRow struct {
	RowUUID string          `json:"row_uuid"`
	Version int             `json:"version"`
	Op      string          `json:"op"`
	Data    json.RawMessage `json:"data,omitempty"`
}

// SyncSnapshotResponse is the full bootstrap payload for one user.
// Cursor is the sync_log high-water mark captured in the same read
// transaction as the snapshot: the client stores it as `since` for the first
// incremental pull, so no change between snapshot and cursor is missed.
type SyncSnapshotResponse struct {
	Cursor      int64                `json:"cursor"`
	Collections map[string][]SyncRow `json:"collections"`
}

// SyncChangesResponse is an incremental feed page.
type SyncChangesResponse struct {
	Cursor int64      `json:"cursor"`
	Rows   []SyncRow  `json:"rows"`
	HasMore bool      `json:"has_more"`
}

// SyncChange is one item in a push batch. RowUUID is the idempotency key
// (a retried push replays the same op); BaseVersion is the version the client
// wrote from (the concurrency check). Data holds a per-collection payload.
type SyncChange struct {
	Collection  string          `json:"collection"`
	RowUUID     string          `json:"row_uuid"`
	Op          string          `json:"op"` // "upsert" | "delete"
	BaseVersion int             `json:"base_version"`
	Data        json.RawMessage `json:"data"`
}

// SyncPushRequest is the push batch body.
type SyncPushRequest struct {
	Changes  []SyncChange `json:"changes"`
	DeviceID string       `json:"device_id"`
}

// SyncPushResult reports the server's disposition of one pushed change.
//   - Status "applied": the row was created or updated; ServerVersion is the
//     new server version, ServerID the int PK the client should adopt.
//   - Status "conflict": the client's base_version was stale; the server kept
//     its own row (LWW); Data carries the current server row so the client can
//     reconcile, and LostEdit is reported in the batch totals.
//   - Status "merged": a same-named tag converged onto an existing server row;
//     MappedToRowUUID is the surviving row's sync_uuid and the client must
//     rewrite its local tag row to that uuid.
//   - Status "ignored": no-op (e.g. delete of a row that doesn't exist).
type SyncPushResult struct {
	RowUUID        string          `json:"row_uuid"`
	Status         string          `json:"status"`
	ServerID       *int            `json:"server_id,omitempty"`
	ServerVersion  int             `json:"server_version"`
	MappedToRowUUID string         `json:"mapped_to_row_uuid,omitempty"`
	Data           json.RawMessage `json:"data,omitempty"`
}

// SyncPushResponse is the push batch outcome.
type SyncPushResponse struct {
	Results   []SyncPushResult `json:"results"`
	Cursor    int64            `json:"cursor"`
	LostEdits int              `json:"lost_edits"`
}

// Per-collection push payloads. The client sends exactly these fields; the
// server ignores anything else (no mass assignment of server-internal
// columns). FK fields referencing other syncable rows arrive EITHER as a
// server int (CardPK / ParentTaskID) OR as a sync_uuid reference
// (CardPKUUID / ParentTaskUUID) for offline-created rows resolved in-batch.

type SyncCardData struct {
	CardID         string          `json:"card_id"`
	Title          string          `json:"title"`
	Body           string          `json:"body"`
	Link           string          `json:"link"`
	IsDeleted      bool            `json:"is_deleted"`
	CardSchemaID   *int            `json:"card_schema_id,omitempty"`
	StructuredData *json.RawMessage `json:"structured_data,omitempty"`
}

type SyncTaskData struct {
	CardPK         *int       `json:"card_pk,omitempty"`
	CardPKUUID     string     `json:"card_pk_uuid,omitempty"`
	ParentTaskID   *int       `json:"parent_task_id,omitempty"`
	ParentTaskUUID string     `json:"parent_task_uuid,omitempty"`
	Title          string     `json:"title"`
	Description    *string    `json:"description"`
	Priority       *string    `json:"priority"`
	Status         string     `json:"status"`
	IsComplete     bool       `json:"is_complete"`
	IsDeleted      bool       `json:"is_deleted"`
	ScheduledDate  *time.Time `json:"scheduled_date"`
	DueDate        *time.Time `json:"due_date"`
	CompletedAt    *time.Time `json:"completed_at"`
	ReminderTime   *time.Time `json:"reminder_time"`
	ReminderSent   bool       `json:"reminder_sent"`
	SortOrder      *int       `json:"sort_order"`
}

type SyncTagData struct {
	Name      string `json:"name"`
	Color     string `json:"color"`
	IsDeleted bool   `json:"is_deleted"`
}
