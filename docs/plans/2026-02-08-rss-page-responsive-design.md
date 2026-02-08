# RSS Page Responsive Design

**Date:** 2026-02-08
**Status:** Design Approved

## Overview

Redesign the RSS page (`RssPage.tsx`) to work well on mobile devices. The current three-panel desktop layout (feeds sidebar, articles list, article reader) doesn't work on small screens.

## User Workflow

Primary mobile workflow: **Browse feeds → Tap article → Read full content → Go back to browse**

Feeds sidebar is **not important** on mobile - user mainly reads articles from "All Feeds" view.

## Design Approach: Bottom Sheet for Feeds

Mobile shows the article list by default. The feeds/folders are hidden behind a bottom sheet. Article reader goes full-screen with a back button.

**Why this approach:** Reading-focused experience with feeds accessible but not in the way. Familiar mobile pattern (like iOS Maps).

## Section 1: Mobile Screen Layout & Navigation

### Default State - Article List
- Full-width article list takes entire screen
- Top bar: "RSS" title with unread count badge, hamburger menu (left), settings gear (right)

### Bottom Sheet - Feeds
- Triggered by hamburger menu
- Slides up from bottom, covers ~70% of screen
- Handle bar at top for drag-to-dismiss
- Shows: All Feeds button, folders (expandable), uncategorized feeds
- Swipe down or tap outside to dismiss

### Article Reader
- Triggered by tapping any article
- Full-screen modal with slide-up animation
- Back button (chevron) top-left returns to article list
- Sticky action bar at bottom (Convert to Card, View Original, Mark Unread)

### Responsive Breakpoints
- `< 768px (md:)`: Mobile layout with bottom sheet
- `≥ 768px`: Current desktop layout unchanged

## Section 2: Mobile Article List View

### Layout
- Full-width list items (no fixed width container)
- Touch targets minimum 44px height
- Generous padding for tapping

### Article Item Design
- Title: `text-base` or `text-lg`, 2-3 line clamp
- Meta below title: feed name + date, `text-xs text-gray-500`
- Unread indicator: left border accent or blue background tint
- Card icon badge as-is (top-right)

### Top Bar Filters
- All/Unread toggle tabs full-width
- Move "Unread only" checkbox into filter icon/menu

### Pagination
- Replace Previous/Next with "Load More" button
- Infinite scroll as future enhancement

## Section 3: Mobile Article Reader View

### Layout
- Full viewport height reader
- Max width ~680px for readability
- Padding `p-4` or `p-6`

### Top Bar
- Back button (chevron-left) top-left
- "Convert" action button top-right
- Solid white bar with shadow

### Content Styling
- `prose-base` or `prose` (not `prose-sm`)
- Ensure code blocks/images don't overflow

### Bottom Action Bar
- Sticky bar above bottom safe area
- Actions: "Convert to Card" (primary), "View Original" (secondary), "Mark Unread" (tertiary)

## Section 4: Feeds Bottom Sheet

### Trigger
- Hamburger menu icon top-left of article list

### Sheet Design
- Slides up from bottom, ~70% screen height
- Dark semi-transparent backdrop (tap to dismiss)
- Drag handle bar at top (rounded pill, ~40px wide)
- Rounded top corners `rounded-t-3xl`

### Content Layout
- "All Feeds" button at top
- Folders section - collapsible
- "Uncategorized" section at bottom
- Full-width feed items with larger touch targets
- Edit/delete icons on tap or long-press

### Interactions
- Swipe down on handle to dismiss
- Tap backdrop to dismiss
- Tap feed/folder → sheet dismisses + articles load
- "Add Feed" button at bottom of sheet

## Section 5: Responsive Breakpoints & Transitions

### Breakpoint Strategy
```tsx
// Mobile: < 768px (md:)
// Desktop: >= 768px
<div className="md:hidden">  {/* Mobile only */}
<div className="hidden md:flex">  {/* Desktop only */}
```

### Animation Strategy
- CSS transitions for smooth panel transitions
- Slide-up for reader (translate-y)
- Slide-in for feed sheet (translate-x)
- Fade-in for bottom sheet (opacity + scale)

## Section 6: Implementation Approach

### Phase 1 - Core Mobile Layout
1. Add `viewState` state: `'list' | 'reader' | 'feeds'`
2. Wrap existing panels in responsive containers
3. Add mobile top bar with hamburger menu
4. Make article list full-width on mobile
5. Test desktop unchanged

### Phase 2 - Mobile Reader
1. Extract reader panel into conditional render
2. Add back button and mobile top bar
3. Make reader full-screen with slide animation
4. Add bottom action bar with safe-area padding

### Phase 3 - Bottom Sheet
1. Build bottom sheet component
2. Move feeds content into sheet on mobile
3. Add swipe/dismiss interactions
4. Connect feed selection to close sheet + load articles

### Key State Changes
```tsx
const [mobileView, setMobileView] = useState<'list' | 'reader' | 'feeds'>('list');

const handleArticleClick = (article) => {
  // existing logic...
  if (isMobile) {
    setMobileView('reader');
  }
};

const handleFeedSelect = (feedId) => {
  setSelectedFeedId(feedId);
  if (isMobile) {
    setMobileView('list');
  }
};
```

### Testing Checklist
- [ ] Desktop unchanged
- [ ] Mobile article list scrolls and taps
- [ ] Mobile reader opens/closes with back button
- [ ] Bottom sheet opens/dismisses
- [ ] Feed selection works and closes sheet
- [ ] Orientation changes handled
