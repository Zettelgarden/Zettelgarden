package services

import (
	"log"
	"sync"
)

// LoopDetector prevents infinite loops by tracking iteration count
type LoopDetector struct {
	mu            sync.RWMutex
	iteration     int
	maxIterations int
}

// NewLoopDetector creates a new loop detector with default max iterations
func NewLoopDetector() *LoopDetector {
	return &LoopDetector{
		iteration:     0,
		maxIterations: 10,
	}
}

// Increment increases the iteration counter and returns true if limit reached
func (ld *LoopDetector) Increment() bool {
	ld.mu.Lock()
	defer ld.mu.Unlock()

	ld.iteration++
	log.Printf("[LoopDetector] Iteration %d/%d", ld.iteration, ld.maxIterations)

	return ld.iteration >= ld.maxIterations
}

// Reset clears the iteration counter
func (ld *LoopDetector) Reset() {
	ld.mu.Lock()
	defer ld.mu.Unlock()

	ld.iteration = 0
	log.Printf("[LoopDetector] Reset")
}

// GetIteration returns the current iteration count
func (ld *LoopDetector) GetIteration() int {
	ld.mu.RLock()
	defer ld.mu.RUnlock()
	return ld.iteration
}

// GetInterventionMessage returns the user-friendly message when limit is reached
func (ld *LoopDetector) GetInterventionMessage() string {
	return "I'm having trouble completing this task. Could you rephrase your request?"
}
