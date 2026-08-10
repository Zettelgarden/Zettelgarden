/**
 * Native keychain store for the mobile shell (Zettelgarden-c6l.3). The JWT
 * (and the non-secret username) live in the OS keychain via
 * react-native-keychain — never in WebView localStorage. The webview's
 * localStorage shim (webviewShim.js) redirects the token/username keys here
 * over the bridge, mirroring the desktop preload.js → Rust keyring pattern.
 *
 * react-native-keychain stores ONE (username, password) pair per service:
 * username = account name, password = JWT.
 */

import * as Keychain from 'react-native-keychain';

export const AUTH_SERVICE = 'com.zettelgarden.auth';

export interface AuthCredentials {
  username: string;
  token: string;
}

export async function getCredentials(): Promise<AuthCredentials | null> {
  const creds = await Keychain.getGenericPassword({ service: AUTH_SERVICE });
  if (!creds) return null;
  return { username: creds.username, token: creds.password };
}

export async function setCredentials(
  username: string,
  token: string,
): Promise<void> {
  await Keychain.setGenericPassword(username, token, { service: AUTH_SERVICE });
}

export async function deleteCredentials(): Promise<void> {
  await Keychain.resetGenericPassword({ service: AUTH_SERVICE });
}

/** Key-based access matching the desktop shim's keychain keys. */
export async function keychainGet(key: string): Promise<string | null> {
  const creds = await getCredentials();
  if (!creds) return null;
  if (key === 'username') return creds.username;
  if (key === 'token') return creds.token;
  return null;
}

/** Merging write: setting one key preserves the other. */
export async function keychainSet(key: string, value: string): Promise<void> {
  const current = (await getCredentials()) ?? { username: '', token: '' };
  if (key === 'username') current.username = value;
  else if (key === 'token') current.token = value;
  await setCredentials(current.username, current.token);
}

export async function keychainDelete(): Promise<void> {
  await deleteCredentials();
}
