package output

// HumanFormatter is implemented by types that want custom human-readable output.
// When --pretty is enabled, FormatHuman() is called instead of JSON output.
type HumanFormatter interface {
	FormatHuman() string
}

// ListFormatter is optionally implemented for custom list item formatting.
// Falls back to a default table format if not implemented.
type ListFormatter interface {
	FormatListItem() string
}

// HeaderFormatter is optionally implemented to provide column headers for lists.
type HeaderFormatter interface {
	FormatListHeader() string
}
