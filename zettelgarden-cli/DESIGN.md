# Zettelgarden CLI Design Document

## Overview

A command-line interface tool for interacting with Zettelgarden, designed primarily for LLM agent consumption rather than direct human use. The tool will communicate with the Zettelgarden backend API over HTTP/HTTPS with full authentication support.

## Primary Design Goals

### 1. LLM-Optimized Output
- **Structured, parseable formats**: JSON as primary output format with optional compact modes
- **Context-efficient**: Minimal token usage while preserving essential information
- **Deterministic**: Consistent output formats for reliable parsing
- **Field selection**: Allow filtering output to only requested fields to reduce token count
- **Pagination awareness**: Smart defaults for list operations to avoid overwhelming context windows

### 2. Programmatic Interface
- **Exit codes**: Meaningful status codes for script integration (0=success, 1=error, 2=not found, etc.)
- **Stderr separation**: Errors and warnings to stderr, data to stdout
- **Machine-readable errors**: JSON error format option for programmatic error handling
- **Idempotent operations**: Safe to retry operations where possible

### 3. Remote API Communication
- **HTTP client**: Full REST API client for Zettelgarden backend
- **Authentication**: JWT token management with persistent storage
- **Configuration**: Support for multiple environments/instances
- **Network resilience**: Timeouts, retries, and clear error messages

## Core Functionality (Phase 1: Cards)

### Card Operations

#### Read Operations
- `zg card get <id>` - Retrieve a single card
- `zg card list [--limit N] [--offset N] [--parent-id ID]` - List cards with pagination
- `zg card search <query> [--limit N]` - Search cards (text or vector)
- `zg card children <id>` - Get child cards of a parent
- `zg card links <id>` - Get backlinks and forward links
- `zg card starred` - List starred cards

#### Write Operations
- `zg card create [--title TITLE] [--body BODY] [--parent-id ID]` - Create new card
- `zg card update <id> [--title TITLE] [--body BODY]` - Update existing card
- `zg card delete <id>` - Delete card
- `zg card star <id>` - Star a card
- `zg card unstar <id>` - Unstar a card

#### Analysis Operations
- `zg card summary <id>` - Get AI-generated summary
- `zg card analyze <id>` - Get AI analysis
- `zg card entities <id>` - List linked entities (PRO feature)

### Output Format Considerations

#### Default Output (JSON)
```json
{
  "id": "123",
  "title": "Card Title",
  "body": "Card content...",
  "created_at": "2025-10-18T10:00:00Z",
  "updated_at": "2025-10-18T10:00:00Z",
  "parent_id": null,
  "is_starred": false
}
```

#### Compact Mode (--compact)
Minimize whitespace, exclude null fields, use shorter field names where unambiguous.

#### Field Selection (--fields)
```bash
zg card get 123 --fields id,title,body
# Returns only requested fields
```

#### List Output (Standard)
```json
{
  "cards": [...],
  "total": 100,
  "limit": 20,
  "offset": 0,
  "has_more": true
}
```

#### Streaming Output (--stream)
For large list operations, support NDJSON (newline-delimited JSON) streaming format:
```bash
zg card list --stream
# Outputs one JSON object per line, allowing incremental processing
```

Example output:
```ndjson
{"id":"1","title":"First Card","body":"..."}
{"id":"2","title":"Second Card","body":"..."}
{"id":"3","title":"Third Card","body":"..."}
```

Benefits for LLM agents:
- Process results incrementally without waiting for complete response
- Reduced memory footprint for large result sets
- Ability to stop processing early if enough results found
- Natural pagination-free interface for large datasets

## Authentication Strategy

### JWT Token Management

1. **Initial Login**
   ```bash
   zg auth login --email user@example.com --password <password>
   # Or with environment variable
   ZG_EMAIL=user@example.com ZG_PASSWORD=<password> zg auth login
   ```

2. **Token Storage**
   - Store JWT in config file: `~/.config/zettelgarden/credentials.json`
   - Format:
     ```json
     {
       "default": {
         "endpoint": "https://zettelgarden.com",
         "token": "eyJ...",
         "token_expiry": "2025-10-19T10:00:00Z"
       }
     }
     ```

3. **Profile Management**
   ```bash
   zg auth login --profile production
   zg card list --profile staging
   export ZG_PROFILE=development
   ```

4. **Token Refresh**
   - Automatically refresh if endpoint supports it
   - Otherwise, clear error message when token expires

### Configuration

#### Config File (`~/.config/zettelgarden/config.json`)
```json
{
  "default_profile": "default",
  "profiles": {
    "default": {
      "endpoint": "https://api.zettelgarden.com",
      "timeout": 30
    },
    "local": {
      "endpoint": "http://localhost:8080",
      "timeout": 10
    }
  },
  "output": {
    "format": "json",
    "compact": false,
    "color": false
  }
}
```

#### Environment Variables
- `ZG_PROFILE` - Active profile
- `ZG_ENDPOINT` - Override endpoint
- `ZG_TOKEN` - Override token (for CI/CD)
- `ZG_OUTPUT_FORMAT` - Output format (json, compact-json)
- `ZG_NO_COLOR` - Disable colored output

## Implementation Technology

### Language Choice: Go
**Rationale:**
- Same language as backend - shared models/types possible
- Excellent HTTP client libraries
- Single binary distribution - no dependencies
- Fast execution - minimal overhead for LLM agents
- Cross-platform compilation

### Key Dependencies
- **cobra** - CLI framework with subcommands
- **viper** - Configuration management
- HTTP client using standard library or `resty`
- JWT parsing (if needed for token validation)

### Project Structure
```
zettelgarden-cli/
├── cmd/
│   ├── root.go           # Root command and global flags
│   ├── auth.go           # Authentication commands
│   ├── card.go           # Card commands
│   └── config.go         # Configuration commands
├── internal/
│   ├── api/
│   │   ├── client.go     # HTTP client wrapper
│   │   ├── auth.go       # Auth API calls
│   │   └── cards.go      # Card API calls
│   ├── config/
│   │   ├── config.go     # Config management
│   │   └── credentials.go # Credential storage
│   └── output/
│       ├── formatter.go  # Output formatting
│       └── fields.go     # Field filtering
├── main.go
├── go.mod
├── go.sum
└── DESIGN.md
```

## Context Management for LLMs

### Key Strategies

1. **Default Limits**
   - Card body: Truncate at 1000 chars by default, use `--full` for complete content
   - List operations: Default to 10 items, require explicit `--limit` for more
   - Search results: Return only top 5 by default

2. **Progressive Disclosure**
   - Minimal info by default (id, title, snippet)
   - Use `--verbose` for full details
   - Separate commands for related data (children, links, entities)

3. **Efficient Queries**
   - Support field selection to request only needed data
   - Provide count operations (`zg card count --parent-id 123`)
   - Offer existence checks (`zg card exists <id>`)

4. **Batch Operations** (Future)
   - `zg card get --ids 1,2,3` - Get multiple cards in one request
   - Accept JSON input via stdin for bulk creates/updates

## Error Handling

### Error Output Format
```json
{
  "error": {
    "code": "NOT_FOUND",
    "message": "Card with ID 123 not found",
    "details": {
      "card_id": "123"
    }
  }
}
```

### Exit Codes
- `0` - Success
- `1` - General error
- `2` - Not found
- `3` - Authentication error
- `4` - Permission denied
- `5` - Invalid input
- `6` - Network error

## Security Considerations

1. **Credential Storage**
   - File permissions: 0600 for credentials file
   - Never log tokens
   - Warn if config file has insecure permissions

2. **HTTPS**
   - Require HTTPS by default
   - Allow `--insecure` flag for local development only

3. **Token Exposure**
   - Recommend using config file over environment variables
   - Clear documentation on token security

## Future Expansions

### Phase 2: Tasks
- `zg task` subcommand with CRUD operations
- Today's tasks listing
- Task completion operations

### Phase 3: Search & Starred
- Advanced search operations
- Starred search management
- Vector search with embeddings

### Phase 4: Files
- File upload/download
- File attachment to cards

### Phase 5: Templates & Entities
- Template operations (PRO)
- Entity management (PRO)

## Design Decisions

1. **Streaming for large operations** ✓ APPROVED
   - List operations will support NDJSON streaming via `--stream` flag
   - Enables LLMs to process large datasets incrementally
   - Reduces memory footprint and allows early stopping

2. **Interactive mode** ✓ DECIDED
   - Pure flag-based interface, no interactive prompts
   - Ensures scriptability and programmatic use
   - All parameters must be provided via flags or environment variables

3. **Caching** ✗ DEFERRED
   - No local caching in initial implementation
   - May revisit based on performance needs
   - Prioritize simplicity for MVP

4. **Watch mode** ✗ DEFERRED
   - No `zg card watch` functionality in MVP
   - May add in future based on agent workflow needs

5. **Offline mode** ✗ NOT PLANNED
   - CLI requires network connectivity to backend
   - No offline operations support

## Success Metrics

1. **Token Efficiency**: Measure average tokens used per operation vs. alternative approaches (API direct, web UI)
2. **Reliability**: Exit code correctness, error handling coverage
3. **Performance**: Response time for common operations
4. **Usability for Agents**: Successful integration with LLM agent frameworks

## Next Steps

1. Set up Go module and basic project structure
2. Implement authentication flow (login, token storage)
3. Implement core card operations (get, list, create)
4. Add output formatting and field selection
5. Testing with real backend
6. Documentation for LLM agent integration
