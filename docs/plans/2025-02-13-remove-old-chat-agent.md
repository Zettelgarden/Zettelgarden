# Remove Old Chat Agent Implementation Plan

> **For Claude:** REQUIRED SUB-SKILL: Use superpowers:executing-plans to implement this plan task-by-task.

**Goal:** Remove deprecated v1 chat agent implementation and feature flag infrastructure, keeping only the v2 ChatService.

**Architecture:** Direct deletion of old code path and removal of feature flag routing. The v2 ChatService is already default and tested, so this is pure cleanup.

**Tech Stack:** Go backend, feature flag system, prompt loader

---

### Task 1: Verify Feature Flag Usage

**Files:**
- Read: `go-backend/services/featureflags/flags.go`
- Grep: `featureflags.IsEnabled`

**Step 1: Check for other feature flags**

Read the feature flags file to see all defined flags:
```bash
cat go-backend/services/featureflags/flags.go
```

Look for all `FeatureFlag*` constants to determine if `ChatAgentV2` is the only flag.

**Step 2: Search for feature flag usage**

Search for all usages of the feature flag system:
```bash
cd go-backend && grep -r "featureflags.IsEnabled" --include="*.go" .
```

**Step 3: Document findings**

If other flags exist, we only remove `ChatAgentV2`. If it's the only flag, we can remove the entire system.

---

### Task 2: Remove Old Chat Agent Implementation

**Files:**
- Delete: `go-backend/handlers/chat_agent.go`

**Step 1: Verify file is deprecated**

Check that the file contains the deprecated old implementation:
```bash
head -20 go-backend/handlers/chat_agent.go
```

Should see `streamAssistantResponse` method and deprecation notice.

**Step 2: Delete the file**

```bash
rm go-backend/handlers/chat_agent.go
```

**Step 3: Verify no remaining references**

Search for any remaining references to the old method:
```bash
cd go-backend && grep -r "streamAssistantResponse" --include="*.go" .
```

Should only find references in `chat_messages.go` that we'll update next.

---

### Task 3: Update Chat Messages Routing

**Files:**
- Modify: `go-backend/handlers/chat_messages.go:219-244`

**Step 1: Read the current routing logic**

```bash
sed -n '219,244p' go-backend/handlers/chat_messages.go
```

This shows the feature flag routing that needs to be removed.

**Step 2: Replace with direct ChatService call**

Find this block:
```go
// Stream the response - route based on feature flag
if featureflags.IsEnabled(featureflags.FeatureFlagChatAgentV2) {
    if s.ChatService == nil {
        log.Printf("[ChatAgentV2] ERROR: ChatService is nil! Falling back to original code.")
        s.streamAssistantResponse(r.Context(), w, userID, conversation, assistantMessageID, req.Model)
        return
    }
    log.Printf("[ChatAgentV2] Using new ChatService for streaming")
    err := s.ChatService.StreamAssistantResponse(r.Context(), w, userID, conversationID, assistantMessageID, req)
    // ... error handling
} else {
    s.streamAssistantResponse(r.Context(), w, userID, conversation, assistantMessageID, req.Model)
}
```

Replace with:
```go
// Stream the response using ChatService
log.Printf("[ChatAgent] Using ChatService for streaming")
err := s.ChatService.StreamAssistantResponse(r.Context(), w, userID, conversationID, assistantMessageID, req)
if err != nil {
    log.Printf("[ChatAgent] Error streaming response: %v", err)
    // Error handling continues below...
}
```

**Step 3: Remove unused imports**

If `featureflags` package was only imported for this check, remove the import:
```bash
# Check if featureflags is used elsewhere in the file
grep -n "featureflags\." go-backend/handlers/chat_messages.go
```

If no other uses, remove the import from the import block.

**Step 4: Commit**

```bash
cd go-backend
git add handlers/chat_messages.go handlers/chat_agent.go
git commit -m "refactor: remove old chat agent and feature flag routing

The v2 ChatService is now the only implementation.
Remove deprecated chat_agent.go and feature flag routing.

Co-Authored-By: Claude Opus 4.6 <noreply@anthropic.com>"
```

---

### Task 4: Update Prompt Loader

**Files:**
- Modify: `go-backend/prompts/loader.go`
- Delete: `go-backend/prompts/zettelgarden_assistant.md`
- Rename: `go-backend/prompts/zettelgarden_assistant_v2.md` -> `go-backend/prompts/zettelgarden_assistant.md`

**Step 1: Read current loader**

```bash
cat go-backend/prompts/loader.go
```

**Step 2: Remove old prompt file**

```bash
rm go-backend/prompts/zettelgarden_assistant.md
```

**Step 3: Rename v2 to be the default**

```bash
mv go-backend/prompts/zettelgarden_assistant_v2.md go-backend/prompts/zettelgarden_assistant.md
```

**Step 4: Update loader (if needed)**

The loader currently loads `zettelgarden_assistant.md`. After renaming, it should work without changes:
```go
func GetZettelgardenAssistantPrompt() (string, error) {
    return LoadPrompt("zettelgarden_assistant.md")
}
```

This should already be correct after the rename.

**Step 5: Verify prompt content**

```bash
cat go-backend/prompts/zettelgarden_assistant.md | head -20
```

Should see the v2 prompt content (with "card_ids" plural, etc).

**Step 6: Commit**

```bash
cd go-backend
git add prompts/
git commit -m "refactor: consolidate to single chat agent prompt

Remove old prompt file and promote v2 prompt to default.

Co-Authored-By: Claude Opus 4.6 <noreply@anthropic.com>"
```

---

### Task 5: Add ChatService Startup Check

**Files:**
- Modify: `go-backend/main.go` (around line 118)

**Step 1: Read current ChatService initialization**

```bash
sed -n '115,125p' go-backend/main.go
```

**Step 2: Add nil check after initialization**

After:
```go
h.ChatService = chat_agent.NewChatService(s.DB, s)
```

Add:
```go
h.ChatService = chat_agent.NewChatService(s.DB, s)
if h.ChatService == nil {
    log.Fatal("ERROR: Failed to initialize ChatService")
}
log.Printf("ChatService initialized successfully")
```

**Step 3: Commit**

```bash
cd go-backend
git add main.go
git commit -m "feat: add ChatService startup check

Fail fast if ChatService initialization fails.
Previously there was fallback logic; after removing old code,
we need explicit check.

Co-Authored-By: Claude Opus 4.6 <noreply@anthropic.com>"
```

---

### Task 6: Remove Feature Flag Infrastructure (Conditional)

**Only execute if Task 1 determined ChatAgentV2 was the only flag.**

**Files:**
- Delete: `go-backend/services/featureflags/flags.go`
- Delete: `go-backend/services/featureflags/service.go` (if exists)
- Modify: `go-backend/main.go` (remove feature flag init)

**Step 1: Remove feature flag files**

```bash
rm -rf go-backend/services/featureflags/
```

**Step 2: Remove feature flag initialization from main.go**

Search for feature flag init:
```bash
grep -n "featureflags" go-backend/main.go
```

Remove any imports and initialization code like:
```go
featureflags.InitFromEnv()
```

**Step 3: Commit**

```bash
cd go-backend
git add services/ main.go
git commit -m "refactor: remove unused feature flag system

ChatAgentV2 was the only remaining flag. After removing
old chat agent, the feature flag system is no longer needed.

Co-Authored-By: Claude Opus 4.6 <noreply@anthropic.com>"
```

---

### Task 7: Run Tests

**Step 1: Run all Go tests**

```bash
cd go-backend
go test ./... -v
```

Expected: All tests pass

**Step 2: Build verification**

```bash
cd go-backend
go build -o main
```

Expected: Clean build with no errors

**Step 3: Check for compilation errors**

Look specifically for:
- Missing imports (featureflags)
- Undefined functions (streamAssistantResponse)
- Nil pointer issues

---

### Task 8: Verification

**Step 1: Start the server**

```bash
cd go-backend
go run main.go
```

**Step 2: Check logs for ChatService initialization**

Should see:
```
Initializing ChatService
ChatService initialized successfully
```

**Step 3: Test chat endpoint**

Send a test chat message via the API or frontend. Verify:
- Chat response streams correctly
- Logs show `[ChatAgent] Using ChatService for streaming`
- No fallback or error messages about old code

---

## Summary

After completion:
- Old `chat_agent.go` removed
- Feature flag routing removed from `chat_messages.go`
- Single prompt file (consolidated v2)
- Feature flag infrastructure removed (if no other flags)
- Startup check ensures ChatService is always initialized
- All tests pass
- Chat functionality works identically
