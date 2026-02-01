"""
Configuration module for Zettelgarden MCP Server.

Handles environment variables and API configuration.
"""

import logging
import os

from utils import ValidationError, validate_required

# Configure logging
LOG_LEVEL = os.environ.get("ZETTELGARDEN_LOG_LEVEL", "WARNING").upper()
logging.basicConfig(
    level=LOG_LEVEL,
    format="%(asctime)s - %(name)s - %(levelname)s - %(message)s"
)
logger = logging.getLogger(__name__)

# Environment configuration
API_URL: str = os.environ.get("ZETTELGARDEN_API_URL", "http://localhost:8080")
TOKEN: str = os.environ.get("ZETTELGARDEN_TOKEN", "")
TIMEOUT: float = float(os.environ.get("ZETTELGARDEN_TIMEOUT", "30"))


def get_headers() -> dict:
    """Get auth headers for API requests."""
    return {"Authorization": f"Bearer {TOKEN}"}


def validate_config() -> tuple[bool, str]:
    """Validate that required configuration is present.

    Returns:
        Tuple of (is_valid, error_message)
    """
    # Validate TOKEN
    if not TOKEN:
        return False, "ZETTELGARDEN_TOKEN environment variable is required"
    try:
        validate_required(TOKEN, "ZETTELGARDEN_TOKEN")
    except ValidationError as e:
        return False, str(e)

    # Validate API_URL
    if not API_URL:
        return False, "ZETTELGARDEN_API_URL cannot be empty"
    try:
        validate_required(API_URL, "ZETTELGARDEN_API_URL")
    except ValidationError as e:
        return False, str(e)

    return True, ""
