# Rolling Chat UI Implementation Plan

> **For Claude:** REQUIRED SUB-SKILL: Use superpowers:executing-plans to implement this plan task-by-task.

**Goal:** Refactor ChatPage from multi-conversation management to a single rolling chat session with minimal UI

**Architecture:** Remove conversation sidebar, simplify state to single persistent session, add clear/restore functionality with one-deep archive stack

**Tech Stack:** React, TypeScript, Vite, existing useChat hook

---

## Task 1: Create ChatUtilityBar Component

**Files:**
- Create: `zettelkasten-front/src/components/chat/ChatUtilityBar.tsx`

**Step 1: Create the utility bar component**

```tsx
import React from "react";
import { Link } from "react-router-dom";

interface ChatUtilityBarProps {
  hasLastCleared: boolean;
  isSending: boolean;
  onClear: () => void;
  onRestoreLast: () => void;
  onInstructions: () => void;
  hasSubscription?: boolean;
}

export function ChatUtilityBar({
  hasLastCleared,
  isSending,
  onClear,
  onRestoreLast,
  onInstructions,
  hasSubscription = true,
}: ChatUtilityBarProps) {
  return (
    <div className="bg-white border-b border-gray-200 px-4 py-2">
      <div className="flex items-center justify-between max-w-6xl mx-auto">
        <div className="flex items-center gap-2">
          <h1 className="text-lg font-semibold text-gray-900">Chat</h1>
          {!hasSubscription && (
            <span className="text-xs bg-yellow-100 text-yellow-700 px-2 py-0.5 rounded-full font-medium">
              PRO
            </span>
          )}
        </div>
        <div className="flex items-center gap-2">
          <button
            onClick={onInstructions}
            className="text-gray-600 hover:text-gray-900 px-3 py-1.5 text-sm font-medium rounded-lg hover:bg-gray-100 transition-colors flex items-center gap-1.5"
          >
            <svg className="w-4 h-4" fill="none" stroke="currentColor" viewBox="0 0 24 24">
              <path strokeLinecap="round" strokeLinejoin="round" strokeWidth={2} d="M12 6.253v13m0-13C10.832 5.477 9.246 5 7.5 5S4.168 5.477 3 6.253v13C4.168 18.477 5.754 18 7.5 18s3.332.477 4.5 1.253m0-13C13.168 5.477 14.754 5 16.5 5c1.746 0 3.332.477 4.5 1.253v13C19.832 18.477 18.246 18 16.5 18c-1.746 0-3.332.477-4.5 1.253" />
            </svg>
            Instructions
          </button>
          <button
            onClick={onClear}
            disabled={isSending}
            className="text-gray-600 hover:text-gray-900 px-3 py-1.5 text-sm font-medium rounded-lg hover:bg-gray-100 transition-colors disabled:opacity-50 disabled:cursor-not-allowed"
          >
            Clear
          </button>
          {hasLastCleared && (
            <button
              onClick={onRestoreLast}
              className="text-blue-600 hover:text-blue-700 px-3 py-1.5 text-sm font-medium rounded-lg hover:bg-blue-50 transition-colors"
            >
              Restore Last
            </button>
          )}
        </div>
      </div>
    </div>
  );
}
```

**Step 2: Export the component**

Ensure the component has proper export for use in ChatPage.

**Step 3: Commit**

```bash
git add zettelkasten-front/src/components/chat/ChatUtilityBar.tsx
git commit -m "feat: add ChatUtilityBar component"
```

---

## Task 2: Add clear/restore methods to useChat hook

**Files:**
- Modify: `zettelkasten-front/src/hooks/useChat.tsx`
- Test: No existing tests for useChat - manual verification

**Step 1: Add LastClearedSession interface and state**

Add to imports and interfaces:

```tsx
// Add to interfaces
export interface LastClearedSession {
  conversationId: string;
  messages: ChatMessage[];
  timestamp: string;
}

// Add to useChat function - new state
const [lastClearedSession, setLastClearedSession] = useState<LastClearedSession | null>(null);
```

**Step 2: Add clearChat method**

```tsx
const clearChat = async () => {
  if (!currentConversation || isSending) return;

  // Archive current session if it has messages
  if (messages.length > 0 && !isDraftConversation) {
    const archived: LastClearedSession = {
      conversationId: currentConversation.id,
      messages: [...messages],
      timestamp: new Date().toISOString(),
    };
    setLastClearedSession(archived);
  }

  // Create new conversation
  await createNewConversation("", selectedModel);
  setMessages([]);
  setError(null);
};

const restoreLastCleared = async () => {
  if (!lastClearedSession) return;

  try {
    await loadConversation(lastClearedSession.conversationId);
    setLastClearedSession(null);
  } catch (error) {
    console.error("Failed to restore last cleared session:", error);
    setError("Failed to restore previous session");
    setLastClearedSession(null); // Clear invalid archive
  }
};
```

**Step 3: Add to return statement**

Add to the return object:

```tsx
return {
  // ... existing returns
  lastClearedSession,
  clearChat,
  restoreLastCleared,
};
```

**Step 4: Commit**

```bash
git add zettelkasten-front/src/hooks/useChat.tsx
git commit -m "feat: add clearChat and restoreLastCleared to useChat hook"
```

---

## Task 3: Update ChatInterface props - remove sidebar related props

**Files:**
- Modify: `zettelkasten-front/src/components/chat/ChatInterface.tsx`

**Step 1: Remove unused prop from interface**

The `compact` and `availableModels` props are already not being used meaningfully. Keep interface simple - only pass what's actually used.

The current props are already minimal - no changes needed to ChatInterface itself. It will continue to work with existing props.

**Step 4: Commit**

```bash
# No changes needed - ChatInterface is already properly structured
```

---

## Task 4: Extend ChatInput to handle /clear command

**Files:**
- Modify: `zettelkasten-front/src/components/chat/ChatInput.tsx`
- Modify: `zettelkasten-front/src/hooks/useChat.tsx`

**Step 1: Read ChatInput to understand current structure**

```bash
# Read the file first to understand how it works
cat zettelkasten-front/src/components/chat/ChatInput.tsx
```

**Step 2: Add onClearCommand callback to ChatInput props**

```tsx
interface ChatInputProps {
  // ... existing props
  onClearCommand?: () => void;
}
```

**Step 3: Detect /clear in input and emit callback**

Add before the submit logic:

```tsx
const handleSubmit = () => {
  // Check for /clear command
  if (value.trim() === '/clear' && onClearCommand) {
    onClearCommand();
    onChange('');
    return;
  }

  // Existing submit logic
  if (!value.trim() || disabled) return;
  onSubmit();
};
```

**Step 4: Update ChatInterface to pass onClearCommand**

```tsx
// In ChatInterface, add to the destructured chatHook
const { clearChat } = chatHook;

// Pass to ChatInput
<ChatInput
  // ... existing props
  onClearCommand={clearChat}
/>
```

**Step 5: Commit**

```bash
git add zettelkasten-front/src/components/chat/ChatInput.tsx
git add zettelkasten-front/src/components/chat/ChatInterface.tsx
git commit -m "feat: add /clear command support to ChatInput"
```

---

## Task 5: Refactor ChatPage - remove conversation state and sidebar

**Files:**
- Modify: `zettelkasten-front/src/pages/ChatPage.tsx`

**Step 1: Remove unused imports**

Remove these imports:
```tsx
import { getConversations } from "../api/chat"; // Remove
import { ConversationSidebar } from "../components/chat/ConversationSidebar"; // Remove
```

**Step 2: Remove conversation-related state**

Delete these state variables:
```tsx
// Delete all of these
const [conversations, setConversations] = useState<ChatConversation[]>([]);
const [sidebarOpen, setSidebarOpen] = useState(false);
const [showAllRecent, setShowAllRecent] = useState(false);
const [searchQuery, setSearchQuery] = useState("");
const [showStarredOnly, setShowStarredOnly] = useState(false);
const [loadingConversations, setLoadingConversations] = useState(false);
const [loadingConversationIds, setLoadingConversationIds] = useState<Set<string>>(new Set());
const [deletingConversationIds, setDeletingConversationIds] = useState<Set<string>>(new Set());
const [starringConversationIds, setStarringConversationIds] = useState<Set<string>>(new Set());
```

**Step 3: Remove unused refs and effects**

Delete:
```tsx
const lastSyncedConversationIdRef = React.useRef<string | null>(null);
const prevConversationIdRef = React.useRef<string | null>(null);
```

Remove the `loadConversations`, `loadConversation`, `deleteConversation`, `starConversation`, `createNewConversationWithMessage` functions - they will no longer be needed.

**Step 4: Simplify useEffect hooks**

Replace the initialization effect:

```tsx
useEffect(() => {
  setDocumentTitle("Chat");

  // Load or create the single rolling session
  initializeChatSession();

  handleUrlParams();
}, []);

const initializeChatSession = async () => {
  // Try to load the most recent conversation, or create new
  try {
    const convs = await getConversations();
    if (convs && convs.length > 0) {
      await chatHook.loadConversation(convs[0].id);
    } else {
      await chatHook.createNewConversation("", chatHook.selectedModel);
    }
  } catch (error) {
    console.error("Failed to initialize chat:", error);
    await chatHook.createNewConversation("", chatHook.selectedModel);
  }
};
```

**Step 5: Remove handlers we don't need**

Delete these functions:
- `loadConversation`
- `createNewConversation`
- `deleteConversation`
- `starConversation`
- `createNewConversationWithMessage`

**Step 6: Update the return JSX**

Replace the entire return with:

```tsx
return (
  <div className="flex flex-col h-screen bg-white">
    {/* Utility Bar */}
    <ChatUtilityBar
      hasLastCleared={!!chatHook.lastClearedSession}
      isSending={chatHook.isSending}
      onClear={chatHook.clearChat}
      onRestoreLast={chatHook.restoreLastCleared}
      onInstructions={() => setShowInstructionsMenu(true)}
      hasSubscription={hasSubscription}
    />

    {/* Chat Area - full width */}
    <div className="flex-1 flex flex-col overflow-hidden">
      {chatHook.currentConversation ? (
        <ChatInterface
          chatHook={chatHook}
          onCardClick={handleCardClick}
          onTaskClick={handleTaskClick}
          onRegenerateMessage={handleRegenerateMessage}
          placeholder="Ask about your cards... Type @ to mention a card, /clear to clear chat"
          compact={false}
          showModelDropdown={true}
        />
      ) : (
        <div className="flex-1 flex items-center justify-center bg-white">
          <div className="text-center text-gray-500 max-w-md mx-auto p-8">
            <div className="w-16 h-16 mx-auto mb-6 rounded-lg bg-gray-100 flex items-center justify-center">
              <svg className="w-8 h-8 text-gray-600" fill="none" stroke="currentColor" viewBox="0 0 24 24">
                <path strokeLinecap="round" strokeLinejoin="round" strokeWidth={2} d="M8 12h.01M12 12h.01M16 12h.01M21 12c0 4.418-4.03 8-9 8a9.863 9.863 0 01-4.255-.949L3 20l1.395-3.72C3.512 15.042 3 13.574 3 12c0-4.418 4.03-8 9-8s9 3.582 9 8z" />
              </svg>
            </div>
            <h3 className="text-xl font-semibold text-gray-900 mb-3">Welcome to Chat</h3>
            <p className="text-gray-600 mb-6 leading-relaxed">Start typing to chat with your knowledge base.</p>
            {!hasSubscription && (
              <div className="text-center text-gray-500 mb-6 p-4 bg-gray-50 rounded-lg">
                AI Agents are a Pro feature.
                <br />
                <Link to="/app/subscribe" className="text-blue-500 hover:underline">
                  Upgrade to Pro to unlock intelligent AI agents that can work with your knowledge base.
                </Link>
              </div>
            )}
          </div>
        </div>
      )}
    </div>

    {/* Instructions Menu */}
    <InstructionsMenu
      isOpen={showInstructionsMenu}
      onClose={() => setShowInstructionsMenu(false)}
    />

    {/* Task Dialog */}
    <TaskDialog
      taskId={selectedTaskId}
      isOpen={showTaskDialog}
      onClose={handleTaskDialogClose}
    />
  </div>
);
```

**Step 7: Update imports**

Add ChatUtilityBar import:

```tsx
import { ChatUtilityBar } from "../components/chat/ChatUtilityBar";
```

Remove getConversations import if still there.

**Step 8: Commit**

```bash
git add zettelkasten-front/src/pages/ChatPage.tsx
git commit -m "refactor: simplify ChatPage to single rolling session"
```

---

## Task 6: Remove ConversationSidebar component and imports

**Files:**
- Delete: `zettelkasten-front/src/components/chat/ConversationSidebar.tsx`
- Modify: `zettelkasten-front/src/pages/ChatPage.tsx` (already done in Task 5)

**Step 1: Delete the component file**

```bash
rm zettelkasten-front/src/components/chat/ConversationSidebar.tsx
```

**Step 2: Verify no other files import it**

```bash
grep -r "ConversationSidebar" zettelkasten-front/src/
```

Should find no results (ChatPage was already updated).

**Step 3: Commit**

```bash
git add zettelkasten-front/src/components/chat/ConversationSidebar.tsx
git commit -m "refactor: remove unused ConversationSidebar component"
```

---

## Task 7: Add model selector to user settings page

**Files:**
- Modify: `zettelkasten-front/src/pages/SettingsPage.tsx` (or similar - find correct file)
- Create: `zettelkasten-front/src/components/settings/ModelSelector.tsx`

**Step 1: Find the settings page**

```bash
find zettelkasten-front/src -name "*Settings*" -o -name "*settings*"
```

**Step 2: Read the settings page structure**

```bash
cat zettelkasten-front/src/pages/[SettingsPageFile].tsx
```

**Step 3: Create ModelSelector component**

```tsx
import React, { useState, useEffect } from "react";
import { getChatModels, ChatModel } from "../../api/chat";

interface ModelSelectorProps {
  currentModel: string;
  onModelChange: (model: string) => void;
}

export function ModelSelector({ currentModel, onModelChange }: ModelSelectorProps) {
  const [models, setModels] = useState<ChatModel[]>([]);
  const [loading, setLoading] = useState(true);

  useEffect(() => {
    const loadModels = async () => {
      try {
        const fetched = await getChatModels();
        setModels(fetched);
      } catch (error) {
        console.error("Failed to load chat models:", error);
      } finally {
        setLoading(false);
      }
    };
    loadModels();
  }, []);

  if (loading) {
    return <div className="text-sm text-gray-500">Loading models...</div>;
  }

  return (
    <div className="space-y-2">
      <label className="block text-sm font-medium text-gray-700">
        Default Chat Model
      </label>
      <select
        value={currentModel}
        onChange={(e) => onModelChange(e.target.value)}
        className="w-full px-3 py-2 border border-gray-300 rounded-lg focus:ring-2 focus:ring-blue-500 focus:border-blue-500"
      >
        {models.map((model) => (
          <option key={model.value} value={model.value}>
            {model.label}
          </option>
        ))}
      </select>
      <p className="text-xs text-gray-500">
        This model will be used for all new chat conversations.
      </p>
    </div>
  );
}
```

**Step 4: Add to settings page**

Add the ModelSelector to the settings page, connecting to localStorage:

```tsx
import { ModelSelector } from "../components/settings/ModelSelector";

// In the settings page component
const [chatModel, setChatModel] = useState(() =>
  localStorage.getItem('chatSelectedModel') || "google/gemini-2.5-flash"
);

const handleModelChange = (model: string) => {
  setChatModel(model);
  localStorage.setItem('chatSelectedModel', model);
};

// In the JSX, add the section
<section>
  <h3>Chat Settings</h3>
  <ModelSelector
    currentModel={chatModel}
    onModelChange={handleModelChange}
  />
</section>
```

**Step 5: Commit**

```bash
git add zettelkasten-front/src/components/settings/ModelSelector.tsx
git add zettelkasten-front/src/pages/[SettingsPage].tsx
git commit -m "feat: add model selector to user settings"
```

---

## Task 8: Remove model selector from ChatInterface

**Files:**
- Modify: `zettelkasten-front/src/components/chat/ChatInterface.tsx`

**Step 1: Remove model dropdown from ChatInterface**

Remove the `showModelDropdown` prop and related UI elements:

- Remove `showModelDropdown` from props interface
- Remove the model dropdown button and UI
- Remove `getChatModels` import and related state
- Remove `availableModels` state and fetching
- Remove the model indicator section below input

Keep the input simple without model selection.

**Step 2: Commit**

```bash
git add zettelkasten-front/src/components/chat/ChatInterface.tsx
git commit -m "refactor: remove model selector from ChatInterface"
```

---

## Task 9: Clean up useChat hook - remove unused multi-conversation methods

**Files:**
- Modify: `zettelkasten-front/src/hooks/useChat.tsx`

**Step 1: Review and remove unnecessary code**

The following can be simplified:
- Remove `onConversationChange` callback complexity (no longer syncing to context)
- Remove draft conversation special handling (simplified flow)
- Remove `realConversationIdRef` complexity
- Keep core: sendMessage, loadConversation, streaming, tool calls

Simplify the hook to focus on single session management.

**Step 2: Commit**

```bash
git add zettelkasten-front/src/hooks/useChat.tsx
git commit -m "refactor: simplify useChat hook for single session"
```

---

## Task 10: Final verification and testing

**Files:**
- Manual testing

**Step 1: Start the dev server**

```bash
cd zettelkasten-front
npm start
```

**Step 2: Test checklist**

1. Page loads with empty state or existing conversation
2. Send a message - works correctly
3. Click "Clear" - conversation archived, new session created
4. Click "Restore Last" - archived conversation restored
5. Type `/clear` in input - same as Clear button
6. Instructions button opens menu
7. PRO badge shows for non-subscribed users
8. @ card references still work
9. Message regeneration still works
10. Page refresh maintains session

**Step 3: Build check**

```bash
npm run build
```

Ensure no TypeScript errors.

**Step 4: Final commit**

```bash
git add .
git commit -m "docs: complete rolling chat UI refactor"
```

---

## Summary

This refactoring:
1. Creates a minimal `ChatUtilityBar` component
2. Adds clear/restore functionality to `useChat`
3. Extends `ChatInput` to handle `/clear` command
4. Removes `ConversationSidebar` and all related state
5. Simplifies `ChatPage` to single rolling session
6. Moves model selector to settings page
7. Cleans up unused code in `useChat` and `ChatInterface`

The result is a dramatically simpler chat interface that behaves like a persistent terminal session.
