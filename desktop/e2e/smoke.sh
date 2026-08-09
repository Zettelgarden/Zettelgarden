#!/bin/bash
# Desktop E2E smoke (Zettelgarden-77j): runs the REAL Tauri app against a live
# Go backend under xvfb, driving a scripted offline-CRUD scenario inside the
# webview (injected via ZG_E2E). Verifies: fresh-mirror bootstrap through the
# real IPC, keychain (file store) session persistence across restarts, offline
# create/edit/delete of cards + tasks with ZERO fetch() on the hot path,
# reconnect + reconciliation, and the sidebar sync indicator states.
#
# Usage: desktop/e2e/smoke.sh
# Requires: Go toolchain, Node, xvfb-run, cargo (all present on the dev box).

set -euo pipefail

REPO_ROOT="$(cd "$(dirname "${BASH_SOURCE[0]}")/../.." && pwd)"
FRONT="$REPO_ROOT/zettelkasten-front"
DESKTOP="$REPO_ROOT/desktop"
E2E_PORT=18131
E2E_BASE="http://localhost:${E2E_PORT}"

RUN_DIR="$(mktemp -d /tmp/zg-e2e-XXXXXX)"
BACKEND_BIN="$RUN_DIR/zg-backend"
APP_DATA="$RUN_DIR/appdata"            # fresh app-data dir per scenario RUN (same across the 2 launches)
KEYCHAIN_FILE="$RUN_DIR/keychain.json" # file keychain, persists across launches
REPORT="$RUN_DIR/report.jsonl"

echo "== E2E run dir: $RUN_DIR"

# A stale backend squatting on E2E_PORT (e.g. from an aborted run) would make
# the readiness curl succeed against the WRONG process and silently starve the
# app of the CORS fix — kill anything on the port before we start.
if ss -tlnp 2>/dev/null | grep -q ":$E2E_PORT "; then
  echo "== killing stale listener on port $E2E_PORT"
  fuser -k "$E2E_PORT/tcp" 2>/dev/null || true
  sleep 1
fi

# ---------------------------------------------------------------------------
echo "== building Go backend"
(
  cd "$REPO_ROOT/go-backend"
  go build -o "$BACKEND_BIN" .
)

echo "== building frontend dist (VITE_URL=$E2E_BASE/api)"
# The Tauri binary embeds zettelkasten-front/dist at cargo-build time, so the
# dist MUST be built (to the real dist/) before the desktop build.
(
  cd "$FRONT"
  VITE_URL="$E2E_BASE/api" VITE_ENV=dev npx vite build --outDir dist --emptyOutDir >/dev/null
)

echo "== building desktop binary (release + custom-protocol = embedded dist)"
(
  cd "$DESKTOP/src-tauri"
  # tauri's is_dev() = !custom-protocol: a plain cargo build stays in dev mode
  # and loads devUrl. Also touch lib.rs so a dist-only change forces re-embed.
  touch src/lib.rs
  cargo build --release --quiet --features tauri/custom-protocol 2>&1 | tail -5 || true
)
APP_BIN="$DESKTOP/src-tauri/target/release/zettelgarden-desktop"
[ -x "$APP_BIN" ] || { echo "app binary missing: $APP_BIN"; exit 1; }

# ---------------------------------------------------------------------------
echo "== booting backend on port $E2E_PORT"
cd "$REPO_ROOT/go-backend"
ZETTEL_DEV=true \
ZETTEL_PORT="$E2E_PORT" \
ZETTEL_URL="$E2E_BASE" \
SQLITE_PATH="$RUN_DIR/server.db" \
SECRET_KEY="e2e-secret-key-for-jwt-signing-32-chars-min" \
STORAGE_DIR="$RUN_DIR/files" \
ZETTEL_BACKEND_LOG_LOCATION="$RUN_DIR/backend.log" \
TYPESENSE_HOST="http://127.0.0.1:8108" TYPESENSE_PASSWORD="x" TYPESENSE_COLLECTION="zg_e2e" \
ZETTEL_LLM_KEY="e2e-llm-key" ZETTEL_LLM_ENDPOINT="http://127.0.0.1:59999/v1" \
GITHUB_AUTH_ENABLED=false \
"$BACKEND_BIN" >"$RUN_DIR/backend.stdout" 2>&1 &
BACKEND_PID=$!
cd "$REPO_ROOT"
trap 'kill $BACKEND_PID 2>/dev/null || true' EXIT

# wait for the backend to accept requests (must be OUR pid: the port is
# pre-cleaned, so a listening process on it is by definition ours)
for i in $(seq 1 60); do
  if ! kill -0 "$BACKEND_PID" 2>/dev/null; then
    echo "backend died:"; tail -20 "$RUN_DIR/backend.stdout" || true; exit 1
  fi
  if curl -sf "$E2E_BASE/api/settings" >/dev/null 2>&1; then
    echo "== backend up (pid $BACKEND_PID)"
    break
  fi
  sleep 1
done

# ---------------------------------------------------------------------------
run_scenario() {
  local scenario="$1"
  local appdata="$2"
  local report_file="$3"
  echo "== scenario: $scenario"
  set +e
  timeout 240 xvfb-run -a env \
    XDG_DATA_HOME="$appdata" \
    ZG_E2E=1 \
    ZG_E2E_SCENARIO="$scenario" \
    ZG_E2E_OUTPUT="$report_file" \
    ZG_KEYCHAIN_FILE="$KEYCHAIN_FILE" \
    "$APP_BIN" >"$RUN_DIR/app-$scenario.stdout" 2>&1
  local code=$?
  set -e
  echo "  app exit code: $code"
  if [ "$code" -ne 0 ]; then
    echo "  app stdout tail:"; tail -20 "$RUN_DIR/app-$scenario.stdout" || true
    return 1
  fi
}

assert_report() {
  local report_file="$1"
  local scenario="$2"
  [ -f "$report_file" ] || { echo "FAIL: no report file for $scenario"; return 1; }
  echo "  -- report:"
  cat "$report_file"
  # The FINAL done line decides: it carries the scenario's ok flag (probe
  # diagnostics inside the run may report individual failures by design).
  local last
  last="$(tail -1 "$report_file")"
  if ! echo "$last" | grep -q '"ok":true'; then
    echo "FAIL: scenario $scenario did not pass (final: $last)"
    return 1
  fi
}

# --- launch 1: fresh bootstrap + login + offline CRUD + reconcile ----------
if run_scenario fresh "$APP_DATA" "$REPORT.fresh"; then
  assert_report "$REPORT.fresh" fresh
else
  echo "FAIL: fresh scenario failed"; exit 1
fi

echo "== keychain file: $(cat "$KEYCHAIN_FILE" | head -c 120)..."
grep -q '"token"' "$KEYCHAIN_FILE" || { echo "FAIL: keychain file missing token"; exit 1; }

echo "== mirror.db cards table:"
python3 - "$APP_DATA" <<'PYEOF'
import json, os, sqlite3, sys
appdata = sys.argv[1]
db = None
for root, dirs, files in os.walk(appdata):
    if 'mirror.db' in files:
        db = os.path.join(root, 'mirror.db')
        break
if not db:
    print("  no mirror.db under", appdata); sys.exit(0)
conn = sqlite3.connect(db)
for c, in conn.execute("SELECT DISTINCT collection FROM mirror_rows"):
    print(f"  -- {c}:")
    for uuid, ver, data in conn.execute("SELECT row_uuid, version, data FROM mirror_rows WHERE collection=?", (c,)):
        d = json.loads(data)
        print(f"    {uuid[:8]} v{ver} title={d.get('title')} id={d.get('id')} name={d.get('name')}")
outbox = conn.execute("SELECT count(*) FROM sync_outbox").fetchone()[0]
print(f"  outbox rows: {outbox}")
PYEOF

# --- launch 2: relaunch — keychain session + instant offline open ----------
rm -f "$REPORT.session"
if run_scenario session "$APP_DATA" "$REPORT.session"; then
  assert_report "$REPORT.session" session
else
  echo "FAIL: session scenario failed"; exit 1
fi

echo ""
echo "== E2E SMOKE PASSED =="
echo "  run dir: $RUN_DIR"
