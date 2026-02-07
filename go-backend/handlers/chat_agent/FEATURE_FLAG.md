# Chat Agent V2 Feature Flag

## Overview

The `ChatAgentV2` feature flag enables routing between the original `chat_agent.go` code and the refactored `handlers/chat_agent/` package for safe testing.

## How to Enable

### Via Environment Variable
```bash
export ZETTELGARDEN_FEATURE_CHAT_AGENT_V2=true
cd go-backend && go run main.go
```

### Via .env-bash
Add to your `go-backend/.env-bash`:
```bash
export ZETTELGARDEN_FEATURE_CHAT_AGENT_V2=true
```

## How It Works

When `ChatAgentV2` is enabled:
1. `StreamMessageRoute` routes to `ChatService.StreamAssistantResponse`
2. Uses the refactored code in `handlers/chat_agent/`
3. Logs `[ChatAgentV2]` prefix for easy identification

When disabled (default):
1. Uses original `Handler.streamAssistantResponse`
2. Original proven code path
3. No changes to existing behavior

## Testing Checklist

When testing with the flag enabled:
- [ ] Start a new chat conversation
- [ ] Send messages that trigger tool calls
- [ ] Verify streaming responses work
- [ ] Verify tool execution completes
- [ ] Check for `[ChatAgentV2]` log entries
- [ ] Monitor for any errors

## Rollback

If issues occur, simply disable the flag:
```bash
# Unset the variable
unset ZETTELGARDEN_FEATURE_CHAT_AGENT_V2

# Or set to false
export ZETTELGARDEN_FEATURE_CHAT_AGENT_V2=false
```

Then restart the backend.

## Files Modified

- `services/featureflags/flags.go` - Added `FeatureFlagChatAgentV2` constant
- `handlers/chat_messages.go` - Added routing logic
- `handlers/chat_agent/streaming.go` - Exported `StreamAssistantResponse` method

## Current Status

- ✅ Feature flag defined
- ✅ Routing implemented
- ✅ Build successful
- ⏳ Ready for testing "in anger"
