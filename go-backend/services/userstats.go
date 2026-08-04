package services

import (
	"fmt"
	"log"

	"go-backend/models"
)

// This file replaces the Postgres user_stats triggers in
// schema/0093-user-stats-triggers.sql. Those triggers are dropped for BOTH
// drivers by migration 0145 (migration design doc Phase 5). Per the Phase 5
// decision, Go maintains user_stats on both Postgres and SQLite — there is no
// driver gating — so Postgres and SQLite share one code path and there is no
// double-counting during the cutover window (the PG triggers are gone before
// this code first runs).
//
// Reach: the Go server is the only writer to the source tables (cards, tasks,
// files, llm_query_log, revenue). The standalone cmd/* binaries and scripts/*
// do not insert/delete/update any of them (verified 2026-07-25), so wiring
// these calls at the services-layer write sites reaches every write path — no
// need for a shared package that cmd binaries import (the Phase 4 caveat does
// not apply here).
//
// All functions are best-effort: they log on error and never fail the caller's
// request (same pattern as CreateAuditEvent). user_stats is a cached aggregate
// read only by the admin user list (handlers/users.go) and admin stats; a
// transiently stale counter is a minor display issue, never data loss.

// Integer counter columns in user_stats maintained as +/-1 deltas.
const (
	userStatCardCount = "card_count"
	userStatTaskCount = "task_count"
	userStatFileCount = "file_count"
)

// bumpIntCounter applies a +1/-1 delta to a user_stats integer counter via an
// upsert keyed on user_id (user_stats.user_id is the PK on both drivers),
// creating the row lazily if it does not exist. Decrements floor at 0 to match
// the old trigger's GREATEST(..., 0). CASE WHEN is used for the floor instead
// of GREATEST/MAX because Postgres spells the scalar function GREATEST and
// SQLite spells it MAX — CASE WHEN is identical on both. delta and col are
// package constants, never caller input, so they are safe to interpolate.
func bumpIntCounter(db models.Database, userID int, col string, delta int) {
	seed := delta
	if delta < 0 {
		seed = 0 // never seed a negative count for a previously-missing row
	}
	q := fmt.Sprintf(`
		INSERT INTO user_stats (user_id, %s, updated_at)
		VALUES ($1, %d, CURRENT_TIMESTAMP)
		ON CONFLICT (user_id) DO UPDATE SET
			%s = CASE WHEN user_stats.%s + %d < 0 THEN 0 ELSE user_stats.%s + %d END,
			updated_at = CURRENT_TIMESTAMP`,
		col, seed, col, col, delta, col, delta)
	if _, err := db.Exec(q, userID); err != nil {
		log.Printf("user_stats %s bump (delta %d) failed for user %d: %v", col, delta, userID, err)
	}
}

// addAmount adds a positive amount to a user_stats column (revenue_cents,
// llm_cost_usd) via an upsert. $2 is referenced twice (seed value + delta) on
// purpose; modernc.org/sqlite binds both numbered params to the same positional
// arg. No flooring — amounts are strictly additive.
func addAmount(db models.Database, userID int, col string, amount any) {
	q := fmt.Sprintf(`
		INSERT INTO user_stats (user_id, %s, updated_at)
		VALUES ($1, $2, CURRENT_TIMESTAMP)
		ON CONFLICT (user_id) DO UPDATE SET
			%s = user_stats.%s + $2,
			updated_at = CURRENT_TIMESTAMP`,
		col, col, col)
	if _, err := db.Exec(q, userID, amount); err != nil {
		log.Printf("user_stats %s add failed for user %d: %v", col, userID, err)
	}
}

// --- Cards ------------------------------------------------------------------

// IncrementUserCardCount bumps user_stats.card_count by 1 (card created).
func IncrementUserCardCount(db models.Database, userID int) {
	bumpIntCounter(db, userID, userStatCardCount, +1)
}

// DecrementUserCardCount reduces user_stats.card_count by 1, floored at 0
// (card soft-deleted).
func DecrementUserCardCount(db models.Database, userID int) {
	bumpIntCounter(db, userID, userStatCardCount, -1)
}

// --- Tasks ------------------------------------------------------------------

// IncrementUserTaskCount bumps user_stats.task_count by 1 (task created).
func IncrementUserTaskCount(db models.Database, userID int) {
	bumpIntCounter(db, userID, userStatTaskCount, +1)
}

// DecrementUserTaskCount reduces user_stats.task_count by 1, floored at 0
// (task soft-deleted).
func DecrementUserTaskCount(db models.Database, userID int) {
	bumpIntCounter(db, userID, userStatTaskCount, -1)
}

// --- Files ------------------------------------------------------------------

// IncrementUserFileCount bumps user_stats.file_count by 1 (file uploaded).
//
// NOTE: there is intentionally no DecrementUserFileCount. The old PG trigger
// fired AFTER hard DELETE, but the Go server only ever soft-deletes files
// (UPDATE files SET is_deleted=true), so file_count never decremented in
// practice — a pre-existing overcount tracked in beads issue Zettelgarden-y6s.
func IncrementUserFileCount(db models.Database, userID int) {
	bumpIntCounter(db, userID, userStatFileCount, +1)
}

// --- Cost & revenue ---------------------------------------------------------

// AddUserLLMCost accumulates an LLM query cost (USD) into user_stats.llm_cost_usd.
func AddUserLLMCost(db models.Database, userID int, costUSD float64) {
	addAmount(db, userID, "llm_cost_usd", costUSD)
}

// AddUserRevenue accumulates a payment amount (cents) into user_stats.revenue_cents.
func AddUserRevenue(db models.Database, userID int, amountCents int64) {
	addAmount(db, userID, "revenue_cents", amountCents)
}
