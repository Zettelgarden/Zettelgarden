/**
 * RN-side bridge host tests (Zettelgarden-c6l.4): handleBridgeMessage routes
 * shim requests to the settings store; responseScript builds the
 * injectJavaScript payload. AsyncStorage is the in-memory __mocks__ copy.
 */

import AsyncStorage from '@react-native-async-storage/async-storage';
import { handleBridgeMessage, responseScript } from '../src/bridge';

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
