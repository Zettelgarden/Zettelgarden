export {};

/**
 * Mobile (React Native WebView) bridge, installed by mobile/webviewShim.js
 * before the web app loads. Absent on plain web builds — everything here is
 * optional at the type level so web code compiles without a mobile shell.
 *
 * The SQLite commands (c6l.2) and the keychain token commands (c6l.3) extend
 * this surface; c6l.4 ships the settings + shell contract only.
 */
declare global {
  interface Window {
    zgMobile?: {
      /** Resolves once the RN-side bridge has primed (settings read). */
      ready: Promise<unknown>;
      platform: 'android' | 'ios';
      /**
       * Generic request/response bridge call. The sync engine's
       * MobileStorageAdapter (c6l.2) drives sql_* commands through this;
       * settings and (c6l.3) keychain use the typed helpers.
       */
      invoke: (cmd: string, args?: unknown) => Promise<unknown>;
      /** Non-secret shell settings (server URL, account). */
      loadSettings: () => Promise<{ serverUrl?: string }>;
      saveSettings: (settings: unknown) => Promise<void>;
    };
  }
}
