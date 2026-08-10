'use strict';

/**
 * Webview shim (Zettelgarden-c6l.3): re-exports the injected script source
 * (generated from webviewShimSource.js — see that file for the Hermes
 * rationale) and the createShim factory (used by unit tests).
 *
 * App.tsx consumes shimSource(Platform.OS) for
 * injectedJavaScriptBeforeContentLoaded.
 */

const { createShim } = require('./webviewShimSource');
const shimSource = require('./shimSource.generated.js');

module.exports = { createShim, shimSource };
