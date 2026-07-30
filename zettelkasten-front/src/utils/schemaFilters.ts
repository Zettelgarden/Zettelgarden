/**
 * Schema filter utilities for testing and reuse
 */

export interface ParsedFilter {
  operator: string;
  value: string;
}

/**
 * Parse filter operator and value from filter string (e.g., "gte:5", "active", "lte:10")
 */
export function parseFilterValue(filterValue: string): ParsedFilter {
  // Check for operator prefix (gte, lte, gt, lt, ne)
  const operatorMatch = filterValue.match(/^(gte|lte|gt|lt|ne):(.+)$/);
  if (operatorMatch) {
    return { operator: operatorMatch[1], value: operatorMatch[2] };
  }
  // Default is equality
  return { operator: "eq", value: filterValue };
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
  if (typeof value === "number") return null;
  if (typeof value !== "string") return null;
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
  predicate: (a: number, b: number) => boolean
): boolean {
  const cardTime = parseDateValue(cardValue);
  const filterTime = parseDateValue(filterValue);
  if (cardTime !== null && filterTime !== null) {
    return predicate(cardTime, filterTime);
  }

  const numCardValue = typeof cardValue === "number" ? cardValue : parseFloat(cardValue);
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
  if (cardValue === null || cardValue === undefined || cardValue === "") {
    return false;
  }

  switch (operator) {
    case "eq":
      return String(cardValue).toLowerCase() === String(value).toLowerCase();
    case "ne":
      return String(cardValue).toLowerCase() !== String(value).toLowerCase();
    case "gt":
      return compareOrdered(cardValue, value, (a, b) => a > b);
    case "gte":
      return compareOrdered(cardValue, value, (a, b) => a >= b);
    case "lt":
      return compareOrdered(cardValue, value, (a, b) => a < b);
    case "lte":
      return compareOrdered(cardValue, value, (a, b) => a <= b);
    default:
      return String(cardValue).toLowerCase() === String(value).toLowerCase();
  }
}

/**
 * Parse filters string (e.g., "status=active,priority=high") into object
 */
export function parseFiltersString(filtersStr: string): Record<string, string> {
  const result: Record<string, string> = {};
  filtersStr.split(',').forEach(f => {
    const [key, value] = f.split('=').map(s => s.trim());
    if (key && value) {
      result[key] = value;
    }
  });
  return result;
}

/**
 * Apply filters to a card's structured data
 */
export function applyFiltersToCard(
  card: { title?: string; structured_data?: Record<string, any> | null },
  filters: Record<string, string>
): boolean {
  // Check each filter
  for (const [fieldName, filterValue] of Object.entries(filters)) {
    const cardValue = card.structured_data?.[fieldName];

    // Special case for "title" field
    if (fieldName === "title") {
      if (!matchesFilter(card.title || "", filterValue)) {
        return false;
      }
    } else if (!matchesFilter(cardValue, filterValue)) {
      return false;
    }
  }
  return true;
}
