# RSS Page Responsive Mobile Implementation Plan

> **For Claude:** REQUIRED SUB-SKILL: Use superpowers:executing-plans to implement this plan task-by-task.

**Goal:** Make the RSS page (`RssPage.tsx`) work well on mobile devices with a bottom sheet for feeds and full-screen article reader.

**Architecture:** Mobile-first responsive design using Tailwind CSS breakpoints. On mobile (< 768px), show article list by default with feeds in a bottom sheet. Article reader slides up full-screen. Desktop layout unchanged.

**Tech Stack:** React 18, TypeScript, Tailwind CSS 3, Vitest for testing

---

## Phase 1: Core Mobile Layout

### Task 1: Add Mobile View State

**Files:**
- Modify: `zettelkasten-front/src/pages/RssPage.tsx`

**Step 1: Add mobile view state type**

Add after line 68 (after existing state declarations):

```tsx
// Mobile navigation state
const [mobileView, setMobileView] = useState<'list' | 'reader' | 'feeds'>('list');
```

**Step 2: Add isMobile helper**

Add after line 73 (after articlesPerPage constant):

```tsx
// Mobile breakpoint detection
const isMobile = window.innerWidth < 768;
```

**Step 3: Commit**

```bash
cd zettelkasten-front
git add src/pages/RssPage.tsx
git commit -m "feat(rss): add mobile view state

Add mobileView state to track mobile navigation:
- list: article list view (default)
- reader: full-screen article reader
- feeds: feeds bottom sheet

Also add isMobile helper for breakpoint detection.
"
```

---

### Task 2: Add Mobile Top Bar Component

**Files:**
- Create: `zettelkasten-front/src/components/rss/RssMobileTopBar.tsx`

**Step 1: Write the component**

Create file with:

```tsx
import React from "react";

interface RssMobileTopBarProps {
  title: string;
  unreadCount?: number;
  onMenuClick: () => void;
  onSettingsClick: () => void;
  rightAction?: React.ReactNode;
}

export function RssMobileTopBar({
  title,
  unreadCount = 0,
  onMenuClick,
  onSettingsClick,
  rightAction,
}: RssMobileTopBarProps) {
  return (
    <div className="sticky top-0 z-40 bg-white border-b border-gray-200 px-4 py-3 flex items-center justify-between md:hidden">
      {/* Left: Hamburger menu */}
      <button
        onClick={onMenuClick}
        className="p-2 -ml-2 text-gray-600 hover:text-gray-900 hover:bg-gray-100 rounded-lg"
        aria-label="Open feeds"
      >
        <svg className="w-6 h-6" fill="none" viewBox="0 0 24 24" stroke="currentColor">
          <path strokeLinecap="round" strokeLinejoin="round" strokeWidth={2} d="M4 6h16M4 12h16M4 18h16" />
        </svg>
      </button>

      {/* Center: Title with unread badge */}
      <div className="flex items-center gap-2">
        <h1 className="text-lg font-semibold text-gray-900">{title}</h1>
        {unreadCount > 0 && (
          <span className="bg-red-500 text-white text-xs font-bold px-2 py-0.5 rounded-full">
            {unreadCount > 99 ? "99+" : unreadCount}
          </span>
        )}
      </div>

      {/* Right: Settings or action button */}
      {rightAction ? (
        rightAction
      ) : (
        <button
          onClick={onSettingsClick}
          className="p-2 -mr-2 text-gray-600 hover:text-gray-900 hover:bg-gray-100 rounded-lg"
          aria-label="Settings"
        >
          <svg className="w-6 h-6" fill="none" viewBox="0 0 24 24" stroke="currentColor">
            <path strokeLinecap="round" strokeLinejoin="round" strokeWidth={2} d="M10.325 4.317c.426-1.756 2.924-1.756 3.35 0a1.724 1.724 0 002.573 1.066c1.543-.94 3.31.826 2.37 2.37a1.724 1.724 0 001.065 2.572c1.756.426 1.756 2.924 0 3.35a1.724 1.724 0 00-1.066 2.573c.94 1.543-.826 3.31-2.37 2.37a1.724 1.724 0 00-2.572 1.065c-.426 1.756-2.924 1.756-3.35 0a1.724 1.724 0 00-2.573-1.066c-1.543.94-3.31-.826-2.37-2.37a1.724 1.724 0 00-1.065-2.572c-1.756-.426-1.756-2.924 0-3.35a1.724 1.724 0 001.066-2.573c-.94-1.543.826-3.31 2.37-2.37.996.608 2.296.07 2.572-1.065z" />
            <path strokeLinecap="round" strokeLinejoin="round" strokeWidth={2} d="M15 12a3 3 0 11-6 0 3 3 0 016 0z" />
          </svg>
        </button>
      )}
    </div>
  );
}
```

**Step 2: Commit**

```bash
cd zettelkasten-front
git add src/components/rss/RssMobileTopBar.tsx
git commit -m "feat(rss): add mobile top bar component

Mobile-only top bar with:
- Hamburger menu (left) for feeds bottom sheet
- Title with unread count badge (center)
- Settings gear (right)

Uses md:hidden to only show on mobile.
"
```

---

### Task 3: Wrap Article List in Responsive Container

**Files:**
- Modify: `zettelkasten-front/src/pages/RssPage.tsx`

**Step 1: Import MobileTopBar**

Add to imports (around line 26):

```tsx
import { RssMobileTopBar } from "../components/rss/RssMobileTopBar";
```

**Step 2: Add mobile top bar handler**

Add after line 222 (after handleRefresh):

```tsx
const handleMobileMenuClick = useCallback(() => {
  setMobileView('feeds');
}, []);
```

**Step 3: Replace article list header**

Find the article list header section (around line 854-862) and wrap the Articles panel:

```tsx
{/* Middle Panel: Articles */}
<div className="w-80 border-r border-gray-200 bg-white flex-shrink-0 flex flex-col hidden md:flex">
```

Change to:

```tsx
{/* Middle Panel: Articles */}
<div className="w-80 border-r border-gray-200 bg-white flex-shrink-0 flex flex-col hidden md:flex md:relative md:z-0">
```

Then BEFORE this div (after the feeds sidebar closes around line 852), add:

```tsx
{/* Mobile Article List View */}
<div className="flex-1 flex flex-col md:hidden">
  <RssMobileTopBar
    title="RSS"
    unreadCount={totalUnreadCount}
    onMenuClick={handleMobileMenuClick}
    onSettingsClick={() => setShowSettingsMenu(true)}
  />

  {/* Articles list content - same as desktop but full width */}
  <div className="flex-1 bg-white flex flex-col overflow-hidden">
    {/* Filter tabs - copied from desktop */}
    <div className="p-4 border-b border-gray-200">
      <div className="flex bg-gray-100 rounded-lg p-1">
        <button
          onClick={() => setShowUnreadOnly(false)}
          className={`flex-1 py-1.5 px-3 rounded-md text-sm font-medium transition-colors ${!showUnreadOnly
            ? "bg-white text-gray-900 shadow-sm"
            : "text-gray-600 hover:text-gray-900"
            }`}
        >
          All
        </button>
        <button
          onClick={() => setShowUnreadOnly(true)}
          className={`flex-1 py-1.5 px-3 rounded-md text-sm font-medium transition-colors relative ${showUnreadOnly
            ? "bg-white text-gray-900 shadow-sm"
            : "text-gray-600 hover:text-gray-900"
            }`}
        >
          Unread
          {currentUnreadCount > 0 && (
            <span className="ml-1 bg-blue-500 text-white text-xs px-1.5 py-0.5 rounded-full">
              {currentUnreadCount}
            </span>
          )}
        </button>
      </div>
    </div>

    {/* Articles list - same items but full width */}
    {articles.length === 0 && !loading ? (
      <div className="flex-1 flex items-center justify-center">
        <p className="text-gray-500">No articles found</p>
      </div>
    ) : (
      <>
        <div className="flex-1 overflow-y-auto p-4 space-y-3">
          {articles.map((article) => (
            <div
              key={article.id}
              onClick={() => handleArticleClick(article)}
              className={`p-4 rounded-lg cursor-pointer transition-colors min-h-[60px] ${selectedArticle?.id === article.id
                ? "bg-blue-100 border-l-4 border-blue-600"
                : article.read
                  ? "bg-gray-50 hover:bg-gray-100"
                  : "bg-white hover:bg-gray-100 border-l-4 border-blue-500 shadow-sm"
                }`}
            >
              <div className="flex items-start gap-3">
                <h3 className="font-semibold text-base line-clamp-3 flex-1 text-gray-900">
                  {article.title}
                </h3>
                {article.card_id && (
                  <svg className="w-5 h-5 text-green-600 flex-shrink-0" fill="currentColor" viewBox="0 0 20 20">
                    <title>Converted to card</title>
                    <path d="M7 3a1 1 0 000 2h6a1 1 0 100-2H7zM4 7a1 1 0 011-1h10a1 1 0 110 2H5a1 1 0 01-1-1zM2 11a2 2 0 012-2h12a2 2 0 012 2v4a2 2 0 01-2 2H4a2 2 0 01-2-2v-4z" />
                  </svg>
                )}
              </div>
              <p className="text-sm text-gray-500 mt-2">
                {getFeedName(article.feed_id)} • {new Date(article.fetched_at).toLocaleDateString()}
              </p>
            </div>
          ))}
        </div>

        {/* Pagination - Load More button for mobile */}
        {totalArticles > articlesPerPage && currentPage * articlesPerPage < totalArticles && (
          <div className="p-4 border-t border-gray-200 bg-gray-50">
            <button
              onClick={() => setCurrentPage(p => p + 1)}
              className="w-full bg-white border border-gray-300 text-gray-700 px-4 py-3 rounded-lg hover:bg-gray-50 transition-colors font-medium"
            >
              Load More Articles ({totalArticles - currentPage * articlesPerPage} remaining)
            </button>
          </div>
        )}
      </>
    )}
  </div>
</div>
```

**Step 4: Hide desktop panels on mobile**

Add `hidden md:flex` class to existing panels (lines 460, 855, 955):

- Line 460: `<div className="w-64...` → `<div className="w-64... hidden md:flex"`
- Line 855: `<div className="w-80...` → `<div className="w-80... hidden md:flex"
- Line 955: `<div className="flex-1...` → `<div className="flex-1... hidden md:flex"

**Step 5: Test and commit**

```bash
cd zettelkasten-front
npm start
# Open browser to localhost:5173, navigate to RSS page
# Test: Mobile view shows article list, desktop shows three panels
# Press Ctrl+C to stop

git add src/pages/RssPage.tsx
git commit -m "feat(rss): add mobile article list view

Mobile-only article list with:
- Full-width layout (md:hidden)
- MobileTopBar with hamburger menu
- Larger touch targets (min-h-[60px])
- Load More button instead of pagination

Desktop panels hidden on mobile with hidden md:flex.
"
```

---

## Phase 2: Mobile Article Reader

### Task 4: Create Mobile Reader Component

**Files:**
- Create: `zettelkasten-front/src/components/rss/RssMobileReader.tsx`

**Step 1: Write the component**

Create file with:

```tsx
import React from "react";
import Markdown from "react-markdown";
import remarkGfm from "remark-gfm";
import { safeHtmlToMarkdown } from "../../utils/markdown";
import { RSSArticle } from "../../api/rss";

interface RssMobileReaderProps {
  article: RSSArticle;
  onBack: () => void;
  onConvert: () => void;
  onMarkAsUnread: () => void;
  getFeedName: (feedId: number) => string;
}

export function RssMobileReader({
  article,
  onBack,
  onConvert,
  onMarkAsUnread,
  getFeedName,
}: RssMobileReaderProps) {
  return (
    <div className="fixed inset-0 bg-white z-50 overflow-y-auto flex flex-col md:hidden animate-slide-up">
      {/* Top bar */}
      <div className="sticky top-0 z-10 bg-white border-b border-gray-200 px-4 py-3 flex items-center justify-between shadow-sm">
        <button
          onClick={onBack}
          className="p-2 -ml-2 text-gray-600 hover:text-gray-900 hover:bg-gray-100 rounded-lg"
          aria-label="Back to articles"
        >
          <svg className="w-6 h-6" fill="none" viewBox="0 0 24 24" stroke="currentColor">
            <path strokeLinecap="round" strokeLinejoin="round" strokeWidth={2} d="M15 19l-7-7 7-7" />
          </svg>
        </button>

        <h2 className="text-base font-semibold text-gray-900 truncate flex-1 mx-4">Article</h2>

        <button
          onClick={onConvert}
          className="p-2 -mr-2 text-blue-600 hover:text-blue-800 hover:bg-blue-50 rounded-lg font-medium text-sm"
        >
          Convert
        </button>
      </div>

      {/* Content */}
      <div className="flex-1">
        <div className="max-w-2xl mx-auto px-4 py-6">
          {/* Title */}
          <h1 className="text-xl font-bold mb-4 text-gray-900 leading-tight">
            {article.title}
          </h1>

          {/* Meta info */}
          <div className="flex flex-wrap items-center gap-3 text-sm text-gray-600 mb-6 pb-4 border-b border-gray-200">
            {article.author && (
              <span className="flex items-center gap-1">
                <svg className="w-4 h-4" fill="currentColor" viewBox="0 0 20 20">
                  <path fillRule="evenodd" d="M10 9a3 3 0 100-6 3 3 0 000 6zm-7 9a7 7 0 1114 0H3z" clipRule="evenodd" />
                </svg>
                {article.author}
              </span>
            )}
            <span className="flex items-center gap-1">
              <svg className="w-4 h-4" fill="currentColor" viewBox="0 0 20 20">
                <path fillRule="evenodd" d="M6 2a1 1 0 00-1 1v1H4a2 2 0 00-2 2v10a2 2 0 002 2h12a2 2 0 002-2V6a2 2 0 00-2-2h-1V3a1 1 0 10-2 0v1H7V3a1 1 0 00-1-1zm0 5a1 1 0 000 2h8a1 1 0 100-2H6z" clipRule="evenodd" />
              </svg>
              {article.published_at
                ? new Date(article.published_at).toLocaleDateString()
                : new Date(article.fetched_at).toLocaleDateString()}
            </span>
            <span className="text-gray-500">
              {getFeedName(article.feed_id)}
            </span>
          </div>

          {/* Content */}
          {article.content && (
            <div className="prose prose-base max-w-none mb-8">
              <Markdown
                remarkPlugins={[remarkGfm]}
                components={{
                  a: ({ href, children, ...props }) => (
                    <a href={href} target="_blank" rel="noopener noreferrer" {...props}>
                      {children}
                    </a>
                  ),
                  img: ({ src, alt, ...props }) => (
                    <img src={src} alt={alt} className="rounded-lg my-4" {...props} />
                  ),
                }}
              >
                {safeHtmlToMarkdown(article.content)}
              </Markdown>
            </div>
          )}
        </div>
      </div>

      {/* Bottom action bar */}
      <div className="sticky bottom-0 bg-white border-t border-gray-200 px-4 py-3 safe-area-inset-bottom">
        <div className="flex gap-2">
          {article.read && (
            <button
              onClick={onMarkAsUnread}
              className="flex-1 bg-gray-100 text-gray-700 px-4 py-3 rounded-lg hover:bg-gray-200 transition-colors flex items-center justify-center gap-2 font-medium"
            >
              <svg className="w-5 h-5" fill="none" viewBox="0 0 24 24" stroke="currentColor">
                <path strokeLinecap="round" strokeLinejoin="round" strokeWidth={2} d="M5 13l4 4L19 7" />
              </svg>
              Mark Unread
            </button>
          )}

          <a
            href={article.url}
            target="_blank"
            rel="noopener noreferrer"
            className="flex-1 bg-gray-100 text-gray-700 px-4 py-3 rounded-lg hover:bg-gray-200 transition-colors flex items-center justify-center gap-2 font-medium text-center"
          >
            <svg className="w-5 h-5" fill="none" viewBox="0 0 24 24" stroke="currentColor">
              <path strokeLinecap="round" strokeLinejoin="round" strokeWidth={2} d="M10 6H6a2 2 0 00-2 2v10a2 2 0 002 2h10a2 2 0 002-2v-4M14 4h6m0 0v6m0-6L10 14" />
            </svg>
            View Original
          </a>

          {!article.card_id && (
            <button
              onClick={onConvert}
              className="flex-1 bg-blue-600 text-white px-4 py-3 rounded-lg hover:bg-blue-700 transition-colors flex items-center justify-center gap-2 font-medium"
            >
              <svg className="w-5 h-5" fill="currentColor" viewBox="0 0 20 20">
                <path d="M7 3a1 1 0 000 2h6a1 1 0 100-2H7zM4 7a1 1 0 011-1h10a1 1 0 110 2H5a1 1 0 01-1-1zM2 11a2 2 0 012-2h12a2 2 0 012 2v4a2 2 0 01-2 2H4a2 2 0 01-2-2v-4z" />
              </svg>
              Convert
            </button>
          )}

          {article.card_id && (
            <button
              onClick={() => window.location.href = `/app/card/${article.card_id}`}
              className="flex-1 bg-green-600 text-white px-4 py-3 rounded-lg hover:bg-green-700 transition-colors flex items-center justify-center gap-2 font-medium"
            >
              <svg className="w-5 h-5" fill="currentColor" viewBox="0 0 20 20">
                <path d="M7 3a1 1 0 000 2h6a1 1 0 100-2H7zM4 7a1 1 0 011-1h10a1 1 0 110 2H5a1 1 0 01-1-1zM2 11a2 2 0 012-2h12a2 2 0 012 2v4a2 2 0 01-2 2H4a2 2 0 01-2-2v-4z" />
              </svg>
              View Card
            </button>
          )}
        </div>
      </div>

      <style>{`
        @keyframes slide-up {
          from {
            transform: translateY(100%);
          }
          to {
            transform: translateY(0);
          }
        }
        .animate-slide-up {
          animation: slide-up 0.3s ease-out;
        }
        .safe-area-inset-bottom {
          padding-bottom: env(safe-area-inset-bottom, 16px);
        }
      `}</style>
    </div>
  );
}
```

**Step 2: Add Tailwind animation**

Check if `tailwind.config.js` has animations. If not, the inline `<style>` tag handles it.

**Step 3: Commit**

```bash
cd zettelkasten-front
git add src/components/rss/RssMobileReader.tsx
git commit -m "feat(rss): add mobile article reader component

Full-screen mobile reader with:
- Slide-up animation
- Back button and Convert action in top bar
- prose-base for better mobile readability
- Sticky bottom action bar with safe area inset
- Mark Unread, View Original, Convert actions

Uses md:hidden to only show on mobile.
"
```

---

### Task 5: Integrate Mobile Reader

**Files:**
- Modify: `zettelkasten-front/src/pages/RssPage.tsx`

**Step 1: Import MobileReader**

Add to imports (around line 32):

```tsx
import { RssMobileReader } from "../components/rss/RssMobileReader";
```

**Step 2: Update handleArticleClick to show mobile reader**

Find handleArticleClick (around line 224) and modify:

```tsx
const handleArticleClick = useCallback(async (article: RSSArticle) => {
  // Prevent duplicate requests for the same article
  if (markingAsRead.has(article.id)) {
    return;
  }

  setSelectedArticle(article);

  // Mobile: show reader view
  if (window.innerWidth < 768) {
    setMobileView('reader');
  }

  if (!article.read) {
    setMarkingAsRead((prev) => new Set(prev).add(article.id));
    try {
      await markAsRead(article.id, true);
      setArticles((prev) =>
        prev.map((a) => (a.id === article.id ? { ...a, read: true } : a))
      );
      // Refresh unread counts (fire and forget, non-blocking)
      refreshUnreadCounts().catch(() => {
        // Silently fail - counts will update on next refresh
      });
    } catch (error) {
      console.error("Failed to mark as read:", error);
      setErrorMessage("Failed to mark article as read. Please try again.");
      setTimeout(() => setErrorMessage(""), 3000);
    } finally {
      setMarkingAsRead((prev) => {
        const next = new Set(prev);
        next.delete(article.id);
        return next;
      });
    }
  }
}, [markingAsRead, refreshUnreadCounts]);
```

**Step 3: Add mobile back handler**

Add after handleArticleClick:

```tsx
const handleMobileBack = useCallback(() => {
  setMobileView('list');
}, []);
```

**Step 4: Render mobile reader conditionally**

Find the end of the component return (before the Dialogs section, around line 1067) and add:

```tsx
{/* Mobile Reader */}
{selectedArticle && mobileView === 'reader' && (
  <RssMobileReader
    article={selectedArticle}
    onBack={handleMobileBack}
    onConvert={handleConvertClick}
    onMarkAsUnread={handleMarkAsUnread}
    getFeedName={getFeedName}
  />
)}
```

**Step 5: Test and commit**

```bash
cd zettelkasten-front
npm start
# Test: On mobile (< 768px), click article → reader slides up
# Test: Back button returns to list
# Test: Desktop still shows side-by-side reader

git add src/pages/RssPage.tsx
git commit -m "feat(rss): integrate mobile reader

- Show mobile reader when article clicked on mobile
- Back button returns to article list
- Desktop reader unchanged
"
```

---

## Phase 3: Bottom Sheet for Feeds

### Task 6: Create Bottom Sheet Component

**Files:**
- Create: `zettelkasten-front/src/components/rss/RssFeedsBottomSheet.tsx`

**Step 1: Write the component**

Create file with:

```tsx
import React, { useEffect, useRef } from "react";
import { RSSFeed, RSSFolder } from "../../api/rss";

interface RssFeedsBottomSheetProps {
  isOpen: boolean;
  onClose: () => void;
  feeds: RSSFeed[];
  folders: RSSFolder[];
  selectedFeedId: number | null;
  selectedFolder: string | null;
  expandedFolders: Set<string>;
  unreadCounts: { feeds: Record<number, number>; folders: Record<string, number> };
  onFeedSelect: (feedId: number | null) => void;
  onFolderSelect: (folder: string | null) => void;
  onToggleFolder: (folderName: string) => void;
  onAddFeed: () => void;
  onCreateFolder: () => void;
  onEditFeed: (feed: RSSFeed) => void;
  onDeleteFeed: (feed: RSSFeed) => void;
  onEditFolder: (folder: RSSFolder) => void;
  onDeleteFolder: (folder: RSSFolder) => void;
  showAddFeedDialog?: boolean;
}

export function RssFeedsBottomSheet({
  isOpen,
  onClose,
  feeds,
  folders,
  selectedFeedId,
  selectedFolder,
  expandedFolders,
  unreadCounts,
  onFeedSelect,
  onFolderSelect,
  onToggleFolder,
  onAddFeed,
  onCreateFolder,
  onEditFeed,
  onDeleteFeed,
  onEditFolder,
  onDeleteFolder,
  showAddFeedDialog,
}: RssFeedsBottomSheetProps) {
  const sheetRef = useRef<HTMLDivElement>(null);
  const backdropRef = useRef<HTMLDivElement>(null);

  // Close on escape key
  useEffect(() => {
    const handleEscape = (e: KeyboardEvent) => {
      if (e.key === 'Escape' && isOpen) {
        onClose();
      }
    };
    document.addEventListener('keydown', handleEscape);
    return () => document.removeEventListener('keydown', handleEscape);
  }, [isOpen, onClose]);

  // Close on backdrop click
  const handleBackdropClick = (e: React.MouseEvent) => {
    if (e.target === backdropRef.current) {
      onClose();
    }
  };

  // Helper functions
  const getFeedsByFolder = (folderName: string | null) => {
    return feeds.filter((f) => f.folder === folderName || (folderName === null && !f.folder));
  };

  const getUnreadCountForFeed = (feedId: number): number => {
    return unreadCounts.feeds[feedId] || 0;
  };

  const getUnreadCountForFolder = (folderName: string): number => {
    return unreadCounts.folders[folderName] || 0;
  };

  const renderUnreadBadge = (count: number) => {
    if (count === 0) return null;
    return (
      <span className="ml-2 bg-red-500 text-white text-xs font-bold px-2 py-0.5 rounded-full">
        {count > 99 ? "99+" : count}
      </span>
    );
  };

  if (!isOpen) return null;

  return (
    <div
      ref={backdropRef}
      onClick={handleBackdropClick}
      className="fixed inset-0 z-50 md:hidden flex items-end justify-end"
    >
      {/* Backdrop */}
      <div className="absolute inset-0 bg-black/50 transition-opacity" />

      {/* Sheet */}
      <div
        ref={sheetRef}
        className="relative w-full max-h-[70vh] bg-white rounded-t-3xl shadow-2xl flex flex-col animate-slide-in"
        onClick={(e) => e.stopPropagation()}
      >
        {/* Drag handle */}
        <div className="flex justify-center pt-3 pb-1" onClick={onClose}>
          <div className="w-10 h-1 bg-gray-300 rounded-full" />
        </div>

        {/* Header */}
        <div className="px-4 py-3 border-b border-gray-200">
          <div className="flex items-center justify-between">
            <h2 className="text-lg font-semibold text-gray-900">Feeds</h2>
            <button
              onClick={onClose}
              className="p-2 -mr-2 text-gray-400 hover:text-gray-600 hover:bg-gray-100 rounded-lg"
            >
              <svg className="w-5 h-5" fill="none" viewBox="0 0 24 24" stroke="currentColor">
                <path strokeLinecap="round" strokeLinejoin="round" strokeWidth={2} d="M6 18L18 6M6 6l12 12" />
              </svg>
            </button>
          </div>
        </div>

        {/* Content */}
        <div className="flex-1 overflow-y-auto px-4 py-4 space-y-4">
          {/* All Feeds button */}
          <button
            onClick={() => {
              onFeedSelect(null);
              onFolderSelect(null);
              onClose();
            }}
            className={`w-full text-left px-4 py-3 rounded-xl transition-colors font-medium ${
              selectedFolder === null && selectedFeedId === null
                ? "bg-blue-100 text-blue-900"
                : "hover:bg-gray-100"
            }`}
          >
            All Feeds ({feeds.length})
          </button>

          {/* Folders */}
          {folders.map((folder) => {
            const folderFeeds = getFeedsByFolder(folder.name);
            const isExpanded = expandedFolders.has(folder.name);
            const isSelected = selectedFolder === folder.name && selectedFeedId === null;
            const unreadCount = getUnreadCountForFolder(folder.name);

            return (
              <div key={folder.id} className="bg-gray-50 rounded-xl p-3">
                <div
                  className={`flex items-center rounded-lg transition-colors ${
                    isSelected ? "bg-blue-100" : "hover:bg-gray-100"
                  }`}
                >
                  <button
                    onClick={() => {
                      onFolderSelect(folder.name);
                      onFeedSelect(null);
                      onClose();
                    }}
                    className="flex-1 text-left px-3 py-2 flex items-center"
                  >
                    <svg
                      className={`w-4 h-4 text-gray-400 mr-2 transition-transform ${
                        isExpanded ? "rotate-90" : ""
                      }`}
                      fill="currentColor"
                      viewBox="0 0 20 20"
                      onClick={(e) => {
                        e.stopPropagation();
                        onToggleFolder(folder.name);
                      }}
                    >
                      <path fillRule="evenodd" d="M7.293 14.707a1 1 0 010-1.414L10.586 10 7.293 6.707a1 1 0 011.414-1.414l4 4a1 1 0 010 1.414l-4 4a1 1 0 01-1.414 0z" clipRule="evenodd" />
                    </svg>
                    <span className="font-medium">{folder.name}</span>
                    <span className="text-gray-400 text-sm ml-2">({folderFeeds.length})</span>
                    {renderUnreadBadge(unreadCount)}
                  </button>
                </div>

                {isExpanded && (
                  <div className="ml-2 mt-2 space-y-1">
                    {folderFeeds.map((feed) => {
                      const unreadCount = getUnreadCountForFeed(feed.id);
                      return (
                        <button
                          key={feed.id}
                          onClick={() => {
                            onFeedSelect(feed.id);
                            onFolderSelect(null);
                            onClose();
                          }}
                          className={`w-full text-left px-3 py-2 rounded-lg text-sm transition-colors ${
                            selectedFeedId === feed.id
                              ? "bg-blue-50 text-blue-900"
                              : "hover:bg-gray-100"
                          }`}
                        >
                          <div className="flex items-center justify-between">
                            <span className="truncate">{feed.name}</span>
                            {renderUnreadBadge(unreadCount)}
                          </div>
                        </button>
                      );
                    })}
                    {folderFeeds.length === 0 && (
                      <div className="px-3 py-2 text-sm text-gray-400 italic">No feeds</div>
                    )}
                  </div>
                )}
              </div>
            );
          })}

          {/* Uncategorized */}
          {(() => {
            const uncategorizedFeeds = getFeedsByFolder(null);
            if (uncategorizedFeeds.length === 0) return null;

            const isExpanded = expandedFolders.has("__uncategorized__");
            return (
              <div className="bg-gray-50 rounded-xl p-3">
                <button
                  onClick={() => onToggleFolder("__uncategorized__")}
                  className="flex items-center px-3 py-2 w-full text-left"
                >
                  <svg
                    className={`w-4 h-4 text-gray-400 mr-2 transition-transform ${
                      isExpanded ? "rotate-90" : ""
                    }`}
                    fill="currentColor"
                    viewBox="0 0 20 20"
                  >
                    <path fillRule="evenodd" d="M7.293 14.707a1 1 0 010-1.414L10.586 10 7.293 6.707a1 1 0 011.414-1.414l4 4a1 1 0 010 1.414l-4 4a1 1 0 01-1.414 0z" clipRule="evenodd" />
                  </svg>
                  <span className="font-medium text-gray-700">Uncategorized</span>
                  <span className="text-gray-400 text-sm ml-2">({uncategorizedFeeds.length})</span>
                </button>

                {isExpanded && (
                  <div className="ml-2 mt-2 space-y-1">
                    {uncategorizedFeeds.map((feed) => {
                      const unreadCount = getUnreadCountForFeed(feed.id);
                      return (
                        <button
                          key={feed.id}
                          onClick={() => {
                            onFeedSelect(feed.id);
                            onFolderSelect(null);
                            onClose();
                          }}
                          className={`w-full text-left px-3 py-2 rounded-lg text-sm transition-colors ${
                            selectedFeedId === feed.id
                              ? "bg-blue-50 text-blue-900"
                              : "hover:bg-gray-100"
                          }`}
                        >
                          <div className="flex items-center justify-between">
                            <span className="truncate">{feed.name}</span>
                            {renderUnreadBadge(unreadCount)}
                          </div>
                        </button>
                      );
                    })}
                  </div>
                )}
              </div>
            );
          })()}

          {/* Action buttons */}
          <div className="pt-2 border-t border-gray-200 space-y-2">
            <button
              onClick={() => {
                onCreateFolder();
                onClose();
              }}
              className="w-full text-left px-4 py-3 text-blue-600 hover:bg-blue-50 rounded-lg transition-colors flex items-center"
            >
              <svg className="w-5 h-5 mr-3" fill="currentColor" viewBox="0 0 20 20">
                <path fillRule="evenodd" d="M10 3a1 1 0 011 1v5h5a1 1 0 110 2h-5v5a1 1 0 11-2 0v-5H4a1 1 0 110-2h5V4a1 1 0 011-1z" clipRule="evenodd" />
              </svg>
              Create New Folder
            </button>
            <button
              onClick={() => {
                onAddFeed();
                onClose();
              }}
              className="w-full bg-blue-600 text-white px-4 py-3 rounded-lg hover:bg-blue-700 transition-colors flex items-center justify-center font-medium"
            >
              <svg className="w-5 h-5 mr-2" fill="currentColor" viewBox="0 0 20 20">
                <path fillRule="evenodd" d="M10 3a1 1 0 011 1v5h5a1 1 0 110 2h-5v5a1 1 0 11-2 0v-5H4a1 1 0 110-2h5V4a1 1 0 011-1z" clipRule="evenodd" />
              </svg>
              Add Feed
            </button>
          </div>
        </div>
      </div>

      <style>{`
        @keyframes slide-in {
          from {
            transform: translateY(100%);
          }
          to {
            transform: translateY(0);
          }
        }
        .animate-slide-in {
          animation: slide-in 0.3s ease-out;
        }
      `}</style>
    </div>
  );
}
```

**Step 2: Commit**

```bash
cd zettelkasten-front
git add src/components/rss/RssFeedsBottomSheet.tsx
git commit -m "feat(rss): add feeds bottom sheet component

Mobile-only bottom sheet with:
- Drag handle at top
- Backdrop with tap to dismiss
- All Feeds, folders (expandable), uncategorized
- Add Feed and Create Folder buttons
- Slide-in animation

Uses md:hidden to only show on mobile.
"
```

---

### Task 7: Integrate Bottom Sheet

**Files:**
- Modify: `zettelkasten-front/src/pages/RssPage.tsx`

**Step 1: Import bottom sheet**

Add to imports (around line 32):

```tsx
import { RssFeedsBottomSheet } from "../components/rss/RssFeedsBottomSheet";
```

**Step 2: Update mobile menu handler**

Modify handleMobileMenuClick (around line 225):

```tsx
const handleMobileMenuClick = useCallback(() => {
  setMobileView('feeds');
}, []);
```

**Step 3: Update feed selection handlers**

Update handleFeedSelect and folder selection to close sheet on mobile. Find existing feed selection logic and add mobileView reset:

After the existing feed selection handlers, add mobile-aware wrappers or modify existing handlers.

Update handleFeedSelect (add after getFeedsByFolder around line 290):

```tsx
const handleFeedSelectMobile = useCallback((feedId: number | null) => {
  setSelectedFeedId(feedId);
  setSelectedFolder(null);
  setShowFeedMenuId(null);
  setMobileView('list');
}, []);
```

Update handleFolderSelectMobile:

```tsx
const handleFolderSelectMobile = useCallback((folderName: string | null) => {
  setSelectedFolder(folderName);
  setSelectedFeedId(null);
  setMobileView('list');
}, []);
```

**Step 4: Render bottom sheet**

Add before the Dialogs section (around line 1068):

```tsx
{/* Mobile Feeds Bottom Sheet */}
<RssFeedsBottomSheet
  isOpen={mobileView === 'feeds'}
  onClose={() => setMobileView('list')}
  feeds={feeds}
  folders={folders}
  selectedFeedId={selectedFeedId}
  selectedFolder={selectedFolder}
  expandedFolders={expandedFolders}
  unreadCounts={unreadCounts}
  onFeedSelect={handleFeedSelectMobile}
  onFolderSelect={handleFolderSelectMobile}
  onToggleFolder={toggleFolderExpanded}
  onAddFeed={() => setShowAddFeedDialog(true)}
  onCreateFolder={() => setShowCreateFolderDialog(true)}
  onEditFeed={handleEditFeed}
  onDeleteFeed={handleDeleteFeed}
  onEditFolder={handleEditFolder}
  onDeleteFolder={handleDeleteFolder}
/>
```

**Step 5: Test and commit**

```bash
cd zettelkasten-front
npm start
# Test: Hamburger menu opens bottom sheet
# Test: Tap feed/folder → sheet closes, articles load
# Test: Swipe down/tap backdrop to close
# Test: Add Feed/Create Folder buttons work

git add src/pages/RssPage.tsx
git commit -m "feat(rss): integrate feeds bottom sheet

Mobile feeds bottom sheet with:
- Opens from hamburger menu
- Feed/folder selection closes sheet and loads articles
- Add Feed and Create Folder buttons
- Backdrop and escape key to close

Mobile workflow complete: list → feeds → reader
"
```

---

## Phase 4: Testing & Polish

### Task 8: Manual Testing Checklist

**Files:**
- Test: Manual browser testing

**Step 1: Desktop testing**

Open browser at >= 768px width, navigate to RSS page:

```bash
cd zettelkasten-front
npm start
```

Checklist:
- [ ] Three-panel layout visible (feeds, articles, reader)
- [ ] Feeds sidebar works (expand folders, select feeds)
- [ ] Article list shows articles
- [ ] Reader panel shows selected article
- [ ] All dialogs work (Add Feed, Convert, etc.)
- [ ] Settings menu works
- [ ] Responsive resize works (drag to < 768px)

**Step 2: Mobile testing**

Use browser DevTools or test on actual mobile device:

Checklist:
- [ ] Article list shows by default (full width)
- [ ] Top bar with hamburger, title, settings
- [ ] Hamburger opens feeds bottom sheet
- [ ] Bottom sheet has drag handle, backdrop
- [ ] Feed selection works and closes sheet
- [ ] Article tap opens reader (slide-up animation)
- [ ] Reader back button returns to list
- [ ] Reader actions work (Convert, View Original, Mark Unread)
- [ ] Bottom action bar visible in reader
- [ ] All/Unread filter tabs work
- [ ] Load More button loads more articles
- [ ] Orientation change handled

**Step 3: Edge cases**

Checklist:
- [ ] No articles state
- [ ] Loading state
- [ ] Error messages display correctly
- [ ] Long article titles truncate properly
- [ ] Large unread counts show "99+"
- [ ] Empty folders show "No feeds"
- [ ] Safe area inset on devices with home indicator

**Step 4: Commit any fixes**

```bash
cd zettelkasten-front
git add .
git commit -m "fix(rss): mobile polish and testing fixes

Fix any issues found during manual testing:
- Safe area inset for home indicator
- Truncation for long titles
- Empty state handling
- Error message positioning
"
```

---

### Task 9: Update Tests (Optional/Time Permitting)

**Files:**
- Modify: `zettelkasten-front/src/pages/__tests__/RssPage.test.tsx`

**Step 1: Add mobile view state tests**

Add test cases for mobile navigation:

```tsx
describe("Mobile navigation", () => {
  beforeEach(() => {
    // Mock window.innerWidth for mobile
    Object.defineProperty(window, 'innerWidth', {
      writable: true,
      configurable: true,
      value: 375,
    });
  });

  it("should show article list by default on mobile", () => {
    render(<RssPage />);
    expect(screen.getByText("RSS")).toBeInTheDocument();
    // Mobile top bar should be visible
    expect(screen.queryByLabelText("Open feeds")).toBeInTheDocument();
  });

  it("should show feeds bottom sheet when menu clicked", () => {
    render(<RssPage />);
    fireEvent.click(screen.getByLabelText("Open feeds"));
    // Bottom sheet should be visible
    expect(screen.getByText("All Feeds")).toBeInTheDocument();
  });

  it("should show reader when article clicked on mobile", () => {
    render(<RssPage />);
    const firstArticle = screen.queryByText(/Test Article/);
    if (firstArticle) {
      fireEvent.click(firstArticle);
      // Reader should be visible
      expect(screen.getByLabelText("Back to articles")).toBeInTheDocument();
    }
  });
});
```

**Step 2: Run tests**

```bash
cd zettelkasten-front
npm test -- src/pages/__tests__/RssPage.test.tsx --run
```

**Step 3: Commit**

```bash
git add src/pages/__tests__/RssPage.test.tsx
git commit -m "test(rss): add mobile navigation tests

Add test coverage for:
- Mobile article list view
- Bottom sheet navigation
- Mobile reader opening
"
```

---

## Completion Checklist

Before marking complete, verify:

- [ ] Desktop unchanged (three-panel layout works)
- [ ] Mobile article list scrolls and taps correctly
- [ ] Mobile reader opens/closes with back button
- [ ] Bottom sheet opens/dismisses properly
- [ ] Feed selection works and closes sheet
- [ ] Orientation changes handled
- [ ] No console errors
- [ ] All commits pushed to feature branch

---

## Final Review

Once implementation is complete:

```bash
cd /home/nick/code/Zettelgarden/.worktrees/rss-responsive-mobile
git log --oneline
```

Expected commits (7-9 commits):
1. feat(rss): add mobile view state
2. feat(rss): add mobile top bar component
3. feat(rss): add mobile article list view
4. feat(rss): add mobile article reader component
5. feat(rss): integrate mobile reader
6. feat(rss): add feeds bottom sheet component
7. feat(rss): integrate feeds bottom sheet
8. fix(rss): mobile polish and testing fixes
9. test(rss): add mobile navigation tests (optional)

Merge back to master when approved.
