package output

import (
	"fmt"
	"io"
	"reflect"
	"strings"
)

// writeHuman outputs data in human-readable format
func writeHuman(w io.Writer, data any) error {
	// Check if type implements custom formatter
	if hf, ok := data.(HumanFormatter); ok {
		fmt.Fprintln(w, hf.FormatHuman())
		return nil
	}

	// Handle common types
	switch v := data.(type) {
	case map[string]any:
		return formatMap(w, v)
	case []any:
		return formatSlice(w, v)
	default:
		// Check if it's a slice using reflection
		rv := reflect.ValueOf(data)
		if rv.Kind() == reflect.Slice {
			return formatReflectSlice(w, rv)
		}
		// Last resort: just print as-is with basic formatting
		fmt.Fprintf(w, "%v\n", data)
		return nil
	}
}

// formatMap formats a map as key: value pairs
func formatMap(w io.Writer, m map[string]any) error {
	// Special handling for simple message maps
	if msg, ok := m["message"]; ok && len(m) == 1 {
		fmt.Fprintf(w, "%v\n", msg)
		return nil
	}

	for key, val := range m {
		fmt.Fprintf(w, "%s: %v\n", key, val)
	}
	return nil
}

// formatSlice formats a slice of any types
func formatSlice(w io.Writer, s []any) error {
	if len(s) == 0 {
		fmt.Fprintln(w, "(empty)")
		return nil
	}

	for i, item := range s {
		if hf, ok := item.(HumanFormatter); ok {
			fmt.Fprintln(w, hf.FormatHuman())
		} else {
			fmt.Fprintf(w, "%d. %v\n", i+1, item)
		}
	}
	return nil
}

// formatReflectSlice handles slices of typed data
func formatReflectSlice(w io.Writer, rv reflect.Value) error {
	if rv.Len() == 0 {
		fmt.Fprintln(w, "(empty)")
		return nil
	}

	// Check if elements implement ListFormatter
	var header string
	var items []string

	for i := 0; i < rv.Len(); i++ {
		item := rv.Index(i).Interface()

		// Get header from first item if available
		if i == 0 {
			if hf, ok := item.(HeaderFormatter); ok {
				header = hf.FormatListHeader()
			}
		}

		if lf, ok := item.(ListFormatter); ok {
			items = append(items, lf.FormatListItem())
		} else if hf, ok := item.(HumanFormatter); ok {
			items = append(items, hf.FormatHuman())
		} else {
			items = append(items, fmt.Sprintf("%v", item))
		}
	}

	if header != "" {
		fmt.Fprintln(w, header)
		fmt.Fprintln(w, strings.Repeat("-", len(header)))
	}

	for _, item := range items {
		fmt.Fprintln(w, item)
	}

	return nil
}

// writeHumanList outputs a list with pagination info
func writeHumanList(w io.Writer, items any, total, limit, offset int) error {
	rv := reflect.ValueOf(items)
	if rv.Kind() != reflect.Slice {
		// Not a slice, just format normally
		return writeHuman(w, items)
	}

	if rv.Len() == 0 {
		fmt.Fprintln(w, "(no results)")
		return nil
	}

	// Format the slice
	if err := formatReflectSlice(w, rv); err != nil {
		return err
	}

	// Add pagination info if relevant
	if total > limit {
		fmt.Fprintf(w, "\nShowing %d-%d of %d\n", offset+1, min(offset+limit, total), total)
	}

	return nil
}

// writeHumanMessage outputs a simple message
func writeHumanMessage(w io.Writer, msg string) error {
	fmt.Fprintln(w, msg)
	return nil
}

func min(a, b int) int {
	if a < b {
		return a
	}
	return b
}
