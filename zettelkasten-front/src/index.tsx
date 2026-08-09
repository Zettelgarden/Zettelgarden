import React from 'react';
import ReactDOM from 'react-dom/client';
import './index.css';
import App from './App';
import { AuthProvider } from './contexts/AuthContext';
import { SettingsProvider } from './contexts/SettingsContext';
import { SyncProvider } from './contexts/SyncContext';
import { HashRouter } from 'react-router-dom';
import { Buffer } from 'buffer';
import { getSyncClient } from './data/syncClient';
import { SyncDataProvider } from './data/syncProvider';
import { registerSyncProvider } from './data/provider';
globalThis.Buffer = Buffer;

const rootElement = document.getElementById('root');

/**
 * In the desktop app the local-first sync provider is registered BEFORE the
 * first render, so the very first data queries hit the local mirror (no
 * HTTP-fallback window — the app opens instantly offline). The web app has
 * no engine: getSyncClient() resolves null immediately and rendering is
 * synchronous.
 */
async function boot() {
  try {
    const client = await getSyncClient();
    if (client) {
      registerSyncProvider(new SyncDataProvider(client.engine));
    }
  } catch (err) {
    // Engine init failure (e.g. mirror DB unavailable): fall back to the
    // HTTP thin-client behavior rather than blocking the app.
    console.error('sync client init failed:', err);
  }

  if (rootElement) {
    const root = ReactDOM.createRoot(rootElement);
    root.render(
      <React.StrictMode>
        <HashRouter>
          <SyncProvider>
            <SettingsProvider>
              <AuthProvider>
                <App />
              </AuthProvider>
            </SettingsProvider>
          </SyncProvider>
        </HashRouter>
      </React.StrictMode>,
    );
  } else {
    console.error('Root element not found');
  }
}

void boot();
