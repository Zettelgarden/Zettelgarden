/**
 * WebView ↔ RN bridge host (Zettelgarden-c6l.4). Receives postMessage
 * requests from the webview shim (webviewShim.js → window.zgMobile) and
 * dispatches them. c6l.2 adds the SQLite commands (op-sqlite on the RN JS
 * thread); c6l.3 adds the keychain token commands. Settings (server URL,
 * account) ship in this phase.
 */

import { Platform } from 'react-native';
import { loadSettings, saveSettings } from './settingsStore';
import { keychainDelete, keychainGet, keychainSet } from './keychain';
import {
  sqlBegin,
  sqlCommit,
  sqlExec,
  sqlQuery,
  sqlReset,
  sqlRollback,
} from './sqlite';

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
      // Keychain auth (c6l.3): the webview's localStorage shim redirects the
      // token/username keys here; the JWT lives in the OS keychain, never in
      // WebView storage.
      case 'keychain_get': {
        const { key } = (req.args ?? {}) as { key?: string };
        return {
          id: req.id,
          ok: true,
          result: key ? await keychainGet(key) : null,
        };
      }
      case 'keychain_set': {
        const { key, value } = (req.args ?? {}) as {
          key?: string;
          value?: string;
        };
        if (!key || typeof value !== 'string') {
          return {
            id: req.id,
            ok: false,
            error: 'keychain_set requires key + value',
          };
        }
        await keychainSet(key, value);
        return { id: req.id, ok: true, result: null };
      }
      case 'keychain_delete': {
        await keychainDelete();
        return { id: req.id, ok: true, result: null };
      }
      // SQLite bridge (c6l.2): the webview sync engine's MobileStorageAdapter
      // drives these. The executor is op-sqlite on the RN JS thread — the
      // webview never touches SQLite directly.
      case 'sql_begin': {
        await sqlBegin();
        return { id: req.id, ok: true, result: null };
      }
      case 'sql_commit': {
        await sqlCommit();
        return { id: req.id, ok: true, result: null };
      }
      case 'sql_rollback': {
        await sqlRollback();
        return { id: req.id, ok: true, result: null };
      }
      case 'sql_exec': {
        const { sql, params } = (req.args ?? {}) as {
          sql: string;
          params?: unknown[];
        };
        if (typeof sql !== 'string') {
          return { id: req.id, ok: false, error: 'sql_exec requires sql' };
        }
        return {
          id: req.id,
          ok: true,
          result: await sqlExec(sql, params ?? []),
        };
      }
      case 'sql_query': {
        const { sql, params } = (req.args ?? {}) as {
          sql: string;
          params?: unknown[];
        };
        if (typeof sql !== 'string') {
          return { id: req.id, ok: false, error: 'sql_query requires sql' };
        }
        return {
          id: req.id,
          ok: true,
          result: await sqlQuery(sql, params ?? []),
        };
      }
      case 'db_reset': {
        await sqlReset();
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
