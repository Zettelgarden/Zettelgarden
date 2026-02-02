# Mobile ViewPage Progressive Disclosure Design

## Overview

Redesign the ViewPage mobile experience using a progressive disclosure pattern with stack-based navigation. The current desktop layout doesn't work well on small screens—this design creates a mobile-first experience while keeping desktop unchanged.

## Date

2025-02-01

## Section 1: Mobile Layout Structure

**Core Concept**: A stack-based navigation with clean reading focus and contextual access to features.

**Layout Hierarchy**:
```
┌─────────────────────────┐
│ Minimal Header          │ ← Sticky top, title + star + pin + overflow
├─────────────────────────┤
│                         │
│ Card Content            │ ← Scrollable main area
│ (body, files, facts)    │
│                         │
├─────────────────────────┤
│ Bottom Navigation Bar   │ ← Sticky bottom, 3-4 quick actions
└─────────────────────────┘
```

**Mobile-Only Container**: Wrap the existing ViewPage in a `<ViewPageMobile>` component that:
- Uses CSS media query or `useBreakpoint` hook to detect mobile (`< 768px`)
- On mobile: renders the new progressive disclosure layout
- On desktop: renders existing ViewPage unchanged

**Stack Navigation**: When navigating to a new card (parent, child, sibling), the new card slides in from the right with a native-feeling transition. A back button appears in the header.

## Section 2: Minimal Header Component

**`<MobileViewHeader>`** - A compact, action-dense header optimized for touch targets.

**Left side**:
- Back button (only when in stack navigation) - chevron-left icon
- Card title (truncated with ellipsis after ~40 chars)

**Right side**:
- Star icon (toggle)
- Pin icon (toggle)
- Overflow menu button (3-dot icon)

**Overflow menu contents**:
- Edit card
- Create child card
- Create task
- Resummarize
- Recategorize
- Share/Open ID discovery
- Switch view mode (summary/analysis/tree)

**Styling**:
- Height: 56px (standard mobile toolbar)
- Background: White with subtle bottom border
- Sticky positioning with backdrop blur for modern feel
- Touch targets: minimum 44x44px

**Behavior**:
- Tapping the title shows full title in a modal if truncated
- Star/Pin show immediate visual feedback (filled vs outline icons)
- Overflow menu slides up from bottom as a bottom sheet with grouped actions

## Section 3: Bottom Navigation Bar

**`<MobileBottomBar>`** - Fixed bottom bar for quick contextual actions.

**Three primary tabs** (always visible):
1. **Related** - Shows count of parent/children/siblings
2. **Entities** - Shows entity count if any exist
3. **Tags** - Shows tag count

**Interaction**:
- Tap any tab to slide up a bottom sheet with the corresponding content
- Bottom sheet is dismissible by pulling down or tapping backdrop
- Each tab shows a badge count when content exists

**Bottom Sheet Content**:

*Related Sheet*:
- Parent card (tap to navigate)
- Siblings (previous/next with card titles)
- Children (expandable list, tap any to navigate)
- Quick actions: "Create child card" button at bottom

*Entities Sheet*:
- List of linked entities with type badges
- Tap entity to open entity detail modal
- "Show facts" button if facts exist

*Tags Sheet*:
- Scrollable tag list with pill styling
- Tap tag to navigate to tag search
- Long-press to remove tag (with confirmation)

**Styling**:
- Height: 56px
- Icons + labels for clarity
- Border top with subtle shadow
- Active tab highlighted with accent color

## Section 4: Card Content Area

**`<MobileCardContent>`** - Optimized reading experience with tabbed metadata.

**Main content**:
- Card body with markdown rendering (unchanged from desktop)
- Full-width content with comfortable padding (16px sides)
- Code blocks horizontally scrollable
- Images responsive to viewport width

**Metadata tabs** (below body, above bottom bar):
- Files (if any)
- Facts (if any)
- References (if any)

**Tab behavior**:
- Default: no tabs shown if no metadata exists
- Single tab type: show inline without tab switcher
- Multiple tabs: show tab bar with 2-3 tabs max
- Tab content expands inline, not full screen

**Files tab**:
- Thumbnail grid for images, list for other files
- Tap to download/preview

**Facts tab**:
- Compact fact list with structured data display
- Tap fact to edit (if editable)

**References tab**:
- Categorized references (same as desktop)
- Tap any reference card to navigate to it

**Typography**:
- Base font: 16px (prevents zoom on iOS)
- Line height: 1.6 for readability
- Comfortable margins between sections

## Section 5: Stack Navigation & Transitions

**Navigation Behavior** - Native-feeling card transitions.

**Stack structure**:
- Maintain a navigation stack: `[rootCard, childCard, grandchildCard, ...]`
- Each new navigation pushes to stack
- Back button pops from stack

**Transition animation**:
- New card slides in from right (300ms ease-out)
- Old card slides out to left
- Reverse animation on back (slides in from left)
- CSS transform: `translateX()` for performance

**Back button behavior**:
- Only appears when stack depth > 1
- Shows chevron-left + "Back" label
- Tapping returns to previous card, keeping scroll position
- Long-press shows stack history for deep jumps

**Navigation triggers**:
- Parent card in Related sheet → push parent
- Sibling cards in Related sheet → push sibling
- Child card in Related sheet → push child
- Reference cards → push referenced card
- Entity cards → push entity card

**Scroll preservation**:
- Each card's scroll position saved when navigating away
- Restored when navigating back
- Uses simple state object: `{ [cardId]: scrollY }`

**Keyboard shortcuts**:
- Disable desktop shortcuts ('c', 't', 's') on mobile to prevent accidental triggers
- Or map to accessible buttons if desired

## Section 6: Component Architecture & State Management

**Component structure**:
```
ViewPage.tsx (entry point)
├── ViewPageDesktop (existing layout)
└── ViewPageMobile (new mobile container)
    ├── MobileViewHeader
    ├── MobileCardContent
    ├── MobileBottomBar
    └── MobileBottomSheet (shared)
        ├── RelatedSheetContent
        ├── EntitiesSheetContent
        └── TagsSheetContent
```

**State management** - Extend existing patterns:

**New state in `ViewPageContainer`**:
- `navigationStack`: Array of visited card IDs
- `scrollPositions`: Record of cardId → scrollY
- `activeBottomSheet`: 'related' | 'entities' | 'tags' | null
- `activeMetadataTab`: 'files' | 'facts' | 'references' | null

**New actions**:
- `navigateToCard(cardId, direction)`: Push to stack with animation
- `navigateBack()`: Pop from stack
- `openBottomSheet(sheet)`: Set active sheet
- `closeBottomSheet()`: Clear active sheet
- `setMetadataTab(tab)`: Switch metadata view

**Custom hooks**:
- `useMobileNavigation()`: Manages stack and scroll state
- `useBreakpoint()`: Detects mobile viewport (`window.innerWidth < 768`)
- `useBottomSheet()`: Manages sheet open/close with spring animation

**Shared state**:
- Reuse existing `useViewPageContainer` data
- Reuse existing contexts (`TagContext`, `UIStateContext`, `DialogStateContext`)
- Mobile-specific state isolated to prevent desktop impact

## Section 7: CSS & Animation Implementation

**Mobile-first styling approach**:

**Media query strategy**:
```css
/* Desktop default (existing styles) */
.ViewPage { ... }

/* Mobile overrides */
@media (max-width: 767px) {
  .ViewPage.mobile-layout { ... }
}
```

**CSS Modules or Tailwind**:
- Use existing Tailwind classes for consistency
- Add mobile-specific utilities: `.mobile-only`, `.desktop-only`
- Custom CSS for animations in `ViewPage.module.css`

**Key animations**:

*Slide transition*:
```css
.card-slide-enter { transform: translateX(100%); }
.card-slide-enter-active { transform: translateX(0); transition: transform 300ms ease-out; }
.card-slide-exit { transform: translateX(0); }
.card-slide-exit-active { transform: translateX(-30%); transition: transform 300ms ease-out; opacity: 0; }
```

*Bottom sheet*:
```css
.bottom-sheet { transform: translateY(100%); transition: transform 250ms cubic-bezier(0.4, 0, 0.2, 1); }
.bottom-sheet.open { transform: translateY(0); }
```

*Backdrop*:
```css
.backdrop { opacity: 0; transition: opacity 250ms; pointer-events: none; }
.backdrop.open { opacity: 1; pointer-events: auto; }
```

**Performance**:
- Use `transform` and `opacity` for GPU acceleration
- Avoid animating `height`, `width`, or layout properties
- `will-change: transform` on animated elements
- Debounce resize handlers for breakpoint detection

**Z-index layering**:
- Bottom sheet: 1000
- Backdrop: 999
- Header: 100
- Bottom bar: 100

## Section 8: Edge Cases & Error Handling

**Error states**:

*Network errors*:
- Show toast/snackbar notification on navigation failure
- Keep user on current card, don't push to stack
- Retry option in notification

*Missing card data*:
- Show skeleton loader while fetching
- If card not found: clear stack, show error message, offer return to root

*Scroll state edge cases*:
- Scroll position > 0 when navigating back: restore smoothly
- Card content changed: reset scroll to top
- Very long cards: cap saved scroll position to reasonable max

**Edge case behaviors**:

*Deep linking*:
- URL with card ID: start stack with that card as root
- Browser back button: sync with app stack (use React Router state)

*Orientation change*:
- Preserve stack and scroll state
- Re-render appropriate layout (mobile ↔ desktop)
- Bottom sheet closes on orientation change for safety

*Pinned cards*:
- Pinned card always at bottom of stack
- Visual indicator (pin icon + accent) in header

*View mode switching*:
- Summary/Analysis views: replace current card in stack (don't push)
- Tree view: full-screen modal overlay, separate from stack
- Back button returns to previous card in normal view

**Accessibility**:
- Focus trap in bottom sheet when open
- Escape key closes bottom sheet
- Back button closes bottom sheet before exiting stack
- ARIA labels for icon-only buttons
- Keyboard navigation through bottom sheet content

## Section 9: Testing Strategy

**Unit tests** (Vitest + RTL):

*Component tests*:
- `MobileViewHeader`: renders with all actions, overflow menu opens/closes
- `MobileBottomBar`: tab badges show correct counts, bottom sheet opens on tap
- `MobileCardContent`: markdown renders, tabs switch correctly
- `MobileBottomSheet`: dismisses on backdrop tap, trap focus works

*Hook tests*:
- `useMobileNavigation`: stack pushes/pops correctly, scroll positions saved/restored
- `useBreakpoint`: returns correct mobile/desktop state
- `useBottomSheet`: open/close state management

**Integration tests**:

*Navigation flow*:
- Navigate to parent → stack has 2 cards, back button appears
- Navigate back → returns to previous card, scroll position restored
- Deep navigation (3+ cards) → back button works at each level

*Bottom sheet flow*:
- Open Related sheet → shows parent/children/siblings
- Tap child card → sheet closes, navigates to child
- Tap entity → opens entity modal

**E2E tests** (Playwright or similar):

*Critical paths*:
- Mobile user opens card → navigates to child → returns to parent
- Mobile user creates child card → appears in Related sheet
- Mobile user tags card → appears in Tags sheet
- Orientation change → layout adapts, state preserved

**Visual regression**:
- Screenshots for each mobile view state
- Compare against baseline on each change

**Device testing**:
- Test on actual iOS/Android devices
- Verify touch targets, gestures, and native-feeling transitions
