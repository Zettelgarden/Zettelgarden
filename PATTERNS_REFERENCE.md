# Current Patterns Reference: Backend Testability Analysis

**Supporting Document for Zettelgarden-9xw.8 Review**

---

## Pattern 1: Package-Level Globals (main.go:22-24)

### Current Code
```go
var s *server.Server
var h *handlers.Handler
```

### Where Used
- `main()` initializes these (lines 46-51)
- Route registration functions read `h` (lines 27-32)
- Middleware closures capture `h` implicitly

### Problem Manifestation
```go
// main.go:27-29
func addProtectedRoute(r *mux.Router, path string, handler http.HandlerFunc, method string) *mux.Route {
    return r.HandleFunc(path, h.APIKeyOrJWTMiddleware(handlers.LogRoute(handler))).Methods(method)
                                 ^ Implicit dependency on global h
}
```

### Why It's a Problem
1. **Inaccessible in tests** - Test code cannot set these globals safely
2. **Tight coupling** - Router registration is bound to main() initialization order
3. **No flexibility** - Cannot test with different Handler configurations
4. **Initialization order dependency** - Must understand main() flow to modify routes

### Test Impact
```go
// handlers/cards_test.go - Current workaround
func makeCardRequestSuccess(s *Handler, t *testing.T, id int) *httptest.ResponseRecorder {
    // Must manually recreate router because can't access main's router
    router := mux.NewRouter()
    router.HandleFunc("/api/cards/{id}", s.JwtMiddleware(s.GetCardRoute))
    // ... missing h.APIKeyOrJWTMiddleware which would be used in main!
    router.ServeHTTP(rr, req)
    return rr
}
```

---

## Pattern 2: Monolithic main() with Interleaved Concerns

### Current Code Structure (main.go:35-249)

```go
func main() {
    // 1. LOGGING (lines 36-43)
    if os.Getenv("ZETTEL_DEV") != "true" {
        file, err := handlers.OpenLogFile(os.Getenv("ZETTEL_BACKEND_LOG_LOCATION"))
        if err != nil { log.Fatal(err) }
        log.SetOutput(file)
    }

    // 2. DATABASE (lines 45-46)
    s = bootstrap.InitServer()

    // 3. HANDLER SETUP (lines 48-51)
    h = &handlers.Handler{
        Server: s,
        DB:     s.DB,
    }

    // 4. STRIPE (line 54)
    stripe.Key = os.Getenv("STRIPE_SECRET_KEY")

    // 5. S3 (line 56)
    s.S3 = h.CreateS3Client()

    // 6. MAIL (lines 58-63)
    s.Mail = &mail.MailClient{
        Host:     os.Getenv("MAIL_HOST"),
        Password: os.Getenv("MAIL_PASSWORD"),
        Queue:    mail.NewEmailQueue(),
        DB:       s.DB,
    }

    // 7. TYPESENSE (lines 65-72) - ASYNC
    typesenseClient, err := bootstrap.InitTypesense()
    if err == nil {
        s.TypesenseClient = typesenseClient
        go func() {  // <-- Fire and forget!
            log.Printf("updating typesense")
            h.InitSearchCollection()
        }()
    }

    // 8. JWT (line 74)
    s.JwtSecretKey = []byte(os.Getenv("SECRET_KEY"))

    // 9. LLM (lines 75-76)
    config := openai.DefaultConfig(os.Getenv("ZETTEL_LLM_KEY"))
    config.BaseURL = os.Getenv("ZETTEL_LLM_ENDPOINT")
    // NOT STORED - OpenAI config is created but not used

    // 10. BACKGROUND TASK (lines 78-86) - ASYNC, CONDITIONAL
    if os.Getenv("ZETTEL_RUN_CHUNKING_EMBEDDING") == "true" {
        go func() {
            start := time.Now()
            // Commented out code
            elapsed := time.Since(start)
            fmt.Printf("Operation took %v\n", elapsed)
        }()
    }

    // 11. ROUTER BUILDING (lines 88-231)
    r := mux.NewRouter()
    addProtectedRoute(r, "/api/auth", h.CheckTokenRoute, "GET")
    addRoute(r, "/api/auth/github", h.StartGitHubOAuthRoute, "GET")
    // ... 135 more route registrations ...
    addProtectedRoute(r, "/api/chat/instructions", h.UpdateInstructionsRoute, "PUT")

    // 12. CORS (lines 233-240)
    c := cors.New(cors.Options{
        AllowedOrigins:   []string{os.Getenv("ZETTEL_URL")},
        AllowCredentials: true,
        AllowedHeaders:   []string{"authorization", "content-type"},
        AllowedMethods:   []string{"GET", "POST", "PUT", "PATCH", "DELETE"},
    })
    handler := c.Handler(r)

    // 13. SERVER START (lines 244-248)
    port := os.Getenv("ZETTEL_PORT")
    if port == "" {
        port = "8080"
    }
    http.ListenAndServe(":"+port, handler)
}
```

### Problem: Interleaved Concerns

| Step | Responsibility | Testable? | Modifiable? |
|------|---|---|---|
| 1 | Logging config | No | Only in main() |
| 2-3 | Database & handler | Partially | test Setup() works |
| 4-6 | External services | No | No way to mock |
| 7 | Async service init | No | Fire and forget |
| 8-9 | LLM config | No | Config not stored |
| 10 | Background task | No | Buried in main() |
| 11 | Router building | No | Cannot access |
| 12 | CORS setup | No | Only in main() |
| 13 | Server startup | N/A | Blocks forever |

### Test Impact
Every test must:
1. Set up environment variables
2. Call `bootstrap.InitServer()`
3. Create handlers manually
4. Duplicate middleware logic
5. Cannot test configuration variations

---

## Pattern 3: Implicit Middleware Closure (main.go:27-32)

### Current Code
```go
func addProtectedRoute(r *mux.Router, path string, handler http.HandlerFunc, method string) *mux.Route {
    return r.HandleFunc(path, h.APIKeyOrJWTMiddleware(handlers.LogRoute(handler))).Methods(method)
}

func addRoute(r *mux.Router, path string, handler http.HandlerFunc, method string) *mux.Route {
    return r.HandleFunc(path, handlers.LogRoute(handler)).Methods(method)
}
```

### How It's Used
```go
// 138+ route registrations follow this pattern
addProtectedRoute(r, "/api/cards", h.CreateCardRoute, "POST")  // line 110
addProtectedRoute(r, "/api/cards/{id}", h.GetCardRoute, "GET")  // line 115
addRoute(r, "/api/login", h.LoginRoute, "POST")                 // line 92
```

### Middleware Order
```
Request → LogRoute → APIKeyOrJWTMiddleware → Handler
```

But this order is **implicit**:
- Reading main.go shows the order
- Changing order is manual and error-prone
- Testing order requires mocking execution

### Pattern Example: Handler Struct Method

```go
// handlers/auth.go:44-82
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

        if err != nil {
            if err == jwt.ErrSignatureInvalid {
                http.Error(w, "Invalid token signature", http.StatusUnauthorized)
                return
            }

            log.Printf("err 3: %v", err)
            http.Error(w, "Invalid token", http.StatusUnauthorized)
            return
        }

        if !token.Valid {
            log.Printf("err 4: %v", err)
            http.Error(w, "Invalid token", http.StatusUnauthorized)
            return
        }

        ctx := context.WithValue(r.Context(), "current_user", claims.Sub)
        next.ServeHTTP(w, r.WithContext(ctx))
    }
}
```

### Test Pattern: Duplicated Mini-Router

```go
// handlers/cards_test.go:19-36
func makeCardRequestSuccess(s *Handler, t *testing.T, id int) *httptest.ResponseRecorder {
    token, _ := tests.GenerateTestJWT(1)

    req, err := http.NewRequest("GET", "/api/cards/"+strconv.Itoa(id), nil)
    if err != nil {
        t.Fatal(err)
    }
    req.Header.Set("Authorization", "Bearer "+token)
    req.SetPathValue("id", strconv.Itoa(id))

    rr := httptest.NewRecorder()
    router := mux.NewRouter()
    // DUPLICATED from main.go - must keep in sync!
    router.HandleFunc("/api/cards/{id}", s.JwtMiddleware(s.GetCardRoute))
    router.ServeHTTP(rr, req)

    return rr
}
```

### Problem: Duplicated Logic

- **main.go line 115** has `h.APIKeyOrJWTMiddleware(handlers.LogRoute(h.GetCardRoute))`
- **cards_test.go line 32** has `s.JwtMiddleware(s.GetCardRoute)` (DIFFERENT!)
- These will diverge when main.go changes

---

## Pattern 4: Bootstrap Package Initialization

### Current Code (bootstrap/bootstrap.go)

```go
func InitServer() *server.Server {
    dbConfig := models.DatabaseConfig{
        Host:         os.Getenv("DB_HOST"),
        Port:         os.Getenv("DB_PORT"),
        User:         os.Getenv("DB_USER"),
        Password:     os.Getenv("DB_PASS"),
        DatabaseName: os.Getenv("DB_NAME"),
    }

    db, err := server.ConnectToDatabase(dbConfig)
    if err != nil {
        log.Fatalf("Unable to connect to the database: %v\n", err)
    }

    s := &server.Server{
        DB:        db,
        SchemaDir: "./schema",
    }

    server.RunMigrations(s)
    return s
}
```

### Current Code (bootstrap/typesense.go)

```go
func InitTypesense() (*typesense.Client, error) {
    ctx := context.Background()

    client := GetTypesenseClient()
    collectionName := os.Getenv("TYPESENSE_COLLECTION")

    _, err := client.Collection(collectionName).Retrieve(ctx)
    if err == nil {
        // Collection exists
        fmt.Println("Collection already exists:", collectionName)
        return client, nil
    }

    // Create schema and collection...
    schema := &api.CollectionSchema{
        Name: collectionName,
        // ... 100+ lines of schema definition ...
    }
    _, err = client.Collections().Create(context.Background(), schema)
    return client, err
}
```

### Problem: Partial Initialization

In main.go (line 65-72):
```go
typesenseClient, err := bootstrap.InitTypesense()
if err == nil {    // <-- Error silently ignored!
    s.TypesenseClient = typesenseClient
    go func() {    // <-- Fire and forget!
        log.Printf("updating typesense")
        h.InitSearchCollection()
    }()
}
```

### Test Impact
- Cannot test Typesense initialization independently
- Cannot test graceful degradation when Typesense unavailable
- Async initialization has no coordination
- Error handling is implicit ("swallowed")

---

## Pattern 5: Handler Struct (handlers/handlers.go)

### Current Code
```go
type Handler struct {
    DB     *sql.DB
    Server *server.Server
}
```

### Embedded Dependencies (via Server)

```go
// server/server.go
type Server struct {
    DB              *sql.DB
    S3              *s3.Client
    Testing         bool
    JwtSecretKey    []byte
    StripeKey       string
    Mail            *mail.MailClient
    TestInspector   *TestInspector
    SchemaDir       string
    LLMClient       *models.LLMClient
    TypesenseClient *typesense.Client
}
```

### Usage in Handlers

```go
// handlers/auth.go:57-59
token, err := jwt.ParseWithClaims(tokenStr, claims, func(token *jwt.Token) (interface{}, error) {
    return s.Server.JwtSecretKey, nil  // Access through Server
})

// handlers/auth.go:312
s.Server.Mail.SendEmail("...", user.Email, messageBody)  // Access through Server
```

### Problem: Flat Namespace

All services share the Server struct with no clear separation:
```go
h.Server.DB           // Database
h.Server.S3           // File storage
h.Server.Mail         // Email
h.Server.TypesenseClient  // Search
h.Server.LLMClient    // AI
```

### Test Problem: Cannot Partially Mock

```go
// To test file upload with mock S3, must create entire Server:
testHandler := &Handler{
    DB: realDB,  // Must be real
    Server: &Server{
        DB: realDB,
        S3: mockS3,    // Can mock this
        Mail: &mail.MailClient{},  // Must provide something
        // ... must provide all 10 fields ...
    },
}
```

---

## Pattern 6: Test Infrastructure (tests/conftest.go)

### Strengths

```go
// lines 29-64: Setup() provides:
func Setup() *server.Server {
    // 1. Creates test database
    db, err := server.ConnectToDatabase(dbConfig)

    // 2. Populates test data
    s.DB = db
    s.Testing = true
    s.SchemaDir = "../schema"

    // 3. Runs migrations
    server.RunMigrations(s)
    err = importTestData(s)

    return s
}

// lines 66-68: Teardown() provides:
func Teardown() {
    server.ResetDatabase(S)
}
```

### Test Data Generation (lines 244-509)

```go
func generateData() map[string]interface{} {
    // Generates:
    // - 10 users (User 1 is admin)
    // - 24 cards with relationships
    // - 20 files
    // - 20 tasks
    // - Keywords, tags, entities
    // - Backlinks

    return map[string]interface{}{
        "users": users,
        "cards": cards,
        // ...
    }
}
```

### Helper Functions

```go
func GenerateTestJWT(userID int) (string, error)  // line 534
func ParseJsonResponse(t *testing.T, body []byte, x interface{})  // line 70
func CreateJsonBody(t *testing.T, v interface{}) *bytes.Reader  // line 558
func StringToReader(s string) *strings.Reader  // line 78
```

### Weakness: Global State

```go
var S *server.Server  // line 24 - GLOBAL (similar to main.go's globals!)

var setupOnce sync.Once  // line 26
var db *sql.DB          // line 27
```

---

## Pattern 7: Handler Test Pattern (handlers/cards_test.go)

### Setup Pattern

```go
// Line 57-66
func TestGetCardSuccess(t *testing.T) {
    s := setup()                  // Create handler with test server
    defer tests.Teardown()        // Cleanup after test

    var logCount int
    _ = s.DB.QueryRow("SELECT count(*) FROM card_views").Scan(&logCount)
    if logCount != 0 {
        t.Errorf("wrong log count, got %v want %v", logCount, 0)
    }
```

### Test Helper (setup function)

```go
// handlers/auth_test.go:14-24
func setup() *Handler {
    S := tests.Setup()
    s := &Handler{
        DB:     S.DB,
        Server: S,
    }

    S.S3 = s.CreateS3Client()
    return s
}
```

### Mini-Router Pattern

```go
// handlers/cards_test.go:19-36
func makeCardRequestSuccess(s *Handler, t *testing.T, id int) *httptest.ResponseRecorder {
    token, _ := tests.GenerateTestJWT(1)

    req, err := http.NewRequest("GET", "/api/cards/"+strconv.Itoa(id), nil)
    if err != nil {
        t.Fatal(err)
    }
    req.Header.Set("Authorization", "Bearer "+token)
    req.SetPathValue("id", strconv.Itoa(id))

    rr := httptest.NewRecorder()
    router := mux.NewRouter()
    router.HandleFunc("/api/cards/{id}", s.JwtMiddleware(s.GetCardRoute))
    router.ServeHTTP(rr, req)

    return rr
}
```

### Assertion Pattern

```go
// handlers/cards_test.go:68-85
if status := rr.Code; status != http.StatusOK {
    log.Printf("err %v", rr.Body.String())
    t.Errorf("handler returned wrong status code: got %v want %v", status, http.StatusOK)
}

var card models.Card
tests.ParseJsonResponse(t, rr.Body.Bytes(), &card)
if card.ID != 1 {
    t.Errorf("handler returned wrong card, got %v want %v", card.ID, 1)
}
```

---

## Pattern 8: Middleware Function Signature

### Pattern Definition

```go
// All middleware follow this pattern:
func (s *Handler) SomeMiddleware(next http.HandlerFunc) http.HandlerFunc {
    return func(w http.ResponseWriter, r *http.Request) {
        // Pre-processing
        if someCheck {
            http.Error(w, "error", http.StatusUnauthorized)
            return
        }

        // Context modification
        ctx := context.WithValue(r.Context(), "key", value)

        // Call next handler
        next.ServeHTTP(w, r.WithContext(ctx))
    }
}
```

### Examples

**JwtMiddleware** (auth.go:44-82)
```go
func (s *Handler) JwtMiddleware(next http.HandlerFunc) http.HandlerFunc {
    return func(w http.ResponseWriter, r *http.Request) {
        // ... validation logic ...
        ctx := context.WithValue(r.Context(), "current_user", claims.Sub)
        next.ServeHTTP(w, r.WithContext(ctx))
    }
}
```

**APIKeyOrJWTMiddleware** (auth.go:317-358)
```go
func (s *Handler) APIKeyOrJWTMiddleware(next http.HandlerFunc) http.HandlerFunc {
    return func(w http.ResponseWriter, r *http.Request) {
        // Try JWT first
        if userID, err := s.validateJWTToken(tokenStr); err == nil {
            ctx := context.WithValue(r.Context(), "current_user", userID)
            next.ServeHTTP(w, r.WithContext(ctx))
            return
        }

        // Try API key
        if userID, apiKeyID, err := s.validateAPIKey(tokenStr); err == nil {
            ctx := context.WithValue(r.Context(), "current_user", userID)
            ctx = context.WithValue(ctx, "api_key_id", apiKeyID)
            go s.updateAPIKeyLastUsed(apiKeyID)
            next.ServeHTTP(w, r.WithContext(ctx))
            return
        }

        // Fail
        http.Error(w, "Invalid authentication token", http.StatusUnauthorized)
    }
}
```

**Admin** (auth.go:28-42)
```go
func (s *Handler) Admin(next http.HandlerFunc) http.HandlerFunc {
    return func(w http.ResponseWriter, r *http.Request) {
        userID := r.Context().Value("current_user").(int)
        user, err := s.QueryUser(userID)
        if err != nil {
            http.Error(w, "User not found", http.StatusBadRequest)
            return
        }
        if !user.IsAdmin {
            http.Error(w, "Access denied", http.StatusUnauthorized)
            return
        }
        next.ServeHTTP(w, r)
    }
}
```

**LogRoute** (log.go:18-34)
```go
func LogRoute(next http.HandlerFunc) http.HandlerFunc {
    debug := os.Getenv("ZETTEL_DEBUG")
    if debug == "true" {
        return func(w http.ResponseWriter, r *http.Request) {
            userID, ok := r.Context().Value("current_user").(int)
            if !ok {
                log.Printf("- %s %s", r.Method, r.URL.Path)
            } else {
                log.Printf("- %s %s - user %v", r.Method, r.URL.Path, userID)
            }
            next.ServeHTTP(w, r)
        }
    }
    return func(w http.ResponseWriter, r *http.Request) {
        next.ServeHTTP(w, r)
    }
}
```

### Problem: Implicit Ordering

When composing: `h.APIKeyOrJWTMiddleware(handlers.LogRoute(handler))`

What's the actual order?
1. LogRoute runs first (innermost)
2. APIKeyOrJWTMiddleware runs second (outermost)
3. handler runs last

But reading left-to-right suggests opposite order!

---

## Pattern 9: Environment Variable Loading

### Scattered Throughout Code

```go
// main.go:37
os.Getenv("ZETTEL_DEV")

// main.go:38
os.Getenv("ZETTEL_BACKEND_LOG_LOCATION")

// main.go:54
os.Getenv("STRIPE_SECRET_KEY")

// main.go:59-60
os.Getenv("MAIL_HOST")
os.Getenv("MAIL_PASSWORD")

// main.go:74
os.Getenv("SECRET_KEY")

// main.go:75-76
os.Getenv("ZETTEL_LLM_KEY")
os.Getenv("ZETTEL_LLM_ENDPOINT")

// bootstrap/bootstrap.go:12-17
os.Getenv("DB_HOST")
os.Getenv("DB_PORT")
os.Getenv("DB_USER")
os.Getenv("DB_PASS")
os.Getenv("DB_NAME")

// bootstrap/typesense.go:14-15
os.Getenv("TYPESENSE_HOST")
os.Getenv("TYPESENSE_PASSWORD")

// bootstrap/typesense.go:26
os.Getenv("TYPESENSE_COLLECTION")

// handlers/log.go:19
os.Getenv("ZETTEL_DEBUG")

// main.go:234
os.Getenv("ZETTEL_URL")

// main.go:244
os.Getenv("ZETTEL_PORT")
```

### Problem: Configuration Spread

- **40+ environment variables** used throughout codebase
- No central source of truth
- Difficult to document required configuration
- Difficult to test with different configurations
- No validation of required vs. optional

---

## Pattern 10: CORS Configuration (main.go:233-240)

### Current Code
```go
c := cors.New(cors.Options{
    AllowedOrigins:   []string{os.Getenv("ZETTEL_URL")},
    AllowCredentials: true,
    AllowedHeaders:   []string{"authorization", "content-type"},
    AllowedMethods:   []string{"GET", "POST", "PUT", "PATCH", "DELETE"},
})

handler := c.Handler(r)
```

### Problem: Tight Coupling

1. **Only configurable in main()** - Cannot test CORS with different origins
2. **Applied after router** - Tests cannot verify CORS behavior (see cards_test.go - no CORS in tests)
3. **Not pluggable** - Cannot toggle CORS on/off for testing
4. **One origin only** - Does not support multiple origins per environment

### Production vs. Test Gap

**Production** (main.go):
```
Request → CORS middleware → Router → Auth middleware → Handler
```

**Tests** (handlers/cards_test.go):
```
Request → Router → Auth middleware → Handler
```
(No CORS in tests!)

---

## Summary of Current Patterns

| Pattern | Location | Strength | Weakness |
|---------|----------|----------|----------|
| Package globals | main.go:22-24 | Simple | Untestable, tight coupling |
| Monolithic main() | main.go:35-249 | All in one place | Mixed concerns, hard to test |
| Route registration | main.go:27-32 | Manual, explicit | Boilerplate, duplicated |
| Middleware closures | main.go routing | Functional, pure | Implicit ordering, hard to test |
| Bootstrap functions | bootstrap/*.go | Modular | Partial initialization, error swallowing |
| Handler struct | handlers/handlers.go | Centralized deps | Flat namespace, can't mock partially |
| Test infrastructure | tests/conftest.go | Comprehensive data | Global state, mini-router duplication |
| Handler tests | handlers/*_test.go | Hermetic, fast | Cannot test composition or CORS |
| Configuration | Scattered | Each file independent | No single source of truth |
| CORS setup | main.go:233-240 | Works | Not testable, production/test gap |

---

## Conclusion

The current patterns are **functional but inflexible**:

- **Good:** Business logic (handlers, services) is well-separated and testable
- **Good:** Test infrastructure is comprehensive for individual components
- **Good:** Middleware functions are pure and independently composable

- **Bad:** Initialization is monolithic and tightly coupled to startup
- **Bad:** Router is inaccessible for testing composition
- **Bad:** Configuration is scattered with no validation
- **Bad:** Dependencies cannot be partially mocked
- **Bad:** CORS behavior diverges between production and tests

**The refactoring opportunity** is to keep the good patterns while fixing the bad ones through extraction and composition.
