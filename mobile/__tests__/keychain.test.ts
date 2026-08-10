/**
 * Keychain store tests (Zettelgarden-c6l.3): the JWT + username pair
 * round-trips through react-native-keychain (the __mocks__ in-memory store),
 * and merging writes preserve the sibling field.
 */

import * as Keychain from 'react-native-keychain';
import {
  getCredentials,
  setCredentials,
  deleteCredentials,
  keychainGet,
  keychainSet,
  keychainDelete,
  AUTH_SERVICE,
} from '../src/keychain';

const store = (Keychain as unknown as { __store: Map<string, unknown> })
  .__store;

beforeEach(() => {
  store.clear();
});

test('getCredentials returns null when nothing is stored', async () => {
  await expect(getCredentials()).resolves.toBeNull();
});

test('setCredentials then getCredentials round-trips the pair', async () => {
  await setCredentials('nick', 'jwt-abc');
  await expect(getCredentials()).resolves.toEqual({
    username: 'nick',
    token: 'jwt-abc',
  });
});

test('deleteCredentials removes the pair', async () => {
  await setCredentials('nick', 'jwt-abc');
  await deleteCredentials();
  await expect(getCredentials()).resolves.toBeNull();
});

test('keychainGet reads the token and username by key', async () => {
  await setCredentials('nick', 'jwt-abc');
  await expect(keychainGet('token')).resolves.toBe('jwt-abc');
  await expect(keychainGet('username')).resolves.toBe('nick');
  await expect(keychainGet('other')).resolves.toBeNull();
});

test('keychainSet("token") preserves the stored username', async () => {
  await setCredentials('nick', 'old-jwt');
  await keychainSet('token', 'new-jwt');
  await expect(getCredentials()).resolves.toEqual({
    username: 'nick',
    token: 'new-jwt',
  });
});

test('keychainSet("username") preserves the stored token', async () => {
  await setCredentials('nick', 'jwt-abc');
  await keychainSet('username', 'sara');
  await expect(getCredentials()).resolves.toEqual({
    username: 'sara',
    token: 'jwt-abc',
  });
});

test('keychainSet on an empty store creates the pair with the other field empty', async () => {
  await keychainSet('token', 'first-jwt');
  await expect(getCredentials()).resolves.toEqual({
    username: '',
    token: 'first-jwt',
  });
});

test('keychainDelete clears the service entry', async () => {
  await setCredentials('nick', 'jwt');
  await keychainDelete();
  await expect(
    Keychain.getGenericPassword({ service: AUTH_SERVICE }),
  ).resolves.toBe(false);
});
