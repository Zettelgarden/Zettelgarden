package output

import (
	"encoding/json"
	"io"
)

// encodeJSON encodes the response with or without indentation
func encodeJSON(w io.Writer, v any) error {
	if isPretty() {
		encoder := json.NewEncoder(w)
		encoder.SetIndent("", "  ")
		return encoder.Encode(v)
	}
	return json.NewEncoder(w).Encode(v)
}

// writeJSONResponse wraps data in a success response and writes JSON
func writeJSONResponse(w io.Writer, data any) error {
	return encodeJSON(w, Response{
		Success: true,
		Data:    data,
	})
}

// writeJSONError wraps error in a response and writes JSON
func writeJSONError(w io.Writer, errMsg string, details string) error {
	return encodeJSON(w, Response{
		Success: false,
		Error:   errMsg,
		Details: details,
	})
}

// writeJSONList wraps a list in a response with pagination and writes JSON
func writeJSONList(w io.Writer, items any, total, limit, offset int) error {
	return encodeJSON(w, Response{
		Success: true,
		Data:    items,
		Total:   total,
		Limit:   limit,
		Offset:  offset,
	})
}
