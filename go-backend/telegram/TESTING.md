# Telegram Bot Testing Checklist

## Prerequisites
1. Create a Telegram bot via @BotFather
2. Get your Telegram user ID via @userinfobot
3. Set environment variables:
   - TELEGRAM_BOT_TOKEN=your_token
   - TELEGRAM_ALLOWED_USER_ID=your_id
   - TELEGRAM_ZETTELGARDEN_USER_ID=1
   - TELEGRAM_ENABLED=true

## Basic Functionality Tests
- [ ] Start the bot: `./go-backend/main`
- [ ] Send `/start` to bot - should receive help message
- [ ] Send `/help` to bot - should see commands
- [ ] Send a regular message - should receive AI response
- [ ] Check web UI - conversation should appear with messages
- [ ] Send `/clear` - should confirm new conversation created
- [ ] Test with unauthorized Telegram account - should reject

## Agent Tool Tests
- [ ] "Search for cards about [topic]"
- [ ] "Create a card about [topic]"
- [ ] "What cards do I have?"

## Edge Cases
- [ ] Empty message - should be ignored
- [ ] Very long message (>4096 chars) - should be chunked
- [ ] Unknown command - should show error with help
- [ ] Bot restart with active conversation - should resume

## Security Tests
- [ ] Unauthorized user tries to send message - should be rejected and logged
- [ ] Multiple messages in quick succession - should handle gracefully
