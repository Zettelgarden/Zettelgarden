#!/usr/bin/env python3
"""
Zettelgarden MCP Server

Provides Claude with access to Zettelgarden cards, tasks, and search.

Configuration:
    Set environment variables:
    - ZETTELGARDEN_API_URL: Base URL (default: http://localhost:8080)
    - ZETTELGARDEN_TOKEN: JWT auth token (required)

Usage:
    python server.py
"""

import os
import json
from datetime import datetime
from typing import Optional
import httpx
from mcp.server import Server
from mcp.server.stdio import stdio_server
from mcp.types import Tool, TextContent

# Configuration
API_URL = os.environ.get("ZETTELGARDEN_API_URL", "http://localhost:8080")
TOKEN = os.environ.get("ZETTELGARDEN_TOKEN", "")

server = Server("zettelgarden")


def get_headers() -> dict:
    """Get auth headers for API requests."""
    return {"Authorization": f"Bearer {TOKEN}"}


def format_card(card: dict) -> str:
    """Format a card for display."""
    lines = [
        f"**{card.get('card_id', 'N/A')}**: {card.get('title', 'Untitled')}",
        f"  ID: {card.get('id')} | Created: {card.get('created_at', 'N/A')[:10] if card.get('created_at') else 'N/A'}",
    ]
    if card.get("is_starred"):
        lines[0] += " ⭐"
    if card.get("body"):
        body_preview = card["body"][:200] + "..." if len(card["body"]) > 200 else card["body"]
        lines.append(f"  {body_preview}")
    if card.get("tags"):
        tags = ", ".join(f"#{t['name']}" for t in (card["tags"] or []))
        lines.append(f"  Tags: {tags}")
    return "\n".join(lines)


def format_task(task: dict) -> str:
    """Format a task for display."""
    status = "✓" if task.get("is_complete") else "○"
    priority = f"[{task.get('priority')}]" if task.get("priority") else ""
    scheduled = f"📅 {task.get('scheduled_date')[:10]}" if task.get("scheduled_date") else ""
    card_ref = f"(Card: {task['card']['card_id']})" if task.get("card") else ""

    lines = [
        f"{status} {priority} {task.get('title', 'Untitled')} {scheduled} {card_ref}".strip(),
        f"   ID: {task.get('id')} | Status: {task.get('status', 'N/A')}",
    ]
    if task.get("description"):
        desc_preview = task["description"][:100] + "..." if len(task["description"]) > 100 else task["description"]
        lines.append(f"   {desc_preview}")
    if task.get("tags"):
        tags = ", ".join(f"#{t['name']}" for t in (task["tags"] or []))
        lines.append(f"   Tags: {tags}")
    if task.get("blocked_by"):
        incomplete_blockers = [bt for bt in task["blocked_by"] if not bt.get("is_complete")]
        if incomplete_blockers:
            blocker_titles = ", ".join(bt.get("title", "Untitled") for bt in incomplete_blockers)
            lines.append(f"   🚧 Blocked by: {blocker_titles}")
    return "\n".join(lines)


def format_template(template: dict) -> str:
    """Format a template for display."""
    lines = [
        f"**{template.get('name', 'Unnamed Template')}** (ID: {template.get('id', 'N/A')})",
        f"Created: {template.get('created_at', 'N/A')[:10] if template.get('created_at') else 'N/A'}"
    ]

    if template.get("title"):
        lines.append(f"Title Template: {template.get('title')}")

    lines.append("")
    lines.append("Body:")
    lines.append(template.get("body", "*No content*"))

    return "\n".join(lines)


def format_template_list(templates: list) -> str:
    """Format a list of templates for display."""
    if not templates:
        return "No templates found."

    lines = [f"Templates ({len(templates)}):"]
    lines.append("")

    for template in templates:
        name = template.get('name', 'Unnamed Template')
        id = template.get('id', 'N/A')
        title = template.get('title', 'No title')[:50] + "..." if template.get('title') and len(template.get('title', '')) > 50 else template.get('title', 'No title')
        created = template.get('created_at', 'N/A')[:10] if template.get('created_at') else 'N/A'
        lines.append(f"**{name}** (ID: {id})")
        lines.append(f"  Title: {title}")
        lines.append(f"  Created: {created}")
        lines.append("")

    return "\n".join(lines)


def format_schema(schema: dict) -> str:
    """Format a schema for display."""
    lines = [
        f"**{schema.get('name', 'Unnamed Schema')}** (ID: {schema.get('id', 'N/A')}, Slug: {schema.get('slug', 'N/A')})",
        f"Created: {schema.get('created_at', 'N/A')[:10] if schema.get('created_at') else 'N/A'} | Updated: {schema.get('updated_at', 'N/A')[:10] if schema.get('updated_at') else 'N/A'}"
    ]

    lines.append("")
    lines.append("Fields:")
    fields = schema.get('fields', [])
    if not fields:
        lines.append("  No fields defined")
    else:
        for field in fields:
            required = " required" if field.get('required') else ""
            options = f" [{', '.join(field.get('options', []))}]" if field.get('options') else ""
            lines.append(f"  - {field.get('name', 'unknown')} ({field.get('type', 'unknown')}){required}{options}")

    return "\n".join(lines)


def format_schema_list(schemas: list) -> str:
    """Format a list of schemas for display."""
    if not schemas:
        return "No schemas found."

    lines = [f"Schemas ({len(schemas)}):"]
    lines.append("")

    for schema in schemas:
        name = schema.get('name', 'Unnamed Schema')
        id = schema.get('id', 'N/A')
        slug = schema.get('slug', 'N/A')
        field_count = len(schema.get('fields', []))
        created = schema.get('created_at', 'N/A')[:10] if schema.get('created_at') else 'N/A'
        lines.append(f"**{name}** (ID: {id}, Slug: {slug})")
        lines.append(f"  Fields: {field_count} | Created: {created}")
        lines.append("")

    return "\n".join(lines)


# =============================================================================
# TOOLS
# =============================================================================

@server.list_tools()
async def list_tools() -> list[Tool]:
    """List available tools."""
    return [
        # Card tools
        Tool(
            name="search_cards",
            description="Search for cards by text query. Supports tag filters (#tag), entity filters (@[entity]), exclusions (!term), and full-text search.",
            inputSchema={
                "type": "object",
                "properties": {
                    "query": {
                        "type": "string",
                        "description": "Search query. Examples: 'python', '#project', '@[John Doe]', 'python #learning'"
                    },
                    "full_text": {
                        "type": "boolean",
                        "description": "Search card body in addition to title (default: false)",
                        "default": False
                    },
                    "limit": {
                        "type": "integer",
                        "description": "Max results to return (default: 20)",
                        "default": 20
                    }
                },
                "required": ["query"]
            }
        ),
        Tool(
            name="get_card",
            description="Get a specific card by its numeric ID (pk) or card_id string. Returns full card with body, children, references, tasks, and entities.",
            inputSchema={
                "type": "object",
                "properties": {
                    "card_id": {
                        "type": ["integer", "string"],
                        "description": "Card ID - either numeric pk (e.g., 123) or card_id string (e.g., '1a2')"
                    }
                },
                "required": ["card_id"]
            }
        ),
        Tool(
            name="create_card",
            description="Create a new card in Zettelgarden. Optionally associate with a schema and provide structured data.",
            inputSchema={
                "type": "object",
                "properties": {
                    "title": {
                        "type": "string",
                        "description": "Card title"
                    },
                    "body": {
                        "type": "string",
                        "description": "Card body content (markdown supported)"
                    },
                    "card_id": {
                        "type": "string",
                        "description": "Optional card_id (e.g., '1a' for child of card 1). Leave empty for auto-generated root ID."
                    },
                    "link": {
                        "type": "string",
                        "description": "Optional URL link"
                    },
                    "schema_id": {
                        "type": "integer",
                        "description": "Optional schema ID to associate with this card"
                    },
                    "structured_data": {
                        "type": "object",
                        "description": "Optional structured data as key-value pairs matching the schema's field definitions"
                    }
                },
                "required": ["title", "body"]
            }
        ),
        Tool(
            name="update_card",
            description="Update an existing card's title, body, link, schema, or structured data. Set schema_id to null to remove schema association.",
            inputSchema={
                "type": "object",
                "properties": {
                    "id": {
                        "type": "integer",
                        "description": "Card numeric ID (pk)"
                    },
                    "title": {
                        "type": "string",
                        "description": "New title (optional)"
                    },
                    "body": {
                        "type": "string",
                        "description": "New body content (optional)"
                    },
                    "link": {
                        "type": "string",
                        "description": "New link (optional)"
                    },
                    "schema_id": {
                        "type": ["integer", "null"],
                        "description": "New schema ID (optional). Set to null to remove schema association."
                    },
                    "structured_data": {
                        "type": "object",
                        "description": "Structured data as key-value pairs (optional). Must match schema's field definitions."
                    }
                },
                "required": ["id"]
            }
        ),
        Tool(
            name="list_starred_cards",
            description="Get all starred/pinned cards.",
            inputSchema={
                "type": "object",
                "properties": {}
            }
        ),
        Tool(
            name="get_card_children",
            description="Get all direct children of a card.",
            inputSchema={
                "type": "object",
                "properties": {
                    "card_id": {
                        "type": "integer",
                        "description": "Parent card numeric ID (pk)"
                    }
                },
                "required": ["card_id"]
            }
        ),
        Tool(
            name="get_next_child_id",
            description="Get the next available child card ID for a parent card. Takes the parent card's numeric ID and returns the next ID like '1a2.3'",
            inputSchema={
                "type": "object",
                "properties": {
                    "card_pk": {
                        "type": "integer",
                        "description": "Parent card's numeric primary key"
                    }
                },
                "required": ["card_pk"]
            }
        ),
        # Template tools
        Tool(
            name="list_templates",
            description="Get all templates for the current user.",
            inputSchema={
                "type": "object",
                "properties": {}
            }
        ),
        Tool(
            name="get_template",
            description="Get a specific template by its numeric ID.",
            inputSchema={
                "type": "object",
                "properties": {
                    "template_id": {
                        "type": "integer",
                        "description": "Template ID"
                    }
                },
                "required": ["template_id"]
            }
        ),
        # Schema tools
        Tool(
            name="list_schemas",
            description="List all schemas for the current user.",
            inputSchema={
                "type": "object",
                "properties": {}
            }
        ),
        Tool(
            name="get_schema",
            description="Get a specific schema by its numeric ID or slug. Returns full schema with field definitions.",
            inputSchema={
                "type": "object",
                "properties": {
                    "schema_ref": {
                        "type": ["integer", "string"],
                        "description": "Schema ID (numeric) or slug (string). Example: 123 or 'book-review'"
                    }
                },
                "required": ["schema_ref"]
            }
        ),
        Tool(
            name="create_schema",
            description="Create a new schema with custom fields. Schemas define structured data types for cards.",
            inputSchema={
                "type": "object",
                "properties": {
                    "name": {
                        "type": "string",
                        "description": "Schema name (e.g., 'Book Review', 'Project Tracker')"
                    },
                    "fields": {
                        "type": "array",
                        "description": "Array of field definitions. Each field must have: name, type (text|number|date|boolean|select|multi-select|link_to_card), required (bool), and options (for select/multi-select types)",
                        "items": {
                            "type": "object",
                            "properties": {
                                "name": {"type": "string", "description": "Field name"},
                                "type": {"type": "string", "enum": ["text", "number", "date", "boolean", "select", "multi-select", "link_to_card"], "description": "Field type"},
                                "required": {"type": "boolean", "description": "Whether the field is required"},
                                "options": {"type": "array", "items": {"type": "string"}, "description": "Options for select/multi-select types"}
                            },
                            "required": ["name", "type"]
                        }
                    }
                },
                "required": ["name", "fields"]
            }
        ),
        Tool(
            name="update_schema",
            description="Update an existing schema's name and/or fields.",
            inputSchema={
                "type": "object",
                "properties": {
                    "schema_id": {
                        "type": "integer",
                        "description": "Schema numeric ID"
                    },
                    "name": {
                        "type": "string",
                        "description": "New schema name"
                    },
                    "fields": {
                        "type": "array",
                        "description": "Array of field definitions (same format as create_schema)",
                        "items": {
                            "type": "object",
                            "properties": {
                                "name": {"type": "string"},
                                "type": {"type": "string", "enum": ["text", "number", "date", "boolean", "select", "multi-select", "link_to_card"]},
                                "required": {"type": "boolean"},
                                "options": {"type": "array", "items": {"type": "string"}}
                            },
                            "required": ["name", "type"]
                        }
                    }
                },
                "required": ["schema_id"]
            }
        ),
        Tool(
            name="delete_schema",
            description="Delete a schema. This is a soft delete - cards using this schema will not be deleted but will lose their structured data display.",
            inputSchema={
                "type": "object",
                "properties": {
                    "schema_id": {
                        "type": "integer",
                        "description": "Schema numeric ID to delete"
                    }
                },
                "required": ["schema_id"]
            }
        ),
        Tool(
            name="get_schema_cards",
            description="Get all cards that use a specific schema. Returns cards with their structured data.",
            inputSchema={
                "type": "object",
                "properties": {
                    "schema_ref": {
                        "type": ["integer", "string"],
                        "description": "Schema ID (numeric) or slug (string). Example: 123 or 'book-review'"
                    }
                },
                "required": ["schema_ref"]
            }
        ),
        # Task tools
        Tool(
            name="list_tasks",
            description="List tasks with optional filters. Use to see today's tasks, incomplete tasks, or filter by priority/status.",
            inputSchema={
                "type": "object",
                "properties": {
                    "completed": {
                        "type": "boolean",
                        "description": "Filter by completion status. true=completed only, false=incomplete only, omit for all"
                    },
                    "scheduled_date": {
                        "type": "string",
                        "description": "Filter by scheduled date (YYYY-MM-DD format). Use 'today' for today's date."
                    },
                    "priority": {
                        "type": "string",
                        "description": "Filter by priority (e.g., 'high', 'medium', 'low')"
                    },
                    "status": {
                        "type": "string",
                        "description": "Filter by status"
                    },
                    "card_id": {
                        "type": "integer",
                        "description": "Filter by associated card ID"
                    },
                    "limit": {
                        "type": "integer",
                        "description": "Max results (default: 50)",
                        "default": 50
                    }
                }
            }
        ),
        Tool(
            name="get_task",
            description="Get a specific task by ID.",
            inputSchema={
                "type": "object",
                "properties": {
                    "task_id": {
                        "type": "integer",
                        "description": "Task ID"
                    }
                },
                "required": ["task_id"]
            }
        ),
        Tool(
            name="create_task",
            description="Create a new task.",
            inputSchema={
                "type": "object",
                "properties": {
                    "title": {
                        "type": "string",
                        "description": "Task title"
                    },
                    "description": {
                        "type": "string",
                        "description": "Task description (optional)"
                    },
                    "scheduled_date": {
                        "type": "string",
                        "description": "Scheduled date (YYYY-MM-DD). Use 'today' for today."
                    },
                    "priority": {
                        "type": "string",
                        "description": "Priority level (e.g., 'high', 'medium', 'low')"
                    },
                    "card_pk": {
                        "type": "integer",
                        "description": "Optional: Associate with a card by its numeric ID"
                    }
                },
                "required": ["title"]
            }
        ),
        Tool(
            name="update_task",
            description="Update a task's title, description, status, priority, scheduled date, or completion.",
            inputSchema={
                "type": "object",
                "properties": {
                    "task_id": {
                        "type": "integer",
                        "description": "Task ID"
                    },
                    "title": {
                        "type": "string",
                        "description": "New title"
                    },
                    "description": {
                        "type": "string",
                        "description": "New description"
                    },
                    "is_complete": {
                        "type": "boolean",
                        "description": "Mark as complete/incomplete"
                    },
                    "priority": {
                        "type": "string",
                        "description": "New priority"
                    },
                    "scheduled_date": {
                        "type": "string",
                        "description": "New scheduled date (YYYY-MM-DD or 'today')"
                    },
                    "status": {
                        "type": "string",
                        "description": "New status"
                    }
                },
                "required": ["task_id"]
            }
        ),
        Tool(
            name="complete_task",
            description="Mark a task as complete.",
            inputSchema={
                "type": "object",
                "properties": {
                    "task_id": {
                        "type": "integer",
                        "description": "Task ID to complete"
                    }
                },
                "required": ["task_id"]
            }
        ),
        Tool(
            name="parse_url",
            description="Parse a URL to extract article content (title, body, author, excerpt). Returns the parsed content for preview before creating a card.",
            inputSchema={
                "type": "object",
                "properties": {
                    "url": {
                        "type": "string",
                        "description": "URL to parse and extract content from"
                    }
                },
                "required": ["url"]
            }
        ),
        Tool(
            name="create_article",
            description="Create a new article card from a URL. Automatically parses the URL, extracts content, adds the link, and tags with #to-read #reference.",
            inputSchema={
                "type": "object",
                "properties": {
                    "url": {
                        "type": "string",
                        "description": "URL of the article to import"
                    },
                    "card_id": {
                        "type": "string",
                        "description": "Optional card_id (e.g., '1a'). Leave empty for auto-generated root ID."
                    },
                    "tags": {
                        "type": "string",
                        "description": "Optional custom tags (default: '#to-read #reference'). Provide as space-separated tags like '#tag1 #tag2'"
                    }
                },
                "required": ["url"]
            }
        ),
    ]


@server.call_tool()
async def call_tool(name: str, arguments: dict) -> list[TextContent]:
    """Handle tool calls."""

    async with httpx.AsyncClient(timeout=30.0) as client:
        try:
            result = await _handle_tool(client, name, arguments)
            return [TextContent(type="text", text=result)]
        except httpx.HTTPStatusError as e:
            return [TextContent(type="text", text=f"API Error {e.response.status_code}: {e.response.text}")]
        except Exception as e:
            return [TextContent(type="text", text=f"Error: {str(e)}")]


async def _handle_tool(client: httpx.AsyncClient, name: str, args: dict) -> str:
    """Route tool calls to handlers."""

    # Card tools
    if name == "search_cards":
        return await search_cards(client, args)
    elif name == "get_card":
        return await get_card(client, args)
    elif name == "create_card":
        return await create_card(client, args)
    elif name == "update_card":
        return await update_card(client, args)
    elif name == "list_starred_cards":
        return await list_starred_cards(client)
    elif name == "get_card_children":
        return await get_card_children(client, args)
    elif name == "get_next_child_id":
        return await get_next_child_id(client, args)

    # Template tools
    elif name == "list_templates":
        return await list_templates(client)
    elif name == "get_template":
        return await get_template(client, args)

    # Schema tools
    elif name == "list_schemas":
        return await list_schemas(client)
    elif name == "get_schema":
        return await get_schema(client, args)
    elif name == "create_schema":
        return await create_schema(client, args)
    elif name == "update_schema":
        return await update_schema(client, args)
    elif name == "delete_schema":
        return await delete_schema(client, args)
    elif name == "get_schema_cards":
        return await get_schema_cards(client, args)

    # Task tools
    elif name == "list_tasks":
        return await list_tasks(client, args)
    elif name == "get_task":
        return await get_task(client, args)
    elif name == "create_task":
        return await create_task(client, args)
    elif name == "update_task":
        return await update_task(client, args)
    elif name == "complete_task":
        return await complete_task(client, args)

    # Article tools
    elif name == "parse_url":
        return await parse_url(client, args)
    elif name == "create_article":
        return await create_article(client, args)

    else:
        return f"Unknown tool: {name}"


# =============================================================================
# CARD HANDLERS
# =============================================================================

async def search_cards(client: httpx.AsyncClient, args: dict) -> str:
    """Search for cards."""
    query = args.get("query", "")
    full_text = args.get("full_text", False)
    limit = args.get("limit", 20)

    resp = await client.post(
        f"{API_URL}/api/search",
        headers=get_headers(),
        json={
            "search_term": query,
            "full_text": full_text,
            "show_cards": True,
            "show_entities": False,
            "show_facts": False,
            "per_page": limit,
            "page": 1
        }
    )
    resp.raise_for_status()
    data = resp.json()

    results = data.get("results", [])
    total = data.get("total", 0)

    if not results:
        return f"No cards found for query: '{query}'"

    lines = [f"Found {total} cards for '{query}' (showing {len(results)}):"]
    lines.append("")

    for r in results:
        card_id = r.get("metadata", {}).get("card_id", "N/A")
        title = r.get("title", "Untitled")
        preview = r.get("preview", "")[:150]
        tags = ", ".join(f"#{t['name']}" for t in (r.get("tags") or []))

        lines.append(f"**{card_id}**: {title}")
        lines.append(f"  pk={r.get('metadata', {}).get('id', 'N/A')} | {tags}")
        if preview:
            lines.append(f"  {preview}...")
        lines.append("")

    return "\n".join(lines)


async def get_card(client: httpx.AsyncClient, args: dict) -> str:
    """Get a specific card."""
    card_id = args.get("card_id")

    # If it's a string card_id, search for it first
    if isinstance(card_id, str) and not card_id.isdigit():
        # Search by card_id
        resp = await client.post(
            f"{API_URL}/api/search",
            headers=get_headers(),
            json={"search_term": card_id, "show_cards": True, "per_page": 10}
        )
        resp.raise_for_status()
        results = resp.json().get("results", [])

        # Find exact match
        for r in results:
            if r.get("metadata", {}).get("card_id") == card_id:
                card_id = r.get("metadata", {}).get("id")
                break
        else:
            return f"Card with card_id '{args.get('card_id')}' not found"

    resp = await client.get(
        f"{API_URL}/api/cards/{card_id}",
        headers=get_headers()
    )
    resp.raise_for_status()
    card = resp.json()

    lines = [
        f"# {card.get('card_id', 'N/A')}: {card.get('title', 'Untitled')}",
        f"",
        f"**ID:** {card.get('id')} | **Created:** {card.get('created_at', '')[:10]} | **Updated:** {card.get('updated_at', '')[:10]}",
    ]

    if card.get("is_starred"):
        lines.append("**Starred:** Yes ⭐")

    # Schema information
    schema_id = card.get("schema_id")
    if schema_id is not None:
        lines.append(f"**Schema ID:** {schema_id}")

    if card.get("parent"):
        p = card["parent"]
        lines.append(f"**Parent:** {p.get('card_id', 'N/A')} - {p.get('title', '')}")

    if card.get("tags"):
        tags = ", ".join(f"#{t['name']}" for t in (card["tags"] or []))
        lines.append(f"**Tags:** {tags}")

    if card.get("link"):
        lines.append(f"**Link:** {card['link']}")

    # Structured data
    structured_data = card.get("structured_data")
    if structured_data:
        lines.append("")
        lines.append("## Structured Data")
        for key, value in structured_data.items():
            if isinstance(value, list):
                lines.append(f"  {key}: {', '.join(str(v) for v in value)}")
            else:
                lines.append(f"  {key}: {value}")

    lines.append("")
    lines.append("## Body")
    lines.append(card.get("body", "*No content*"))

    # Children
    children = card.get("children", [])
    if children:
        lines.append("")
        lines.append(f"## Children ({len(children)})")
        for c in children[:10]:
            lines.append(f"- **{c.get('card_id', 'N/A')}**: {c.get('title', '')}")
        if len(children) > 10:
            lines.append(f"  ... and {len(children) - 10} more")

    # References
    refs = card.get("references", [])
    if refs:
        lines.append("")
        lines.append(f"## References ({len(refs)})")
        for r in refs[:10]:
            lines.append(f"- **{r.get('card_id', 'N/A')}**: {r.get('title', '')}")

    # Tasks
    tasks = card.get("tasks", [])
    if tasks:
        lines.append("")
        lines.append(f"## Tasks ({len(tasks)})")
        for t in tasks[:5]:
            status = "✓" if t.get("is_complete") else "○"
            lines.append(f"- {status} {t.get('title', '')}")

    # Entities
    entities = card.get("entities", [])
    if entities:
        lines.append("")
        lines.append(f"## Entities ({len(entities)})")
        for e in entities[:10]:
            lines.append(f"- {e.get('name', '')} ({e.get('type', '')})")

    return "\n".join(lines)


async def create_card(client: httpx.AsyncClient, args: dict) -> str:
    """Create a new card."""
    card_id = args.get("card_id", "")

    # If no card_id provided, get next root ID
    if not card_id:
        resp = await client.get(
            f"{API_URL}/api/cards/next-root-id",
            headers=get_headers()
        )
        resp.raise_for_status()
        card_id = resp.json().get("new_id", "")

    # Build request payload
    payload = {
        "card_id": card_id,
        "title": args.get("title", ""),
        "body": args.get("body", ""),
        "link": args.get("link", "")
    }

    # Add schema_id and structured_data if provided
    if "schema_id" in args:
        payload["schema_id"] = args["schema_id"]
    if "structured_data" in args:
        payload["structured_data"] = args["structured_data"]

    resp = await client.post(
        f"{API_URL}/api/cards",
        headers=get_headers(),
        json=payload
    )
    resp.raise_for_status()
    card = resp.json()

    result = f"Created card: **{card.get('card_id', 'N/A')}**: {card.get('title', '')} (pk={card.get('id')})"
    if card.get("schema_id"):
        result += f"\nSchema ID: {card.get('schema_id')}"
    if card.get("structured_data"):
        result += f"\nStructured data: {card.get('structured_data')}"
    return result


async def update_card(client: httpx.AsyncClient, args: dict) -> str:
    """Update an existing card."""
    card_id = args.get("id")

    # First get the current card
    resp = await client.get(
        f"{API_URL}/api/cards/{card_id}",
        headers=get_headers()
    )
    resp.raise_for_status()
    card = resp.json()

    # Build update payload with existing values as defaults
    update_data = {
        "card_id": card.get("card_id", ""),
        "title": args.get("title", card.get("title", "")),
        "body": args.get("body", card.get("body", "")),
        "link": args.get("link", card.get("link", ""))
    }

    # Add schema_id if provided (including null to remove schema)
    if "schema_id" in args:
        update_data["schema_id"] = args["schema_id"]

    # Add structured_data if provided
    if "structured_data" in args:
        update_data["structured_data"] = args["structured_data"]

    resp = await client.put(
        f"{API_URL}/api/cards/{card_id}",
        headers=get_headers(),
        json=update_data
    )
    resp.raise_for_status()
    updated_card = resp.json()

    result = f"Updated card: **{card.get('card_id', 'N/A')}**: {update_data['title']} (pk={card_id})"
    if "schema_id" in args:
        result += f"\nSchema ID: {updated_card.get('schema_id')}"
    if "structured_data" in args:
        result += f"\nStructured data: {updated_card.get('structured_data')}"
    return result


async def list_starred_cards(client: httpx.AsyncClient) -> str:
    """Get all starred cards."""
    resp = await client.get(
        f"{API_URL}/api/cards/starred",
        headers=get_headers()
    )
    resp.raise_for_status()
    cards = resp.json()

    if not cards:
        return "No starred cards."

    lines = [f"Starred cards ({len(cards)}):"]
    lines.append("")

    for card in cards:
        lines.append(format_card(card))
        lines.append("")

    return "\n".join(lines)


async def get_card_children(client: httpx.AsyncClient, args: dict) -> str:
    """Get children of a card."""
    card_id = args.get("card_id")

    resp = await client.get(
        f"{API_URL}/api/cards/{card_id}/children",
        headers=get_headers()
    )
    resp.raise_for_status()
    children = resp.json()

    if not children:
        return f"Card {card_id} has no children."

    lines = [f"Children of card {card_id} ({len(children)}):"]
    lines.append("")

    for c in children:
        lines.append(f"- **{c.get('card_id', 'N/A')}**: {c.get('title', '')} (pk={c.get('id')})")

    return "\n".join(lines)


async def get_next_child_id(client: httpx.AsyncClient, args: dict) -> str:
    """Get the next available child ID for a parent card."""
    card_pk = args.get("card_pk")

    resp = await client.get(
        f"{API_URL}/api/cards/{card_pk}/next-child-id",
        headers=get_headers()
    )
    resp.raise_for_status()
    data = resp.json()

    if data.get("error", True):
        return f"Error getting next child ID for card {card_pk}"
    else:
        next_id = data.get("new_id", "unknown")
        return f"Next child ID for card {card_pk}: {next_id}"


# =============================================================================
# ARTICLE HANDLERS
# =============================================================================

async def parse_url(client: httpx.AsyncClient, args: dict) -> str:
    """Parse a URL to extract article content."""
    url = args.get("url", "")

    resp = await client.post(
        f"{API_URL}/api/url/parse",
        headers=get_headers(),
        json={"url": url}
    )
    resp.raise_for_status()
    result = resp.json()

    lines = [
        f"# Parsed Article from {url}",
        "",
        f"**Title:** {result.get('title', 'Untitled')}",
    ]

    if result.get("author"):
        lines.append(f"**Author:** {result.get('author')}")
    if result.get("site_name"):
        lines.append(f"**Site:** {result.get('site_name')}")
    if result.get("excerpt"):
        lines.append(f"**Excerpt:** {result.get('excerpt')[:200]}...")

    lines.append("")
    lines.append("## Content Preview")
    content = result.get("content", "")
    preview = content[:500] + "..." if len(content) > 500 else content
    lines.append(preview)

    return "\n".join(lines)


async def create_article(client: httpx.AsyncClient, args: dict) -> str:
    """Create a new article card from a URL."""
    url = args.get("url", "")
    card_id = args.get("card_id", "")
    custom_tags = args.get("tags", "")

    # Use the consolidated /api/articles endpoint
    article_resp = await client.post(
        f"{API_URL}/api/articles",
        headers=get_headers(),
        json={
            "url": url,
            "card_id": card_id,
            "tags": custom_tags,
        }
    )
    article_resp.raise_for_status()
    card = article_resp.json()

    lines = [
        f"Created article card: **{card.get('card_id', 'N/A')}**: {card.get('title', 'Untitled')}",
        f"pk={card.get('id')} | Source: {url}",
    ]

    return "\n".join(lines)


# =============================================================================
# TEMPLATE HANDLERS
# =============================================================================

async def list_templates(client: httpx.AsyncClient) -> str:
    """Get all templates for the current user."""
    resp = await client.get(
        f"{API_URL}/api/templates",
        headers=get_headers()
    )
    resp.raise_for_status()
    templates = resp.json()

    return format_template_list(templates)


async def get_template(client: httpx.AsyncClient, args: dict) -> str:
    """Get a specific template."""
    template_id = args.get("template_id")

    resp = await client.get(
        f"{API_URL}/api/templates/{template_id}",
        headers=get_headers()
    )
    resp.raise_for_status()
    template = resp.json()

    return format_template(template)


# =============================================================================
# SCHEMA HANDLERS
# =============================================================================

async def list_schemas(client: httpx.AsyncClient) -> str:
    """List all schemas for the current user."""
    resp = await client.get(
        f"{API_URL}/api/schemas",
        headers=get_headers()
    )
    resp.raise_for_status()
    schemas = resp.json()

    return format_schema_list(schemas)


async def get_schema(client: httpx.AsyncClient, args: dict) -> str:
    """Get a specific schema by ID or slug."""
    schema_ref = args.get("schema_ref")

    resp = await client.get(
        f"{API_URL}/api/schemas/{schema_ref}",
        headers=get_headers()
    )
    resp.raise_for_status()
    schema = resp.json()

    return format_schema(schema)


async def create_schema(client: httpx.AsyncClient, args: dict) -> str:
    """Create a new schema."""
    resp = await client.post(
        f"{API_URL}/api/schemas",
        headers=get_headers(),
        json={
            "name": args.get("name"),
            "fields": args.get("fields", [])
        }
    )
    resp.raise_for_status()
    schema = resp.json()

    lines = [
        f"Created schema: **{schema.get('name', 'Unnamed')}** (ID: {schema.get('id')}, Slug: {schema.get('slug')})",
        f"  Fields: {len(schema.get('fields', []))}"
    ]

    return "\n".join(lines)


async def update_schema(client: httpx.AsyncClient, args: dict) -> str:
    """Update an existing schema."""
    schema_id = args.get("schema_id")

    # First get the current schema
    resp = await client.get(
        f"{API_URL}/api/schemas/{schema_id}",
        headers=get_headers()
    )
    resp.raise_for_status()
    current = resp.json()

    # Build update payload
    update_data = {
        "id": schema_id,
        "name": args.get("name", current.get("name")),
        "fields": args.get("fields", current.get("fields"))
    }

    resp = await client.put(
        f"{API_URL}/api/schemas/{schema_id}",
        headers=get_headers(),
        json=update_data
    )
    resp.raise_for_status()
    schema = resp.json()

    return f"Updated schema: **{schema.get('name')}** (ID: {schema_id}, Slug: {schema.get('slug')})"


async def delete_schema(client: httpx.AsyncClient, args: dict) -> str:
    """Delete a schema (soft delete)."""
    schema_id = args.get("schema_id")

    resp = await client.delete(
        f"{API_URL}/api/schemas/{schema_id}",
        headers=get_headers()
    )
    resp.raise_for_status()
    result = resp.json()

    lines = [f"Deleted schema (ID: {schema_id})"]

    if result.get("warning"):
        lines.append("")
        lines.append(f"⚠️ {result.get('warning')}")
        lines.append(f"   Cards affected: {result.get('cards_affected', 0)}")

    return "\n".join(lines)


async def get_schema_cards(client: httpx.AsyncClient, args: dict) -> str:
    """Get all cards that use a specific schema."""
    schema_ref = args.get("schema_ref")

    resp = await client.get(
        f"{API_URL}/api/schemas/{schema_ref}/cards",
        headers=get_headers()
    )
    resp.raise_for_status()
    cards = resp.json()

    if not cards:
        return f"No cards found for schema: {schema_ref}"

    lines = [f"Cards using schema '{schema_ref}' ({len(cards)}):"]
    lines.append("")

    for card in cards:
        lines.append(f"**{card.get('card_id', 'N/A')}**: {card.get('title', 'Untitled')}")
        lines.append(f"  pk={card.get('id')} | Updated: {card.get('updated_at', '')[:10]}")

        # Show structured data if present
        structured_data = card.get('structured_data')
        if structured_data:
            lines.append("  Data:")
            for key, value in structured_data.items():
                if isinstance(value, list):
                    lines.append(f"    {key}: {', '.join(str(v) for v in value)}")
                else:
                    lines.append(f"    {key}: {value}")
        lines.append("")

    return "\n".join(lines)


# =============================================================================
# TASK HANDLERS
# =============================================================================

async def list_tasks(client: httpx.AsyncClient, args: dict) -> str:
    """List tasks with filters."""
    params = {"limit": args.get("limit", 50)}

    if "completed" in args:
        params["completed"] = "true" if args["completed"] else "false"

    scheduled = args.get("scheduled_date")
    if scheduled:
        if scheduled.lower() == "today":
            scheduled = datetime.now().strftime("%Y-%m-%d")
        params["scheduled_date"] = scheduled

    if args.get("priority"):
        params["priority"] = args["priority"]
    if args.get("status"):
        params["status"] = args["status"]
    if args.get("card_id"):
        params["card_id"] = args["card_id"]

    resp = await client.get(
        f"{API_URL}/api/tasks",
        headers=get_headers(),
        params=params
    )
    resp.raise_for_status()
    data = resp.json()

    tasks = data.get("tasks", [])
    total = data.get("total", 0)

    if not tasks:
        return "No tasks found matching criteria."

    lines = [f"Tasks ({len(tasks)} of {total}):"]
    lines.append("")

    for task in tasks:
        lines.append(format_task(task))
        lines.append("")

    return "\n".join(lines)


async def get_task(client: httpx.AsyncClient, args: dict) -> str:
    """Get a specific task."""
    task_id = args.get("task_id")

    resp = await client.get(
        f"{API_URL}/api/tasks/{task_id}",
        headers=get_headers()
    )
    resp.raise_for_status()
    task = resp.json()

    status = "✓ Complete" if task.get("is_complete") else "○ Incomplete"

    lines = [
        f"# Task: {task.get('title', 'Untitled')}",
        f"",
        f"**ID:** {task.get('id')} | **Status:** {status}",
        f"**Priority:** {task.get('priority', 'None')} | **Workflow Status:** {task.get('status', 'N/A')}",
    ]

    if task.get("description"):
        lines.append("")
        lines.append(f"**Description:** {task['description']}")

    if task.get("scheduled_date"):
        lines.append(f"**Scheduled:** {task['scheduled_date'][:10]}")
    if task.get("dueDate"):
        lines.append(f"**Due:** {task['dueDate'][:10]}")
    if task.get("completed_at"):
        lines.append(f"**Completed:** {task['completed_at'][:10]}")

    if task.get("card"):
        c = task["card"]
        lines.append(f"**Card:** {c.get('card_id', 'N/A')} - {c.get('title', '')} (pk={c.get('id')})")

    if task.get("tags"):
        tags = ", ".join(f"#{t['name']}" for t in (task["tags"] or []))
        lines.append(f"**Tags:** {tags}")

    # Dependencies
    if task.get("blocked_by"):
        lines.append("")
        lines.append(f"**Blocked By ({len(task['blocked_by'])} tasks):**")
        for bt in task["blocked_by"]:
            status_icon = "✓" if bt.get("is_complete") else "○"
            lines.append(f"  {status_icon} {bt.get('title', 'Untitled')} (id={bt.get('id')})")

    if task.get("blocks"):
        lines.append("")
        lines.append(f"**Blocking ({len(task['blocks'])} tasks):**")
        for bt in task["blocks"]:
            status_icon = "✓" if bt.get("is_complete") else "○"
            lines.append(f"  {status_icon} {bt.get('title', 'Untitled')} (id={bt.get('id')})")

    lines.append("")
    lines.append(f"**Created:** {task.get('created_at', '')[:10]} | **Updated:** {task.get('updated_at', '')[:10]}")

    return "\n".join(lines)


async def create_task(client: httpx.AsyncClient, args: dict) -> str:
    """Create a new task."""
    scheduled = args.get("scheduled_date")
    if scheduled and scheduled.lower() == "today":
        scheduled = datetime.now().strftime("%Y-%m-%d")

    task_data = {
        "title": args.get("title", ""),
        "description": args.get("description"),
        "priority": args.get("priority"),
        "card_pk": args.get("card_pk", 0),
        "is_complete": False
    }

    if scheduled:
        task_data["scheduled_date"] = f"{scheduled}T00:00:00Z"

    resp = await client.post(
        f"{API_URL}/api/tasks",
        headers=get_headers(),
        json=task_data
    )
    resp.raise_for_status()
    result = resp.json()

    return f"Created task: '{args.get('title')}' (id={result.get('id')})"


async def update_task(client: httpx.AsyncClient, args: dict) -> str:
    """Update a task."""
    task_id = args.get("task_id")

    # Get current task
    resp = await client.get(
        f"{API_URL}/api/tasks/{task_id}",
        headers=get_headers()
    )
    resp.raise_for_status()
    task = resp.json()

    # Build a clean update payload with only the fields the backend expects
    # Don't send back nested objects like 'card' and 'tags' which can cause parsing issues
    update_data = {
        "id": task.get("id"),
        "card_pk": task.get("card_pk", 0),
        "user_id": task.get("user_id"),
        "title": task.get("title", ""),
        "description": task.get("description"),
        "priority": task.get("priority"),
        "status": task.get("status", ""),
        "is_complete": task.get("is_complete", False),
        "scheduled_date": task.get("scheduled_date"),
        "due_date": task.get("due_date"),
        "reminder_time": task.get("reminder_time"),
    }

    # Apply updates from args
    if "title" in args:
        update_data["title"] = args["title"]
    if "description" in args:
        update_data["description"] = args["description"]
    if "is_complete" in args:
        update_data["is_complete"] = args["is_complete"]
    if "priority" in args:
        update_data["priority"] = args["priority"]
    if "status" in args:
        update_data["status"] = args["status"]
    if "scheduled_date" in args:
        scheduled = args["scheduled_date"]
        if scheduled.lower() == "today":
            scheduled = datetime.now().strftime("%Y-%m-%d")
        update_data["scheduled_date"] = f"{scheduled}T00:00:00Z"

    resp = await client.put(
        f"{API_URL}/api/tasks/{task_id}",
        headers=get_headers(),
        json=update_data
    )
    resp.raise_for_status()

    return f"Updated task: '{update_data.get('title')}' (id={task_id})"


async def complete_task(client: httpx.AsyncClient, args: dict) -> str:
    """Mark a task as complete."""
    args["is_complete"] = True
    args["status"] = "done"  # Set status to match is_complete state
    return await update_task(client, args)


# =============================================================================
# MAIN
# =============================================================================

async def run_server():
    """Run the MCP server."""
    if not TOKEN:
        import sys
        print("Error: ZETTELGARDEN_TOKEN environment variable is required", file=sys.stderr)
        print("Get your token from the Zettelgarden web UI after logging in", file=sys.stderr)
        sys.exit(1)

    async with stdio_server() as (read_stream, write_stream):
        await server.run(read_stream, write_stream, server.create_initialization_options())


async def test_cli(args):
    """CLI for testing the API connection."""
    import sys

    if not TOKEN:
        print("Error: ZETTELGARDEN_TOKEN environment variable is required")
        print("Export it first: export ZETTELGARDEN_TOKEN='your-token'")
        sys.exit(1)

    print(f"API URL: {API_URL}")
    print(f"Token: {TOKEN[:20]}...{TOKEN[-10:]}" if len(TOKEN) > 30 else f"Token: {TOKEN}")
    print()

    async with httpx.AsyncClient(timeout=30.0) as client:
        command = args[0] if args else "help"

        try:
            if command == "search":
                query = " ".join(args[1:]) if len(args) > 1 else ""
                if not query:
                    print("Usage: python server.py search <query>")
                    sys.exit(1)
                result = await search_cards(client, {"query": query, "limit": 10})
                print(result)

            elif command == "card":
                if len(args) < 2:
                    print("Usage: python server.py card <id>")
                    sys.exit(1)
                card_id = int(args[1]) if args[1].isdigit() else args[1]
                result = await get_card(client, {"card_id": card_id})
                print(result)

            elif command == "tasks":
                filters = {}
                if len(args) > 1:
                    if args[1] == "today":
                        filters["scheduled_date"] = "today"
                    elif args[1] == "incomplete":
                        filters["completed"] = False
                result = await list_tasks(client, filters)
                print(result)

            elif command == "task":
                if len(args) < 2:
                    print("Usage: python server.py task <id>")
                    sys.exit(1)
                result = await get_task(client, {"task_id": int(args[1])})
                print(result)

            elif command == "complete":
                if len(args) < 2:
                    print("Usage: python server.py complete <task_id>")
                    sys.exit(1)
                result = await complete_task(client, {"task_id": int(args[1])})
                print(result)

            elif command == "starred":
                result = await list_starred_cards(client)
                print(result)

            elif command == "create":
                # create <title> [body]
                if len(args) < 2:
                    print("Usage: python server.py create <title> [body]")
                    print("  Example: python server.py create 'My Note' 'Note content here'")
                    sys.exit(1)
                title = args[1]
                body = " ".join(args[2:]) if len(args) > 2 else ""
                result = await create_card(client, {"title": title, "body": body})
                print(result)

            elif command == "update":
                # update <id> --title "New title" --body "New body"
                if len(args) < 2:
                    print("Usage: python server.py update <id> [--title 'New title'] [--body 'New body']")
                    sys.exit(1)
                card_id = int(args[1])
                update_args = {"id": card_id}

                # Parse --title and --body flags
                i = 2
                while i < len(args):
                    if args[i] == "--title" and i + 1 < len(args):
                        update_args["title"] = args[i + 1]
                        i += 2
                    elif args[i] == "--body" and i + 1 < len(args):
                        update_args["body"] = args[i + 1]
                        i += 2
                    else:
                        i += 1

                if len(update_args) == 1:
                    print("Error: Must provide --title or --body to update")
                    sys.exit(1)

                result = await update_card(client, update_args)
                print(result)

            elif command == "ping":
                # Just test the connection
                resp = await client.get(f"{API_URL}/api/auth", headers=get_headers())
                if resp.status_code == 200:
                    print("✓ Connected successfully!")
                    print(f"  User: {resp.json()}")
                else:
                    print(f"✗ Connection failed: {resp.status_code}")
                    print(f"  {resp.text}")

            elif command == "parse-url":
                if len(args) < 2:
                    print("Usage: python server.py parse-url <url>")
                    sys.exit(1)
                url = args[1]
                result = await parse_url(client, {"url": url})
                print(result)

            elif command == "article":
                if len(args) < 2:
                    print("Usage: python server.py article <url> [--card-id 'id'] [--tags '#tag1 #tag2']")
                    sys.exit(1)
                url = args[1]
                article_args = {"url": url}

                # Parse optional flags
                i = 2
                while i < len(args):
                    if args[i] == "--card-id" and i + 1 < len(args):
                        article_args["card_id"] = args[i + 1]
                        i += 2
                    elif args[i] == "--tags" and i + 1 < len(args):
                        article_args["tags"] = args[i + 1]
                        i += 2
                    else:
                        i += 1

                result = await create_article(client, article_args)
                print(result)

            else:
                print("Zettelgarden MCP Server - Test CLI")
                print()
                print("Usage: python server.py <command> [args]")
                print()
                print("Commands:")
                print("  ping              Test API connection")
                print("  search <query>    Search for cards")
                print("  card <id>         Get a card by ID or card_id")
                print("  create <title> [body]")
                print("                    Create a new card")
                print("  update <id> --title 'New title' --body 'New body'")
                print("                    Update an existing card")
                print("  starred           List starred cards")
                print("  tasks             List all tasks")
                print("  tasks today       List today's tasks")
                print("  tasks incomplete  List incomplete tasks")
                print("  task <id>         Get a specific task")
                print("  complete <id>     Mark a task as complete")
                print("  parse-url <url>   Parse a URL to extract article content")
                print("  article <url>     Create an article card from a URL")
                print("                    [--card-id 'id'] [--tags '#tag1 #tag2']")
                print()
                print("Environment:")
                print("  ZETTELGARDEN_TOKEN     JWT auth token (required)")
                print("  ZETTELGARDEN_API_URL   API URL (default: http://localhost:8080)")
                print()
                print("To run as MCP server: python server.py --serve")

        except httpx.HTTPStatusError as e:
            print(f"API Error {e.response.status_code}: {e.response.text}")
            sys.exit(1)
        except Exception as e:
            print(f"Error: {e}")
            sys.exit(1)


if __name__ == "__main__":
    import asyncio
    import sys

    if len(sys.argv) > 1 and sys.argv[1] == "--serve":
        asyncio.run(run_server())
    elif len(sys.argv) > 1 and sys.argv[1] != "--help":
        asyncio.run(test_cli(sys.argv[1:]))
    elif len(sys.argv) == 1:
        # Default: run as MCP server
        asyncio.run(run_server())
    else:
        asyncio.run(test_cli([]))
