package services

import (
	"database/sql"
	"encoding/json"
	"fmt"
	"log"
	"reflect"
	"regexp"
	"time"

	"go-backend/models"
	"go-backend/services/backlink"
)

// Sync push application (epic Zettelgarden-v5b, Phase 0b — issue tsv).
//
// ApplySyncPush applies a client's outbox batch inside ONE transaction:
//   - idempotency: row_uuid is the key — a retried op finds the row it
//     already created and becomes an update/no-op, never a duplicate;
//   - optimistic concurrency: the client's base_version is compared to the
//     server version; a stale base (server is ahead) is LWW-resolved by
//     KEEPING the server row (conflict result, current row returned) and
//     counting a lost edit;
//   - FK resolution: offline-created rows referenced by other offline-created
//     rows in the same batch are resolved via row_uuid -> server PK;
//   - tags merge BY NAME (CreateTag semantics: find (user_id, name) including
//     soft-deleted rows and reuse), because tags(name) has no UNIQUE
//     constraint and uuid-upsert alone would silently create duplicate tags;
//   - card parentage is server-authoritative: parent_id is derived from the
//     card_id prefix (DiscoverParentId), batch-aware so an offline parent in
//     the same push is found; a missing parent falls back to self-parent,
//     matching CreateCard/UpdateCard.
//
// Audit events are intentionally NOT written for sync-pushed changes (the
// audit trail is a UI feature; a sync engine would flood it). user_stats
// counters ARE maintained (create/delete), and card body tags are re-derived
// via AddTagsFromCard so the derived card_tags junctions stay correct.

// Sync conflict statuses.
const (
	SyncStatusApplied  = "applied"
	SyncStatusConflict = "conflict"
	SyncStatusMerged   = "merged"
	SyncStatusIgnored  = "ignored"
)

// syncCardIDSpace matches any whitespace run in a card_id. Card IDs are
// stored whitespace-stripped (UpdateCard semantics); the retry matcher must
// normalize identically before comparing a re-pushed payload to the row.
var syncCardIDSpace = regexp.MustCompile(`\s+`)

func normalizeCardID(s string) string {
	return syncCardIDSpace.ReplaceAllString(s, "")
}

type pushContext struct {
	tx             *sql.Tx
	userID         int
	deviceID       string
	uuidToID       map[string]int  // row_uuid -> server PK for rows created in this batch
	cardIDToID     map[string]int  // card_id -> server PK for cards in this batch (parent resolution)
	deletedInBatch map[string]bool // row_uuids soft-deleted earlier in THIS batch (delete-then-recreate)
	lost           int
	results        []models.SyncPushResult
}

// ApplySyncPush applies a push batch transactionally and returns per-change
// results, the sync_log cursor visible to the transaction, and the count of
// discarded (lost) edits.
func ApplySyncPush(tx *sql.Tx, userID int, req *models.SyncPushRequest) ([]models.SyncPushResult, int64, int, error) {
	ctx := &pushContext{
		tx:             tx,
		userID:         userID,
		deviceID:       req.DeviceID,
		uuidToID:       map[string]int{},
		cardIDToID:     map[string]int{},
		deletedInBatch: map[string]bool{},
	}
	// Apply in collection order (cards → tasks → tags) regardless of client
	// send order, so FK references from a task to a card created later in the
	// batch still resolve. Stable within a collection (client order kept).
	ordered := make([]models.SyncChange, 0, len(req.Changes))
	for _, c := range []string{SyncCollectionCards, SyncCollectionTasks, SyncCollectionTags} {
		for _, ch := range req.Changes {
			if ch.Collection == c {
				ordered = append(ordered, ch)
			}
		}
	}
	for _, ch := range ordered {
		switch ch.Collection {
		case SyncCollectionCards:
			if ch.Op == SyncOpDelete {
				ctx.applyCardDelete(ch)
			} else {
				ctx.applyCardUpsert(ch)
			}
		case SyncCollectionTasks:
			if ch.Op == SyncOpDelete {
				ctx.applyTaskDelete(ch)
			} else {
				ctx.applyTaskUpsert(ch)
			}
		case SyncCollectionTags:
			if ch.Op == SyncOpDelete {
				ctx.applyTagDelete(ch)
			} else {
				ctx.applyTagUpsert(ch)
			}
		default:
			ctx.results = append(ctx.results, models.SyncPushResult{
				RowUUID: ch.RowUUID, Status: SyncStatusIgnored,
			})
		}
	}
	var cursor int64
	if err := tx.QueryRow(`SELECT COALESCE(MAX(id), 0) FROM sync_log`).Scan(&cursor); err != nil {
		return nil, 0, 0, fmt.Errorf("read sync cursor: %w", err)
	}
	return ctx.results, cursor, ctx.lost, nil
}

// ---- helpers ---------------------------------------------------------------

func (c *pushContext) emit(collection, rowUUID, op string, version int) {
	if err := EmitChange(c.tx, c.userID, collection, rowUUID, op, version); err != nil {
		// The row write succeeded; a failed emit is logged by EmitChange. The
		// mutation is invisible to sync until the next change — surface loudly.
		log.Printf("sync push: %s %s %s v%d emit failed: %v", collection, op, rowUUID, version, err)
	}
}

func (c *pushContext) applied(ch models.SyncChange, serverID int, version int) {
	c.results = append(c.results, models.SyncPushResult{
		RowUUID: ch.RowUUID, Status: SyncStatusApplied,
		ServerID: &serverID, ServerVersion: version,
	})
}

func (c *pushContext) conflict(ch models.SyncChange, serverID, version int, rowJSON []byte) {
	c.lost++
	c.results = append(c.results, models.SyncPushResult{
		RowUUID: ch.RowUUID, Status: SyncStatusConflict,
		ServerID: &serverID, ServerVersion: version, Data: rowJSON,
	})
}

// retryOrConflict resolves a stale base_version. A stale base usually means a
// genuinely concurrent edit won the LWW race (conflict + lost edit), but it
// also happens when the client re-pushes the SAME outbox entry whose response
// was lost on the wire: the server already applied it and the row advanced by
// exactly one version to the very payload being re-pushed. That case must be
// reported applied with no lost edit, or ordinary network flakiness would
// surface spurious lost-edit counts.
func (c *pushContext) retryOrConflict(collection string, ch models.SyncChange, serverID, version int) {
	if c.retryMatchesCurrent(collection, ch, serverID, version) {
		c.applied(ch, serverID, version)
		return
	}
	c.conflict(ch, serverID, version, c.currentRow(collection, serverID))
}

// retryMatchesCurrent reports whether a stale base_version is an idempotent
// retry of an op the server already applied. Two conditions must BOTH hold:
//
//   - the server row advanced by exactly the one version our apply would have
//     produced (version == base+1), and
//   - the current row is exactly what our apply would have left: for an
//     upsert the re-pushed payload matches the stored columns; for a delete
//     the row is already soft-deleted.
//
// A genuinely stale concurrent edit (different content, or a row that jumped
// more than one version) fails one of these and still conflicts.
func (c *pushContext) retryMatchesCurrent(collection string, ch models.SyncChange, serverID, version int) bool {
	if version != ch.BaseVersion+1 {
		return false
	}
	switch collection {
	case SyncCollectionCards:
		return c.cardRowMatchesPush(ch, serverID)
	case SyncCollectionTasks:
		return c.taskRowMatchesPush(ch, serverID)
	case SyncCollectionTags:
		return c.tagRowMatchesPush(ch, serverID)
	}
	return false
}

// cardRowMatchesPush compares a re-pushed card change against the current row.
func (c *pushContext) cardRowMatchesPush(ch models.SyncChange, serverID int) bool {
	var row struct {
		CardID         sql.NullString
		Title          string
		Body           string
		Link           string
		IsDeleted      bool
		CardSchemaID   *int
		StructuredData sql.NullString
	}
	if err := c.tx.QueryRow(`SELECT card_id, title, body, link, is_deleted, card_schema_id, structured_data FROM cards WHERE id = $1 AND user_id = $2`, serverID, c.userID).
		Scan(&row.CardID, &row.Title, &row.Body, &row.Link, &row.IsDeleted, &row.CardSchemaID, &row.StructuredData); err != nil {
		return false
	}
	if ch.Op == SyncOpDelete {
		// Our delete applied earlier: the row is already soft-deleted. The
		// delete payload carries no meaningful data to compare.
		return row.IsDeleted
	}
	var pushed models.SyncCardData
	if err := json.Unmarshal(ch.Data, &pushed); err != nil {
		return false
	}
	// applyCardUpsert normalizes card_id before storing; normalize identically.
	pushed.CardID = normalizeCardID(pushed.CardID)
	if pushed.CardID != row.CardID.String || pushed.Title != row.Title || pushed.Body != row.Body || pushed.Link != row.Link || pushed.IsDeleted != row.IsDeleted {
		return false
	}
	if !equalIntPtr(pushed.CardSchemaID, row.CardSchemaID) {
		return false
	}
	return structuredDataMatches(pushed.StructuredData, row.StructuredData.String)
}

// taskRowMatchesPush compares a re-pushed task change against the current row,
// resolving the client's uuid FK references the same way applyTaskUpsert does.
func (c *pushContext) taskRowMatchesPush(ch models.SyncChange, serverID int) bool {
	var row struct {
		CardPK        sql.NullInt64
		Title         string
		Description   *string
		Priority      *string
		Status        string
		IsComplete    bool
		IsDeleted     bool
		ScheduledDate *time.Time
		DueDate       *time.Time
		CompletedAt   *time.Time
		ReminderTime  *time.Time
		ReminderSent  bool
		ParentTaskID  *int
		SortOrder     *int
	}
	if err := c.tx.QueryRow(`SELECT card_pk, title, description, priority, status, is_complete, is_deleted, scheduled_date, due_date, completed_at, reminder_time, reminder_sent, parent_task_id, sort_order FROM tasks WHERE id = $1 AND user_id = $2`, serverID, c.userID).
		Scan(&row.CardPK, &row.Title, &row.Description, &row.Priority, &row.Status, &row.IsComplete, &row.IsDeleted, &row.ScheduledDate, &row.DueDate, &row.CompletedAt, &row.ReminderTime, &row.ReminderSent, &row.ParentTaskID, &row.SortOrder); err != nil {
		return false
	}
	if ch.Op == SyncOpDelete {
		return row.IsDeleted
	}
	var pushed models.SyncTaskData
	if err := json.Unmarshal(ch.Data, &pushed); err != nil {
		return false
	}
	cardPK := 0
	if pushed.CardPK != nil {
		cardPK = *pushed.CardPK
	} else if pushed.CardPKUUID != "" {
		id, ok := c.resolveRowUUID(SyncCollectionCards, pushed.CardPKUUID)
		if !ok {
			return false
		}
		cardPK = id
	}
	storedCardPK := 0
	if row.CardPK.Valid {
		storedCardPK = int(row.CardPK.Int64)
	}
	var parentTaskID *int
	if pushed.ParentTaskID != nil {
		parentTaskID = pushed.ParentTaskID
	} else if pushed.ParentTaskUUID != "" {
		id, ok := c.resolveRowUUID(SyncCollectionTasks, pushed.ParentTaskUUID)
		if !ok {
			return false
		}
		parentTaskID = &id
	}
	if storedCardPK != cardPK || pushed.Title != row.Title || pushed.Status != row.Status ||
		pushed.IsComplete != row.IsComplete || pushed.IsDeleted != row.IsDeleted || pushed.ReminderSent != row.ReminderSent {
		return false
	}
	if !equalStrPtr(pushed.Description, row.Description) || !equalStrPtr(pushed.Priority, row.Priority) {
		return false
	}
	if !equalIntPtr(pushed.SortOrder, row.SortOrder) || !equalIntPtr(parentTaskID, row.ParentTaskID) {
		return false
	}
	return equalTimePtr(pushed.ScheduledDate, row.ScheduledDate) && equalTimePtr(pushed.DueDate, row.DueDate) &&
		equalTimePtr(pushed.CompletedAt, row.CompletedAt) && equalTimePtr(pushed.ReminderTime, row.ReminderTime)
}

// tagRowMatchesPush compares a re-pushed tag change against the current row.
func (c *pushContext) tagRowMatchesPush(ch models.SyncChange, serverID int) bool {
	var row struct {
		Name      string
		Color     string
		IsDeleted bool
	}
	if err := c.tx.QueryRow(`SELECT name, color, is_deleted FROM tags WHERE id = $1 AND user_id = $2`, serverID, c.userID).
		Scan(&row.Name, &row.Color, &row.IsDeleted); err != nil {
		return false
	}
	if ch.Op == SyncOpDelete {
		return row.IsDeleted
	}
	var pushed models.SyncTagData
	if err := json.Unmarshal(ch.Data, &pushed); err != nil {
		return false
	}
	return pushed.Name == row.Name && pushed.Color == row.Color && pushed.IsDeleted == row.IsDeleted
}

// ---- pointer comparison helpers (nil-aware, instant-based for times) -------

func equalStrPtr(a, b *string) bool {
	if a == nil || b == nil {
		return a == nil && b == nil
	}
	return *a == *b
}

func equalIntPtr(a, b *int) bool {
	if a == nil || b == nil {
		return a == nil && b == nil
	}
	return *a == *b
}

func equalTimePtr(a, b *time.Time) bool {
	if a == nil || b == nil {
		return a == nil && b == nil
	}
	return a.Equal(*b)
}

// structuredDataMatches compares a pushed structured_data payload against the
// stored column value semantically (JSON round-trip), so key order and number
// formatting differences never false-negative a retry.
func structuredDataMatches(pushed *json.RawMessage, stored string) bool {
	if pushed == nil {
		return stored == ""
	}
	if stored == "" {
		return false
	}
	var p, s any
	if err := json.Unmarshal(*pushed, &p); err != nil || json.Unmarshal([]byte(stored), &s) != nil {
		return false
	}
	return reflect.DeepEqual(p, s)
}

func (c *pushContext) merged(ch models.SyncChange, serverID int, version int, mappedTo string) {
	c.results = append(c.results, models.SyncPushResult{
		RowUUID: ch.RowUUID, Status: SyncStatusMerged,
		ServerID: &serverID, ServerVersion: version, MappedToRowUUID: mappedTo,
		Data: c.currentRow(SyncCollectionTags, serverID),
	})
}

func (c *pushContext) ignored(ch models.SyncChange) {
	c.results = append(c.results, models.SyncPushResult{
		RowUUID: ch.RowUUID, Status: SyncStatusIgnored,
	})
}

// currentRow returns the live columns of a synced row as JSON, or nil if the
// row is missing. Used for conflict responses so the client can reconcile.
func (c *pushContext) currentRow(collection string, serverID int) []byte {
	var query string
	switch collection {
	case SyncCollectionCards:
		query = `SELECT id, card_id, title, body, link, is_deleted, parent_id, created_at, updated_at, card_schema_id, structured_data, version, sync_uuid FROM cards WHERE id = $1 AND user_id = $2`
	case SyncCollectionTasks:
		query = `SELECT id, card_pk, user_id, scheduled_date, due_date, created_at, updated_at, completed_at, title, description, priority, status, is_complete, is_deleted, reminder_time, reminder_sent, parent_task_id, sort_order, version, sync_uuid FROM tasks WHERE id = $1 AND user_id = $2`
	case SyncCollectionTags:
		query = `SELECT id, name, color, user_id, is_deleted, created_at, updated_at, version, sync_uuid FROM tags WHERE id = $1 AND user_id = $2`
	default:
		return nil
	}
	rows, err := c.tx.Query(query, serverID, c.userID)
	if err != nil {
		return nil
	}
	defer rows.Close()
	out, err := RowsToJSON(rows)
	if err != nil || len(out) == 0 {
		return nil
	}
	b, err := json.Marshal(out[0])
	if err != nil {
		return nil
	}
	return b
}

// resolveRowUUID resolves a sync_uuid reference (offline-created FK target) to
// a server int PK, consulting rows created earlier in this batch first.
func (c *pushContext) resolveRowUUID(table string, rowUUID string) (int, bool) {
	if id, ok := c.uuidToID[rowUUID]; ok {
		return id, true
	}
	var id int
	var uuid string
	err := c.tx.QueryRow(fmt.Sprintf(`SELECT id, sync_uuid FROM %s WHERE sync_uuid = $1 AND user_id = $2`, table), rowUUID, c.userID).Scan(&id, &uuid)
	if err == nil {
		c.uuidToID[rowUUID] = id
		return id, true
	}
	return 0, false
}

// deriveCardParent resolves the parent int PK for a card_id, consulting the
// current batch's offline-created cards first (so an offline parent in the
// same push is found), then the server. Returns 0 when the card is a root or
// the parent cannot be found (callers fall back to self-parent, matching
// CreateCard/UpdateCard).
func (c *pushContext) deriveCardParent(cardID string) int {
	parentCardID := DiscoverParentId(cardID)
	if parentCardID == cardID || parentCardID == "" {
		return 0
	}
	if id, ok := c.cardIDToID[parentCardID]; ok {
		return id
	}
	parent, err := GetPartialCardByCardID(c.tx, c.userID, parentCardID)
	if err != nil || parent.ID == 0 {
		return 0
	}
	return parent.ID
}

// ensureSelfParent mirrors CreateCard/UpdateCard: roots and unresolved parents
// self-parent.
func (c *pushContext) ensureSelfParent(collection string, id int, cardID string) {
	parent := c.deriveCardParent(cardID)
	if parent == 0 {
		if _, err := c.tx.Exec(`UPDATE cards SET parent_id = $1 WHERE id = $1`, id); err != nil {
			log.Printf("sync push: self-parent card %d: %v", id, err)
		}
	} else if parent != id {
		if _, err := c.tx.Exec(`UPDATE cards SET parent_id = $1 WHERE id = $2`, parent, id); err != nil {
			log.Printf("sync push: parent card %d -> %d: %v", id, parent, err)
		}
	}
}

// ---- cards -----------------------------------------------------------------

func (c *pushContext) applyCardUpsert(ch models.SyncChange) {
	var data models.SyncCardData
	if err := json.Unmarshal(ch.Data, &data); err != nil {
		c.ignored(ch)
		return
	}
	// Match UpdateCard: strip all whitespace from card_id before proceeding.
	data.CardID = normalizeCardID(data.CardID)

	var id, version int
	err := c.tx.QueryRow(`SELECT id, version FROM cards WHERE sync_uuid = $1 AND user_id = $2`, ch.RowUUID, c.userID).Scan(&id, &version)
	if err == sql.ErrNoRows {
		// Create. version 1 is the base for a never-synced row; the client
		// adopts the server's version after the push.
		now := time.Now().UTC()
		res, err := c.tx.Exec(`
			INSERT INTO cards (card_id, title, body, link, user_id, parent_id, card_schema_id, structured_data, created_at, updated_at, version, sync_uuid, is_deleted)
			VALUES ($1, $2, $3, $4, $5, 0, $6, $7, $8, $8, 1, $9, $10)`,
			data.CardID, data.Title, data.Body, data.Link, c.userID,
			data.CardSchemaID, data.StructuredData, now, ch.RowUUID, data.IsDeleted)
		if err != nil {
			c.ignored(ch)
			return
		}
		newID, _ := res.LastInsertId()
		id = int(newID)
		c.ensureSelfParent(SyncCollectionCards, id, data.CardID)
		c.finishCardWrite(id)
		if !data.IsDeleted {
			IncrementUserCardCount(c.tx, c.userID)
		}
		c.uuidToID[ch.RowUUID] = id
		c.cardIDToID[data.CardID] = id
		c.emit(SyncCollectionCards, ch.RowUUID, SyncOpUpsert, 1)
		c.applied(ch, id, 1)
		return
	}
	if err != nil {
		c.ignored(ch)
		return
	}

	// Update path with optimistic concurrency. A row soft-deleted earlier in
	// THIS batch is a delete-then-recreate: the upsert resurrects it (writes
	// the recreate, no conflict), bypassing both the create-retry shortcut and
	// the stale-base check.
	if ch.BaseVersion == 0 && !c.deletedInBatch[ch.RowUUID] {
		// Idempotent retry of a create already applied: a row with our
		// row_uuid exists and the client believes it never synced — the only
		// way that happens is our own create landing earlier. No write, no
		// lost edit.
		c.applied(ch, id, version)
		return
	}
	if ch.BaseVersion < version && !c.deletedInBatch[ch.RowUUID] {
		c.retryOrConflict(SyncCollectionCards, ch, id, version)
		return
	}
	newVersion := max(version, ch.BaseVersion) + 1
	_, err = c.tx.Exec(`
		UPDATE cards SET card_id = $1, title = $2, body = $3, link = $4, card_schema_id = $5, structured_data = $6, is_deleted = $7, updated_at = $8, version = $9
		WHERE id = $10 AND user_id = $11`,
		data.CardID, data.Title, data.Body, data.Link, data.CardSchemaID, data.StructuredData, data.IsDeleted, time.Now().UTC(), newVersion, id, c.userID)
	if err != nil {
		c.ignored(ch)
		return
	}
	// A resurrection re-adds the row the batch's delete removed from the
	// user's count (matching the create path).
	if c.deletedInBatch[ch.RowUUID] && !data.IsDeleted {
		IncrementUserCardCount(c.tx, c.userID)
	}
	c.ensureSelfParent(SyncCollectionCards, id, data.CardID)
	c.finishCardWrite(id)
	c.cardIDToID[data.CardID] = id
	c.emit(SyncCollectionCards, ch.RowUUID, SyncOpUpsert, newVersion)
	c.applied(ch, id, newVersion)
}

func (c *pushContext) applyCardDelete(ch models.SyncChange) {
	var id, version int
	err := c.tx.QueryRow(`SELECT id, version FROM cards WHERE sync_uuid = $1 AND user_id = $2`, ch.RowUUID, c.userID).Scan(&id, &version)
	if err == sql.ErrNoRows {
		c.ignored(ch) // already deleted or never existed
		return
	}
	if err != nil {
		c.ignored(ch)
		return
	}
	if ch.BaseVersion < version {
		c.retryOrConflict(SyncCollectionCards, ch, id, version)
		return
	}
	newVersion := version + 1
	if _, err := c.tx.Exec(`UPDATE cards SET is_deleted = TRUE, updated_at = $1, version = $2 WHERE id = $3`, time.Now().UTC(), newVersion, id); err != nil {
		c.ignored(ch)
		return
	}
	DecrementUserCardCount(c.tx, c.userID)
	deleteCardTypesense(id)
	c.deletedInBatch[ch.RowUUID] = true
	c.emit(SyncCollectionCards, ch.RowUUID, SyncOpDelete, newVersion)
	c.applied(ch, id, newVersion)
}

// finishCardWrite re-runs the derived side effects CreateCard/UpdateCard
// perform: card_tags re-derivation from the body (emits tag changes, never
// junction entries), backlinks, and the search index.
func (c *pushContext) finishCardWrite(id int) {
	if err := AddTagsFromCard(c.tx, c.userID, id); err != nil {
		log.Printf("sync push: re-derive tags for card %d: %v", id, err)
	}
	card, err := GetFullCard(c.tx, c.userID, id)
	if err != nil {
		log.Printf("sync push: fetch card %d for side effects: %v", id, err)
		return
	}
	backlinks := ExtractBacklinks(card.Body)
	var schema *models.SchemaDefinition
	if card.SchemaID != nil {
		schema = backlink.GetSchemaByID(c.tx, c.userID, *card.SchemaID)
	}
	structuredBacklinks := ExtractBacklinksFromStructuredData(c.tx, c.userID, card.StructuredData, schema)
	UpdateBacklinks(c.tx, card.ID, append(backlinks, structuredBacklinks...))
	UpsertCardToTypesense(c.tx, card)
}

// ---- tasks -----------------------------------------------------------------

func (c *pushContext) applyTaskUpsert(ch models.SyncChange) {
	var data models.SyncTaskData
	if err := json.Unmarshal(ch.Data, &data); err != nil {
		c.ignored(ch)
		return
	}
	cardPK, parentTaskID := c.resolveTaskFKs(&data)

	var id, version int
	err := c.tx.QueryRow(`SELECT id, version FROM tasks WHERE sync_uuid = $1 AND user_id = $2`, ch.RowUUID, c.userID).Scan(&id, &version)
	if err == sql.ErrNoRows {
		now := time.Now().UTC()
		res, err := c.tx.Exec(`
			INSERT INTO tasks (card_pk, user_id, scheduled_date, due_date, created_at, updated_at, completed_at, title, description, priority, status, is_complete, is_deleted, reminder_time, reminder_sent, parent_task_id, sort_order, version, sync_uuid)
			VALUES ($1, $2, $3, $4, $5, $5, $6, $7, $8, $9, $10, $11, $12, $13, $14, $15, $16, 1, $17)`,
			cardPK, c.userID, data.ScheduledDate, data.DueDate, now, data.CompletedAt,
			data.Title, data.Description, data.Priority, data.Status, data.IsComplete,
			data.IsDeleted, data.ReminderTime, data.ReminderSent, parentTaskID, data.SortOrder, ch.RowUUID)
		if err != nil {
			c.ignored(ch)
			return
		}
		newID, _ := res.LastInsertId()
		id = int(newID)
		if !data.IsDeleted {
			IncrementUserTaskCount(c.tx, c.userID)
		}
		if err := AddTagsFromTask(c.tx, c.userID, id); err != nil {
			log.Printf("sync push: re-derive tags for task %d: %v", id, err)
		}
		c.uuidToID[ch.RowUUID] = id
		c.emit(SyncCollectionTasks, ch.RowUUID, SyncOpUpsert, 1)
		c.applied(ch, id, 1)
		return
	}
	if err != nil {
		c.ignored(ch)
		return
	}

	if ch.BaseVersion == 0 && !c.deletedInBatch[ch.RowUUID] {
		c.applied(ch, id, version) // idempotent retry of an applied create
		return
	}
	if ch.BaseVersion < version && !c.deletedInBatch[ch.RowUUID] {
		c.retryOrConflict(SyncCollectionTasks, ch, id, version)
		return
	}
	newVersion := max(version, ch.BaseVersion) + 1
	_, err = c.tx.Exec(`
		UPDATE tasks SET card_pk = $1, scheduled_date = $2, due_date = $3, updated_at = $4, completed_at = $5, title = $6, description = $7, priority = $8, status = $9, is_complete = $10, is_deleted = $11, reminder_time = $12, reminder_sent = $13, parent_task_id = $14, sort_order = $15, version = $16
		WHERE id = $17 AND user_id = $18`,
		cardPK, data.ScheduledDate, data.DueDate, time.Now().UTC(), data.CompletedAt,
		data.Title, data.Description, data.Priority, data.Status, data.IsComplete,
		data.IsDeleted, data.ReminderTime, data.ReminderSent, parentTaskID, data.SortOrder,
		newVersion, id, c.userID)
	if err != nil {
		c.ignored(ch)
		return
	}
	// A resurrection re-adds the row the batch's delete removed from the
	// user's count (matching the create path).
	if c.deletedInBatch[ch.RowUUID] && !data.IsDeleted {
		IncrementUserTaskCount(c.tx, c.userID)
	}
	if err := AddTagsFromTask(c.tx, c.userID, id); err != nil {
		log.Printf("sync push: re-derive tags for task %d: %v", id, err)
	}
	c.emit(SyncCollectionTasks, ch.RowUUID, SyncOpUpsert, newVersion)
	c.applied(ch, id, newVersion)
}

func (c *pushContext) applyTaskDelete(ch models.SyncChange) {
	var id, version int
	err := c.tx.QueryRow(`SELECT id, version FROM tasks WHERE sync_uuid = $1 AND user_id = $2`, ch.RowUUID, c.userID).Scan(&id, &version)
	if err == sql.ErrNoRows {
		c.ignored(ch)
		return
	}
	if err != nil {
		c.ignored(ch)
		return
	}
	if ch.BaseVersion < version {
		c.retryOrConflict(SyncCollectionTasks, ch, id, version)
		return
	}
	newVersion := version + 1
	if _, err := c.tx.Exec(`UPDATE tasks SET is_deleted = TRUE, updated_at = $1, version = $2 WHERE id = $3`, time.Now().UTC(), newVersion, id); err != nil {
		c.ignored(ch)
		return
	}
	DecrementUserTaskCount(c.tx, c.userID)
	c.deletedInBatch[ch.RowUUID] = true
	c.emit(SyncCollectionTasks, ch.RowUUID, SyncOpDelete, newVersion)
	c.applied(ch, id, newVersion)
}

// resolveTaskFKs turns the client's card/task references into server int PKs:
// an explicit int wins; otherwise the sync_uuid reference is resolved against
// this batch first, then the server.
func (c *pushContext) resolveTaskFKs(data *models.SyncTaskData) (cardPK int, parentTaskID *int) {
	if data.CardPK != nil {
		cardPK = *data.CardPK
	} else if data.CardPKUUID != "" {
		if id, ok := c.resolveRowUUID(SyncCollectionCards, data.CardPKUUID); ok {
			cardPK = id
		}
	}
	if data.ParentTaskID != nil {
		parentTaskID = data.ParentTaskID
	} else if data.ParentTaskUUID != "" {
		if id, ok := c.resolveRowUUID(SyncCollectionTasks, data.ParentTaskUUID); ok {
			parentTaskID = &id
		}
	}
	return cardPK, parentTaskID
}

// ---- tags ------------------------------------------------------------------

func (c *pushContext) applyTagUpsert(ch models.SyncChange) {
	var data models.SyncTagData
	if err := json.Unmarshal(ch.Data, &data); err != nil {
		c.ignored(ch)
		return
	}
	var id, version int
	var existingUUID string
	err := c.tx.QueryRow(`SELECT id, version, sync_uuid FROM tags WHERE sync_uuid = $1 AND user_id = $2`, ch.RowUUID, c.userID).Scan(&id, &version, &existingUUID)
	if err == sql.ErrNoRows {
		// Name-keyed merge: CreateTag semantics — find (user_id, name) including
		// soft-deleted rows and reuse, so two devices creating "Work" offline
		// converge to ONE server row.
		var mergedID int
		var mergedVersion int
		var mergedUUID string
		err := c.tx.QueryRow(`SELECT id, version, sync_uuid FROM tags WHERE user_id = $1 AND name = $2`, c.userID, data.Name).Scan(&mergedID, &mergedVersion, &mergedUUID)
		if err == nil {
			if ch.BaseVersion <= mergedVersion && !c.deletedInBatch[mergedUUID] {
				// The server row is as new or newer: keep it, tell the client to
				// adopt its uuid. No write, no feed entry (nothing changed).
				c.merged(ch, mergedID, mergedVersion, mergedUUID)
				return
			}
			// The client's edit is newer (or resurrects a batch-deleted row):
			// apply it onto the merged row, keeping the version monotonic.
			newVersion := max(mergedVersion, ch.BaseVersion) + 1
			if _, err := c.tx.Exec(`UPDATE tags SET name = $1, color = $2, is_deleted = $3, updated_at = $4, version = $5 WHERE id = $6`, data.Name, data.Color, data.IsDeleted, time.Now().UTC(), newVersion, mergedID); err != nil {
				c.ignored(ch)
				return
			}
			c.emit(SyncCollectionTags, mergedUUID, SyncOpUpsert, newVersion)
			c.merged(ch, mergedID, newVersion, mergedUUID)
			return
		}
		// No row by name either: create with the client's uuid.
		if _, err := c.tx.Exec(`INSERT INTO tags (name, color, user_id, is_deleted, created_at, updated_at, version, sync_uuid) VALUES ($1, $2, $3, $4, $5, $5, 1, $6)`, data.Name, data.Color, c.userID, data.IsDeleted, time.Now().UTC(), ch.RowUUID); err != nil {
			c.ignored(ch)
			return
		}
		var newID int
		if err := c.tx.QueryRow(`SELECT id FROM tags WHERE sync_uuid = $1 AND user_id = $2`, ch.RowUUID, c.userID).Scan(&newID); err != nil {
			c.ignored(ch)
			return
		}
		c.emit(SyncCollectionTags, ch.RowUUID, SyncOpUpsert, 1)
		c.applied(ch, newID, 1)
		return
	}
	if err != nil {
		c.ignored(ch)
		return
	}

	if ch.BaseVersion == 0 && !c.deletedInBatch[ch.RowUUID] {
		c.applied(ch, id, version) // idempotent retry of an applied create
		return
	}
	if ch.BaseVersion < version && !c.deletedInBatch[ch.RowUUID] {
		c.retryOrConflict(SyncCollectionTags, ch, id, version)
		return
	}
	newVersion := max(version, ch.BaseVersion) + 1
	if _, err := c.tx.Exec(`UPDATE tags SET name = $1, color = $2, is_deleted = $3, updated_at = $4, version = $5 WHERE id = $6`, data.Name, data.Color, data.IsDeleted, time.Now().UTC(), newVersion, id); err != nil {
		c.ignored(ch)
		return
	}
	c.emit(SyncCollectionTags, ch.RowUUID, SyncOpUpsert, newVersion)
	c.applied(ch, id, newVersion)
}

func (c *pushContext) applyTagDelete(ch models.SyncChange) {
	var id, version int
	var existingUUID string
	err := c.tx.QueryRow(`SELECT id, version, sync_uuid FROM tags WHERE sync_uuid = $1 AND user_id = $2`, ch.RowUUID, c.userID).Scan(&id, &version, &existingUUID)
	if err == sql.ErrNoRows {
		// The client may still hold a pre-merge uuid; fall back to a name-keyed
		// delete of the tag the client believes it is deleting.
		var data models.SyncTagData
		_ = json.Unmarshal(ch.Data, &data)
		if data.Name != "" {
			if err := c.tx.QueryRow(`SELECT id, version, sync_uuid FROM tags WHERE user_id = $1 AND name = $2`, c.userID, data.Name).Scan(&id, &version, &existingUUID); err != nil {
				c.ignored(ch)
				return
			}
			ch.RowUUID = existingUUID
		} else {
			c.ignored(ch)
			return
		}
	} else if err != nil {
		c.ignored(ch)
		return
	}
	if ch.BaseVersion < version {
		c.retryOrConflict(SyncCollectionTags, ch, id, version)
		return
	}
	newVersion := version + 1
	if _, err := c.tx.Exec(`UPDATE tags SET is_deleted = TRUE, updated_at = $1, version = $2 WHERE id = $3`, time.Now().UTC(), newVersion, id); err != nil {
		c.ignored(ch)
		return
	}
	c.deletedInBatch[existingUUID] = true
	c.emit(SyncCollectionTags, existingUUID, SyncOpDelete, newVersion)
	c.applied(ch, id, newVersion)
}
