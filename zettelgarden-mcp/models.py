"""
Data models and type definitions for Zettelgarden MCP Server.

Provides type hints for API responses and internal data structures.
"""

from typing import Optional, Required, TypedDict, NotRequired


# Card models
class CardMetadata(TypedDict):
    id: int
    card_id: str


class CardTag(TypedDict):
    id: int
    name: str


class CardReference(TypedDict):
    id: int
    card_id: str
    title: str


class Card(TypedDict):
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


# Task models
class TaskCard(TypedDict):
    id: int
    card_id: str
    title: str


class TaskTag(TypedDict):
    id: int
    name: str


class Task(TypedDict):
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


# Template models
class Template(TypedDict):
    id: int
    name: str
    title: NotRequired[Optional[str]]
    body: str
    created_at: str


# Schema models
class SchemaField(TypedDict):
    name: str
    type: str  # text|number|date|boolean|select|multi-select|link_to_card
    required: bool
    options: NotRequired[list[str]]


class Schema(TypedDict):
    id: int
    name: str
    slug: str
    created_at: str
    updated_at: NotRequired[str]
    fields: list[SchemaField]


# Search models
class SearchResultTag(TypedDict):
    id: int
    name: str


class SearchResultMetadata(TypedDict):
    id: int
    card_id: str


class SearchResult(TypedDict):
    title: str
    preview: str
    metadata: SearchResultMetadata
    tags: NotRequired[list[SearchResultTag]]


class SearchResponse(TypedDict):
    results: list[SearchResult]
    total: int
