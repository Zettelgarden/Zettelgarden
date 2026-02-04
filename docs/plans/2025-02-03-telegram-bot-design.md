# Telegram Bot for Zettelgarden Chat

**Date:** 2025-02-03
**Status:** Design Approved

## Overview

Add a Telegram bot that integrates with Zettelgarden's existing chat/agent functionality. The bot allows a single authorized user (initially) to chat with their knowledge base via Telegram, with access to all existing agent tools (search cards, create cards, etc.).

## Architecture

### Service Integration

The Telegram bot will be integrated into `go-backend` as a new service module:

- **New service module**: `go-backend/telegram/` - Contains the Telegram bot logic
- **Handler**: `telegram/bot.go` - Main bot handler that receives messages and orchestrates responses
- **Configuration**: Environment variables for bot token and allowed user ID
- **Integration**: Reuses existing chat models, handlers, and agent tools

The bot will run as a goroutine in the main application, similar to how scheduled jobs are handled.

### Data Flow

**Message Flow:**
1. User sends message to Telegram bot
2. Bot receives update via long polling
3. Bot verifies sender ID matches `TELEGRAM_ALLOWED_USER_ID` env var
4. Bot retrieves or creates the user's dedicated Telegram conversation (stored in `chat_conversations` table)
5. Message is passed to `SaveChatMessage` and `processAssistantResponse` (existing chat logic)
6. Agent processes with tools (search_cards, create_card, etc.)
7. Response is sent back via Telegram API

**Conversation State:**
- One active conversation per Zettelgarden user for Telegram
- `/clear` creates a NEW conversation ID (archives the old one naturally)
- Bot always looks up the *latest* Telegram conversation for the user
- Old conversations remain in the database and can be viewed in the web UI
- Each conversation gets an auto-generated title like "Telegram Chat 2025-02-03"

**Special Commands:**
- `/clear` - Clear conversation context (creates new conversation ID)
- `/help` - Show available commands

## Environment Configuration

**New Environment Variables:**
```bash
# Telegram Bot Configuration
TELEGRAM_BOT_TOKEN=your_bot_token_from_botfather
TELEGRAM_ALLOWED_USER_ID=123456789  # Your Telegram numeric user ID
TELEGRAM_ENABLED=true               # Optional: enable/disable bot
```

**Getting your Telegram user ID:**
1. Message `@userinfobot` on Telegram
2. It replies with your numeric ID

**Getting a bot token:**
1. Message `@BotFather` on Telegram
2. `/newbot` and follow prompts
3. BotFather gives you the token

## Implementation Structure

**New files to create:**
```
go-backend/telegram/
  ├── bot.go           # Main bot handler with message processing
  ├── client.go        # Telegram API client wrapper
  └── commands.go      # Command handlers (/clear, /help)
```

**Modifications to existing files:**
```
go-backend/
  ├── main.go                  # Start bot goroutine if enabled
  ├── models/chat.go           # (no changes needed - reuses existing)
  └── handlers/
      ├── chat_messages.go     # (reuses existing - no changes)
      └── chat_db.go           # May need helper for "get latest Telegram convo"
```

**Key dependencies:**
- `github.com/go-telegram-bot-api/telegram-bot-api/v5` - Popular Go Telegram library

**Database:**
- No new tables needed - reuses `chat_conversations` and `chat_messages`
- Optional: Add `telegram_user_id` column to `users` table for future multi-user support

## Error Handling & Edge Cases

**Security:**
- Reject any message from `TELEGRAM_ALLOWED_USER_ID` mismatch
- Log all rejected access attempts

**API Failures:**
- Telegram API timeout: Retry with exponential backoff
- Message too long (>4096 chars): Split into multiple messages
- Markdown formatting errors: Fall back to plain text

**Bot Rate Limits:**
- Telegram limits: 30 messages/sec to same chat
- Implement small delay between chunks if sending long responses

**Conversation Edge Cases:**
- Bot starts with no existing conversation: Create new one automatically
- User deletes web UI conversation while bot is active: Bot creates new one on next message
- Database connection loss: Log error, respond with "Service temporarily unavailable"

**Testing Edge Cases:**
- Empty messages: Ignore silently
- Special characters in markdown: Handle/escape properly

## Testing & Deployment

**Development Testing:**
1. Create test bot via BotFather
2. Use your own Telegram ID for `TELEGRAM_ALLOWED_USER_ID`
3. Test locally with `TELEGRAM_ENABLED=true`
4. Verify: basic chat, /clear command, tool calls (search, create card, etc.)

**Docker Deployment:**
```yaml
# docker-compose.yml - add service
telegram-bot:
  build: ./go-backend
  environment:
    - TELEGRAM_ENABLED=true
    - TELEGRAM_BOT_TOKEN=${TELEGRAM_BOT_TOKEN}
    - TELEGRAM_ALLOWED_USER_ID=${TELEGRAM_ALLOWED_USER_ID}
  # Uses same DB/redis as main backend
```

**Monitoring:**
- Log all bot messages for debugging
- Track conversation IDs per user
- Optional: Add bot status to admin UI

## Future Enhancements (Out of Scope)

- Multi-user support with account linking
- Webhook mode instead of long polling
- Inline keyboard for card selection
- Task notifications
