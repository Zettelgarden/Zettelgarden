package handlers

import "testing"

func TestHabitsHandlerCompile(t *testing.T) {
	// Compile check
	if true != false {
		t.Error("handler file should compile")
	}
}
