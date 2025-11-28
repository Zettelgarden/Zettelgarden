import React, { useState, useEffect, useMemo } from "react";
import { getUsers } from "../../api/users";
import { User } from "../../models/User";
import {
  useReactTable,
  getCoreRowModel,
  getSortedRowModel,
  getFilteredRowModel,
  flexRender,
  createColumnHelper,
  FilterFn,
  SortingState,
  ColumnDef,
} from "@tanstack/react-table";
import { Link } from "react-router-dom";
import { rankItem } from "@tanstack/match-sorter-utils";

const fuzzyFilter: FilterFn<any> = (row, columnId, value, addMeta) => {
  const itemRank = rankItem(row.getValue(columnId), value);
  addMeta({ itemRank });
  return itemRank.passed;
};

export function AdminUserIndex() {
  const [users, setUsers] = useState<User[]>([]);
  const [sorting, setSorting] = useState<SortingState>([
    { id: "last_seen", desc: true }
  ]);
  const [globalFilter, setGlobalFilter] = useState("");

  useEffect(() => {
    const fetchUsers = async () => {
      let tempUsers = await getUsers();
      setUsers(tempUsers);
    };
    fetchUsers();
  }, []);

  const columnHelper = createColumnHelper<User>();

  const columns = useMemo<ColumnDef<User, any>[]>(
    () => [
      columnHelper.accessor("id", {
        header: "ID",
        cell: (info) => info.getValue(),
      }),
      columnHelper.accessor("username", {
        header: "Name",
        cell: (info) => (
          <Link
            to={`/admin/user/${info.row.original.id}`}
            className={`hover:text-blue-800 ${info.row.original.is_admin ? "text-purple-600" : "text-blue-600"
              }`}
          >
            {info.getValue()}
          </Link>
        ),
      }),
      columnHelper.accessor("last_seen", {
        header: "Last Seen",
        cell: (info) => info.getValue() ? new Date(info.getValue()).toLocaleString() : 'Never',
      }),
      columnHelper.accessor("email_validated", {
        header: "Email Validated",
        cell: (info) => (
          <span
            className={`px-2 py-1 rounded text-sm ${info.getValue()
                ? "bg-green-100 text-green-800"
                : "bg-yellow-100 text-yellow-800"
              }`}
          >
            {info.getValue() ? "Verified" : "Pending"}
          </span>
        ),
      }),
      columnHelper.accessor("stripe_subscription_status", {
        header: "Subscription",
        cell: (info) => (
          <span
            className={`px-2 py-1 rounded text-sm ${info.getValue() === "active"
                ? "bg-green-100 text-green-800"
                : info.getValue() === "trialing"
                  ? "bg-blue-100 text-blue-800"
                  : "bg-red-100 text-red-800"
              }`}
          >
            {info.getValue()}
          </span>
        ),
      }),
      columnHelper.accessor("created_at", {
        header: "Created At",
        cell: (info) => new Date(info.getValue()).toLocaleString(),
      }),
      columnHelper.accessor("card_count", {
        header: "Cards",
        cell: (info) => info.getValue(),
      }),
      columnHelper.accessor("task_count", {
        header: "Tasks",
        cell: (info) => info.getValue(),
      }),
      columnHelper.accessor("file_count", {
        header: "Files",
        cell: (info) => info.getValue(),
      }),
      columnHelper.accessor("chat_message_count", {
        header: "Chats",
        cell: (info) => info.getValue(),
      }),
      columnHelper.accessor("revenue", {
        header: "Revenue",
        cell: (info) => `$${Number(info.getValue() || 0).toFixed(2)}`,
      }),
      columnHelper.accessor("llm_cost", {
        header: "Cost",
        cell: (info) => `$${Number(info.getValue() || 0).toFixed(4)}`,
      }),
    ],
    []
  );

  const table = useReactTable({
    data: users,
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
        <h1 className="text-2xl font-bold">Users</h1>
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
