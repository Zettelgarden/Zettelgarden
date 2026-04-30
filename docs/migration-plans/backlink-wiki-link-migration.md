# Migration Plan: `[card_id]` → `[[card_id]]` Wiki-Link Syntax

## Overview

Switch the backlink syntax from `[card_id]` to `[[card_id]]` (and optionally `[[card_id|display text]]`). This eliminates ambiguity with standard markdown links, removes fragile parsing hacks, improves readability, and makes future export to Obsidian/Logseq/Foam essentially free.

## Why

The current `[card_id]` syntax has systemic problems:

1. **Ambiguous with markdown links** — requires `isMarkdownLink()` exclusion check in parser
2. **Fragile frontend rendering** — `preprocessCardLinks()` converts `[id]` to `[id](#)`, then the `<a>` renderer intercepts `href="#"`, which can collide with real anchor links
3. **No display text** — `[42]` is cryptic in raw text
4. **Duplicate logic** — backlink extraction exists in both `services/cards.go` AND `services/tools/card/card.go` with identical bugs
5. **Dead columns** — `backlinks.source_id TEXT` and `backlinks.target_id TEXT` are defined but never written to
6. **Non-standard** — every other zettelkasten tool uses `[[wiki-links]]`

## New Syntax

| Old | New | Notes |
|-----|-----|-------|
| `[42]` | `[[42]]` | Basic link by card ID |
| `[1.3]` | `[[1.3]]` | Child card link |
| `[42] - Meeting Notes` (in addBacklinkToBody) | `[[42\|Meeting Notes]]` | Link with display text |
| `[text](url)` | `[text](url)` | Unchanged — still standard markdown |

The pipe `|` for display text follows the same convention as Obsidian, Logseq, and Foam.

---

## Phase 0: Prerequisite — Deduplicate Backlink Logic

**Before** touching the syntax, consolidate the two copies of backlink extraction.

### Current duplication

| Function | `services/cards.go` | `services/tools/card/card.go` |
|----------|---------------------|-------------------------------|
| Extract from body | `ExtractBacklinks()` | `extractBacklinks()` |
| Extract from structured data | `ExtractBacklinksFromStructuredData()` | `extractBacklinksFromStructuredData()` |
| Write to DB | `UpdateBacklinks()` | `updateBacklinks()` |
| Markdown link check | `isMarkdownLink()` | `isMarkdownLink()` |

### Action

1. Delete the four functions from `services/tools/card/card.go`
2. Have `services/tools/card/card.go` call the public functions from `services/cards.go`
3. Run `go test ./...` to verify no regressions

**Estimated time:** 1 hour

---

## Phase 1: Backend — Update Extraction and Storage

### 1.1 Update `ExtractBacklinks` in `services/cards.go`

**File:** `go-backend/services/cards.go`

Replace the current implementation:

```go
// OLD
func ExtractBacklinks(text string) []string {
    re := regexp.MustCompile(`\[([^\]]+)\]`)
    matches := re.FindAllStringSubmatch(text, -1)
    var backlinks []string
    for _, match := range matches {
        if len(match) > 1 {
            if !isMarkdownLink(text, match[0]) {
                backlinks = append(backlinks, match[1])
            }
        }
    }
    return backlinks
}
```

With:

```go
// NEW
func ExtractBacklinks(text string) []string {
    // Match [[card_id]] or [[card_id|display text]]
    re := regexp.MustCompile(`\[\[([^\]|]+?)(?:\|[^\]]*)?\]\]`)
    matches := re.FindAllStringSubmatch(text, -1)
    var backlinks []string
    for _, match := range matches {
        if len(match) > 1 && strings.TrimSpace(match[1]) != "" {
            backlinks = append(backlinks, strings.TrimSpace(match[1]))
        }
    }
    return backlinks
}
```

Key changes:
- Regex targets `[[...]]` — no ambiguity with markdown `[text](url)`
- Supports optional `|display text` after the card_id
- `isMarkdownLink()` check is no longer needed
- Empty `[[]]` is excluded

### 1.2 Delete `isMarkdownLink` helper

**File:** `go-backend/services/cards.go`

Remove the `isMarkdownLink()` function entirely. It's no longer called.

### 1.3 Update tests

**File:** `go-backend/services/references_test.go`

Update `TestExtractBacklinks_Duplicates` and `TestGetReferences_MalformedBacklinks_Integration`:

```go
// OLD test cases
{input: "This has [link1] and [link2]", expected: []string{"link1", "link2"}},
{input: "[a] [b] [a] [c] [b] [a]", expected: []string{"a", "b", "a", "c", "b", "a"}},

// NEW test cases
{input: "This has [[link1]] and [[link2]]", expected: []string{"link1", "link2"}},
{input: "[[a]] [[b]] [[a]] [[c]] [[b]] [[a]]", expected: []string{"a", "b", "a", "c", "b", "a"}},
{input: "[[42|Meeting Notes]]", expected: []string{"42"}},  // display text is ignored
{input: "[[1.3]]", expected: []string{"1.3"}},
```

Also update the malformed backlink tests:
```go
// OLD: markdown link exclusion test
{name: "markdown link syntax", body: "This has [text](url)", expected: 0},
{name: "valid backlinks only", body: "This has [valid1] and [valid2]", expected: 2},

// NEW: markdown links are irrelevant, wiki-links are separate
{name: "markdown link not matched", body: "This has [text](url)", expected: 0},
{name: "valid backlinks only", body: "This has [[valid1]] and [[valid2]]", expected: 2},
{name: "wiki-link with display text", body: "This has [[valid1|Some Title]]", expected: 1},
```

Update `TestGetDirectLinks_MixedExistence_Integration`:
```go
// OLD
Body: "References [existing_target], [nonexistent1], and [nonexistent2]",
// NEW
Body: "References [[existing_target]], [[nonexistent1]], and [[nonexistent2]]",
```

### 1.4 No changes needed to

- `UpdateBacklinks()` — still receives `[]string` of card_ids, unchanged
- `GetBacklinks()` — queries the `backlinks` table, unchanged
- `GetReferences()` / `GetDirectLinks()` / `CategorizeReferences()` — call `ExtractBacklinks`, unchanged interface
- `ExtractBacklinksFromStructuredData()` — extracts from JSONB, not from body text, unchanged
- `backlinks` table schema — `source_id_int`/`target_id_int` still work fine

**Estimated time:** 2-3 hours

---

## Phase 2: Frontend — Update Rendering and Input

### 2.1 Replace `preprocessCardLinks` in `CardBody.tsx`

**File:** `zettelkasten-front/src/components/cards/CardBody.tsx`

Delete `preprocessCardLinks`:
```typescript
// DELETE THIS
function preprocessCardLinks(body: string): string {
  return body.replace(/\[([A-Za-z0-9_.-/]+)\](?!\()/g, "[$1](#)");
}
```

Add a new `preprocessWikiLinks`:
```typescript
function preprocessWikiLinks(body: string): string {
  // Convert [[card_id]] and [[card_id|display text]] to markdown links
  // [[42]] → [42](zg://card/42)
  // [[42|Meeting Notes]] → [Meeting Notes](zg://card/42)
  return body.replace(/\[\[([^\]|]+?)(?:\|([^\]]*))?\]\]/g, (_match, cardId, displayText) => {
    const label = displayText || cardId;
    return `[${label}](zg://card/${cardId})`;
  });
}
```

Update `useCardMarkdown` to call `preprocessWikiLinks` instead of `preprocessCardLinks`.

### 2.2 Update the `<a>` component in `CardBody.tsx`

**File:** `zettelkasten-front/src/components/cards/CardBody.tsx`

```typescript
// OLD
a({ children, href, ...props }: any) {
  if (href === "#") {
    const cardId = children as string;
    return <CardLinkWithPreview currentCard={card} card_id={cardId} ... />;
  }
  ...
}

// NEW
a({ children, href, ...props }: any) {
  if (href?.startsWith("zg://card/")) {
    const cardId = href.replace("zg://card/", "");
    return <CardLinkWithPreview currentCard={card} card_id={cardId} ... />;
  }
  ...
}
```

This fixes the `href="#"` collision with real anchor links.

### 2.3 Update `addBacklinkToBody` in `cardActions.ts`

**File:** `zettelkasten-front/src/utils/cardActions.ts`

```typescript
// OLD
export function addBacklinkToBody(card: Card, selectedCard: PartialCard): Card {
  const backlinkText = "\n\n[" + selectedCard.card_id + "] - " + selectedCard.title;
  return { ...card, body: card.body + backlinkText };
}

// NEW
export function addBacklinkToBody(card: Card, selectedCard: PartialCard): Card {
  const backlinkText = "\n\n[[" + selectedCard.card_id + "|" + selectedCard.title + "]]";
  return { ...card, body: card.body + backlinkText };
}
```

### 2.4 Update `useCardReference` hook

**File:** `zettelkasten-front/src/components/cards/useCardReference.ts`

Change the trigger from `[` to `[[`:

```typescript
// OLD: triggers on single [
handleReferenceSelect: inserts "[" + card.card_id + "]"
handleBracketKey: triggers on event.key === '['

// NEW: triggers on double [[
handleReferenceSelect: inserts "[[" + card.card_id + "]]"
handleBracketKey: detect when user types second [, show dialog
```

This is the most delicate frontend change. The user types `[`, nothing happens (normal bracket). On the second `[`, the reference dialog opens. On selection, `[` + `card_id + `]]` is inserted.

**Alternative simpler approach:** Keep triggering on `[`, but always insert `[[card_id]]` on selection. This avoids changing the trigger logic — just changes what gets inserted. Users who want a literal `[` can just type it (it won't be a backlink anymore).

I recommend the simpler approach for the initial migration.

### 2.5 Update `CardLinkWithPreview` component

**File:** `zettelkasten-front/src/components/cards/CardLinkWithPreview.tsx`

This component currently receives a `card_id` string prop and shows it as `[card_id] title`. Update the display:

```typescript
// OLD display
<span>[{card_id}]</span>

// NEW display — show card_id, but rendering comes from the parent <a> component
// The card_id is still passed as a prop, no change needed to the interface
```

Minimal change — the display format `[42]` in the *UI* (not in storage) is fine to keep or update to `[[42]]`. The key change is in body storage.

### 2.6 Update `CardBodyTextArea.tsx` preview mode

**File:** `zettelkasten-front/src/components/cards/CardBodyTextArea.tsx`

The preview tab uses `<ReactMarkdown>` on the raw body. Currently `[card_id]` renders as plain text (no link). After the change, `[[card_id]]` will also render as plain text in the simple preview.

Consider using the same `preprocessWikiLinks` function here too for consistent preview rendering.

### 2.7 Update front-end tests

**File:** `zettelkasten-front/src/components/cards/CardBody.test.ts`

Add tests for `preprocessWikiLinks`:
```typescript
describe("preprocessWikiLinks", () => {
  it("converts [[card_id]] to markdown link", () => {
    expect(preprocessWikiLinks("[[42]]")).toBe("[42](zg://card/42)");
  });
  it("converts [[card_id|display text]] to markdown link with display text", () => {
    expect(preprocessWikiLinks("[[42|Meeting Notes]]")).toBe("[Meeting Notes](zg://card/42)");
  });
  it("does not touch markdown links", () => {
    const input = "[click here](https://example.com)";
    expect(preprocessWikiLinks(input)).toBe(input);
  });
  it("does not touch single brackets", () => {
    const input = "[some regular text]";
    expect(preprocessWikiLinks(input)).toBe(input);
  });
});
```

**Estimated time:** 3-4 hours

---

## Phase 3: Data Migration — Convert Existing Bodies

This is the riskiest phase. Every card body in the database needs its `[card_id]` references converted to `[[card_id]]`.

### 3.1 Migration strategy

**Do NOT use a raw SQL regex.** The `[text](url)` exclusion is too complex for PostgreSQL regex and risks corrupting markdown links.

Instead, write a Go migration script that:
1. Loads cards in batches
2. Uses the same `ExtractBacklinks` logic to find old-style links
3. Replaces them with `[[card_id]]`
4. Validates the result (markdown links are preserved)
5. Updates the body and re-runs backlink extraction

### 3.2 Migration script

**New file:** `go-backend/cmd/backlink_migration/main.go`

```go
package main

import (
    "database/sql"
    "fmt"
    "log"
    "regexp"
    "strings"

    _ "github.com/lib/pq"
)

// Old regex — matches [card_id] that is NOT followed by (
var oldBacklinkRegex = regexp.MustCompile(`\[([^\]]+)\]`)

func isMarkdownLink(text string, match string) bool {
    pos := strings.Index(text, match)
    if pos == -1 {
        return false
    }
    if pos+len(match) < len(text) && text[pos+len(match)] == '(' {
        return true
    }
    return false
}

// Known card ID pattern: alphanumeric, dots, hyphens, underscores, slashes
var cardIDPattern = regexp.MustCompile(`^[A-Za-z0-9_.\-/]+$`)

func migrateBody(body string, validCardIDs map[string]bool) (string, bool) {
    matches := oldBacklinkRegex.FindAllStringSubmatchIndex(body, -1)

    if len(matches) == 0 {
        return body, false
    }

    // Process matches in reverse to preserve indices
    changed := false
    result := body

    for i := len(matches) - 1; i >= 0; i-- {
        match := matches[i]
        fullMatchStart := match[0]
        fullMatchEnd := match[1]
        contentStart := match[2]
        contentEnd := match[3]

        content := body[contentStart:contentEnd]

        // Skip if followed by ( — it's a markdown link
        if isMarkdownLink(body, body[fullMatchStart:fullMatchEnd]) {
            continue
        }

        // Skip if content doesn't look like a card ID
        if !cardIDPattern.MatchString(content) {
            continue
        }

        // Optional: only convert if the card ID actually exists
        // (comment out for dry-run or to convert all)
        // if !validCardIDs[content] {
        //     continue
        // }

        // Replace [card_id] with [[card_id]]
        replacement := "[[" + content + "]]"
        result = result[:fullMatchStart] + replacement + result[fullMatchEnd:]
        changed = true
    }

    return result, changed
}

func main() {
    // Connect to DB using env vars
    // Batch through all cards
    // For each card, run migrateBody
    // If changed, UPDATE cards SET body = $1 WHERE id = $2
    // Then re-run backlink extraction (call UpdateBacklinks)
    //
    // Include --dry-run flag that logs changes without writing
    // Include --batch-size flag (default 100)
    // Log progress: "Processed 100/5000 cards, updated 42"
}
```

### 3.3 Validation

After migration:

1. **Spot check** — Query 20 random cards and verify:
   - `[[card_id]]` for card references
   - `[text](url)` preserved for markdown links
   - `{{tasks:...}}` and `{{schema:...}}` untouched

2. **Regression query** — Find any remaining old-style backlinks:
   ```sql
   SELECT id, card_id, body
   FROM cards
   WHERE body ~ '\[[A-Za-z0-9_.\-/]+\]'  -- old-style brackets
     AND NOT body ~ '\[\['                 -- no wiki-links (hasn't been migrated)
     AND is_deleted = false;
   ```

3. **Backlink integrity** — Verify backlink counts match:
   ```sql
   SELECT COUNT(*) FROM backlinks;  -- should be same or higher after re-extraction
   ```

**Estimated time:** 3-4 hours (script + testing + running)

---

## Phase 4: Cleanup

### 4.1 Remove dead columns from `backlinks` table

**Migration file:** `go-backend/schema/0143-drop-backlink-text-columns.sql`

```sql
ALTER TABLE backlinks DROP COLUMN IF EXISTS source_id;
ALTER TABLE backlinks DROP COLUMN IF EXISTS target_id;
```

### 4.2 Clean up `isMarkdownLink`

Already removed in Phase 1, but verify no other references exist:
```bash
grep -r "isMarkdownLink" go-backend/
```

### 4.3 Update `CardBodyHelpPopover.tsx`

**File:** `zettelkasten-front/src/components/cards/CardBodyHelpPopover.tsx`

```typescript
// OLD
{
  name: "Card Links",
  description: "Link to other cards by their ID",
  syntax: "[<CardID>]",
  example: "[my-note]\n[20250201-reading-list]"
}

// NEW
{
  name: "Card Links",
  description: "Link to other cards by their ID, optionally with display text",
  syntax: "[[<CardID>]] or [[<CardID>|display text]]",
  example: "[[my-note]]\n[[20250201-reading-list|January Reading]]"
}
```

### 4.4 Clean up display-only `[card_id]` in UI components

These are NOT in card bodies — they're just UI labels like `[42] Meeting Notes`. Decide whether to update these to `[[42]]` or leave as-is. Recommendation: leave as `[42]` in UI chrome since it's just a visual label, not stored syntax.

Files that show `[card_id]` as UI labels (not body text):
- `ViewPageHeader.tsx`
- `CardTreeItem.tsx`
- `EditorToolbar.tsx`
- `CardIdDiscoveryDialog.tsx`
- `ChatSidebar.tsx`
- `ChatInterface.tsx`
- `CardPreview.tsx`
- `TaskListItem.tsx`
- `FactDialog.tsx`
- `SearchCardDetailPanel.tsx`

No code changes needed — these are just display strings.

**Estimated time:** 1 hour

---

## Phase 5: Verification and Rollout

### 5.1 Test matrix

| Scenario | Test |
|----------|------|
| New card with `[[42]]` | Backlinks table updated correctly |
| New card with `[[42\|Meeting]]` | Backlinks extracted as `42`, display text shown |
| Card body with `[text](url)` | Not treated as backlink |
| Card body with `{{tasks:...}}` | Not affected |
| Card body with `[[42]]` and `[link](url)` mixed | Both work correctly |
| Edit existing card | Backlinks re-extracted correctly |
| Delete card with backlinks | Blocked correctly |
| Chat agent creates card with backlinks | Works with new syntax |
| Structured data `link_to_card` fields | Still extracted correctly (unchanged) |
| Frontend card link click | Navigates to correct card |
| Frontend backlink insertion | Inserts `[[id\|title]]` |
| Frontend preview mode | Shows card links correctly |
| Obsidian export | `[[42]]` links work natively |

### 5.2 Rollback plan

If issues are discovered after deployment:

1. The migration script can be reversed — replace `[[card_id]]` with `[card_id]` using a similar batch process
2. Keep the old `ExtractBacklinks` function available behind a feature flag for one release cycle
3. Backups of card bodies before migration (the migration script should log original bodies)

### 5.3 Feature flag (optional but recommended)

Add a `use_wiki_links` boolean in the user settings or server config. During transition:
- If `true`: use `[[wiki-link]]` extraction
- If `false`: use old `[card_id]` extraction
- Remove the flag after full migration

---

## Summary

| Phase | Description | Time | Risk |
|-------|-------------|------|------|
| 0 | Deduplicate backlink logic | 1h | Low |
| 1 | Backend extraction update | 2-3h | Low |
| 2 | Frontend rendering + input | 3-4h | Medium |
| 3 | Data migration script | 3-4h | **High** |
| 4 | Cleanup | 1h | Low |
| 5 | Verification | 2h | Low |
| | **Total** | **~12-15h** | |

### Recommended execution order

1. **Phase 0** in one PR — pure refactor, no behavior change
2. **Phase 1 + 2** in one PR — both old and new syntax work simultaneously (the new `ExtractBacklinks` handles `[[...]]`, old `[...]` in existing bodies just stop being treated as links temporarily)
3. **Phase 3** as a separate PR with the migration script — run as a one-shot command
4. **Phase 4 + 5** in a cleanup PR after migration is verified

This ordering means the system is never broken — Phase 1+2 just changes what syntax is *generated*, and Phase 3 converts existing data to match.
