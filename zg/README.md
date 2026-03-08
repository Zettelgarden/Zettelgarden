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
zg card list [--limit N]      # List cards
zg card create -t "Title"     # Create card
zg card update <id> -t "New"  # Update card
zg card delete <id>           # Delete card
zg card search "query"        # Search cards
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
zg task list [--completed]    # List tasks
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
