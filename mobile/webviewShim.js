'use strict';

/**
 * Zettelgarden mobile webview shim (epic Zettelgarden-v5b, Phase 3a —
 * Zettelgarden-c6l.4). Injected into the WebView before the web app loads via
 * react-native-webview's injectedJavaScriptBeforeContentLoaded.
 *
 * Exposes `window.zgMobile` (the contract in
 * zettelkasten-front/src/types/mobile.d.ts) over the postMessage bridge to
 * the RN shell (mobile/src/bridge.ts). c6l.3 extends the same bridge with the
 * keychain token commands; c6l.2 adds the SQLite commands.
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
 * @param {{ postMessage: (msg: string) => void, platform: string }} deps
 * @returns {{ bridge: { onResponse: (p: {id:number, ok:boolean, result?:unknown, error?:string}) => void },
 *             api: { ready: Promise<unknown>, platform: string,
 *                    loadSettings: () => Promise<unknown>,
 *                    saveSettings: (s: unknown) => Promise<unknown> } }}
 */
function createShim({ postMessage, platform }) {
  var pending = new Map();
  var seq = 0;

  function invoke(cmd, args) {
    var id = ++seq;
    return new Promise(function (resolve, reject) {
      pending.set(id, { resolve: resolve, reject: reject });
      postMessage(JSON.stringify({ id: id, cmd: cmd, args: args }));
    });
  }

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
      // Ready = first ping round-trip; c6l.3 chains keychain priming here.
      ready: invoke('ping')
        .then(function () {
          return undefined;
        })
        .catch(function () {
          return undefined;
        }),
      platform: platform,
      loadSettings: function () {
        return invoke('load_settings');
      },
      saveSettings: function (settings) {
        return invoke('save_settings', settings);
      },
    },
  };
}

/**
 * The JavaScript source to inject into the WebView. Installs the shim
 * factory, wires window.ReactNativeWebView.postMessage, and exposes
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
    '  var factory = (' +
    factory +
    ');\n' +
    '  var shim = factory({ postMessage: post, platform: ' +
    JSON.stringify(platform) +
    ' });\n' +
    '  window.__zgMobileBridge = shim.bridge;\n' +
    '  window.zgMobile = shim.api;\n' +
    '  shim.api.ready.catch(function () {});\n' +
    '})();'
  );
}

module.exports = { createShim, shimSource };
