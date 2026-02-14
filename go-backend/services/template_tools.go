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
// controlled by the FeatureFlagTemplateTools feature flag.
//
// - Feature flag DISABLED (default): Uses legacy registration in this file
// - Feature flag ENABLED: Uses services/tools/template package
//
// To enable the new domain package:
//   export ZETTELGARDEN_FEATURE_TEMPLATE_TOOLS_V2=true
package services

import (
	"fmt"

	"go-backend/services/featureflags"
	"go-backend/services/tools"
	"go-backend/services/tools/template"
)

// RegisterTemplateTools registers all template-related tools.
//
// This method supports two registration paths controlled by feature flag:
// 1. Legacy path (default): Registers tools directly from this file
// 2. New path (feature flag): Delegates to services/tools/template package
func (tr *ToolRegistry) RegisterTemplateTools() {
	if featureflags.IsEnabled(featureflags.FeatureFlagTemplateTools) {
		// NEW: Use the domain package
		tr.registerTemplateToolsV2()
	} else {
		// LEGACY: Use the original registration in this file
		tr.registerTemplateToolsLegacy()
	}
}

// registerTemplateToolsV2 uses the new template domain package
func (tr *ToolRegistry) registerTemplateToolsV2() {
	RegisterTool(tr, "get_template",
		"Get a specific template by its numeric ID. Returns the full template details including name, title, and body templates.",
		handleGetTemplateV2,
		ToolParam{Name: "template_id", Type: "integer", Required: true, Desc: "The numeric ID of the template to retrieve"},
	)

	RegisterTool(tr, "list_templates",
		"Get all templates for the current user. Templates are reusable card structures with variable substitution.",
		handleListTemplatesV2,
	)

	RegisterTool(tr, "get_next_child_id",
		"Get the next available child card ID for a parent card (e.g., '1a2.3'). This is useful for creating structured card hierarchies.",
		handleGetNextChildIDV2,
		ToolParam{Name: "card_pk", Type: "integer", Required: true, Desc: "The primary key ID of the parent card"},
	)
}

// registerTemplateToolsLegacy is the original template tools registration.
// Kept for backward compatibility and as a fallback.
func (tr *ToolRegistry) registerTemplateToolsLegacy() {
	RegisterTool(tr, "get_template",
		"Get a specific template by its numeric ID. Returns the full template details including name, title, and body templates.",
		handleGetTemplateLegacy,
		ToolParam{Name: "template_id", Type: "integer", Required: true, Desc: "The numeric ID of the template to retrieve"},
	)

	RegisterTool(tr, "list_templates",
		"Get all templates for the current user. Templates are reusable card structures with variable substitution.",
		handleListTemplatesLegacy,
	)

	RegisterTool(tr, "get_next_child_id",
		"Get the next available child card ID for a parent card (e.g., '1a2.3'). This is useful for creating structured card hierarchies.",
		handleGetNextChildIDLegacy,
		ToolParam{Name: "card_pk", Type: "integer", Required: true, Desc: "The primary key ID of the parent card"},
	)
}

// V2 template tool handlers (use domain package logic)

func handleGetTemplateV2(args map[string]interface{}, ctx *ToolContext) (map[string]interface{}, error) {
	templateID, err := getIntParam(args, "template_id")
	if err != nil {
		return nil, err
	}

	tmpl, err := template.GetTemplate(ctx.DB, ctx.UserID, templateID)
	if err != nil {
		return nil, fmt.Errorf("failed to get template: %v", err)
	}

	return StructToMap(tmpl), nil
}

func handleListTemplatesV2(args map[string]interface{}, ctx *ToolContext) (map[string]interface{}, error) {
	templates, err := template.GetTemplates(ctx.DB, ctx.UserID)
	if err != nil {
		return nil, fmt.Errorf("failed to get templates: %v", err)
	}

	var results []map[string]interface{}
	for _, tmpl := range templates {
		results = append(results, StructToMap(tmpl))
	}

	data := map[string]interface{}{
		"templates": results,
	}
	metadata := tools.NewMetadata(tools.WithTotal(len(templates)))
	return tools.WrapToolSuccessWithMetadata(data, metadata), nil
}

func handleGetNextChildIDV2(args map[string]interface{}, ctx *ToolContext) (map[string]interface{}, error) {
	cardPK, err := getIntParam(args, "card_pk")
	if err != nil {
		return nil, err
	}

	nextID, err := template.GetNextChildCardID(ctx.DB, ctx.UserID, cardPK)
	if err != nil {
		return nil, fmt.Errorf("failed to get next child ID: %v", err)
	}

	if nextID == "" {
		return map[string]interface{}{
			"error":   true,
			"message": "Parent card not found or error occurred",
			"new_id":  "",
		}, nil
	}

	return map[string]interface{}{
		"error":   false,
		"message": "",
		"new_id":  nextID,
	}, nil
}

// Legacy template tool handlers (kept for backward compatibility)

func handleGetTemplateLegacy(args map[string]interface{}, ctx *ToolContext) (map[string]interface{}, error) {
	templateID, err := getIntParam(args, "template_id")
	if err != nil {
		return nil, err
	}

	tmpl, err := template.GetTemplate(ctx.DB, ctx.UserID, templateID)
	if err != nil {
		return nil, fmt.Errorf("failed to get template: %v", err)
	}

	return StructToMap(tmpl), nil
}

func handleListTemplatesLegacy(args map[string]interface{}, ctx *ToolContext) (map[string]interface{}, error) {
	templates, err := template.GetTemplates(ctx.DB, ctx.UserID)
	if err != nil {
		return nil, fmt.Errorf("failed to get templates: %v", err)
	}

	var results []map[string]interface{}
	for _, tmpl := range templates {
		results = append(results, StructToMap(tmpl))
	}

	data := map[string]interface{}{
		"templates": results,
	}
	metadata := tools.NewMetadata(tools.WithTotal(len(templates)))
	return tools.WrapToolSuccessWithMetadata(data, metadata), nil
}

func handleGetNextChildIDLegacy(args map[string]interface{}, ctx *ToolContext) (map[string]interface{}, error) {
	cardPK, err := getIntParam(args, "card_pk")
	if err != nil {
		return nil, err
	}

	nextID, err := template.GetNextChildCardID(ctx.DB, ctx.UserID, cardPK)
	if err != nil {
		return nil, fmt.Errorf("failed to get next child ID: %v", err)
	}

	if nextID == "" {
		return map[string]interface{}{
			"error":   true,
			"message": "Parent card not found or error occurred",
			"new_id":  "",
		}, nil
	}

	return map[string]interface{}{
		"error":   false,
		"message": "",
		"new_id":  nextID,
	}, nil
}
