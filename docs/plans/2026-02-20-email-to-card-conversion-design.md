# Email to Card Conversion - Design

## Overview

Add ability to convert emails to cards, mirroring the existing RSS article conversion feature. Users can convert interesting emails to knowledge cards with optional editing, tagging, and card linking.

## Architecture

### Data Flow

1. User clicks "Convert to Card" button in `EmailDetailPage` header
2. `EmailConvertDialog` opens with pre-filled data (subject as title, HTML→markdown content)
3. User edits title/content, adds optional tags, optionally links to existing card
4. On submit: POST to `/api/emails/{id}/convert` with conversion params
5. Backend creates/updates card and creates `email_card_links` record
6. Frontend updates email state with `card_id`, shows green icon

### Components

- **New**: `EmailConvertDialog.tsx` - conversion dialog (similar to `RssConvertDialog.tsx`)
- **Modify**: `EmailDetailPage.tsx` - add button, state management, dialog integration
- **New**: `convertEmailToCard()` in `email.ts` API client
- **New**: Backend handler and route for email conversion
- **Modify**: `Email` interface to include `card_id?: number`

## Backend Changes

### Route (`routes/email.go`)

```go
addProtectedRoute(r, h, "/api/emails/{id}/convert", h.ConvertEmailToCardRoute, "POST")
```

### Handler (`handlers/email_sync.go`)

New function `ConvertEmailToCardRoute`:
- Accepts params: `title`, `body`, `tags`, `card_id` (same structure as RSS)
- Creates new card or updates existing card if `card_id` provided
- Creates `email_card_links` record associating email with card
- Returns `{"id": cardId}`

### Model (`models/email_sync.go`)

Add `CardID *int` field to `Email` struct for frontend conversion status tracking.

### Database

The `email_card_links` table already exists from schema 0115. No schema changes needed.

## Frontend Changes

### New Component: `EmailConvertDialog.tsx`

Props:
- `isOpen: boolean`
- `onClose: () => void`
- `email: Email | null`
- `onConverted: (cardId: number) => void`

Behavior:
- Pre-fills title with `email.subject`
- Converts `email.body_html` to markdown using `safeHtmlToMarkdown`
- Accepts optional tags and card_id for linking
- Nearly identical to `RssConvertDialog.tsx`

### API: `email.ts`

```typescript
export interface ConvertEmailParams {
  title?: string;
  body?: string;
  tags?: string;
  card_id?: string;
}

export function convertEmailToCard(id: number, params?: ConvertEmailParams): Promise<ConvertCardResponse> {
  return getData(apiClient.post<ConvertCardResponse>(`/emails/${id}/convert`, params));
}
```

### Modify `Email` Interface

```typescript
export interface Email {
  // ... existing fields
  card_id?: number;  // Add conversion status tracking
}
```

### Modify `EmailDetailPage.tsx`

State additions:
- `showConvertDialog: boolean`
- Handle conversion success to update local `email` state with `card_id`

UI additions:
- "Convert to Card" button in header (between Archive and Create Task)
- Green card icon (same as RSS) when `email.card_id` exists

## Error Handling

- Backend: Validate required fields, return 400/404 appropriately
- Frontend: Show error messages in dialog, handle network failures
- Maintain existing error patterns from RSS conversion

## Testing Considerations

- Backend: Test conversion with new/existing card, tags, linking
- Frontend: Test dialog open/close, form submission, success/error states
- Integration: Verify email_card_links record creation
- UI: Verify green icon appears after conversion
