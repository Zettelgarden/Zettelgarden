# Telegram Bot

The Zettelgarden Telegram bot allows you to chat with your knowledge base via Telegram.

## Setup

### 1. Create a Telegram Bot

1. Open Telegram and search for `@BotFather`
2. Send `/newbot` and follow the prompts
3. Copy the bot token (looks like `123456789:ABCdefGhIJKlmNoPQRsTUVwxyZ`)

### 2. Get Your Telegram User ID

1. Open Telegram and search for `@userinfobot`
2. Send any message
3. Copy your numeric user ID

### 3. Configure Environment Variables

Add to your environment:

```bash
TELEGRAM_BOT_TOKEN=your_bot_token_here
TELEGRAM_ALLOWED_USER_ID=your_telegram_user_id
TELEGRAM_ZETTELGARDEN_USER_ID=1  # Your Zettelgarden user ID
TELEGRAM_ENABLED=true
```

### 4. Start the Bot

The bot will start automatically when you run the backend with `TELEGRAM_ENABLED=true`.

## Usage

- Send any message to chat with your knowledge base
- `/help` - Show available commands
- `/clear` - Start a new conversation

## Features

- Full access to AI agent tools (search cards, create cards, etc.)
- Conversation history preserved in web UI
- One conversation per user (new on `/clear`)

## Security

- Only the configured `TELEGRAM_ALLOWED_USER_ID` can use the bot
- All unauthorized access attempts are logged
