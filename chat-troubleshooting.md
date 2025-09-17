# Chat Feature Troubleshooting Guide

## 🔧 Fixed: JSON Marshaling Error

**Error**: `pq: invalid input syntax for type json`

**Solution**: Updated the JSONB handling in `handlers/chat.go` to properly handle null values:
- Changed from `[]byte` to `*string` for JSONB fields
- Added proper null handling for tool_calls
- Fixed JSON marshaling/unmarshaling for PostgreSQL JSONB

## 🚀 Testing Steps

### 1. Apply Database Migration
Make sure the chat schema is applied:
```sql
-- Run the migration file
\i go-backend/schema/0067-chat-system.sql

-- Verify tables exist
SELECT table_name FROM information_schema.tables
WHERE table_schema = 'public' AND table_name LIKE 'chat_%';
```

### 2. Start Backend
```bash
cd go-backend
go run main.go
```

### 3. Test Basic Flow
1. **Login** to the React app
2. **Navigate to Chat** via sidebar
3. **Create conversation** with "+ New Chat"
4. **Send a simple message** like "Hello"

## 🐛 Common Issues & Solutions

### Issue: "No conversations yet"
**Cause**: Database migration not applied or API errors
**Solution**:
- Check backend logs for errors
- Verify chat tables exist in database
- Test API endpoints manually

### Issue: Messages not saving
**Cause**: JSON marshaling or database constraint issues
**Solution**:
- Check backend logs for specific errors
- Verify user has valid subscription
- Test with simple text messages first

### Issue: Tool calls not working
**Cause**: Missing cards in database or tool execution errors
**Solution**:
- Verify you have cards in your database
- Check if search API works independently
- Look for tool execution errors in logs

### Issue: Frontend errors
**Cause**: CORS, authentication, or API connection issues
**Solution**:
- Check browser console for specific errors
- Verify VITE_URL environment variable
- Ensure you're logged in with valid token

## 🧪 Test Queries

### Manual API Testing
```bash
# Create conversation
curl -X POST http://localhost:8080/api/chat/conversations \
  -H "Authorization: Bearer YOUR_TOKEN" \
  -H "Content-Type: application/json" \
  -d '{"title": "Test Chat", "model": "gpt-4o-mini"}'

# Send message
curl -X POST http://localhost:8080/api/chat/conversations/CONV_ID/messages \
  -H "Authorization: Bearer YOUR_TOKEN" \
  -H "Content-Type: application/json" \
  -d '{"content": "Hello world"}'
```

### Database Checks
```sql
-- Check conversations
SELECT * FROM chat_conversations LIMIT 5;

-- Check messages
SELECT id, role, content, tool_calls FROM chat_messages LIMIT 5;

-- Check usage quotas
SELECT * FROM chat_usage_quotas;
```

## 📊 What Should Work Now

1. ✅ **Basic messaging**: User messages should save and display
2. ✅ **Conversation management**: Create, list, select conversations
3. ✅ **JSONB handling**: No more JSON marshaling errors
4. ✅ **Tool integration**: Tools should execute (if cards exist)
5. ✅ **Usage quotas**: Should track and display correctly
6. ✅ **React integration**: Full UI should work seamlessly

## 🔍 Debugging Tips

### Backend Logs
Look for:
- `Error saving user message:` - Database issues
- `Error executing tool:` - Tool execution problems
- `Error generating chat response:` - LLM integration issues

### Database Debugging
```sql
-- Check if user has subscription
SELECT has_subscription FROM users WHERE id = YOUR_USER_ID;

-- Check recent chat activity
SELECT c.title, m.role, m.content, m.created_at
FROM chat_conversations c
JOIN chat_messages m ON c.id = m.conversation_id
ORDER BY m.created_at DESC LIMIT 10;
```

### React Debugging
- Check Network tab for failed API calls
- Look for console errors in browser
- Verify chat state in React DevTools

The JSON marshaling issue has been fixed - try testing again! 🎉