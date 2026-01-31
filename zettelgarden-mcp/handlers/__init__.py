"""
Handler modules for Zettelgarden MCP Server.

Each module contains handlers for a specific resource type.
"""

from .cards import (
    search_cards,
    get_card,
    create_card,
    update_card,
    list_starred_cards,
    get_card_children,
    get_next_child_id,
    format_card,
)
from .tasks import (
    list_tasks,
    get_task,
    create_task,
    update_task,
    complete_task,
    format_task,
)
from .schemas import (
    list_schemas,
    get_schema,
    create_schema,
    update_schema,
    delete_schema,
    get_schema_cards,
    format_schema,
    format_schema_list,
)
from .templates import (
    list_templates,
    get_template,
    format_template,
    format_template_list,
)
from .articles import (
    parse_url,
    create_article,
)

__all__ = [
    # Cards
    "search_cards",
    "get_card",
    "create_card",
    "update_card",
    "list_starred_cards",
    "get_card_children",
    "get_next_child_id",
    "format_card",
    # Tasks
    "list_tasks",
    "get_task",
    "create_task",
    "update_task",
    "complete_task",
    "format_task",
    # Schemas
    "list_schemas",
    "get_schema",
    "create_schema",
    "update_schema",
    "delete_schema",
    "get_schema_cards",
    "format_schema",
    "format_schema_list",
    # Templates
    "list_templates",
    "get_template",
    "format_template",
    "format_template_list",
    # Articles
    "parse_url",
    "create_article",
]
