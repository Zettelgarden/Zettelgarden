// NOTE: Tests for memory domain functions require a test database setup.
//
// The memory package functions (GetUserMemory, UpdateUserMemory) interact
// with PostgreSQL and require proper database fixtures for testing.
//
// Future test implementation should include:
// - Table-driven tests for GetUserMemory with various database states
// - Table-driven tests for UpdateUserMemory with insert and update scenarios
// - Error handling tests for database connection failures
// - Integration tests with test database fixtures
//
// For now, the functionality is verified through integration tests in
// services/memory_tools_test.go which test the full tool registration
// and handler execution paths.
package memory
