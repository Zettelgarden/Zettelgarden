# Chat API Testing Setup

## 🚀 Quick Start

### 1. Run the Database Migration
First, make sure the new chat schema is applied:

```bash
# Apply the migration (adjust command based on your migration system)
# This will create the new chat tables
```

### 2. Start the Go Backend
```bash
cd go-backend
go run main.go
```

The server should start on `http://localhost:8080`

### 3. Open the Test Frontend
Open `chat-test.html` in your web browser. You can either:
- Double-click the file to open in browser
- Use a local server: `python3 -m http.server 3000` and visit `http://localhost:3000/chat-test.html`

## 🧪 Testing Flow

### Step 1: Authentication
1. Use existing Zettelgarden credentials or create a test user
2. Default values are pre-filled: `test@example.com` / `password`
3. Click "Login" - you should see a success message

### Step 2: Create Conversation
1. Once authenticated, the chat interface appears
2. Click "+ New Conversation" to create your first chat
3. The conversation should appear in the list and become active

### Step 3: Test Chat with Tools
Try these test messages to verify tool integration:

**Test Basic Search:**
```
Find my cards about programming
```

**Test Specific Card Lookup:**
```
Show me card ID 1
```

**Test Hierarchical Browse:**
```
What are the child cards of my main project card?
```

**Test Metadata Filter:**
```
Show me my starred cards from this week
```

### Step 4: Verify Tool Calls
- Messages should show user input, then assistant response
- You should see "tool" messages in between showing the tool calls
- Tool results appear as JSON data

### Step 5: Check Usage Quotas
- Click "Refresh Quotas" to see your daily usage limits
- Verify that counters increment after sending messages

## 🔍 What to Look For

### ✅ Success Indicators
- [ ] Login works and shows authenticated state
- [ ] Conversations can be created and listed
- [ ] Messages can be sent and responses received
- [ ] Tool calls appear in the chat (as tool messages)
- [ ] Usage quotas display and update
- [ ] Error handling works gracefully

### ❌ Potential Issues
- **CORS errors**: Make sure backend allows frontend origin
- **Database errors**: Check if migration was applied
- **Tool errors**: Verify you have some test cards in the database
- **Rate limiting**: Check if quotas are properly configured

## 🛠️ Debugging

### Check Backend Logs
Watch for:
- Tool execution logs
- Database query errors
- Authentication issues
- LLM API calls

### Browser Console
- Check Network tab for API calls
- Look for JavaScript errors
- Verify request/response data

### Database Queries
Check if data is being created:
```sql
SELECT * FROM chat_conversations;
SELECT * FROM chat_messages;
SELECT * FROM chat_tool_calls;
SELECT * FROM chat_usage_quotas;
```

## 📊 Expected Tool Responses

The tools should return JSON data like:

**search_cards:**
```json
{
  "cards": [...],
  "query": "programming",
  "search_type": "semantic",
  "total": 5
}
```

**get_card_by_id:**
```json
{
  "id": 1,
  "title": "My Card",
  "body": "Card content...",
  "card_id": "main/project",
  "tags": ["programming", "project"]
}
```

The LLM should then use this data to provide helpful responses about your cards!