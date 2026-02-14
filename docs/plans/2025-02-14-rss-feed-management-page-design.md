# RSS Feed Management Page Design

**Date:** 2025-02-14
**Status:** Approved
**Author:** Claude

## Overview

A dedicated feed management page at `/app/rss/manage` providing at-a-glance overview and bulk operations for RSS feeds and folders.

## Goals

1. **At-a-glance overview:** See all feeds with key status info in a sortable table
2. **Bulk operations:** Enable/disable, delete, move, and tag multiple feeds at once
3. **Folder management:** Centralized place to manage folders alongside feeds

## Page Structure

### Route & Navigation

- **Route:** `/app/rss/manage`
- **Access:** From RSS reader settings menu → "Manage Feeds"
- **Breadcrumbs:** "RSS → Manage Feeds"
- **Back button:** Returns to RSS reader

### Layout

```
┌─────────────────────────────────────────────────────────────┐
│  RSS → Manage Feeds                    [Add Feed] [Refresh] │
├──────────────┬──────────────────────────────────────────────┤
│              │ ┌──────────────────────────────────────────┐ │
│  Folders     │ │ [Bulk Actions: Enable | Disable | Delete] │ │
│              │ ├──────────────────────────────────────────┤ │
│ □ All Feeds  │ │ ┌───┬──────┬─────┬────┬────┬─────┐      │ │
│ ▼ Tech       │ │ │ ☐ │ Name │ URL │ ...│    │     │      │ │
│ ▶ News       │ │ ├───┼──────┼─────┼────┼────┼─────┤      │ │
│ ▶ Science    │ │ │ ☑ │ Feed1│ ... │    │    │     │      │ │
│              │ │ │ ☑ │ Feed2│ ... │    │    │     │      │ │
│ [Create... ] │ │ └───┴──────┴─────┴────┴────┴─────┘      │ │
│              │ └──────────────────────────────────────────┘ │
└──────────────┴──────────────────────────────────────────────┘
```

## Left Panel: Folder Management

**Components:**
- "All Feeds" option (default, shows all feeds)
- Folder list with:
  - Folder name
  - Feed count badge
  - Rename action
  - Delete action (with confirmation)
- "Create Folder" button (bottom)

**Interactions:**
- Click folder → Filter feeds table to that folder
- Click "All Feeds" → Show all feeds
- Delete folder → Prompt: "Move feeds to uncategorized" OR "Delete all feeds in folder"

## Right Panel: Feeds Table

**Columns:**

| Column | Description |
|--------|-------------|
| Checkbox | For bulk selection |
| Name | Feed name + favicon |
| URL | Feed URL (truncated, hover tooltip) |
| Folder | Folder badge |
| Tags | Comma-separated tags (truncated) |
| Status | Enabled ✓/disabled icon, priority star |
| Last Fetched | Relative time (e.g., "2h ago", "Never") |
| Error Status | Icon + tooltip if last_error exists |
| Unread Count | Number of unread articles |
| Actions | Edit, delete, refresh single feed |

**Features:**
- Sortable by any column
- Search/filter by name or URL
- Pagination (25, 50, 100 per page)
- Sticky header

**Bulk Actions Toolbar** (appears when items selected):
- Enable / Disable
- Move to folder → (dropdown selector)
- Set tags → (input with replace/append options)
- Delete

## Dialogs

### Feed Edit Dialog
Reuse `RssEditFeedDialog` with:
- All current fields (name, URL, folder, auto-tags, fetch interval, enabled, priority)
- Add "Delete this feed" button

### Folder Rename Dialog
Reuse `RssEditFolderDialog`:
- Rename input
- Auto-update feeds' folder field

### Bulk Move to Folder Dialog
- Folder dropdown
- "Create new folder" option
- Feed count display

### Bulk Set Tags Dialog
- Tag input field
- Radio: Replace / Append to existing

### Bulk Delete Confirmation
- Feed count
- Warning about article deletion
- Cannot be undone warning

## Data Flow

### Loading
- Parallel fetch: feeds, folders, unread counts
- Skeleton loaders during fetch
- Reuse existing API: `listFeeds()`, `listFolders()`, `getUnreadCounts()`

### Updates
- Optimistic UI updates with rollback on error
- Refresh unread counts after mutations
- Toast notifications for results

### Error Handling
- Toast notifications for operation results
- Inline error indicators (last_error field)
- Retry buttons on failures

### Refresh
- Manual refresh button in header
- Auto-refresh counts after mutations
- Optional: Polling for fetch status

## Edge Cases

### Empty States
- **No feeds:** "No feeds yet. Add your first feed to get started."
- **No folders:** Only "All Feeds" shown
- **Folder empty:** "No feeds in this folder."
- **No search results:** "No feeds match your search."

### Edge Cases
- Single feed: Hide bulk actions, show individual only
- All selected: Show "Select all (N)" in header checkbox
- Deleted folder selected: Auto-select "All Feeds"
- Concurrent edits: Last write wins, warn if stale

### Mobile Responsive
- Stack panels vertically (folders top, feeds bottom)
- Collapse columns on small screens
- Expandable rows for hidden columns

## Components to Create

| Component | Purpose |
|-----------|---------|
| `RssManagePage.tsx` | Main page component |
| `RssManageFolderPanel.tsx` | Left folder sidebar |
| `RssManageFeedsTable.tsx` | Feeds table with bulk actions |
| `RssBulkMoveDialog.tsx` | Bulk move to folder |
| `RssBulkTagsDialog.tsx` | Bulk set tags |
| `RssBulkDeleteDialog.tsx` | Bulk delete confirmation |

## Route Addition

Add to `App.tsx`:
```tsx
<Route path="/app/rss/manage" element={<RssManagePage />} />
```

## API Usage (Existing)

All required API functions already exist in `src/api/rss.ts`:
- `listFeeds()`, `updateFeed()`, `deleteFeed()`
- `listFolders()`, `updateFolder()`, `deleteFolder()`, `createFolder()`
- `markFeedAsRead()`, `refreshFeeds()`
- `getUnreadCounts()`
