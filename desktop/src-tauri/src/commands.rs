use serde::{Deserialize, Serialize};
use tauri::Manager;

/// Secure token storage via the OS keychain (macOS Keychain, Windows Credential
/// Manager, Linux Secret Service / keyring). The webview never sees the raw
/// token in localStorage — the preload shim routes get/set through these
/// commands (epic Zettelgarden-v5b, Phase 2a).
///
/// Service/account names are shared with the web app's expectations: the app
/// stores `token` and `username` keys (the web app used localStorage for
/// these; the shim redirects exactly those keys to the keychain).

const SERVICE: &str = "zettelgarden";

/// The secret commands only serve the bundled app origin. withGlobalTauri is
/// off and the CSP blunts injected scripts, but window.__TAURI_INTERNALS__ is
/// still page-visible, so an XSS that gets code running could otherwise call
/// get_secret directly. Gate on the webview origin as defense in depth: only
/// the app bundle (tauri://localhost / http://tauri.localhost) and the local
/// dev server (http://localhost) may read or write keychain entries.
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

/// Credential store: the OS keychain (keyring crate) by default, or a JSON
/// file when ZG_KEYCHAIN_FILE is set. The file backend exists for headless /
/// container environments with no Secret Service daemon (and for the
/// deterministic E2E smoke) — it is NOT encrypted, so it must only be used
/// where the data directory is trusted (the flag is opt-in).
enum CredentialStore {
    Os,
    File {
        path: std::path::PathBuf,
        cache: std::sync::Mutex<serde_json::Map<String, serde_json::Value>>,
    },
}

fn credential_store() -> CredentialStore {
    match std::env::var("ZG_KEYCHAIN_FILE") {
        Ok(path) => CredentialStore::File {
            path: std::path::PathBuf::from(path),
            cache: std::sync::Mutex::new(serde_json::Map::new()),
        },
        Err(_) => CredentialStore::Os,
    }
}

impl CredentialStore {
    fn get(&self, key: &str) -> Result<Option<String>, String> {
        match self {
            CredentialStore::Os => {
                let entry = keyring::Entry::new(SERVICE, key).map_err(|e| e.to_string())?;
                match entry.get_password() {
                    Ok(v) => Ok(Some(v)),
                    Err(keyring::Error::NoEntry) => Ok(None),
                    Err(e) => Err(e.to_string()),
                }
            }
            CredentialStore::File { path, cache } => {
                let map = self.load(path, cache)?;
                Ok(map.get(key).and_then(|v| v.as_str()).map(String::from))
            }
        }
    }

    fn set(&self, key: &str, value: &str) -> Result<(), String> {
        match self {
            CredentialStore::Os => {
                let entry = keyring::Entry::new(SERVICE, key).map_err(|e| e.to_string())?;
                entry.set_password(value).map_err(|e| e.to_string())
            }
            CredentialStore::File { path, cache } => {
                let mut map = self.load(path, cache)?;
                map.insert(
                    key.to_string(),
                    serde_json::Value::String(value.to_string()),
                );
                self.store(path, cache, map)
            }
        }
    }

    fn delete(&self, key: &str) -> Result<(), String> {
        match self {
            CredentialStore::Os => {
                let entry = keyring::Entry::new(SERVICE, key).map_err(|e| e.to_string())?;
                match entry.delete_credential() {
                    Ok(()) => Ok(()),
                    Err(keyring::Error::NoEntry) => Ok(()),
                    Err(e) => Err(e.to_string()),
                }
            }
            CredentialStore::File { path, cache } => {
                let mut map = self.load(path, cache)?;
                map.remove(key);
                self.store(path, cache, map)
            }
        }
    }

    fn load(
        &self,
        path: &std::path::Path,
        cache: &std::sync::Mutex<serde_json::Map<String, serde_json::Value>>,
    ) -> Result<serde_json::Map<String, serde_json::Value>, String> {
        let mut cache = cache.lock().map_err(|e| e.to_string())?;
        if !cache.is_empty() {
            return Ok(cache.clone());
        }
        match std::fs::read_to_string(path) {
            Ok(raw) => {
                let parsed: serde_json::Map<String, serde_json::Value> =
                    serde_json::from_str(&raw).map_err(|e| format!("keychain file parse: {e}"))?;
                *cache = parsed.clone();
                Ok(parsed)
            }
            Err(_) => Ok(serde_json::Map::new()),
        }
    }

    fn store(
        &self,
        path: &std::path::Path,
        cache: &std::sync::Mutex<serde_json::Map<String, serde_json::Value>>,
        map: serde_json::Map<String, serde_json::Value>,
    ) -> Result<(), String> {
        if let Some(parent) = path.parent() {
            std::fs::create_dir_all(parent).map_err(|e| format!("keychain mkdir: {e}"))?;
        }
        let raw = serde_json::to_string_pretty(&map).map_err(|e| e.to_string())?;
        std::fs::write(path, raw).map_err(|e| format!("keychain write: {e}"))?;
        let mut cache = cache.lock().map_err(|e| e.to_string())?;
        *cache = map;
        Ok(())
    }
}

#[tauri::command]
pub fn get_secret(window: tauri::WebviewWindow, key: String) -> Result<Option<String>, String> {
    if !trusted_origin(&window) {
        return Err("get_secret denied: untrusted webview origin".to_string());
    }
    credential_store().get(&key)
}

#[tauri::command]
pub fn set_secret(window: tauri::WebviewWindow, key: String, value: String) -> Result<(), String> {
    if !trusted_origin(&window) {
        return Err("set_secret denied: untrusted webview origin".to_string());
    }
    credential_store().set(&key, &value)
}

#[tauri::command]
pub fn delete_secret(window: tauri::WebviewWindow, key: String) -> Result<(), String> {
    if !trusted_origin(&window) {
        return Err("delete_secret denied: untrusted webview origin".to_string());
    }
    credential_store().delete(&key)
}

/// App settings persisted to the app data dir (server URL, account, ui flags).
#[derive(Debug, Clone, Default, Serialize, Deserialize)]
#[serde(rename_all = "camelCase")]
pub struct AppSettings {
    pub server_url: Option<String>,
    pub account_email: Option<String>,
    /// Stub for Phase 2b: offline/pending-change UI state lives here once the
    /// sync engine is wired in.
    pub pending_changes: Option<i64>,
}

#[tauri::command]
pub fn load_settings(app: tauri::AppHandle) -> Result<AppSettings, String> {
    let path = app
        .path()
        .app_data_dir()
        .map_err(|e| e.to_string())?
        .join("settings.json");
    match std::fs::read_to_string(&path) {
        Ok(raw) => serde_json::from_str(&raw).map_err(|e| e.to_string()),
        Err(_) => Ok(AppSettings::default()),
    }
}

#[tauri::command]
pub fn save_settings(app: tauri::AppHandle, settings: AppSettings) -> Result<(), String> {
    let dir = app.path().app_data_dir().map_err(|e| e.to_string())?;
    std::fs::create_dir_all(&dir).map_err(|e| e.to_string())?;
    let raw = serde_json::to_string_pretty(&settings).map_err(|e| e.to_string())?;
    std::fs::write(dir.join("settings.json"), raw).map_err(|e| e.to_string())
}

// Window controls (the existing Electron shell had these for the frameless
// Linux titlebar; keep parity).
#[tauri::command]
pub fn window_minimize(window: tauri::WebviewWindow) -> Result<(), String> {
    window.minimize().map_err(|e| e.to_string())
}

#[tauri::command]
pub fn window_maximize(window: tauri::WebviewWindow) -> Result<(), String> {
    if window.is_maximized().unwrap_or(false) {
        window.unmaximize().map_err(|e| e.to_string())
    } else {
        window.maximize().map_err(|e| e.to_string())
    }
}

#[tauri::command]
pub fn window_close(window: tauri::WebviewWindow) -> Result<(), String> {
    window.close().map_err(|e| e.to_string())
}

#[tauri::command]
pub fn window_is_maximized(window: tauri::WebviewWindow) -> Result<bool, String> {
    window.is_maximized().map_err(|e| e.to_string())
}

// ---- tests -----------------------------------------------------------------

#[cfg(test)]
mod tests {
    use super::*;

    #[test]
    fn file_keychain_roundtrip_and_persistence() {
        let dir = std::env::temp_dir().join(format!("zg-keychain-test-{}", std::process::id()));
        let path = dir.join("secrets.json");
        std::fs::remove_file(&path).ok();

        // Process 1 writes.
        {
            let store = super::CredentialStore::File {
                path: path.clone(),
                cache: std::sync::Mutex::new(serde_json::Map::new()),
            };
            store.set("token", "jwt-abc").expect("set");
            store.set("username", "nick").expect("set");
            assert_eq!(store.get("token").unwrap().as_deref(), Some("jwt-abc"));
        }
        // A fresh store instance (new process) reads the same file.
        {
            let store = super::CredentialStore::File {
                path: path.clone(),
                cache: std::sync::Mutex::new(serde_json::Map::new()),
            };
            assert_eq!(store.get("token").unwrap().as_deref(), Some("jwt-abc"));
            assert_eq!(store.get("username").unwrap().as_deref(), Some("nick"));
            store.delete("token").expect("delete");
            assert_eq!(store.get("token").unwrap(), None);
        }
        std::fs::remove_dir_all(&dir).ok();
    }

    #[test]
    fn settings_roundtrip_via_filesystem() {
        let dir = std::env::temp_dir().join(format!("zg-settings-test-{}", std::process::id()));
        let settings = AppSettings {
            server_url: Some("https://zg.example.com".into()),
            account_email: Some("you@example.com".into()),
            pending_changes: Some(3),
        };
        let raw = serde_json::to_string_pretty(&settings).expect("serialize");
        std::fs::create_dir_all(&dir).expect("mkdir");
        let path = dir.join("settings.json");
        std::fs::write(&path, raw).expect("write");
        let loaded: AppSettings =
            serde_json::from_str(&std::fs::read_to_string(&path).expect("read")).expect("parse");
        assert_eq!(loaded.server_url.as_deref(), Some("https://zg.example.com"));
        assert_eq!(loaded.account_email.as_deref(), Some("you@example.com"));
        assert_eq!(loaded.pending_changes, Some(3));
        std::fs::remove_dir_all(&dir).ok();
    }

    #[test]
    fn settings_default_when_missing() {
        assert!(AppSettings::default().server_url.is_none());
        assert!(AppSettings::default().pending_changes.is_none());
    }
}
