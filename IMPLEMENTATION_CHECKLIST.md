# Implementation Checklist: Testability Refactoring

**Supporting document for Zettelgarden-9xw.8 review**
**Use this to track implementation progress across all 5 phases**

---

## Phase 1: Extract Configuration

**Effort:** 1-2 hours | **Risk:** Low | **Prerequisite:** None

### Create New Files

- [ ] Create `/home/nick/code/Zettelgarden/go-backend/pkg/config/config.go`
  - [ ] Define `ServerConfig` struct
  - [ ] Define `DatabaseConfig` struct
  - [ ] Define `ServiceConfig` struct
  - [ ] Define `MiddlewareConfig` struct
  - [ ] Implement `LoadConfig()` function
  - [ ] Add environment variable validation
  - [ ] Add default values for optional config

- [ ] Create `/home/nick/code/Zettelgarden/go-backend/pkg/config/config_test.go`
  - [ ] Test LoadConfig() with valid env vars
  - [ ] Test LoadConfig() with missing optional vars
  - [ ] Test LoadConfig() with invalid values

### Modify Existing Files

- [ ] Modify `/home/nick/code/Zettelgarden/go-backend/bootstrap/bootstrap.go`
  - [ ] Change `InitServer()` to accept `DatabaseConfig` parameter
  - [ ] Remove direct `os.Getenv()` calls
  - [ ] Update call sites

- [ ] Modify `/home/nick/code/Zettelgarden/go-backend/bootstrap/typesense.go`
  - [ ] Change `InitTypesense()` to accept config parameter
  - [ ] Remove direct `os.Getenv()` calls

- [ ] Modify `/home/nick/code/Zettelgarden/go-backend/main.go`
  - [ ] Add `config := config.LoadConfig()` after imports
  - [ ] Update `bootstrap.InitServer()` call to pass config
  - [ ] Update `bootstrap.InitTypesense()` call to pass config
  - [ ] Update `cors.New()` to use config values
  - [ ] Update port retrieval to use config

### Validation

- [ ] Verify main.go still compiles
- [ ] Verify tests still pass with `source .env-bash && go test ./...`
- [ ] Verify server still starts: `go run main.go`
- [ ] Check: No new env var reads in main.go

---

## Phase 2: Extract Router Builder

**Effort:** 2-3 hours | **Risk:** Low | **Prerequisite:** Phase 1 or independent

### Create New Files

- [ ] Create `/home/nick/code/Zettelgarden/go-backend/pkg/routes/builder.go`
  - [ ] Define `RouterOptions` struct
  - [ ] Implement `BuildRouter(handler *handlers.Handler, opts RouterOptions) *mux.Router`
  - [ ] Implement `registerAuthRoutes(r *mux.Router, h *handlers.Handler)`
  - [ ] Implement `registerCardRoutes(r *mux.Router, h *handlers.Handler)`
  - [ ] Implement `registerUserRoutes(r *mux.Router, h *handlers.Handler)`
  - [ ] Implement `registerTaskRoutes(r *mux.Router, h *handlers.Handler)`
  - [ ] Implement `registerFileRoutes(r *mux.Router, h *handlers.Handler)`
  - [ ] Implement `registerSearchRoutes(r *mux.Router, h *handlers.Handler)`
  - [ ] Implement `registerEntityRoutes(r *mux.Router, h *handlers.Handler)`
  - [ ] Implement `registerFactRoutes(r *mux.Router, h *handlers.Handler)`
  - [ ] Implement `registerTemplateRoutes(r *mux.Router, h *handlers.Handler)`
  - [ ] Implement `registerSubscriptionRoutes(r *mux.Router, h *handlers.Handler)`
  - [ ] Implement `registerChatRoutes(r *mux.Router, h *handlers.Handler)`
  - [ ] Implement `registerStatsRoutes(r *mux.Router, h *handlers.Handler)`
  - [ ] Implement `registerTagRoutes(r *mux.Router, h *handlers.Handler)`
  - [ ] Implement `registerMailingListRoutes(r *mux.Router, h *handlers.Handler)`
  - [ ] Add comprehensive comments to each group

- [ ] Create `/home/nick/code/Zettelgarden/go-backend/pkg/routes/builder_test.go`
  - [ ] Test BuildRouter() returns non-nil *mux.Router
  - [ ] Test all protected routes require auth
  - [ ] Test all public routes don't require auth
  - [ ] Test route count matches expected (138)
  - [ ] Test specific routes exist (sample: cards CRUD, tasks CRUD)
  - [ ] Test route methods correct (GET, POST, PUT, DELETE)
  - [ ] Test route paths with parameters work

### Modify Existing Files

- [ ] Modify `/home/nick/code/Zettelgarden/go-backend/main.go`
  - [ ] Remove lines 27-32 (addProtectedRoute, addRoute functions)
  - [ ] Replace lines 88-231 with: `r := routes.BuildRouter(h, routes.DefaultOptions())`
  - [ ] Add import: `"go-backend/pkg/routes"`

- [ ] Verify no changes needed to `handlers/handlers.go` (globals still used, will fix in Phase 3)

### Validation

- [ ] Verify main.go compiles
- [ ] Verify no new globals needed
- [ ] Verify all 138 routes registered in BuildRouter
- [ ] Run builder_test.go: `go test ./pkg/routes -v`
- [ ] Verify server still starts: `go run main.go`
- [ ] Verify handler tests still pass

---

## Phase 3: Middleware Composition Layer

**Effort:** 2-3 hours | **Risk:** Low | **Prerequisite:** Phase 1-2 recommended

### Create New Files

- [ ] Create `/home/nick/code/Zettelgarden/go-backend/pkg/middleware/chain.go`
  - [ ] Define `HandlerChain` struct with `middlewares []func(http.HandlerFunc) http.HandlerFunc`
  - [ ] Implement `NewHandlerChain() *HandlerChain`
  - [ ] Implement `Add(mw func(http.HandlerFunc) http.HandlerFunc) *HandlerChain`
  - [ ] Implement `Build(handler http.HandlerFunc) http.HandlerFunc`
  - [ ] Add comments explaining reverse-order application
  - [ ] Add predefined chains: `ProtectedChain()`, `AdminChain()`, `PublicChain()`

- [ ] Create `/home/nick/code/Zettelgarden/go-backend/pkg/middleware/chain_test.go`
  - [ ] Test middleware applied in correct order
  - [ ] Test context propagation through chain
  - [ ] Test empty chain
  - [ ] Test single middleware
  - [ ] Test multiple middlewares
  - [ ] Test predefined chains

- [ ] Create `/home/nick/code/Zettelgarden/go-backend/pkg/middleware/cors.go`
  - [ ] Define `CORSConfig` struct
  - [ ] Implement `WithCORS(handler http.Handler, config CORSConfig) http.Handler`
  - [ ] Move CORS configuration from main.go

### Modify Existing Files

- [ ] Modify `/home/nick/code/Zettelgarden/go-backend/pkg/routes/builder.go`
  - [ ] Use `middleware.ProtectedChain()` instead of inline nesting
  - [ ] Use `middleware.AdminChain()` for admin routes
  - [ ] Use `middleware.PublicChain()` for public routes
  - [ ] Clean up: Remove `addProtectedRoute`, `addRoute` patterns

- [ ] Modify `/home/nick/code/Zettelgarden/go-backend/main.go`
  - [ ] Replace lines 233-240 with: `handler := middleware.WithCORS(r, corsConfig)`
  - [ ] Add import: `"go-backend/pkg/middleware"`

- [ ] Update `pkg/routes/builder_test.go`
  - [ ] Verify chain-based routing still works

### Validation

- [ ] Verify main.go compiles
- [ ] Run chain_test.go: `go test ./pkg/middleware -v`
- [ ] Verify order is correct: "m2, m1, handler"
- [ ] Verify server still starts
- [ ] Verify handler tests still pass

---

## Phase 4: CORS Wrapper Extraction

**Effort:** 1 hour | **Risk:** Very Low | **Prerequisite:** Phase 3

### Files Already Created

- [x] `/home/nick/code/Zettelgarden/go-backend/pkg/middleware/cors.go` (Phase 3)

### Create New Files

- [ ] Create `/home/nick/code/Zettelgarden/go-backend/pkg/middleware/cors_test.go`
  - [ ] Test CORS headers present in response
  - [ ] Test CORS origin validation
  - [ ] Test CORS with invalid origin
  - [ ] Test CORS methods allowed
  - [ ] Test CORS headers allowed
  - [ ] Test OPTIONS request

### Modify Existing Files

- [x] Modified in Phase 3: `/home/nick/code/Zettelgarden/go-backend/main.go`

- [ ] Create `/home/nick/code/Zettelgarden/go-backend/integration/cors_test.go`
  - [ ] Test CORS + Auth integration
  - [ ] Test OPTIONS request bypasses auth
  - [ ] Test preflight requests

### Validation

- [ ] Run cors_test.go: `go test ./pkg/middleware -v`
- [ ] Run integration cors_test.go: `go test ./integration -v`
- [ ] Verify CORS headers in actual requests

---

## Phase 5: Test Suite Expansion

**Effort:** 3-4 hours | **Risk:** Very Low | **Prerequisite:** Phases 1-4

### Create New Files

- [ ] Create `/home/nick/code/Zettelgarden/go-backend/integration/router_test.go`
  - [ ] Test all 138 routes exist
  - [ ] Test protected routes return 401 without auth
  - [ ] Test public routes return not 401
  - [ ] Test route parameters work
  - [ ] Test invalid routes return 404

- [ ] Create `/home/nick/code/Zettelgarden/go-backend/integration/middleware_test.go`
  - [ ] Test JwtMiddleware validation
  - [ ] Test APIKeyOrJWTMiddleware fallback
  - [ ] Test Admin middleware permission check
  - [ ] Test context propagation
  - [ ] Test LogRoute logging (when debug enabled)

- [ ] Create `/home/nick/code/Zettelgarden/go-backend/integration/full_flow_test.go`
  - [ ] Test complete request flow: Auth → Log → Handler
  - [ ] Test error handling through middleware
  - [ ] Test context available to handler
  - [ ] Test response headers from all layers

- [ ] Create `/home/nick/code/Zettelgarden/go-backend/integration/config_test.go`
  - [ ] Test different config variations
  - [ ] Test optional config loading
  - [ ] Test required config validation
  - [ ] Test environment-specific configs

### Enhance Existing Files

- [ ] Modify `/home/nick/code/Zettelgarden/go-backend/tests/conftest.go`
  - [ ] Add `CreateTestHandler()` function
  - [ ] Add `CreateTestRouter()` function
  - [ ] Add middleware test helpers
  - [ ] Add request building helpers
  - [ ] Document new helper patterns

### Optional: Documentation

- [ ] Create `/home/nick/code/Zettelgarden/go-backend/ROUTING.md`
  - [ ] Document route organization by group
  - [ ] Document middleware composition patterns
  - [ ] Document how to add new routes
  - [ ] Document how to add new middleware
  - [ ] Document testing patterns

- [ ] Create `/home/nick/code/Zettelgarden/go-backend/CONFIGURATION.md`
  - [ ] Document all configuration options
  - [ ] Document required vs. optional
  - [ ] Document defaults
  - [ ] Document environment-specific setup

### Validation

- [ ] Run all integration tests: `go test ./integration -v`
- [ ] Verify test count increased by ~30+
- [ ] Verify handler tests still pass
- [ ] Run full test suite: `source .env-bash && go test ./...`
- [ ] Check test coverage: `go test -cover ./...`

---

## Cross-Phase Checklist

### Code Quality

- [ ] No new linter warnings: `golangci-lint run ./...`
- [ ] All code formatted: `go fmt ./...`
- [ ] All imports organized: `goimports -w ./...`

### Documentation

- [ ] Update README.md with new testing patterns
- [ ] Update CLAUDE.md with testability improvements
- [ ] Add comments to new files
- [ ] Document breaking changes (if any)

### Testing

- [ ] Unit tests: `go test ./handlers -v`
- [ ] Service tests: `go test ./services -v`
- [ ] Router tests: `go test ./pkg/routes -v`
- [ ] Middleware tests: `go test ./pkg/middleware -v`
- [ ] Integration tests: `go test ./integration -v`
- [ ] All tests: `source .env-bash && go test ./...`
- [ ] Test coverage > 70%: `go test -cover ./...`

### Backwards Compatibility

- [ ] No breaking changes to handler signatures
- [ ] No breaking changes to API
- [ ] main.go behavior unchanged
- [ ] All existing tests pass
- [ ] Handler tests pass without modification

### Performance

- [ ] Startup time unchanged
- [ ] Memory usage unchanged
- [ ] Request latency unchanged
- [ ] Test execution time reasonable

### Deployment

- [ ] Code compiles: `go build -o main`
- [ ] Docker build works
- [ ] Environment variables documented
- [ ] Migration from old to new is seamless

---

## Rollout Plan

### Week 1: Phase 1-2
- Monday: Phase 1 (config extraction) - 1-2 hours
- Tuesday: Phase 2 (router builder) - 2-3 hours
- Wed-Thu: Review, test, fix issues
- Friday: Deploy to staging

### Week 2: Phase 3-4
- Monday: Phase 3 (middleware chains) - 2-3 hours
- Tuesday: Phase 4 (CORS wrapper) - 1 hour
- Wed-Thu: Integration testing
- Friday: Deploy to staging

### Week 3: Phase 5
- Monday-Wednesday: Phase 5 (test expansion) - 3-4 hours
- Thursday-Friday: Documentation and cleanup

### Week 4: Polish
- Monday-Wednesday: Code review, fixes
- Thursday-Friday: Production deployment

---

## Risk Mitigation

### Low-Risk Items (Phases 1-2)
- ✅ Code extraction (no behavior changes)
- ✅ Tests only verify existing behavior
- ✅ Can rollback easily if issues found

### Medium-Risk Items (Phase 3-4)
- ⚠️ Middleware composition refactoring
- ⚠️ Mitigation: Thorough integration tests before deploy
- ⚠️ Mitigation: Staging environment verification

### Testing Strategy

1. **Unit Tests:** Verify each component independently
2. **Integration Tests:** Verify components work together
3. **Staging Verification:** Run against staging environment
4. **Canary Deployment:** Deploy to 10% of traffic first
5. **Monitoring:** Alert on any request latency increase

---

## Success Criteria

### Before Refactoring
- [ ] Baseline metrics captured:
  - [ ] Startup time: ___ seconds
  - [ ] Test execution time: ___ seconds
  - [ ] Memory usage: ___ MB
  - [ ] Request latency: ___ ms

### After Phase 2
- [ ] main.go reduced to ~100 lines
- [ ] Router builder exists and tested
- [ ] All 138 routes accessible via BuildRouter()
- [ ] No behavior changes

### After Phase 3
- [ ] Middleware composition explicit
- [ ] Middleware ordering testable
- [ ] All middleware tests passing
- [ ] No behavior changes

### After Phase 4
- [ ] CORS independently testable
- [ ] CORS behavior consistent across production and tests
- [ ] No behavior changes

### After Phase 5
- [ ] 30+ new integration tests
- [ ] Test coverage > 70%
- [ ] Router composition tested
- [ ] Middleware integration tested
- [ ] Configuration variations tested

### Final Metrics
- [ ] main.go: 249 lines → ~50 lines (80% reduction)
- [ ] Router registration: implicit → explicit
- [ ] Middleware ordering: implicit → explicit
- [ ] Configuration: scattered → centralized
- [ ] Test coverage: +30+ new tests
- [ ] Testable components: Router + Middleware + Config + Dependencies

---

## Questions to Answer During Implementation

1. **Should BuildRouter() cache routes or recreate each call?**
   - Recommend: No caching for tests, use simple creation

2. **How to test optional services (Typesense)?**
   - Recommend: Make service pointers nullable, add tests for nil checks

3. **How to handle middleware that reads environment variables?**
   - Recommend: Pass config to middleware instead of reading env vars

4. **Should tests use real database or mocks?**
   - Recommend: Keep using real test database (conftest.go pattern)

5. **How to organize routes by feature?**
   - Recommend: One registerXxxRoutes() function per feature

6. **Should CORS be always enabled or configurable?**
   - Recommend: Always enabled, configurable via config struct

---

## Notes

- All estimates assume no major blockers
- Estimated total time: 9-12 hours (can be spread over 3-4 weeks)
- No code changes required until Phase 1 actually starts
- Can pause and resume between phases without issues
- Each phase is independently valuable; stop at any point is acceptable

---

## Sign-Off

**Review completed:** 2026-01-15
**Analysis documents:**
- ✅ `/home/nick/code/Zettelgarden/TESTABILITY_REVIEW.md`
- ✅ `/home/nick/code/Zettelgarden/PATTERNS_REFERENCE.md`
- ✅ `/home/nick/code/Zettelgarden/TESTABILITY_ANALYSIS_SUMMARY.md`
- ✅ `/home/nick/code/Zettelgarden/ARCHITECTURE_DIAGRAMS.md`
- ✅ `/home/nick/code/Zettelgarden/IMPLEMENTATION_CHECKLIST.md` (this file)

**Status:** Ready for Phase 1 implementation

**Next step:** Create implementation beads for each phase (Zettelgarden-9xw.8.1 through 9xw.8.5)
