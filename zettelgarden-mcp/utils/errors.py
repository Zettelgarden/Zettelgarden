"""
Error handling utilities for Zettelgarden MCP Server.

Provides custom exception classes and error handling patterns.
"""

import functools
import logging
from typing import Any, Callable

import httpx

logger = logging.getLogger(__name__)


# =============================================================================
# Custom Exception Classes
# =============================================================================

class ZettelgardenError(Exception):
    """Base exception for Zettelgarden MCP errors."""

    def __init__(self, message: str, details: dict | None = None):
        self.message = message
        self.details = details or {}
        super().__init__(self.message)


class APIError(ZettelgardenError):
    """Exception raised when API request fails."""

    def __init__(self, message: str, status_code: int | None = None, response_text: str | None = None):
        super().__init__(message)
        self.status_code = status_code
        self.response_text = response_text


class AuthError(ZettelgardenError):
    """Exception raised when authentication fails."""

    def __init__(self, message: str = "Authentication failed"):
        super().__init__(message)


class ValidationError(ZettelgardenError):
    """Exception raised when input validation fails."""

    def __init__(self, message: str, field: str | None = None):
        super().__init__(message)
        self.field = field


class NotFoundError(ZettelgardenError):
    """Exception raised when a resource is not found."""

    def __init__(self, resource_type: str, identifier: str):
        message = f"{resource_type} not found: {identifier}"
        super().__init__(message)
        self.resource_type = resource_type
        self.identifier = identifier


# =============================================================================
# Error Handler Decorator
# =============================================================================

def handle_errors(
    *,
    reraise: bool = False,
    default_message: str = "An error occurred"
) -> Callable:
    """Decorator for async functions to standardize error handling.

    Args:
        reraise: If True, re-raise exceptions after logging. If False, return error string.
        default_message: Default error message if exception is not a ZettelgardenError

    Returns:
        Decorator function

    Examples:
        @handle_errors()
        async def my_handler(client, args):
            ...

        @handle_errors(reraise=True, default_message="Failed to create card")
        async def create_card(client, args):
            ...
    """

    def decorator(func: Callable) -> Callable:
        @functools.wraps(func)
        async def wrapper(*args: Any, **kwargs: Any) -> Any:
            try:
                return await func(*args, **kwargs)
            except ZettelgardenError as e:
                logger.warning(f"{func.__name__} failed: {e.message}")
                if reraise:
                    raise
                return f"Error: {e.message}"
            except httpx.HTTPStatusError as e:
                logger.error(f"{func.__name__} API error: {e.response.status_code} - {e.response.text}")
                if reraise:
                    raise APIError(
                        f"API Error {e.response.status_code}",
                        status_code=e.response.status_code,
                        response_text=e.response.text
                    )
                return f"API Error {e.response.status_code}: {e.response.text}"
            except httpx.RequestError as e:
                logger.error(f"{func.__name__} network error: {e}")
                if reraise:
                    raise
                return f"Network Error: Unable to reach API server"
            except Exception as e:
                logger.exception(f"{func.__name__} unexpected error: {e}")
                if reraise:
                    raise
                return f"{default_message}: {str(e)}"

        return wrapper

    return decorator


# =============================================================================
# Error Formatting Utilities
# =============================================================================

def format_error(error: Exception) -> str:
    """Format an exception into a user-friendly error string.

    Args:
        error: The exception to format

    Returns:
        Formatted error string
    """
    if isinstance(error, APIError):
        if error.status_code:
            return f"API Error {error.status_code}: {error.message}"
        return f"API Error: {error.message}"
    elif isinstance(error, AuthError):
        return f"Authentication Error: {error.message}"
    elif isinstance(error, ValidationError):
        return f"Validation Error: {error.message}"
    elif isinstance(error, NotFoundError):
        return f"Not Found: {error.message}"
    elif isinstance(error, ZettelgardenError):
        return f"Error: {error.message}"
    else:
        return f"Error: {str(error)}"


def format_api_error(response: httpx.Response) -> str:
    """Format an HTTP error response into a user-friendly string.

    Args:
        response: The HTTP response with error status

    Returns:
        Formatted error string
    """
    status = response.status_code
    try:
        error_data = response.json()
        message = error_data.get("error", error_data.get("message", response.text))
    except Exception:
        message = response.text[:200]  # Truncate long responses

    return f"API Error {status}: {message}"
