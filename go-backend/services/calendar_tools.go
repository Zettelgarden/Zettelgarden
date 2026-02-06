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
// - calendar_tools.go: Calendar integration
// - article_tools.go: Article parsing and creation
// - memory_tools.go: User memory operations
package services

import (
	"fmt"
	"time"
)

// RegisterCalendarTools registers all calendar-related tools
func (tr *ToolRegistry) RegisterCalendarTools() {
	RegisterTool(tr, "list_external_calendars",
		"List all external calendar subscriptions for the current user. Returns calendar names, URLs, sync status, colors, and last sync times.",
		handleListExternalCalendars,
	)

	RegisterTool(tr, "list_external_events",
		"List external calendar events within a date range. Returns events with titles, times, locations, linked cards, and descriptions.",
		handleListExternalEvents,
		ToolParam{Name: "start", Type: "string", Required: true, Desc: "Start date in ISO 8601 format (e.g., '2026-01-01T00:00:00Z')"},
		ToolParam{Name: "end", Type: "string", Required: true, Desc: "End date in ISO 8601 format (e.g., '2026-12-31T23:59:59Z')"},
	)

	RegisterTool(tr, "link_event_to_card",
		"Link an external calendar event to a card. Creates a bidirectional association between the event and card.",
		handleLinkEventToCard,
		ToolParam{Name: "event_id", Type: "integer", Required: true, Desc: "The external event ID"},
		ToolParam{Name: "card_pk", Type: "integer", Required: true, Desc: "The card primary key to link to"},
	)
}

// Calendar tool handlers

func handleListExternalCalendars(args map[string]interface{}, ctx *ToolContext) (map[string]interface{}, error) {
	service := NewExternalEventService(ctx.DB, nil)
	calendars, err := service.GetCalendars(ctx.UserID)
	if err != nil {
		return nil, fmt.Errorf("failed to get calendars: %v", err)
	}

	var results []map[string]interface{}
	for _, cal := range calendars {
		results = append(results, StructToMap(cal))
	}

	return map[string]interface{}{
		"calendars": results,
		"total":     len(calendars),
	}, nil
}

func handleListExternalEvents(args map[string]interface{}, ctx *ToolContext) (map[string]interface{}, error) {
	startStr, err := getStringParam(args, "start")
	if err != nil {
		return nil, err
	}

	endStr, err := getStringParam(args, "end")
	if err != nil {
		return nil, err
	}

	// Parse ISO 8601 dates
	start, err := time.Parse(time.RFC3339, startStr)
	if err != nil {
		return nil, fmt.Errorf("invalid start date format: %w", err)
	}

	end, err := time.Parse(time.RFC3339, endStr)
	if err != nil {
		return nil, fmt.Errorf("invalid end date format: %w", err)
	}

	service := NewExternalEventService(ctx.DB, nil)
	events, total, err := service.GetEventsInRange(ctx.UserID, start, end, 100, 0)
	if err != nil {
		return nil, fmt.Errorf("failed to get events: %v", err)
	}

	var results []map[string]interface{}
	for _, event := range events {
		results = append(results, StructToMap(event))
	}

	return map[string]interface{}{
		"events": results,
		"total":  total,
	}, nil
}

func handleLinkEventToCard(args map[string]interface{}, ctx *ToolContext) (map[string]interface{}, error) {
	eventID, err := getIntParam(args, "event_id")
	if err != nil {
		return nil, err
	}

	cardPK, err := getIntParam(args, "card_pk")
	if err != nil {
		return nil, err
	}

	service := NewExternalEventService(ctx.DB, nil)
	event, err := service.LinkEventToCard(ctx.DB, ctx.UserID, eventID, cardPK)
	if err != nil {
		return nil, fmt.Errorf("failed to link event to card: %v", err)
	}

	return StructToMap(event), nil
}
