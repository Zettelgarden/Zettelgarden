/**
 * Webview shim tests (Zettelgarden-c6l.3): bridge protocol + keychain-backed
 * localStorage interception. Constructing the shim fires the initial `ping`
 * and primes both keychain keys (token/username); every enqueued write posts
 * a bridge message that must be responded to before the per-key chain can
 * advance. `drain()` drives the whole conversation to quiescence, so tests
 * exercise the real sequencing (prime → migration → setItem → removeItem →
 * clear) instead of responding once.
 */

const { createShim, shimSource } = require('../webviewShim');

/** A minimal in-memory Storage for the real (pre-shim) localStorage. */
function makeRealStorage(initial = {}) {
  const map = new Map(Object.entries(initial));
  return {
    getItem: k => (map.has(k) ? map.get(k) : null),
    setItem: (k, v) => void map.set(k, String(v)),
    removeItem: k => void map.delete(k),
    clear: () => map.clear(),
    key: i => [...map.keys()][i] ?? null,
    get length() {
      return map.size;
    },
  };
}

function makeShim(real = makeRealStorage()) {
  const posted = [];
  const postMessage = msg => posted.push(msg);
  const shim = createShim({
    postMessage,
    platform: 'android',
    localStorage: real,
  });
  return { shim, posted, real };
}

function postedFor(posted, cmd) {
  const raw = posted.map(JSON.parse).find(m => m.cmd === cmd);
  expect(raw).toBeTruthy();
  return raw;
}

/** Lets the per-key write queues (microtasks) post their bridge messages. */
const flush = () => new Promise(r => setTimeout(r, 0));

/**
 * Responds to every bridge message as it appears (ok), letting sequential
 * prime/migration/write chains advance, until the queue is quiescent.
 * keychain_get responses come from `token`/`username` (null default).
 */
async function drain(shim, posted, { token, username } = {}) {
  const responded = new Set();
  for (let i = 0; i < 30; i++) {
    let changed = false;
    for (const m of posted.map(JSON.parse)) {
      if (responded.has(m.id)) continue;
      responded.add(m.id);
      changed = true;
      if (m.cmd === 'ping') {
        shim.bridge.onResponse({ id: m.id, ok: true, result: 'pong' });
      } else if (m.cmd === 'keychain_get') {
        shim.bridge.onResponse({
          id: m.id,
          ok: true,
          result: m.args.key === 'token' ? (token ?? null) : (username ?? null),
        });
      } else {
        shim.bridge.onResponse({ id: m.id, ok: true, result: null });
      }
    }
    if (!changed) break;
    await flush();
  }
}

/** Drains the handshake then awaits the shim's ready promise. */
async function primeWith(shim, posted, values) {
  await drain(shim, posted, values);
  await shim.api.ready;
}

describe('bridge protocol (c6l.4 base)', () => {
  test('loadSettings posts a load_settings bridge request', () => {
    const { shim, posted } = makeShim();
    const p = shim.api.loadSettings();
    const req = postedFor(posted, 'load_settings');
    shim.bridge.onResponse({
      id: req.id,
      ok: true,
      result: { serverUrl: 'http://x' },
    });
    return expect(p).resolves.toEqual({ serverUrl: 'http://x' });
  });

  test('generic invoke posts an arbitrary command (sqlite bridge surface)', () => {
    const { shim, posted } = makeShim();
    const p = shim.api.invoke('sql_query', { sql: 'SELECT 1', params: [] });
    const req = postedFor(posted, 'sql_query');
    expect(req.args).toEqual({ sql: 'SELECT 1', params: [] });
    shim.bridge.onResponse({ id: req.id, ok: true, result: [{ 1: 1 }] });
    return expect(p).resolves.toEqual([{ 1: 1 }]);
  });

  test('a failed response rejects the pending call', () => {
    const { shim, posted } = makeShim();
    const p = shim.api.loadSettings();
    const req = postedFor(posted, 'load_settings');
    shim.bridge.onResponse({ id: req.id, ok: false, error: 'boom' });
    return expect(p).rejects.toThrow('boom');
  });

  test('ready resolves after ping + keychain prime', async () => {
    const { shim, posted } = makeShim();
    await primeWith(shim, posted);
    const gets = posted.map(JSON.parse).filter(m => m.cmd === 'keychain_get');
    expect(gets.map(g => g.args.key)).toEqual(['token', 'username']);
  });

  test('late onResponse for an unknown id is ignored', () => {
    const { shim } = makeShim();
    expect(() => shim.bridge.onResponse({ id: 999, ok: true })).not.toThrow();
  });

  test('shimSource installs the globals + localStorage replacement', () => {
    const source = shimSource('android');
    expect(source).toContain('window.__zgMobileInstalled');
    expect(source).toContain('window.zgMobile = shim.api');
    expect(source).toContain('"android"');
    expect(source).toContain('Object.defineProperty(window, "localStorage"');
  });
});

describe('keychain-backed localStorage shim (c6l.3)', () => {
  test('getItem("token") returns the keychain value after prime', async () => {
    const { shim, posted } = makeShim();
    await primeWith(shim, posted, { token: 'jwt-123', username: 'nick' });
    expect(shim.localStorageShim.getItem('token')).toBe('jwt-123');
    expect(shim.localStorageShim.getItem('username')).toBe('nick');
  });

  test('setItem("token") queues a keychain write and updates the cache', async () => {
    const { shim, posted } = makeShim();
    await primeWith(shim, posted);
    shim.localStorageShim.setItem('token', 'jwt-new');
    expect(shim.localStorageShim.getItem('token')).toBe('jwt-new');
    await drain(shim, posted);
    const set = postedFor(posted, 'keychain_set');
    expect(set.args).toEqual({ key: 'token', value: 'jwt-new' });
  });

  test('removeItem("token") queues a keychain delete after a prior set', async () => {
    const { shim, posted } = makeShim();
    await primeWith(shim, posted);
    shim.localStorageShim.setItem('token', 'jwt-x');
    await drain(shim, posted);
    posted.length = 0; // reset after the setItem
    shim.localStorageShim.removeItem('token');
    expect(shim.localStorageShim.getItem('token')).toBeNull();
    await drain(shim, posted);
    const del = postedFor(posted, 'keychain_delete');
    expect(del.args).toBeUndefined();
  });

  test('clear() queues keychain deletes for both keys', async () => {
    const { shim, real, posted } = makeShim();
    await primeWith(shim, posted);
    shim.localStorageShim.setItem('token', 't');
    shim.localStorageShim.setItem('username', 'u');
    shim.localStorageShim.setItem('uiPref', 'dark');
    await drain(shim, posted);
    posted.length = 0;
    shim.localStorageShim.clear();
    expect(shim.localStorageShim.getItem('token')).toBeNull();
    expect(shim.localStorageShim.getItem('username')).toBeNull();
    await drain(shim, posted);
    const deletes = posted
      .map(JSON.parse)
      .filter(m => m.cmd === 'keychain_delete');
    expect(deletes).toHaveLength(2);
    // Non-key keys go to real storage.
    expect(real.getItem('uiPref')).toBeNull();
    expect(real.length).toBe(0);
  });

  test('non-key keys pass through to real localStorage', () => {
    const { shim, real } = makeShim();
    shim.localStorageShim.setItem('uiPref', 'dark');
    expect(real.getItem('uiPref')).toBe('dark');
    expect(shim.localStorageShim.getItem('uiPref')).toBe('dark');
    expect(shim.localStorageShim.length).toBe(1);
  });

  test('legacy plaintext token in webview storage migrates to the keychain', async () => {
    const { shim, real, posted } = makeShim(
      makeRealStorage({ token: 'legacy-token', username: 'legacy-user' }),
    );
    await primeWith(shim, posted);
    expect(shim.localStorageShim.getItem('token')).toBe('legacy-token');
    // The plaintext was moved into the keychain (keychain_set queued) and
    // removed from real WebView storage.
    const sets = posted.map(JSON.parse).filter(m => m.cmd === 'keychain_set');
    expect(sets.map(s => s.args)).toEqual([
      { key: 'token', value: 'legacy-token' },
      { key: 'username', value: 'legacy-user' },
    ]);
    expect(real.getItem('token')).toBeNull();
    expect(real.getItem('username')).toBeNull();
  });

  test('api.getToken reads the keychain directly', async () => {
    const { shim, posted } = makeShim();
    await primeWith(shim, posted);
    posted.length = 0;
    const p = shim.api.getToken();
    const req = postedFor(posted, 'keychain_get');
    expect(req.args.key).toBe('token');
    shim.bridge.onResponse({ id: req.id, ok: true, result: 'direct-jwt' });
    await expect(p).resolves.toBe('direct-jwt');
  });
});
