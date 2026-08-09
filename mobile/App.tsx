/**
 * Zettelgarden Mobile — React Native WebView shell (epic Zettelgarden-v5b,
 * Phase 3a — issue c6l.1).
 *
 * Hosts the existing zettelkasten-front Vite build (the same responsive web
 * UI the desktop Tauri shell wraps). The sync engine runs inside the WebView;
 * Phase 3a follow-ons add the WebView↔native bridge (c6l.2), keychain auth
 * (c6l.3), and shell detection + server-URL config (c6l.4).
 *
 * Dev default: the Android emulator reaches the host machine via 10.0.2.2.
 * A real configurable server URL lands with c6l.4 (settings surface).
 */

import React from 'react';
import {
  ActivityIndicator,
  SafeAreaView,
  StatusBar,
  StyleSheet,
  Text,
  View,
} from 'react-native';
import { WebView, type WebViewProps } from 'react-native-webview';

// Override at build time with the Vite dev server / bundled dist origin.
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
});

function App() {
  const webViewProps: WebViewProps = {
    source: { uri: DEFAULT_SERVER_URL },
    style: styles.webview,
    javaScriptEnabled: true,
    domStorageEnabled: true,
    setSupportMultipleWindows: false,
    originWhitelist: ['*'],
    startInLoadingState: true,
    renderLoading: () => (
      <View style={styles.loading}>
        <ActivityIndicator size="large" />
      </View>
    ),
    renderError: (errorName) => (
      <View style={styles.loading}>
        <Text style={styles.error}>
          Could not load Zettelgarden: {errorName}
          {'\n\n'}
          Start the web dev server or configure a server URL (c6l.4).
        </Text>
      </View>
    ),
  };

  return (
    <SafeAreaView style={styles.container}>
      <StatusBar barStyle="dark-content" />
      <WebView {...webViewProps} />
    </SafeAreaView>
  );
}

export default App;
