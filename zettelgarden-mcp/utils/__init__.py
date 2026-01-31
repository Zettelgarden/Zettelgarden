"""
Utility modules for Zettelgarden MCP Server.
"""

from .date import parse_date_param, format_datetime_for_api
from .errors import (
    ZettelgardenError,
    APIError,
    AuthError,
    ValidationError,
    NotFoundError,
    handle_errors,
    format_error,
    format_api_error,
)
from .validation import (
    validate_required,
    validate_card_id,
    validate_date_string,
    validate_schema_field_type,
    validate_schema_fields,
    validate_positive_int,
)

__all__ = [
    # Date utilities
    "parse_date_param",
    "format_datetime_for_api",
    # Error handling
    "ZettelgardenError",
    "APIError",
    "AuthError",
    "ValidationError",
    "NotFoundError",
    "handle_errors",
    "format_error",
    "format_api_error",
    # Validation
    "validate_required",
    "validate_card_id",
    "validate_date_string",
    "validate_schema_field_type",
    "validate_schema_fields",
    "validate_positive_int",
]
