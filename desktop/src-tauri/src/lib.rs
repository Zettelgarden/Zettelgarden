mod commands;

#[cfg_attr(mobile, tauri::mobile_entry_point)]
pub fn run() {
    tauri::Builder::default()
        .plugin(tauri_plugin_store::Builder::new().build())
        .invoke_handler(tauri::generate_handler![
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
            let shim = include_str!("../preload.js");
            let _window = tauri::WebviewWindowBuilder::new(
                app,
                "main",
                tauri::WebviewUrl::App("index.html".into()),
            )
            .title("Zettelgarden")
            .inner_size(1400.0, 900.0)
            .min_inner_size(800.0, 600.0)
            .initialization_script(shim)
            .build()?;
            Ok(())
        })
        .run(tauri::generate_context!())
        .expect("error while running tauri application");
}
