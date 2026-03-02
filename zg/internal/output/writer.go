package output

import (
	"encoding/json"
	"io"
	"os"
)

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

// WriteSuccess writes a success response with data
func WriteSuccess(w io.Writer, data any) error {
	return json.NewEncoder(w).Encode(Response{
		Success: true,
		Data:    data,
	})
}

// WriteError writes an error response
func WriteError(w io.Writer, errMsg string, details string) error {
	return json.NewEncoder(w).Encode(Response{
		Success: false,
		Error:   errMsg,
		Details: details,
	})
}

// WriteList writes a list response with pagination
func WriteList(w io.Writer, items any, total, limit, offset int) error {
	return json.NewEncoder(w).Encode(Response{
		Success: true,
		Data:    items,
		Total:   total,
		Limit:   limit,
		Offset:  offset,
	})
}

// IsTTY returns true if output is a terminal (for pretty-printing)
func IsTTY() bool {
	fileInfo, _ := os.Stdout.Stat()
	return (fileInfo.Mode() & os.ModeCharDevice) != 0
}
