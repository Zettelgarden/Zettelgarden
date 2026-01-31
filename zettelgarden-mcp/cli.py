"""
CLI for testing the Zettelgarden MCP Server.

Provides a command-line interface for testing API interactions.
"""

import sys
from datetime import datetime
from typing import Optional

import httpx

import handlers
from config import API_URL, TOKEN, validate_config


async def test_cli(args: list[str]) -> None:
    """CLI for testing the API connection."""
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
                result = await handlers.search_cards(client, {"query": query, "limit": 10})
                print(result)

            elif command == "card":
                if len(args) < 2:
                    print("Usage: python server.py card <id>")
                    sys.exit(1)
                card_id = int(args[1]) if args[1].isdigit() else args[1]
                result = await handlers.get_card(client, {"card_id": card_id})
                print(result)

            elif command == "tasks":
                filters = {}
                if len(args) > 1:
                    if args[1] == "today":
                        filters["scheduled_date"] = "today"
                    elif args[1] == "incomplete":
                        filters["completed"] = False
                result = await handlers.list_tasks(client, filters)
                print(result)

            elif command == "task":
                if len(args) < 2:
                    print("Usage: python server.py task <id>")
                    sys.exit(1)
                result = await handlers.get_task(client, {"task_id": int(args[1])})
                print(result)

            elif command == "complete":
                if len(args) < 2:
                    print("Usage: python server.py complete <task_id>")
                    sys.exit(1)
                result = await handlers.complete_task(client, {"task_id": int(args[1])})
                print(result)

            elif command == "starred":
                result = await handlers.list_starred_cards(client)
                print(result)

            elif command == "create":
                # create <title> [body]
                if len(args) < 2:
                    print("Usage: python server.py create <title> [body]")
                    print("  Example: python server.py create 'My Note' 'Note content here'")
                    sys.exit(1)
                title = args[1]
                body = " ".join(args[2:]) if len(args) > 2 else ""
                result = await handlers.create_card(client, {"title": title, "body": body})
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

                result = await handlers.update_card(client, update_args)
                print(result)

            elif command == "ping":
                # Just test the connection
                from .config import get_headers
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
                result = await handlers.parse_url(client, {"url": url})
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

                result = await handlers.create_article(client, article_args)
                print(result)

            else:
                print_help()

        except httpx.HTTPStatusError as e:
            print(f"API Error {e.response.status_code}: {e.response.text}")
            sys.exit(1)
        except Exception as e:
            print(f"Error: {e}")
            sys.exit(1)


def print_help() -> None:
    """Print the CLI help message."""
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
