# Code Review Guidelines

This document outlines guidelines for code review in the Zettelgarden project.

## Handler-Service Separation

### Principle: Handlers must not make database queries

**Handlers** (`handlers/`) are responsible for:
- HTTP request/response handling
- Request validation and authentication
- Calling service functions for business logic
- Returning appropriate HTTP status codes and responses

**Services** (`services/`) are responsible for:
- Business logic
- Database queries and operations
- Data transformations

### What goes in handlers?

✅ **Handler functions should:**
- Extract user context from requests (`r.Context().Value("current_user")`)
- Parse request bodies and path variables
- Validate input data
- Call service functions
- Return HTTP responses (JSON, status codes, errors)

✅ **Example handler (correct):**
```go
func (s *Handler) GetTagsRoute(w http.ResponseWriter, r *http.Request) {
    userID := r.Context().Value("current_user").(int)

    tags, err := services.QueryTags(s.GetDB(), userID)
    if err != nil {
        http.Error(w, err.Error(), http.StatusBadRequest)
        return
    }

    w.Header().Set("Content-Type", "application/json")
    json.NewEncoder(w).Encode(tags)
}
```

### What goes in services?

✅ **Service functions should:**
- Accept a `models.Database` interface as first parameter
- Contain all database query logic (Query, QueryRow, Exec)
- Perform business logic and data transformations
- Return domain models or errors

✅ **Example service function (correct):**
```go
func QueryTags(db models.Database, userID int) ([]models.Tag, error) {
    tags := []models.Tag{}
    query := `SELECT id, name, user_id, color FROM tags
              WHERE is_deleted = false AND user_id = $1`

    rows, err := db.Query(query, userID)
    if err != nil {
        return tags, err
    }
    defer rows.Close()

    for rows.Next() {
        var tag models.Tag
        if err := rows.Scan(&tag.ID, &tag.Name, &tag.UserID, &tag.Color); err != nil {
            return tags, err
        }
        tags = append(tags, tag)
    }
    return tags, nil
}
```

### Code Review Checklist

When reviewing code that touches the database, verify:

- [ ] No `.Query()`, `.QueryRow()`, or `.Exec()` calls in `handlers/*.go`
- [ ] All database operations use `s.GetDB()` to support test transactions
- [ ] Database queries are in `services/*.go` with appropriate error handling
- [ ] Service functions accept `models.Database` (not `*sql.DB`) as first parameter
- [ ] `defer rows.Close()` is called after any `db.Query()` call

### Migration Pattern

When moving DB queries from handlers to services:

1. Create a new function in `services/` that accepts `models.Database` as first parameter
2. Move the query logic from handler to the service function
3. Update the handler to call `s.GetDB()` and pass to the service function
4. Update any tests that call the old handler function to use the new service function
5. Run tests to verify behavior is unchanged

### Rationale

This separation provides:

- **Testability:** Service functions can be tested with a mock database interface without setting up full HTTP handlers
- **Reusability:** Business logic can be called from multiple handlers or contexts
- **Maintainability:** Clear separation of concerns makes code easier to understand and modify
- **Transaction Support:** Using `s.GetDB()` ensures test transactions work correctly
