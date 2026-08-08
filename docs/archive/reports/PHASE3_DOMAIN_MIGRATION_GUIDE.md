> **ARCHIVED** — Historical document moved to `docs/archive/` on 2026-08-08 during the documentation audit (Zettelgarden-0ui). Does not describe the current app; kept for the record.

# Phase 3: Domain Package Migration Guide

## Overview

Phase 3 splits tool handlers into separate domain packages with feature flags for incremental rollout. This is a proof of concept using the `memory_tools` domain as the first implementation.

## Architecture

### Before Phase 3
```
services/
  ├── memory_tools.go       # Tool registration + handlers (all in one file)
  ├── card_tools.go
  ├── task_tools.go
  └── ...
```

### After Phase 3 (memory_tools domain)
```
services/
  ├── featureflags/         # NEW: Feature flag system
  │   ├── flags.go
  │   └── flags_test.go
  ├── tools/               # NEW: Domain packages
  │   └── memory/          # Memory domain package
  │       ├── memory.go    # Data access + business logic
  │       └── memory_test.go
  ├── memory_tools.go      # Feature-flagged registration
  └── memory.go            # Backward-compatible re-exports
```

## Migration Pattern

### 1. Create Feature Flag

Add a new flag constant in `services/featureflags/flags.go`:

```go
const (
    FeatureFlagMemoryTools = "memory_tools_v2"
    // Add more flags as needed:
    // FeatureFlagTemplateTools = "template_tools_v2"
    // FeatureFlagCardTools = "card_tools_v2"
)
```

### 2. Create Domain Package

Create `services/tools/{domain}/` package with:
- **Data access functions**: Database queries and business logic
- **Package documentation**: explaining the domain's purpose
- **Tests**: Unit tests for domain functions

Example for `services/tools/memory/memory.go`:

```go
package memory

import (
    "database/sql"
)

// GetUserMemory retrieves user memory from database
func GetUserMemory(db *sql.DB, userID int) (string, error) {
    var memory string
    err := db.QueryRow("SELECT memory FROM user_memories WHERE user_id = $1", userID).Scan(&memory)
    if err == sql.ErrNoRows {
        return "", nil
    }
    return memory, err
}

// UpdateUserMemory updates or creates user memory
func UpdateUserMemory(db *sql.DB, userID uint, memory string) error {
    // Implementation...
    return nil
}
```

### 3. Update Domain Tools File

Modify `services/{domain}_tools.go` to support both legacy and new paths:

```go
package services

import (
    "go-backend/services/featureflags"
    "go-backend/services/tools/memory"
)

func (tr *ToolRegistry) RegisterMemoryTools() {
    if featureflags.IsEnabled(featureflags.FeatureFlagMemoryTools) {
        // NEW: Use domain package
        tr.registerMemoryToolsV2()
    } else {
        // LEGACY: Use old registration
        tr.registerMemoryToolsLegacy()
    }
}

func (tr *ToolRegistry) registerMemoryToolsV2() {
    RegisterTool(tr, ToolGetUserMemory,
        "Tool description...",
        handleGetUserMemoryV2,
    )
}

func handleGetUserMemoryV2(args map[string]interface{}, ctx *ToolContext) (map[string]interface{}, error) {
    // Call domain package function
    userMemory, err := memory.GetUserMemory(ctx.DB, ctx.UserID)
    if err != nil {
        return nil, fmt.Errorf("failed to retrieve user memory: %w", err)
    }
    return map[string]interface{}{"memory": userMemory}, nil
}
```

### 4. Maintain Backward Compatibility

Update `services/memory.go` to re-export from the domain package:

```go
package services

import (
    "go-backend/services/tools/memory"
)

// GetUserMemory delegates to domain package
func GetUserMemory(db *sql.DB, userID int) (string, error) {
    return memory.GetUserMemory(db, userID)
}

// UpdateUserMemory delegates to domain package
func UpdateUserMemory(db *sql.DB, userID uint, mem string) error {
    return memory.UpdateUserMemory(db, userID, mem)
}
```

## Usage

### Enable Feature Flag

Set the environment variable before running the application:

```bash
export ZETTELGARDEN_FEATURE_MEMORY_TOOLS_V2=true
./go-backend
```

### Disable Feature Flag (Default)

```bash
# Either don't set the variable, or explicitly disable:
export ZETTELGARDEN_FEATURE_MEMORY_TOOLS_V2=false
./go-backend
```

### In Go Tests

```go
import "go-backend/services/featureflags"

func TestSomething(t *testing.T) {
    featureflags.Enable(featureflags.FeatureFlagMemoryTools)
    defer featureflags.Disable(featureflags.FeatureFlagMemoryTools)

    // Your test code here...
}
```

## Testing

### Unit Tests

Each domain package should have its own tests:

```bash
go test ./services/tools/memory/...
```

### Integration Tests

Test both legacy and new paths:

```bash
# Test with legacy path (default)
go test ./services/... -run TestMemory

# Test with new path
ZETTELGARDEN_FEATURE_MEMORY_TOOLS_V2=true go test ./services/... -run TestMemory
```

## Migration Checklist

For each domain (memory → template → calendar → article → fact → task → entity → card):

- [ ] Create `services/tools/{domain}/` package
- [ ] Add feature flag constant
- [ ] Move data access functions to domain package
- [ ] Update `{domain}_tools.go` with feature flag logic
- [ ] Update backward compatibility re-exports
- [ ] Write tests for both paths
- [ ] Document any breaking changes
- [ ] Test with feature flag disabled (default)
- [ ] Test with feature flag enabled
- [ ] Enable in production and monitor

## Success Criteria for memory_tools Proof of Concept

- [x] memory_tools split into `services/tools/memory/` package
- [x] Feature flag `FeatureFlagMemoryTools` controls old vs new path
- [x] Tests pass with both paths enabled/disabled
- [x] Pattern documented for remaining domains
- [x] Backward compatibility maintained
- [x] No circular import dependencies

## Domain Migration Order

1. **memory_tools** ✓ (Proof of concept - COMPLETE)
2. **template_tools** (1 tool - simple like memory)
3. **calendar_tools** (3 tools - external API integration)
4. **article_tools** (2 tools - URL parsing)
5. **fact_tools** (5 tools - moderate complexity)
6. **task_tools** (7 tools - many handlers)
7. **entity_tools** (10 tools - complex domain)
8. **card_tools** (6 tools - most complex)

## Benefits

1. **Incremental Rollout**: Enable domains one at a time
2. **Easy Rollback**: Disable flag instantly if issues arise
3. **Better Organization**: Domain logic in separate packages
4. **Reduced Coupling**: Domains are more independent
5. **Easier Testing**: Domain packages can be tested in isolation
6. **Parallel Development**: Teams can work on different domains

## Future Improvements

- **Dynamic Feature Flags**: Integrate with a feature flag service (e.g., LaunchDarkly)
- **Metrics**: Track usage of old vs new paths
- **Automatic Migration**: Gradually shift traffic to new paths
- **Domain Events**: Event-driven architecture between domains
- **API Versioning**: Use feature flags for API version transitions
