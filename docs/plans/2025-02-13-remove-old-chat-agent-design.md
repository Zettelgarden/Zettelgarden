# Remove Old Chat Agent Design

**Date**: 2025-02-13
**Status**: Approved

## Overview

Remove the deprecated v1 chat agent implementation and associated feature flag infrastructure, keeping only the v2 ChatService which is now the default and has been validated.

## Problem

The codebase currently contains two chat agent implementations:
- **Old (v1)**: `handlers/chat_agent.go` - deprecated, marked for removal
- **New (v2)**: `handlers/chat_agent/` package - refactored, now default

A feature flag (`FeatureFlagChatAgentV2`) controls routing between them. Since v2 is tested and enabled by default, the old code and flag infrastructure are dead code that should be removed.

## Solution

### Files to Delete

1. `go-backend/handlers/chat_agent.go` - Old chat agent implementation
2. `go-backend/prompts/zettelgarden_assistant.md` - Old prompt file

### Files to Modify

1. **`go-backend/handlers/chat_messages.go`**
   - Remove feature flag routing logic (~lines 219-244)
   - Replace with direct `ChatService.StreamAssistantResponse()` call
   - Remove fallback to old `streamAssistantResponse` method

2. **`go-backend/prompts/loader.go`**
   - Update `GetZettelgardenAssistantPrompt()` to load v2 prompt
   - Either load `zettelgarden_assistant_v2.md` or rename the file first

3. **`go-backend/main.go`**
   - Remove feature flag initialization if no other flags exist
   - Add startup check to verify ChatService is initialized

4. **`go-backend/services/featureflags/`**
   - Evaluate if `FeatureFlagChatAgentV2` is the only flag
   - If yes, remove entire feature flag system
   - Otherwise, remove only the ChatAgentV2 flag

### Safety Considerations

- Current routing has fallback if `ChatService` is nil - after removal, add startup check to fail fast
- V2 already has better error handling in `chat_agent/errors.go`
- No rollback needed since v2 is already the default

## Testing

Since v2 is already default and tested:
1. Run existing unit tests - ensure no tests explicitly test old code path
2. Post-deployment: verify chat works end-to-end
3. Check logs confirm ChatService is used (no fallback messages)

## Success Criteria

- Chat functionality works identically to current behavior
- Codebase is simpler with no dead code
- All tests pass
