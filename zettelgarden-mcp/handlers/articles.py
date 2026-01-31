"""
Article handlers for Zettelgarden MCP Server.

Provides functions for parsing URLs and creating article cards from web content.
"""

import httpx

from config import API_URL, get_headers


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
