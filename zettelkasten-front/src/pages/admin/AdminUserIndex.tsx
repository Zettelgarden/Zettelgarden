import React, { useState, useEffect, useMemo } from 'react';
import { getUsers, GetUsersResponse } from '../../api/users';
import { User } from '../../models/User';
import {
  useReactTable,
  getCoreRowModel,
  getSortedRowModel,
  getFilteredRowModel,
  createColumnHelper,
  SortingState,
  ColumnDef,
} from '@tanstack/react-table';
import { Link } from 'react-router-dom';
import { fuzzyFilter } from '../../utils/tableFilters';
import { Badge, getSubscriptionStatusBadge } from '../../components/ui/Badge';
import { AdminTableContainer } from '../../components/admin/AdminTable';
import { AdminErrorDisplay } from '../../components/admin/AdminErrorDisplay';

interface ErrorState {
  message: string;
  details?: string;
}

const PER_PAGE = 50;

export function AdminUserIndex() {
  const [users, setUsers] = useState<User[]>([]);
  const [sorting, setSorting] = useState<SortingState>([
    { id: 'last_seen', desc: true },
  ]);
  const [globalFilter, setGlobalFilter] = useState('');
  const [isLoading, setIsLoading] = useState(true);
  const [error, setError] = useState<ErrorState | null>(null);
  const [page, setPage] = useState(1);
  const [totalUsers, setTotalUsers] = useState(0);
  const [totalPages, setTotalPages] = useState(0);

  useEffect(() => {
    const fetchUsers = async () => {
      setIsLoading(true);
      setError(null);
      try {
        const response: GetUsersResponse = await getUsers({
          page,
          per_page: PER_PAGE,
        });
        setUsers(response.users);
        setTotalUsers(response.pagination.total);
        setTotalPages(response.pagination.total_pages);
      } catch (err) {
        const message =
          err instanceof Error ? err.message : 'Failed to load users';
        setError({
          message,
          details: err instanceof Error ? err.stack : undefined,
        });
      } finally {
        setIsLoading(false);
      }
    };
    fetchUsers();
  }, [page]);

  const columnHelper = createColumnHelper<User>();

  const columns = useMemo<ColumnDef<User, any>[]>(
    () => [
      columnHelper.accessor('id', {
        header: 'ID',
        cell: (info) => info.getValue(),
      }),
      columnHelper.accessor('username', {
        header: 'Name',
        cell: (info) => (
          <Link
            to={`/admin/user/${info.row.original.id}`}
            className={`hover:text-blue-800 ${
              info.row.original.is_admin ? 'text-purple-600' : 'text-blue-600'
            }`}
          >
            {info.getValue()}
          </Link>
        ),
      }),
      columnHelper.accessor('last_seen', {
        header: 'Last Seen',
        cell: (info) =>
          info.getValue()
            ? new Date(info.getValue()).toLocaleString()
            : 'Never',
      }),
      columnHelper.accessor('email_validated', {
        header: 'Email Validated',
        cell: (info) => (
          <Badge color={info.getValue() ? 'success' : 'warning'} dot>
            {info.getValue() ? 'Verified' : 'Pending'}
          </Badge>
        ),
      }),
      columnHelper.accessor('stripe_subscription_status', {
        header: 'Subscription',
        cell: (info) => {
          const badge = getSubscriptionStatusBadge(info.getValue() as string);
          return <Badge color={badge.color}>{badge.label}</Badge>;
        },
      }),
      columnHelper.accessor('created_at', {
        header: 'Created At',
        cell: (info) => new Date(info.getValue()).toLocaleString(),
      }),
      columnHelper.accessor('card_count', {
        header: 'Cards',
        cell: (info) => info.getValue(),
      }),
      columnHelper.accessor('task_count', {
        header: 'Tasks',
        cell: (info) => info.getValue(),
      }),
      columnHelper.accessor('file_count', {
        header: 'Files',
        cell: (info) => info.getValue(),
      }),
      columnHelper.accessor('chat_message_count', {
        header: 'Chats',
        cell: (info) => info.getValue(),
      }),
      columnHelper.accessor('revenue', {
        header: 'Revenue',
        cell: (info) => `$${Number(info.getValue() || 0).toFixed(2)}`,
      }),
      columnHelper.accessor('llm_cost', {
        header: 'Cost',
        cell: (info) => `$${Number(info.getValue() || 0).toFixed(4)}`,
      }),
    ],
    [],
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
        searchValue={globalFilter ?? ''}
        onSearchChange={setGlobalFilter}
        searchPlaceholder="Search all columns..."
        isLoading={isLoading}
        hideOnMobile={[
          'revenue',
          'llm_cost',
          'file_count',
          'task_count',
          'chat_message_count',
        ]}
        sorting={sorting}
        onSortingChange={setSorting}
      />
      {/* Pagination controls - only show if there are multiple pages */}
      {totalPages > 1 && (
        <div className="mt-4 flex flex-col sm:flex-row items-center justify-between gap-3 border-t pt-4">
          <div className="text-sm text-slate-600">
            Showing {(page - 1) * PER_PAGE + 1}-
            {Math.min(page * PER_PAGE, totalUsers)} of {totalUsers} users
          </div>
          <div className="flex items-center gap-2">
            <button
              onClick={() => setPage(1)}
              disabled={page === 1}
              className="px-3 py-1 text-sm border border-slate-300 rounded disabled:opacity-50 disabled:cursor-not-allowed hover:bg-slate-50"
            >
              First
            </button>
            <button
              onClick={() => setPage((prev) => Math.max(1, prev - 1))}
              disabled={page === 1}
              className="px-3 py-1 text-sm border border-slate-300 rounded disabled:opacity-50 disabled:cursor-not-allowed hover:bg-slate-50"
            >
              Previous
            </button>
            <span className="text-sm text-slate-600">
              Page {page} of {totalPages}
            </span>
            <button
              onClick={() => setPage((prev) => Math.min(totalPages, prev + 1))}
              disabled={page === totalPages}
              className="px-3 py-1 text-sm border border-slate-300 rounded disabled:opacity-50 disabled:cursor-not-allowed hover:bg-slate-50"
            >
              Next
            </button>
            <button
              onClick={() => setPage(totalPages)}
              disabled={page === totalPages}
              className="px-3 py-1 text-sm border border-slate-300 rounded disabled:opacity-50 disabled:cursor-not-allowed hover:bg-slate-50"
            >
              Last
            </button>
          </div>
        </div>
      )}
    </div>
  );
}
