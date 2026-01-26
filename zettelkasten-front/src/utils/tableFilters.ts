import { rankItem } from "@tanstack/match-sorter-utils";
import { FilterFn } from "@tanstack/react-table";

/**
 * Fuzzy filter function for TanStack Table.
 * Uses rankItem to determine if a row matches the filter value.
 *
 * @example
 * ```tsx
 * const table = useReactTable({
 *   data,
 *   columns,
 *   state: { globalFilter },
 *   onGlobalFilterChange: setGlobalFilter,
 *   globalFilterFn: fuzzyFilter,
 *   // ...
 * });
 * ```
 */
export const fuzzyFilter: FilterFn<any> = (
  row,
  columnId,
  value,
  addMeta
) => {
  const itemRank = rankItem(row.getValue(columnId), value);
  addMeta({ itemRank });
  return itemRank.passed;
};
