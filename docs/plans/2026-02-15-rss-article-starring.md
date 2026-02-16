# RSS Article Starring Implementation Plan

> **For Claude:** REQUIRED SUB-SKILL: Use superpowers:executing-plans to implement this plan task-by-task.

**Goal:** Add ability to star/unstar RSS articles with a separate "Starred" feed in the sidebar.

**Architecture:** Add `is_starred` boolean column to `rss_articles` table. Create POST/DELETE endpoints for starring. Add star toggle in article list and reader view. Add separate Starred feed in sidebar.

**Tech Stack:** Go (backend), React/TypeScript (frontend), PostgreSQL, SQL migrations

---

## Task 1: Database Migration

**Files:**
- Create: `go-backend/schema/0115-add-rss-article-starred.sql`

**Step 1: Write the migration file**

Create migration to add `is_starred` column:

```sql
-- Migration: Add is_starred to rss_articles
-- Description: Add boolean column for starring articles and index for filtering

ALTER TABLE rss_articles ADD COLUMN is_starred BOOLEAN DEFAULT FALSE;

CREATE INDEX idx_rss_articles_starred ON rss_articles(user_id, is_starred);

COMMENT ON COLUMN rss_articles.is_starred IS 'Whether the article is starred by the user';
```

**Step 2: Run migration to verify it works**

Run: `cd go-backend && source .env-bash && go run main.go &` then check schema
Expected: Column and index created successfully
Stop server after verification

**Step 3: Commit**

```bash
git add go-backend/schema/0115-add-rss-article-starred.sql
git commit -m "feat: add is_starred column to rss_articles table"
```

---

## Task 2: Backend Model Update

**Files:**
- Modify: `go-backend/models/rss.go:23-35`

**Step 1: Update RSSArticle struct**

Add `IsStarred` field to the struct:

```go
// RSSArticle represents an article fetched from an RSS feed
type RSSArticle struct {
	ID          int        `json:"id"`
	UserID      int        `json:"user_id"`
	FeedID      int        `json:"feed_id"`
	Title       string     `json:"title"`
	Content     *string    `json:"content,omitempty"`
	Author      *string    `json:"author,omitempty"`
	URL         string     `json:"url"`
	PublishedAt *time.Time `json:"published_at,omitempty"`
	FetchedAt   time.Time  `json:"fetched_at"`
	Read        bool       `json:"read"`
	CardID      *int       `json:"card_id,omitempty"`
	IsStarred   bool       `json:"is_starred"`
}
```

**Step 2: Run existing tests to verify no breakage**

Run: `cd go-backend && source .env-bash && go test ./models/...`
Expected: All tests pass

**Step 3: Commit**

```bash
git add go-backend/models/rss.go
git commit -m "feat: add IsStarred field to RSSArticle model"
```

---

## Task 3: Backend Star/Unstar Handlers

**Files:**
- Modify: `go-backend/handlers/rss.go` (add new functions at end)

**Step 1: Add star handler function**

```go
// StarRSSArticleRoute handles starring an RSS article
func (s *Handler) StarRSSArticleRoute(w http.ResponseWriter, r *http.Request) {
	userID := r.Context().Value("current_user").(int)
	articleID, err := strconv.Atoi(mux.Vars(r)["id"])
	if err != nil {
		http.Error(w, "Invalid article ID", http.StatusBadRequest)
		return
	}

	// Verify article exists and belongs to user
	var exists bool
	err = s.DB.QueryRow(
		"SELECT EXISTS(SELECT 1 FROM rss_articles WHERE id = $1 AND user_id = $2)",
		articleID, userID,
	).Scan(&exists)
	if err != nil || !exists {
		http.Error(w, "Article not found", http.StatusNotFound)
		return
	}

	// Star the article
	_, err = s.DB.Exec(
		"UPDATE rss_articles SET is_starred = TRUE WHERE id = $1 AND user_id = $2",
		articleID, userID,
	)
	if err != nil {
		log.Printf("Error starring article: %v", err)
		http.Error(w, "Failed to star article", http.StatusInternalServerError)
		return
	}

	w.WriteHeader(http.StatusNoContent)
}
```

**Step 2: Add unstar handler function**

```go
// UnstarRSSArticleRoute handles unstarring an RSS article
func (s *Handler) UnstarRSSArticleRoute(w http.ResponseWriter, r *http.Request) {
	userID := r.Context().Value("current_user").(int)
	articleID, err := strconv.Atoi(mux.Vars(r)["id"])
	if err != nil {
		http.Error(w, "Invalid article ID", http.StatusBadRequest)
		return
	}

	// Unstar the article
	_, err = s.DB.Exec(
		"UPDATE rss_articles SET is_starred = FALSE WHERE id = $1 AND user_id = $2",
		articleID, userID,
	)
	if err != nil {
		log.Printf("Error unstarring article: %v", err)
		http.Error(w, "Failed to unstar article", http.StatusInternalServerError)
		return
	}

	w.WriteHeader(http.StatusNoContent)
}
```

**Step 3: Commit**

```bash
git add go-backend/handlers/rss.go
git commit -m "feat: add star/unstar handlers for RSS articles"
```

---

## Task 4: Update List Articles Handler for Starred Filter

**Files:**
- Modify: `go-backend/handlers/rss.go` - find and modify `ListRSSArticlesRoute`

**Step 1: Add starred parameter handling**

Find the `ListRSSArticlesRoute` function and add the starred filter. Look for where query parameters are parsed and add:

```go
// Add to query parameter parsing section
starred := r.URL.Query().Get("starred") == "true"
```

Then modify the SQL query to include the filter. The query WHERE clause should be updated to:

```go
// Build WHERE clause
whereClause := "user_id = $1"
args := []interface{}{userID}
argPos := 2

if folder != nil {
	whereClause += fmt.Sprintf(" AND f.folder = $%d", argPos)
	args = append(args, *folder)
	argPos++
}

if unread {
	whereClause += fmt.Sprintf(" AND a.read = FALSE")
}

// Add starred filter
if starred {
	whereClause += fmt.Sprintf(" AND a.is_starred = TRUE")
}

// Continue with feed_id filter if present...
```

**Step 2: Test the endpoint manually**

Run: `curl -H "Authorization: Bearer YOUR_TOKEN" "http://localhost:8080/api/rss/articles?starred=true"`
Expected: Returns only starred articles

**Step 3: Commit**

```bash
git add go-backend/handlers/rss.go
git commit -m "feat: add starred filter to RSS articles list endpoint"
```

---

## Task 5: Add Routes

**Files:**
- Modify: `go-backend/routes/rss.go` - add new routes after article convert route

**Step 1: Add star/unstar routes**

Add these lines after the convert route (around line 23):

```go
addProtectedRoute(r, h, "/api/rss/articles/{id}/star", h.StarRSSArticleRoute, "POST")
addProtectedRoute(r, h, "/api/rss/articles/{id}/star", h.UnstarRSSArticleRoute, "DELETE")
```

**Step 2: Verify routes are registered**

Run: `cd go-backend && go run main.go &` then `curl -X POST http://localhost:8080/api/rss/articles/1/star -H "Authorization: Bearer YOUR_TOKEN"`
Expected: 204 No Content (or 404 if article doesn't exist)

**Step 3: Commit**

```bash
git add go-backend/routes/rss.go
git commit -m "feat: add star/unstar routes for RSS articles"
```

---

## Task 6: Frontend Type Updates

**Files:**
- Modify: `zettelkasten-front/src/api/rss.ts:20-32`

**Step 1: Add is_starred to RSSArticle interface**

Update the interface:

```typescript
export interface RSSArticle {
  id: number;
  user_id: number;
  feed_id: number;
  title: string;
  content?: string;
  author?: string;
  url: string;
  published_at?: string;
  fetched_at: string;
  read: boolean;
  card_id?: number;
  is_starred?: boolean;
}
```

**Step 2: Add starred to ArticleFilters interface**

Find and modify `ArticleFilters`:

```typescript
export interface ArticleFilters {
  folder?: string;
  unread?: boolean;
  feed_id?: number;
  starred?: boolean;
  limit?: number;
  offset?: number;
}
```

**Step 3: Commit**

```bash
git add zettelkasten-front/src/api/rss.ts
git commit -m "feat: add is_starred to RSSArticle and starred filter types"
```

---

## Task 7: Frontend API Functions

**Files:**
- Modify: `zettelkasten-front/src/api/rss.ts` - add after `markAsRead` function

**Step 1: Add star/unstar API functions**

Add these functions after the `markAsRead` function (around line 196):

```typescript
export async function starArticle(id: number): Promise<void> {
  return getData(apiClient.post<void>(`/rss/articles/${id}/star`, undefined));
}

export async function unstarArticle(id: number): Promise<void> {
  return getData(apiClient.delete<void>(`/rss/articles/${id}/star`));
}
```

**Step 2: Update listArticles to use starred filter**

Find the `listArticles` function and add the starred parameter handling:

```typescript
export function listArticles(filters?: ArticleFilters): Promise<PaginatedArticlesResponse> {
  const params = new URLSearchParams();
  if (filters?.folder) params.set("folder", filters.folder);
  if (filters?.unread) params.set("unread", "true");
  if (filters?.feed_id) params.set("feed_id", filters.feed_id.toString());
  if (filters?.starred) params.set("starred", "true");
  if (filters?.limit) params.set("limit", filters.limit.toString());
  if (filters?.offset) params.set("offset", filters.offset.toString());

  const query = params.toString();
  return getData(apiClient.get<PaginatedArticlesResponse>(`/rss/articles${query ? `?${query}` : ""}`));
}
```

**Step 3: Commit**

```bash
git add zettelkasten-front/src/api/rss.ts
git commit -m "feat: add star/unstar API functions and starred filter"
```

---

## Task 8: RssPage State and Handlers

**Files:**
- Modify: `zettelkasten-front/src/pages/RssPage.tsx`

**Step 1: Add isStarredFeedActive state**

Find the state declarations (around line 57) and add:

```typescript
const [isStarredFeedActive, setIsStarredFeedActive] = useState(false);
```

**Step 2: Import star/unstar functions**

Add to imports at top:

```typescript
import {
  // ... existing imports
  starArticle,
  unstarArticle,
} from "../api/rss";
```

**Step 3: Add handleStarArticle handler**

Add after `handleMarkAsUnread` (around line 240):

```typescript
const handleStarArticle = useCallback(async (articleId: number) => {
  try {
    await starArticle(articleId);
    updateArticle(articleId, { is_starred: true });
  } catch (error) {
    console.error("Failed to star article:", error);
    setErrorMessage("Failed to star article. Please try again.");
    setTimeout(() => setErrorMessage(""), 3000);
  }
}, [updateArticle]);

const handleUnstarArticle = useCallback(async (articleId: number) => {
  try {
    await unstarArticle(articleId);
    updateArticle(articleId, { is_starred: false });
  } catch (error) {
    console.error("Failed to unstar article:", error);
    setErrorMessage("Failed to unstar article. Please try again.");
    setTimeout(() => setErrorMessage(""), 3000);
  }
}, [updateArticle]);
```

**Step 4: Add handleSelectStarredFeed handler**

Add after `handleSelectSmartFeed`:

```typescript
const handleSelectStarredFeed = useCallback(() => {
  setIsStarredFeedActive(true);
  setIsSmartFeedActive(false);
  setSelectedFolder(null);
  setSelectedFeedId(null);
}, []);
```

**Step 5: Update article filters to include starred**

Find `articleFilters` useMemo and add starred:

```typescript
const articleFilters = useMemo(() => ({
  folder: selectedFolder ?? undefined,
  feed_id: selectedFeedId ?? undefined,
  unread: showUnreadOnly || undefined,
  starred: isStarredFeedActive || undefined,
}), [selectedFolder, selectedFeedId, showUnreadOnly, isStarredFeedActive]);
```

**Step 6: Update reset effect to include starred feed**

Find the `useEffect` that resets page and add `isStarredFeedActive`:

```typescript
useEffect(() => {
  resetToFirstPage();
}, [selectedFolder, selectedFeedId, showUnreadOnly, isSmartFeedActive, isStarredFeedActive, resetToFirstPage]);
```

**Step 7: Commit**

```bash
git add zettelkasten-front/src/pages/RssPage.tsx
git commit -m "feat: add state and handlers for starred feed"
```

---

## Task 9: RssArticlesPanel Star Icon

**Files:**
- Modify: `zettelkasten-front/src/components/rss/RssArticlesPanel.tsx`

**Step 1: Add onToggleStar prop**

Add to props interface:

```typescript
interface RssArticlesPanelProps {
  // ... existing props
  onToggleStar: (articleId: number, isStarred: boolean) => void;
}
```

**Step 2: Add star icon to article list items**

Find the article div rendering (around line 112) and add the star icon. Update the section with the smart score and card indicators:

```typescript
<div className="flex items-start gap-2">
  <h3 className="font-medium text-sm line-clamp-2 mb-1 flex-1">
    {article.title}
  </h3>
  <div className="flex items-center gap-1 flex-shrink-0">
    {/* Add star icon here */}
    <button
      onClick={(e) => {
        e.stopPropagation();
        onToggleStar(article.id, article.is_starred || false);
      }}
      className="p-0.5 hover:bg-gray-200 rounded"
      title={article.is_starred ? "Unstar" : "Star"}
    >
      <svg
        className={`w-4 h-4 ${article.is_starred ? "text-amber-500 fill-amber-500" : "text-gray-400"}`}
        fill={article.is_starred ? "currentColor" : "none"}
        stroke="currentColor"
        viewBox="0 0 20 20"
      >
        <path
          strokeLinecap="round"
          strokeLinejoin="round"
          strokeWidth={article.is_starred ? 0 : 2}
          d="M11.049 2.927c.3-.921 1.603-.921 1.902 0l1.519 4.674a1 1 0 00.95.69h4.915c.969 0 1.371 1.24.588 1.81l-3.976 2.888a1 1 0 00-.364 1.118l1.518 4.674c.3.922-.755 1.688-1.538 1.118l-3.976-2.888a1 1 0 00-1.176 0l-3.976 2.888c-.783.57-1.838-.197-1.538-1.118l1.518-4.674a1 1 0 00-.364-1.118l-3.976-2.888c-.784-.57-.38-1.81.588-1.81h4.914a1 1 0 00.951-.69l1.519-4.674z"
        />
      </svg>
    </button>
    {hasSmartScore && articleWithScore.smart_score && (
      // ... existing smart score code
    )}
    {article.card_id && (
      // ... existing card indicator code
    )}
  </div>
</div>
```

**Step 3: Commit**

```bash
git add zettelkasten-front/src/components/rss/RssArticlesPanel.tsx
git commit -m "feat: add star icon toggle to article list items"
```

---

## Task 10: RssReaderPanel Star Button

**Files:**
- Modify: `zettelkasten-front/src/components/rss/RssReaderPanel.tsx`

**Step 1: Add onToggleStar prop**

Find the props interface and add:

```typescript
interface RssReaderPanelProps {
  // ... existing props
  onToggleStar: (articleId: number, isStarred: boolean) => void;
}
```

**Step 2: Add star button to toolbar**

Find the toolbar section with Convert and Mark as Unread buttons. Add a star button:

```typescript
// In the toolbar button section, add:
<button
  onClick={() => selectedArticle && onToggleStar(selectedArticle.id, selectedArticle.is_starred || false)}
  className={`flex items-center gap-2 px-4 py-2 rounded-lg border ${
    selectedArticle?.is_starred
      ? "bg-amber-50 border-amber-300 text-amber-700"
      : "bg-white border-gray-300 text-gray-700 hover:bg-gray-50"
  }`}
  title={selectedArticle?.is_starred ? "Unstar article" : "Star article"}
>
  <svg
    className={`w-5 h-5 ${selectedArticle?.is_starred ? "fill-amber-500 text-amber-500" : "text-gray-500"}`}
    fill={selectedArticle?.is_starred ? "currentColor" : "none"}
    stroke="currentColor"
    viewBox="0 0 20 20"
  >
    <path
      strokeLinecap="round"
      strokeLinejoin="round"
      strokeWidth={selectedArticle?.is_starred ? 0 : 2}
      d="M11.049 2.927c.3-.921 1.603-.921 1.902 0l1.519 4.674a1 1 0 00.95.69h4.915c.969 0 1.371 1.24.588 1.81l-3.976 2.888a1 1 0 00-.364 1.118l1.518 4.674c.3.922-.755 1.688-1.538 1.118l-3.976-2.888a1 1 0 00-1.176 0l-3.976 2.888c-.783.57-1.838-.197-1.538-1.118l1.518-4.674a1 1 0 00-.364-1.118l-3.976-2.888c-.784-.57-.38-1.81.588-1.81h4.914a1 1 0 00.951-.69l1.519-4.674z"
    />
  </svg>
  <span>{selectedArticle?.is_starred ? "Starred" : "Star"}</span>
</button>
```

**Step 3: Commit**

```bash
git add zettelkasten-front/src/components/rss/RssReaderPanel.tsx
git commit -m "feat: add star button to reader panel toolbar"
```

---

## Task 11: RssFeedsPanel Starred Feed Item

**Files:**
- Modify: `zettelkasten-front/src/components/rss/RssFeedsPanel.tsx`

**Step 1: Add starred feed props to interface**

Update props:

```typescript
interface RssFeedsPanelProps {
  // ... existing props
  isStarredFeedActive: boolean;
  starredCount?: number;
  onSelectStarredFeed: () => void;
}
```

**Step 2: Add starred feed item to sidebar**

Find the Smart Feed item and add a Starred feed item after it. Look for the smart feed rendering section and add:

```typescript
{/* Add this after the Smart Feed item */}
<button
  onClick={onSelectStarredFeed}
  className={`w-full flex items-center gap-2 px-3 py-2 rounded-md text-sm transition-colors ${
    isStarredFeedActive
      ? "bg-amber-100 text-amber-900 font-medium"
      : "text-gray-700 hover:bg-gray-100"
  }`}
>
  <svg
    className={`w-5 h-5 ${isStarredFeedActive ? "fill-amber-500 text-amber-500" : "text-gray-500"}`}
    fill={isStarredFeedActive ? "currentColor" : "none"}
    stroke="currentColor"
    viewBox="0 0 20 20"
  >
    <path
      strokeLinecap="round"
      strokeLinejoin="round"
      strokeWidth={isStarredFeedActive ? 0 : 2}
      d="M11.049 2.927c.3-.921 1.603-.921 1.902 0l1.519 4.674a1 1 0 00.95.69h4.915c.969 0 1.371 1.24.588 1.81l-3.976 2.888a1 1 0 00-.364 1.118l1.518 4.674c.3.922-.755 1.688-1.538 1.118l-3.976-2.888a1 1 0 00-1.176 0l-3.976 2.888c-.783.57-1.838-.197-1.538-1.118l1.518-4.674a1 1 0 00-.364-1.118l-3.976-2.888c-.784-.57-.38-1.81.588-1.81h4.914a1 1 0 00.951-.69l1.519-4.674z"
    />
  </svg>
  <span>Starred</span>
  {starredCount !== undefined && starredCount > 0 && (
    <span className="ml-auto text-xs bg-gray-200 px-2 py-0.5 rounded-full">
      {starredCount}
    </span>
  )}
</button>
```

**Step 3: Commit**

```bash
git add zettelkasten-front/src/components/rss/RssFeedsPanel.tsx
git commit -m "feat: add Starred feed item to sidebar"
```

---

## Task 12: Wire Everything Together in RssPage

**Files:**
- Modify: `zettelkasten-front/src/pages/RssPage.tsx`

**Step 1: Calculate starred count**

Add after `totalUnreadCount` useMemo:

```typescript
// Calculate starred count
const starredCount = useMemo(() => {
  return articles.filter(a => a.is_starred).length;
}, [articles]);
```

**Step 2: Pass starred props to RssDesktopLayout**

Update the `RssDesktopLayout` component call to include:

```typescript
<RssDesktopLayout
  // ... existing props
  isStarredFeedActive={isStarredFeedActive}
  starredCount={starredCount}
  onSelectStarredFeed={handleSelectStarredFeed}
  onToggleStar={async (articleId, isStarred) => {
    if (isStarred) {
      await handleUnstarArticle(articleId);
    } else {
      await handleStarArticle(articleId);
    }
  }}
  // ... rest of props
/>
```

**Step 3: Update RssDesktopLayout props type**

Update the `RssDesktopLayoutProps` interface to include:

```typescript
interface RssDesktopLayoutProps {
  // ... existing props
  isStarredFeedActive: boolean;
  starredCount?: number;
  onSelectStarredFeed: () => void;
  onToggleStar: (articleId: number, isStarred: boolean) => void;
}
```

**Step 4: Wire through RssDesktopLayout to children**

Update `RssDesktopLayout.tsx` to pass props through to `RssFeedsPanel`, `RssArticlesPanel`, and `RssReaderPanel`.

**Step 5: Commit**

```bash
git add zettelkasten-front/src/pages/RssPage.tsx
git add zettelkasten-front/src/components/rss/RssDesktopLayout.tsx
git commit -m "feat: wire starred feed and star toggle through RSS page"
```

---

## Task 13: Mobile Layout Support

**Files:**
- Modify: `zettelkasten-front/src/components/rss/RssMobileLayout.tsx`

**Step 1: Add starred props to RssMobileLayout**

Update props interface to match desktop:

```typescript
interface RssMobileLayoutProps {
  // ... existing props
  isStarredFeedActive: boolean;
  starredCount?: number;
  onSelectStarredFeed: () => void;
  onToggleStar: (articleId: number, isStarred: boolean) => void;
}
```

**Step 2: Pass starred feed handler to mobile components**

Add `onSelectStarredFeed={onSelectStarredFeed}` to relevant mobile components.

**Step 3: Update RssPage to pass mobile props**

Update the `RssMobileLayout` call in `RssPage.tsx` to include starred props.

**Step 4: Commit**

```bash
git add zettelkasten-front/src/components/rss/RssMobileLayout.tsx
git add zettelkasten-front/src/pages/RssPage.tsx
git commit -m "feat: add starred feed support to mobile layout"
```

---

## Task 14: Testing

**Step 1: Manual testing - Backend**

```bash
cd go-backend
source .env-bash
go run main.go
```

Test endpoints:
```bash
# Get articles
curl -H "Authorization: Bearer YOUR_TOKEN" http://localhost:8080/api/rss/articles

# Star an article
curl -X POST -H "Authorization: Bearer YOUR_TOKEN" http://localhost:8080/api/rss/articles/1/star

# Get starred articles
curl -H "Authorization: Bearer YOUR_TOKEN" http://localhost:8080/api/rss/articles?starred=true

# Unstar
curl -X DELETE -H "Authorization: Bearer YOUR_TOKEN" http://localhost:8080/api/rss/articles/1/star
```

**Step 2: Manual testing - Frontend**

```bash
cd zettelkasten-front
npm start
```

Test flows:
1. Star an article from article list - verify icon fills
2. Unstar the same article - verify icon unfills
3. Star from reader view - verify button updates
4. Click "Starred" in sidebar - verify only starred articles show
5. Refresh page - verify starred state persists

**Step 3: Write tests for critical backend functions**

Create `go-backend/handlers/rss_test.go` if not exists, add:

```go
func TestStarRSSArticleRoute(t *testing.T) {
    // Test starring an article
    // Test unstarring an article
    // Test starring non-existent article returns 404
}
```

**Step 4: Commit tests**

```bash
git add go-backend/handlers/rss_test.go
git commit -m "test: add tests for star/unstar article endpoints"
```

---

## Task 15: Documentation

**Files:**
- Modify: `CLAUDE.md` - update RSS features section

**Step 1: Update CLAUDE.md**

Add to RSS section:

```markdown
### RSS Feed Client (continued)
- **Starring**: Star/unstar articles for later reference
  - Star icon in article list and reader view
  - Dedicated Starred feed in sidebar
  - Filtered API endpoint for starred articles
```

**Step 2: Commit**

```bash
git add CLAUDE.md
git commit -m "docs: update RSS starring feature documentation"
```

---

## Completion Checklist

- [ ] All 15 tasks completed
- [ ] All tests passing
- [ ] Manual testing successful
- [ ] Documentation updated
- [ ] Clean git history with logical commits
