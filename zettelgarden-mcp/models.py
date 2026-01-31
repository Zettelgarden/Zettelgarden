"""
Data models and type definitions for Zettelgarden MCP Server.

Provides type hints for API responses and internal data structures.

Usage:
    from models import Card, Task, SearchResponse

    def process_card(card: Card) -> str:
        return card['title']
"""

from typing import Optional, TypedDict, NotRequired


# =============================================================================
# Card Models
# =============================================================================

class CardMetadata(TypedDict):
    """Card metadata from search results."""
    id: int
    card_id: str


class CardTag(TypedDict):
    """Tag associated with a card."""
    id: int
    name: str


class CardReference(TypedDict):
    """Reference to another card."""
    id: int
    card_id: str
    title: str


class Card(TypedDict):
    """Full card data from API."""
    id: int
    card_id: str
    title: str
    body: str
    created_at: str
    updated_at: str
    is_starred: NotRequired[bool]
    link: NotRequired[str]
    schema_id: NotRequired[Optional[int]]
    structured_data: NotRequired[Optional[dict]]
    parent: NotRequired[Optional[dict]]
    tags: NotRequired[list[CardTag]]
    children: NotRequired[list[dict]]
    references: NotRequired[list[CardReference]]
    tasks: NotRequired[list[dict]]
    entities: NotRequired[list[dict]]


class CreateCardArgs(TypedDict):
    """Arguments for creating a card."""
    title: str
    body: str
    card_id: NotRequired[str]
    link: NotRequired[str]
    schema_id: NotRequired[Optional[int]]
    structured_data: NotRequired[dict]


class UpdateCardArgs(TypedDict):
    """Arguments for updating a card."""
    id: int
    title: NotRequired[str]
    body: NotRequired[str]
    link: NotRequired[str]
    schema_id: NotRequired[Optional[int]]
    structured_data: NotRequired[dict]


# =============================================================================
# Task Models
# =============================================================================

class TaskCard(TypedDict):
    """Card associated with a task."""
    id: int
    card_id: str
    title: str


class TaskTag(TypedDict):
    """Tag associated with a task."""
    id: int
    name: str


class Task(TypedDict):
    """Full task data from API."""
    id: int
    title: str
    description: NotRequired[Optional[str]]
    is_complete: bool
    status: NotRequired[str]
    priority: NotRequired[Optional[str]]
    scheduled_date: NotRequired[Optional[str]]
    dueDate: NotRequired[Optional[str]]
    due_date: NotRequired[Optional[str]]
    completed_at: NotRequired[Optional[str]]
    created_at: str
    updated_at: str
    card_pk: int
    user_id: NotRequired[int]
    card: NotRequired[Optional[TaskCard]]
    tags: NotRequired[list[TaskTag]]
    blocked_by: NotRequired[list[dict]]
    blocks: NotRequired[list[dict]]
    reminder_time: NotRequired[Optional[str]]


class CreateTaskArgs(TypedDict):
    """Arguments for creating a task."""
    title: str
    description: NotRequired[Optional[str]]
    scheduled_date: NotRequired[str]
    priority: NotRequired[str]
    card_pk: NotRequired[int]


class UpdateTaskArgs(TypedDict):
    """Arguments for updating a task."""
    task_id: int
    title: NotRequired[str]
    description: NotRequired[Optional[str]]
    is_complete: NotRequired[bool]
    priority: NotRequired[str]
    scheduled_date: NotRequired[str]
    status: NotRequired[str]


# =============================================================================
# Template Models
# =============================================================================

class Template(TypedDict):
    """Template data from API."""
    id: int
    name: str
    title: NotRequired[Optional[str]]
    body: str
    created_at: str


# =============================================================================
# Schema Models
# =============================================================================

class SchemaField(TypedDict):
    """Schema field definition.

    Valid field types:
    - text: Single-line text input
    - number: Numeric input
    - date: Date picker
    - boolean: Toggle/checkbox
    - select: Single select from options
    - multi-select: Multiple select from options
    - link_to_card: Reference to another card
    """
    name: str
    type: str  # text|number|date|boolean|select|multi-select|link_to_card
    required: bool
    options: NotRequired[list[str]]


class Schema(TypedDict):
    """Schema data from API."""
    id: int
    name: str
    slug: str
    created_at: str
    updated_at: NotRequired[str]
    fields: list[SchemaField]


class CreateSchemaArgs(TypedDict):
    """Arguments for creating a schema."""
    name: str
    fields: list[SchemaField]


class UpdateSchemaArgs(TypedDict):
    """Arguments for updating a schema."""
    schema_id: int
    name: NotRequired[str]
    fields: NotRequired[list[SchemaField]]


# =============================================================================
# Search Models
# =============================================================================

class SearchResultTag(TypedDict):
    """Tag in a search result."""
    id: int
    name: str


class SearchResultMetadata(TypedDict):
    """Metadata for a search result."""
    id: int
    card_id: str


class SearchResult(TypedDict):
    """Single search result."""
    title: str
    preview: str
    metadata: SearchResultMetadata
    tags: NotRequired[list[SearchResultTag]]


class SearchResponse(TypedDict):
    """Search API response."""
    results: list[SearchResult]
    total: int


class SearchCardsArgs(TypedDict):
    """Arguments for searching cards."""
    query: str
    full_text: NotRequired[bool]
    limit: NotRequired[int]


# =============================================================================
# Error Models
# =============================================================================

class ErrorResponse(TypedDict):
    """Error response from API."""
    error: NotRequired[str]
    message: NotRequired[str]
    status_code: NotRequired[int]
