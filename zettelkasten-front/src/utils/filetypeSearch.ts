export interface ParsedFiletypeQuery {
  /** The search text with any filetype: tokens removed */
  searchText: string;
  /** The file type from the first filetype: token, or null when absent */
  filetype: string | null;
}

/**
 * Parses the FileVault search-box syntax `filetype:TYPE` (e.g. `filetype:pdf`,
 * `filetype:image/png`) out of the raw input. The remaining text becomes the
 * filename search term, so `filetype:pdf quarterly` filters by pdf and searches
 * for "quarterly". The first token wins when multiple are present.
 */
export function parseFiletypeFilter(input: string): ParsedFiletypeQuery {
  const regex = /filetype:(\S+)/g;
  const matches = Array.from(input.matchAll(regex));
  if (matches.length === 0) {
    return { searchText: input.trim(), filetype: null };
  }
  const filetype = matches[0][1];
  const searchText = input.replace(regex, '').trim();
  return { searchText, filetype };
}
