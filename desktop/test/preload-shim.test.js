import { describe, it, expect } from 'vitest';
import { createShim } from '../src-tauri/preload.js';

/** Minimal localStorage double backed by a Map. */
function makeLocalStorage() {
  const store = new Map();
  return {
    get length() {
      return store.size;
    },
    key: (i) => [...store.keys()][i] ?? null,
    getItem: (k) => (store.has(k) ? store.get(k) : null),
    setItem: (k, v) => store.set(k, String(v)),
    removeItem: (k) => store.delete(k),
    clear: () => store.clear(),
    _store: store,
  };
}

/** Records every keychain invoke so tests can assert ordering. */
function makeInvoke(ops) {
  return (cmd, args) => {
    ops.push([cmd, args]);
    return Promise.resolve(cmd === 'get_secret' ? null : undefined);
  };
}

describe('keychain localStorage shim', () => {
  it('routes token/username to the keychain and passes other keys through', async () => {
    const real = makeLocalStorage();
    real.setItem('theme', 'dark');
    const ops = [];
    const shim = createShim({ localStorage: real, invoke: makeInvoke(ops) });
    await shim.ready;

    shim.shim.setItem('token', 'jwt-1');
    shim.shim.setItem('theme', 'light');
    expect(shim.shim.getItem('token')).toBe('jwt-1'); // from the shim cache
    expect(real.getItem('token')).toBeNull(); // never in real storage
    expect(real.getItem('theme')).toBe('light'); // passthrough
    await shim.flush();
    expect(ops).toContainEqual(['set_secret', { key: 'token', value: 'jwt-1' }]);

    shim.shim.removeItem('token');
    expect(shim.shim.getItem('token')).toBeNull();
    await shim.flush();
    expect(ops).toContainEqual(['delete_secret', { key: 'token' }]);
  });

  it('length and key() include keychain keys', async () => {
    const real = makeLocalStorage();
    real.setItem('theme', 'dark');
    const ops = [];
    const shim = createShim({ localStorage: real, invoke: makeInvoke(ops) });
    await shim.ready;

    // Prime the keychain cache with a token.
    const ops2 = [];
    shim.shim.setItem('token', 'jwt-1');
    await shim.flush();
    void ops2;

    expect(shim.shim.length).toBe(2); // token + theme
    const keys = [];
    for (let i = 0; i < shim.shim.length; i++) keys.push(shim.shim.key(i));
    expect(keys).toContain('token');
    expect(keys).toContain('theme');
  });

  it('clear() removes keychain keys (no stale token after logout)', async () => {
    const real = makeLocalStorage();
    real.setItem('theme', 'dark');
    const ops = [];
    const shim = createShim({ localStorage: real, invoke: makeInvoke(ops) });
    await shim.ready;
    shim.shim.setItem('token', 'jwt-1');
    shim.shim.setItem('username', 'nick');
    await shim.flush();

    shim.shim.clear();
    await shim.flush();
    expect(ops.filter(([c]) => c === 'delete_secret').map(([, a]) => a.key).sort()).toEqual(
      ['token', 'username'],
    );
    expect(shim.shim.getItem('token')).toBeNull();
    expect(real.length).toBe(0);
    expect(shim.shim.length).toBe(0);
  });

  it('logout-then-login applies keychain ops in order (delete before set)', async () => {
    const real = makeLocalStorage();
    const ops = [];
    const shim = createShim({ localStorage: real, invoke: makeInvoke(ops) });
    await shim.ready;
    shim.shim.setItem('token', 'old-jwt');
    await shim.flush();

    // Logout (removeItem) followed immediately by login (setItem): the
    // keychain must end with the NEW token — the delete must not land after
    // the set.
    ops.length = 0;
    shim.shim.removeItem('token');
    shim.shim.setItem('token', 'new-jwt');
    await shim.flush();

    const ordered = ops.map(([c, a]) => `${c}:${a.key}`).join(',');
    expect(ordered).toBe('delete_secret:token,set_secret:token');
    const lastSet = ops.filter(([c]) => c === 'set_secret').pop();
    expect(lastSet[1].value).toBe('new-jwt');
    expect(shim.shim.getItem('token')).toBe('new-jwt');
  });

  it('migrates legacy token/username out of real localStorage at prime', async () => {
    const real = makeLocalStorage();
    real.setItem('token', 'legacy-jwt'); // pre-shim plaintext install
    real.setItem('username', 'nick');
    real.setItem('theme', 'dark');
    const ops = [];
    const shim = createShim({ localStorage: real, invoke: makeInvoke(ops) });
    await shim.ready;

    expect(shim.shim.getItem('token')).toBe('legacy-jwt'); // still visible
    expect(real.getItem('token')).toBeNull(); // migrated out of webview storage
    expect(real.getItem('username')).toBeNull();
    expect(real.getItem('theme')).toBe('dark'); // non-keychain key untouched
    await shim.flush();
    expect(ops).toContainEqual(['set_secret', { key: 'token', value: 'legacy-jwt' }]);
    expect(ops).toContainEqual(['set_secret', { key: 'username', value: 'nick' }]);
  });

  it('a failing keychain op does not stall later writes for the same key', async () => {
    const real = makeLocalStorage();
    const ops = [];
    let failNext = true;
    const invoke = (cmd, args) => {
      ops.push([cmd, args]);
      if (cmd === 'set_secret' && failNext) {
        failNext = false;
        return Promise.reject(new Error('keychain locked'));
      }
      return Promise.resolve(null);
    };
    const shim = createShim({ localStorage: real, invoke });
    await shim.ready;

    shim.shim.setItem('token', 'a');
    shim.shim.setItem('token', 'b'); // must still run after 'a' fails
    await shim.flush();
    expect(ops.filter(([c]) => c === 'set_secret').length).toBe(2);
  });
});
