"""
Schema handlers for Zettelgarden MCP Server.

Provides functions for listing, getting, creating, updating, and deleting schemas.
"""

import httpx
import logging

from config import API_URL, get_headers
from utils import validate_required, validate_schema_fields, ValidationError, format_api_error

logger = logging.getLogger(__name__)


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
    # Validate inputs
    try:
        validate_required(args.get("name"), "Schema name")
        validate_schema_fields(args.get("fields", []))
    except ValidationError as e:
        logger.warning(f"Schema validation failed: {e}")
        return f"Validation Error: {e}"

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

    # Validate fields if provided
    if "fields" in args:
        try:
            validate_schema_fields(args["fields"])
        except ValidationError as e:
            logger.warning(f"Schema validation failed: {e}")
            return f"Validation Error: {e}"

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
