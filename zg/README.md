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
```

Or create manually:
```json
{
  "api_url": "http://localhost:8080",
  "token": "zg_live_..."
}
```

The config file is written with `0600` permissions so the token is not
world-readable.

**For local testing:** You can also use `./config.json` in the zg directory and run:
```bash
./zg --config ./config.json card list
```

### Authentication (durable API keys)

The web UI session tokens are short-lived JWTs that expire after **15 days**,
so CLI auth would silently break. Use a durable API key instead (Settings →
API Keys in the app); API keys never expire and work with the same `Bearer`
header.

**Interactive setup (recommended):**
```bash
zg auth login          # prompts for email/password, mints + stores an API key
```

**Manual setup:** create an API key in the web UI (Settings → API Keys), then:
```bash
zg auth set zg_live_...    # store the key created in the web UI
```

**Inspect / revoke:**
```bash
zg auth status             # shows token source + warns if a JWT is configured
zg auth keys               # lists the API keys on the account (metadata only)
zg auth revoke             # revokes the stored API key and clears local storage
```

**Token precedence** (highest first): `--token` flag > `ZETTELGARDEN_TOKEN`
env var > OS keyring (libsecret/Keychain/Credential Manager) > config file.
The API URL resolves the same way: `--url` flag > `ZETTELGARDEN_API_URL` env
var > `api_url` in the config file. Tokens are stored in the OS keyring when
available, otherwise in the config file (0600). Set `ZETTELGARDEN_NO_KEYRING=1`
to skip the keyring entirely (e.g. headless CI). If a short-lived JWT is
configured in place of an API key, zg prints a warning before it expires.

Note: a token passed via `--token` (or `ZETTELGARDEN_TOKEN`) is not managed by
`zg auth revoke`, which only clears the keyring/config copy — and a `--token`
flag value is visible in `ps`/shell history. Prefer the keyring/config storage
so the key stays revocable and out of process listings.

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
zg card next-id               # Get the next root card ID for a new top-level card
zg card next-child-id <id>    # Get the next child card ID under a parent card
zg card summaries <id>        # List AI summaries for a card (--latest for the most recent)
zg card star <id>             # Star a card
zg card unstar <id>           # Unstar a card
zg card children <id>         # List child cards of a card
zg card unsorted              # List cards with no card id (--limit/--offset)
zg card suggest-title "body"  # Ask the AI to suggest a title from a body
```

**Context-friendly defaults:** By default, `list` and `search` truncate body/preview content to 300 characters to avoid polluting LLM context. Use `--full` to get complete content:

```bash
zg card list --full           # Show full preview content
zg card search "query" --full # Show full preview content
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

### Articles

Create a card from a web page URL — the content is extracted, converted to
markdown, and stored as a new card tagged `#to-read #reference` by default:

```bash
zg article create "https://example.com/post"   # Create an article card from a URL
zg article create "https://example.com/post" -c 4.2   # With an explicit card ID
zg article create "https://example.com/post" -t "#read-later"  # Custom tags
zg parse-url "https://example.com/post"       # Fetch a URL as markdown (feeds article create)
```

### Tags

```bash
zg tag list                  # List all tags
zg tag create <name>         # Create a tag (--color, default black)
zg tag delete <id>           # Delete a tag by ID
zg tag cards <card-id>       # List tags on a card
zg tag add <card-id> <tag>   # Tag a card (appends #tag to its body)
```

Tags are derived from `#hashtags` in card bodies, so `zg tag add` fetches the
card, appends the hashtag if missing, and saves it back.

### Schemas

```bash
zg schema list               # List structured-data schemas (with card counts)
zg schema get <id>           # Get a schema by ID
```

Use `zg schema list` to find a schema id for
`zg card set-structured-data <id> -s <schema-id>`.

### Stats

```bash
zg stats                     # Daily card/task activity for the last 30 days (--days N)
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
