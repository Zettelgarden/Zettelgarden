# Phase 3: Incremental Domain Migration - Design

## Overview

### Current State

- **Phase 1.2 complete**: Tool handlers split into domain registration files (`card_tools.go`, `task_tools.go`, etc.) with simplified `RegisterTool()` API
- **Phase 3 partial (reverted)**: Attempted full migration of all 8 domains was reverted due to chat functionality breaking
- **Proof of concept**: `memory_tools` domain package exists but feature flag logic has been removed

### Approach: Ultra-Incremental with Staging Verification

Migrate **one domain at a time**, starting with the simplest (`template_tools` with 3 tools). Each migration will:

1. Create `services/tools/{domain}/` package with data access and business logic
2. Add feature flag to toggle between legacy and new paths
3. Merge to master with feature flag **OFF by default**
4. Deploy to staging and enable flag for 1-2 days of real chat testing
5. If issues arise, revert by flipping env var (5-minute rollback)
6. Once proven, enable flag in production and proceed to next domain

### Why This Works

- **Blast radius limited**: One domain at a time means issues are isolated
- **Real-world testing**: Actual chat conversations catch integration issues unit tests miss
- **Instant rollback**: Feature flag means reverting is just an env var change
- **Builds confidence**: Each successful migration proves the pattern before tackling more complex domains

---

## Template Tools Migration Pattern

### Target Domain: `template_tools`

**Why start here?**
- Only 3 tools (smallest domain)
- Well-contained functionality (template CRUD operations)
- No external API dependencies
- Clear success criteria (templates work or don't)

### File Structure Changes

```
services/
  ├── tools/                        # NEW
  │   └── template/                 # NEW
  │       ├── template.go           # NEW: Data access + business logic
  │       └── template_test.go      # NEW: Domain package tests
  ├── template_tools.go             # MODIFY: Add feature flag logic
  ├── templates.go                  # MODIFY: Re-export from domain package
  └── featureflags/
      └── flags.go                  # MODIFY: Add FeatureFlagTemplateTools
```

### What Moves Where

**To `services/tools/template/template.go`:**
- Data access functions (`GetTemplate`, `CreateTemplate`, etc.)
- Business logic that operates on templates
- Keeps template-specific logic isolated

**Stays in `services/template_tools.go`:**
- Tool registration (`RegisterTool()` calls)
- Handler functions (`handleGetTemplates`, `handleCreateTemplate`, etc.)
- Feature flag logic to switch between legacy/v2 paths

**Key point**: Handlers call the domain package functions instead of doing database work directly.

---

## Feature Flag Implementation

### Feature Flag Additions

In `services/featureflags/flags.go`, add:

```go
const (
    FeatureFlagMemoryTools = "memory_tools_v2"  // Already exists
    FeatureFlagTemplateTools = "template_tools_v2"  // NEW
)
```

Environment variable: `ZETTELGARDEN_FEATURE_TEMPLATE_TOOLS_V2=true`

### Registration Pattern in `template_tools.go`

```go
func (tr *ToolRegistry) RegisterTemplateTools() {
    if featureflags.IsEnabled(featureflags.FeatureFlagTemplateTools) {
        tr.registerTemplateToolsV2()  // New path
    } else {
        tr.registerTemplateToolsLegacy()  // Legacy path (default)
    }
}
```

### Handler Pattern

Both legacy and v2 handlers call the same domain package functions:

```go
func handleCreateTemplateLegacy(args, ctx) (result, error) {
    // Call domain package
    return template.CreateTemplate(ctx.DB, userID, name, content)
}

func handleCreateTemplateV2(args, ctx) (result, error) {
    // Same call to domain package
    return template.CreateTemplate(ctx.DB, userID, name, content)
}
```

**Why both paths?**: The feature flag tests the *registration pattern*, not the implementation. If issues arise, we flip back to legacy without changing any business logic.

---

## Rollback Plan & Testing Strategy

### Rollback Procedure (5 minutes or less)

If issues are detected in staging:

```bash
# SSH into staging server
ssh staging-server

# Disable the feature flag
unset ZETTELGARDEN_FEATURE_TEMPLATE_TOOLS_V2

# Or explicitly set to false
export ZETTELGARDEN_FEATURE_TEMPLATE_TOOLS_V2=false

# Restart the backend service
sudo systemctl restart zettelgarden-backend

# Verify chat is working
curl -X POST https://staging.zettelgarden.com/api/chat # test endpoint
```

**Why this works**: Feature flag defaults to OFF. Disabling it switches back to legacy path without any code changes.

### Staging Verification Protocol

Before merging to master with flag enabled:

1. **Deploy to staging** with feature flag OFF (default)
2. **Enable flag**: `export ZETTELGARDEN_FEATURE_TEMPLATE_TOOLS_V2=true` and restart
3. **Test chat conversations** that use templates:
   - "List my templates"
   - "Get template ID 1"
   - "Use template X to create a card"
4. **Monitor for 24-48 hours** for any chat errors or tool failures
5. **If issues**: Disable flag and investigate
6. **If clean**: Enable flag in production

### Success Criteria

- Chat conversations using templates work identically with flag on/off
- No increase in error rates
- No performance degradation
- All existing tests pass

---

## Remaining Domains & Implementation Order

### Domain Migration Roadmap

After `template_tools` is proven in production, migrate remaining domains in order of complexity:

| Domain | Tools | Complexity | Dependencies | Notes |
|--------|-------|------------|--------------|-------|
| **template** | 3 | Simple | None | **Start here** |
| **calendar** | 3 | Medium | External CalDAV API | Has external deps, good test of integration |
| **article** | 2 | Medium | URL parsing, HTML | Small tool count but external fetches |
| **fact** | 4 | Medium | facts.go | Moderate complexity |
| **task** | 7 | High | tasks.go, task_statuses.go | Many handlers, more complex |
| **entity** | 10 | High | entity.go | Most tools, complex relationships |
| **card** | 6 | High | cards.go, search | Core domain, heavily used |

### Implementation Checklist per Domain

For each domain migration:

- [ ] Create `services/tools/{domain}/{domain}.go` with data access functions
- [ ] Add feature flag constant to `featureflags/flags.go`
- [ ] Update `{domain}_tools.go` with feature flag logic
- [ ] Write/update tests for both paths
- [ ] Run full test suite: `go test ./...`
- [ ] Create PR with flag OFF by default
- [ ] Merge to master
- [ ] Deploy to staging
- [ ] Enable flag in staging: `export ZETTELGARDEN_FEATURE_{DOMAIN}_TOOLS_V2=true`
- [ ] Test chat conversations for 1-2 days
- [ ] If clean, enable in production; else disable and debug

### Migration Progress (as of Feb 6, 2026)

- [x] **template_tools** (3 tools) - ✅ Merged to master (commit: 9f0d3e5b)
- [x] **calendar_tools** (3 tools) - ✅ Merged to master (commit: 720c2e3a)
- [x] **article_tools** (2 tools) - ✅ Merged to master (commit: 4108dc21)
- [x] **fact_tools** (4 tools) - ✅ Merged to master (commit: 33beae9a)
- [x] **task_tools** (7 tools) - ✅ Merged to master (commit: d9a99307)
- [ ] **entity_tools** (10 tools) - Next domain
- [ ] **card_tools** (6 tools) - Final domain

**Status: 5 of 7 domains complete (71%)**

### Total Timeline Estimate

- 1 domain per week = ~7 weeks for all 7 remaining domains
- Each domain: 1-2 days dev + 1-2 days staging verification
- **On track**: 5 domains completed in first session

---

## Pre-work Completed

### Cleanup (Feb 6, 2026)

- Removed feature flag logic from `services/memory_tools.go`
- Simplified `services/memory_tools_test.go` to remove feature flag tests
- Feature flag infrastructure remains available for use in new migrations

### Files Modified

- `services/memory_tools.go` - Simplified to single registration path
- `services/memory_tools_test.go` - Removed feature flag test cases

---

## References

- Original Phase 3 attempt: commit `41cbb71b`
- Revert: commit `b183f3de`
- Phase 1.2 (simplified registration): commit `6b39d358`
- Phase 3 documentation: `services/PHASE3_DOMAIN_MIGRATION_GUIDE.md`
