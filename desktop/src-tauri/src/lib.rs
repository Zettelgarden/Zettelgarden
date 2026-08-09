mod commands;
mod e2e;
mod sync_db;

use tauri::Manager;

#[cfg_attr(mobile, tauri::mobile_entry_point)]
pub fn run() {
    tauri::Builder::default()
        .plugin(tauri_plugin_store::Builder::new().build())
        .invoke_handler(tauri::generate_handler![
            e2e::e2e_report,
            sync_db::sync_ping,
            sync_db::sync_begin,
            sync_db::sync_commit,
            sync_db::sync_rollback,
            sync_db::sync_exec,
            sync_db::sync_query,
            commands::get_secret,
            commands::set_secret,
            commands::delete_secret,
            commands::load_settings,
            commands::save_settings,
            commands::window_minimize,
            commands::window_maximize,
            commands::window_close,
            commands::window_is_maximized,
        ])
        .setup(|app| {
            let sync_db = sync_db::init(app.handle()).map_err(|e| {
                std::io::Error::new(std::io::ErrorKind::Other, format!("sync db init: {e}"))
            })?;
            app.manage(sync_db);
            let shim = include_str!("../preload.js");
            let mut builder = tauri::WebviewWindowBuilder::new(
                app,
                "main",
                tauri::WebviewUrl::App("index.html".into()),
            )
            .title("Zettelgarden")
            .inner_size(1400.0, 900.0)
            .min_inner_size(800.0, 600.0)
            .initialization_script(shim);
            // E2E smoke (Zettelgarden-77j): inject the scripted-scenario
            // bridge only when the orchestrator asks for it.
            if std::env::var("ZG_E2E").as_deref() == Ok("1") {
                let scenario =
                    std::env::var("ZG_E2E_SCENARIO").unwrap_or_else(|_| "fresh".to_string());
                let script = format!(
                    "window.__ZG_E2E_SCENARIO__ = {};\n{}",
                    serde_json::to_string(&scenario).unwrap_or_else(|_| "\"fresh\"".to_string()),
                    e2e::E2E_INIT_SCRIPT,
                );
                builder = builder.initialization_script(&script);
            }
            let _window = builder.build()?;
            Ok(())
        })
        .run(tauri::generate_context!())
        .expect("error while running tauri application");
}
