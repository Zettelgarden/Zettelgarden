package services

import (
	"database/sql"
	"fmt"
	"go-backend/models"
	"log"
)

// Allowed values for the enum-like columns. Kept in sync with the frontend
// types in src/types/taskPage.ts (SortField, SortDirection, ViewMode).
var (
	allowedTaskSortFields = map[string]bool{
		"updated_at": true, "title": true, "priority": true, "status": true,
		"id": true, "scheduled_date": true, "manual": true,
	}
	allowedTaskSortDirections = map[string]bool{"asc": true, "desc": true}
	allowedTaskViewModes      = map[string]bool{"list": true, "matrix": true, "kanban": true}
)

// validateTaskSavedSearchFields returns an error if any enum field is invalid.
func validateTaskSavedSearchFields(sortField, sortDirection, viewMode string) error {
	if !allowedTaskSortFields[sortField] {
		return fmt.Errorf("invalid sort_field: %q", sortField)
	}
	if !allowedTaskSortDirections[sortDirection] {
		return fmt.Errorf("invalid sort_direction: %q", sortDirection)
	}
	if !allowedTaskViewModes[viewMode] {
		return fmt.Errorf("invalid view_mode: %q", viewMode)
	}
	return nil
}

// scanTaskSavedSearch maps a row to a TaskSavedSearch.
const taskSavedSearchSelectColumns = `id, user_id, name, filter_string, sort_field, sort_direction, view_mode, created_at, updated_at`

func scanTaskSavedSearch(scanner interface{ Scan(...interface{}) error }, s *models.TaskSavedSearch) error {
	return scanner.Scan(
		&s.ID, &s.UserID, &s.Name, &s.FilterString, &s.SortField,
		&s.SortDirection, &s.ViewMode, &s.CreatedAt, &s.UpdatedAt,
	)
}

// GetTaskSavedSearches retrieves all saved task searches for a user, newest first.
func GetTaskSavedSearches(db models.Database, userID int) ([]models.TaskSavedSearch, error) {
	query := `
		SELECT ` + taskSavedSearchSelectColumns + `
		FROM task_saved_searches
		WHERE user_id = $1
		ORDER BY created_at DESC
	`
	rows, err := db.Query(query, userID)
	if err != nil {
		log.Printf("Error querying task saved searches: %v", err)
		return nil, err
	}
	defer rows.Close()

	var searches []models.TaskSavedSearch
	for rows.Next() {
		var s models.TaskSavedSearch
		if err := scanTaskSavedSearch(rows, &s); err != nil {
			log.Printf("Error scanning task saved search: %v", err)
			return nil, err
		}
		searches = append(searches, s)
	}
	return searches, rows.Err()
}

// GetTaskSavedSearch retrieves a single saved task search by ID, scoped to the user.
func GetTaskSavedSearch(db models.Database, userID int, searchID int) (models.TaskSavedSearch, error) {
	var s models.TaskSavedSearch
	query := `
		SELECT ` + taskSavedSearchSelectColumns + `
		FROM task_saved_searches
		WHERE id = $1 AND user_id = $2
	`
	err := db.QueryRow(query, searchID, userID).Scan(
		&s.ID, &s.UserID, &s.Name, &s.FilterString, &s.SortField,
		&s.SortDirection, &s.ViewMode, &s.CreatedAt, &s.UpdatedAt,
	)
	if err != nil {
		if err == sql.ErrNoRows {
			return models.TaskSavedSearch{}, fmt.Errorf("task saved search not found")
		}
		log.Printf("Error getting task saved search: %v", err)
		return models.TaskSavedSearch{}, err
	}
	return s, nil
}

// CreateTaskSavedSearch creates a new saved task search and returns its ID.
func CreateTaskSavedSearch(db models.Database, userID int, params models.CreateTaskSavedSearchParams) (int, error) {
	if err := validateTaskSavedSearchFields(params.SortField, params.SortDirection, params.ViewMode); err != nil {
		return 0, err
	}

	var id int
	query := `
		INSERT INTO task_saved_searches (user_id, name, filter_string, sort_field, sort_direction, view_mode)
		VALUES ($1, $2, $3, $4, $5, $6)
		RETURNING id
	`
	err := db.QueryRow(query, userID, params.Name, params.FilterString,
		params.SortField, params.SortDirection, params.ViewMode).Scan(&id)
	if err != nil {
		log.Printf("Error creating task saved search: %v", err)
		return 0, err
	}
	return id, nil
}

// UpdateTaskSavedSearch updates an existing saved task search, scoped to the user.
func UpdateTaskSavedSearch(db models.Database, userID int, searchID int, params models.UpdateTaskSavedSearchParams) error {
	// Ensure the search exists and belongs to the user.
	if _, err := GetTaskSavedSearch(db, userID, searchID); err != nil {
		return err
	}

	// Validate any provided enum fields.
	if params.SortField != nil && !allowedTaskSortFields[*params.SortField] {
		return fmt.Errorf("invalid sort_field: %q", *params.SortField)
	}
	if params.SortDirection != nil && !allowedTaskSortDirections[*params.SortDirection] {
		return fmt.Errorf("invalid sort_direction: %q", *params.SortDirection)
	}
	if params.ViewMode != nil && !allowedTaskViewModes[*params.ViewMode] {
		return fmt.Errorf("invalid view_mode: %q", *params.ViewMode)
	}

	query := `UPDATE task_saved_searches SET updated_at = NOW()`
	args := []interface{}{}
	argCount := 1

	add := func(value interface{}, column string) {
		query += fmt.Sprintf(`, %s = $%d`, column, argCount)
		args = append(args, value)
		argCount++
	}

	if params.Name != nil {
		add(*params.Name, "name")
	}
	if params.FilterString != nil {
		add(*params.FilterString, "filter_string")
	}
	if params.SortField != nil {
		add(*params.SortField, "sort_field")
	}
	if params.SortDirection != nil {
		add(*params.SortDirection, "sort_direction")
	}
	if params.ViewMode != nil {
		add(*params.ViewMode, "view_mode")
	}

	query += fmt.Sprintf(` WHERE id = $%d AND user_id = $%d`, argCount, argCount+1)
	args = append(args, searchID, userID)

	if _, err := db.Exec(query, args...); err != nil {
		log.Printf("Error updating task saved search: %v", err)
		return err
	}
	return nil
}

// DeleteTaskSavedSearch deletes a saved task search, scoped to the user.
// Returns sql.ErrNoRows if no row was deleted.
func DeleteTaskSavedSearch(db models.Database, userID int, searchID int) error {
	result, err := db.Exec(
		`DELETE FROM task_saved_searches WHERE id = $1 AND user_id = $2`,
		searchID, userID,
	)
	if err != nil {
		log.Printf("Error deleting task saved search: %v", err)
		return err
	}
	rowsAffected, err := result.RowsAffected()
	if err != nil {
		return err
	}
	if rowsAffected == 0 {
		return sql.ErrNoRows
	}
	return nil
}
