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
import {
  StatusBadge,
  getSubscriptionStatusBadge,
} from '../../components/admin/StatusBadge';
import { AdminTableContainer } from '../../components/admin/AdminTable';
import { AdminErrorDisplay } from '../../components/admin/AdminErrorDisplay';

interface ErrorState {
  message: string;
  details?: string;
}

// Helper to get ALL users by fetching with high per_page
const fetchAllUsers = async (): Promise<User[]> => {
  const allUsers: User[] = [];
  let page = 1;
  const PER_PAGE = 100;
  let hasMore = true;

  while (hasMore) {
    const response: GetUsersResponse = await getUsers({
      page,
      per_page: PER_PAGE,
    });
    allUsers.push(...response.users);
    hasMore = page < response.pagination.total_pages;
    page++;
  }

  return allUsers;
};

// Sort users by last_seen (most recent first, nulls last)
const sortUsersByLastSeen = (users: User[]): User[] => {
  return [...users].sort((a, b) => {
    const aTime = a.last_seen ? new Date(a.last_seen).getTime() : 0;
    const bTime = b.last_seen ? new Date(b.last_seen).getTime() : 0;
    if (aTime === 0 && bTime === 0) return 0;
    if (aTime === 0) return 1; // nulls last
    if (bTime === 0) return -1; // nulls last
    return bTime - aTime; // descending
  });
};

const TOP_N = 10;

export function AdminDashboard() {
  const [users, setUsers] = useState<User[]>([]);
  const [sorting, setSorting] = useState<SortingState>([
    { id: 'last_seen', desc: true },
  ]);
  const [globalFilter, setGlobalFilter] = useState('');
  const [isLoading, setIsLoading] = useState(true);
  const [error, setError] = useState<ErrorState | null>(null);
  const [totalCount, setTotalCount] = useState(0);

  useEffect(() => {
    const fetchDashboardUsers = async () => {
      setIsLoading(true);
      setError(null);
      try {
        const allUsers = await fetchAllUsers();
        setTotalCount(allUsers.length);
        const sorted = sortUsersByLastSeen(allUsers);
        setUsers(sorted.slice(0, TOP_N));
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
    fetchDashboardUsers();
  }, []);

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
          <StatusBadge
            value={info.getValue()}
            type={info.getValue() ? 'success' : 'warning'}
            label={info.getValue() ? 'Verified' : 'Pending'}
          />
        ),
      }),
      columnHelper.accessor('stripe_subscription_status', {
        header: 'Subscription',
        cell: (info) => {
          const badge = getSubscriptionStatusBadge(info.getValue() as string);
          return <StatusBadge type={badge.type} label={badge.label} />;
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
      <div className="mb-4">
        <h1 className="text-2xl font-bold text-slate-800">Dashboard</h1>
        <p className="text-slate-600">
          Showing top {TOP_N} most recently active users of {totalCount} total
        </p>
      </div>

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
        title="Recent Users"
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
    </div>
  );
}
