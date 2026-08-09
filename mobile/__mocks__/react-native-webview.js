/**
 * Jest mock for react-native-webview: the native module is unavailable in
 * the jest environment. Render a plain View so App-level tests can mount.
 */

import React from 'react';
import { View } from 'react-native';

const WebView = (props) => <View {...props} testID="webview" />;

WebView.postMessage = jest.fn();

module.exports = {
  WebView,
  // Keep the named export shape used by App.tsx (`type WebViewProps` is a
  // type-only import and vanishes at runtime).
  __esModule: true,
};
