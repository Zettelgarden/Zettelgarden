-- Phase 1 spike mini-schema (NOT the Phase 2 consolidated schema).
--
-- Purpose: a minimal SQLite schema covering the cards read+write path and its
-- immediate side-effect tables, so the end-to-end cards flow (CreateCard +
-- GetFullCard + tag/backlink/audit side effects) can be exercised against an
-- in-memory SQLite DB. This validates driver, pragmas, RETURNING, JSONB-as-TEXT
-- scanning, time.Time scanning from DATETIME columns (D5), and the first NOW()
-- -> app-side time translations — before the full Phase 2 consolidated schema
-- exists.
--
-- Derived by hand from the Postgres migration files (not from pg_dump). The
-- Phase 2 consolidated schema (schema/sqlite/schema.sqlite.sql) will supersede
-- this once a live `pg_dump --schema-only` is available; until then this file
-- is a throwaway spike artifact kept under testdata to make that clear.
--
-- SQLite-specific adaptations applied here (these are the same rules Phase 2
-- will apply to every table):
--   * SERIAL -> INTEGER PRIMARY KEY AUTOINCREMENT
--   * TIMESTAMP[TZ] -> DATETIME (NOT TEXT — modernc needs the declared type to
--     return time.Time; see migration design doc D5)
--   * JSONB -> TEXT (modernc scans it into *json.RawMessage / []byte fine)
--   * DEFAULT NOW() / DEFAULT CURRENT_TIMESTAMP -> DEFAULT (datetime('now'))
--   * BOOLEAN kept as-is (SQLite affinity treats it as INTEGER 0/1)
--   * FK to schema_definitions dropped (that table is not in this mini-schema)

CREATE TABLE IF NOT EXISTS users (
    id INTEGER PRIMARY KEY AUTOINCREMENT,
    username TEXT,
    email TEXT,
    password TEXT,
    created_at DATETIME DEFAULT (datetime('now')),
    updated_at DATETIME DEFAULT (datetime('now'))
);

CREATE TABLE IF NOT EXISTS cards (
    id INTEGER PRIMARY KEY AUTOINCREMENT,
    card_id TEXT,
    user_id INT,
    title TEXT,
    body TEXT,
    link TEXT,
    is_deleted BOOLEAN DEFAULT FALSE,
    created_at DATETIME DEFAULT (datetime('now')),
    updated_at DATETIME DEFAULT (datetime('now')),
    parent_id INT,
    is_literature_card BOOLEAN DEFAULT false,
    is_flashcard BOOLEAN DEFAULT false,
    flashcard_state TEXT,
    flashcard_reps INT DEFAULT 0,
    flashcard_lapses INT DEFAULT 0,
    flashcard_last_review DATETIME,
    flashcard_due DATETIME,
    flashcard_stability REAL DEFAULT 0,
    flashcard_difficulty REAL DEFAULT 0,
    card_schema_id INT,
    structured_data TEXT,
    created_by_agent_id INT NULL REFERENCES users(id) ON DELETE SET NULL,
    FOREIGN KEY (user_id) REFERENCES users(id)
);

CREATE INDEX IF NOT EXISTS idx_cards_user_created ON cards(user_id, created_at);
CREATE INDEX IF NOT EXISTS idx_cards_card_schema_id ON cards(card_schema_id);
CREATE INDEX IF NOT EXISTS idx_cards_created_by_agent ON cards(created_by_agent_id) WHERE created_by_agent_id IS NOT NULL;

CREATE TABLE IF NOT EXISTS tags (
    id INTEGER PRIMARY KEY AUTOINCREMENT,
    name TEXT NOT NULL,
    color TEXT,
    user_id INT,
    is_deleted BOOLEAN DEFAULT FALSE,
    created_at DATETIME DEFAULT (datetime('now')),
    updated_at DATETIME DEFAULT (datetime('now'))
);

CREATE TABLE IF NOT EXISTS card_tags (
    card_pk INT,
    tag_id INT,
    PRIMARY KEY (card_pk, tag_id),
    FOREIGN KEY (card_pk) REFERENCES cards(id) ON DELETE CASCADE,
    FOREIGN KEY (tag_id) REFERENCES tags(id) ON DELETE CASCADE
);

-- backlinks mirrors the Postgres shape (both text and integer link columns).
CREATE TABLE IF NOT EXISTS backlinks (
    source_id TEXT,
    target_id TEXT,
    created_at DATETIME DEFAULT (datetime('now')),
    updated_at DATETIME DEFAULT (datetime('now')),
    source_id_int INT,
    target_id_int INT
);

CREATE TABLE IF NOT EXISTS audit_events (
    id INTEGER PRIMARY KEY AUTOINCREMENT,
    user_id INTEGER NOT NULL,
    entity_id INTEGER NOT NULL,
    entity_type TEXT NOT NULL,
    action TEXT NOT NULL,
    details TEXT NOT NULL,
    created_at DATETIME NOT NULL DEFAULT (datetime('now'))
);
