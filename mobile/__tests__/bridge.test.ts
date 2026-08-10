/**
 * RN-side bridge host tests (Zettelgarden-c6l.4): handleBridgeMessage routes
 * shim requests to the settings store; responseScript builds the
 * injectJavaScript payload. AsyncStorage is the in-memory __mocks__ copy.
 */

import AsyncStorage from '@react-native-async-storage/async-storage';
import { handleBridgeMessage, responseScript } from '../src/bridge';

jest.mock('@op-engineering/op-sqlite');
const { open, __instances, __pushResult } = jest.requireMock(
  '@op-engineering/op-sqlite',
) as {
  open: jest.Mock;
  __instances: Array<{
    execute: jest.Mock;
    close: jest.Mock;
    delete: jest.Mock;
  }>;
  __pushResult: (result: unknown) => void;
};

beforeEach(async () => {
  await AsyncStorage.clear();
});

test('ping returns pong', async () => {
  const resp = await handleBridgeMessage(
    JSON.stringify({ id: 1, cmd: 'ping' }),
  );
  expect(resp).toEqual({ id: 1, ok: true, result: 'pong' });
});

test('load_settings returns the stored server URL', async () => {
  const save = await handleBridgeMessage(
    JSON.stringify({
      id: 2,
      cmd: 'save_settings',
      args: { serverUrl: 'http://zg.local' },
    }),
  );
  expect(save.ok).toBe(true);
  const load = await handleBridgeMessage(
    JSON.stringify({ id: 3, cmd: 'load_settings' }),
  );
  expect(load).toEqual({
    id: 3,
    ok: true,
    result: { serverUrl: 'http://zg.local' },
  });
});

test('load_settings with nothing stored returns an empty object', async () => {
  const load = await handleBridgeMessage(
    JSON.stringify({ id: 4, cmd: 'load_settings' }),
  );
  expect(load.result).toEqual({});
});

test('malformed JSON is rejected with id 0', async () => {
  const resp = await handleBridgeMessage('not json');
  expect(resp.ok).toBe(false);
  expect(resp.id).toBe(0);
});

test('unknown commands return an error response', async () => {
  const resp = await handleBridgeMessage(
    JSON.stringify({ id: 5, cmd: 'explode' }),
  );
  expect(resp.ok).toBe(false);
  expect(resp.error).toContain('unknown command');
});

test('responseScript calls the shim onResponse with the payload', () => {
  const script = responseScript({ id: 7, ok: true, result: { a: 1 } });
  expect(script).toContain('__zgMobileBridge.onResponse');
  expect(script).toContain('"id":7');
  expect(script.trim().endsWith('true;')).toBe(true);
});

describe('keychain bridge commands (c6l.3)', () => {
  const keychainStore = (
    jest.requireMock('react-native-keychain') as unknown as {
      __store: Map<string, unknown>;
    }
  ).__store;

  beforeEach(() => {
    keychainStore.clear();
  });

  test('keychain_get returns the stored value or null', async () => {
    const empty = await handleBridgeMessage(
      JSON.stringify({ id: 20, cmd: 'keychain_get', args: { key: 'token' } }),
    );
    expect(empty).toEqual({ id: 20, ok: true, result: null });

    const set = await handleBridgeMessage(
      JSON.stringify({
        id: 21,
        cmd: 'keychain_set',
        args: { key: 'token', value: 'jwt-xyz' },
      }),
    );
    expect(set.ok).toBe(true);

    const get = await handleBridgeMessage(
      JSON.stringify({ id: 22, cmd: 'keychain_get', args: { key: 'token' } }),
    );
    expect(get).toEqual({ id: 22, ok: true, result: 'jwt-xyz' });
  });

  test('keychain_set rejects missing value', async () => {
    const resp = await handleBridgeMessage(
      JSON.stringify({ id: 23, cmd: 'keychain_set', args: { key: 'token' } }),
    );
    expect(resp.ok).toBe(false);
    expect(resp.error).toContain('keychain_set');
  });

  test('keychain_delete clears the stored pair', async () => {
    await handleBridgeMessage(
      JSON.stringify({
        id: 24,
        cmd: 'keychain_set',
        args: { key: 'token', value: 'jwt' },
      }),
    );
    const del = await handleBridgeMessage(
      JSON.stringify({ id: 25, cmd: 'keychain_delete' }),
    );
    expect(del.ok).toBe(true);
    const get = await handleBridgeMessage(
      JSON.stringify({ id: 26, cmd: 'keychain_get', args: { key: 'token' } }),
    );
    expect(get.result).toBeNull();
  });
});

describe('sqlite bridge commands (c6l.2)', () => {
  /** The op-sqlite DB instance the executor opened (mock). */
  const db = () => __instances[__instances.length - 1];

  test('sql_exec runs the statement with params and returns rowsAffected', async () => {
    __pushResult({ rowsAffected: 2, rows: [] });
    const resp = await handleBridgeMessage(
      JSON.stringify({
        id: 10,
        cmd: 'sql_exec',
        args: { sql: 'INSERT INTO t (a) VALUES (?)', params: ['x'] },
      }),
    );
    expect(resp).toEqual({ id: 10, ok: true, result: { rowsAffected: 2 } });
    expect(db().execute).toHaveBeenCalledWith('INSERT INTO t (a) VALUES (?)', [
      'x',
    ]);
  });

  test('sql_query returns the row objects from the executor', async () => {
    __pushResult({ rowsAffected: 0, rows: [{ row_uuid: 'r1', version: 1 }] });
    const resp = await handleBridgeMessage(
      JSON.stringify({
        id: 11,
        cmd: 'sql_query',
        args: { sql: 'SELECT row_uuid, version FROM mirror_rows', params: [] },
      }),
    );
    expect(resp.ok).toBe(true);
    expect(resp.result).toEqual([{ row_uuid: 'r1', version: 1 }]);
  });

  test('sql_begin/commit/rollback run on the executor', async () => {
    for (const cmd of ['sql_begin', 'sql_commit', 'sql_rollback']) {
      const resp = await handleBridgeMessage(JSON.stringify({ id: 12, cmd }));
      expect(resp).toEqual({ id: 12, ok: true, result: null });
    }
    const calls = db().execute.mock.calls.map(c => c[0]);
    expect(calls.slice(-3)).toEqual(['BEGIN IMMEDIATE', 'COMMIT', 'ROLLBACK']);
  });

  test('sql_exec/sql_query reject a missing sql argument', async () => {
    const execResp = await handleBridgeMessage(
      JSON.stringify({ id: 13, cmd: 'sql_exec', args: { params: [] } }),
    );
    expect(execResp.ok).toBe(false);
    expect(execResp.error).toContain('sql');
    const queryResp = await handleBridgeMessage(
      JSON.stringify({ id: 14, cmd: 'sql_query' }),
    );
    expect(queryResp.ok).toBe(false);
  });

  test('db_reset closes and deletes the mirror database', async () => {
    // Force the executor to open (first sql command), then reset.
    await handleBridgeMessage(JSON.stringify({ id: 15, cmd: 'sql_begin' }));
    const before = __instances.length;
    const resp = await handleBridgeMessage(
      JSON.stringify({ id: 16, cmd: 'db_reset' }),
    );
    expect(resp).toEqual({ id: 16, ok: true, result: null });
    const last = db();
    expect(last.close).toHaveBeenCalled();
    expect(last.delete).toHaveBeenCalled();
    expect(open).toHaveBeenCalledTimes(before); // no re-open until next command
  });
});
