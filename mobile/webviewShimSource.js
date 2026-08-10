/**
 * Zettelgarden mobile webview shim — THE injected script (Zettelgarden-c6l.3).
 *
 * This file is BOTH:
 *   1. The source of truth for what runs inside the WebView (via
 *      injectedJavaScriptBeforeContentLoaded), stringified at build time by
 *      scripts/buildShimSource.js into shimSource.generated.js; and
 *   2. A CommonJS module (UMD footer) so unit tests can require createShim
 *      directly.
 *
 * Why not Function.prototype.toString? RN 0.7x ships Hermes, whose compiled
 * functions do not preserve reliable toString() output — the serialized
 * factory threw "bytecode is not defined" on device (worked in node/V8). A
 * real source file has no such engine dependency.
 *
 * What it does: installs window.zgMobile + window.__zgMobileBridge over the
 * postMessage bridge (RN → src/bridge.ts), replaces window.localStorage with
 * a keychain-backed proxy (token/username keys redirect to the OS keychain —
 * never persisted in WebView storage), primes the keychain on every page
 * load (token effectively injected per navigation), and revokes on logout
 * (removeItem → keychain delete → AuthContext boots to login).
 *
 * __ZG_PLATFORM__ is replaced with the actual platform ('android'|'ios') at
 * build time.
 */

var zgShimFactory = (function () {
  'use strict';

  /**
   * Builds the shim API. Pure function: no imports, no closure references —
   * safe to serialize (the file itself is stringified, so this is a
   * readability nicety rather than a correctness requirement).
   */
  function createShim(deps) {
    var postMessage = deps.postMessage;
    var platform = deps.platform;
    var real = deps.localStorage;

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

    function keychainGet(key) {
      return invoke('keychain_get', { key: key });
    }
    function keychainSet(key, value) {
      return invoke('keychain_set', { key: key, value: value });
    }
    function keychainDelete() {
      return invoke('keychain_delete');
    }

    // Per-key write serialization: localStorage is synchronous, so writes are
    // fire-and-forget — but they MUST apply to the keychain in call order. A
    // naive logout-then-login (removeItem then setItem) could otherwise race
    // and leave the delete landing after the set (stale logged-out state).
    var queues = new Map();
    function enqueue(key, op) {
      var prev = queues.get(key) || Promise.resolve();
      var next = prev.then(op, op); // run even if the previous op failed
      queues.set(key, next.catch(function () {})); // keep the chain alive
      return next;
    }

    function prime() {
      if (primed) return Promise.resolve();
      primed = true;
      // per-iteration const bindings: the enqueued closures capture the right
      // key/legacy value even though the loop advances before their
      // microtasks run.
      var chain = Promise.resolve();
      for (var i = 0; i < KEYS.length; i++) {
        (function (key) {
          chain = chain
            .then(function () {
              return keychainGet(key);
            })
            .then(function (v) {
              if (v !== null) {
                cache.set(key, v);
              } else if (real.getItem(key) !== null) {
                // Pre-shim installs left token/username in real WebView
                // storage: migrate the plaintext copy into the keychain and
                // remove it, so the credential never lingers in WebView
                // storage.
                var legacy = real.getItem(key);
                cache.set(key, legacy);
                enqueue(key, function () {
                  return keychainSet(key, legacy);
                });
                real.removeItem(key);
              }
            });
        })(KEYS[i]);
      }
      return chain;
    }

    function presentKeys() {
      return KEYS.filter(function (k) {
        return cache.has(k);
      });
    }

    function realKeys() {
      var out = [];
      for (var i = 0; i < real.length; i++) {
        var k = real.key(i);
        if (KEYS.indexOf(k) === -1) out.push(k);
      }
      return out;
    }

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

  return { createShim: createShim };
})();

// Boot — runs inside the WebView only (window is undefined in Node require).
(function () {
  'use strict';

  if (typeof window === 'undefined' || window.__zgMobileInstalled) return;
  window.__zgMobileInstalled = true;

  if (window.console && window.console.log) {
    console.log(
      '[zg-mobile] boot: rnw=' +
        (window.ReactNativeWebView ? 'yes' : 'no'),
    );
  }

  var post =
    window.ReactNativeWebView &&
    window.ReactNativeWebView.postMessage.bind(window.ReactNativeWebView);
  if (!post) {
    if (window.console && window.console.log) {
      console.log('[zg-mobile] ReactNativeWebView unavailable — shim skipped');
    }
    return;
  }

  var realStorage = null;
  try {
    realStorage = window.localStorage;
    var shim = zgShimFactory.createShim({
      postMessage: post,
      platform: '__ZG_PLATFORM__',
      localStorage: realStorage,
    });
    window.__zgMobileBridge = shim.bridge;
    window.zgMobile = shim.api;
    Object.defineProperty(window, 'localStorage', {
      value: shim.localStorageShim,
      configurable: true,
    });
    if (window.console && window.console.log) {
      console.log('[zg-mobile] shim installed');
      shim.api.ready
        .then(function () {
          console.log('[zg-mobile] bridge ready');
        })
        .catch(function () {});
    }
    shim.api.ready.catch(function () {});
  } catch (e) {
    if (window.console && window.console.log) {
      console.log('[zg-mobile] install failed: ' + (e && (e.message || e.stack)));
    }
  }
})();

// UMD footer: allows unit tests to import createShim directly; no-op in the
// WebView (typeof module === 'undefined' there).
if (typeof module !== 'undefined' && module.exports) {
  module.exports = { createShim: zgShimFactory.createShim };
}
