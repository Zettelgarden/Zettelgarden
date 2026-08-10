#!/bin/bash
# Mobile E2E smoke (Zettelgarden-c6l.5): boots a HARDWARE-ACCELERATED Android
# emulator (KVM required — the github ubuntu-24.04 runners have it; this dev
# box does NOT, see the c6l.5 notes), installs the real APK, and verifies the
# full launch chain against a live Go backend:
#   RN runtime boot -> webview shim installed -> bridge handshake (ping +
#   keychain prime) -> WebView navigates to the frontend (vite preview of the
#   real dist) -> no renderer crash.
#
# Engine-level offline convergence (mobile adapter + desktop client, one
# account) is covered by the sync-engine harness scenario 11; this smoke
# proves the shipped device stack actually boots end-to-end. In-app CRUD
# scripting (the desktop ZG_E2E scenario pattern) is a follow-up (see notes).
#
# Usage: mobile/e2e/smoke.sh
# Requires: Go toolchain, Node, JDK 17+, Android SDK (ANDROID_HOME), /dev/kvm.

set -euo pipefail

REPO_ROOT="$(cd "$(dirname "${BASH_SOURCE[0]}")/../.." && pwd)"
FRONT="$REPO_ROOT/zettelkasten-front"
MOBILE="$REPO_ROOT/mobile"
E2E_PORT=18131                     # Go backend
E2E_WEBVIEW_PORT=5173              # vite preview serving dist (app's default URL)
E2E_AVD="${ZG_E2E_AVD:-zg-e2e}"
E2E_IMAGE="${ZG_E2E_IMAGE:-system-images;android-36;google_apis;x86_64}"
E2E_PKG="com.zettelgarden.mobile"
# Local escape hatches (the dev box has no KVM and shares port 5173):
#   ZG_E2E_ALLOW_SOFTWARE=1  run the emulator with -no-accel
#   ZG_E2E_SKIP_PREVIEW=1    don't kill/start the :5173 server (use whatever
#                            is already there — a running dev server)
ALLOW_SOFTWARE="${ZG_E2E_ALLOW_SOFTWARE:-}"
SKIP_PREVIEW="${ZG_E2E_SKIP_PREVIEW:-}"
PREVIEW_PID=""

RUN_DIR="$(mktemp -d /tmp/zg-mobile-e2e-XXXXXX)"
BACKEND_BIN="$RUN_DIR/zg-backend"

export ANDROID_HOME="${ANDROID_HOME:?ANDROID_HOME must be set (mobile toolchain)}"
export PATH="$ANDROID_HOME/platform-tools:$ANDROID_HOME/emulator:$ANDROID_HOME/cmdline-tools/latest/bin:$PATH"
export JAVA_HOME="${JAVA_HOME:?JAVA_HOME must be set (JDK 17+)}"

if [ ! -e /dev/kvm ] && [ -z "$ALLOW_SOFTWARE" ]; then
  echo "FAIL: /dev/kvm not available — a hardware-accelerated emulator is required." >&2
  echo "  This dev box has no KVM (nested virt not exposed); run this in CI" >&2
  echo "  (ubuntu-24.04 runners have KVM) or set ZG_E2E_ALLOW_SOFTWARE=1 for a" >&2
  echo "  slow software-emulated run." >&2
  exit 2
fi
command -v adb >/dev/null || { echo "FAIL: adb not found"; exit 2; }
command -v go >/dev/null || { echo "FAIL: go not found"; exit 2; }

# A stale listener on the webview port (dev servers on this box) would serve
# the WRONG app to the smoke — kill it before we start (unless the caller
# explicitly reuses the existing server via ZG_E2E_SKIP_PREVIEW).
PORTS="$E2E_PORT"
[ -z "$SKIP_PREVIEW" ] && PORTS="$PORTS $E2E_WEBVIEW_PORT"
for port in $PORTS; do
  if ss -tlnp 2>/dev/null | grep -q ":$port "; then
    echo "== killing stale listener on port $port"
    fuser -k "$port/tcp" 2>/dev/null || true
    sleep 1
  fi
done

echo "== run dir: $RUN_DIR"

# ---------------------------------------------------------------------------
echo "== building Go backend"
(
  cd "$REPO_ROOT/go-backend"
  go build -o "$BACKEND_BIN" .
)

echo "== building frontend dist (VITE_URL=http://10.0.2.2:$E2E_PORT/api)"
# The emulator reaches the host's loopback via 10.0.2.2, so the REST/sync
# base URL in the dist must point there (the desktop smoke uses localhost).
(
  cd "$FRONT"
  VITE_URL="http://10.0.2.2:${E2E_PORT}/api" VITE_ENV=dev \
    npx vite build --outDir dist --emptyOutDir >/dev/null
)

echo "== building mobile debug APK (JS bundle embedded, no Metro needed)"
(
  cd "$MOBILE/android"
  "$MOBILE/android/gradlew" -p "$MOBILE/android" assembleDebug --console=plain -q
)
APK="$MOBILE/android/app/build/outputs/apk/debug/app-debug.apk"
[ -f "$APK" ] || { echo "FAIL: APK missing: $APK"; exit 1; }

# ---------------------------------------------------------------------------
echo "== booting backend on port $E2E_PORT"
cd "$REPO_ROOT/go-backend"
ZETTEL_DEV=true \
ZETTEL_PORT="$E2E_PORT" \
ZETTEL_URL="http://localhost:${E2E_PORT}" \
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
trap 'kill $BACKEND_PID 2>/dev/null || true; [ -n "${PREVIEW_PID:-}" ] && kill $PREVIEW_PID 2>/dev/null || true' EXIT

for i in $(seq 1 60); do
  if ! kill -0 "$BACKEND_PID" 2>/dev/null; then
    echo "backend died:"; tail -20 "$RUN_DIR/backend.stdout" || true; exit 1
  fi
  if curl -sf "http://localhost:${E2E_PORT}/api/settings" >/dev/null 2>&1; then
    echo "== backend up (pid $BACKEND_PID)"
    break
  fi
  sleep 1
done

if [ -z "$SKIP_PREVIEW" ]; then
  echo "== serving frontend dist on :$E2E_WEBVIEW_PORT (vite preview)"
  (
    cd "$FRONT"
    npx vite preview --host 0.0.0.0 --port "$E2E_WEBVIEW_PORT" --strictPort >"$RUN_DIR/preview.stdout" 2>&1 &
    echo $! >"$RUN_DIR/preview.pid"
  )
  PREVIEW_PID="$(cat "$RUN_DIR/preview.pid")"
  for i in $(seq 1 30); do
    if curl -sf "http://localhost:${E2E_WEBVIEW_PORT}/" >/dev/null 2>&1; then
      echo "== preview up (pid $PREVIEW_PID)"
      break
    fi
    sleep 1
  done
else
  echo "== reusing existing server on :$E2E_WEBVIEW_PORT (ZG_E2E_SKIP_PREVIEW)"
fi

# ---------------------------------------------------------------------------
echo "== preparing emulator ($E2E_AVD, $E2E_IMAGE)"
sdkmanager --list_installed >/dev/null 2>&1 || true
if ! avdmanager list avd 2>/dev/null | grep -q "Name: $E2E_AVD"; then
  echo "== creating AVD"
  echo no | avdmanager create avd -n "$E2E_AVD" -k "$E2E_IMAGE" -d pixel_5 --force
fi

adb kill-server 2>/dev/null || true
adb start-server 2>/dev/null || true
ACCEL_FLAGS=""
[ -z "$ALLOW_SOFTWARE" ] || ACCEL_FLAGS="-no-accel -feature -Vulkan"
nohup emulator -avd "$E2E_AVD" -no-window -no-audio -no-boot-anim \
  -gpu swiftshader_indirect -memory 4096 -cores 8 $ACCEL_FLAGS \
  >"$RUN_DIR/emulator.stdout" 2>&1 &
EMU_PID=$!
trap 'kill $BACKEND_PID $PREVIEW_PID $EMU_PID 2>/dev/null || true; adb emu kill 2>/dev/null || true' EXIT

echo "== waiting for emulator boot (KVM: ~60-120s; software: minutes)"
booted=""
for i in $(seq 1 90); do
  if ! kill -0 "$EMU_PID" 2>/dev/null; then
    echo "emulator died:"; tail -20 "$RUN_DIR/emulator.stdout" || true; exit 1
  fi
  boot="$(adb -s emulator-5554 shell getprop sys.boot_completed 2>/dev/null || true | tr -d '\r')"
  if [ "$boot" = "1" ]; then echo "== emulator booted (+$((i*5))s)"; booted=1; break; fi
  sleep 5
done
[ -n "$booted" ] || { echo "FAIL: emulator did not boot in time"; tail -20 "$RUN_DIR/emulator.stdout"; exit 1; }

# ---------------------------------------------------------------------------
echo "== installing + launching $E2E_PKG"
adb -s emulator-5554 install -r "$APK" >/dev/null
adb -s emulator-5554 logcat -c
adb -s emulator-5554 shell am start -n "$E2E_PKG/.MainActivity"

echo "== waiting for launch health markers (logcat)"
declare -A SEEN=(
  [rn_boot]="Running \"ZettelgardenMobile\""
  [shim]="\[zg-mobile\] shim installed"
  [bridge]="\[zg-mobile\] bridge ready"
  [load_end]="\[zg-mobile\] onLoadEnd"
  [crash]="F DEBUG|FATAL EXCEPTION"
)
declare -A FOUND=()
for i in $(seq 1 60); do
  logcat="$(adb -s emulator-5554 logcat -d 2>/dev/null || true | tr -d '\r')"
  for k in rn_boot shim bridge load_end; do
    if [ -z "${FOUND[$k]:-}" ] && echo "$logcat" | grep -q "${SEEN[$k]}"; then
      FOUND[$k]=1
      echo "  [ok] $k"
    fi
  done
  # A crash in OUR app (or its webview renderer) is fatal for this smoke;
  # unrelated system crashes on the image are ignored.
  if echo "$logcat" | grep -E "F DEBUG|FATAL EXCEPTION|libwebviewchromium" | grep -q "$E2E_PKG"; then
    echo "FAIL: crash detected in logcat:"
    echo "$logcat" | grep -E "F DEBUG|FATAL EXCEPTION|libwebviewchromium" | grep "$E2E_PKG" | tail -5
    exit 1
  fi
  if [ "${#FOUND[@]}" -ge 4 ]; then break; fi
  sleep 5
done

for k in rn_boot shim bridge load_end; do
  [ -n "${FOUND[$k]:-}" ] || { echo "FAIL: missing launch marker: $k"; exit 1; }
done

# The app must still be alive after the launch chain (no repeated renderer
# crash killed it).
if [ -z "$(adb -s emulator-5554 shell pidof "$E2E_PKG" 2>/dev/null || true | tr -d '\r')" ]; then
  echo "FAIL: $E2E_PKG process is no longer running"; exit 1
fi
echo "  [ok] app process alive"

echo "  -- logcat health lines:"
adb -s emulator-5554 logcat -d 2>/dev/null | grep -E "Running \"ZettelgardenMobile\"|zg-mobile" | tail -8

echo ""
echo "== MOBILE E2E SMOKE PASSED =="
echo "  run dir: $RUN_DIR"
echo "  markers: RN boot + shim install + bridge handshake + webview load, no crash"
