import React, { useState, useEffect, useMemo } from "react";
import { getMailingListSubscribers, MailingListSubscriber, unsubscribeMailingList } from "../../api/users";
import {
  useReactTable,
  getCoreRowModel,
  getSortedRowModel,
  getFilteredRowModel,
  flexRender,
  createColumnHelper,
  SortingState,
  ColumnDef,
} from "@tanstack/react-table";
import { fuzzyFilter } from "../../utils/tableFilters";
import { StatusBadge } from "../../components/admin/StatusBadge";

export function AdminMailingListPage() {
  const [subscribers, setSubscribers] = useState<MailingListSubscriber[]>([]);
  const [sorting, setSorting] = useState<SortingState>([]);
  const [globalFilter, setGlobalFilter] = useState("");
  const [isLoading, setIsLoading] = useState(false);

  const fetchSubscribers = async () => {
    try {
      const data = await getMailingListSubscribers();
      setSubscribers(data);
    } catch (error) {
      console.error("Error fetching subscribers:", error);
    }
  };

  useEffect(() => {
    fetchSubscribers();
  }, []);

  const handleUnsubscribe = async (email: string) => {
    if (!window.confirm(`Are you sure you want to unsubscribe ${email}?`)) {
      return;
    }

    setIsLoading(true);
    try {
      await unsubscribeMailingList(email);
      // Refresh the subscribers list
      await fetchSubscribers();
    } catch (error) {
      console.error("Error unsubscribing:", error);
      alert("Failed to unsubscribe. Please try again.");
    } finally {
      setIsLoading(false);
    }
  };

  const columnHelper = createColumnHelper<MailingListSubscriber>();

  const columns = useMemo<ColumnDef<MailingListSubscriber, any>[]>(
    () => [
      columnHelper.accessor("id", {
        header: "ID",
        cell: (info) => info.getValue(),
      }),
      columnHelper.accessor("email", {
        header: "Email",
        cell: (info) => info.getValue(),
      }),
      columnHelper.accessor("subscribed", {
        header: "Status",
        cell: (info) => (
          <StatusBadge
            value={info.getValue()}
            type={info.getValue() ? "success" : "error"}
            label={info.getValue() ? "Subscribed" : "Unsubscribed"}
          />
        ),
      }),
      columnHelper.accessor("welcome_email_sent", {
        header: "Welcome Email",
        cell: (info) => (
          <StatusBadge
            value={info.getValue()}
            type={info.getValue() ? "info" : "warning"}
            label={info.getValue() ? "Sent" : "Pending"}
          />
        ),
      }),
      columnHelper.accessor("has_account", {
        header: "Account Status",
        cell: (info) => (
          <StatusBadge
            value={info.getValue()}
            type={info.getValue() ? "info" : "neutral"}
            label={info.getValue() ? "Has Account" : "No Account"}
          />
        ),
      }),
      columnHelper.accessor("created_at", {
        header: "Created At",
        cell: (info) => new Date(info.getValue()).toLocaleString(),
      }),
      columnHelper.accessor("updated_at", {
        header: "Updated At",
        cell: (info) => new Date(info.getValue()).toLocaleString(),
      }),
      columnHelper.display({
        id: "actions",
        header: "Actions",
        cell: (info) => (
          <button
            onClick={() => handleUnsubscribe(info.row.original.email)}
            disabled={!info.row.original.subscribed || isLoading}
            className={`px-3 py-1 rounded text-sm ${
              !info.row.original.subscribed || isLoading
                ? "bg-gray-100 text-gray-400 cursor-not-allowed"
                : "bg-red-500 text-white hover:bg-red-600"
            }`}
          >
            {isLoading ? "..." : "Unsubscribe"}
          </button>
        ),
      }),
    ],
    [isLoading]
  );

  const table = useReactTable({
    data: subscribers,
    columns,
    state: {
      sorting,
      globalFilter,
    },
    onSortingChange: setSorting,
    onGlobalFilterChange: setGlobalFilter,
    globalFilterFn: fuzzyFilter,
    getCoreRowModel: getCoreRowModel(),
    getSortedRowModel: getSortedRowModel(),
    getFilteredRowModel: getFilteredRowModel(),
  });

  return (
    <div className="container mx-auto px-4">
      <div className="flex justify-between items-center mb-4">
        <h1 className="text-2xl font-bold">Mailing List Subscribers</h1>
        <input
          type="text"
          value={globalFilter ?? ""}
          onChange={(e) => setGlobalFilter(e.target.value)}
          className="px-4 py-2 border rounded-lg"
          placeholder="Search all columns..."
        />
      </div>
      <div className="overflow-x-auto">
        <table className="min-w-full bg-white shadow-md rounded">
          <thead className="bg-gray-800 text-white">
            {table.getHeaderGroups().map((headerGroup) => (
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
                    {{
                      asc: " 🔼",
                      desc: " 🔽",
                    }[header.column.getIsSorted() as string] ?? null}
                  </th>
                ))}
              </tr>
            ))}
          </thead>
          <tbody>
            {table.getRowModel().rows.map((row) => (
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
    </div>
  );
} 