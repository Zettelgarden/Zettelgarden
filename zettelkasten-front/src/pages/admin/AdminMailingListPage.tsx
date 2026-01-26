import React, { useState, useEffect, useMemo } from "react";
import { getMailingListSubscribers, MailingListSubscriber, unsubscribeMailingList } from "../../api/users";
import {
  useReactTable,
  getCoreRowModel,
  getSortedRowModel,
  getFilteredRowModel,
  createColumnHelper,
  SortingState,
  ColumnDef,
} from "@tanstack/react-table";
import { fuzzyFilter } from "../../utils/tableFilters";
import { StatusBadge } from "../../components/admin/StatusBadge";
import { AdminTableContainer } from "../../components/admin/AdminTable";
import { AdminErrorDisplay } from "../../components/admin/AdminErrorDisplay";

interface ErrorState {
  message: string;
  details?: string;
}

export function AdminMailingListPage() {
  const [subscribers, setSubscribers] = useState<MailingListSubscriber[]>([]);
  const [sorting, setSorting] = useState<SortingState>([]);
  const [globalFilter, setGlobalFilter] = useState("");
  const [isLoading, setIsLoading] = useState(true);
  const [error, setError] = useState<ErrorState | null>(null);

  const fetchSubscribers = async () => {
    setIsLoading(true);
    setError(null);
    try {
      const data = await getMailingListSubscribers();
      setSubscribers(data);
    } catch (err) {
      const message = err instanceof Error ? err.message : "Failed to load subscribers";
      setError({ message, details: err instanceof Error ? err.stack : undefined });
    } finally {
      setIsLoading(false);
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
    setError(null);
    try {
      await unsubscribeMailingList(email);
      // Refresh the subscribers list
      await fetchSubscribers();
    } catch (err) {
      const message = err instanceof Error ? err.message : "Failed to unsubscribe. Please try again.";
      setError({ message, details: err instanceof Error ? err.stack : undefined });
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
      {error && (
        <AdminErrorDisplay
          message={error.message}
          details={error.details}
          severity="error"
          onRetry={fetchSubscribers}
          onDismiss={() => setError(null)}
        />
      )}
      <AdminTableContainer
        title="Mailing List Subscribers"
        table={table}
        searchValue={globalFilter ?? ""}
        onSearchChange={setGlobalFilter}
        searchPlaceholder="Search all columns..."
        isLoading={isLoading}
      />
    </div>
  );
} 