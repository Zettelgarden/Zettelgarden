package services

import (
	"testing"
)

// TestNewLoopDetector verifies that a new loop detector starts at iteration 0
func TestNewLoopDetector(t *testing.T) {
	ld := NewLoopDetector()

	if ld.GetIteration() != 0 {
		t.Errorf("Expected initial iteration count 0, got %d", ld.GetIteration())
	}
}

// TestIncrement verifies that Increment increases the counter
func TestIncrement(t *testing.T) {
	ld := NewLoopDetector()

	// First increment
	shouldBlock := ld.Increment()
	if ld.GetIteration() != 1 {
		t.Errorf("Expected iteration count 1, got %d", ld.GetIteration())
	}
	if shouldBlock {
		t.Errorf("Expected NOT to block after 1 iteration")
	}

	// Second increment
	shouldBlock = ld.Increment()
	if ld.GetIteration() != 2 {
		t.Errorf("Expected iteration count 2, got %d", ld.GetIteration())
	}
	if shouldBlock {
		t.Errorf("Expected NOT to block after 2 iterations")
	}
}

// TestIncrementMaxIterations verifies that Increment returns true at max iterations
func TestIncrementMaxIterations(t *testing.T) {
	ld := NewLoopDetector()

	// Increment up to max iterations (10)
	for i := 0; i < 9; i++ {
		shouldBlock := ld.Increment()
		if shouldBlock {
			t.Errorf("Expected NOT to block at iteration %d", i+1)
		}
	}

	// 10th increment should trigger block
	shouldBlock := ld.Increment()
	if !shouldBlock {
		t.Errorf("Expected to block at iteration 10 (max iterations)")
	}
	if ld.GetIteration() != 10 {
		t.Errorf("Expected iteration count 10, got %d", ld.GetIteration())
	}
}

// TestReset verifies that Reset clears the counter
func TestReset(t *testing.T) {
	ld := NewLoopDetector()

	// Increment a few times
	for i := 0; i < 5; i++ {
		ld.Increment()
	}

	if ld.GetIteration() != 5 {
		t.Errorf("Expected iteration count 5 before reset, got %d", ld.GetIteration())
	}

	// Reset
	ld.Reset()

	if ld.GetIteration() != 0 {
		t.Errorf("Expected iteration count 0 after reset, got %d", ld.GetIteration())
	}

	// After reset, can increment again
	ld.Increment()
	if ld.GetIteration() != 1 {
		t.Errorf("Expected iteration count 1 after reset + increment, got %d", ld.GetIteration())
	}
}

// TestResetAfterBlock verifies that Reset clears the counter even after blocking
func TestResetAfterBlock(t *testing.T) {
	ld := NewLoopDetector()

	// Increment to max iterations
	for i := 0; i < 10; i++ {
		ld.Increment()
	}

	// Should be blocking
	if ld.GetIteration() != 10 {
		t.Errorf("Expected iteration count 10 at max, got %d", ld.GetIteration())
	}

	// Reset
	ld.Reset()

	if ld.GetIteration() != 0 {
		t.Errorf("Expected iteration count 0 after reset, got %d", ld.GetIteration())
	}
}

// TestGetIteration verifies that GetIteration returns the current count
func TestGetIteration(t *testing.T) {
	ld := NewLoopDetector()

	if ld.GetIteration() != 0 {
		t.Errorf("Expected initial iteration count 0, got %d", ld.GetIteration())
	}

	ld.Increment()
	if ld.GetIteration() != 1 {
		t.Errorf("Expected iteration count 1, got %d", ld.GetIteration())
	}

	ld.Increment()
	ld.Increment()
	if ld.GetIteration() != 3 {
		t.Errorf("Expected iteration count 3, got %d", ld.GetIteration())
	}
}

// TestGetInterventionMessage verifies the intervention message
func TestGetInterventionMessage(t *testing.T) {
	ld := NewLoopDetector()

	msg := ld.GetInterventionMessage()
	if msg == "" {
		t.Errorf("Expected non-empty intervention message")
	}

	expectedMsg := "I'm having trouble completing this task. Could you rephrase your request?"
	if msg != expectedMsg {
		t.Errorf("Expected message '%s', got '%s'", expectedMsg, msg)
	}
}

// TestConcurrentIncrement verifies thread safety of Increment
func TestConcurrentIncrement(t *testing.T) {
	ld := NewLoopDetector()
	const goroutines = 100
	done := make(chan bool)

	for i := 0; i < goroutines; i++ {
		go func() {
			ld.Increment()
			done <- true
		}()
	}

	// Wait for all goroutines
	for i := 0; i < goroutines; i++ {
		<-done
	}

	if ld.GetIteration() != goroutines {
		t.Errorf("Expected iteration count %d, got %d", goroutines, ld.GetIteration())
	}
}

// TestConcurrentReset verifies thread safety of Reset
func TestConcurrentReset(t *testing.T) {
	ld := NewLoopDetector()
	const goroutines = 50
	done := make(chan bool)

	// Half increment, half reset
	for i := 0; i < goroutines; i++ {
		if i%2 == 0 {
			go func() {
				ld.Increment()
				done <- true
			}()
		} else {
			go func() {
				ld.Reset()
				done <- true
			}()
		}
	}

	// Wait for all goroutines
	for i := 0; i < goroutines; i++ {
		<-done
	}

	// We can't predict exact count due to race conditions, but it should be valid
	count := ld.GetIteration()
	if count < 0 || count > goroutines/2 {
		t.Errorf("Iteration count %d is outside expected range [0, %d]", count, goroutines/2)
	}
}

// TestMaxIterationsDefault verifies the default max iterations is 10
func TestMaxIterationsDefault(t *testing.T) {
	ld := NewLoopDetector()

	// Should not block at 9
	for i := 0; i < 9; i++ {
		if ld.Increment() {
			t.Errorf("Should not block before iteration 10")
		}
	}

	// Should block at 10
	if !ld.Increment() {
		t.Errorf("Should block at iteration 10")
	}
}
