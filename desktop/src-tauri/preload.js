/**
 * Zettelgarden desktop preload shim (Tauri v2, Phase 2a — issue c5b).
 *
 * Runs before the web app loads and intercepts the token/username keys the
 * web app historically kept in localStorage, redirecting them to the OS
 * keychain via the Rust commands. The web app is otherwise unchanged; the
 * shell owns credential storage.
 *
 * Also exposes `window.zgDesktop` for the shell: settings, and (stubbed for
 * Phase 2b) online/pending-change status that the UI indicators will read.
 */
(function () {
  'use strict';

  const KEYS = new Set(['token', 'username']);

  function invoke(cmd, args) {
    const api = window.__TAURI__;
    if (!api || !api.core) {
      return Promise.reject(new Error('__TAURI__ bridge unavailable'));
    }
    return api.core.invoke(cmd, args);
  }

  // ---- localStorage shim: token/username -> keychain ----------------------
  const realStorage = window.localStorage;

  const keychainGet = (key) =>
    invoke('get_secret', { key }).then((v) => v ?? null);
  const keychainSet = (key, value) =>
    invoke('set_secret', { key, value }).then(() => undefined);
  const keychainDelete = (key) =>
    invoke('delete_secret', { key }).then(() => undefined);

  // Async persistence: localStorage is synchronous, so the shim is a hybrid —
  // reads resolve from an in-memory cache (primed from the keychain at boot),
  // writes go to both the cache and the keychain (fire-and-forget). The auth
  // boot flow awaits the prime before the app checks the token.
  let cache = new Map();
  let primed = false;
  const prime = async () => {
    if (primed) return;
    try {
      for (const key of KEYS) {
        const v = await keychainGet(key);
        if (v !== null) cache.set(key, v);
      }
    } catch (err) {
      console.error('[zg-desktop] keychain prime failed:', err);
    } finally {
      primed = true;
    }
  };

  const storageProxy = new Proxy(realStorage, {
    get(target, prop) {
      if (prop === 'getItem') {
        return (key) => (KEYS.has(key) ? (cache.get(key) ?? null) : target.getItem(key));
      }
      if (prop === 'setItem') {
        return (key, value) => {
          if (KEYS.has(key)) {
            cache.set(key, value);
            keychainSet(key, String(value)).catch((e) =>
              console.error('[zg-desktop] keychain write failed:', e),
            );
            return undefined;
          }
          return target.setItem(key, value);
        };
      }
      if (prop === 'removeItem') {
        return (key) => {
          if (KEYS.has(key)) {
            cache.delete(key);
            keychainDelete(key).catch((e) =>
              console.error('[zg-desktop] keychain delete failed:', e),
            );
            return undefined;
          }
          return target.removeItem(key);
        };
      }
      const v = target[prop];
      return typeof v === 'function' ? v.bind(target) : v;
    },
  });

  // ---- expose shell API ---------------------------------------------------
  window.zgDesktop = {
    /** Resolves once the keychain has been read into the shim cache. */
    ready: prime(),
    platform: 'linux',

    getToken: () => keychainGet('token'),
    setToken: (t) => keychainSet('token', t),

    loadSettings: () => invoke('load_settings'),
    saveSettings: (settings) => invoke('save_settings', { settings }),

    windowControls: {
      minimize: () => invoke('window_minimize'),
      maximize: () => invoke('window_maximize'),
      close: () => invoke('window_close'),
      isMaximized: () => invoke('window_is_maximized'),
    },

    // Phase 2b stubs: the sync engine will drive these.
    getSyncStatus: () => ({ online: navigator.onLine, pendingChanges: 0 }),
    onSyncStatus: (_cb) => () => {},
  };

  // ---- install ------------------------------------------------------------
  Object.defineProperty(window, 'localStorage', { value: storageProxy });
  window.addEventListener('DOMContentLoaded', () => {
    window.zgDesktop.ready.catch(() => undefined);
  });
})();
