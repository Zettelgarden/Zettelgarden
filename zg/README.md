# zg - Zettelgarden CLI

A standalone CLI tool for Zettelgarden card and task operations.

## Installation

```bash
go build -o zg ./cmd/zg
cp zg /usr/local/bin/
```

## Configuration

Create `~/.config/zettelgarden/config.json`:

```bash
mkdir -p ~/.config/zettelgarden
cp config.example.json ~/.config/zettelgarden/config.json
# Edit ~/.config/zettelgarden/config.json with your JWT token
```

Or create manually:
```json
{
  "api_url": "http://localhost:8080",
  "token": "your-jwt-token-here"
}
```

Get your token from the Zettelgarden web UI.

**For local testing:** You can also use `./config.json` in the zg directory and run:
```bash
./zg --config ./config.json card list
```

## Usage

### Cards

```bash
zg card get <id>              # Get a card
zg card list [--limit N]      # List cards via search (preview truncated to 300 chars by default)
zg card list --starred        # List starred cards
zg card create -t "Title"     # Create card
zg card update <id> -t "New"  # Update card
zg card delete <id>           # Delete card
zg card search "query"        # Search cards (preview truncated by default)
```

**Context-friendly defaults:** By default, `list` and `search` truncate body/preview content to 300 characters to avoid polluting LLM context. Use `--full` to get complete content:

```bash
zg card list --full           # Show full preview content
zg card search "query" --full # Show full preview content
```

### Facts

```bash
zg fact get <id>              # Get a fact by ID
zg fact list [--search "q"]   # List facts (text truncated to 300 chars by default)
zg fact update <id> -t "..."  # Update fact text
zg fact delete <id>           # Delete a fact
zg fact similar <id>          # Find similar facts (uses embeddings)
zg fact link <fact-id> <card-id>  # Link a fact to a card
zg fact merge <id1> <id2>     # Merge fact2 into fact1 (fact2 is deleted)
```

**Context-friendly defaults:** By default, `list` and `similar` truncate fact text to 300 characters. Use `--full` to get complete content:

```bash
zg fact list --full           # Show full fact text
zg fact similar 42 --full     # Show full text for similar facts
```

Examples:
```bash
# List all facts
zg fact list

# Search for facts containing "project"
zg fact list --search "project"

# Update a fact's text
zg fact update 42 -t "The new fact text goes here"

# Delete a fact
zg fact delete 42

# Find facts similar to fact #42
zg fact similar 42 --limit 5

# Link fact #10 to card #5
zg fact link 10 5

# Merge fact #20 into fact #10 (fact #20 is deleted)
zg fact merge 10 20
```

### Structured Data

```bash
zg card get-structured-data <id>                           # Get structured data for a card
zg card set-structured-data <id> -s <schema-id> -d '{...}' # Set (replace) structured data
zg card patch-structured-data <id> -d '{...}'              # Patch (merge) into existing data
zg card clear-structured-data <id>                         # Clear structured data
```

Examples:
```bash
# Get structured data
zg card get-structured-data 42

# Set structured data with schema
zg card set-structured-data 42 -s 1 -d '{"title":"My Item","count":5}'

# Patch (merge) new values
zg card patch-structured-data 42 -d '{"count":10}'

# Clear structured data
zg card clear-structured-data 42
```

### Tasks

```bash
zg task get <id>              # Get a task
zg task list [--completed|--incomplete]  # List tasks (filter by completion state)
zg task create -t "Title"     # Create task
zg task update <id> --complete # Update task
zg task complete <id>         # Mark complete
zg task delete <id>           # Delete task
```

### Templates

```bash
zg template list              # List all templates
zg template get <id>          # Get a template by ID
```

### Global Flags

- `--pretty` - Pretty-print JSON output
- `--config <path>` - Custom config path
- `--url <url>` - Override API URL
- `--token <token>` - Override auth token

## Output Format

All commands return JSON:

```json
{
  "success": true,
  "data": { /* result */ }
}
```

Errors:
```json
{
  "success": false,
  "error": "Error message"
}
```
