// Package services provides business logic and tool implementations for Zettelgarden.
//
// Tool Registry Infrastructure:
// - registry.go: Core tool registration and execution
// - context.go: Tool execution context with transaction support
// - params.go: Parameter extraction and validation helpers
// - types.go: Tool type definitions and constants
//
// Domain-specific tools:
// - card_tools.go: Card CRUD operations and search
// - task_tools.go: Task management and scheduling
// - entity_tools.go: Entity management and linking
// - fact_tools.go: Fact extraction and retrieval
// - template_tools.go: Card template management
// - article_tools.go: Article parsing and creation
// - memory_tools.go: User memory operations
package services

import openai "github.com/sashabaranov/go-openai"

// Tool represents a tool that can be called by the LLM
type Tool struct {
	Definition openai.Tool
	Handler    func(args map[string]interface{}, ctx *ToolContext) (map[string]interface{}, error)
}

// Tool name constants
const (
	ToolSearchCards        = "search_cards"
	ToolGetCardByID        = "get_card_by_id"
	ToolBrowseCardHierarchy = "browse_card_hierarchy"
	ToolCreateCard         = "create_card"
	ToolUpdateCard         = "update_card"
	ToolGetCardAnalysis    = "get_card_analysis"
	ToolSearchFacts        = "search_facts"
	ToolGetCardFacts       = "get_card_facts"
	ToolGetEntityFacts     = "get_entity_facts"
	ToolGetFactCards       = "get_fact_cards"
	ToolGetTasks           = "get_tasks"
	ToolCreateTask         = "create_task"
	ToolUpdateTask         = "update_task"
	ToolGetTaskByID              = "get_task_by_id"
	ToolCompleteTask             = "complete_task"
	ToolDeleteTask               = "delete_task"
	ToolCompleteAndScheduleTask  = "complete_and_schedule_task"
	ToolGetEntityByName          = "get_entity_by_name"
	ToolSearchEntities           = "search_entities"
	ToolGetCardsByEntity         = "get_cards_by_entity"
	ToolGetEntityByID            = "get_entity_by_id"
	ToolMergeEntities            = "merge_entities"
	ToolUpdateEntity             = "update_entity"
	ToolDeleteEntity             = "delete_entity"
	ToolAddEntityToCard          = "add_entity_to_card"
	ToolRemoveEntityFromCard     = "remove_entity_from_card"
	ToolGetSimilarEntities       = "get_similar_entities"
	ToolGetUserMemory    = "get_user_memory"
	ToolGetTemplate      = "get_template"
	ToolListTemplates    = "list_templates"
	ToolGetNextChildID   = "get_next_child_id"
	// Article tools
	ToolParseURL      = "parse_url"
	ToolCreateArticle = "create_article"
)
