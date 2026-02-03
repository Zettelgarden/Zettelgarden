# Rolling Chat UI Redesign

**Date:** 2026-02-03
**Status:** Design
**Author:** Claude + Nick

## Overview

Refactor the ChatPage from a multi-conversation model to a **single rolling chat session** - like a terminal that's always running. Dramatically simplify the UI by removing conversation management features.

## Motivation

- **Different mental model** - Terminal-style persistent session with clear/branch capabilities
- **Simplicity** - Users don't need to manage multiple conversations
- **Reduced complexity** - Remove significant state and UI surface area

## Current State

ChatPage currently manages:
- Multiple conversations with sidebar navigation
- Starring, searching, filtering conversations
- Per-conversation model selection
- Conversation creation/deletion/selection

## Proposed Design

### Mental Model

- **One persistent session** - Like a terminal that's always running
- **Always exists** - Page loads into an active chat (even if empty)
- **Never branches** - No parallel conversations
- **Can be cleared** - Archive current session, restore last cleared on demand
- **Model choice lives in settings** - Not a per-conversation decision

### UI Layout

```
┌─────────────────────────────────────────────────┐
│  [Instructions] [Clear] [Restore Last]          │  ← Minimal utility bar
├─────────────────────────────────────────────────┤
│                                                 │
│  Message stream (scrolls)                        │
│                                                 │
│  - AI response 1                                │
│  - User message 2                               │
│  - AI response 2                                │
│                                                 │
├─────────────────────────────────────────────────┤
│  [Input box - @ for cards, # for tasks]         │
└─────────────────────────────────────────────────┘
```

**Changes:**
- Remove `ConversationSidebar` entirely
- Remove chat header (title, model indicator)
- Remove hamburger menu
- Full width for chat content
- Empty state = just the input box, ready to type
- PRO upsell = subtle badge in utility bar

### State Management

**Removed:**
- `conversations` array
- `sidebarOpen`
- `showAllRecent`
- `searchQuery`
- `showStarredOnly`
- `loadingConversations`
- `loadingConversationIds`
- `deletingConversationIds`
- `starringConversationIds`
- `currentConversationId` exposed to context

**Kept/simplified:**
- `messages` - one rolling list
- `isLoading` - for streaming
- `regeneratingMessageIds` - for regeneration
- `lastClearedSession` - `{ messages, timestamp, conversationId }`

**Internal only (not user-visible):**
- `currentConversationId` - backing ID for the session
- Backend API still uses conversation model

### Interaction Patterns

**Clearing the chat:**
- Button in utility bar OR `/clear` keyboard command
- Current messages archived to `lastClearedSession`
- New conversation ID created for fresh session
- Input box focuses, ready for new input

**Restoring last cleared:**
- "Restore last" button appears only when `lastClearedSession` exists
- One-deep stack - can only restore the last cleared session
- Swaps current messages with archived ones

**Card/Task linking:**
- All existing `@card` and `#task` functionality unchanged

**Message regeneration:**
- Keep existing functionality
- Works the same in rolling context

**URL params:**
- Keep `?message=` for pre-filling from other parts of app
- Remove `?new=` - always loads rolling session

### Backend Considerations

**No immediate changes needed:**
- Keep using existing conversation API
- One active conversation ID at a time
- Create new conversation on clear

**Data:**
- Existing conversations remain in database but become inaccessible via UI
- Low prod data impact - acceptable loss of access

## Implementation Notes

### Components to Remove/Modify

- **Delete:** `ConversationSidebar` component
- **Simplify:** `ChatPage` state tree
- **Simplify:** `useChat` hook (remove multi-conversation methods)
- **Modify:** `ChatInterface` (keep mostly same, remove sidebar props)
- **Add:** Minimal utility bar component

### Utility Bar Placement

Top-right corner (collapsed when not needed).

### Keyboard Shortcuts

- `/clear` - Clear current session and archive

## Migration Strategy

Single-pass deletion refactor:
1. Remove conversation sidebar and related state
2. Strip multi-conversation logic from ChatPage
3. Simplify useChat hook
4. Add utility bar with clear/restore/instructions
5. Update layout (remove header, expand chat width)

## Open Questions

None - design approved.

## Future Considerations

- If needed, add hidden "archive view" for old conversations
- Export tool for legacy conversations if users request it
