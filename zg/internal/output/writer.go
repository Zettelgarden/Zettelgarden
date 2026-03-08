package output

import (
	"io"
	"os"
	"sync"
)

var (
	prettyPrint bool
	prettyMutex sync.Mutex
)

// SetPretty sets whether to use human-readable output
func SetPretty(v bool) {
	prettyMutex.Lock()
	defer prettyMutex.Unlock()
	prettyPrint = v
}

// isPretty returns the current pretty-print setting
func isPretty() bool {
	prettyMutex.Lock()
	defer prettyMutex.Unlock()
	return prettyPrint
}

// Response is the standard API response structure (for JSON mode)
type Response struct {
	Success bool   `json:"success"`
	Data    any    `json:"data,omitempty"`
	Error   string `json:"error,omitempty"`
	Details string `json:"details,omitempty"`
	Total   int    `json:"total,omitempty"`
	Limit   int    `json:"limit,omitempty"`
	Offset  int    `json:"offset,omitempty"`
}

// WriteSuccess writes a success response with data.
// In JSON mode: wraps in {"success":true,"data":...}
// In pretty mode: uses human-readable formatting
func WriteSuccess(w io.Writer, data any) error {
	if isPretty() {
		return writeHuman(w, data)
	}
	return writeJSONResponse(w, data)
}

// WriteError writes an error response.
// In JSON mode: wraps in {"success":false,"error":...}
// In pretty mode: shows error with optional details
func WriteError(w io.Writer, errMsg string, details string) error {
	if isPretty() {
		if details != "" {
			return writeHumanMessage(w, "Error: "+errMsg+"\n  "+details)
		}
		return writeHumanMessage(w, "Error: "+errMsg)
	}
	return writeJSONError(w, errMsg, details)
}

// WriteList writes a list response with pagination.
// In JSON mode: wraps with pagination metadata
// In pretty mode: shows as formatted table with pagination info
func WriteList(w io.Writer, items any, total, limit, offset int) error {
	if isPretty() {
		return writeHumanList(w, items, total, limit, offset)
	}
	return writeJSONList(w, items, total, limit, offset)
}

// WriteMessage writes a simple success message.
// Useful for operations like delete/update that don't return data.
func WriteMessage(w io.Writer, msg string) error {
	if isPretty() {
		return writeHumanMessage(w, msg)
	}
	return writeJSONResponse(w, map[string]string{"message": msg})
}

// IsTTY returns true if output is a terminal (for potential auto-detection)
func IsTTY() bool {
	fileInfo, _ := os.Stdout.Stat()
	return (fileInfo.Mode() & os.ModeCharDevice) != 0
}
