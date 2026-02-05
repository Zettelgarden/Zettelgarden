# Phase 3 Architecture Diagram

## Component Overview

```
┌─────────────────────────────────────────────────────────────────────────┐
│                         Application Layer                               │
│                      (go-backend/main.go)                               │
└─────────────────────────────────────────────────────────────────────────┘
                                   │
                                   ▼
┌─────────────────────────────────────────────────────────────────────────┐
│                      Services Package Layer                              │
│                    (services/registry.go)                                │
│                                                                         │
│  ┌───────────────────────────────────────────────────────────────────┐ │
│  │                    ToolRegistry                                     │ │
│  │                                                                    │ │
│  │  tools: map[string]Tool                                           │ │
│  │                                                                    │ │
│  │  RegisterCardTools()    → services/card_tools.go                  │ │
│  │  RegisterTaskTools()    → services/task_tools.go                  │ │
│  │  RegisterMemoryTools()  → services/memory_tools.go  ◄───── Phase 3│ │
│  │  ...                                                                │ │
│  └───────────────────────────────────────────────────────────────────┘ │
└─────────────────────────────────────────────────────────────────────────┘
                                   │
                                   ▼
┌─────────────────────────────────────────────────────────────────────────┐
│                 Feature Flag Control Layer                               │
│              (services/featureflags/flags.go)                            │
│                                                                         │
│  ZETTELGARDEN_FEATURE_MEMORY_TOOLS_V2=true                              │
│                                                                         │
│  IsEnabled(featureflags.FeatureFlagMemoryTools)  →  true/false         │
└─────────────────────────────────────────────────────────────────────────┘
                                   │
                                   ▼
┌─────────────────────────────────────────────────────────────────────────┐
│              Memory Tools Registration Layer                             │
│            (services/memory_tools.go)                                    │
│                                                                         │
│  func (tr *ToolRegistry) RegisterMemoryTools() {                        │
│      if featureflags.IsEnabled(FeatureFlagMemoryTools) {                │
│          tr.registerMemoryToolsV2()      // New path                    │
│      } else {                                                            │
│          tr.registerMemoryToolsLegacy()  // Legacy path                  │
│      }                                                                  │
│  }                                                                      │
│                                                                         │
│  Both paths call: memory.GetUserMemory(ctx.DB, ctx.UserID)              │
└─────────────────────────────────────────────────────────────────────────┘
                                   │
                                   ▼
┌─────────────────────────────────────────────────────────────────────────┐
│               Domain Package Layer (NEW in Phase 3)                      │
│           (services/tools/memory/memory.go)                              │
│                                                                         │
│  ┌─────────────────────────────────────────────────────────────────┐   │
│  │  Memory Domain Package                                           │   │
│  │                                                                  │   │
│  │  GetUserMemory(db, userID) → string, error                       │   │
│  │  UpdateUserMemory(db, userID, memory) → error                    │   │
│  │                                                                  │   │
│  │  • No circular imports                                          │   │
│  │  • Domain-specific business logic                               │   │
│  │  • Testable in isolation                                        │   │
│  └─────────────────────────────────────────────────────────────────┘   │
└─────────────────────────────────────────────────────────────────────────┘
                                   │
                                   ▼
┌─────────────────────────────────────────────────────────────────────────┐
│                       Database Layer                                      │
│                      (PostgreSQL)                                         │
│                                                                         │
│  user_memories table                                                    │
│  - user_id                                                              │
│  - memory                                                               │
│  - created_at                                                           │
│  - updated_at                                                           │
└─────────────────────────────────────────────────────────────────────────┘
```

## Data Flow

### Legacy Path (Feature Flag Disabled)

```
ToolRegistry.RegisterMemoryTools()
    │
    ▼
registerMemoryToolsLegacy()
    │
    ├── RegisterTool("get_user_memory", handleGetUserMemoryLegacy)
    │
    ▼
handleGetUserMemoryLegacy(args, ctx)
    │
    ▼
memory.GetUserMemory(ctx.DB, ctx.UserID)  ← Domain package
    │
    ▼
Database Query → Result
```

### New Path (Feature Flag Enabled)

```
ToolRegistry.RegisterMemoryTools()
    │
    ▼
registerMemoryToolsV2()
    │
    ├── RegisterTool("get_user_memory", handleGetUserMemoryV2)
    │
    ▼
handleGetUserMemoryV2(args, ctx)
    │
    ▼
memory.GetUserMemory(ctx.DB, ctx.UserID)  ← Domain package
    │
    ▼
Database Query → Result
```

**Key Point**: Both paths use the same domain package, ensuring identical behavior.

## Import Graph

```
services/
│
├── registry.go
│   └── imports: (no domain packages)
│
├── memory_tools.go
│   ├── imports: services/featureflags
│   └── imports: services/tools/memory
│
├── memory.go
│   └── imports: services/tools/memory
│
├── featureflags/
│   └── flags.go (no domain imports)
│
└── tools/
    └── memory/
        └── memory.go (no services imports - avoids circular dependency)
```

## Backward Compatibility

```
External Code (handlers, tests, etc.)
    │
    ▼
services.GetUserMemory()  ← Re-export from domain package
    │
    ▼
services/tools/memory.GetUserMemory()  ← Actual implementation
```

This ensures existing code continues to work without changes.

## Migration Pattern

For each domain (memory → template → calendar → article → fact → task → entity → card):

1. Create `services/tools/{domain}/` package
2. Move data access functions to domain package
3. Add feature flag constant
4. Update `{domain}_tools.go` with feature flag logic
5. Update backward compatibility re-exports
6. Test both paths

## Benefits

1. **Incremental Rollout**: Enable domains one at a time
2. **Easy Rollback**: Disable flag instantly
3. **Better Organization**: Domain logic in separate packages
4. **No Circular Imports**: Clean dependency graph
5. **Backward Compatible**: No breaking changes
6. **Parallel Development**: Teams can work independently

## Feature Flag Environment Variables

```bash
# Enable memory_tools v2 (new domain package)
ZETTELGARDEN_FEATURE_MEMORY_TOOLS_V2=true

# Future flags:
# ZETTELGARDEN_FEATURE_TEMPLATE_TOOLS_V2=true
# ZETTELGARDEN_FEATURE_CARD_TOOLS_V2=true
# ZETTELGARDEN_FEATURE_TASK_TOOLS_V2=true
# etc.
```
