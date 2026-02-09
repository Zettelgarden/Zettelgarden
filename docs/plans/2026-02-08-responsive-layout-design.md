# Responsive Layout Design for Zettelgarden

**Date:** 2026-02-08
**Status:** Design Approved
**Author:** Claude + User

## Overview

Apply the successful RSS Page responsive pattern to core pages (Search, Tasks, Dashboard) using a reusable component system.

## Goal

Create a consistent mobile experience across the app with:
- Shared responsive layout components
- Breakpoint at 768px (matching RSS)
- View state management for mobile navigation
- Bottom sheet pattern for mobile filters

## Page Targets

### SearchPage (3-Column)
- **Desktop**: Filters sidebar | Results list | Card detail
- **Mobile**: List view | Detail view | Filters bottom sheet
- **Left Panel**: Starred searches + advanced filter controls
- **Right Panel**: Full card detail (edit, backlinks, entities)

### TaskPage (2-Column Enhanced)
- **Desktop**: Main content | Quick lists sidebar (optional)
- **Mobile**: Enhanced list view + Filters bottom sheet
- **Keep**: Existing toolbar, view modes, task dialog pattern
- **Add**: Right panel with quick task lists (today, week, overdue)

### DashboardPage (Responsive Stack)
- **Desktop**: Main content | Stats sidebar
- **Mobile**: Stacked single column, scrollable
- **Keep**: Simple landing page, no view switching needed

## New Shared Components

```
components/layout/
├── ResponsiveLayout.tsx       # Wrapper for responsive pages
├── MobileTopBar.tsx           # Generic mobile top bar
├── MobileBottomSheet.tsx      # Generic bottom sheet
└── useResponsiveLayout.ts     # Hook for breakpoint + view state
```

### useResponsiveLayout Hook

```typescript
interface UseResponsiveLayoutReturn {
  isMobile: boolean;
  mobileView: 'list' | 'detail' | 'filters';
  setMobileView: (view: 'list' | 'detail' | 'filters') => void;
}
```

- Manages breakpoint detection (768px)
- Handles window resize events
- Provides mobile view state for navigation

### ResponsiveLayout Component

```typescript
interface ResponsiveLayoutProps {
  mobileView: 'list' | 'detail' | 'filters';
  setMobileView: (view: 'list' | 'detail' | 'filters') => void;
  children: (isMobile: boolean) => React.ReactNode;
}
```

- Wraps page content
- Renders children with `isMobile` boolean
- Enables conditional mobile/desktop rendering

## Implementation Order

1. **Create shared layout components**
   - `useResponsiveLayout` hook
   - `ResponsiveLayout` wrapper
   - `MobileTopBar` component
   - `MobileBottomSheet` component

2. **Refactor RssPage to use shared components**
   - Validates components on working code
   - Ensures API is correct

3. **Apply to SearchPage**
   - Extract `SearchResultsList` component
   - Create `SearchFiltersSidebar` (left panel)
   - Create `SearchCardDetail` (right panel)
   - Add `MobileSearchSheet` for filters
   - Update to use `ResponsiveLayout`

4. **Apply to TaskPage**
   - Create `TaskQuickLists` (right panel, optional)
   - Add `MobileTaskSheet` for filters
   - Keep existing task dialog pattern
   - Update to use `ResponsiveLayout`

5. **Apply to DashboardPage**
   - Add responsive stacking
   - Create `MobileDashboardStats` (horizontal scroll)
   - Keep simple, no view switching

## Data Flow Pattern

```typescript
function SearchPage() {
  // Page-specific state
  const [searchResults, setSearchResults] = useState<SearchResult[]>([]);
  const [selectedCard, setSelectedCard] = useState<Card | null>(null);

  // Shared layout state
  const { isMobile, mobileView, setMobileView } = useResponsiveLayout();

  const handleCardClick = (card: Card) => {
    setSelectedCard(card);
    if (isMobile) {
      setMobileView('detail');
    }
  };

  return (
    <ResponsiveLayout mobileView={mobileView} setMobileView={setMobileView}>
      {(isMobile) => (
        isMobile ? (
          // Mobile rendering based on mobileView
          mobileView === 'list' && <MobileResultsList />
          mobileView === 'detail' && <MobileCardDetail />
          mobileView === 'filters' && <MobileFiltersSheet />
        ) : (
          // Desktop multi-column
          <>
            <FiltersSidebar />
            <ResultsList />
            <CardDetailPanel />
          </>
        )
      )}
    </ResponsiveLayout>
  );
}
```

## Key Principles

1. **State stays in page component** - Not duplicated between mobile/desktop
2. **Mobile view is presentation only** - Same data, different layout
3. **URL synchronization** - Selected items reflected in URL
4. **Back button handling** - Mobile view changes on browser back

## Edge Cases

- **Resize during interaction**: Selected card persists, layout adapts
- **URL state on mobile**: Direct links open correct view
- **Bottom sheet dismissal**: Click outside, swipe down, browser back
- **Loading states**: Show skeleton in current view
- **Keyboard navigation**: Tab through panels, screen reader support

## Testing

- Unit tests for `useResponsiveLayout` hook
- Integration tests for view transitions
- Visual regression at 375px, 768px, 1024px, 1440px
- Browser back button behavior on mobile

## File Structure After

```
zettelkasten-front/src/
├── components/
│   ├── layout/                    # New shared layout components
│   ├── search/                    # Updated with 3-column
│   ├── tasks/                     # Updated with right panel
│   └── dashboard/                 # Updated with mobile stacking
├── pages/
│   ├── cards/SearchPage.tsx       # Refactored
│   ├── tasks/TaskPage.tsx         # Enhanced
│   └── DashboardPage.tsx          # Responsive stacking
└── hooks/
    └── useResponsiveLayout.ts     # Shared hook
```

## Migration Notes

- RSS components become reference implementations
- Extract common patterns as we go
- Keep existing components working during transition
- Progressive enhancement - mobile first
