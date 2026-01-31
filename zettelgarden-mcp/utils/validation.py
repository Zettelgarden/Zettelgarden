"""
Validation utilities for Zettelgarden MCP Server.

Provides input validation for API parameters and environment variables.
"""

import re
from typing import Any, Optional


class ValidationError(Exception):
    """Raised when input validation fails."""
    pass


# Card ID pattern: e.g., "1", "1a", "1a2", "1a2b3"
CARD_ID_PATTERN = re.compile(r"^[a-zA-Z0-9]+$")

# Date pattern: YYYY-MM-DD
DATE_PATTERN = re.compile(r"^\d{4}-\d{2}-\d{2}$")

# Schema field types
VALID_SCHEMA_FIELD_TYPES = {
    "text",
    "number",
    "date",
    "boolean",
    "select",
    "multi-select",
    "link_to_card",
}


def validate_required(value: Any, field_name: str) -> None:
    """Validate that a required value is present and not empty.

    Args:
        value: The value to validate
        field_name: Name of the field for error messages

    Raises:
        ValidationError: If value is None or empty
    """
    if value is None:
        raise ValidationError(f"{field_name} is required")
    if isinstance(value, str) and not value.strip():
        raise ValidationError(f"{field_name} cannot be empty")
    if isinstance(value, (list, dict)) and len(value) == 0:
        raise ValidationError(f"{field_name} cannot be empty")


def validate_card_id(card_id: Any) -> str:
    """Validate a card ID.

    Args:
        card_id: Card ID to validate (can be int or str)

    Returns:
        The validated card ID as a string

    Raises:
        ValidationError: If card_id is invalid
    """
    if card_id is None:
        raise ValidationError("card_id is required")

    # Allow numeric IDs (for pk lookups)
    if isinstance(card_id, int):
        if card_id <= 0:
            raise ValidationError("card_id must be a positive integer")
        return str(card_id)

    if not isinstance(card_id, str):
        raise ValidationError("card_id must be a string or integer")

    if not card_id.strip():
        raise ValidationError("card_id cannot be empty")

    # Check pattern for string card IDs
    if not CARD_ID_PATTERN.match(card_id):
        raise ValidationError(f"Invalid card_id format: '{card_id}'")

    return card_id


def validate_date_string(date_str: Optional[str]) -> Optional[str]:
    """Validate a date string in YYYY-MM-DD format.

    Args:
        date_str: Date string to validate (or None)

    Returns:
        The validated date string or None

    Raises:
        ValidationError: If date_str is invalid format
    """
    if date_str is None:
        return None

    if not isinstance(date_str, str):
        raise ValidationError("Date must be a string")

    # Allow "today" as a special value
    if date_str.lower() == "today":
        return date_str

    if not DATE_PATTERN.match(date_str):
        raise ValidationError(
            f"Invalid date format: '{date_str}'. Expected YYYY-MM-DD or 'today'"
        )

    return date_str


def validate_schema_field_type(field_type: str) -> str:
    """Validate a schema field type.

    Args:
        field_type: Field type to validate

    Returns:
        The validated field type

    Raises:
        ValidationError: If field_type is invalid
    """
    if field_type not in VALID_SCHEMA_FIELD_TYPES:
        raise ValidationError(
            f"Invalid field type: '{field_type}'. Must be one of: {', '.join(sorted(VALID_SCHEMA_FIELD_TYPES))}"
        )
    return field_type


def validate_schema_fields(fields: list[dict]) -> list[dict]:
    """Validate schema field definitions.

    Args:
        fields: List of field definitions

    Returns:
        The validated fields list

    Raises:
        ValidationError: If fields are invalid
    """
    if not isinstance(fields, list):
        raise ValidationError("Fields must be a list")

    for i, field in enumerate(fields):
        if not isinstance(field, dict):
            raise ValidationError(f"Field {i} must be a dictionary")

        # Validate required 'name' field
        if "name" not in field:
            raise ValidationError(f"Field {i} missing required 'name' attribute")

        # Validate required 'type' field
        if "type" not in field:
            raise ValidationError(f"Field {i} ('{field.get('name', 'unnamed')}') missing required 'type' attribute")

        # Validate field type
        validate_schema_field_type(field["type"])

        # Validate that select/multi-select have options
        if field["type"] in ("select", "multi-select"):
            if "options" not in field or not field["options"]:
                raise ValidationError(
                    f"Field {i} ('{field['name']}') of type '{field['type']}' must have 'options' defined"
                )

    return fields


def validate_positive_int(value: Any, field_name: str) -> int:
    """Validate a positive integer.

    Args:
        value: Value to validate
        field_name: Name of the field for error messages

    Returns:
        The validated integer

    Raises:
        ValidationError: If value is invalid
    """
    if value is None:
        raise ValidationError(f"{field_name} is required")

    try:
        int_value = int(value)
    except (TypeError, ValueError):
        raise ValidationError(f"{field_name} must be an integer")

    if int_value <= 0:
        raise ValidationError(f"{field_name} must be a positive integer")

    return int_value
