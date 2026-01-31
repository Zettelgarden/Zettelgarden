"""
Tool definitions and registration for Zettelgarden MCP Server.

Defines all available MCP tools and routes tool calls to handlers.
"""

import httpx
from mcp.types import Tool, TextContent

import handlers


def list_tools() -> list[Tool]:
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


async def handle_tool(client: httpx.AsyncClient, name: str, args: dict) -> str:
    """Route tool calls to handlers."""

    # Card tools
    if name == "search_cards":
        return await handlers.search_cards(client, args)
    elif name == "get_card":
        return await handlers.get_card(client, args)
    elif name == "create_card":
        return await handlers.create_card(client, args)
    elif name == "update_card":
        return await handlers.update_card(client, args)
    elif name == "list_starred_cards":
        return await handlers.list_starred_cards(client)
    elif name == "get_card_children":
        return await handlers.get_card_children(client, args)
    elif name == "get_next_child_id":
        return await handlers.get_next_child_id(client, args)

    # Template tools
    elif name == "list_templates":
        return await handlers.list_templates(client)
    elif name == "get_template":
        return await handlers.get_template(client, args)

    # Schema tools
    elif name == "list_schemas":
        return await handlers.list_schemas(client)
    elif name == "get_schema":
        return await handlers.get_schema(client, args)
    elif name == "create_schema":
        return await handlers.create_schema(client, args)
    elif name == "update_schema":
        return await handlers.update_schema(client, args)
    elif name == "delete_schema":
        return await handlers.delete_schema(client, args)
    elif name == "get_schema_cards":
        return await handlers.get_schema_cards(client, args)

    # Task tools
    elif name == "list_tasks":
        return await handlers.list_tasks(client, args)
    elif name == "get_task":
        return await handlers.get_task(client, args)
    elif name == "create_task":
        return await handlers.create_task(client, args)
    elif name == "update_task":
        return await handlers.update_task(client, args)
    elif name == "complete_task":
        return await handlers.complete_task(client, args)

    # Article tools
    elif name == "parse_url":
        return await handlers.parse_url(client, args)
    elif name == "create_article":
        return await handlers.create_article(client, args)

    else:
        return f"Unknown tool: {name}"
