"""
Configuration module for Zettelgarden MCP Server.

Handles environment variables and API configuration.
"""

import os
from typing import Required

# Environment configuration
API_URL: str = os.environ.get("ZETTELGARDEN_API_URL", "http://localhost:8080")
TOKEN: str = os.environ.get("ZETTELGARDEN_TOKEN", "")


def get_headers() -> dict:
    """Get auth headers for API requests."""
    return {"Authorization": f"Bearer {TOKEN}"}


def validate_config() -> tuple[bool, str]:
    """Validate that required configuration is present.

    Returns:
        Tuple of (is_valid, error_message)
    """
    if not TOKEN:
        return False, "ZETTELGARDEN_TOKEN environment variable is required"
    return True, ""
