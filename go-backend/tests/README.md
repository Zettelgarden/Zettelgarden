# Backend Testing Guide

This guide explains how to write tests for the Zettelgarden Go backend.

## Table of Contents

- [Quick Start](#quick-start)
- [Test Infrastructure](#test-infrastructure)
- [Transaction-per-Test Pattern](#transaction-per-test-pattern)
- [Test Types](#test-types)
- [Writing Handler Tests](#writing-handler-tests)
- [Writing Service Tests](#writing-service-tests)
- [Common Patterns](#common-patterns)
- [Test Checklist](#test-checklist)

## Quick Start

```go
func TestMyFeature(t *testing.T) {
    s := tests.Setup()          // Initialize test environment
    defer tests.Teardown()      // Cleanup (always use defer!)

    userID := 1
    // ... your test code here
}
```

## Test Infrastructure

### Key Files

- `tests/conftest.go` - Test setup, teardown, and helper functions
- `tests/examples_test.go` - Example test patterns (copy these!)
- `*_test.go` - Test files in each package (e.g., `handlers/cards_test.go`)

### Core Functions

| Function | Purpose |
|----------|---------|
| `tests.Setup()` | Initialize test environment, create transaction |
| `tests.Teardown()` | Rollback transaction, clean up |
| `tests.GenerateTestJWT(userID)` | Generate auth token for user |
| `tests.CreateJsonBody(t, obj)` | Create JSON request body |
| `tests.ParseJsonResponse(t, body, &obj)` | Parse JSON response |

### Test Fixtures

The following test data is loaded automatically:
- **Users**: 3 users (ID 1 is admin, ID 2 is `test@test.com`)
- **Cards**: 14 cards with various patterns
- **Files**: 5 files
- **Tasks**: 5 tasks
- **Tags**: 3 tags
- **Entities**: 3 entities
- **Backlinks**: 3 backlinks

See `conftest.go:generateData()` for fixture details.

## Transaction-per-Test Pattern

This framework uses a **transaction-per-test pattern** for test isolation:

```
┌─────────────────────────────────────────────────────────────┐
│  Test Suite Start                                           │
│  ├─ Connect to database                                     │
│  ├─ Run migrations (once)                                   │
│  └─ Load test fixtures (once)                               │
├─────────────────────────────────────────────────────────────┤
│  Test 1                                                     │
│  ├─ Setup(): BEGIN TRANSACTION                             │
│  ├─ Run test (all DB operations use transaction)           │
│  └─ Teardown(): ROLLBACK (discard all changes)             │
├─────────────────────────────────────────────────────────────┤
│  Test 2                                                     │
│  ├─ Setup(): BEGIN TRANSACTION (fresh, clean state)        │
│  ├─ Run test                                               │
│  └─ Teardown(): ROLLACK                                    │
├─────────────────────────────────────────────────────────────┤
│  ...                                                       │
└─────────────────────────────────────────────────────────────┘
```

**Key Benefits:**
- Each test starts with clean, identical data
- Tests run quickly (no table drops/recreates)
- Tests never affect production database
- Tests can run sequentially without interference

**Important:**
- Always call `defer tests.Teardown()` immediately after `tests.Setup()`
- Never use `h.DB` directly - use `h.GetDB()` which returns the test transaction
- Tests run with `parallel=1` to avoid database conflicts

## Test Types

### 1. Handler Tests

Test HTTP endpoints, middleware, authentication, and request/response handling.

**Use when:** Testing routes, request validation, auth, error responses.

**File location:** `handlers/<feature>_test.go`

### 2. Service Tests

Test business logic, database operations, and service functions.

**Use when:** Testing algorithms, data processing, database queries.

**File location:** `services/<feature>_test.go`

### 3. Integration Tests

Test end-to-end flows across multiple layers.

**Use when:** Testing complex workflows, feature interactions.

**File location:** Can be in either `handlers/` or `services/`

## Writing Handler Tests

Handler tests simulate HTTP requests to your routes.

### Pattern

```go
func TestGetCardRoute(t *testing.T) {
    // Setup
    h := handlers.NewHandler()
    defer tests.Teardown()

    // Generate auth token
    token, _ := tests.GenerateTestJWT(1)

    // Create request
    body := tests.CreateJsonBody(t, map[string]string{
        "title": "Test Card",
        "body": "Test content",
    })
    req, _ := http.NewRequest("POST", "/api/cards", body)
    req.Header.Set("Authorization", "Bearer "+token)
    req.Header.Set("Content-Type", "application/json")

    // Execute request
    rr := httptest.NewRecorder()
    router := mux.NewRouter()
    router.HandleFunc("/api/cards", h.JwtMiddleware(h.CreateCardRoute))
    router.ServeHTTP(rr, req)

    // Check response
    if status := rr.Code; status != http.StatusCreated {
        t.Errorf("expected status %d, got %d", http.StatusCreated, status)
    }

    var response models.Card
    tests.ParseJsonResponse(t, rr.Body.Bytes(), &response)

    if response.Title != "Test Card" {
        t.Errorf("expected title 'Test Card', got '%s'", response.Title)
    }
}
```

### Handler Test Checklist

- [ ] Use `handlers.NewHandler()` for setup
- [ ] Add authentication header with `GenerateTestJWT()`
- [ ] Set Content-Type for POST/PUT requests
- [ ] Test both success and error cases
- [ ] Test authorization (wrong user, no auth)
- [ ] Check HTTP status codes
- [ ] Parse and validate response body

## Writing Service Tests

Service tests call service functions directly without HTTP layer.

### Pattern

```go
func TestCreateCard(t *testing.T) {
    // Setup
    s := tests.Setup()
    defer tests.Teardown()

    userID := 1
    params := models.EditCardParams{
        Title:  "Test Card",
        Body:   "Test content with [backlink]",
        CardID: "test123",
    }

    // Execute
    card, err := services.CreateCard(s.DB, userID, params)
    if err != nil {
        t.Fatalf("CreateCard failed: %v", err)
    }

    // Verify
    if card.Title != params.Title {
        t.Errorf("expected title %v, got %v", params.Title, card.Title)
    }

    if card.CardID != params.CardID {
        t.Errorf("expected card_id %v, got %v", params.CardID, card.CardID)
    }
}
```

### Service Test Checklist

- [ ] Use `tests.Setup()` (not `handlers.NewHandler()`)
- [ ] Use `s.DB` (the test database connection)
- [ ] Test error cases (invalid input, not found, etc.)
- [ ] Verify database state after operations
- [ ] Test edge cases (empty input, duplicates, etc.)

## Common Patterns

### Authentication

```go
// Generate token for user ID 1
token, _ := tests.GenerateTestJWT(1)
req.Header.Set("Authorization", "Bearer "+token)

// Test without authentication
req, _ := http.NewRequest("GET", "/api/cards/1", nil)
// No auth header - should return 401
```

### Creating Test Data

```go
// Use service functions
card, _ := services.CreateCard(s.DB, userID, params)

// Or direct SQL
_, err := s.DB.Exec(`
    INSERT INTO cards (card_id, user_id, title, body)
    VALUES ($1, $2, $3, $4)
`, "test123", userID, "Test", "Content")
```

### Testing Errors

```go
// Test that error is returned
_, err := services.CreateCard(s.DB, userID, invalidParams)
if err == nil {
    t.Error("expected error for invalid params, got nil")
}

// Test specific error message
if !strings.Contains(err.Error(), "title is required") {
    t.Errorf("unexpected error message: %v", err)
}
```

### Table-Driven Tests

```go
func TestValidateCardID(t *testing.T) {
    tests := []struct {
        name    string
        cardID  string
        wantErr bool
    }{
        {"valid", "ABC123", false},
        {"empty", "", true},
        {"too long", strings.Repeat("A", 100), true},
    }

    for _, tt := range tests {
        t.Run(tt.name, func(t *testing.T) {
            err := services.ValidateCardID(tt.cardID)
            if (err != nil) != tt.wantErr {
                t.Errorf("ValidateCardID() error = %v, wantErr %v", err, tt.wantErr)
            }
        })
    }
}
```

### Subtests

```go
func TestCardOperations(t *testing.T) {
    t.Run("create", func(t *testing.T) {
        // Test card creation
    })

    t.Run("update", func(t *testing.T) {
        // Test card update
    })

    t.Run("delete", func(t *testing.T) {
        // Test card deletion
    })
}
```

## Handler.GetDB() vs Handler.DB

**CRITICAL:** When writing handler tests, always use `h.GetDB()` not `h.DB`.

```go
// WRONG - bypasses test transaction
func (h *Handler) MyFunction() error {
    row := h.DB.QueryRow("SELECT ...")  // ❌ Uses real DB
}

// CORRECT - uses test transaction during testing
func (h *Handler) MyFunction() error {
    db := h.GetDB()  // ✅ Returns test transaction when testing
    row := db.QueryRow("SELECT ...")
}
```

**Why:** `h.GetDB()` returns `h.Tx` (test transaction) during testing, but `h.DB` (production connection) otherwise. This is the magic that makes the transaction-per-test pattern work.

## Test Checklist

Before marking a test as complete, verify:

### Setup/Teardown
- [ ] `tests.Setup()` or `handlers.NewHandler()` called first
- [ ] `defer tests.Teardown()` on next line
- [ ] No early returns that skip Teardown

### Test Coverage
- [ ] Success case tested
- [ ] Error cases tested (invalid input, not found, unauthorized)
- [ ] Edge cases tested (empty input, duplicates, boundary values)
- [ ] Authorization tested (wrong user, no auth, admin-only)

### Assertions
- [ ] Status code checked (for handler tests)
- [ ] Response body parsed and validated
- [ ] Error messages checked when appropriate
- [ ] Database state verified after operations

### Naming
- [ ] Test name describes what is being tested
- [ ] Subtests used for related cases
- [ ] Test function starts with `Test`

### Clean Code
- [ ] No hardcoded IDs that might conflict with fixtures
- [ ] Test data is unique (use random values or timestamps)
- [ ] No `t.Log()` statements in test logic (use for debugging only)

## Running Tests

```bash
# Run all tests
go test ./...

# Run tests in specific package
go test ./handlers/...
go test ./services/...

# Run specific test
go test ./handlers -run TestGetCard

# Run with verbose output
go test -v ./handlers

# Run with coverage
go test -cover ./...
```

## Resources

- [Go Testing Guide](https://golang.org/pkg/testing/)
- [Table-Driven Tests](https://dave.cheney.net/2019/05/07/prefer-table-driven-tests)
- [Testify](https://github.com/stretchr/testify) - Optional assertion library
