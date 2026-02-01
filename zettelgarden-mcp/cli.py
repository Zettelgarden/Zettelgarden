"""
CLI for testing the Zettelgarden MCP Server.

Provides a command-line interface for testing API interactions.
"""

import argparse
import asyncio
import sys

import httpx

import handlers
from config import API_URL, TOKEN, TIMEOUT


def build_parser() -> argparse.ArgumentParser:
    """Build the argument parser for the CLI.

    Returns:
        Configured ArgumentParser with subcommands
    """
    parser = argparse.ArgumentParser(
        prog="zettelgarden-mcp",
        description="Zettelgarden MCP Server - Test CLI",
        epilog=(
            "Environment:\n"
            "  ZETTELGARDEN_TOKEN     JWT auth token (required)\n"
            "  ZETTELGARDEN_API_URL   API URL (default: http://localhost:8080)\n\n"
            "To run as MCP server: python server.py --serve"
        ),
        formatter_class=argparse.RawDescriptionHelpFormatter,
    )

    subparsers = parser.add_subparsers(
        dest="command",
        title="Commands",
        description="Available commands for interacting with Zettelgarden API",
        metavar="COMMAND",
    )

    # ping command
    subparsers.add_parser(
        "ping",
        help="Test API connection",
        description="Test the connection to the Zettelgarden API server",
    )

    # search command
    search_parser = subparsers.add_parser(
        "search",
        help="Search for cards",
        description="Search for cards by text query",
    )
    search_parser.add_argument(
        "query",
        help="Search query (e.g., 'python', '#project', '@[John Doe]')"
    )
    search_parser.add_argument(
        "-l", "--limit",
        type=int,
        default=10,
        help="Max results to return (default: 10)"
    )

    # card command
    card_parser = subparsers.add_parser(
        "card",
        help="Get a card",
        description="Get a specific card by ID or card_id",
    )
    card_parser.add_argument(
        "id",
        help="Card ID - numeric pk or card_id string (e.g., 123 or '1a2')"
    )

    # create command
    create_parser = subparsers.add_parser(
        "create",
        help="Create a new card",
        description="Create a new card in Zettelgarden",
    )
    create_parser.add_argument("title", help="Card title")
    create_parser.add_argument("body", nargs="?", default="", help="Card body content")
    create_parser.add_argument("--card-id", help="Optional card_id (e.g., '1a')")
    create_parser.add_argument("--link", help="Optional URL link")

    # update command
    update_parser = subparsers.add_parser(
        "update",
        help="Update a card",
        description="Update an existing card's title, body, or link",
    )
    update_parser.add_argument("id", type=int, help="Card numeric ID (pk)")
    update_parser.add_argument("--title", help="New title")
    update_parser.add_argument("--body", help="New body content")
    update_parser.add_argument("--link", help="New link")

    # starred command
    subparsers.add_parser(
        "starred",
        help="List starred cards",
        description="Get all starred/pinned cards",
    )

    # tasks command
    tasks_parser = subparsers.add_parser(
        "tasks",
        help="List tasks",
        description="List tasks with optional filters",
    )
    tasks_parser.add_argument("--today", action="store_true", help="Show today's tasks")
    tasks_parser.add_argument("--incomplete", action="store_true", help="Show incomplete tasks only")
    tasks_parser.add_argument("--completed", action="store_true", help="Show completed tasks only")
    tasks_parser.add_argument("--priority", help="Filter by priority (e.g., 'high', 'medium', 'low')")
    tasks_parser.add_argument("--limit", type=int, default=50, help="Max results (default: 50)")

    # task command (get single task)
    task_parser = subparsers.add_parser(
        "task",
        help="Get a task",
        description="Get a specific task by ID",
    )
    task_parser.add_argument("id", type=int, help="Task ID")

    # complete command
    complete_parser = subparsers.add_parser(
        "complete",
        help="Mark a task as complete",
        description="Mark a specific task as complete",
    )
    complete_parser.add_argument("id", type=int, help="Task ID to complete")

    # parse-url command
    url_parser = subparsers.add_parser(
        "parse-url",
        help="Parse a URL",
        description="Parse a URL to extract article content",
    )
    url_parser.add_argument("url", help="URL to parse")

    # article command
    article_parser = subparsers.add_parser(
        "article",
        help="Create an article card",
        description="Create a new article card from a URL",
    )
    article_parser.add_argument("url", help="URL of the article to import")
    article_parser.add_argument("--card-id", help="Optional card_id (e.g., '1a')")
    article_parser.add_argument("--tags", help="Optional custom tags (default: '#to-read #reference')")

    # schemas command
    subparsers.add_parser(
        "schemas",
        help="List schemas",
        description="List all schemas for the current user",
    )

    # schema command (get single schema)
    schema_parser = subparsers.add_parser(
        "schema",
        help="Get a schema",
        description="Get a specific schema by ID or slug",
    )
    schema_parser.add_argument("ref", help="Schema ID (numeric) or slug (string)")

    # templates command
    subparsers.add_parser(
        "templates",
        help="List templates",
        description="Get all templates for the current user",
    )

    # template command (get single template)
    template_parser = subparsers.add_parser(
        "template",
        help="Get a template",
        description="Get a specific template by ID",
    )
    template_parser.add_argument("id", type=int, help="Template ID")

    return parser


async def run_command(args: argparse.Namespace) -> None:
    """Execute the command specified by the parsed arguments.

    Args:
        args: Parsed command-line arguments
    """
    async with httpx.AsyncClient(timeout=TIMEOUT) as client:
        try:
            if args.command == "ping":
                from config import get_headers
                resp = await client.get(f"{API_URL}/api/auth", headers=get_headers())
                if resp.status_code == 200:
                    print("✓ Connected successfully!")
                    print(f"  User: {resp.json()}")
                else:
                    print(f"✗ Connection failed: {resp.status_code}")
                    print(f"  {resp.text}")

            elif args.command == "search":
                result = await handlers.search_cards(
                    client,
                    {"query": args.query, "limit": args.limit}
                )
                print(result)

            elif args.command == "card":
                card_id = int(args.id) if args.id.isdigit() else args.id
                result = await handlers.get_card(client, {"card_id": card_id})
                print(result)

            elif args.command == "create":
                create_args = {"title": args.title, "body": args.body}
                if args.card_id:
                    create_args["card_id"] = args.card_id
                if args.link:
                    create_args["link"] = args.link
                result = await handlers.create_card(client, create_args)
                print(result)

            elif args.command == "update":
                update_args = {"id": args.id}
                if args.title:
                    update_args["title"] = args.title
                if args.body:
                    update_args["body"] = args.body
                if args.link:
                    update_args["link"] = args.link

                if len(update_args) == 1:
                    print("Error: Must provide at least one of --title, --body, or --link")
                    sys.exit(1)

                result = await handlers.update_card(client, update_args)
                print(result)

            elif args.command == "starred":
                result = await handlers.list_starred_cards(client)
                print(result)

            elif args.command == "tasks":
                filters = {"limit": args.limit}
                if args.today:
                    filters["scheduled_date"] = "today"
                if args.incomplete:
                    filters["completed"] = False
                if args.completed:
                    filters["completed"] = True
                if args.priority:
                    filters["priority"] = args.priority
                result = await handlers.list_tasks(client, filters)
                print(result)

            elif args.command == "task":
                result = await handlers.get_task(client, {"task_id": args.id})
                print(result)

            elif args.command == "complete":
                result = await handlers.complete_task(client, {"task_id": args.id})
                print(result)

            elif args.command == "parse-url":
                result = await handlers.parse_url(client, {"url": args.url})
                print(result)

            elif args.command == "article":
                article_args = {"url": args.url}
                if args.card_id:
                    article_args["card_id"] = args.card_id
                if args.tags:
                    article_args["tags"] = args.tags
                result = await handlers.create_article(client, article_args)
                print(result)

            elif args.command == "schemas":
                result = await handlers.list_schemas(client)
                print(result)

            elif args.command == "schema":
                result = await handlers.get_schema(client, {"schema_ref": args.ref})
                print(result)

            elif args.command == "templates":
                result = await handlers.list_templates(client)
                print(result)

            elif args.command == "template":
                result = await handlers.get_template(client, {"template_id": args.id})
                print(result)

        except httpx.HTTPStatusError as e:
            print(f"API Error {e.response.status_code}: {e.response.text}")
            sys.exit(1)
        except Exception as e:
            print(f"Error: {e}")
            sys.exit(1)


async def test_cli(cli_args: list[str]) -> None:
    """CLI entry point for testing the API connection.

    Args:
        cli_args: Command-line arguments (excluding program name)
    """
    parser = build_parser()

    # If no args or help requested, show help without requiring TOKEN
    if not cli_args or cli_args[0] in ("--help", "-h", "help"):
        parser.print_help()
        return

    args = parser.parse_args(cli_args)

    # If no command specified, show help without requiring TOKEN
    if not args.command:
        parser.print_help()
        return

    # For commands that need API access, check TOKEN
    if not TOKEN:
        print("Error: ZETTELGARDEN_TOKEN environment variable is required")
        print("Export it first: export ZETTELGARDEN_TOKEN='your-token'")
        sys.exit(1)

    print(f"API URL: {API_URL}")
    print(f"Token: {TOKEN[:20]}...{TOKEN[-10:]}" if len(TOKEN) > 30 else f"Token: {TOKEN}")
    print()

    await run_command(args)
