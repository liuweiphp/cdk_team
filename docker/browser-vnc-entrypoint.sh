#!/usr/bin/env bash
set -euo pipefail

DISPLAY_ID="${DISPLAY_ID:-:99}"
SCREEN_SIZE="${SCREEN_SIZE:-1440x900x24}"
PROFILE_ROOT="${YFJC_PROFILE_DIR:-/tmp/yfjc_profiles}"
PROFILE_NAME="${PROFILE_NAME:-inspect}"
TARGET_URL="${TARGET_URL:-https://www.yfjc.xyz/}"
CHROME_BIN="${CHROME_BIN:-/ms-playwright/chromium-1194/chrome-linux/chrome}"
USER_DATA_DIR="${PROFILE_ROOT}/${PROFILE_NAME}"
CDP_WS_URL_FILE="${YFJC_CDP_WS_URL_FILE:-${PROFILE_ROOT}/.cdp/browser-vnc-ws-url.txt}"
export CDP_WS_URL_FILE

mkdir -p "${USER_DATA_DIR}"
mkdir -p "$(dirname "${CDP_WS_URL_FILE}")"

Xvfb "${DISPLAY_ID}" -screen 0 "${SCREEN_SIZE}" -ac +extension GLX +render -noreset &
export DISPLAY="${DISPLAY_ID}"

sleep 2
fluxbox >/tmp/fluxbox.log 2>&1 &
x11vnc -display "${DISPLAY_ID}" -forever -shared -nopw -rfbport 5900 -quiet >/tmp/x11vnc.log 2>&1 &
websockify --web=/usr/share/novnc/ 0.0.0.0:6080 localhost:5900 >/tmp/novnc.log 2>&1 &
socat TCP-LISTEN:9222,fork,reuseaddr,bind=0.0.0.0 TCP:127.0.0.1:9223 >/tmp/socat-cdp.log 2>&1 &
python3 - <<'PY' >/tmp/cdp-ws-url.log 2>&1 &
import json
import os
import time
import urllib.request

target = os.environ["CDP_WS_URL_FILE"]
for _ in range(60):
    try:
        with urllib.request.urlopen("http://127.0.0.1:9223/json/version", timeout=1) as resp:
            data = json.load(resp)
        ws_url = (data.get("webSocketDebuggerUrl") or "").strip()
        if ws_url:
            ws_url = ws_url.replace("ws://127.0.0.1:9223", "ws://browser-vnc:9222")
            ws_url = ws_url.replace("ws://localhost:9223", "ws://browser-vnc:9222")
            with open(target, "w", encoding="utf-8") as handle:
                handle.write(ws_url)
            break
    except Exception:
        time.sleep(1)
PY

exec "${CHROME_BIN}" \
  --no-sandbox \
  --disable-dev-shm-usage \
  --disable-blink-features=AutomationControlled \
  --no-first-run \
  --no-default-browser-check \
  --remote-debugging-address=127.0.0.1 \
  --remote-debugging-port=9223 \
  --lang=zh-CN,zh,en-US,en \
  --window-size=1440,900 \
  --user-data-dir="${USER_DATA_DIR}" \
  "${TARGET_URL}"
