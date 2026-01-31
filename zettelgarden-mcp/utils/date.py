"""
Date utility functions for Zettelgarden MCP Server.

Provides consistent date parsing and formatting for API interactions.
"""

from datetime import datetime


def parse_date_param(date_str: str | None) -> str | None:
    """Parse a date parameter, handling 'today' and YYYY-MM-DD formats.

    Args:
        date_str: Date string (e.g., 'today', '2024-01-31') or None

    Returns:
        ISO formatted date string (YYYY-MM-DD) or None if input is None

    Examples:
        >>> parse_date_param('today')
        '2024-01-31'  # Current date
        >>> parse_date_param('2024-12-25')
        '2024-12-25'
        >>> parse_date_param(None)
        None
    """
    if not date_str:
        return None

    if date_str.lower() == "today":
        return datetime.now().strftime("%Y-%m-%d")

    # Assume already in YYYY-MM-DD format if not "today"
    return date_str


def format_datetime_for_api(dt: datetime) -> str:
    """Format a datetime object for API consumption.

    Args:
        dt: DateTime object to format

    Returns:
        ISO formatted datetime string (YYYY-MM-DDTHH:MM:SSZ)

    Examples:
        >>> dt = datetime(2024, 1, 31, 14, 30, 0)
        >>> format_datetime_for_api(dt)
        '2024-01-31T14:30:00Z'
    """
    return dt.strftime("%Y-%m-%dT%H:%M:%SZ")
