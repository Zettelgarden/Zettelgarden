import React from 'react';
import ReactDOM from 'react-dom/client';
import './index.css';
import App from './App';
import { AuthProvider } from './contexts/AuthContext';
import { SettingsProvider } from './contexts/SettingsContext';
import { SyncProvider } from './contexts/SyncContext';
import { HashRouter } from 'react-router-dom';
import { Buffer } from 'buffer';
globalThis.Buffer = Buffer;

const rootElement = document.getElementById('root');

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
