"""
Card handlers for Zettelgarden MCP Server.

Provides functions for searching, getting, creating, and updating cards.
"""

import httpx

from config import API_URL, get_headers


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

    # Handle schema_id and structured_data updates
    # If schema_id is explicitly provided in args, use that (including null to remove)
    if "schema_id" in args:
        update_data["schema_id"] = args["schema_id"]
    # Otherwise, if updating structured_data and card already has a schema, preserve it
    elif "structured_data" in args and card.get("schema_id") is not None:
        update_data["schema_id"] = card.get("schema_id")

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
