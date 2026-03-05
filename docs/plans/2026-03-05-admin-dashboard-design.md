# Admin Dashboard Design

**Date:** 2026-03-05
**Status:** Approved

## Overview
Create a new dashboard view at `/admin` that shows the 10 most recently active users (by `last_seen`). Move the full paginated user list to `/admin/users`.

## Components

### New `AdminDashboard.tsx` component
- Fetch all users using existing `getUsers()` API
- Sort client-side by `last_seen` descending (handle `null` values)
- Slice to top 10
- Reuse the existing `AdminTableContainer` component for consistency
- Display the same full columns as `AdminUserIndex`: ID, name, last seen, email validated, subscription, created at, cards, tasks, files, chats, revenue, cost
- Show a note like "Showing top 10 of X total users"

### Update `AdminPage.tsx` routing
- Route `/admin` → `AdminDashboard` (new)
- Route `/admin/users` → `AdminUserIndex` (current)
- Update sidebar: "Dashboard" and "All Users" as separate links

### Update sidebar navigation
- Add "Dashboard" link pointing to `/admin`
- Change "Users" to "All Users" pointing to `/admin/users`

## Data flow
```
AdminDashboard mounts
  → calls getUsers()
  → receives all users (paginated, may need multiple calls or increase per_page)
  → sorts by last_seen DESC
  → takes first 10
  → renders table
```

## Edge cases handled
- Users with `null` last_seen are sorted to the end
- If fewer than 10 users, show all available
- Loading and error states reused from existing pattern
