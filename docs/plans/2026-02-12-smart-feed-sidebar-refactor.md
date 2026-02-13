# Smart Feed Sidebar Refactor Design

**Date**: 2026-02-12
**Status**: Design Approved
**Author**: Design collaboration via brainstorming

## Overview

Refactor the Smart Feed feature from a toggle option in the articles panel to a first-class feed item in the sidebar, and add age-based scoring to prioritize recent content.

## Problem Statement

The current Smart Feed implementation has UX issues:
1. **Buried toggle**: Smart Feed is hidden as a third tab option ("All/Unread/Smart"), not discoverable
2. **Stale content**: Smart feed includes very old articles without recency weighting
3. **Static list desire**: Users want the smart feed to remain stable throughout the day (articles shouldn't disappear when viewed)

## Goals

- Make Smart Feed a prominent, always-accessible option
- Add time-based decay so recent articles are prioritized
- Keep the smart feed list stable (show both read and unread articles)
- 14-day hard cutoff to rotate out old content

## Design Changes

### 1. Sidebar Changes

**Before:**
```
┌─────────────────────┐
│ All Feeds (12)      │
│ ├─ 📁 Tech (5)     │
│ └─ 📁 News (7)     │
│ [Unread only?]      │
└─────────────────────┘
```

**After:**
```
┌─────────────────────┐
│ ⭐ Smart Feed      │  ← NEW, at top, no badge
│ ├─ All Feeds (12)  │
│ ├─ 📁 Tech (5)     │
│ └─ 📁 News (7)     │
│ [Unread only?]      │
└─────────────────────┘
```

**Visual Design:**
- Icon: Star/sparkle (same as current article panel)
- Text: "Smart Feed"
- No badge (no unread/total count)
- Highlighted when active (blue background like other selections)
- Position: First item, before "All Feeds"

### 2. Articles Panel Changes

**Before:**
```
┌─────────────────────────────────┐
│ [All]  [Unread (12)] [Smart] │  ← 3 tabs
└─────────────────────────────────┘
```

**After:**
```
┌─────────────────────────────────┐
│ [All]  [Unread (12)]         │  ← 2 tabs only
└─────────────────────────────────┘

When Smart Feed active:
┌─────────────────────────────────┐
│ Smart Feed                    │  ← Header only, no tabs
└─────────────────────────────────┘
```

**Key Behaviors:**
- When Smart Feed selected: No filter tabs shown
- When regular feed/folder selected: "All" and "Unread" tabs work normally
- Smart Feed shows BOTH read AND unread articles (static list)
- `showUnreadOnly` state is ignored when Smart Feed is active

### 3. Backend Scoring Changes

**New Scoring Formula:**
```
baseScore = volumeScore + interactionBonus + priorityBonus
ageDecay = exp(-articleAgeInDays / decayConstant)
finalScore = baseScore × ageDecay
```

**Age Decay Function:**
- `decayConstant = 7` (articles lose ~63% of base score every 7 days)
- Day 0 article: 100% of base score
- Day 7 article: ~37% of base score
- Day 14 article: ~14% of base score

**Hard Cutoff:**
- Articles older than 14 days are excluded entirely
- Applied at SQL level: `published_at > NOW() - INTERVAL '14 days'`

**Constants:**
```go
const (
    SmartFeedMaxAgeDays = 14  // Hard cutoff
    SmartFeedDecayDays   = 7   // Decay constant
)
```

### 4. State Management

**New state in `RssPage.tsx`:**
```typescript
const [isSmartFeedActive, setIsSmartFeedActive] = useState(false);
```

**Selection logic:**
```typescript
// Smart Feed selected
isSmartFeedActive = true, selectedFeedId = null, selectedFolder = null

// Regular feed selected
isSmartFeedActive = false, selectedFeedId = X, selectedFolder = null

// Folder selected
isSmartFeedActive = false, selectedFeedId = null, selectedFolder = X
```

**Article hook usage:**
```typescript
// When Smart Feed is active
const smartArticles = useSmartRssArticles({});  // No filters

// When regular feed/folder is selected
const regularArticles = useRssArticles({
  folder: selectedFolder ?? undefined,
  feed_id: selectedFeedId ?? undefined,
  unread: showUnreadOnly || undefined
});
```

## API Changes

No new endpoints needed. Backend modifies existing `/rss/smart` endpoint:

**Backend Changes (`go-backend/services/smart_feed.go`):**

1. Add to WHERE clause:
```go
whereClause += " AND published_at > NOW() - INTERVAL '14 days'"
```

2. Calculate age decay for each article:
```go
articleAgeDays := time.Since(article.PublishedAt).Hours() / 24
ageDecay := math.Exp(-articleAgeDays / SmartFeedDecayDays)
finalScore := baseScore * ageDecay
```

## Frontend Changes

### Files to Modify

1. **`RssPage.tsx`**:
   - Add `isSmartFeedActive` state
   - Add handlers for smart feed selection
   - Remove `showSmartFeed` toggle usage
   - Update article hook selection logic

2. **`RssFeedsPanel.tsx`**:
   - Add "Smart Feed" button at top of sidebar
   - Add `onSelectSmartFeed` prop
   - Update selection highlighting logic

3. **`RssArticlesPanel.tsx`**:
   - Remove "Smart" tab from filter tabs
   - Add `isSmartFeedActive` prop
   - Conditionally hide filter tabs when smart feed active
   - Update header text when smart feed active

4. **`RssDesktopLayout.tsx`**:
   - Add `isSmartFeedActive` prop
   - Pass `onSelectSmartFeed` handler
   - Remove `onToggleSmartFeed` handler

5. **`RssMobileLayout.tsx`**:
   - Add "Smart Feed" item to feeds bottom sheet
   - Add `onSelectSmartFeed` handler
   - Remove smart feed toggle from any filter UI

## Mobile Changes

**Feeds Bottom Sheet:**
- Add "Smart Feed" as first item (before "All Feeds")
- Same star icon and styling
- Tappable to select

**Mobile Navigation:**
```typescript
handleSmartFeedSelectMobile={() => {
  setIsSmartFeedActive(true);
  setSelectedFolder(null);
  setSelectedFeedId(null);
  setMobileView('list');
}}
```

## Testing Strategy

1. **Backend tests**:
   - Verify 14-day cutoff excludes old articles
   - Verify age decay calculation is correct
   - Verify final score排序 works correctly

2. **Frontend tests**:
   - Smart Feed item appears in sidebar
   - Clicking Smart Feed activates correctly
   - Filter tabs hidden when Smart Feed active
   - Article list shows both read and unread
   - Pagination works correctly

3. **Manual testing**:
   - Navigate between Smart Feed and regular feeds
   - Verify recency ranking (newer articles appear higher)
   - Verify articles don't disappear after viewing

## Edge Cases

| Situation | Behavior |
|-----------|----------|
| No articles in last 14 days | Empty smart feed, show "No articles found" |
| User clicks Smart Feed while on specific feed | Clears feed selection, shows smart feed |
| User clicks regular feed while on Smart Feed | Deactivates smart feed, shows selected feed |
| All articles older than 14 days | Smart feed shows empty state |

## Future Enhancements (Out of Scope)

- User-configurable cutoff period (7/14/30 days)
- Folder-specific smart feeds
- Negative signals (penalty for skipped articles)
- Configurable decay rate
