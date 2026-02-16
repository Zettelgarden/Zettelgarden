# Sidebar RSS Feed Quick Add Design

**Date:** 2026-02-16
**Author:** Claude
**Status:** Approved

## Overview

Add a 5th quick action button to the Sidebar that allows users to add RSS feeds directly from anywhere in the application, without navigating to the RSS page.

## Problem

Users can currently only add RSS feeds by navigating to the RSS page. For a quick action like adding a feed, this is unnecessarily frictionful. The Sidebar already has quick actions for cards, tasks, and chat - adding feeds should be equally accessible.

## Solution

Add a 5th quick action button to `SidebarHeader` that opens the existing `RssAddFeedDialog` component.

## Architecture

### Component Changes

#### Sidebar.tsx
- Add state: `const [showAddFeedDialog, setShowAddFeedDialog] = useState(false);`
- Add handler: `handleAddFeed()` that sets the state to true
- Pass `onAddFeed` prop to `SidebarHeader`
- Render `RssAddFeedDialog` component with:
  - `isOpen={showAddFeedDialog}`
  - `onClose={() => setShowAddFeedDialog(false)}`
  - `folders={folders}` (need to fetch from RSS context)
  - `onFeedAdded={handleFeedAdded}` (handles the new feed)
- Import `RssAddFeedDialog` from `../components/rss/RssAddFeedDialog`

#### SidebarHeader.tsx
- Add new prop: `onAddFeed?: () => void`
- Add 5th button in the quick actions grid (when `!isCollapsed`)
- Use appropriate RSS/feed icon

### Data Flow

1. User clicks "Add Feed" button in Sidebar
2. `RssAddFeedDialog` opens
3. User enters feed URL and selects folder
4. Dialog calls API to add feed
5. On success, `onFeedAdded` callback is invoked
6. Dialog closes
7. **No toast notification** (silent success)
8. **No navigation** - user stays where they are

### RSS Context Integration

The Sidebar needs access to:
- `folders` list - to pass to `RssAddFeedDialog` for folder selection dropdown
- Optional: `refreshFeeds` or similar - if we want to refresh the RSS context after adding

Looking at `useRSS` and `useRssData`, we may need to expose folders from the RSS context or fetch them directly.

## UI Design

### Button Appearance
- 5th button in the quick actions grid
- Visible only when sidebar is expanded
- Uses RSS/feed icon (e.g., from heroicons or lucide-react)
- Consistent styling with existing quick action buttons

### Collapsed State
- When sidebar is collapsed, the button is hidden (same as other quick actions)

## Error Handling

- API errors are handled internally by `RssAddFeedDialog`
- No additional error handling needed in Sidebar
- Dialog displays error messages for:
  - Invalid URLs
  - Duplicate feeds
  - Network failures

## Edge Cases

1. **No folders exist** - Dialog handles this (allows no folder selection)
2. **Duplicate feed URL** - API handles this, dialog shows error
3. **User cancels** - Dialog closes without side effects
4. **Mobile sidebar** - Button works the same way
5. **Collapsed sidebar** - Button is hidden (expected behavior)

## Testing Checklist

1. ✅ Click "Add Feed" button opens dialog
2. ✅ Cancel button closes dialog
3. ✅ Successfully add a feed
4. ✅ Error displayed for invalid URL
5. ✅ Error displayed for duplicate feed
6. ✅ Button hidden when sidebar collapsed
7. ✅ Works on mobile
8. ✅ Folder dropdown populated correctly

## Implementation Notes

- RSS is **NOT** a PRO feature - no subscription gating needed
- The `RssAddFeedDialog` component is already well-tested from RSS page usage
- Changes are localized to Sidebar components - minimal risk
