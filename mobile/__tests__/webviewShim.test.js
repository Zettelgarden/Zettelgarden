/**
 * Webview shim protocol tests (Zettelgarden-c6l.4): createShim must post
 * request/response bridge messages over postMessage and resolve/reject the
 * returned promises when onResponse fires. Note: constructing the shim fires
 * the initial `ping` (the `ready` handshake), so command messages are
 * located by cmd rather than assumed to be first.
 */

const { createShim, shimSource } = require('../webviewShim');

function makeShim() {
  const posted = [];
  const postMessage = msg => posted.push(msg);
  const shim = createShim({ postMessage, platform: 'android' });
  return { shim, posted };
}

function postedFor(posted, cmd) {
  const raw = posted.map(JSON.parse).find(m => m.cmd === cmd);
  expect(raw).toBeTruthy();
  return raw;
}

test('loadSettings posts a load_settings bridge request', () => {
  const { shim, posted } = makeShim();
  const p = shim.api.loadSettings();
  const req = postedFor(posted, 'load_settings');
  expect(typeof req.id).toBe('number');
  shim.bridge.onResponse({
    id: req.id,
    ok: true,
    result: { serverUrl: 'http://x' },
  });
  return expect(p).resolves.toEqual({ serverUrl: 'http://x' });
});

test('saveSettings passes args and resolves on ok', () => {
  const { shim, posted } = makeShim();
  const p = shim.api.saveSettings({ serverUrl: 'https://y' });
  const req = postedFor(posted, 'save_settings');
  expect(req.args).toEqual({ serverUrl: 'https://y' });
  shim.bridge.onResponse({ id: req.id, ok: true, result: null });
  return expect(p).resolves.toBeNull();
});

test('a failed response rejects the pending call', () => {
  const { shim, posted } = makeShim();
  const p = shim.api.loadSettings();
  const req = postedFor(posted, 'load_settings');
  shim.bridge.onResponse({ id: req.id, ok: false, error: 'boom' });
  return expect(p).rejects.toThrow('boom');
});

test('ready resolves after the ping round-trip', () => {
  const { shim, posted } = makeShim();
  const req = postedFor(posted, 'ping');
  shim.bridge.onResponse({ id: req.id, ok: true, result: 'pong' });
  return expect(shim.api.ready).resolves.toBeUndefined();
});

test('late onResponse for an unknown id is ignored', () => {
  const { shim } = makeShim();
  expect(() => shim.bridge.onResponse({ id: 999, ok: true })).not.toThrow();
});

test('shimSource installs the global zgMobile with the platform', () => {
  const source = shimSource('android');
  expect(source).toContain('window.__zgMobileInstalled');
  expect(source).toContain('window.zgMobile = shim.api');
  expect(source).toContain('"android"');
  expect(source).toContain('window.__zgMobileBridge = shim.bridge');
});
