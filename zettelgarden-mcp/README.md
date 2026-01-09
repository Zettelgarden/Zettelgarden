# Zettelgarden MCP Server

Connect Claude to your Zettelgarden data via the Model Context Protocol (MCP).

## Setup

### 1. Install dependencies

```bash
cd zettelgarden-mcp
pip install -r requirements.txt
```

Or with uv:
```bash
uv pip install -r requirements.txt
```

### 2. Get your auth token

1. Open Zettelgarden in your browser
2. Log in to your account
3. Open browser DevTools (F12) → Network tab
4. Make any action in the app
5. Find a request to /api/* and copy the `Authorization: Bearer <token>` header value

Or extract from localStorage:
```javascript
// In browser console
localStorage.getItem('token')
```

### 3. Configure Claude Code

Add to `~/.claude/settings.json`:

```json
{
  "mcpServers": {
    "zettelgarden": {
      "command": "python",
      "args": ["/home/nick/code/Zettelgarden/zettelgarden-mcp/server.py"],
      "env": {
        "ZETTELGARDEN_TOKEN": "your-jwt-token-here",
        "ZETTELGARDEN_API_URL": "http://localhost:8080"
      }
    }
  }
}
```

### 4. Restart Claude Code

The tools will be available after restart.

## Available Tools

### Cards
- `search_cards` - Search by text, tags (#tag), entities (@[name])
- `get_card` - Get full card by ID or card_id
- `create_card` - Create a new card
- `update_card` - Update title, body, or link
- `list_starred_cards` - Get all starred cards
- `get_card_children` - Get children of a card

### Tasks
- `list_tasks` - List with filters (completed, scheduled_date, priority)
- `get_task` - Get task details
- `create_task` - Create a new task
- `update_task` - Update task fields
- `complete_task` - Mark task as complete

## Example Usage

Once configured, you can ask Claude:
- "Search my cards for python"
- "What are my tasks for today?"
- "Create a card about MCP integration"
- "Show me card 1a"
- "Mark task 42 as complete"
