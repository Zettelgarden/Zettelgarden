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

/**
 * Check if a card value matches a filter
 */
export function matchesFilter(cardValue: any, filterValue: string): boolean {
  const { operator, value } = parseFilterValue(filterValue);

  // Handle undefined/null values
  if (cardValue === null || cardValue === undefined || cardValue === "") {
    return false;
  }

  // Convert values for comparison
  const numCardValue = typeof cardValue === "number" ? cardValue : parseFloat(cardValue);
  const numFilterValue = parseFloat(value);

  switch (operator) {
    case "eq":
      return String(cardValue).toLowerCase() === String(value).toLowerCase();
    case "ne":
      return String(cardValue).toLowerCase() !== String(value).toLowerCase();
    case "gt":
      return !isNaN(numCardValue) && !isNaN(numFilterValue) && numCardValue > numFilterValue;
    case "gte":
      return !isNaN(numCardValue) && !isNaN(numFilterValue) && numCardValue >= numFilterValue;
    case "lt":
      return !isNaN(numCardValue) && !isNaN(numFilterValue) && numCardValue < numFilterValue;
    case "lte":
      return !isNaN(numCardValue) && !isNaN(numFilterValue) && numCardValue <= numFilterValue;
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
