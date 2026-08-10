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
       * settings and keychain (c6l.3) use the typed helpers below.
       */
      invoke: (cmd: string, args?: unknown) => Promise<unknown>;
      /** Non-secret shell settings (server URL, account). */
      loadSettings: () => Promise<{ serverUrl?: string }>;
      saveSettings: (settings: unknown) => Promise<void>;
      /**
       * Keychain-backed token helpers (c6l.3). The web app normally reads
       * the token via localStorage.getItem('token') — the shim redirects
       * that to the keychain — so these exist for shell-level callers.
       */
      getToken: () => Promise<string | null>;
      setToken: (token: string) => void;
      deleteToken: () => void;
    };
  }
}
