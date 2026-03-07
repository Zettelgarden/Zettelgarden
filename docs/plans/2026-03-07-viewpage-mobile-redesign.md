# ViewPage Mobile Redesign

**Date:** 2026-03-07
**Status:** Approved

## Overview

Redesign the mobile experience for ViewPage.tsx using a hybrid wrapper pattern with accordion sections and a navigation bottom sheet, inspired by the successful RssPage mobile implementation.

## Goals

1. Make side panel content (tags, entities, related cards) easily accessible on mobile
2. Provide clean navigation for card hierarchy (parent/siblings/children)
3. Keep view modes accessible but not prominent (rarely used)
4. Maintain code reuse with shared hooks and components

## Architecture

### New Files

```
src/components/cards/
├── ViewMobileLayout.tsx         # Main mobile wrapper component
├── ViewMobileAccordion.tsx      # Reusable collapsible section component
└── ViewNavigationSheet.tsx      # Bottom sheet for hierarchy navigation
```

### Modified Files

```
src/pages/cards/ViewPage.tsx               # Add mobile detection and conditional render
src/components/cards/ViewPageSidePanels.tsx # Extract sections for reuse
```

### Data Flow

- `ViewPage.tsx` detects mobile using window.innerWidth < 768 (same as RssPage)
- Passes all existing props to `ViewMobileLayout` when on mobile
- `ViewMobileLayout` uses existing `useViewPageContainer` data - no new hooks needed
- Accordion sections reuse logic from `ViewPageSidePanels`

## Component Designs

### ViewMobileLayout

**Responsibilities:**
- Render `MobileTopBar` with title and view mode menu
- Render main content area (`ViewCardContentSection`)
- Organize side panel content into accordion sections
- Handle navigation bottom sheet state

**State:**
```typescript
const [expandedSections, setExpandedSections] = useState<Set<string>>(new Set(['tags']));
const [showNavSheet, setShowNavSheet] = useState(false);
```

**Layout:**
```
┌─────────────────────────────────┐
│ MobileTopBar                    │
│ [≡] Card Title         [⋯]     │  ← menu has view modes + navigate
├─────────────────────────────────┤
│                                 │
│   Main Content                  │
│   (markdown body, children)     │
│                                 │
├─────────────────────────────────┤
│ ▼ Tags                    [edit]│  ← expanded by default
│   #tag1 #tag2 #tag3             │
├─────────────────────────────────┤
│ ► Navigation                    │  ← collapsed
├─────────────────────────────────┤
│ ► Linked Entities               │  ← collapsed
├─────────────────────────────────┤
│ ► Related Cards                 │  ← collapsed
├─────────────────────────────────┤
│ ► Details                       │  ← collapsed (created, updated, link)
└─────────────────────────────────┘
```

### ViewMobileAccordion

**A reusable collapsible section with sticky header:**

```typescript
interface ViewMobileAccordionProps {
  title: string;
  icon?: React.ReactNode;
  defaultExpanded?: boolean;
  rightElement?: React.ReactNode;  // e.g., edit button for tags
  children: React.ReactNode;
}
```

**Behavior:**
- Tap header to expand/collapse
- Smooth height animation
- Visual indicator: `▼` when expanded, `►` when collapsed
- Sticky headers for quick navigation while scrolling

### ViewNavigationSheet

**A bottom sheet for card hierarchy navigation:**

```typescript
interface ViewNavigationSheetProps {
  isOpen: boolean;
  onClose: () => void;
  parentCard: Card | null;
  prevSibling: PartialCard | null;
  nextSibling: PartialCard | null;
  viewingCard: Card;  // for children access
  onNavigate: (cardId: number) => void;
}
```

**Triggered by:** "Navigate" option in the top bar menu

**Layout:**
```
┌─────────────────────────────────┐
│ ━━━━━━━                         │  ← drag handle
│ Navigate                        │
├─────────────────────────────────┤
│                                 │
│   ┌─────────────────────┐       │
│   │ ↑ Parent Card       │       │
│   │   "Parent Title"    │       │
│   └─────────────────────┘       │
│                                 │
│   ┌─────────┐  ┌─────────┐      │
│   │ ← Prev  │  │  Next → │      │
│   └─────────┘  └─────────┘      │
│                                 │
│   Children (if any)             │
│   • Child 1                     │
│   • Child 2                     │
│   • ...                         │
│                                 │
└─────────────────────────────────┘
```

**Uses:** Existing `MobileBottomSheet` component for swipe-to-dismiss

### View Mode Menu

The `⋯` menu in MobileTopBar contains:
- **View Mode** submenu:
  - Normal (default)
  - Tree View
  - Summary
  - Analysis
- ─────────────
- **Navigate...** → opens ViewNavigationSheet

## ViewPage.tsx Changes

1. Add mobile detection:
```typescript
const [isMobile, setIsMobile] = useState(() =>
  typeof window !== 'undefined' && window.innerWidth < 768
);

useEffect(() => {
  const handleResize = () => {
    setIsMobile(window.innerWidth < 768);
  };
  window.addEventListener('resize', handleResize);
  return () => window.removeEventListener('resize', handleResize);
}, []);
```

2. Conditionally render layouts:
```typescript
return (
  <div className="overflow-x-hidden">
    {isMobile ? (
      <ViewMobileLayout
        viewingCard={viewingCard}
        parentCard={parentCard}
        // ... all other props
      />
    ) : (
      // existing desktop layout
    )}
  </div>
);
```

## Implementation Notes

- Reuse existing `MobileBottomSheet` component for navigation sheet
- Reuse existing `useViewPageContainer` hook - no new data fetching
- Extract section rendering logic from `ViewPageSidePanels` into small helper functions for reuse
- Default expanded sections: Tags only (most commonly edited)
- All other sections collapsed by default to reduce scroll

## Success Criteria

1. Side panel sections are accessible via accordion with minimal scrolling
2. Card hierarchy navigation is available on demand via bottom sheet
3. View modes are accessible but don't clutter the UI
4. Desktop experience remains unchanged
5. Mobile layout matches the quality of RssPage mobile experience
