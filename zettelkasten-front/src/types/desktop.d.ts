export {};

/**
 * Tauri desktop bridge, installed by desktop/src-tauri/preload.js. Absent on
 * plain web builds — everything here is optional at the type level so web
 * code compiles without a desktop shell.
 */
declare global {
  interface Window {
    zgDesktop?: {
      /** Resolves once the keychain has been read into the shim cache. */
      ready: Promise<unknown>;
      platform: string;
      getToken: () => Promise<string | null>;
      setToken: (token: string) => void;
      loadSettings: () => Promise<{ serverUrl?: string }>;
      saveSettings: (settings: unknown) => Promise<void>;
      windowControls: {
        minimize: () => Promise<void>;
        maximize: () => Promise<void>;
        close: () => Promise<void>;
        isMaximized: () => Promise<boolean>;
      };
      getSyncStatus: () => { online: boolean; pendingChanges: number };
      onSyncStatus: (cb: () => void) => () => void;
    };
  }
}
