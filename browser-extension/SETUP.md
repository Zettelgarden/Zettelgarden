# Setup Instructions

## Prerequisites

1. Download Mozilla's Readability.js library:
   ```bash
   cd browser-extension/shared/content
   curl -O https://raw.githubusercontent.com/mozilla/readability/main/Readability.js
   ```

## Build

Run the build script to copy shared files into browser directories:
```bash
cd browser-extension
./build.sh
```

## Firefox Setup

### Development
1. Open Firefox and navigate to `about:debugging#/runtime/this-firefox`
2. Click "Load Temporary Add-on"
3. Select the `firefox/manifest.json` file
4. Reload the extension after making changes to shared files (run `./build.sh` again)

### Production Build
```bash
cd firefox
web-ext build --source-dir=. --artifacts-dir=../dist
```

## Chrome Setup

### Development
1. Open Chrome and navigate to `chrome://extensions/`
2. Enable "Developer mode"
3. Click "Load unpacked"
4. Select the `chrome/` folder

### Production Build
1. Zip the contents: `chrome/` folder + `shared/` folder
2. Upload to Chrome Web Store

## Usage

1. Click the extension icon
2. Enter your Zettelgarden server URL (e.g., `https://zettelgarden.example.com`)
3. Login with your email/password
4. Choose:
   - **Quick Card**: Manual title/content entry
   - **Article**: Extract content from current page using Readability

## Testing Article Extraction

Works best on article pages:
- News sites
- Blogs
- Medium posts
- Documentation pages

The extension will parse the page content client-side, so it works even on paywalled content!