# Architecture Diagrams: Testability Analysis

**Visual reference for Zettelgarden-9xw.8 review**

---

## Diagram 1: Current Architecture (Monolithic)

```
┌─────────────────────────────────────────────────────────────────┐
│                         main() Function                         │
│                       (249 lines, 13 steps)                     │
├─────────────────────────────────────────────────────────────────┤
│                                                                 │
│  Step 1: Load Logging Config                                   │
│  ├─→ os.Getenv("ZETTEL_DEV")                                   │
│  └─→ os.Getenv("ZETTEL_BACKEND_LOG_LOCATION")                 │
│                                                                 │
│  Step 2: Initialize Server                                     │
│  ├─→ bootstrap.InitServer()  ───→ Database Connection          │
│  ├─→ h = &handlers.Handler{...}                               │
│  └─→ Runs Migrations                                           │
│                                                                 │
│  Step 3: Initialize Stripe                                     │
│  └─→ stripe.Key = os.Getenv("STRIPE_SECRET_KEY")             │
│                                                                 │
│  Step 4: Initialize S3                                         │
│  └─→ s.S3 = h.CreateS3Client()                                │
│                                                                 │
│  Step 5: Initialize Mail Client                               │
│  ├─→ Host = os.Getenv("MAIL_HOST")                            │
│  ├─→ Password = os.Getenv("MAIL_PASSWORD")                    │
│  └─→ s.Mail = &mail.MailClient{...}                           │
│                                                                 │
│  Step 6: Initialize Typesense (ASYNC!)                         │
│  ├─→ bootstrap.InitTypesense()                                │
│  ├─→ if err == nil { (ERROR SWALLOWED!)                       │
│  └─→ go func() { h.InitSearchCollection() }()                 │
│                                                                 │
│  Step 7: Load JWT Secret                                       │
│  └─→ s.JwtSecretKey = os.Getenv("SECRET_KEY")                │
│                                                                 │
│  Step 8: Load LLM Config                                       │
│  ├─→ os.Getenv("ZETTEL_LLM_KEY")                              │
│  └─→ os.Getenv("ZETTEL_LLM_ENDPOINT")                         │
│                                                                 │
│  Step 9: Background Task (ASYNC, CONDITIONAL!)                 │
│  └─→ if os.Getenv("ZETTEL_RUN_CHUNKING_EMBEDDING") == "true" │
│                                                                 │
│  Step 10: Build Router (138 ROUTES!)                           │
│  ├─→ r := mux.NewRouter()                                      │
│  ├─→ addProtectedRoute(r, "/api/auth", ...)                  │
│  ├─→ addProtectedRoute(r, "/api/cards", ...)                 │
│  ├─→ ... 136 more routes ...                                  │
│  └─→ All routes use global h in closures                      │
│                                                                 │
│  Step 11: Setup CORS                                           │
│  ├─→ cors.New(cors.Options{...})                              │
│  └─→ handler := c.Handler(r)                                  │
│                                                                 │
│  Step 12: Get Port                                             │
│  └─→ os.Getenv("ZETTEL_PORT")                                 │
│                                                                 │
│  Step 13: Start Server (BLOCKS!)                               │
│  └─→ http.ListenAndServe(":"+port, handler)                   │
│                                                                 │
└─────────────────────────────────────────────────────────────────┘

              ↓

   ✗ Router is inaccessible to tests
   ✗ Configuration is scattered
   ✗ Globals prevent DI
   ✗ Middleware ordering is implicit
   ✗ CORS not testable separately
   ✗ Async init uncoordinated
```

---

## Diagram 2: Global State Dependencies

```
                    ┌──────────────┐
                    │  main.go:22  │
                    │  var s *server.Server
                    │  var h *handlers.Handler
                    └──────────────┘
                         ↑
                ┌────────┴────────┐
                │                 │
          ┌─────▼─────┐    ┌──────▼──────┐
          │ main():46 │    │ main():48-51 │
          │ s = Init  │    │ h = &Handler │
          │ Server()  │    └──────────────┘
          └───────────┘
                │
                │ h reaches into:
                ├──→ s.DB           (Database)
                ├──→ s.S3           (File Storage) - line 56
                ├──→ s.Mail         (Email)       - line 58
                ├──→ s.TypesenseClient (Search)  - line 67
                ├──→ s.JwtSecretKey (Auth)       - line 74
                └──→ s.LLMClient    (AI)         - (not stored)

          ┌─────────────────────────────────────┐
          │  Route Registration Functions       │
          │  (main.go:27-32)                    │
          │                                     │
          │  func addProtectedRoute(...) {      │
          │    return r.HandleFunc(path,        │
          │      h.APIKeyOrJWTMiddleware(...))  │
          │                        ↑            │
          │                        │            │
          │              Implicit global h      │
          │              (only accessible       │
          │               inside main)          │
          └─────────────────────────────────────┘
```

---

## Diagram 3: Middleware Ordering (Current - Implicit)

```
Reading Left-to-Right (Confusing!)
═════════════════════════════════

h.APIKeyOrJWTMiddleware(handlers.LogRoute(handler))
└─────────────────────┬──────────────────────┘
                      │
            Is it this order? Auth → Log → Handler
            Or this order?  Log → Auth → Handler


Actual Nesting Hierarchy (What Really Happens)
═════════════════════════════════════════════

handlers.LogRoute(handler)        → Creates LogRoute wrapper
    ↓
    returns: LogRoute(handler)     → Function that logs then calls handler

h.APIKeyOrJWTMiddleware(above)   → Creates Auth wrapper around LogRoute
    ↓
    returns: Auth(LogRoute(handler))  → Function that auths then calls LogRoute


Actual Execution Order
═════════════════════

Request
  ↓
Auth Middleware
  ├─ Check Authorization header
  ├─ Extract user from JWT or API key
  ├─ Add to context
  ↓
Log Middleware
  ├─ Read user from context
  ├─ Log request
  ↓
Handler Function
  ├─ Process request
  ├─ Return response
```

---

## Diagram 4: Current Test Pattern (Mini-Routers)

```
handlers/cards_test.go Pattern
══════════════════════════════

func TestGetCardSuccess(t *testing.T) {
    s := setup()            ← Creates Handler with test DB
    defer tests.Teardown()  ← Cleanup

    req := createRequest()  ← HTTP request
    token := generateJWT()  ← Test JWT

    rr := httptest.NewRecorder()
    router := mux.NewRouter()  ← ← ← MINI ROUTER!

    // DUPLICATED from main.go line 115:
    router.HandleFunc("/api/cards/{id}",
        s.JwtMiddleware(s.GetCardRoute))

    router.ServeHTTP(rr, req)

    assert.Equal(t, http.StatusOK, rr.Code)
}


The Problem: Duplication
═════════════════════════

main.go:115
    addProtectedRoute(r, "/api/cards/{id}", h.GetCardRoute, "GET")
    ├─ Uses: h.APIKeyOrJWTMiddleware(handlers.LogRoute(handler))
    └─ Has: LogRoute + APIKeyOrJWT

cards_test.go:32
    router.HandleFunc("/api/cards/{id}",
        s.JwtMiddleware(s.GetCardRoute))
    ├─ Uses: s.JwtMiddleware(handler)  ← DIFFERENT!
    └─ Missing: LogRoute, APIKeyOrJWT

Result: Tests don't match production behavior!
```

---

## Diagram 5: Request Flow - Production vs. Test

```
PRODUCTION (main.go)
════════════════════

Client Request
  ↓
CORS Middleware (main.go:233)
  ├─ Check origin
  ├─ Add CORS headers
  ↓
Router (main.go:88)
  ├─ Match path/method
  ↓
LogRoute Middleware (main.go:28-32)
  ├─ Log request (if debug)
  ↓
APIKeyOrJWTMiddleware (main.go:28)
  ├─ Extract auth header
  ├─ Validate JWT or API key
  ├─ Add user to context
  ↓
Handler Function
  ├─ Process business logic
  ↓
Response
  ├─ Handler writes response
  ├─ LogRoute forwards
  ├─ APIKeyOrJWT forwards
  ├─ Router forwards
  ├─ CORS adds headers
  ↓
Client Response


TEST (handlers/*_test.go)
═════════════════════════

Test Request
  ↓
Router (locally created)
  ├─ Match path/method
  ↓
JwtMiddleware (s.JwtMiddleware)
  ├─ Extract auth header
  ├─ Validate JWT only  ← Different!
  ├─ Add user to context
  ↓
Handler Function
  ├─ Process business logic
  ↓
Response


Differences
═══════════

Production has:           Test has:
✓ CORS middleware        ✗ No CORS
✓ LogRoute middleware    ✗ No LogRoute
✓ APIKeyOrJWT           ✗ Only JwtMiddleware
✓ Full auth chain       ✗ Partial auth

Production Coverage: 100%  vs.  Test Coverage: ~60%
```

---

## Diagram 6: Dependency Graph (Current)

```
                    ┌─────────────────┐
                    │    main.go      │
                    │ (entry point)   │
                    └────────┬────────┘
                             │
         ┌───────────────────┼───────────────────┐
         │                   │                   │
         ▼                   ▼                   ▼
    ┌─────────────┐  ┌──────────────┐  ┌─────────────────┐
    │ bootstrap/  │  │ handlers.go  │  │ handlers/auth.go│
    │ bootstrap   │  │              │  │ (middleware)    │
    └──────┬──────┘  └──────┬───────┘  └─────────┬───────┘
           │                │                     │
           ▼                ▼                     ▼
    ┌─────────────────────────────────────────────────────┐
    │          server.Server struct                       │
    │  ┌──────────────────────────────────────────────┐  │
    │  │ DB: *sql.DB                                  │  │
    │  │ S3: *s3.Client              (line 56)        │  │
    │  │ Mail: *mail.MailClient      (line 58)        │  │
    │  │ TypesenseClient: *typesense.Client (line 67) │  │
    │  │ LLMClient: *models.LLMClient (not stored!)   │  │
    │  │ JwtSecretKey: []byte        (line 74)        │  │
    │  │ StripeKey: string           (line 54)        │  │
    │  │ Testing: bool               (test setup)     │  │
    │  │ TestInspector: *TestInspector (test setup)   │  │
    │  │ SchemaDir: string                            │  │
    │  └──────────────────────────────────────────────┘  │
    └─────────────────────────────────────────────────────┘
                             │
                ┌────────────┼────────────┐
                │            │            │
                ▼            ▼            ▼
           ┌─────────┐  ┌─────────┐  ┌─────────┐
           │  Cards  │  │  Tasks  │  │ Users   │
           │ Handlers│  │Handlers │  │Handlers │
           └─────────┘  └─────────┘  └─────────┘


Problems with this graph:
═════════════════════════

1. Everything flows through server.Server
   → Cannot mock individual dependencies
   → Must create entire Server for unit tests

2. LLMClient is not stored
   → Is created but unused?
   → Cannot be tested

3. S3, Mail, Typesense added after Server creation
   → Not part of initialization graph
   → Harder to understand flow

4. No way to express "needs DB and Mail but not S3"
   → DI is all-or-nothing

5. server.Server is a flat list
   → No clear separation of concerns
   → Mixing HTTP-level and service-level deps
```

---

## Diagram 7: Recommended Architecture (After Refactoring)

```
                    ┌──────────────────────┐
                    │      main.go         │
                    │   (< 50 lines)       │
                    └──────────┬───────────┘
                               │
                ┌──────────────┼──────────────┐
                │              │              │
                ▼              ▼              ▼
    ┌──────────────────┐ ┌──────────────┐ ┌─────────────┐
    │ pkg/config/      │ │ pkg/bootstrap│ │ pkg/routes/ │
    │ LoadConfig()     │ │ InitDeps()   │ │ BuildRouter │
    └────────┬─────────┘ └──────┬───────┘ └──────┬──────┘
             │                  │                 │
             │                  ▼                 │
             │          ┌──────────────────┐     │
             │          │  Dependencies    │     │
             │          │  struct          │     │
             │          │  ├─ DB           │     │
             │          │  ├─ S3           │     │
             │          │  ├─ Mail         │     │
             │          │  ├─ Handler      │     │
             │          │  └─ ...          │     │
             │          └────────┬─────────┘     │
             │                   │               │
             └───────────────────┼───────────────┘
                                 │
                    ┌────────────┴────────────┐
                    │                         │
                    ▼                         ▼
    ┌──────────────────────────┐  ┌──────────────────────┐
    │ pkg/middleware/          │  │ handlers/            │
    │ ├─ chain.go              │  │ (unchanged)          │
    │ │  (HandlerChain)        │  │ ├─ auth.go           │
    │ ├─ cors.go               │  │ ├─ cards.go          │
    │ └─ wrappers.go           │  │ └─ ...               │
    └──────────────────────────┘  └──────────────────────┘
                    │
                    ▼
    ┌────────────────────────────────┐
    │  http.Server                   │
    │  Listen and Serve              │
    └────────────────────────────────┘


Advantages
══════════

1. Clear separation of concerns
   ✓ Config in one place
   ✓ Dependencies in another
   ✓ Routing in another

2. Each layer is testable
   ✓ Config can be tested with different values
   ✓ Router can be tested without server
   ✓ Dependencies can be partially mocked

3. Middleware is explicit
   ✓ HandlerChain makes ordering clear
   ✓ CORS is a separate layer
   ✓ Easy to add new middleware

4. Easier to extend
   ✓ Adding routes: modify BuildRouter()
   ✓ Adding middleware: modify HandlerChain
   ✓ Adding config: modify LoadConfig()
```

---

## Diagram 8: Middleware Composition (After Refactoring)

```
Using HandlerChain
═════════════════

protected := NewHandlerChain().
    Add(handlers.LogRoute).
    Add(h.APIKeyOrJWTMiddleware)

r.HandleFunc("/api/cards/{id}",
    protected.Build(h.GetCardRoute)).
    Methods("GET")


Explicit Order
══════════════

protected.middlewares = [
    handlers.LogRoute,          // Added first (runs second)
    h.APIKeyOrJWTMiddleware,    // Added second (runs first)
]

Build() applies in reverse:
1. Start with handler
2. Wrap with APIKeyOrJWTMiddleware (added second, wraps first)
3. Wrap with LogRoute (added first, wraps second)

Result:
    APIKeyOrJWTMiddleware
        ↓
    LogRoute
        ↓
    Handler


Testable Chain
══════════════

func TestChainOrder(t *testing.T) {
    called := []string{}

    h1 := func(w http.ResponseWriter, r *http.Request) {
        called = append(called, "handler")
    }

    m1 := func(next http.HandlerFunc) http.HandlerFunc {
        return func(w http.ResponseWriter, r *http.Request) {
            called = append(called, "m1")
            next.ServeHTTP(w, r)
        }
    }

    m2 := func(next http.HandlerFunc) http.HandlerFunc {
        return func(w http.ResponseWriter, r *http.Request) {
            called = append(called, "m2")
            next.ServeHTTP(w, r)
        }
    }

    chain := NewHandlerChain().Add(m1).Add(m2)
    handler := chain.Build(h1)

    rr := httptest.NewRecorder()
    req := httptest.NewRequest("GET", "/", nil)
    handler(rr, req)

    assert.Equal(t, []string{"m2", "m1", "handler"}, called)
}
```

---

## Diagram 9: Configuration Centralization (After Refactoring)

```
pkg/config/config.go
════════════════════

type Config struct {
    Server      ServerConfig
    Database    DatabaseConfig
    Services    ServiceConfig
    Middleware  MiddlewareConfig
}

ServerConfig:
    Port: "8080"
    LogLevel: "debug"
    EnableDebug: true

DatabaseConfig:
    Host: "localhost"
    Port: "5432"
    User: "postgres"
    Password: "***"
    Database: "zettelkasten"

ServiceConfig:
    S3Config: {...}
    MailConfig: {...}
    TypesenseConfig: {...}
    StripeKey: "sk_test_..."
    JWTSecret: "secret"
    LLMKey: "key"
    LLMEndpoint: "https://..."

MiddlewareConfig:
    CORSOrigins: []string{"http://localhost:3000"}
    LogRequests: true
    RateLimitPerSec: 100


Single Source of Truth
══════════════════════

Before:
    40+ os.Getenv() calls scattered throughout

After:
    config := LoadConfig()

    config.Database.Host       // One place to change
    config.Services.S3.Bucket  // Clearly documented
    config.Middleware.CORS     // Easy to test variations
```

---

## Diagram 10: Test Organization (After Refactoring)

```
Current Test Structure
══════════════════════

handlers/
├── cards_test.go         ✓ Handler logic
├── users_test.go         ✓ Handler logic
├── auth_test.go          ✓ Handler logic
└── ...

services/
├── cards_test.go         ✓ Business logic
└── ...

tests/
└── conftest.go           ✓ Test infrastructure


New Test Structure (Added)
══════════════════════════

handlers/
├── *_test.go             ✓ Handler logic (unchanged)
└── ...

services/
├── *_test.go             ✓ Business logic (unchanged)
└── ...

pkg/
├── config/
│   └── config_test.go    ✓ Configuration loading
├── routes/
│   └── builder_test.go   ✓ Router composition
├── middleware/
│   ├── chain_test.go     ✓ Middleware ordering
│   └── cors_test.go      ✓ CORS behavior
└── bootstrap/
    └── dependencies_test.go  ✓ Dependency initialization

integration/
├── router_test.go        ✓ Full request flow
├── middleware_test.go    ✓ Auth middleware integration
└── cors_test.go          ✓ CORS + auth integration

tests/
└── conftest.go           ✓ Test infrastructure (enhanced)


Test Coverage Summary
════════════════════

Layer 1: Unit Tests (Handlers + Services)
    Status: ✓ Existing and comprehensive
    Coverage: Business logic

Layer 2: Integration Tests (Routes + Middleware)
    Status: ✗ Missing (WILL ADD)
    Coverage: Composition and ordering

Layer 3: System Tests (Full Flow)
    Status: ✓ Manual/manual (can automate)
    Coverage: End-to-end behavior

After refactoring:
    ✓ Unit tests: Unchanged
    ✓ Integration tests: New
    ✓ System tests: Can now be automated
```

---

## Summary

| Current | Recommended |
|---------|------------|
| ❌ Monolithic main() | ✅ Separated concerns |
| ❌ Global variables | ✅ Dependency injection |
| ❌ Implicit middleware | ✅ Explicit HandlerChain |
| ❌ Untestable router | ✅ Testable BuildRouter() |
| ❌ Scattered config | ✅ Central Config struct |
| ❌ CORS not testable | ✅ Testable CORS wrapper |
| ❌ Test duplication | ✅ Single pattern |
| ❌ Partial test coverage | ✅ Complete coverage |
