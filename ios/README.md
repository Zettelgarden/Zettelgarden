# Zettelgarden iOS Workspace

This folder contains the iOS host app scaffold plus a share extension that lets you send links or text snippets to Zettelgarden.

## Project generation

The project is defined via [XcodeGen](https://github.com/yonaskolb/XcodeGen) so it can stay in source control cleanly. To create the Xcode workspace:

```sh
brew install xcodegen   # if you don't already have it
cd ios
xcodegen generate
open Zettelgarden.xcodeproj
```

Regenerate the project whenever you edit `project.yml`.

## Targets

- **ZettelgardenApp** – a SwiftUI host app that stores API credentials and default options in the shared app group. The initial screen lets you paste the API base URL (e.g. `https://your-domain`), supply an API token, and choose whether cards or tasks should be the default destination.
- **ZettelgardenShareExtension** – the action extension that appears in the iOS share sheet. It reads the configuration saved by the host app, pre-fills content from the shared item, and POSTs to the backend.
- **ZettelgardenShared** – a lightweight static framework for code shared between the app and the extension (API client, models, configuration stores).

## Configuration

1. **Bundle IDs & App Group** – Update `project.yml` (search for `com.zettelgarden`) with your real bundle identifier prefix. The shared App Group defaults to `group.com.zettelgarden` (see `AppGroups.swift` and the entitlements). Create the group in the Apple Developer portal and adjust the string if necessary.
2. **API Auth** – The host app persists the base URL and bearer token to the app group `UserDefaults`. The share extension reads this at runtime, so the extension works offline after you configure it once.
3. **Backend Contract** – `ZettelgardenAPI` POSTs to `api/v1/{cards|tasks}` with a JSON payload containing `title`, `body`, and an optional `sourceURL`. Adjust `SharePayload` if your backend expects a different schema.

## Running locally

- You can run the app target directly in the simulator to test configuration persistence.
- To test the share extension in the simulator, build `ZettelgardenApp`, then from the extension scheme choose a host app such as Safari. Xcode will prompt you to select a host the first time you run it.

## Next steps

- Wire up richer validation and success UI in the host app.
- Persist multiple tokens or environments if you need staging vs production.
- Extend `ShareExtensionViewModel` to support attachments like images once the backend exposes an upload endpoint.
