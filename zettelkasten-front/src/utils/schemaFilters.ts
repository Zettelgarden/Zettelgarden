/**
 * Schema filter utilities for testing and reuse
 */

export interface ParsedFilter {
  operator: string;
  value: string;
}

/**
 * Encode characters that have structural meaning in the filter directive
 * syntax so they can appear literally inside values.
 *
 * Backslash escaping cannot be used here: CommonMark (the markdown parser
 * that processes card bodies) strips `\` before punctuation, so `\\|` would
 * be turned back into a raw `|` before the schema-table parser sees it.
 * Percent-encoding survives markdown untouched.
 *
 * Encodes: `%` -> `%25`, `|` -> `%7C`, `,` -> `%2C`, `=` -> `%3D`, `:` -> `%3A`
 *
 * @example encodeFilterValue('a|b') -> 'a%7Cb'
 */
export function encodeFilterValue(value: string): string {
  return value
    .replace(/%/g, '%25')
    .replace(/\|/g, '%7C')
    .replace(/,/g, '%2C')
    .replace(/=/g, '%3D')
    .replace(/:/g, '%3A');
}

/**
 * Decode characters encoded with {@link encodeFilterValue}.
 *
 * @example decodeFilterValue('a%7Cb') -> 'a|b'
 */
export function decodeFilterValue(value: string): string {
  return value.replace(/%25|%7C|%2C|%3D|%3A/gi, (match) => {
    switch (match.toLowerCase()) {
      case '%25':
        return '%';
      case '%7c':
        return '|';
      case '%2c':
        return ',';
      case '%3d':
        return '=';
      case '%3a':
        return ':';
      default:
        return match;
    }
  });
}

/**
 * Parse filter operator and value from filter string (e.g., "gte:5", "active", "lte:10")
 * The operator prefix is matched on the raw (encoded) value, so an encoded
 * literal like "gt%3A5" stays a literal "gt:5", not a greater-than filter.
 * The value is then percent-decoded.
 */
export function parseFilterValue(filterValue: string): ParsedFilter {
  // Check for operator prefix (gte, lte, gt, lt, ne)
  const operatorMatch = filterValue.match(/^(gte|lte|gt|lt|ne):(.+)$/);
  if (operatorMatch) {
    return {
      operator: operatorMatch[1],
      value: decodeFilterValue(operatorMatch[2]),
    };
  }
  // Default is equality
  return { operator: 'eq', value: decodeFilterValue(filterValue) };
}

/** Matches ISO-style dates: YYYY-MM-DD, optionally followed by a time component. */
const ISO_DATE_PATTERN =
  /^\d{4}-\d{2}-\d{2}(?:[T\s]\d{2}:\d{2}(?::\d{2})?(?:\.\d+)?(?:Z|[+-]\d{2}:?\d{2})?)?$/;

/**
 * Parse a value into a numeric timestamp, but ONLY when it looks like a date.
 * Plain numbers and bare years are intentionally rejected so that numeric
 * fields keep using numeric comparison in {@link compareOrdered}.
 */
function parseDateValue(value: any): number | null {
  if (value instanceof Date) {
    const t = value.getTime();
    return isNaN(t) ? null : t;
  }
  if (typeof value === 'number') return null;
  if (typeof value !== 'string') return null;
  const str = value.trim();
  if (!ISO_DATE_PATTERN.test(str)) return null;
  const t = Date.parse(str);
  return isNaN(t) ? null : t;
}

/**
 * Evaluate gt/gte/lt/lte. Dates are compared first (so ISO date strings like
 * "2026-01-01" compare chronologically rather than being parsed as the leading
 * year), then numbers, then the comparison returns false (preserving the
 * historical "non-numeric values never match inequality filters" behavior).
 */
function compareOrdered(
  cardValue: any,
  filterValue: string,
  predicate: (a: number, b: number) => boolean,
): boolean {
  const cardTime = parseDateValue(cardValue);
  const filterTime = parseDateValue(filterValue);
  if (cardTime !== null && filterTime !== null) {
    return predicate(cardTime, filterTime);
  }

  const numCardValue =
    typeof cardValue === 'number' ? cardValue : parseFloat(cardValue);
  const numFilterValue = parseFloat(filterValue);
  if (!isNaN(numCardValue) && !isNaN(numFilterValue)) {
    return predicate(numCardValue, numFilterValue);
  }

  return false;
}

/**
 * Check if a card value matches a filter
 */
export function matchesFilter(cardValue: any, filterValue: string): boolean {
  const { operator, value } = parseFilterValue(filterValue);

  // Handle undefined/null values
  if (cardValue === null || cardValue === undefined || cardValue === '') {
    return false;
  }

  switch (operator) {
    case 'eq':
      return String(cardValue).toLowerCase() === String(value).toLowerCase();
    case 'ne':
      return String(cardValue).toLowerCase() !== String(value).toLowerCase();
    case 'gt':
      return compareOrdered(cardValue, value, (a, b) => a > b);
    case 'gte':
      return compareOrdered(cardValue, value, (a, b) => a >= b);
    case 'lt':
      return compareOrdered(cardValue, value, (a, b) => a < b);
    case 'lte':
      return compareOrdered(cardValue, value, (a, b) => a <= b);
    default:
      return String(cardValue).toLowerCase() === String(value).toLowerCase();
  }
}

/**
 * Parse filters string (e.g., "status=active,priority=high") into object.
 * Splits on `,` and the first `=` per segment, so raw `=` inside a value is
 * kept ("title=a=b" -> value "a=b"). Values with `,` or `|` must be
 * percent-encoded (e.g. "title=a%2Cb") - see {@link encodeFilterValue}.
 * Keys are decoded; values stay encoded until {@link parseFilterValue}.
 */
export function parseFiltersString(filtersStr: string): Record<string, string> {
  const result: Record<string, string> = {};
  filtersStr.split(',').forEach((f) => {
    const eqIndex = f.indexOf('=');
    if (eqIndex === -1) return;
    const key = decodeFilterValue(f.slice(0, eqIndex).trim());
    const value = f.slice(eqIndex + 1).trim();
    if (key && value) {
      result[key] = value;
    }
  });
  return result;
}

/**
 * Parse a filter string into groups of AND conditions, where `||` separates
 * OR groups and `,` separates AND conditions within a group.
 *
 * Values containing `|`, `,`, `=`, `:` must be percent-encoded
 * (e.g. "title=a%7C%7Cb||status=done" for a literal "a||b").
 *
 * Examples:
 * - "status=active,priority=high"                 -> [{ status: 'active', priority: 'high' }]
 * - "status=active||priority=high"                -> [{ status: 'active' }, { priority: 'high' }]
 * - "status=active,priority=high||status=done"    -> (status=active AND priority=high) OR (status=done)
 */
export function parseFilterGroups(
  filterStr: string,
): Array<Record<string, string>> {
  if (!filterStr) return [];
  return filterStr
    .split('||')
    .map((group) => group.trim())
    .filter((group) => group.length > 0)
    .map((group) => parseFiltersString(group));
}

/**
 * Apply filters to a card's structured data
 */
export function applyFiltersToCard(
  card: { title?: string; structured_data?: Record<string, any> | null },
  filters: Record<string, string>,
): boolean {
  // Check each filter
  for (const [fieldName, filterValue] of Object.entries(filters)) {
    const cardValue = card.structured_data?.[fieldName];

    // Special case for "title" field
    if (fieldName === 'title') {
      if (!matchesFilter(card.title || '', filterValue)) {
        return false;
      }
    } else if (!matchesFilter(cardValue, filterValue)) {
      return false;
    }
  }
  return true;
}

/**
 * Apply filter groups to a card: a card matches if ANY group matches, where
 * each group is AND-ed internally (OR of AND groups, i.e. DNF).
 */
export function applyFilterGroupsToCard(
  card: { title?: string; structured_data?: Record<string, any> | null },
  groups: Array<Record<string, string>>,
): boolean {
  if (!groups || groups.length === 0) return true;
  return groups.some((group) => applyFiltersToCard(card, group));
}
