# Sidebar Search Bar Implementation Plan

> **For Claude:** REQUIRED SUB-SKILL: Use superpowers:executing-plans to implement this plan task-by-task.

**Goal:** Add a search input bar to the Sidebar that navigates to SearchPage and executes search on Enter

**Architecture:** Create a new SidebarSearchBar component with local state for input, integrate into Sidebar.tsx, use React Router's navigate() to go to /app/search?term={query}

**Tech Stack:** React, TypeScript, React Router, Tailwind CSS

---

### Task 1: Create SidebarSearchBar component

**Files:**
- Create: `zettelkasten-front/src/components/sidebar/SidebarSearchBar.tsx`

**Step 1: Create the component file**

Write: `zettelkasten-front/src/components/sidebar/SidebarSearchBar.tsx`

```tsx
import React, { useState } from "react";
import { useNavigate } from "react-router-dom";

interface SidebarSearchBarProps {
  isCollapsed: boolean;
}

export function SidebarSearchBar({ isCollapsed }: SidebarSearchBarProps) {
  const [searchTerm, setSearchTerm] = useState("");
  const navigate = useNavigate();

  const handleKeyDown = (e: React.KeyboardEvent<HTMLInputElement>) => {
    if (e.key === "Enter") {
      const trimmed = searchTerm.trim();
      if (trimmed) {
        navigate(`/app/search?term=${encodeURIComponent(trimmed)}`);
        setSearchTerm(""); // Clear input after navigation
      }
    }
  };

  if (isCollapsed) {
    return null;
  }

  return (
    <div className="px-3 pb-3">
      <div className="relative">
        {/* Search icon */}
        <svg
          className="absolute left-3 top-1/2 transform -translate-y-1/2 w-4 h-4 text-gray-400"
          fill="none"
          stroke="currentColor"
          viewBox="0 0 24 24"
        >
          <path
            strokeLinecap="round"
            strokeLinejoin="round"
            strokeWidth={2}
            d="M21 21l-6-6m2-5a7 7 0 11-14 0 7 7 0 0114 0z"
          />
        </svg>
        <input
          type="text"
          value={searchTerm}
          onChange={(e) => setSearchTerm(e.target.value)}
          onKeyDown={handleKeyDown}
          placeholder="Search cards..."
          className="w-full pl-9 pr-3 py-2 text-sm border border-gray-300 rounded-md focus:outline-none focus:ring-2 focus:ring-blue-500 focus:border-transparent"
        />
      </div>
    </div>
  );
}
```

**Step 2: Commit the new component**

```bash
git add zettelkasten-front/src/components/sidebar/SidebarSearchBar.tsx
git commit -m "feat: add SidebarSearchBar component

- Create minimal search input with icon
- Handle Enter key to navigate to search page
- Hide when sidebar is collapsed

Co-Authored-By: Claude Opus 4.6 <noreply@anthropic.com>"
```

---

### Task 2: Integrate SidebarSearchBar into Sidebar component

**Files:**
- Modify: `zettelkasten-front/src/components/Sidebar.tsx:213-222`

**Step 1: Import the new component**

Add import at line 26:

```tsx
import { SidebarSearchBar } from "./sidebar/SidebarSearchBar";
```

**Step 2: Add SidebarSearchBar to the render**

Replace lines 213-222 (the section with SidebarHeader) to include the search bar:

```tsx
<SidebarHeader
  onNewStandardCard={handleNewStandardCard}
  onNewArticle={handleAddArticle}
  onNewTask={handleNewTask}
  onNewChat={handleNewChat}
  onAddFeed={handleAddFeed}
  isCollapsed={isSidebarCollapsed}
  onToggleCollapse={toggleSidebarCollapsed}
  unreadInboxCount={unreadInboxCount}
/>

<SidebarSearchBar isCollapsed={isSidebarCollapsed} />
```

**Step 3: Verify the component works**

Run: `cd zettelkasten-front && npm start`

Expected: Development server starts, sidebar shows search bar below header when expanded, hidden when collapsed

**Step 4: Test the functionality**

1. Type text in the search input
2. Press Enter
3. Should navigate to `/app/search?term=<your_text>`
4. SearchPage should execute search automatically

**Step 5: Commit the integration**

```bash
git add zettelkasten-front/src/components/Sidebar.tsx
git commit -m "feat: integrate SidebarSearchBar into Sidebar

- Add search bar below header in expanded state
- Import and render SidebarSearchBar component

Co-Authored-By: Claude Opus 4.6 <noreply@anthropic.com>"
```

---

### Task 3: Verify edge cases

**Step 1: Test empty input**

1. Click in search bar
2. Press Enter without typing anything
Expected: No navigation occurs

**Step 2: Test whitespace-only input**

1. Type only spaces in search bar
2. Press Enter
Expected: No navigation occurs (trim() handles this)

**Step 3: Test special characters**

1. Type search with special characters: `test & query`
2. Press Enter
Expected: Navigates to `/app/search?term=test%20%26%20query`

**Step 4: Test collapsed state**

1. Collapse sidebar
2. Verify search bar is hidden
3. Expand sidebar
4. Verify search bar reappears

---

### Task 4: Test on mobile

**Step 1: Open mobile view (or use browser dev tools)**

1. Open sidebar on mobile
2. Verify search bar appears in mobile sidebar
3. Test search functionality works the same way

---

### Task 5: Final verification and cleanup

**Step 1: Check for TypeScript errors**

Run: `cd zettelkasten-front && npm run build`

Expected: Build succeeds without errors

**Step 2: Lint check**

Run: `cd zettelkasten-front && npm run lint` (if available)

**Step 3: Final commit if any adjustments needed**

```bash
git add -A
git commit -m "fix: any minor adjustments from testing

Co-Authored-By: Claude Opus 4.6 <noreply@anthropic.com>"
```

---

## Testing Checklist

- [ ] Search bar appears below header when sidebar is expanded
- [ ] Search bar is hidden when sidebar is collapsed
- [ ] Typing and pressing Enter navigates to search page with query
- [ ] Empty input doesn't trigger navigation
- [ ] Special characters are properly encoded
- [ ] Input clears after navigation
- [ ] Works on mobile view
- [ ] TypeScript build passes
