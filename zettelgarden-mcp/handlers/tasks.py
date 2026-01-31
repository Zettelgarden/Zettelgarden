"""
Task handlers for Zettelgarden MCP Server.

Provides functions for listing, getting, creating, and updating tasks.
"""

from datetime import datetime
from typing import Optional

import httpx

from config import API_URL, get_headers
from utils import parse_date_param, format_datetime_for_api


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


async def list_tasks(client: httpx.AsyncClient, args: dict) -> str:
    """List tasks with filters."""
    params = {"limit": args.get("limit", 50)}

    if "completed" in args:
        params["completed"] = "true" if args["completed"] else "false"

    scheduled = parse_date_param(args.get("scheduled_date"))
    if scheduled:
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
    scheduled = parse_date_param(args.get("scheduled_date"))

    task_data = {
        "title": args.get("title", ""),
        "description": args.get("description"),
        "priority": args.get("priority"),
        "card_pk": args.get("card_pk", 0),
        "is_complete": False
    }

    if scheduled:
        # Format as full datetime for API
        scheduled_dt = datetime.fromisoformat(f"{scheduled}T00:00:00")
        task_data["scheduled_date"] = format_datetime_for_api(scheduled_dt)

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
        scheduled = parse_date_param(args["scheduled_date"])
        if scheduled:
            scheduled_dt = datetime.fromisoformat(f"{scheduled}T00:00:00")
            update_data["scheduled_date"] = format_datetime_for_api(scheduled_dt)

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
