package output

import (
	"encoding/json"
	"io"
	"os"
	"sync"
)

var (
	prettyPrint bool
	prettyMutex sync.Mutex
)

// SetPretty sets whether to pretty-print JSON output
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

// Response is the standard API response structure
type Response struct {
	Success bool        `json:"success"`
	Data    any         `json:"data,omitempty"`
	Error   string      `json:"error,omitempty"`
	Details string      `json:"details,omitempty"`
	Total   int         `json:"total,omitempty"`
	Limit   int         `json:"limit,omitempty"`
	Offset  int         `json:"offset,omitempty"`
}

// encodeJSON encodes the response with or without indentation
func encodeJSON(w io.Writer, v any) error {
	if isPretty() {
		encoder := json.NewEncoder(w)
		encoder.SetIndent("", "  ")
		return encoder.Encode(v)
	}
	return json.NewEncoder(w).Encode(v)
}

// WriteSuccess writes a success response with data
func WriteSuccess(w io.Writer, data any) error {
	return encodeJSON(w, Response{
		Success: true,
		Data:    data,
	})
}

// WriteError writes an error response
func WriteError(w io.Writer, errMsg string, details string) error {
	return encodeJSON(w, Response{
		Success: false,
		Error:   errMsg,
		Details: details,
	})
}

// WriteList writes a list response with pagination
func WriteList(w io.Writer, items any, total, limit, offset int) error {
	return encodeJSON(w, Response{
		Success: true,
		Data:    items,
		Total:   total,
		Limit:   limit,
		Offset:  offset,
	})
}

// IsTTY returns true if output is a terminal (for auto pretty-print in future)
func IsTTY() bool {
	fileInfo, _ := os.Stdout.Stat()
	return (fileInfo.Mode() & os.ModeCharDevice) != 0
}
