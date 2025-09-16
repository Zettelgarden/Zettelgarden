# Card ID System

Zettelgarden uses a flexible Card ID system that supports hierarchical organization of notes. This document describes how Card IDs work, their formats, and the parent-child relationship logic.

## Overview

Card IDs are human-readable strings that serve as unique identifiers for cards within a user's collection. They support hierarchical relationships where cards can have parent-child relationships, enabling users to organize their knowledge in a structured, tree-like manner.

## Supported Formats

The system supports two main formats to ensure backward compatibility while providing flexibility for different organizational styles.

### Legacy Format (Alternating Separators)

The original format uses alternating `/` and `.` separators to denote hierarchy levels:

```
root
root/level1
root/level1.level2
root/level1.level2/level3
root/level1.level2/level3.level4
```

**Examples:**
- `SP24` (root card)
- `SP24/P` (child of SP24)
- `SP24/P.19` (child of SP24/P)
- `1957/A.135` (child of 1957/A)
- `1957/A.135/B.2` (child of 1957/A.135/B)

**Parent Resolution:**
- `SP24/P.19` → parent is `SP24/P`
- `1957/A.135/B.2` → parent is `1957/A.135/B`

### Modern Format (Flexible Separators)

The new format supports any combination of `.`, `/`, and `-` separators with a simpler hierarchy structure:

```
[name][separator][number][separator][number]...
```

**Examples:**
- `cardA` (root card)
- `cardA.1` (child of cardA)
- `cardA.1.2` (child of cardA.1)
- `project/v2/1` (child of project/v2)
- `notes-daily-1` (child of notes-daily)
- `mixed.1/2-3` (supports mixed separators)

**Parent Resolution:**
- `cardA.1.2` → parent is `cardA.1`
- `project/v2/1` → parent is `project/v2`
- `notes-daily-1` → parent is `notes-daily`

## Parent-Child Relationship Logic

The system automatically determines parent-child relationships using the `getParentId()` function, which implements a dual-format detection strategy:

### Algorithm

1. **Try Modern Format First**: Attempt to find the last separator and remove the final segment
2. **Validation**: If the result differs from the input, the modern format worked
3. **Root Card Detection**: If no separators exist, the card is a root card (parent = self)
4. **Fallback to Legacy**: If modern format returns the same ID and separators exist, use legacy alternating logic

### Implementation Details

```go
func getParentId(cardID string) string {
    // Try new format first
    parentFromNew := getParentIdNewFormat(cardID)
    if parentFromNew != cardID {
        return parentFromNew  // New format worked
    }

    // Check if it's a root card
    if !hasAnySeparators(cardID) {
        return cardID  // Root card
    }

    // Fall back to old alternating format
    return getParentIdAlternating(cardID)
}
```

## Root Cards

Root cards are cards without any parent relationship:
- They contain no separators (`.`, `/`, `-`)
- Their parent ID is themselves
- Examples: `inbox`, `projects`, `1`, `SP24`

## Child ID Generation

When creating child cards, the system automatically generates the next available ID:

### Backend (Go)
The `getNextRootCardID()` function finds the highest numeric root card and increments it.

### Frontend (TypeScript)
The `generateChildId()` function:
1. Fetches existing children of the parent card
2. Extracts numeric suffixes from child IDs
3. Finds the highest number and increments it
4. Defaults to `.1` if no numbered children exist

**Examples:**
- Parent: `cardA`, no children → suggests `cardA.1`
- Parent: `cardA`, children: `cardA.1`, `cardA.2` → suggests `cardA.3`
- Parent: `SP24/P`, children: `SP24/P.1`, `SP24/P.19` → suggests `SP24/P.20`

## Database Schema

Card parent-child relationships are stored in the database:

```sql
-- Cards table
CREATE TABLE cards (
    id SERIAL PRIMARY KEY,
    card_id VARCHAR NOT NULL,
    user_id INTEGER NOT NULL,
    parent_id INTEGER,  -- References cards.id
    title TEXT,
    body TEXT,
    -- ... other fields
);
```

The `parent_id` field references the database ID (`cards.id`) of the parent card, not the string `card_id`.

## API Endpoints

### Get Card Children
```
GET /api/cards/{id}/children
```
Returns all direct children of the specified card.

### Get Next Root ID
```
GET /api/cards/next-root-id
```
Returns the next available numeric root card ID.

## Backward Compatibility

The dual-format system ensures that:
- Existing cards with legacy alternating format continue to work
- New cards can use the flexible modern format
- Parent-child relationships are preserved regardless of format
- Migration between formats is not required

## Best Practices

### For Users
- **Consistency**: Stick to one format within a hierarchy branch when possible
- **Meaningful Names**: Use descriptive root card names (e.g., `projects`, `research`, `daily`)
- **Hierarchy Depth**: Avoid overly deep hierarchies (3-4 levels max recommended)

### For Developers
- **Always use `getParentId()`**: Never manually parse card IDs
- **Test both formats**: Ensure new features work with both legacy and modern formats
- **Preserve relationships**: When updating card IDs, maintain parent-child links

## Migration Notes

If migrating from legacy to modern format:
1. **No database migration required**: The system supports both formats simultaneously
2. **Gradual transition**: Users can create new cards in modern format while keeping existing legacy cards
3. **API compatibility**: All endpoints work with both formats transparently

## Examples in Practice

### Academic Research Hierarchy (Legacy Format)
```
research
research/AI
research/AI.papers
research/AI.papers/transformers
research/AI.papers/transformers.attention
```

### Project Management (Modern Format)
```
projects
projects.web-app
projects.web-app.1
projects.web-app.2
projects.mobile-app
projects.mobile-app.1
```

### Daily Notes (Mixed Approach)
```
daily
daily.2024
daily.2024.01
daily.2024.01.15
daily.2024.01.16
```