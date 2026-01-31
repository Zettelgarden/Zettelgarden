"""
Template handlers for Zettelgarden MCP Server.

Provides functions for listing and getting templates.
"""

import httpx

from config import API_URL, get_headers


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
