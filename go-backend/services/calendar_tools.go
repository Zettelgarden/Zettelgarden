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
//
// PHASE 3: Domain Package Migration with Feature Flags
// ---------------------------------------------------
// This file now supports both legacy and new domain package registration
// controlled by the FeatureFlagCalendarTools feature flag.
//
// - Feature flag DISABLED (default): Uses legacy registration in this file
// - Feature flag ENABLED: Uses services/tools/calendar package
//
// To enable the new domain package:
//   export ZETTELGARDEN_FEATURE_CALENDAR_TOOLS_V2=true
package services

import (
	"fmt"
	"time"

	"go-backend/services/featureflags"
	"go-backend/services/tools/calendar"
)

// RegisterCalendarTools registers all calendar-related tools.
//
// This method supports two registration paths controlled by feature flag:
// 1. Legacy path (default): Registers tools directly from this file
// 2. New path (feature flag): Delegates to services/tools/calendar package
func (tr *ToolRegistry) RegisterCalendarTools() {
	if featureflags.IsEnabled(featureflags.FeatureFlagCalendarTools) {
		// NEW: Use the domain package
		tr.registerCalendarToolsV2()
	} else {
		// LEGACY: Use the original registration in this file
		tr.registerCalendarToolsLegacy()
	}
}

// registerCalendarToolsV2 uses the new calendar domain package
func (tr *ToolRegistry) registerCalendarToolsV2() {
	RegisterTool(tr, "list_external_calendars",
		"List all external calendar subscriptions for the current user. Returns calendar names, URLs, sync status, colors, and last sync times.",
		handleListExternalCalendarsV2,
	)

	RegisterTool(tr, "list_external_events",
		"List external calendar events within a date range. Returns events with titles, times, locations, linked cards, and descriptions.",
		handleListExternalEventsV2,
		ToolParam{Name: "start", Type: "string", Required: true, Desc: "Start date in ISO 8601 format (e.g., '2026-01-01T00:00:00Z')"},
		ToolParam{Name: "end", Type: "string", Required: true, Desc: "End date in ISO 8601 format (e.g., '2026-12-31T23:59:59Z')"},
	)

	RegisterTool(tr, "link_event_to_card",
		"Link an external calendar event to a card. Creates a bidirectional association between the event and card.",
		handleLinkEventToCardV2,
		ToolParam{Name: "event_id", Type: "integer", Required: true, Desc: "The external event ID"},
		ToolParam{Name: "card_pk", Type: "integer", Required: true, Desc: "The card primary key to link to"},
	)
}

// registerCalendarToolsLegacy is the original calendar tools registration.
// Kept for backward compatibility and as a fallback.
func (tr *ToolRegistry) registerCalendarToolsLegacy() {
	RegisterTool(tr, "list_external_calendars",
		"List all external calendar subscriptions for the current user. Returns calendar names, URLs, sync status, colors, and last sync times.",
		handleListExternalCalendarsLegacy,
	)

	RegisterTool(tr, "list_external_events",
		"List external calendar events within a date range. Returns events with titles, times, locations, linked cards, and descriptions.",
		handleListExternalEventsLegacy,
		ToolParam{Name: "start", Type: "string", Required: true, Desc: "Start date in ISO 8601 format (e.g., '2026-01-01T00:00:00Z')"},
		ToolParam{Name: "end", Type: "string", Required: true, Desc: "End date in ISO 8601 format (e.g., '2026-12-31T23:59:59Z')"},
	)

	RegisterTool(tr, "link_event_to_card",
		"Link an external calendar event to a card. Creates a bidirectional association between the event and card.",
		handleLinkEventToCardLegacy,
		ToolParam{Name: "event_id", Type: "integer", Required: true, Desc: "The external event ID"},
		ToolParam{Name: "card_pk", Type: "integer", Required: true, Desc: "The card primary key to link to"},
	)
}

// V2 calendar tool handlers (use domain package logic)

func handleListExternalCalendarsV2(args map[string]interface{}, ctx *ToolContext) (map[string]interface{}, error) {
	calendars, err := calendar.GetCalendars(ctx.DB, ctx.UserID)
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

func handleListExternalEventsV2(args map[string]interface{}, ctx *ToolContext) (map[string]interface{}, error) {
	startStr, err := getStringParam(args, "start")
	if err != nil {
		return nil, err
	}

	endStr, err := getStringParam(args, "end")
	if err != nil {
		return nil, err
	}

	// Parse ISO 8601 dates
	start, err := calendar.ParseISODate(startStr)
	if err != nil {
		return nil, fmt.Errorf("invalid start date format: %w", err)
	}

	end, err := calendar.ParseISODate(endStr)
	if err != nil {
		return nil, fmt.Errorf("invalid end date format: %w", err)
	}

	// Validate date range
	if err := calendar.ValidateDateRange(start, end); err != nil {
		return nil, err
	}

	events, total, err := calendar.GetEventsInRange(ctx.DB, ctx.UserID, start, end, 100, 0)
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

func handleLinkEventToCardV2(args map[string]interface{}, ctx *ToolContext) (map[string]interface{}, error) {
	eventID, err := getIntParam(args, "event_id")
	if err != nil {
		return nil, err
	}

	cardPK, err := getIntParam(args, "card_pk")
	if err != nil {
		return nil, err
	}

	event, err := calendar.LinkEventToCard(ctx.DB, ctx.UserID, eventID, cardPK)
	if err != nil {
		return nil, fmt.Errorf("failed to link event to card: %v", err)
	}

	return StructToMap(event), nil
}

// Legacy calendar tool handlers (kept for backward compatibility)

func handleListExternalCalendarsLegacy(args map[string]interface{}, ctx *ToolContext) (map[string]interface{}, error) {
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

func handleListExternalEventsLegacy(args map[string]interface{}, ctx *ToolContext) (map[string]interface{}, error) {
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

func handleLinkEventToCardLegacy(args map[string]interface{}, ctx *ToolContext) (map[string]interface{}, error) {
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
