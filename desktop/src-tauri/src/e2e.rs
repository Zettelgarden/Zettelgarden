// E2E smoke support (Zettelgarden-77j): an env-gated init script + report
// command that let the desktop app run a scripted offline-CRUD scenario in a
// real webview. Active ONLY when ZG_E2E=1; in normal runs none of this ships.
//
// The init script runs before the app bundle: it wraps window.fetch to count
// HTTP calls (the offline phase must make ZERO) and exposes __zgE2E.report /
// __zgE2E.done which forward to the e2e_report command (writing JSON lines to
// ZG_E2E_OUTPUT; done() exits the process so the orchestrator can assert).

pub const E2E_INIT_SCRIPT: &str = r#"(function () {
  'use strict';
  var invoke = function (cmd, args) {
    var internals = window.__TAURI_INTERNALS__;
    if (internals && typeof internals.invoke === 'function') {
      return internals.invoke(cmd, args);
    }
    return Promise.reject(new Error('Tauri bridge unavailable for ' + cmd));
  };
  var fetchCount = 0;
  var origFetch = window.fetch.bind(window);
  window.fetch = function () {
    fetchCount++;
    return origFetch.apply(window, arguments);
  };
  var t0 = Date.now();
  window.__zgE2E = {
    scenario: typeof __ZG_E2E_SCENARIO__ !== 'undefined' ? __ZG_E2E_SCENARIO__ : 'fresh',
    now: function () { return Date.now() - t0; },
    fetchCount: function () { return fetchCount; },
    resetFetchCount: function () { fetchCount = 0; },
    report: function (entry) {
      entry.ts = Date.now() - t0;
      return invoke('e2e_report', { entry: JSON.stringify(entry) }).catch(function () {});
    },
    done: function (ok, summary) {
      summary.ok = ok;
      return invoke('e2e_report', { entry: JSON.stringify(summary), isFinal: true, ok: ok })
        .catch(function () {});
    },
  };
})();
"#;

use std::io::Write;

#[tauri::command]
pub fn e2e_report(
    _app: tauri::AppHandle,
    entry: String,
    is_final: Option<bool>,
    ok: Option<bool>,
) -> Result<(), String> {
    // The bridge is only injected when ZG_E2E=1 (see lib.rs); refuse the
    // command outright outside an E2E run so a stray page can't even reach
    // the file write (Zettelgarden-23n).
    if std::env::var("ZG_E2E").as_deref() != Ok("1") {
        return Err("e2e_report is only available when ZG_E2E=1".to_string());
    }
    let Some(path) = std::env::var("ZG_E2E_OUTPUT").ok() else {
        return Ok(()); // E2E run without an output path configured; no-op
    };
    let mut line = entry;
    if !line.ends_with('\n') {
        line.push('\n');
    }
    let mut f = std::fs::OpenOptions::new()
        .create(true)
        .append(true)
        .open(&path)
        .map_err(|e| format!("e2e_report open: {e}"))?;
    f.write_all(line.as_bytes())
        .map_err(|e| format!("e2e_report write: {e}"))?;
    if is_final.unwrap_or(false) {
        // Scenario finished: exit so the orchestrator can assert on the file
        // and the exit code (1 on failure).
        let code = if ok.unwrap_or(false) { 0 } else { 1 };
        std::process::exit(code);
    }
    Ok(())
}
