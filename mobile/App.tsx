/**
 * Zettelgarden Mobile — React Native WebView shell (epic Zettelgarden-v5b,
 * Phase 3a — issues c6l.1 / c6l.4).
 *
 * Hosts the existing zettelkasten-front Vite build (the same responsive web
 * UI the desktop Tauri shell wraps). The sync engine runs inside the WebView;
 * c6l.2 adds the WebView↔SQLite bridge (op-sqlite), c6l.3 the keychain token
 * commands. c6l.4 ships the shell bridge + server-URL settings surface.
 *
 * Dev default: the Android emulator reaches the host machine via 10.0.2.2;
 * a user-configured server URL (gear → Settings) overrides it at runtime
 * through window.zgMobile.loadSettings → resolveBaseUrl.
 */

import React, { useCallback, useRef, useState } from 'react';
import {
  ActivityIndicator,
  Modal,
  Platform,
  Pressable,
  StatusBar,
  StyleSheet,
  Text,
  TextInput,
  View,
} from 'react-native';
import { SafeAreaView } from 'react-native-safe-area-context';
import {
  WebView,
  type WebViewMessageEvent,
  type WebViewProps,
} from 'react-native-webview';
import { shimSource } from './webviewShim';
import { handleBridgeMessage, responseScript } from './src/bridge';
import { loadSettings, saveSettings } from './src/settingsStore';

// Dev default: Android emulator → host loopback. Overridden by settings.
const DEFAULT_SERVER_URL = 'http://10.0.2.2:5173';

const styles = StyleSheet.create({
  container: {
    flex: 1,
    backgroundColor: '#fff',
  },
  webview: {
    flex: 1,
  },
  loading: {
    position: 'absolute',
    top: 0,
    left: 0,
    right: 0,
    bottom: 0,
    alignItems: 'center',
    justifyContent: 'center',
    backgroundColor: '#fff',
  },
  error: {
    padding: 24,
    textAlign: 'center',
    color: '#666',
  },
  gear: {
    position: 'absolute',
    right: 16,
    bottom: 48,
    width: 44,
    height: 44,
    borderRadius: 22,
    backgroundColor: 'rgba(30,30,30,0.75)',
    alignItems: 'center',
    justifyContent: 'center',
    zIndex: 10,
  },
  gearText: {
    color: '#fff',
    fontSize: 20,
  },
  modalBackdrop: {
    flex: 1,
    backgroundColor: 'rgba(0,0,0,0.4)',
    alignItems: 'center',
    justifyContent: 'center',
  },
  modalCard: {
    width: '85%',
    backgroundColor: '#fff',
    borderRadius: 12,
    padding: 20,
  },
  modalTitle: {
    fontSize: 17,
    fontWeight: '600',
    marginBottom: 12,
  },
  input: {
    borderWidth: 1,
    borderColor: '#ccc',
    borderRadius: 8,
    paddingHorizontal: 12,
    paddingVertical: 10,
    fontSize: 15,
  },
  modalActions: {
    flexDirection: 'row',
    justifyContent: 'flex-end',
    marginTop: 16,
    gap: 8,
  },
  btn: {
    paddingHorizontal: 16,
    paddingVertical: 8,
    borderRadius: 8,
  },
  btnPrimary: {
    backgroundColor: '#2563eb',
  },
  btnPrimaryText: {
    color: '#fff',
  },
});

function App() {
  // react-native-webview v14 exposes instance commands (injectJavaScript) on
  // the ref at runtime, but its public types omit the forwarded ref under
  // React 19 — typed as any (documented lib gap).
  const webViewRef = useRef<any>(null);
  const [settingsOpen, setSettingsOpen] = useState(false);
  const [serverUrl, setServerUrl] = useState('');

  const openSettings = useCallback(async () => {
    const settings = await loadSettings();
    setServerUrl(settings.serverUrl ?? DEFAULT_SERVER_URL);
    setSettingsOpen(true);
  }, []);

  const persistSettings = useCallback(async () => {
    await saveSettings({ serverUrl: serverUrl.trim() });
    setSettingsOpen(false);
    // The webview re-reads settings lazily (resolveBaseUrl on engine init /
    // next shell boot); no reload needed for the settings contract itself.
  }, [serverUrl]);

  const onMessage = useCallback((event: WebViewMessageEvent) => {
    handleBridgeMessage(event.nativeEvent.data).then(resp => {
      webViewRef.current?.injectJavaScript(responseScript(resp));
    });
  }, []);

  // Launch health markers for the E2E smoke (mobile/e2e/smoke.sh): the RN
  // runtime boot, the WebView navigation, and (from inside the webview) the
  // shim install + bridge handshake appear in logcat.
  const onLoadStart = useCallback((event: { nativeEvent: { url: string } }) => {
    console.log('[zg-mobile] onLoadStart', event.nativeEvent.url);
  }, []);
  const onLoadEnd = useCallback((event: { nativeEvent: { url: string } }) => {
    console.log('[zg-mobile] onLoadEnd', event.nativeEvent.url);
  }, []);

  const webViewProps: WebViewProps = {
    source: { uri: DEFAULT_SERVER_URL },
    style: styles.webview,
    javaScriptEnabled: true,
    domStorageEnabled: true,
    setSupportMultipleWindows: false,
    originWhitelist: ['*'],
    injectedJavaScriptBeforeContentLoaded: shimSource(Platform.OS),
    onMessage,
    onLoadStart,
    onLoadEnd,
    startInLoadingState: true,
    renderLoading: () => (
      <View style={styles.loading}>
        <ActivityIndicator size="large" />
      </View>
    ),
    renderError: errorName => (
      <View style={styles.loading}>
        <Text style={styles.error}>
          Could not load Zettelgarden: {errorName}
          {'\n\n'}
          Start the web dev server, or tap the gear to set a server URL.
        </Text>
      </View>
    ),
  };

  return (
    <SafeAreaView style={styles.container}>
      <StatusBar barStyle="dark-content" />
      <WebView {...webViewProps} ref={webViewRef} />

      <Pressable style={styles.gear} onPress={openSettings}>
        <Text style={styles.gearText}>⚙</Text>
      </Pressable>

      <Modal
        visible={settingsOpen}
        transparent
        animationType="fade"
        onRequestClose={() => setSettingsOpen(false)}
      >
        <View style={styles.modalBackdrop}>
          <View style={styles.modalCard}>
            <Text style={styles.modalTitle}>Server URL</Text>
            <TextInput
              style={styles.input}
              value={serverUrl}
              onChangeText={setServerUrl}
              placeholder="https://zg.example.com"
              autoCapitalize="none"
              autoCorrect={false}
              keyboardType="url"
            />
            <View style={styles.modalActions}>
              <Pressable
                style={styles.btn}
                onPress={() => setSettingsOpen(false)}
              >
                <Text>Cancel</Text>
              </Pressable>
              <Pressable
                style={[styles.btn, styles.btnPrimary]}
                onPress={persistSettings}
              >
                <Text style={styles.btnPrimaryText}>Save</Text>
              </Pressable>
            </View>
          </View>
        </View>
      </Modal>
    </SafeAreaView>
  );
}

export default App;
