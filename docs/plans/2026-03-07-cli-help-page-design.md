# CLI Help Page Design

Date: 2026-03-07

## Overview
Add a "CLI Tool" section to the existing HelpPage to documenting the `zg` CLI tool. This provides a quick reference for CLI commands within the existing help system.

## Design Decisions
- **Location**: Add section to existing `HelpPage.tsx` under "Advanced" category
- **Detail level**: Quick reference (per user preference)
- **Format**: Use existing HelpSection components for consistency

## Content Structure

### Section 1: Overview
- Brief intro to `zg` CLI
- Why use CLI (automation, scripting, integration)
- Link to config setup

### Section 2: Configuration
- Config file location: `~/.config/zettelgarden/config.json`
- Required fields: `api_url`, `token`
- Example config JSON
### Section 3: Command Reference Table
| Command | Description | Flags |
|--------|-------------|-------|
| Card commands | | |
| Task commands | | |
| Template commands | | |
### Section 4: Tips
- Using `--pretty` for readable output
- Common flag patterns
- Getting help with `zg --help`
## Implementation
- Add content to `helpContent.ts`
- No backend changes needed
- No new components needed
