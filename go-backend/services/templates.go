// Package services provides business logic and tool implementations for Zettelgarden.
//
// PHASE 3: Domain Package Migration
// ----------------------------------
// This file provides backward-compatible re-exports from the template domain package.
// External code can continue using services.GetTemplate(), services.GetTemplates(),
// and services.GetNextChildCardID() while the implementation is delegated to
// the services/tools/template domain package.
package services

import (
	"database/sql"

	"go-backend/models"
	"go-backend/services/tools/template"
)

// GetTemplate retrieves a template by ID for a specific user.
// Delegates to the template domain package.
func GetTemplate(db *sql.DB, userID, templateID int) (models.CardTemplate, error) {
	return template.GetTemplate(db, userID, templateID)
}

// GetTemplates retrieves all templates for a specific user.
// Delegates to the template domain package.
func GetTemplates(db *sql.DB, userID int) ([]models.CardTemplate, error) {
	return template.GetTemplates(db, userID)
}

// GetNextChildCardID returns the next available child card ID for a parent card.
// Delegates to the template domain package.
func GetNextChildCardID(db *sql.DB, userID int, parentID int) (string, error) {
	return template.GetNextChildCardID(db, userID, parentID)
}
