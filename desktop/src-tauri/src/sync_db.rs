use rusqlite::{params_from_iter, Connection};
use serde_json::{json, Value};
use std::sync::Mutex;
use tauri::{AppHandle, Manager, State};

/// Local mirror database for the sync engine (epic Zettelgarden-v5b, Phase
/// 2b — issue fv3). The webview cannot run better-sqlite3, so the Rust shell
/// owns the SQLite connection and exposes the engine's storage surface as
/// generic commands: `sync_exec` (one INSERT/UPDATE/DELETE), `sync_query`
/// (one SELECT), and `sync_begin`/`sync_commit`/`sync_rollback` for the
/// engine's transactional boundaries.
///
/// The mirror schema (mirror_rows / sync_outbox / sync_meta) is created by
/// the TS adapter on first use — the Rust side stays a dumb SQL runner so the
/// schema lives in exactly one place (the engine's SqliteStorageAdapter).
///
/// Concurrency: a single connection behind a mutex. Transaction control is
/// additionally guarded by an `in_tx` flag so a misbehaving webview cannot
/// nest BEGINs (each invoke releases the mutex before the next runs, so the
/// JS adapter serializes whole transactions with its own promise queue).

pub struct SyncDb {
    conn: Mutex<Connection>,
    in_tx: Mutex<bool>,
}

fn db_path(app: &AppHandle) -> Result<std::path::PathBuf, String> {
    let dir = app
        .path()
        .app_data_dir()
        .map_err(|e| format!("app_data_dir: {e}"))?
        .join("sync");
    std::fs::create_dir_all(&dir).map_err(|e| format!("mkdir {}: {e}", dir.display()))?;
    Ok(dir.join("mirror.db"))
}

pub fn init(app: &AppHandle) -> Result<SyncDb, String> {
    let conn = Connection::open(db_path(app)?).map_err(|e| format!("open mirror.db: {e}"))?;
    conn.pragma_update(None, "journal_mode", "WAL")
        .map_err(|e| format!("WAL pragma: {e}"))?;
    conn.pragma_update(None, "foreign_keys", "ON")
        .map_err(|e| format!("FK pragma: {e}"))?;
    Ok(SyncDb {
        conn: Mutex::new(conn),
        in_tx: Mutex::new(false),
    })
}

/// The sync commands only serve the bundled app origin (same policy as the
/// keychain commands in commands.rs).
fn trusted_origin(window: &tauri::WebviewWindow) -> bool {
    let Ok(url) = window.url() else {
        return false;
    };
    match (url.scheme(), url.host_str()) {
        ("tauri", Some("localhost")) => true,
        ("http" | "https", Some("localhost")) => true,
        _ => false,
    }
}

fn bind_param(p: &Value) -> rusqlite::types::Value {
    match p {
        Value::Null => rusqlite::types::Value::Null,
        Value::Bool(b) => rusqlite::types::Value::Integer(i64::from(*b)),
        Value::Number(n) => {
            if let Some(i) = n.as_i64() {
                rusqlite::types::Value::Integer(i)
            } else {
                rusqlite::types::Value::Real(n.as_f64().unwrap_or(0.0))
            }
        }
        Value::String(s) => rusqlite::types::Value::Text(s.clone()),
        _ => rusqlite::types::Value::Text(p.to_string()),
    }
}

/// Serialize one query row as a JSON object keyed by column name.
fn row_to_json(row: &rusqlite::Row) -> rusqlite::Result<Value> {
    let stmt = row.as_ref();
    let mut out = serde_json::Map::new();
    for (i, name) in stmt.column_names().into_iter().enumerate() {
        let value = match rusqlite::types::Value::from(row.get_ref(i)?) {
            rusqlite::types::Value::Null => Value::Null,
            rusqlite::types::Value::Integer(i) => json!(i),
            rusqlite::types::Value::Real(f) => json!(f),
            rusqlite::types::Value::Text(s) => json!(s),
            rusqlite::types::Value::Blob(b) => json!(b),
        };
        out.insert(name.to_string(), value);
    }
    Ok(Value::Object(out))
}

#[tauri::command]
pub fn sync_ping(window: tauri::WebviewWindow, db: State<'_, SyncDb>) -> Result<(), String> {
    if !trusted_origin(&window) {
        return Err("sync_ping denied: untrusted webview origin".to_string());
    }
    let conn = db.conn.lock().map_err(|e| e.to_string())?;
    conn.query_row("SELECT 1", [], |_| Ok(()))
        .map_err(|e| format!("mirror db ping: {e}"))
}

#[tauri::command]
pub fn sync_begin(window: tauri::WebviewWindow, db: State<'_, SyncDb>) -> Result<(), String> {
    if !trusted_origin(&window) {
        return Err("sync_begin denied: untrusted webview origin".to_string());
    }
    let mut in_tx = db.in_tx.lock().map_err(|e| e.to_string())?;
    if *in_tx {
        return Err("sync_begin: transaction already open".to_string());
    }
    let conn = db.conn.lock().map_err(|e| e.to_string())?;
    conn.execute_batch("BEGIN IMMEDIATE")
        .map_err(|e| format!("sync_begin: {e}"))?;
    *in_tx = true;
    Ok(())
}

#[tauri::command]
pub fn sync_commit(window: tauri::WebviewWindow, db: State<'_, SyncDb>) -> Result<(), String> {
    if !trusted_origin(&window) {
        return Err("sync_commit denied: untrusted webview origin".to_string());
    }
    let mut in_tx = db.in_tx.lock().map_err(|e| e.to_string())?;
    if !*in_tx {
        return Err("sync_commit: no open transaction".to_string());
    }
    let conn = db.conn.lock().map_err(|e| e.to_string())?;
    conn.execute_batch("COMMIT").map_err(|e| format!("sync_commit: {e}"))?;
    *in_tx = false;
    Ok(())
}

#[tauri::command]
pub fn sync_rollback(window: tauri::WebviewWindow, db: State<'_, SyncDb>) -> Result<(), String> {
    if !trusted_origin(&window) {
        return Err("sync_rollback denied: untrusted webview origin".to_string());
    }
    let mut in_tx = db.in_tx.lock().map_err(|e| e.to_string())?;
    if !*in_tx {
        return Err("sync_rollback: no open transaction".to_string());
    }
    let conn = db.conn.lock().map_err(|e| e.to_string())?;
    conn.execute_batch("ROLLBACK")
        .map_err(|e| format!("sync_rollback: {e}"))?;
    *in_tx = false;
    Ok(())
}

/// Execute one statement (INSERT/UPDATE/DELETE/DDL). Params are positional
/// and MUST be numbered (`?1`, `?2`, …) — rusqlite has no anonymous `?`.
#[tauri::command]
pub fn sync_exec(
    window: tauri::WebviewWindow,
    db: State<'_, SyncDb>,
    sql: String,
    params: Vec<Value>,
) -> Result<Value, String> {
    if !trusted_origin(&window) {
        return Err("sync_exec denied: untrusted webview origin".to_string());
    }
    let conn = db.conn.lock().map_err(|e| e.to_string())?;
    let values: Vec<rusqlite::types::Value> = params.iter().map(bind_param).collect();
    let mut stmt = conn
        .prepare(&sql)
        .map_err(|e| format!("sync_exec prepare: {e}"))?;
    let n = stmt
        .execute(params_from_iter(values))
        .map_err(|e| format!("sync_exec: {e}"))?;
    let last_id = conn.last_insert_rowid();
    Ok(json!({ "changes": n, "lastInsertRowId": last_id }))
}

/// Run one SELECT and return rows as JSON objects (column-name keyed).
#[tauri::command]
pub fn sync_query(
    window: tauri::WebviewWindow,
    db: State<'_, SyncDb>,
    sql: String,
    params: Vec<Value>,
) -> Result<Value, String> {
    if !trusted_origin(&window) {
        return Err("sync_query denied: untrusted webview origin".to_string());
    }
    let conn = db.conn.lock().map_err(|e| e.to_string())?;
    let values: Vec<rusqlite::types::Value> = params.iter().map(bind_param).collect();
    let mut stmt = conn
        .prepare(&sql)
        .map_err(|e| format!("sync_query prepare: {e}"))?;
    let rows = stmt
        .query_map(params_from_iter(values), row_to_json)
        .map_err(|e| format!("sync_query: {e}"))?;
    let mut out = Vec::new();
    for row in rows {
        out.push(row.map_err(|e| format!("sync_query row: {e}"))?);
    }
    Ok(Value::Array(out))
}

#[cfg(test)]
mod tests {
    use super::*;

    fn mem_db() -> SyncDb {
        let conn = Connection::open_in_memory().expect("open");
        conn.pragma_update(None, "journal_mode", "WAL").ok();
        SyncDb {
            conn: Mutex::new(conn),
            in_tx: Mutex::new(false),
        }
    }

    #[test]
    fn exec_query_roundtrip() {
        let db = mem_db();
        {
            let conn = db.conn.lock().unwrap();
            conn.execute_batch(
                "CREATE TABLE t (id INTEGER PRIMARY KEY, name TEXT, n INTEGER, f REAL, flag BOOLEAN)",
            )
            .unwrap();
        }
        // Params via sync_exec with numbered placeholders.
        let out = {
            let conn = db.conn.lock().unwrap();
            conn.execute(
                "INSERT INTO t (name, n, f, flag) VALUES (?1, ?2, ?3, ?4)",
                rusqlite::params!["hello", 7i64, 3.5, 1i64],
            )
            .unwrap();
            conn.last_insert_rowid()
        };
        assert_eq!(out, 1);

        let rows: Vec<Value> = {
            let conn = db.conn.lock().unwrap();
            let mut stmt = conn.prepare("SELECT id, name, n, f, flag FROM t").unwrap();
            let mapped = stmt
                .query_map([], |r| row_to_json(r))
                .unwrap()
                .collect::<Result<Vec<_>, _>>()
                .unwrap();
            mapped
        };
        assert_eq!(rows.len(), 1);
        assert_eq!(rows[0]["name"], json!("hello"));
        assert_eq!(rows[0]["n"], json!(7));
        assert_eq!(rows[0]["f"], json!(3.5));
        assert_eq!(rows[0]["flag"], json!(1));
    }

    #[test]
    fn begin_commit_isolation() {
        let db = mem_db();
        {
            let conn = db.conn.lock().unwrap();
            conn.execute_batch("CREATE TABLE t (id INTEGER PRIMARY KEY, v TEXT)").unwrap();
        }
        {
            let mut in_tx = db.in_tx.lock().unwrap();
            let conn = db.conn.lock().unwrap();
            conn.execute_batch("BEGIN IMMEDIATE").unwrap();
            *in_tx = true;
            conn.execute("INSERT INTO t (v) VALUES (?1)", rusqlite::params!["x"])
                .unwrap();
        }
        // Uncommitted write must be visible on the same connection…
        {
            let conn = db.conn.lock().unwrap();
            let n: i64 = conn.query_row("SELECT COUNT(*) FROM t", [], |r| r.get(0)).unwrap();
            assert_eq!(n, 1);
        }
        // …and rolled back.
        {
            let mut in_tx = db.in_tx.lock().unwrap();
            let conn = db.conn.lock().unwrap();
            conn.execute_batch("ROLLBACK").unwrap();
            *in_tx = false;
        }
        {
            let conn = db.conn.lock().unwrap();
            let n: i64 = conn.query_row("SELECT COUNT(*) FROM t", [], |r| r.get(0)).unwrap();
            assert_eq!(n, 0);
        }
    }
}
