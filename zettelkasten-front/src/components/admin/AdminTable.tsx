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
}: AdminTableProps<TData>) {
  const headerGroups = table.getHeaderGroups();
  const rows = table.getRowModel().rows;

  // Show loading skeleton
  if (isLoading) {
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

  // Show table with data
  return (
    <div className="overflow-x-auto">
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
}

/**
 * AdminTableContainer - A complete table container with title and search.
 *
 * Combines the table with a header section containing title and search input.
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
}: AdminTableContainerProps<TData>) {
  return (
    <div className="container mx-auto px-4">
      <div className="flex justify-between items-center mb-4">
        <h1 className="text-2xl font-bold">{title}</h1>
        <div className="flex items-center gap-2">
          <input
            type="text"
            value={searchValue}
            onChange={(e) => onSearchChange(e.target.value)}
            className="px-4 py-2 border rounded-lg"
            placeholder={searchPlaceholder}
          />
          {headerContent}
        </div>
      </div>
      <AdminTable
        table={table}
        isLoading={isLoading}
        emptyMessage={emptyMessage}
        skeletonRows={skeletonRows}
        className={className}
      />
    </div>
  );
}
