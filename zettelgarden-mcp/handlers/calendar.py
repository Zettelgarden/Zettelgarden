"""
Calendar handlers for Zettelgarden MCP Server.
"""

import httpx
from datetime import datetime, timedelta
from typing import Any


async def list_external_calendars(client: httpx.AsyncClient) -> str:
    """List all external calendar subscriptions for the current user."""
    try:
        response = await client.get("/user/external-calendars")
        response.raise_for_status()
        data = response.json()

        if not data:
            return "No external calendars found."

        calendars = data
        result = f"Found {len(calendars)} external calendar(s):\n\n"

        for cal in calendars:
            result += f"- **{cal['name']}** (ID: {cal['id']})\n"
            result += f"  URL: {cal['url']}\n"
            result += f"  Sync: {'enabled' if cal['sync_enabled'] else 'disabled'}\n"
            result += f"  Color: {cal['color']}\n"
            if cal.get('last_synced_at'):
                result += f"  Last synced: {cal['last_synced_at']}\n"
            if cal.get('last_error'):
                result += f"  Last error: {cal['last_error']}\n"
            result += "\n"

        return result.strip()
    except httpx.HTTPError as e:
        return f"Error listing external calendars: {e}"
    except Exception as e:
        return f"Unexpected error: {e}"


async def list_external_events(client: httpx.AsyncClient, args: dict[str, Any]) -> str:
    """List external calendar events within a date range."""
    start = args.get("start")
    end = args.get("end")

    if not start or not end:
        return "Error: Both 'start' and 'end' dates are required (ISO 8601 format)."

    try:
        # Validate date format
        try:
            datetime.fromisoformat(start.replace('Z', '+00:00'))
            datetime.fromisoformat(end.replace('Z', '+00:00'))
        except ValueError:
            return "Error: Invalid date format. Use ISO 8601 format (e.g., '2026-01-01T00:00:00Z')."

        response = await client.get(
            "/user/external-events",
            params={"start": start, "end": end}
        )
        response.raise_for_status()
        data = response.json()

        events = data.get("events", [])
        total = data.get("total", 0)

        if not events:
            return f"No external events found between {start} and {end}."

        result = f"Found {total} external event(s):\n\n"

        for event in events:
            start_time = datetime.fromisoformat(event['start_time'].replace('Z', '+00:00'))
            result += f"- **{event['title']}** (ID: {event['id']})\n"
            result += f"  Time: {start_time.strftime('%Y-%m-%d %H:%M')}"
            if event.get('all_day'):
                result += " (all day)"
            result += "\n"
            if event.get('location'):
                result += f"  Location: {event['location']}\n"
            if event.get('card_pk') and event['card_pk'] > 0:
                result += f"  Linked to card: {event['card_pk']}\n"
            if event.get('description'):
                desc = event['description'][:100] + "..." if len(event['description']) > 100 else event['description']
                result += f"  Description: {desc}\n"
            result += "\n"

        return result.strip()
    except httpx.HTTPError as e:
        return f"Error listing external events: {e}"
    except Exception as e:
        return f"Unexpected error: {e}"


async def link_event_to_card(client: httpx.AsyncClient, args: dict[str, Any]) -> str:
    """Link an external calendar event to a card."""
    event_id = args.get("event_id")
    card_pk = args.get("card_pk")

    if event_id is None or card_pk is None:
        return "Error: Both 'event_id' and 'card_pk' are required."

    try:
        response = await client.put(
            f"/user/external-events/{event_id}/link",
            json={"card_pk": card_pk}
        )
        response.raise_for_status()
        event = response.json()

        return f"Successfully linked event '{event['title']}' to card {card_pk}."
    except httpx.HTTPStatusError as e:
        if e.response.status_code == 404:
            return "Error: Event or card not found."
        elif e.response.status_code == 400:
            return f"Error: {e.response.json().get('error', 'Invalid request')}"
        else:
            return f"Error linking event to card: {e}"
    except httpx.HTTPError as e:
        return f"Error linking event to card: {e}"
    except Exception as e:
        return f"Unexpected error: {e}"
