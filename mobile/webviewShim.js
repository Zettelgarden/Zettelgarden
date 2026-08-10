'use strict';

/**
 * Zettelgarden mobile webview shim (epic Zettelgarden-v5b, Phase 3a —
 * Zettelgarden-c6l.3). Injected into the WebView before the web app loads via
 * react-native-webview's injectedJavaScriptBeforeContentLoaded.
 *
 * c6l.4 shipped the shell contract (window.zgMobile: ready/platform/invoke/
 * settings). c6l.3 adds the keychain-backed localStorage shim: the web app
 * keeps reading `localStorage.getItem('token')`, but the shim redirects the
 * token/username keys to the OS keychain over the bridge (RN →
 * src/keychain.ts → react-native-keychain) — the JWT never persists in
 * WebView storage, exactly the desktop preload.js pattern. Every page load
 * re-primes from the keychain, so the token is effectively injected per
 * navigation; logout removes it (keychain delete) and the webview boots to
 * the login screen.
 *
 * Structure mirrors desktop/src-tauri/preload.js: a dependency-free factory
 * (createShim) that is both unit-testable and serialized into the injected
 * script via toString() — so the webview code and the tested code are the
 * same function.
 */

/**
 * Builds the shim API. Pure function: no imports, no closure references —
 * it must be serializable with toString() into the injected WebView script.
 *
 * @param {{
 *   postMessage: (msg: string) => void,
 *   platform: string,
 *   localStorage: Storage,
 * }} deps
 * @returns {{
 *   bridge: { onResponse: (p: {id:number, ok:boolean, result?:unknown, error?:string}) => void },
 *   api: { ready: Promise<unknown>, platform: string,
 *          invoke: (cmd: string, args?: unknown) => Promise<unknown>,
 *          loadSettings: () => Promise<unknown>, saveSettings: (s: unknown) => Promise<unknown>,
 *          getToken: () => Promise<string|null>, setToken: (t: string) => void,
 *          deleteToken: () => void },
 *   localStorageShim: Storage,
 * }}
 */
function createShim({ postMessage, platform, localStorage: real }) {
  var pending = new Map();
  var seq = 0;

  function invoke(cmd, args) {
    var id = ++seq;
    return new Promise(function (resolve, reject) {
      pending.set(id, { resolve: resolve, reject: reject });
      postMessage(JSON.stringify({ id: id, cmd: cmd, args: args }));
    });
  }

  // ---- keychain-backed localStorage shim (token/username only) ----
  var KEYS = ['token', 'username'];
  var cache = new Map();
  var primed = false;

  var keychainGet = function (key) {
    return invoke('keychain_get', { key: key });
  };
  var keychainSet = function (key, value) {
    return invoke('keychain_set', { key: key, value: value });
  };
  var keychainDelete = function () {
    return invoke('keychain_delete');
  };

  // Per-key write serialization: localStorage is synchronous, so writes are
  // fire-and-forget — but they MUST apply to the keychain in call order. A
  // naive logout-then-login (removeItem then setItem) could otherwise race
  // and leave the delete landing after the set (stale logged-out state).
  var queues = new Map();
  var enqueue = function (key, op) {
    var prev = queues.get(key) || Promise.resolve();
    var next = prev.then(op, op); // run even if the previous op failed
    queues.set(
      key,
      next.catch(function () {}),
    ); // keep the chain alive
    return next;
  };

  var prime = async function () {
    if (primed) return;
    try {
      // const per-iteration bindings: the enqueued closures capture the
      // right key/legacy value even though the loop advances before their
      // microtasks run (the desktop preload.js original uses the same shape
      // via `for (const key of KEYS)`).
      for (var i = 0; i < KEYS.length; i++) {
        const key = KEYS[i];
        const v = await keychainGet(key);
        if (v !== null) {
          cache.set(key, v);
        } else if (real.getItem(key) !== null) {
          // Pre-shim installs left token/username in real WebView storage:
          // migrate the plaintext copy into the keychain and remove it, so
          // the credential never lingers in WebView storage.
          const legacy = real.getItem(key);
          cache.set(key, legacy);
          enqueue(key, function () {
            return keychainSet(key, legacy);
          });
          real.removeItem(key);
        }
      }
    } finally {
      primed = true;
    }
  };

  var presentKeys = function () {
    return KEYS.filter(function (k) {
      return cache.has(k);
    });
  };

  var realKeys = function () {
    var out = [];
    for (var i = 0; i < real.length; i++) {
      var k = real.key(i);
      if (KEYS.indexOf(k) === -1) out.push(k);
    }
    return out;
  };

  var shim = new Proxy(real, {
    get: function (target, prop) {
      if (prop === 'getItem') {
        return function (key) {
          return KEYS.indexOf(key) !== -1
            ? cache.get(key) || null
            : target.getItem(key);
        };
      }
      if (prop === 'setItem') {
        return function (key, value) {
          if (KEYS.indexOf(key) !== -1) {
            cache.set(key, String(value));
            enqueue(key, function () {
              return keychainSet(key, String(value));
            });
            return undefined;
          }
          return target.setItem(key, value);
        };
      }
      if (prop === 'removeItem') {
        return function (key) {
          if (KEYS.indexOf(key) !== -1) {
            cache.delete(key);
            enqueue(key, keychainDelete);
            return undefined;
          }
          return target.removeItem(key);
        };
      }
      // A future clear() must not leave the token in the keychain (that
      // would silently keep a user logged in after "logout").
      if (prop === 'clear') {
        return function () {
          cache.clear();
          for (var i = 0; i < KEYS.length; i++) {
            enqueue(KEYS[i], keychainDelete);
          }
          return target.clear();
        };
      }
      if (prop === 'key') {
        return function (i) {
          return presentKeys().concat(realKeys())[i] || null;
        };
      }
      if (prop === 'length') {
        return presentKeys().length + target.length;
      }
      var v = target[prop];
      return typeof v === 'function' ? v.bind(target) : v;
    },
  });

  var ready = Promise.all([
    invoke('ping').catch(function () {}),
    prime().catch(function () {}),
  ]).then(function () {
    return undefined;
  });

  return {
    bridge: {
      onResponse: function (payload) {
        var p = pending.get(payload && payload.id);
        if (!p) return;
        pending.delete(payload.id);
        if (payload.ok) {
          p.resolve(payload.result);
        } else {
          p.reject(new Error((payload && payload.error) || 'bridge error'));
        }
      },
    },
    api: {
      ready: ready,
      platform: platform,
      // Generic bridge call — the sync engine's MobileStorageAdapter drives
      // sql_* commands through this.
      invoke: function (cmd, args) {
        return invoke(cmd, args);
      },
      loadSettings: function () {
        return invoke('load_settings');
      },
      saveSettings: function (settings) {
        return invoke('save_settings', settings);
      },
      // Token helpers (keychain-backed; the localStorage shim also redirects
      // the token key, so the web app never needs to call these directly).
      getToken: function () {
        return keychainGet('token');
      },
      setToken: function (token) {
        enqueue('token', function () {
          return keychainSet('token', String(token));
        });
      },
      deleteToken: function () {
        enqueue('token', keychainDelete);
      },
    },
    localStorageShim: shim,
  };
}

/**
 * The JavaScript source to inject into the WebView. Installs the shim
 * factory, wires window.ReactNativeWebView.postMessage, replaces
 * window.localStorage with the keychain-backed proxy, and exposes
 * window.__zgMobileBridge + window.zgMobile.
 */
function shimSource(platform) {
  var factory = createShim.toString();
  return (
    '(function () {\n' +
    '  if (typeof window === "undefined" || window.__zgMobileInstalled) return;\n' +
    '  window.__zgMobileInstalled = true;\n' +
    '  var post = window.ReactNativeWebView &&\n' +
    '    window.ReactNativeWebView.postMessage.bind(window.ReactNativeWebView);\n' +
    '  if (!post) return;\n' +
    '  var realStorage = window.localStorage;\n' +
    '  var factory = (' +
    factory +
    ');\n' +
    '  var shim = factory({ postMessage: post, platform: ' +
    JSON.stringify(platform) +
    ', localStorage: realStorage });\n' +
    '  window.__zgMobileBridge = shim.bridge;\n' +
    '  window.zgMobile = shim.api;\n' +
    '  Object.defineProperty(window, "localStorage", { value: shim.localStorageShim, configurable: true });\n' +
    '  shim.api.ready.catch(function () {});\n' +
    '})();'
  );
}

module.exports = { createShim, shimSource };
