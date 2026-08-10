/**
 * Settings store tests (Zettelgarden-c6l.4): server URL round-trips through
 * AsyncStorage (the in-memory __mocks__ copy).
 */

import AsyncStorage from '@react-native-async-storage/async-storage';
import {
  loadSettings,
  saveSettings,
  SERVER_URL_KEY,
} from '../src/settingsStore';

beforeEach(async () => {
  await AsyncStorage.clear();
});

test('loadSettings returns {} when nothing is stored', async () => {
  await expect(loadSettings()).resolves.toEqual({});
});

test('saveSettings persists the server URL and loadSettings reads it back', async () => {
  await saveSettings({ serverUrl: 'https://notes.example.com/api' });
  await expect(loadSettings()).resolves.toEqual({
    serverUrl: 'https://notes.example.com/api',
  });
  await expect(AsyncStorage.getItem(SERVER_URL_KEY)).resolves.toBe(
    'https://notes.example.com/api',
  );
});

test('saveSettings ignores undefined fields', async () => {
  await saveSettings({});
  await expect(AsyncStorage.getItem(SERVER_URL_KEY)).resolves.toBeNull();
});
