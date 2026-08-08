/**
 * Zettelgarden desktop preload shim (Tauri v2, Phase 2a — issue c5b, hardening
 * v5b.8).
 *
 * Runs before the web app loads (Tauri initialization_script) and intercepts
 * the token/username keys the web app historically kept in localStorage,
 * redirecting them to the OS keychain via the Rust commands. The web app is
 * otherwise unchanged; the shell owns credential storage.
 *
 * The core (createShim) is UMD so it is unit-testable under vitest (module
 * export) while still loading as a plain initialization script (attaches
 * ZgPreloadShim to the global). Hardening: per-key serialized keychain writes
 * (logout-then-login cannot reorder), clear()/key()/length include keychain
 * keys, and the page itself never gets the raw __TAURI__ bridge (only the
 * shim's invoke path does).
 */
(function (root, factory) {
  if (typeof module === 'object' && module.exports) {
    module.exports = factory();
  } else {
    root.ZgPreloadShim = factory();
    // Auto-install only when loaded as an initialization script (browser),
    // never when imported by a test runner.
    root.ZgPreloadShim.autoInstall();
  }
})(typeof self !== 'undefined' ? self : this, function () {
  'use strict';

  const DEFAULT_KEYS = ['token', 'username'];

  /**
   * Builds the keychain-backed localStorage shim. `invoke(cmd, args)` must
   * return a Promise. Returns { shim, ready, flush, queueSet, queueDelete }.
   */
  function createShim({ localStorage: real, invoke, keychainKeys = DEFAULT_KEYS }) {
    const KEYS = new Set(keychainKeys);
    const cache = new Map();
    let primed = false;

    const keychainGet = (key) => invoke('get_secret', { key }).then((v) => v ?? null);
    const keychainSet = (key, value) => invoke('set_secret', { key, value });
    const keychainDelete = (key) => invoke('delete_secret', { key });

    // Per-key write serialization: localStorage is synchronous, so writes are
    // fire-and-forget — but they MUST apply to the keychain in call order. A
    // naive logout-then-login (removeItem then setItem) could otherwise race
    // and leave the delete landing after the set (stale logged-out state).
    const queues = new Map();
    const enqueue = (key, op) => {
      const prev = queues.get(key) ?? Promise.resolve();
      const next = prev.then(op, op); // run even if the previous op failed
      queues.set(key, next.catch(() => undefined)); // keep the chain alive
      return next;
    };

    const prime = async () => {
      if (primed) return;
      try {
        for (const key of KEYS) {
          const v = await keychainGet(key);
          if (v !== null) {
            cache.set(key, v);
          } else if (real.getItem(key) !== null) {
            // Pre-shim installs left token/username in real localStorage:
            // migrate the plaintext copy into the keychain and remove it, so
            // the credential never lingers in webview storage.
            const legacy = real.getItem(key);
            cache.set(key, legacy);
            enqueue(key, () => keychainSet(key, legacy));
            real.removeItem(key);
          }
        }
      } finally {
        primed = true;
      }
    };

    const presentKeys = () => [...KEYS].filter((k) => cache.has(k));

    const realKeys = () => {
      const out = [];
      for (let i = 0; i < real.length; i++) {
        const k = real.key(i);
        if (!KEYS.has(k)) out.push(k);
      }
      return out;
    };

    const shim = new Proxy(real, {
      get(target, prop) {
        if (prop === 'getItem') {
          return (key) => (KEYS.has(key) ? (cache.get(key) ?? null) : target.getItem(key));
        }
        if (prop === 'setItem') {
          return (key, value) => {
            if (KEYS.has(key)) {
              cache.set(key, String(value));
              enqueue(key, () => keychainSet(key, String(value)));
              return undefined;
            }
            return target.setItem(key, value);
          };
        }
        if (prop === 'removeItem') {
          return (key) => {
            if (KEYS.has(key)) {
              cache.delete(key);
              enqueue(key, () => keychainDelete(key));
              return undefined;
            }
            return target.removeItem(key);
          };
        }
        // The web app does not call clear() today, but a future clear() must
        // not leave the token in the keychain (that would silently keep a
        // user logged in after "logout").
        if (prop === 'clear') {
          return () => {
            cache.clear();
            for (const key of KEYS) enqueue(key, () => keychainDelete(key));
            return target.clear();
          };
        }
        if (prop === 'key') {
          return (i) => presentKeys().concat(realKeys())[i] ?? null;
        }
        if (prop === 'length') {
          return presentKeys().length + target.length;
        }
        const v = target[prop];
        return typeof v === 'function' ? v.bind(target) : v;
      },
    });

    return {
      shim,
      ready: prime(),
      flush: async () => {
        await prime();
        await Promise.all([...queues.values()]);
      },
      get: async (key) => {
        await prime();
        return cache.get(key) ?? null;
      },
      queueSet: (key, value) => enqueue(key, () => keychainSet(key, value)),
      queueDelete: (key) => enqueue(key, () => keychainDelete(key)),
    };
  }

  function autoInstall() {
    if (typeof window === 'undefined' || !window.localStorage) return;
    if (window.__zgShimInstalled) return;
    window.__zgShimInstalled = true;

    const invoke = (cmd, args) => {
      const internals = window.__TAURI_INTERNALS__;
      if (internals && typeof internals.invoke === 'function') {
        return internals.invoke(cmd, args);
      }
      const api = window.__TAURI__;
      if (api && api.core) return api.core.invoke(cmd, args);
      return Promise.reject(new Error('Tauri bridge unavailable'));
    };

    const shim = createShim({ localStorage: window.localStorage, invoke });

    Object.defineProperty(window, 'localStorage', { value: shim.shim });

    window.zgDesktop = {
      /** Resolves once the keychain has been read into the shim cache. */
      ready: shim.ready,
      platform: 'linux',

      getToken: () => shim.get('token'),
      setToken: (t) => shim.queueSet('token', t),

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
      onSyncStatus: () => () => {},
    };

    window.addEventListener('DOMContentLoaded', () => {
      shim.ready.catch(() => undefined);
    });
  }

  return { createShim, autoInstall };
});
