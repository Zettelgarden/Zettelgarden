# Design: Sidebar Search Bar

**Date:** 2026-02-24

## Overview
Add a minimal search input bar to the Sidebar that allows users to quickly navigate to the SearchPage and execute a search by typing a query and pressing Enter.

## Requirements
- Search bar positioned at the top of sidebar, below the header
- Hidden when sidebar is collapsed
- Minimal design: input field with search icon
- On Enter: navigate directly to `/app/search?term={query}` with no loading feedback

## Architecture
- New `SidebarSearchBar` component in `src/components/sidebar/`
- Integrated into `Sidebar.tsx` between `SidebarHeader` and scrollable middle section
- Uses React Router's `useNavigate` for navigation

## Component Specification

### SidebarSearchBar Component

**Props:**
- `isCollapsed: boolean` - Controls visibility

**State:**
- `searchTerm: string` - Local input value

**Behavior:**
- Renders input with search icon
- Placeholder: "Search cards..."
- `onKeyDown` handler: on Enter → navigate to search page
- Hidden via CSS when `isCollapsed` is true

## Data Flow
1. User types in the search input (local state updates)
2. User presses Enter key
3. `navigate('/app/search?term={encodedSearchTerm}')` is called
4. SearchPage reads the `term` URL param and executes search automatically

## Styling
- Match existing Sidebar styling (white bg, border-r)
- Input: full width, rounded corners, subtle border
- Search icon: SVG positioned inside input on the left
- Height: ~40px for touch-friendly target

## Edge Cases
- **Empty input**: Don't navigate (ignore Enter press)
- **Special characters**: Encode using `encodeURIComponent()`
- **Whitespace-only input**: Trim before validating

## Files to Modify
- `src/components/Sidebar.tsx` - Add SidebarSearchBar component
- `src/components/sidebar/SidebarSearchBar.tsx` - New component
