package handlers

import "testing"

// TestBuildTypesenseSortBy covers the Typesense sort param mapping used by the
// native search path (Zettelgarden-72f.3).
func TestBuildTypesenseSortBy(t *testing.T) {
	cases := []struct {
		sortBy    string
		sortOrder string
		want      string
	}{
		{"date", "desc", "_text_match:desc,created_at:desc"},
		{"date", "asc", "created_at:asc"},
		{"date", "", "_text_match:desc,created_at:desc"},
		{"name", "asc", "name:asc,created_at:asc"},
		{"name", "desc", "name:desc,created_at:desc"},
		{"size", "desc", "size:desc"},
		{"size", "asc", "size:asc"},
		{"type", "desc", "content_type:desc,created_at:desc"},
		{"card", "asc", "card_pk:asc,created_at:asc"},
	}
	for _, c := range cases {
		got := buildTypesenseSortBy(c.sortBy, c.sortOrder)
		if got != c.want {
			t.Errorf("buildTypesenseSortBy(%q, %q) = %q, want %q", c.sortBy, c.sortOrder, got, c.want)
		}
	}
}

// TestFileIDFromDocument covers extracting file_id from a Typesense hit
// document, which can arrive as either a JSON number or an int32.
func TestFileIDFromDocument(t *testing.T) {
	if id, ok := fileIDFromDocument(map[string]interface{}{"file_id": float64(42)}); !ok || id != 42 {
		t.Errorf("float64 file_id: got (%d, %v), want (42, true)", id, ok)
	}
	if id, ok := fileIDFromDocument(map[string]interface{}{"file_id": int32(7)}); !ok || id != 7 {
		t.Errorf("int32 file_id: got (%d, %v), want (7, true)", id, ok)
	}
	if _, ok := fileIDFromDocument(map[string]interface{}{"name": "x"}); ok {
		t.Errorf("missing file_id should return ok=false")
	}
}

func TestBuildTypesenseFilterBy(t *testing.T) {
	if got := buildTypesenseFilterBy(3, ""); got != "user_id:3" {
		t.Errorf("no tag: got %q, want %q", got, "user_id:3")
	}
	if got := buildTypesenseFilterBy(3, "receipts"); got != "user_id:3 && tags:=receipts" {
		t.Errorf("with tag: got %q, want %q", got, "user_id:3 && tags:=receipts")
	}
}
