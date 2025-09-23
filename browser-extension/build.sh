#!/bin/bash

# Build script for browser extensions

# Build Firefox
echo "Building Firefox extension..."
rm -rf firefox/api firefox/content firefox/ui firefox/background.js
cp -r shared/* firefox/

# Build Chrome
echo "Building Chrome extension..."
rm -rf chrome/api chrome/content chrome/ui chrome/background.js
cp -r shared/* chrome/

echo "Done! Extensions built in firefox/ and chrome/ directories"