# Testability + Global State Assessment: go-backend/main.go

**Bead**: Zettelgarden-9xw.8
**Date**: 2026-01-15
**Status**: ANALYSIS ONLY - NO CODE CHANGES

---

## Executive Summary

The Zettelgarden backend has foundational unit/integration tests (in `handlers/`, `services/`, `tests/conftest.go`) but is severely hampered by:

1. **Package-level globals** (`var s *server.Server`, `var h *handlers.Handler`) in `main.go` that cannot be accessed in tests
2. **Monolithic main()** function (lines 35-249) that mixes initialization, middleware setup, and route registration
3. **Hard-coded route registration** making it impossible to test the router without starting the server
4. **Environment variable coupling** throughout bootstrap and initialization
5. **No router builder abstraction** preventing router testing in isolation

This prevents:
- Router-level integration tests (middleware chains, CORS, route ordering)
- Bootstrapping tests with custom configurations
- Dependency injection for testing alternate implementations
- Isolated testing of the complete request pipeline

---

## Current Testing Patterns

### Existing Test Infrastructure (Positive)

#### 1. Test Configuration Setup
**Location**: `go-backend/tests/conftest.go`

```go
var S *server.Server
var setupOnce sync.Once
var db *sql.DB

func Setup() *server.Server {
    setupOnce.Do(func() {
        dbConfig := models.DatabaseConfig{
            Host:     os.Getenv("DB_HOST"),
            Port:     os.Getenv("DB_PORT"),
            User:     os.Getenv("DB_USER"),
            Password: os.Getenv("DB_PASS"),
            DatabaseName: "zettelkasten_testing",
        }
        db, err = server.ConnectToDatabase(dbConfig)
    })
    S = &server.Server{}
    S.DB = db
    S.Testing = true
    S.SchemaDir = "../schema"
    S.Mail = &mail.MailClient{Testing: true, ...}
    S.LLMClient = &models.LLMClient{Testing: true}
    server.RunMigrations(S)
    importTestData(S)
    return S
}

func Teardown() {
    server.ResetDatabase(S)
}
```

**Strengths**:
- Centralized test setup/teardown
- Uses dedicated test database (`zettelkasten_testing`)
- Provides test data generation via `generateData()`
- Supports JWT token generation for authenticated tests

**Weaknesses**:
- Uses sync.Once for database connection (singleton anti-pattern)
- Relies on environment variables (hard to override per-test)
- Mutable global `S *server.Server`

#### 2. Handler Testing Pattern
**Examples**: `handlers/auth_test.go`, `handlers/cards_test.go`, `handlers/users_test.go`

```go
// Test setup helper
func setup() *Handler {
    S := tests.Setup()
    s := &Handler{
        DB:     S.DB,
        Server: S,
    }
    S.S3 = s.CreateS3Client()
    return s
}

// Individual handler test
func TestAuthResetPasswordAndLoginSuccess(t *testing.T) {
    s := setup()
    defer tests.Teardown()

    req, _ := http.NewRequest("GET", "/api/reset-password", bytes.NewBuffer(jsonData))
    req.Header.Set("Authorization", "Bearer "+token)

    rr := httptest.NewRecorder()
    handler := http.HandlerFunc(s.JwtMiddleware(s.ResetPasswordRoute))
    handler.ServeHTTP(rr, req)

    // Assertions
}
```

**Strengths**:
- Uses `httptest.NewRecorder()` for isolated handler testing
- Applies middleware manually to test handler+middleware chains
- Good test data availability

**Weaknesses**:
- Manually sets up route/router per test (fragile, doesn't test actual route registration)
- Middleware manually wrapped rather than testing via router
- No CORS or other global middleware tested
- JWT secret key in tests is hardcoded as empty string (`var jwtKey = []byte("")`)

#### 3. Middleware Testing
**Location**: `handlers/auth.go` - lines 44-82, 317+

```go
func (s *Handler) JwtMiddleware(next http.HandlerFunc) http.HandlerFunc {
    return func(w http.ResponseWriter, r *http.Request) {
        tokenStr := r.Header.Get("Authorization")
        if tokenStr == "" {
            http.Error(w, "Authorization header is missing", http.StatusUnauthorized)
            return
        }
        tokenStr = tokenStr[len("Bearer "):]
        claims := &models.Claims{}
        token, err := jwt.ParseWithClaims(tokenStr, claims, func(token *jwt.Token) (interface{}, error) {
            return s.Server.JwtSecretKey, nil
        })
        if !token.Valid {
            http.Error(w, "Invalid token", http.StatusUnauthorized)
            return
        }
        ctx := context.WithValue(r.Context(), "current_user", claims.Sub)
        next.ServeHTTP(w, r.WithContext(ctx))
    }
}
```

**Tested Via**: Individual handler tests manually applying middleware
**Not Tested**:
- Middleware ordering
- CORS interaction with routes
- LogRoute behavior at router level
- APIKeyOrJWTMiddleware as a real middleware stack

---

## Obstacles to Testing

### Obstacle 1: Package-level Globals (CRITICAL)

**Location**: `go-backend/main.go:22-24`
```go
var s *server.Server
var h *handlers.Handler
```

**Problem**:
- These globals are only populated inside `main()` function
- No way to initialize them for test scenarios
- Test code cannot construct alternative Server or Handler configurations
- Couples all code that might need these to the main() initialization logic

**Impact**:
- Cannot write router integration tests
- Cannot test bootstrap sequence
- Cannot test with mocked dependencies
- Forces all tests through full database setup

### Obstacle 2: Monolithic main() Function (CRITICAL)

**Location**: `go-backend/main.go:35-249`

**Problem**:
The `main()` function is a 214-line procedural script that combines:
- Environment variable parsing (lines 37-43)
- Server initialization (line 46)
- Stripe key setup (line 54)
- S3 client creation (line 56)
- Mail client initialization (lines 58-63)
- Typesense setup (lines 65-72)
- OpenAI client config (lines 75-76)
- Background goroutines for embeddings (lines 78-86)
- Router creation (line 88)
- Route registration (lines 89-232) - 144 lines of addRoute/addProtectedRoute calls
- CORS configuration (lines 233-240)
- Server startup (lines 244-248)

**Why this matters**:
- No way to test or reuse router construction
- Impossible to bootstrap with different configurations
- Testing requires starting actual HTTP server
- Background goroutines started unconditionally
- Mixed concerns (init, routing, startup)

### Obstacle 3: Hard-Coded Route Registration (CRITICAL)

**Location**: `go-backend/main.go:89-232`

```go
func addProtectedRoute(r *mux.Router, path string, handler http.HandlerFunc, method string) *mux.Route {
    return r.HandleFunc(path, h.APIKeyOrJWTMiddleware(handlers.LogRoute(handler))).Methods(method)
}

func addRoute(r *mux.Router, path string, handler http.HandlerFunc, method string) *mux.Route {
    return r.HandleFunc(path, handlers.LogRoute(handler)).Methods(method)
}

func main() {
    r := mux.NewRouter()
    addProtectedRoute(r, "/api/auth", h.CheckTokenRoute, "GET")
    addRoute(r, "/api/auth/github", h.StartGitHubOAuthRoute, "GET")
    // ... 140+ more addRoute/addProtectedRoute calls
}
```

**Problem**:
- Routes are defined inline in main(), not testable as a unit
- Route registration depends on runtime execution of main()
- No way to get a configured router without executing main()
- Changes to routes require testing the entire server

**Test Scenarios Impossible**:
- Verify all routes are registered
- Test route method restrictions (GET vs POST)
- Verify route ordering/priority
- Test regex patterns in route paths
- Validate middleware chain composition

### Obstacle 4: Environment Variable Coupling (SIGNIFICANT)

**Locations**: Throughout main() and bootstrap

```go
// main.go:37-43
if os.Getenv("ZETTEL_DEV") != "true" {
    file, err := handlers.OpenLogFile(os.Getenv("ZETTEL_BACKEND_LOG_LOCATION"))
    log.SetOutput(file)
}

// main.go:54
stripe.Key = os.Getenv("STRIPE_SECRET_KEY")

// main.go:74-76
s.JwtSecretKey = []byte(os.Getenv("SECRET_KEY"))
config := openai.DefaultConfig(os.Getenv("ZETTEL_LLM_KEY"))
config.BaseURL = os.Getenv("ZETTEL_LLM_ENDPOINT")

// bootstrap/bootstrap.go:11-18
func InitServer() *server.Server {
    dbConfig := models.DatabaseConfig{
        Host:     os.Getenv("DB_HOST"),
        Port:     os.Getenv("DB_PORT"),
        User:     os.Getenv("DB_USER"),
        Password: os.Getenv("DB_PASS"),
        DatabaseName: os.Getenv("DB_NAME"),
    }
    db, err := server.ConnectToDatabase(dbConfig)
    if err != nil {
        log.Fatalf("Unable to connect to the database: %v\n", err)
    }
}
```

**Problem**:
- Tests must set environment variables to control initialization
- No way to override configuration per-test
- Tests cannot verify error handling during initialization
- Tests depend on actual environment state

### Obstacle 5: Missing Abstractions (SIGNIFICANT)

**Missing Interfaces**:
- No `RouterBuilder` or similar abstraction
- No configuration object (struct with fields, not env vars)
- No dependency injection framework
- Middleware chain not composable/testable as a unit
- No way to provide mock implementations of external services

**Example of what's missing**:
```go
// This abstraction doesn't exist:
type RouterConfig struct {
    Handler *handlers.Handler
    Middleware []Middleware
    LogRoute bool
}

func BuildRouter(config RouterConfig) http.Handler {
    r := mux.NewRouter()
    // register routes
    return c.Handler(r)
}
```

### Obstacle 6: JWT Secret Key Test Hardcoding (MINOR)

**Location**: `go-backend/tests/conftest.go:534-535`

```go
func GenerateTestJWT(userID int) (string, error) {
    var jwtKey = []byte("")  // Empty key!
    // ...
    tokenString, err := token.SignedString(jwtKey)
}
```

**Problem**:
- JWT is signed with empty key
- Doesn't match actual server's `JwtSecretKey` from environment
- Tests inadvertently pass because middleware uses different key
- Fragile - could break if key validation logic changes

---

## Recommended Refactor Structure

### Goal
Enable comprehensive unit/integration tests of:
1. Router construction and route registration
2. Middleware composition and ordering
3. Bootstrap/initialization sequence
4. Configuration management
5. Error handling during startup

Without breaking existing handler tests or changing runtime behavior.

### Architecture Pattern

```
go-backend/
├── main.go                    (unchanged - startup only)
├── bootstrap/
│   ├── bootstrap.go          (unchanged - DB connection)
│   ├── config.go             (NEW)
│   ├── router.go             (NEW)
│   └── typesense.go          (unchanged)
├── handlers/
│   ├── handlers.go           (unchanged)
│   ├── *.go                  (unchanged)
│   └── *_test.go             (refactored)
├── server/
│   ├── server.go             (unchanged)
│   └── *
├── tests/
│   ├── conftest.go           (refactored)
│   ├── router_test.go        (NEW)
│   └── bootstrap_test.go     (NEW)
└── ...
```

### Phase 1: Configuration Abstraction

**New File**: `go-backend/bootstrap/config.go`

**Purpose**: Extract environment variables and defaults into a structured config object

**Key Concepts**:
```go
type ServerConfig struct {
    // Database
    Database models.DatabaseConfig

    // Server
    Port string
    JwtSecretKey []byte

    // External Services
    StripeKey string
    S3Config aws.Config
    MailConfig mail.Config
    LLMKey string
    LLMEndpoint string
    TypesenseConfig TypesenseConfig

    // Behavior Flags
    DevMode bool
    LogPath string
    RunEmbeddings bool
}

func NewServerConfigFromEnv() (*ServerConfig, error) {
    // Validate and parse all env vars here
    // Return errors for missing required config
}

func NewServerConfigForTesting() *ServerConfig {
    // Pre-configured for test scenarios
}
```

**Benefits**:
- Single source of truth for configuration
- Testable configuration construction
- Can create test-specific configs
- Decouples tests from environment variables
- Validates configuration at bootstrap time

### Phase 2: Router Builder Function

**New File**: `go-backend/bootstrap/router.go`

**Purpose**: Extract route registration logic from main() into a testable function

**Key Concepts**:
```go
type RouterConfig struct {
    Handler *handlers.Handler
    CORS *cors.Options
    LogRequests bool
}

func BuildRouter(cfg *RouterConfig) *mux.Router {
    r := mux.NewRouter()

    // Route registration (144 lines from main.go)
    addProtectedRoute(r, "/api/auth", cfg.Handler.CheckTokenRoute, "GET", cfg.Handler)
    addRoute(r, "/api/auth/github", cfg.Handler.StartGitHubOAuthRoute, "GET")
    // ... all routes ...

    return r
}

func BuildHandler(config http.Handler) http.Handler {
    // Apply CORS
    // Apply other global middleware
    return config
}
```

**Benefits**:
- Router can be tested in isolation via httptest.Server
- Routes can be verified without running full application
- Middleware order testable
- Route registration logic decoupled from main()
- Easy to test route configuration changes

### Phase 3: Dependency Injection for Tests

**Refactor**: `go-backend/tests/conftest.go`

**Changes**:
```go
type TestEnvironment struct {
    Server *server.Server
    Handler *handlers.Handler
    Router *mux.Router
    Client *http.Client  // For integration tests
}

func SetupWithConfig(config *bootstrap.ServerConfig) *TestEnvironment {
    // Initialize with custom config
    server := &server.Server{DB: config.Database...}
    handler := &handlers.Handler{Server: server, DB: server.DB}
    router := bootstrap.BuildRouter(&bootstrap.RouterConfig{Handler: handler})
    return &TestEnvironment{
        Server: server,
        Handler: handler,
        Router: router,
        Client: &http.Client{},
    }
}

func SetupDefault() *TestEnvironment {
    config := bootstrap.NewServerConfigForTesting()
    return SetupWithConfig(config)
}

func (te *TestEnvironment) Teardown() {
    server.ResetDatabase(te.Server)
}
```

**Benefits**:
- Tests can provide custom configurations
- Router tests via httptest.Server
- Easier to mock/swap dependencies
- Test environment explicit and testable

### Phase 4: Bootstrap Initialization (Refactored)

**Refactored**: `go-backend/bootstrap/bootstrap.go`

```go
type ServerInitializer struct {
    Config *ServerConfig
}

func NewServerInitializer(config *ServerConfig) *ServerInitializer {
    return &ServerInitializer{Config: config}
}

func (s *ServerInitializer) Initialize() (*server.Server, error) {
    // Returns error if initialization fails
    db, err := server.ConnectToDatabase(s.Config.Database)
    if err != nil {
        return nil, fmt.Errorf("database connection failed: %w", err)
    }

    server := &server.Server{
        DB: db,
        SchemaDir: "./schema",
        Testing: false,
    }

    if err := server.RunMigrations(); err != nil {
        return nil, fmt.Errorf("migrations failed: %w", err)
    }

    // Initialize services...
    server.S3 = initS3(s.Config)
    server.Mail = initMail(s.Config)
    server.TypesenseClient = initTypesense(s.Config)
    server.JwtSecretKey = s.Config.JwtSecretKey

    return server, nil
}
```

**Benefits**:
- Initialization is explicit and testable
- Errors returned, not fatal
- Can test initialization failures
- Dependencies flow through return values, not globals

### Phase 5: Updated main()

**Simplified main.go**:
```go
func main() {
    // Minimal setup
    config, err := bootstrap.NewServerConfigFromEnv()
    if err != nil {
        log.Fatalf("Invalid configuration: %v", err)
    }

    initializer := bootstrap.NewServerInitializer(config)
    server, err := initializer.Initialize()
    if err != nil {
        log.Fatalf("Initialization failed: %v", err)
    }

    handler := &handlers.Handler{
        Server: server,
        DB: server.DB,
    }

    router := bootstrap.BuildRouter(&bootstrap.RouterConfig{
        Handler: handler,
        CORS: &cors.Options{...},
        LogRequests: config.DevMode,
    })

    httpHandler := bootstrap.BuildHandler(router)

    port := config.Port
    if port == "" {
        port = "8080"
    }

    if err := http.ListenAndServe(":"+port, httpHandler); err != nil {
        log.Fatalf("Server failed: %v", err)
    }
}
```

**Benefits**:
- Minimal, clear, testable
- Easy to follow flow
- Can verify startup behavior
- No globals
- Errors propagate clearly

---

## New Testing Scenarios Enabled

### Router Integration Tests
**File**: `go-backend/tests/router_test.go`

```go
func TestAllRoutesRegistered(t *testing.T) {
    te := tests.SetupDefault()
    defer te.Teardown()

    // Verify all routes are registered
    routes := []string{
        "/api/auth",
        "/api/cards",
        "/api/tasks",
        // ... all 100+ routes
    }

    for _, route := range routes {
        // Test that route exists and has correct method
    }
}

func TestRouteMethodRestrictions(t *testing.T) {
    te := tests.SetupDefault()
    defer te.Teardown()

    // Verify GET /api/cards requires POST, not GET
    req, _ := http.NewRequest("GET", "/api/cards", nil)
    rr := httptest.NewRecorder()
    te.Router.ServeHTTP(rr, req)
    if rr.Code != http.StatusMethodNotAllowed {
        t.Errorf("Expected 405, got %d", rr.Code)
    }
}

func TestMiddlewareOrdering(t *testing.T) {
    te := tests.SetupDefault()
    defer te.Teardown()

    // Verify that CORS headers are set
    req, _ := http.NewRequest("GET", "/api/cards", nil)
    req.Header.Set("Authorization", "Bearer "+testJWT)
    rr := httptest.NewRecorder()
    te.Router.ServeHTTP(rr, req)

    if rr.Header().Get("Access-Control-Allow-Credentials") != "true" {
        t.Error("CORS headers not set by middleware")
    }
}

func TestProtectedRouteBlocking(t *testing.T) {
    te := tests.SetupDefault()
    defer te.Teardown()

    // Protected routes should reject unauthenticated requests
    req, _ := http.NewRequest("GET", "/api/cards/1", nil)
    // No Authorization header
    rr := httptest.NewRecorder()
    te.Router.ServeHTTP(rr, req)

    if rr.Code != http.StatusUnauthorized {
        t.Errorf("Protected route not blocked, got %d", rr.Code)
    }
}
```

### Bootstrap/Configuration Tests
**File**: `go-backend/tests/bootstrap_test.go`

```go
func TestConfigFromEnv(t *testing.T) {
    // Set test env vars
    t.Setenv("DB_HOST", "localhost")
    t.Setenv("SECRET_KEY", "test-key")

    config, err := bootstrap.NewServerConfigFromEnv()
    if err != nil {
        t.Fatalf("Config creation failed: %v", err)
    }

    if config.Database.Host != "localhost" {
        t.Errorf("Wrong host, got %s", config.Database.Host)
    }
}

func TestConfigValidation(t *testing.T) {
    // Missing required env vars should error
    t.Setenv("DB_HOST", "")  // Required

    config, err := bootstrap.NewServerConfigFromEnv()
    if err == nil {
        t.Fatal("Expected validation error for missing DB_HOST")
    }
}

func TestInitializationFailure(t *testing.T) {
    config := bootstrap.NewServerConfigForTesting()
    config.Database.Host = "nonexistent"

    initializer := bootstrap.NewServerInitializer(config)
    server, err := initializer.Initialize()

    if err == nil {
        t.Fatal("Expected connection error")
    }
    if server != nil {
        t.Fatal("Server should be nil on error")
    }
}
```

### Handler Tests Can Be Simplified
**File**: `go-backend/handlers/cards_test.go` (refactored)

```go
func TestGetCardWithRouter(t *testing.T) {
    te := tests.SetupDefault()
    defer te.Teardown()

    token, _ := tests.GenerateTestJWT(1)
    req, _ := http.NewRequest("GET", "/api/cards/1", nil)
    req.Header.Set("Authorization", "Bearer "+token)

    rr := httptest.NewRecorder()
    te.Router.ServeHTTP(rr, req)  // Test via actual router!

    if rr.Code != http.StatusOK {
        t.Errorf("Got %d", rr.Code)
    }
}
```

---

## Impact Analysis

### What Changes
1. **New files**: `bootstrap/config.go`, `bootstrap/router.go`, `tests/router_test.go`, `tests/bootstrap_test.go`
2. **Refactored**: `bootstrap/bootstrap.go`, `tests/conftest.go`, `main.go`
3. **Unchanged**: All handler implementations, all existing handler tests (mostly)

### What Stays the Same
- All business logic in handlers
- All handler method signatures
- All database interactions
- Runtime behavior of the application
- HTTP API contract

### Breaking Changes
**NONE** - This is backward compatible:
- Application starts exactly the same way
- All routes registered identically
- Middleware applied in same order
- No public API changes

### Testing Improvements
- Can write router-level tests
- Can test bootstrap/configuration
- Can verify route registration
- Can test middleware ordering
- Can write tests with custom configs
- Existing handler tests continue to work
- New tests add confidence in plumbing

---

## Migration Path

### Phase 1: Prepare (Non-breaking)
1. Create `bootstrap/config.go` with `ServerConfig` struct
2. Add `bootstrap/router.go` with `BuildRouter()` function
3. Add `tests/router_test.go` with new test scenarios
4. Keep `main.go` unchanged - uses new functions

### Phase 2: Refactor main.go (Non-breaking)
1. Update `main.go` to use `ServerConfig` and `BuildRouter()`
2. Test that application still works
3. Test that existing tests still pass

### Phase 3: Update Test Infrastructure (Non-breaking)
1. Refactor `tests/conftest.go` to use new patterns
2. Update handler tests to optionally use `TestEnvironment`
3. Add new bootstrap tests

### Phase 4: Finalize
1. Add comprehensive router tests
2. Document new testing patterns
3. Update contributing guide

---

## Recommended Beads

Based on this assessment, recommend creating:

### 1. Bead: Create Configuration Abstraction
- **Type**: Task
- **Priority**: P1 (foundation for other work)
- **Description**: Extract environment variables into `ServerConfig` struct with validation
- **Acceptance Criteria**:
  - `bootstrap/config.go` with `ServerConfig` struct
  - `NewServerConfigFromEnv()` with validation
  - `NewServerConfigForTesting()` helper
  - Tests for config construction and validation
- **Depends On**: None
- **Blocks**: Router Builder, Bootstrap Tests

### 2. Bead: Extract Router Builder Function
- **Type**: Task
- **Priority**: P1 (foundation for testing)
- **Description**: Move route registration logic from main() into testable `BuildRouter()` function
- **Acceptance Criteria**:
  - `bootstrap/router.go` with `BuildRouter()`
  - All 144 route registrations moved from main()
  - `RouterConfig` struct for configuration
  - Router can be tested via httptest without starting server
  - No changes to actual routes or middleware
- **Depends On**: Configuration Abstraction
- **Blocks**: Router Integration Tests

### 3. Bead: Router Integration Tests
- **Type**: Task
- **Priority**: P2 (validation)
- **Description**: Test router construction, route registration, middleware ordering
- **Acceptance Criteria**:
  - `tests/router_test.go` with 10+ integration tests
  - Tests verify all routes registered
  - Tests verify method restrictions (POST vs GET)
  - Tests verify protected route blocking
  - Tests verify CORS headers
  - Tests verify middleware ordering
- **Depends On**: Router Builder Function
- **Blocks**: None (but increases confidence in plumbing)

### 4. Bead: Bootstrap/Initialization Tests
- **Type**: Task
- **Priority**: P2 (error handling)
- **Description**: Test bootstrap sequence and initialization error handling
- **Acceptance Criteria**:
  - `tests/bootstrap_test.go` with 8+ tests
  - Config validation tests
  - Database connection failure handling
  - Invalid configuration detection
  - Initialization order verification
- **Depends On**: Configuration Abstraction
- **Blocks**: None

### 5. Bead: Refactor main.go (Non-breaking)
- **Type**: Refactoring
- **Priority**: P2 (cleanup)
- **Description**: Update main() to use new config and router builder functions
- **Acceptance Criteria**:
  - `main()` reduced to <50 lines
  - No package-level globals
  - Uses `ServerConfig` and `BuildRouter()`
  - All tests pass
  - Application behavior unchanged
  - Can start server successfully
- **Depends On**: Configuration Abstraction, Router Builder
- **Blocks**: None

### 6. Bead: Refactor Test Infrastructure
- **Type**: Refactoring
- **Priority**: P3 (test improvement)
- **Description**: Update tests to use new `TestEnvironment` and configurations
- **Acceptance Criteria**:
  - `tests/conftest.go` refactored with `TestEnvironment`
  - Handler tests updated to use new patterns
  - All existing tests still pass
  - Can create test-specific configurations
  - Better test isolation
- **Depends On**: Configuration Abstraction, Router Builder
- **Blocks**: None

---

## Summary of Obstacles and Solutions

| Obstacle | Current Impact | Recommended Solution | Phase |
|----------|-----------------|----------------------|-------|
| Package-level globals | Cannot test initialization | Use function parameters, return values | 1-2 |
| Monolithic main() | Impossible to test router | Extract to `BuildRouter()` function | 2 |
| Hard-coded routes | Cannot verify registration | Router builder with programmatic registration | 2 |
| Env var coupling | Tests brittle, hard to override | `ServerConfig` struct with `FromEnv()` | 1 |
| No abstractions | Cannot inject test dependencies | Router/Config builders enable DI | 1-2 |
| JWT test hardcoding | Fragile JWT tests | Tests can pass custom secrets via config | 3 |

---

## Implementation Checklist

- [ ] Review and approve this assessment
- [ ] Create Bead 1: Configuration Abstraction
- [ ] Create Bead 2: Router Builder Function
- [ ] Create Bead 3: Router Integration Tests
- [ ] Create Bead 4: Bootstrap Tests
- [ ] Create Bead 5: Refactor main.go
- [ ] Create Bead 6: Refactor Test Infrastructure
- [ ] Implement Phase 1 (Config)
- [ ] Implement Phase 2 (Router)
- [ ] Implement Phase 3 (Tests)
- [ ] Implement Phase 4 (main.go)
- [ ] Implement Phase 5 (Test refactoring)
- [ ] Verify all tests pass
- [ ] Update CLAUDE.md with new testing patterns
- [ ] Verify application starts and runs correctly
