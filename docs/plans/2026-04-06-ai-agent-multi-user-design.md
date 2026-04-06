# AI Agent Multi-User Support Design

**Date:** 2026-04-06
**Status:** Approved
**Approach:** Minimal Backend + Workspace Switcher

## Overview

Enable users to connect external AI agents to their Zettelgarden account. Agents can:
- Maintain their own isolated workspace (separate cards, tags, links)
- Write to the user's workspace with full auditability
- Authenticate via API keys
- Be managed through a simple admin interface

This leverages the existing user-based CRUD architecture by treating agents as special user accounts.

## Goals

- Allow users to create/manage AI agent access via API keys
- Enable agents to have isolated workspaces
- Allow agents to contribute to user's workspace with tracking
- Minimal backend changes (reuse existing user_id filtering)
- Simple, focused management UI

## Non-Goals

- Building AI agents within Zettelgarden (agents are external)
- Complex permission systems or ACLs
- Agent dashboards with rich analytics
- Multi-tenant team collaboration (agents only, not human collaborators)

## Architecture

### Data Model

Agents are stored in the `users` table with special flags:

```sql
-- Migration 1: Add agent support to users
ALTER TABLE users ADD COLUMN is_agent BOOLEAN DEFAULT FALSE;
ALTER TABLE users ADD COLUMN owner_user_id INT NULL REFERENCES users(id) ON DELETE CASCADE;
ALTER TABLE users ADD COLUMN api_key_hash VARCHAR(255) NULL;
```

User types:
- **Regular user:** `is_agent=false`, `owner_user_id=NULL`, `api_key_hash=NULL`
- **Agent user:** `is_agent=true`, `owner_user_id=<owner's id>`, `api_key_hash=<hashed key>`

Constraints:
- Agents cannot be admins (`is_admin` must be false when `is_agent=true`)
- Agents inherit subscription status from owner (no separate billing)
- Agent accounts are deactivated when API key is revoked (hash set to NULL)

### Workspace Isolation

Agents have two contexts:
1. **Agent's own workspace:** Cards with `user_id = agent_id` (isolated)
2. **Owner's workspace:** Cards with `user_id = owner_id` and `created_by_agent_id = agent_id` (contributing)

```sql
-- Migration 2: Track card creation source
ALTER TABLE cards ADD COLUMN created_by_agent_id INT NULL REFERENCES users(id) ON DELETE SET NULL;
```

### Activity Tracking

All agent actions are logged for auditability:

```sql
-- Migration 3: Agent activity log
CREATE TABLE agent_activity_log (
    id SERIAL PRIMARY KEY,
    agent_id INT NOT NULL REFERENCES users(id) ON DELETE CASCADE,
    action VARCHAR(50) NOT NULL,
    target_type VARCHAR(50) NOT NULL,
    target_id INT,
    details JSONB,
    created_at TIMESTAMP DEFAULT NOW()
);

CREATE INDEX idx_agent_activity_agent ON agent_activity_log(agent_id);
CREATE INDEX idx_agent_activity_created ON agent_activity_log(created_at DESC);
```

## Backend API

### New Endpoints

#### 1. Create Agent
```
POST /api/agents
Authorization: Bearer <user JWT>
Body: { "name": "Claude Research Agent", "description": "Optional" }
Response: {
  "id": 42,
  "name": "Claude Research Agent",
  "api_key": "zg_live_abc123def456...",
  "created_at": "2026-04-06T10:00:00Z"
}
```

Creates agent user account, generates API key (shown only once).

#### 2. List Agents
```
GET /api/agents
Authorization: Bearer <user JWT>
Response: {
  "agents": [
    {
      "id": 42,
      "name": "Claude Research Agent",
      "description": "...",
      "created_at": "...",
      "last_used": "..."
    }
  ]
}
```

#### 3. Revoke Agent
```
DELETE /api/agents/:id
Authorization: Bearer <user JWT>
Response: 204 No Content
```

Sets `api_key_hash` to NULL, preventing authentication.

#### 4. Get Agent Activity
```
GET /api/agents/:id/activity?page=1&per_page=50
Authorization: Bearer <user JWT>
Response: {
  "logs": [
    {
      "id": 1,
      "action": "create_card",
      "target_type": "card",
      "target_id": 123,
      "details": {"title": "..."},
      "created_at": "..."
    }
  ],
  "pagination": {"page": 1, "per_page": 50, "total": 127}
}
```

### Authentication Changes

Extend `APIKeyOrJWTMiddleware` to recognize agents:

```go
// When API key is provided:
// 1. Look up key_hash in users table
// 2. If found and is_agent=true:
//    - Set current_user = agent_id
//    - Set owner_user_id in context
// 3. If found and is_agent=false:
//    - Set current_user = user_id (existing behavior)
```

### Card Operation Changes

When agent creates/updates cards:

```go
func (s *Handler) CreateCard(w http.ResponseWriter, r *http.Request) {
    currentUserID := r.Context().Value("current_user").(int)
    agentID := r.Context().Value("agent_id")  // nil if not agent
    ownerID := r.Context().Value("owner_user_id")  // nil if not agent
    
    targetUserID := getTargetUserIDFromRequest(r)  // from body or query
    
    if agentID != nil {
        // Agent is making the request
        if targetUserID == ownerID {
            // Writing to owner's workspace
            card.UserID = ownerID
            card.CreatedByAgentID = agentID
        } else if targetUserID == currentUserID {
            // Writing to own workspace
            card.UserID = agentID
            // CreatedByAgentID remains NULL
        } else {
            // Unauthorized
            return http.Error(w, "Forbidden", http.StatusForbidden)
        }
    } else {
        // Regular user
        card.UserID = currentUserID
    }
    
    // ... existing creation logic
}
```

### Activity Logging

Log all agent mutations:

```go
func (s *Handler) logAgentActivity(agentID int, action, targetType string, targetID int, details map[string]interface{}) {
    go func() {
        detailsJSON, _ := json.Marshal(details)
        s.GetDB().Exec(`
            INSERT INTO agent_activity_log (agent_id, action, target_type, target_id, details)
            VALUES ($1, $2, $3, $4, $5)
        `, agentID, action, targetType, targetID, detailsJSON)
    }()
}
```

## Frontend Changes

### 1. Agent Management Page

**Route:** `/settings/agents`

Simple interface with:
- Table: Name, Description, Created, Last Used, Actions
- "Create Agent" button → modal
- "Revoke" button → confirmation
- "View Activity" link → activity log for that agent

### 2. Workspace Switcher Component

**Location:** Main navigation bar

Dropdown menu:
```
👤 My Workspace
🤖 Claude Research Agent
🤖 Data Processing Bot
```

Behavior:
- Updates `currentWorkspace` in context
- All queries use selected workspace's user ID
- URL updates: `?workspace=<user_id>`

### 3. Card Display Updates

- Badge on cards: "Created by [Agent Name]" when `created_by_agent_id` is set
- Optional: subtle visual distinction (light background color)
- Metadata section shows creator info

### 4. Context Updates

**AuthContext additions:**
```typescript
interface AuthContextType {
  // ... existing fields
  currentWorkspace: number;  // user_id of current workspace
  isViewingAgentWorkspace: boolean;
  switchWorkspace: (workspaceUserId: number) => void;
}
```

**Query modifications:**
- All card/tag/task queries include `workspace_user_id` parameter
- Default: current user's ID
- When switched: selected workspace's user ID

### 5. API Client

New file: `src/api/agents.ts`

```typescript
export interface Agent {
  id: number;
  name: string;
  description?: string;
  created_at: string;
  last_used?: string;
}

export interface AgentActivity {
  id: number;
  action: string;
  target_type: string;
  target_id: number;
  details: Record<string, any>;
  created_at: string;
}

export const createAgent = (name: string, description?: string): Promise<Agent & { api_key: string }> => ...
export const listAgents = (): Promise<Agent[]> => ...
export const revokeAgent = (agentId: number): Promise<void> => ...
export const getAgentActivity = (agentId: number, page?: number): Promise<{ logs: AgentActivity[], pagination: any }> => ...
```

## Migration Strategy

1. **Deploy migrations** (zero-downtime, all columns nullable)
2. **Deploy backend changes** (API endpoints, auth middleware)
3. **Deploy frontend changes** (agent management UI, workspace switcher)
4. **Test with internal agent** (create test agent, verify workflows)

## Security Considerations

- API keys are shown only once (at creation)
- Keys are hashed before storage (bcrypt)
- Agents cannot be admins
- Agents can only write to owner's workspace or own workspace
- All actions logged for auditability
- Revocation is instant (key hash set to NULL)

## Success Criteria

- Users can create/revoke agents via UI
- Agents can authenticate with API keys
- Agents can maintain isolated workspaces
- Agents can contribute to user's workspace with tracking
- All agent actions appear in activity log
- Workspace switching works smoothly
- Existing user workflows unchanged

## Future Enhancements (Out of Scope)

- Fine-grained permissions (write to specific tags only)
- Agent rate limiting
- Multiple API keys per agent
- Agent collaboration (multiple users sharing an agent)
- Rich agent analytics dashboard
