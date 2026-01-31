#!/usr/bin/env python3
"""
Zettelgarden MCP Server

Provides Claude with access to Zettelgarden cards, tasks, and search.

Configuration:
    Set environment variables:
    - ZETTELGARDEN_API_URL: Base URL (default: http://localhost:8080)
    - ZETTELGARDEN_TOKEN: JWT auth token (required)
    - ZETTELGARDEN_LOG_LEVEL: Logging level (default: WARNING)

Usage:
    python server.py          # Run as MCP server (default)
    python server.py --serve  # Run as MCP server (explicit)
    python server.py <cmd>    # Run test CLI
"""

import asyncio
import logging
import sys

import httpx
from mcp.server import Server
from mcp.server.stdio import stdio_server
from mcp.types import TextContent

# Import local modules
from config import TOKEN, validate_config
import cli, tools

logger = logging.getLogger(__name__)

# Create MCP server instance
server = Server("zettelgarden")


@server.list_tools()
async def list_tools() -> list:
    """List available tools."""
    return tools.list_tools()


@server.call_tool()
async def call_tool(name: str, arguments: dict) -> list[TextContent]:
    """Handle tool calls."""
    logger.info(f"Tool called: {name} with args: {arguments}")
    async with httpx.AsyncClient(timeout=30.0) as client:
        try:
            result = await tools.handle_tool(client, name, arguments)
            logger.debug(f"Tool {name} returned successfully")
            return [TextContent(type="text", text=result)]
        except httpx.HTTPStatusError as e:
            logger.error(f"API error for {name}: {e.response.status_code} - {e.response.text}")
            return [TextContent(type="text", text=f"API Error {e.response.status_code}: {e.response.text}")]
        except Exception as e:
            logger.exception(f"Error handling tool {name}: {e}")
            return [TextContent(type="text", text=f"Error: {str(e)}")]


async def run_server():
    """Run the MCP server."""
    is_valid, error = validate_config()
    if not is_valid:
        logger.error(f"Configuration error: {error}")
        print(f"Error: {error}", file=sys.stderr)
        print("Get your token from the Zettelgarden web UI after logging in", file=sys.stderr)
        sys.exit(1)

    logger.info("Starting Zettelgarden MCP server")
    async with stdio_server() as (read_stream, write_stream):
        await server.run(read_stream, write_stream, server.create_initialization_options())


def main():
    """Main entry point."""
    if len(sys.argv) > 1 and sys.argv[1] == "--serve":
        asyncio.run(run_server())
    elif len(sys.argv) > 1 and sys.argv[1] != "--help":
        asyncio.run(cli.test_cli(sys.argv[1:]))
    elif len(sys.argv) == 1:
        # Default: run as MCP server
        asyncio.run(run_server())
    else:
        asyncio.run(cli.test_cli([]))


if __name__ == "__main__":
    main()
