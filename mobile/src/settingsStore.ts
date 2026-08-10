/**
 * Non-secret shell settings for the mobile app (Zettelgarden-c6l.4). The
 * server URL lives in AsyncStorage and is exposed to the webview via
 * window.zgMobile.loadSettings(); the sync transport reads it through
 * resolveBaseUrl (zettelkasten-front/src/data/syncClient.ts). Keychain
 * secrets are NOT stored here — they land with c6l.3.
 */

import AsyncStorage from '@react-native-async-storage/async-storage';

export const SERVER_URL_KEY = 'zgServerUrl';

export interface ShellSettings {
  serverUrl?: string;
}

export async function loadSettings(): Promise<ShellSettings> {
  const serverUrl = await AsyncStorage.getItem(SERVER_URL_KEY);
  return serverUrl ? { serverUrl } : {};
}

export async function saveSettings(settings: ShellSettings): Promise<void> {
  if (settings.serverUrl !== undefined) {
    await AsyncStorage.setItem(SERVER_URL_KEY, settings.serverUrl);
  }
}
