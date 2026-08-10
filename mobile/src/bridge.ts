/**
 * WebView ↔ RN bridge host (Zettelgarden-c6l.4). Receives postMessage
 * requests from the webview shim (webviewShim.js → window.zgMobile) and
 * dispatches them. c6l.2 adds the SQLite commands (op-sqlite on the RN JS
 * thread); c6l.3 adds the keychain token commands. Settings (server URL,
 * account) ship in this phase.
 */

import { Platform } from 'react-native';
import { loadSettings, saveSettings } from './settingsStore';

export interface BridgeRequest {
  id: number;
  cmd: string;
  args?: unknown;
}

export interface BridgeResponse {
  id: number;
  ok: boolean;
  result?: unknown;
  error?: string;
}

/**
 * Handles one webview message (the JSON string ReactNativeWebView delivers).
 * Pure async — the caller injects the response back into the webview.
 */
export async function handleBridgeMessage(
  raw: string,
): Promise<BridgeResponse> {
  let req: BridgeRequest;
  try {
    req = JSON.parse(raw) as BridgeRequest;
  } catch {
    return { id: 0, ok: false, error: 'malformed bridge message' };
  }
  if (typeof req.id !== 'number' || typeof req.cmd !== 'string') {
    return { id: 0, ok: false, error: 'bridge message missing id/cmd' };
  }
  try {
    switch (req.cmd) {
      case 'ping':
        return { id: req.id, ok: true, result: 'pong' };
      case 'get_platform':
        return { id: req.id, ok: true, result: Platform.OS };
      case 'load_settings':
        return { id: req.id, ok: true, result: await loadSettings() };
      case 'save_settings': {
        await saveSettings((req.args ?? {}) as Record<string, unknown>);
        return { id: req.id, ok: true, result: null };
      }
      default:
        return { id: req.id, ok: false, error: `unknown command: ${req.cmd}` };
    }
  } catch (err) {
    return { id: req.id, ok: false, error: String(err) };
  }
}

/** Builds the injectJavaScript payload that resolves the shim's pending call. */
export function responseScript(resp: BridgeResponse): string {
  return `window.__zgMobileBridge && window.__zgMobileBridge.onResponse(${JSON.stringify(
    resp,
  )}); true;`;
}
