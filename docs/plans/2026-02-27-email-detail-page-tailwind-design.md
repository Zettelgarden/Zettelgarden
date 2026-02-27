# EmailDetailPage Tailwind Refactor Design

**Date**: 2026-02-27
**Status**: Design Approved

## Overview

Refactor `EmailDetailPage.tsx` from inline styles to Tailwind utility classes, matching the extensive Tailwind patterns used throughout the rest of the codebase (ViewPage, EmailConvertDialog, etc.).

## Problem

The `EmailDetailPage.tsx` component currently uses:
- ~165 lines of inline CSS (`emailStyles` constant)
- Inline `style={{ ... }}` props throughout the component
- Manual style injection via `useEffect`

This is inconsistent with the rest of the codebase which uses Tailwind extensively.

## Solution

### 1. Remove inline styles and convert to Tailwind classes

**Before:**
```tsx
<div style={{ display: "flex", flexDirection: "column", height: "100vh", backgroundColor: "#ffffff" }}>
```

**After:**
```tsx
<div className="flex flex-col h-screen bg-white">
```

### 2. Email Content Styling Strategy (Option B)

The large `emailStyles` CSS block (165 lines) handles sanitizing external email HTML content. This will be kept as a **minimal scoped CSS module** rather than converting to Tailwind classes.

**Rationale:**
- External emails have their own styling that we sanitize, not fully re-style
- The prose plugin would be overkill for this use case
- Email body content needs specific handling that's different from the UI components

**Implementation:**
```tsx
// Create: src/components/email/EmailContent.module.css
.email-content {
  /* Keep essential email content styles only */
  @apply font-sans text-base leading-relaxed text-gray-800 break-words;
  /* ... minimal email-specific styles ... */
}

// Import in component:
import styles from './EmailContent.module.css';

<div className={styles['email-content']} dangerouslySetInnerHTML={{ __html: processedHtml }} />
```

### 3. Pattern Matching with Existing Codebase

Following patterns from `EmailConvertDialog` and `ViewPage`:

| Category | Inline Style | Tailwind Class |
|----------|-------------|----------------|
| Layout | `display: "flex", flexDirection: "column"` | `flex flex-col` |
| Spacing | `padding: "16px 24px"` | `px-6 py-4` |
| Gap | `gap: "12px"` | `gap-3` |
| Colors | `backgroundColor: "#ffffff"` | `bg-white` |
| Typography | `fontSize: "28px", fontWeight: "700"` | `text-2xl font-bold` |
| Borders | `border: "1px solid #e5e7eb"` | `border border-gray-200` |
| Rounded | `borderRadius: "8px"` | `rounded-lg` |
| Width | `maxWidth: "900px"` | `max-w-2xl` |
| Transitions | `transition: "all 0.15s ease"` | `transition-all duration-150` |
| Hover | `onMouseEnter` handlers | `hover:bg-gray-50` |

### 4. Conditional/Interactive Styling

**Mouse enter/leave handlers become Tailwind hover classes:**

**Before:**
```tsx
<button
  style={{ backgroundColor: "#ffffff" }}
  onMouseEnter={(e) => { e.currentTarget.style.backgroundColor = "#f9fafb"; }}
  onMouseLeave={(e) => { e.currentTarget.style.backgroundColor = "#ffffff"; }}
>
```

**After:**
```tsx
<button className="bg-white hover:bg-gray-50">
```

**Conditional classes:**

**Before:**
```tsx
<div style={{
  backgroundColor: isProUser ? "#ffffff" : "#fef3c7",
  color: isProUser ? "#374151" : "#92400e"
}}>
```

**After:**
```tsx
<div className={isProUser ? "bg-white text-gray-700" : "bg-yellow-50 text-yellow-800"}>
```

## Component Structure (Unchanged)

The logical structure remains the same:
1. Header with action buttons (Back, Archive/Unarchive, Convert to Card, Create Task, Extract Facts)
2. Email metadata section (From, To, Date, Folder, Status, Read)
3. Email body (with minimal CSS module)
4. Attachments section
5. Dialogs (CreateTaskWindow, EmailConvertDialog, Fact Extraction)

## Files to Modify

1. **Create**: `zettelkasten-front/src/components/email/EmailContent.module.css`
   - Extracted email content styles from the component

2. **Modify**: `zettelkasten-front/src/pages/EmailDetailPage.tsx`
   - Remove `emailStyles` constant (~165 lines)
   - Remove style injection `useEffect`
   - Convert all inline `style={{}}` props to `className` with Tailwind classes
   - Import `EmailContent.module.css`

## Key Tailwind Classes to Use

```tsx
// Layout
flex flex-col h-screen                // Full height flex container
flex items-center justify-between     // Header layout
flex-1 overflow-y-auto                // Scrollable content area

// Spacing
px-6 py-4                            // Horizontal/vertical padding
space-y-4                             // Vertical gap between children
gap-2 gap-3                           // Gap between flex items

// Typography
text-2xl font-bold text-gray-900      // Subject heading
text-sm font-medium text-gray-700     // Labels
text-xs font-semibold text-gray-500   // Metadata labels

// Buttons
px-4 py-2 text-sm font-medium        // Base button padding
rounded-lg border                     // Rounded border
bg-white hover:bg-gray-50            // Background with hover
transition-colors duration-150        // Smooth transitions

// Colors
bg-white text-gray-700               // Default
bg-blue-600 text-white               // Primary action
bg-gray-100 hover:bg-gray-200        // Secondary
bg-yellow-50 text-yellow-800         // PRO/warning
bg-red-50 text-red-800               // Error

// Status badges
rounded-full px-2 py-1               // Pill shape
text-xs font-semibold                // Small text

// Attachments
border rounded-lg                    // Card style
hover:border-gray-300                // Hover effect
```

## Implementation Steps

1. Create `EmailContent.module.css` with minimal email content styles
2. Update imports in `EmailDetailPage.tsx`
3. Remove `emailStyles` constant and style injection `useEffect`
4. Convert all inline styles to Tailwind classes systematically:
   - Header section
   - Buttons and hover states
   - Email metadata
   - Email body wrapper
   - Attachments section
   - Fact extraction dialog
5. Test visual appearance matches original
6. Test interactive states (hover, disabled, etc.)
7. Test responsive behavior

## Success Criteria

- All inline `style={{}}` props removed
- No `onMouseEnter`/`onMouseLeave` handlers for styling
- Visual appearance matches original
- Interactive states (hover, disabled) work correctly
- Email body content displays properly
- Code is consistent with `EmailConvertDialog` and `ViewPage` patterns
