# Memory Domain Package

## Overview

This is the first domain package created as part of Phase 3: Split Tool Handlers by Domain with Feature Flags.

The memory domain contains tools and data access functions for managing user memory and observations in Zettelgarden.

## Package Structure

```
services/tools/memory/
├── memory.go       # Data access and business logic
├── memory_test.go  # Domain tests
└── README.md       # This file
```

## Exported Functions

### GetUserMemory
```go
func GetUserMemory(db *sql.DB, userID int) (string, error)
```
Retrieves the user's memory from the database. Returns empty string if no memory exists.

### UpdateUserMemory
```go
func UpdateUserMemory(db *sql.DB, userID uint, memory string) error
```
Updates or creates a user's memory in the database.

## Integration with Tool Registry

The memory domain integrates with the tool registry via `services/memory_tools.go`:

1. **Feature Flag Disabled (Default)**: Uses `registerMemoryToolsLegacy()`
2. **Feature Flag Enabled**: Uses `registerMemoryToolsV2()`

Both paths call this package's functions internally, ensuring consistent behavior.

Enable the feature flag:
```bash
export ZETTELGARDEN_FEATURE_MEMORY_TOOLS_V2=true
```

## Pattern for Other Domains

This package serves as a template for other domain packages:

1. Create `services/tools/{domain}/` directory
2. Add domain-specific data access functions
3. Move business logic from `services/{domain}_tools.go`
4. Register tools via feature flag in parent package
5. Maintain backward compatibility with re-exports

## Testing

```bash
# Test the memory domain package
go test ./services/tools/memory/...

# Test with feature flag enabled
ZETTELGARDEN_FEATURE_MEMORY_TOOLS_V2=true go test ./services/... -run TestMemory

# Test with feature flag disabled (default)
go test ./services/... -run TestMemory
```

## Migration Notes

- Created: Phase 3 (2025-02-05)
- Status: Proof of concept - COMPLETE
- Pattern validated: Yes
- Ready for production: Yes (when feature flag is enabled)

## Related Files

- `services/memory_tools.go` - Tool registration with feature flags
- `services/memory.go` - Backward-compatible re-exports
- `services/featureflags/flags.go` - Feature flag system
- `services/PHASE3_DOMAIN_MIGRATION_GUIDE.md` - Complete migration guide
