# Testability & Global State Review: Zettelgarden Backend

**Bead:** Zettelgarden-9xw.8
**Date:** 2026-01-15
**Scope:** Assess testability impact of global variables and monolithic main() in router initialization

---

## Executive Summary

The Zettelgarden backend has **significant testability barriers** caused by:

1. **Package-level globals** (`s`, `h`) in `main.go` that tightly couple router registration to application startup
2. **Monolithic main() function** (249 lines) that mixes configuration, dependency initialization, and routing in a single sequential block
3. **Router registration locked in main()** with no way to test the complete router+middleware stack without starting the HTTP server
4. **Implicit dependencies** embedded in middleware closures, making unit tests brittle and requiring full server setup

**Good news:** The codebase already has strong test infrastructure (`tests/conftest.go`, handler tests with `httptest`), but it **cannot test router composition, middleware stacking, or integration behavior** without refactoring the entrypoint.

---

## Current Pattern Analysis

### 1. Global State in main.go (Lines 22-24)

```go
var s *server.Server
var h *handlers.Handler
```

**Problems:**
- These globals are **only initialized in main()**, making them inaccessible to test code
- The entire routing logic depends on `h` being non-nil
- Cannot test route ordering, middleware precedence, or collision detection
- Impossible to test router composition in isolation

### 2. Monolithic main() Function (Lines 35-249)

The main() function performs:

```
1. Logging setup (lines 36-43)
2. Bootstrap server + DB (lines 45-51)
3. Stripe initialization (lines 53-54)
4. S3 client creation (line 56)
5. Mail client setup (lines 58-63)
6. Typesense initialization (lines 65-72) - async with goroutine
7. JWT secret key loading (line 74)
8. LLM client config (lines 75-76)
9. Background chunking/embedding task (lines 78-86) - conditional async
10. Router creation + 138 route registrations (lines 88-231)
11. CORS configuration (lines 233-240)
12. Server startup (line 248)
```

**Problems:**
- **No separation of concerns** - Configuration, initialization, routing, and startup are interleaved
- **Hard to test individual components** - Cannot configure a router without a full server
- **Middleware testing requires server state** - Tests must use `httptest` mini-routers (see handlers/cards_test.go:31-33)
- **Difficult to add new routes** - Must manually add to main() with correct middleware combination
- **Async initialization not properly tracked** - Typesense and embedding tasks start without coordination

### 3. Route Registration Pattern (Lines 27-33)

```go
func addProtectedRoute(r *mux.Router, path string, handler http.HandlerFunc, method string) *mux.Route {
    return r.HandleFunc(path, h.APIKeyOrJWTMiddleware(handlers.LogRoute(handler))).Methods(method)
}

func addRoute(r *mux.Router, path string, handler http.HandlerFunc, method string) *mux.Route {
    return r.HandleFunc(path, handlers.LogRoute(handler)).Methods(method)
}
```

**Problems:**
- **Middleware closures over global `h`** - Functions cannot be unit tested without global state
- **No abstraction for middleware composition** - Each route manually nests middleware
- **Duplicate middleware logic** - Same ordering pattern repeated 138+ times
- **No way to register routes programmatically** - Cannot build routes from configuration or data structures

### 4. Current Test Approach (handlers/cards_test.go:19-36)

```go
func makeCardRequestSuccess(s *Handler, t *testing.T, id int) *httptest.ResponseRecorder {
    token, _ := tests.GenerateTestJWT(1)
    req, err := http.NewRequest("GET", "/api/cards/"+strconv.Itoa(id), nil)
    // ... setup ...

    rr := httptest.NewRecorder()
    router := mux.NewRouter()
    router.HandleFunc("/api/cards/{id}", s.JwtMiddleware(s.GetCardRoute))
    router.ServeHTTP(rr, req)
    return rr
}
```

**Current Approach:**
- Creates **ad-hoc mini-routers** for each test
- Manually composes the expected middleware chain
- **Cannot test router discovery or integration**
- **Brittle** - must keep test middleware in sync with main.go

---

## Obstacles to Testing

### Obstacle 1: Router Composition Testing
**Cannot test:** Whether all routes are registered, middleware ordering, route collision detection
**Reason:** Router built in main(), not accessible to test code
**Example:** If someone adds a new route with wrong middleware, tests won't catch it

### Obstacle 2: Integration Testing
**Cannot test:** Full request flow through CORS middleware → Auth middleware → Route handler
**Reason:** CORS wrapper applied only in main() after router creation
**Current workaround:** Tests skip CORS entirely, creating a divergence from production behavior

### Obstacle 3: Middleware Precedence Testing
**Cannot test:** Whether LogRoute runs before or after JwtMiddleware
**Cannot test:** Whether APIKeyOrJWTMiddleware properly falls back to JWT on API key failure
**Reason:** Middleware nesting happens in main() closures
**Impact:** Subtle middleware ordering bugs can only be found through integration tests or production

### Obstacle 4: Dependency Injection Flexibility
**Cannot test with:** Mock S3 client, stub mail service, or test database
**Reason:** Server struct is initialized in bootstrap, but S3/Mail added in main() with side effects
**Current workaround:** Handler tests require full environmental setup (see .env-bash)

### Obstacle 5: Configuration Testing
**Cannot test:** Different logging levels, different S3 configurations, optional services (Typesense)
**Reason:** Configuration scattered across main() with no separation
**Impact:** Cannot test degradation when Typesense is unavailable (current code swallows the error on line 66)

---

## Existing Test Infrastructure (Strengths)

### tests/conftest.go
- **Strong:** Provides `Setup()` function that initializes test database
- **Strong:** Generates comprehensive test data (users, cards, files, tasks, entities)
- **Strong:** Provides JWT token generation (`GenerateTestJWT`)
- **Strong:** Handles database teardown and reset
- **Usage:** All handler tests inherit this pattern

### Handler Tests Pattern (handlers/*_test.go)
- **Strong:** Uses `httptest.ResponseRecorder` and `httptest.Server`
- **Strong:** Tests are hermetic - no external dependencies
- **Strong:** Tests can be run in parallel with proper database reset
- **Example:** `TestGetCardSuccess` in cards_test.go (lines 57-86)

### Services Tests (services/*_test.go)
- **Strong:** Tests business logic independent of HTTP layer
- **Good:** Tests database queries directly
- **Example:** tasks_test.go, cards_test.go

---

## Recommended Structure for Refactoring

### Phase 1: Create Router Builder Function

**Goal:** Extract router registration from main() into a testable function

**Pattern:**
```go
// pkg/routes/builder.go
func BuildRouter(handler *handlers.Handler, opts RouterOptions) *mux.Router {
    r := mux.NewRouter()

    // Protected routes (require JWT or API key)
    registerProtectedRoutes(r, handler)

    // Public routes
    registerPublicRoutes(r, handler)

    return r
}
```

**Benefits:**
- Router can be created and inspected without starting server
- Middleware composition is testable
- Routes can be registered from configuration
- Easier to extend (add feature routes)

### Phase 2: Create Middleware Composition Layer

**Goal:** Replace inline middleware nesting with a composable system

**Pattern:**
```go
// pkg/middleware/chain.go
type HandlerChain struct {
    middlewares []func(http.HandlerFunc) http.HandlerFunc
}

func (c *HandlerChain) Add(mw func(http.HandlerFunc) http.HandlerFunc) *HandlerChain {
    c.middlewares = append(c.middlewares, mw)
    return c
}

func (c *HandlerChain) Build(handler http.HandlerFunc) http.HandlerFunc {
    // Apply in reverse order (last registered is outermost)
    for i := len(c.middlewares) - 1; i >= 0; i-- {
        handler = c.middlewares[i](handler)
    }
    return handler
}
```

**Usage in router:**
```go
protected := NewHandlerChain().
    Add(handlers.LogRoute).
    Add(h.APIKeyOrJWTMiddleware)

r.HandleFunc("/api/cards", protected.Build(h.CreateCardRoute)).Methods("POST")
```

**Benefits:**
- Middleware order is explicit and testable
- Reusable chains (protected, admin, public, etc.)
- Easy to add conditional middleware
- Tests can verify middleware composition

### Phase 3: Dependency Injection Structure

**Goal:** Separate dependency initialization from usage

**Pattern:**
```go
// pkg/bootstrap/dependencies.go
type Dependencies struct {
    DB        *sql.DB
    Server    *server.Server
    Handler   *handlers.Handler
    S3        *s3.Client
    Mail      *mail.MailClient
    Typesense *typesense.Client
    LLM       *models.LLMClient
}

func InitDependencies(ctx context.Context, config Config) (*Dependencies, error) {
    // Initialize in order, with error handling
    db, err := ConnectDB(config.DB)
    // ...
    return deps, nil
}
```

**Benefits:**
- Clear initialization order
- Error handling at dependency level
- Test code can create partial dependencies
- Easier to add new services

### Phase 4: Configuration Object

**Goal:** Centralize configuration loading

**Pattern:**
```go
// pkg/config/config.go
type Config struct {
    Server   ServerConfig
    Database DatabaseConfig
    Services ServiceConfig
    Features FeatureConfig
}

func LoadConfig() Config {
    return Config{
        Server:   loadServerConfig(),
        Database: loadDatabaseConfig(),
        Services: loadServiceConfig(),
        Features: loadFeatureConfig(),
    }
}
```

**Benefits:**
- Single source of truth for configuration
- Can be tested independently
- Easier to support different deployment modes

---

## Testing Patterns Enabled by Refactoring

### Test 1: Router Composition
```go
func TestRouterComposition(t *testing.T) {
    handler := createTestHandler(t)
    router := routes.BuildRouter(handler, DefaultRouterOptions())

    // Verify specific routes exist
    assert.NotNil(t, router.Get("/api/cards/{id}"))

    // Verify route methods
    req, _ := http.NewRequest("GET", "/api/cards/1", nil)
    rr := httptest.NewRecorder()
    router.ServeHTTP(rr, req)
    // Should fail auth, not 404
    assert.Equal(t, http.StatusUnauthorized, rr.Code)
}
```

### Test 2: Middleware Chain
```go
func TestProtectedRouteChain(t *testing.T) {
    handler := createTestHandler(t)

    chain := middleware.NewHandlerChain().
        Add(handlers.LogRoute).
        Add(handler.APIKeyOrJWTMiddleware)

    req, _ := http.NewRequest("GET", "/api/cards", nil)
    rr := httptest.NewRecorder()

    finalHandler := chain.Build(handler.GetCardsRoute)
    finalHandler.ServeHTTP(rr, req)

    assert.Equal(t, http.StatusUnauthorized, rr.Code)
}
```

### Test 3: Integration with CORS
```go
func TestRouterWithCORS(t *testing.T) {
    handler := createTestHandler(t)
    router := routes.BuildRouter(handler, DefaultRouterOptions())
    corsHandler := applyMiddleware.CORS(router)

    req, _ := http.NewRequest("OPTIONS", "/api/cards", nil)
    req.Header.Set("Origin", "http://localhost:3000")
    rr := httptest.NewRecorder()
    corsHandler.ServeHTTP(rr, req)

    assert.Equal(t, "http://localhost:3000", rr.Header().Get("Access-Control-Allow-Origin"))
}
```

### Test 4: Partial Dependency Injection
```go
func TestCardHandlerWithMockS3(t *testing.T) {
    mockS3 := &MockS3Client{}
    handler := &handlers.Handler{
        DB: testDB,
        Server: &server.Server{
            DB:     testDB,
            S3:     mockS3,
            Mail:   &mail.MailClient{Testing: true},
        },
    }

    req, _ := http.NewRequest("POST", "/api/files/upload", fileBody)
    req.Header.Set("Authorization", "Bearer "+token)
    rr := httptest.NewRecorder()

    router := routes.BuildRouter(handler, DefaultRouterOptions())
    router.ServeHTTP(rr, req)

    assert.Equal(t, 1, mockS3.PutObjectCallCount)
}
```

---

## File Structure After Refactoring

```
go-backend/
├── main.go                          # Simplified: call bootstrap + start server
├── bootstrap/
│   ├── bootstrap.go                 # Existing
│   ├── typesense.go                 # Existing
│   ├── dependencies.go              # NEW: Centralized init
│   └── config.go                    # NEW: Config loading
├── pkg/
│   ├── routes/
│   │   ├── builder.go               # NEW: Router construction
│   │   └── groups.go                # NEW: Route grouping (auth, cards, etc.)
│   ├── middleware/
│   │   ├── chain.go                 # NEW: Middleware composition
│   │   └── wrappers.go              # NEW: CORS wrapper, logging setup
│   └── config/
│       └── config.go                # NEW: Config structs
├── handlers/                        # Existing
├── models/                          # Existing
├── services/                        # Existing
├── tests/                          # Existing
└── schema/                         # Existing
```

---

## Implementation Roadmap

### Step 1: Extract Configuration (Zettelgarden-9xw.8.1)
- Create `config.go` to load environment variables
- Create `dependencies.go` with initialization logic
- Minimal changes to main.go - just call new functions

### Step 2: Extract Router Building (Zettelgarden-9xw.8.2)
- Create `routes/builder.go` with `BuildRouter()` function
- Move all 138 route registrations from main.go
- Create helper functions `registerProtectedRoutes()`, `registerPublicRoutes()`
- Update main.go to call `routes.BuildRouter()`

### Step 3: Middleware Composition Layer (Zettelgarden-9xw.8.3)
- Create `middleware/chain.go` with `HandlerChain` type
- Convert `addProtectedRoute()` and `addRoute()` to use chains
- Add tests for middleware ordering
- Update routes builder to use chains

### Step 4: CORS Wrapper Extraction (Zettelgarden-9xw.8.4)
- Create `middleware/cors.go` wrapper function
- Make CORS configuration injectable
- Add test for CORS + auth integration

### Step 5: Test Suite Expansion (Zettelgarden-9xw.8.5)
- Add `routes/builder_test.go` - test route existence and methods
- Add `middleware/chain_test.go` - test middleware ordering
- Add integration tests in new `integration/` directory
- Create test helpers in `pkg/testing/` for common test patterns

---

## Obstacles Summary Table

| Obstacle | Current Impact | Root Cause | Recommended Solution |
|----------|---|---|---|
| Router composition untestable | Cannot verify all routes exist/correct middleware | Router only in main() | Extract BuildRouter() |
| Middleware precedence hidden | Subtle bugs in ordering | Inline nesting in main() | Create HandlerChain |
| CORS coverage gap | Production behavior not tested | CORS applied only in main() | Extract CORS wrapper |
| Dependency mocking difficult | Full server required for unit tests | init in main(), stored in globals | Create Dependencies struct |
| Configuration scattered | Cannot test different modes | Env vars loaded throughout main() | Create Config object |
| Async init not coordinated | Possible race conditions | goroutines in main() without sync | Use sync.WaitGroup or channels |

---

## Success Criteria

### Before Refactoring
- [ ] Route registration: 138 routes in main.go
- [ ] Tests: Each handler test creates mini-router (repeated pattern)
- [ ] Coverage: Only individual handler logic tested, not composition

### After Refactoring
- [ ] Route registration: Move to `routes/builder.go`, main.go < 50 lines
- [ ] Tests: Dedicated router composition tests
- [ ] Coverage: Test router, middleware ordering, and full request flow
- [ ] Extensibility: Adding new routes requires only one change (routes builder)
- [ ] Documentation: Clear patterns for adding routes/middleware

---

## Key Findings

### Strengths
1. **Solid test infrastructure** - conftest.go is well-designed
2. **Handler tests are hermetic** - Good isolation from external systems
3. **Middleware functions are pure** - Can be tested independently once called outside main()
4. **Service layer separation** - Business logic testable independently

### Weaknesses
1. **Entrypoint is monolithic** - 249 lines with too many responsibilities
2. **Router is inaccessible** - Cannot test composition without server startup
3. **Globals prevent testing** - Package-level `s` and `h` tightly couple code
4. **Implicit middleware ordering** - Must read main.go to understand request flow
5. **Configuration is implicit** - Environment variables loaded ad-hoc throughout main()

### Opportunities
1. Extract router builder → enables router testing
2. Create middleware chains → makes ordering explicit and testable
3. Centralize config → enables testing different modes
4. Create DI container → enables partial mocking
5. Async initialization management → safer startup, better error handling

---

## Conclusion

The Zettelgarden backend has **good bones** (solid test infrastructure, hermetic handler tests) but **poor testability at the entrypoint level**. The monolithic main() function and global variables prevent testing:

- Router composition
- Middleware integration
- Dependency injection scenarios
- Configuration variations

The recommended refactoring is **low-risk and high-value**:
- **Low-risk:** Extraction doesn't change behavior, only structure
- **High-value:** Enables comprehensive testing of initialization and routing
- **Incremental:** Can be done in 5 phases over time

The primary obstacle is **architectural** (tight coupling of routing to startup), not technical. Once the router is extracted into a testable function, the existing handler test patterns will work perfectly.

---

## Next Steps

1. **Review & approve this analysis** - Confirm understanding of obstacles and recommendations
2. **Create implementation beads** - Zettelgarden-9xw.8.1 through 9xw.8.5
3. **Start Phase 1** - Extract configuration into dedicated functions
4. **Measure improvement** - Count testable routes, middleware chains, and new test coverage
