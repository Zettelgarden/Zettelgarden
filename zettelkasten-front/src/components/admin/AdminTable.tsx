import React from "react";
import {
  Table,
  flexRender,
  RowModel,
  SortingState,
  ColumnDef,
} from "@tanstack/react-table";

/**
 * Props for the AdminTable component
 */
export interface AdminTableProps<TData> {
  /** The TanStack Table instance */
  table: Table<TData>;
  /** Optional additional CSS classes for the table container */
  className?: string;
  /** Optional loading state */
  isLoading?: boolean;
  /** Optional empty state message */
  emptyMessage?: string;
  /** Number of rows to show as skeleton when loading */
  skeletonRows?: number;
  /** Columns to hide on mobile (by column ID) */
  hideOnMobile?: string[];
}

/**
 * Sorting indicator emoji for table headers
 */
const SORT_INDICATORS = {
  asc: " 🔼",
  desc: " 🔽",
} as const;

/**
 * AdminTable component - A reusable table component for admin pages.
 *
 * Provides consistent styling and behavior for all admin tables including:
 * - Sortable headers with visual indicators
 * - Hover effects on rows
 * - Consistent padding and borders
 * - Loading skeleton states
 * - Empty state handling
 * - Mobile-responsive card view for small screens
 *
 * @example
 * ```tsx
 * const table = useReactTable({ data, columns, ... });
 * return <AdminTable table={table} />;
 * ```
 */
export function AdminTable<TData>({
  table,
  className = "",
  isLoading = false,
  emptyMessage = "No data available",
  skeletonRows = 5,
  hideOnMobile = [],
}: AdminTableProps<TData>) {
  const headerGroups = table.getHeaderGroups();
  const rows = table.getRowModel().rows;

  // Filter columns for mobile view (hide those marked as hideOnMobile)
  const mobileHeaders = headerGroups[0]?.headers.filter(
    header => !hideOnMobile.includes(header.id)
  ) || [];

  // Show loading skeleton
  if (isLoading) {
    return (
      <>
        {/* Desktop table skeleton */}
        <div className="hidden md:block overflow-x-auto">
          <table className={`min-w-full bg-white shadow-md rounded ${className}`}>
            <thead className="bg-gray-800 text-white">
              {headerGroups.map((headerGroup) => (
                <tr key={headerGroup.id}>
                  {headerGroup.headers.map((header) => (
                    <th
                      key={header.id}
                      className="py-2 px-4 text-left"
                    >
                      {flexRender(
                        header.column.columnDef.header,
                        header.getContext()
                      )}
                    </th>
                  ))}
                </tr>
              ))}
            </thead>
            <tbody>
              {Array.from({ length: skeletonRows }).map((_, index) => (
                <tr key={`skeleton-${index}`} className="border-b">
                  {headerGroups[0]?.headers.map((header, cellIndex) => (
                    <td key={cellIndex} className="py-2 px-4">
                      <div className="animate-pulse bg-gray-200 h-4 rounded" />
                    </td>
                  ))}
                </tr>
              ))}
            </tbody>
          </table>
        </div>
        {/* Mobile card skeleton */}
        <div className="md:hidden space-y-4">
          {Array.from({ length: skeletonRows }).map((_, index) => (
            <div key={`mobile-skeleton-${index}`} className="bg-white rounded-lg shadow p-4">
              <div className="animate-pulse bg-gray-200 h-5 w-3/4 rounded mb-3" />
              <div className="animate-pulse bg-gray-200 h-4 w-1/2 rounded mb-2" />
              <div className="animate-pulse bg-gray-200 h-4 w-1/3 rounded" />
            </div>
          ))}
        </div>
      </>
    );
  }

  // Show empty state
  if (rows.length === 0) {
    return (
      <div className="overflow-x-auto">
        <table className={`min-w-full bg-white shadow-md rounded ${className}`}>
          <thead className="bg-gray-800 text-white">
            {headerGroups.map((headerGroup) => (
              <tr key={headerGroup.id}>
                {headerGroup.headers.map((header) => (
                  <th
                    key={header.id}
                    className="py-2 px-4 text-left"
                  >
                    {flexRender(
                      header.column.columnDef.header,
                      header.getContext()
                    )}
                  </th>
                ))}
              </tr>
            ))}
          </thead>
          <tbody>
            <tr>
              <td
                colSpan={headerGroups[0]?.headers.length || 1}
                className="py-8 px-4 text-center text-gray-500"
              >
                {emptyMessage}
              </td>
            </tr>
          </tbody>
        </table>
      </div>
    );
  }

  return (
    <>
      {/* Desktop table view */}
      <div className="hidden md:block overflow-x-auto">
        <table className={`min-w-full bg-white shadow-md rounded ${className}`}>
          <thead className="bg-gray-800 text-white">
            {headerGroups.map((headerGroup) => (
              <tr key={headerGroup.id}>
                {headerGroup.headers.map((header) => (
                  <th
                    key={header.id}
                    className="py-2 px-4 text-left cursor-pointer select-none"
                    onClick={header.column.getToggleSortingHandler()}
                  >
                    {flexRender(
                      header.column.columnDef.header,
                      header.getContext()
                    )}
                    {SORT_INDICATORS[header.column.getIsSorted() as string] ?? null}
                  </th>
                ))}
              </tr>
            ))}
          </thead>
          <tbody>
            {rows.map((row) => (
              <tr key={row.id} className="border-b hover:bg-gray-100">
                {row.getVisibleCells().map((cell) => (
                  <td key={cell.id} className="py-2 px-4">
                    {flexRender(cell.column.columnDef.cell, cell.getContext())}
                  </td>
                ))}
              </tr>
            ))}
          </tbody>
        </table>
      </div>

      {/* Mobile card view */}
      <div className="md:hidden space-y-4">
        {rows.map((row) => (
          <div key={row.id} className="bg-white rounded-lg shadow-md overflow-hidden">
            <div className="divide-y divide-gray-200">
              {mobileHeaders.map((header) => {
                const cell = row.getVisibleCells().find(
                  c => c.column.id === header.id
                );
                if (!cell) return null;
                return (
                  <div key={header.id} className="px-4 py-3">
                    <div className="text-xs text-gray-500 font-medium mb-1">
                      {flexRender(
                        header.column.columnDef.header,
                        header.getContext()
                      )}
                    </div>
                    <div className="text-sm">
                      {flexRender(cell.column.columnDef.cell, cell.getContext())}
                    </div>
                  </div>
                );
              })}
            </div>
          </div>
        ))}
      </div>
    </>
  );
}

/**
 * Props for the AdminTableContainer component that wraps table with search
 */
export interface AdminTableContainerProps<TData> extends AdminTableProps<TData> {
  /** Table title */
  title: string;
  /** Current search/filter value */
  searchValue: string;
  /** Search input change handler */
  onSearchChange: (value: string) => void;
  /** Search input placeholder */
  searchPlaceholder?: string;
  /** Additional header content (e.g., action buttons) */
  headerContent?: React.ReactNode;
  /** Columns to hide on mobile */
  hideOnMobile?: string[];
  /** Current sorting state (for mobile sort controls) */
  sorting?: SortingState;
  /** Sorting state change handler */
  onSortingChange?: (sorting: SortingState) => void;
}

/**
 * AdminTableContainer - A complete table container with title and search.
 *
 * Combines the table with a header section containing title and search input.
 * Mobile-responsive with collapsible search on small screens.
 *
 * @example
 * ```tsx
 * return (
 *   <AdminTableContainer
 *     title="Users"
 *     table={table}
 *     searchValue={globalFilter ?? ""}
 *     onSearchChange={(v) => setGlobalFilter(v)}
 *     searchPlaceholder="Search all columns..."
 *     hideOnMobile={["revenue", "llm_cost"]}
 *   />
 * );
 * ```
 */
export function AdminTableContainer<TData>({
  title,
  table,
  searchValue,
  onSearchChange,
  searchPlaceholder = "Search...",
  headerContent,
  isLoading,
  emptyMessage,
  skeletonRows,
  className,
  hideOnMobile,
  sorting,
  onSortingChange,
}: AdminTableContainerProps<TData>) {
  const [showMobileSearch, setShowMobileSearch] = React.useState(false);
  const [showMobileSort, setShowMobileSort] = React.useState(false);

  return (
    <div className="container mx-auto px-4">
      {/* Title and search header */}
      <div className="flex flex-col sm:flex-row sm:justify-between sm:items-center mb-4 gap-3">
        <h1 className="text-xl sm:text-2xl font-bold">{title}</h1>
        <div className="flex items-center gap-2">
          {/* Desktop search */}
          <div className="hidden sm:block">
            <input
              type="text"
              value={searchValue}
              onChange={(e) => onSearchChange(e.target.value)}
              className="px-4 py-2 border rounded-lg w-64"
              placeholder={searchPlaceholder}
            />
          </div>
          {/* Mobile search toggle */}
          <button
            onClick={() => setShowMobileSearch(!showMobileSearch)}
            className="sm:hidden p-2 bg-white border border-gray-300 rounded-lg"
            aria-label="Search"
          >
            🔍
          </button>
          {/* Mobile sort toggle - only show if sorting is controlled */}
          {onSortingChange && sorting !== undefined && (
            <button
              onClick={() => setShowMobileSort(!showMobileSort)}
              className="sm:hidden p-2 bg-white border border-gray-300 rounded-lg"
              aria-label="Sort"
            >
              ⇅
            </button>
          )}
          {headerContent}
        </div>
      </div>

      {/* Mobile search input */}
      {showMobileSearch && (
        <div className="sm:hidden mb-4">
          <input
            type="text"
            value={searchValue}
            onChange={(e) => onSearchChange(e.target.value)}
            className="w-full px-4 py-2 border rounded-lg"
            placeholder={searchPlaceholder}
            autoFocus
            onBlur={() => setShowMobileSearch(false)}
          />
        </div>
      )}

      {/* Mobile sort dropdown */}
      {showMobileSort && onSortingChange && sorting !== undefined && (
        <div className="sm:hidden mb-4 p-4 bg-white rounded-lg border">
          <div className="mb-3">
            <label className="block text-sm font-medium text-gray-700 mb-1">
              Sort by
            </label>
            <select
              className="w-full px-3 py-2 border rounded-lg"
              value={sorting[0]?.id ?? ""}
              onChange={(e) => {
                const columnId = e.target.value;
                if (columnId) {
                  onSortingChange([{ id: columnId, desc: false }]);
                }
              }}
            >
              <option value="">Select column...</option>
              {table.getAllColumns().map((column) => (
                <option key={column.id} value={column.id}>
                  {typeof column.columnDef.header === "string"
                    ? column.columnDef.header
                    : column.id}
                </option>
              ))}
            </select>
          </div>
          {sorting[0] && (
            <div className="flex gap-2">
              <button
                onClick={() => onSortingChange([{ id: sorting[0].id, desc: false }])}
                className={`flex-1 px-3 py-2 border rounded-lg text-sm ${
                  !sorting[0].desc
                    ? "bg-blue-500 text-white"
                    : "bg-white hover:bg-gray-50"
                }`}
              >
                Ascending ↑
              </button>
              <button
                onClick={() => onSortingChange([{ id: sorting[0].id, desc: true }])}
                className={`flex-1 px-3 py-2 border rounded-lg text-sm ${
                  sorting[0].desc
                    ? "bg-blue-500 text-white"
                    : "bg-white hover:bg-gray-50"
                }`}
              >
                Descending ↓
              </button>
            </div>
          )}
        </div>
      )}

      <AdminTable
        table={table}
        isLoading={isLoading}
        emptyMessage={emptyMessage}
        skeletonRows={skeletonRows}
        className={className}
        hideOnMobile={hideOnMobile}
      />
    </div>
  );
}
