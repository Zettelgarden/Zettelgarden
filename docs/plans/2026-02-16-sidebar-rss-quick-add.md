# Sidebar RSS Feed Quick Add Implementation Plan

> **For Claude:** REQUIRED SUB-SKILL: Use superpowers:executing-plans to implement this plan task-by-task.

**Goal:** Add a 5th quick action to the Sidebar dropdown that allows users to add RSS feeds without navigating to the RSS page.

**Architecture:** Use existing `RssAddFeedDialog` component, fetch folders via API, manage dialog state in Sidebar following existing patterns.

**Tech Stack:** React, TypeScript, Tailwind CSS, lucide-react icons

---

## Task 1: Add RSS icon import to SidebarHeader

**Files:**
- Modify: `zettelkasten-front/src/components/sidebar/SidebarHeader.tsx:5`

**Step 1: Add Rss icon import**

Add `Rss` to the lucide-react imports:

```typescript
import { Plus, Rss } from "lucide-react";
```

**Step 2: Verify no errors**

Run: `cd zettelkasten-front && npm run build`
Expected: Builds successfully

**Step 3: Commit**

```bash
git add zettelkasten-front/src/components/sidebar/SidebarHeader.tsx
git commit -m "feat: add Rss icon import to SidebarHeader"
```

---

## Task 2: Add onAddFeed prop to SidebarHeader interface and component

**Files:**
- Modify: `zettelkasten-front/src/components/sidebar/SidebarHeader.tsx:7-14,16-23`

**Step 1: Add onAddFeed to interface**

Add `onAddFeed` to the `SidebarHeaderProps` interface:

```typescript
interface SidebarHeaderProps {
  onNewStandardCard: () => void;
  onNewArticle: () => void;
  onNewTask: () => void;
  onNewChat: () => void;
  onAddFeed: () => void;
  isCollapsed: boolean;
  onToggleCollapse: () => void;
}
```

**Step 2: Add onAddFeed to component destructuring**

Update the component props destructuring:

```typescript
export function SidebarHeader({
  onNewStandardCard,
  onNewArticle,
  onNewTask,
  onNewChat,
  onAddFeed,
  isCollapsed,
  onToggleCollapse,
}: SidebarHeaderProps) {
```

**Step 3: Verify no TypeScript errors**

Run: `cd zettelkasten-front && npx tsc --noEmit`
Expected: No type errors

**Step 4: Commit**

```bash
git add zettelkasten-front/src/components/sidebar/SidebarHeader.tsx
git commit -m "feat: add onAddFeed prop to SidebarHeader"
```

---

## Task 3: Add "Add RSS Feed" button to SidebarHeader dropdown menu

**Files:**
- Modify: `zettelkasten-front/src/components/sidebar/SidebarHeader.tsx:95-130`

**Step 1: Add the button to the dropdown menu**

Add the new button after "New Chat" (before the closing `</div>` of the dropdown):

```typescript
              <button
                onClick={onAddFeed}
                className="w-full text-left px-4 py-3 min-h-[44px] hover:bg-gray-100 flex items-center gap-2"
                role="menuitem"
              >
                <Rss size={16} />
                Add RSS Feed
              </button>
```

Place it after the `hasSubscription &&` block for "New Chat", before the closing `</div>`.

**Step 2: Verify no errors**

Run: `cd zettelkasten-front && npm run build`
Expected: Builds successfully

**Step 3: Commit**

```bash
git add zettelkasten-front/src/components/sidebar/SidebarHeader.tsx
git commit -m "feat: add Add RSS Feed button to SidebarHeader dropdown"
```

---

## Task 4: Add state and handlers to Sidebar for RSS feed dialog

**Files:**
- Modify: `zettelkasten-front/src/components/Sidebar.tsx:1-25,35-36,80-94`

**Step 1: Add RSS-related imports**

Add the imports after line 13:

```typescript
import { listFolders } from "../api/rss";
import { RssAddFeedDialog } from "./rss/RssAddFeedDialog";
import { RSSFolder, RSSFeed } from "../models/Card";
```

Note: Check if `RSSFeed` and `RSSFolder` are already exported from `../models/Card` or need to be imported from `../api/rss`.

**Step 2: Add state for RSS feed dialog**

Add after `showAddArticleDialog` state (around line 35):

```typescript
  const [showAddFeedDialog, setShowAddFeedDialog] = useState(false);
  const [rssFolders, setRssFolders] = useState<RSSFolder[]>([]);
```

**Step 3: Add fetch folders useEffect**

Add after the `handleCloseGettingStarted` useEffect (after line 109):

```typescript
  useEffect(() => {
    async function fetchFolders() {
      try {
        const folders = await listFolders();
        setRssFolders(folders);
      } catch (error) {
        console.error("Failed to fetch RSS folders:", error);
      }
    }
    fetchFolders();
  }, []);
```

**Step 4: Add handleAddFeed handler**

Add after `handleAddArticle` function (around line 94):

```typescript
  function handleAddFeed() {
    setShowAddFeedDialog(true);
  }
```

**Step 5: Add handleFeedAdded handler**

Add after `handleAddFeed`:

```typescript
  function handleFeedAdded(feed: RSSFeed) {
    // Feed added successfully - dialog will close itself
    // No additional action needed - RSS page will show the new feed when visited
    console.log("Feed added:", feed);
  }
```

**Step 6: Verify TypeScript compiles**

Run: `cd zettelkasten-front && npx tsc --noEmit`
Expected: No type errors

**Step 7: Commit**

```bash
git add zettelkasten-front/src/components/Sidebar.tsx
git commit -m "feat: add RSS feed dialog state and handlers to Sidebar"
```

---

## Task 5: Pass onAddFeed prop to SidebarHeader in Sidebar

**Files:**
- Modify: `zettelkasten-front/src/components/Sidebar.tsx:188-195`

**Step 1: Add onAddFeed prop to SidebarHeader**

Update the `SidebarHeader` component call to include the new prop:

```typescript
        <SidebarHeader
          onNewStandardCard={handleNewStandardCard}
          onNewArticle={handleAddArticle}
          onNewTask={handleNewTask}
          onNewChat={handleNewChat}
          onAddFeed={handleAddFeed}
          isCollapsed={isSidebarCollapsed}
          onToggleCollapse={toggleSidebarCollapsed}
        />
```

**Step 2: Verify no errors**

Run: `cd zettelkasten-front && npm run build`
Expected: Builds successfully

**Step 3: Commit**

```bash
git add zettelkasten-front/src/components/Sidebar.tsx
git commit -m "feat: pass onAddFeed handler to SidebarHeader"
```

---

## Task 6: Add RssAddFeedDialog component to Sidebar JSX

**Files:**
- Modify: `zettelkasten-front/src/components/Sidebar.tsx:250-256`

**Step 1: Add RssAddFeedDialog to SidebarModals props**

Find `SidebarModals` component (around line 229) and add the new props. First, check if SidebarModals needs the dialog props or if we should render RssAddFeedDialog directly.

Looking at the code, `SidebarModals` already handles various dialogs. However, for consistency with the design, we should add the RssAddFeedDialog props to SidebarModals.

But first, let's check the SidebarModals component to understand its pattern:

```bash
grep -A 50 "export function SidebarModals" zettelkasten-front/src/components/sidebar/SidebarModals.tsx
```

Actually, the simpler approach is to render `RssAddFeedDialog` directly in the Sidebar component, similar to how other dialogs might be handled.

**Step 1: Add RssAddFeedDialog after SidebarModals**

Add after the `SidebarModals` component (around line 255, before the closing `</>`):

```typescript
      <RssAddFeedDialog
        isOpen={showAddFeedDialog}
        onClose={() => setShowAddFeedDialog(false)}
        folders={rssFolders}
        onFeedAdded={handleFeedAdded}
      />
```

**Step 2: Verify no errors**

Run: `cd zettelkasten-front && npm run build`
Expected: Builds successfully

**Step 3: Commit**

```bash
git add zettelkasten-front/src/components/Sidebar.tsx
git commit -m "feat: render RssAddFeedDialog in Sidebar"
```

---

## Task 7: Verify the implementation works

**Files:**
- Test: Manual browser testing

**Step 1: Start the development server**

Run: `cd zettelkasten-front && npm start`

**Step 2: Test the feature**

1. Open the app in a browser
2. Click the "+" button in the sidebar header
3. Verify "Add RSS Feed" option appears in the dropdown with an RSS icon
4. Click "Add RSS Feed"
5. Verify the dialog opens
6. Verify the folder dropdown shows existing folders
7. Try adding a feed (e.g., `https://www.theverge.com/rss/index.xml`)
8. Verify the feed is added successfully
9. Navigate to RSS page and verify the new feed appears

**Step 4: Edge cases to test**

- Dialog closes when clicking Cancel
- Dialog closes when clicking outside
- Error handling for invalid URLs
- Dialog works on mobile
- Button is hidden when sidebar is collapsed

**Step 5: Commit any fixes**

If any issues are found and fixed:

```bash
git add zettelkasten-front/src/components/Sidebar.tsx zettelkasten-front/src/components/sidebar/SidebarHeader.tsx
git commit -m "fix: address issues found during testing"
```

---

## Task 8: Final verification and cleanup

**Files:**
- All modified files

**Step 1: Run TypeScript check**

Run: `cd zettelkasten-front && npx tsc --noEmit`
Expected: No type errors

**Step 2: Run tests**

Run: `cd zettelkasten-front && npm test`
Expected: All tests pass

**Step 3: Build for production**

Run: `cd zettelkasten-front && npm run build`
Expected: Build succeeds

**Step 4: Final commit**

```bash
git add -A
git commit -m "feat: complete sidebar RSS feed quick add feature

- Add RSS icon import to SidebarHeader
- Add onAddFeed prop to SidebarHeader
- Add 'Add RSS Feed' button to dropdown menu
- Add state and handlers in Sidebar
- Fetch folders from API on mount
- Render RssAddFeedDialog component
- Silent success, no navigation after adding

Tested: Dialog opens, folders load, feed adds successfully"
```

---

## Summary

This plan adds the ability to add RSS feeds directly from the Sidebar without navigating to the RSS page. The implementation:

1. Adds a 5th option "Add RSS Feed" to the Sidebar dropdown menu
2. Uses the existing `RssAddFeedDialog` component
3. Fetches folders from the API for the folder dropdown
4. Follows the existing Sidebar patterns for dialog management
5. Silent success - no toast notification, no navigation

**Files modified:**
- `zettelkasten-front/src/components/sidebar/SidebarHeader.tsx`
- `zettelkasten-front/src/components/Sidebar.tsx`

**Estimated time:** 30-45 minutes
