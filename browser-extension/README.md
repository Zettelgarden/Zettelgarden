# Zettelgarden Browser Extension

Cross-browser extension for creating Zettelgarden cards directly from your browser.

## Features

- **Quick Card**: Create a card with custom title and content
- **Article from Page**: Extract clean article content using Readability.js (bypasses paywalls!)
- **Authentication**: Secure JWT token storage
- **Multi-browser**: Works on both Firefox and Chrome

## Architecture

### Shared Code (`shared/`)
Core functionality that works across both browsers:

- **api/**: API client for Zettelgarden backend
  - `auth.js`: JWT authentication and token management
  - `cards.js`: Card creation (`POST /api/cards`)
  - `config.js`: Server URL configuration

- **ui/**: User interface
  - `popup.html`: Main popup interface
  - `popup.js`: Popup logic and event handling
  - `popup.css`: Styling

- **content/**: Content scripts
  - `readability.js`: Mozilla's Readability library for article extraction
  - `extractor.js`: Injects Readability and extracts article content from current page

- **background.js**: Background service worker for API calls

### Browser-Specific (`firefox/`, `chrome/`)
Only the manifest files differ:
- `firefox/manifest.json`: Firefox manifest (v2 or v3)
- `chrome/manifest.json`: Chrome manifest v3

## Article Extraction Flow

1. User clicks "Article from Page" button
2. Extension injects `extractor.js` content script into current tab
3. Content script runs Readability.js on page DOM (client-side parsing)
4. Extracts: title, author, clean content (HTML → markdown conversion)
5. Sends extracted content back to popup
6. User previews/edits, then submits to Zettelgarden API

**Why Readability.js?**
- ✅ Client-side parsing (works on paywalled content)
- ✅ Same library Firefox Reader Mode uses
- ✅ Works identically on Chrome
- ✅ Produces clean, readable content

## Setup

1. Configure your Zettelgarden server URL in extension settings
2. Log in with your credentials (stores JWT token securely)
3. Click extension icon to create cards

## Development

Build for Firefox:
```bash
cd firefox && web-ext build --source-dir=../shared
```

Build for Chrome:
```bash
# Load `shared/` + `chrome/manifest.json` as unpacked extension
```

## API Requirements

Extension requires these Zettelgarden API endpoints:
- `POST /api/login` - Authentication
- `POST /api/cards` - Card creation (JWT protected)
- `GET /api/auth` - Token validation (JWT protected)