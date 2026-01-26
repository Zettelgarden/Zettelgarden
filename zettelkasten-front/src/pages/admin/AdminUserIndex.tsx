import React, { useState, useEffect, useMemo } from "react";
import { getUsers } from "../../api/users";
import { User } from "../../models/User";
import {
  useReactTable,
  getCoreRowModel,
  getSortedRowModel,
  getFilteredRowModel,
  createColumnHelper,
  SortingState,
  ColumnDef,
} from "@tanstack/react-table";
import { Link } from "react-router-dom";
import { fuzzyFilter } from "../../utils/tableFilters";
import { StatusBadge, getSubscriptionStatusBadge } from "../../components/admin/StatusBadge";
import { AdminTableContainer } from "../../components/admin/AdminTable";
import { AdminErrorDisplay } from "../../components/admin/AdminErrorDisplay";

interface ErrorState {
  message: string;
  details?: string;
}

export function AdminUserIndex() {
  const [users, setUsers] = useState<User[]>([]);
  const [sorting, setSorting] = useState<SortingState>([
    { id: "last_seen", desc: true }
  ]);
  const [globalFilter, setGlobalFilter] = useState("");
  const [isLoading, setIsLoading] = useState(true);
  const [error, setError] = useState<ErrorState | null>(null);

  useEffect(() => {
    const fetchUsers = async () => {
      setIsLoading(true);
      setError(null);
      try {
        const tempUsers = await getUsers();
        setUsers(tempUsers);
      } catch (err) {
        const message = err instanceof Error ? err.message : "Failed to load users";
        setError({ message, details: err instanceof Error ? err.stack : undefined });
      } finally {
        setIsLoading(false);
      }
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
          <StatusBadge
            value={info.getValue()}
            type={info.getValue() ? "success" : "warning"}
            label={info.getValue() ? "Verified" : "Pending"}
          />
        ),
      }),
      columnHelper.accessor("stripe_subscription_status", {
        header: "Subscription",
        cell: (info) => {
          const badge = getSubscriptionStatusBadge(info.getValue() as string);
          return <StatusBadge type={badge.type} label={badge.label} />;
        },
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
      {error && (
        <AdminErrorDisplay
          message={error.message}
          details={error.details}
          severity="error"
          onRetry={() => window.location.reload()}
          onDismiss={() => setError(null)}
        />
      )}
      <AdminTableContainer
        title="Users"
        table={table}
        searchValue={globalFilter ?? ""}
        onSearchChange={setGlobalFilter}
        searchPlaceholder="Search all columns..."
        isLoading={isLoading}
        hideOnMobile={["revenue", "llm_cost", "file_count", "task_count", "chat_message_count"]}
      />
    </div>
  );
}
