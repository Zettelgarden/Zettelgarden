# Admin Dashboard Implementation Plan

> **For Claude:** REQUIRED SUB-SKILL: Use superpowers:executing-plans to implement this plan task-by-task.

**Goal:** Create an admin dashboard at `/admin` showing the 10 most recently active users, move the full user list to `/admin/users`.

**Architecture:** New React component that fetches users via existing API, sorts client-side by `last_seen`, displays top 10 in existing table format. Route updates in AdminPage.tsx.

**Tech Stack:** React, TypeScript, React Router, TanStack Table, existing admin components

---

### Task 1: Create AdminDashboard component

**Files:**
- Create: `zettelkasten-front/src/pages/admin/AdminDashboard.tsx`

**Step 1: Create the AdminDashboard component**

Copy this code to create the new dashboard component:

```tsx
import React, { useState, useEffect, useMemo } from "react";
import { getUsers, GetUsersResponse } from "../../api/users";
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
import { StatusBadge, getSubscriptionStatusBadge } from "../../components/admin/StatusBadge";
import { AdminTableContainer } from "../../components/admin/AdminTable";
import { AdminErrorDisplay } from "../../components/admin/AdminErrorDisplay";

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
    const response: GetUsersResponse = await getUsers({ page, per_page: PER_PAGE });
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
    { id: "last_seen", desc: true }
  ]);
  const [globalFilter, setGlobalFilter] = useState("");
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
        const message = err instanceof Error ? err.message : "Failed to load users";
        setError({ message, details: err instanceof Error ? err.stack : undefined });
      } finally {
        setIsLoading(false);
      }
    };
    fetchDashboardUsers();
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
        searchValue={globalFilter ?? ""}
        onSearchChange={setGlobalFilter}
        searchPlaceholder="Search all columns..."
        isLoading={isLoading}
        hideOnMobile={["revenue", "llm_cost", "file_count", "task_count", "chat_message_count"]}
        sorting={sorting}
        onSortingChange={setSorting}
      />
    </div>
  );
}
```

**Step 2: Add missing import**

The code above uses `fuzzyFilter` but it's not imported. Add this import at the top:

```typescript
import { fuzzyFilter } from "../../utils/tableFilters";
```

**Step 3: Commit**

```bash
git add zettelkasten-front/src/pages/admin/AdminDashboard.tsx
git commit -m "feat(admin): add AdminDashboard component with top 10 recent users"
```

---

### Task 2: Update AdminPage routing

**Files:**
- Modify: `zettelkasten-front/src/pages/admin/AdminPage.tsx:1-20`
- Modify: `zettelkasten-front/src/pages/admin/AdminPage.tsx:168-177`

**Step 1: Add AdminDashboard import**

Add this import after the existing admin page imports (around line 13):

```typescript
import { AdminDashboard } from "./AdminDashboard";
```

**Step 2: Update the Routes section**

Replace the Routes section (lines 168-177) with:

```tsx
<Routes>
  <Route path="/" element={<AdminDashboard />} />
  <Route path="users" element={<AdminUserIndex />} />
  <Route path="user/:id" element={<AdminUserDetailPage />} />
  <Route path="user/:id/edit" element={<AdminEditUserPage />} />
  <Route path="job-queue" element={<AdminJobQueuePage />} />
  <Route path="scheduler" element={<AdminSchedulerPage />} />
  <Route path="mailing-list" element={<AdminMailingListPage />} />
  <Route path="mailing-list/send" element={<AdminMailingListSendPage />} />
  <Route path="mailing-list/history" element={<AdminMailingListHistoryPage />} />
</Routes>
```

**Step 3: Commit**

```bash
git add zettelkasten-front/src/pages/admin/AdminPage.tsx
git commit -m "feat(admin): add dashboard route and move users to /admin/users"
```

---

### Task 3: Update sidebar navigation

**Files:**
- Modify: `zettelkasten-front/src/pages/admin/AdminPage.tsx:94-158`

**Step 1: Update sidebar links**

Replace the `<nav>` section (lines 94-158 approximately) with this updated nav:

```tsx
<nav className="px-4">
  <ul className="space-y-2">
    <li>
      <Link
        to="/admin"
        className="block py-3 px-4 rounded-lg hover:bg-gray-700 text-gray-300 hover:text-white transition-colors min-h-[44px] flex items-center"
        onClick={() => setIsSidebarOpen(false)}
      >
        📊 Dashboard
      </Link>
    </li>
    <li>
      <Link
        to="/admin/users"
        className="block py-3 px-4 rounded-lg hover:bg-gray-700 text-gray-300 hover:text-white transition-colors min-h-[44px] flex items-center"
        onClick={() => setIsSidebarOpen(false)}
      >
        👥 All Users
      </Link>
    </li>
    <li>
      <Link
        to="/admin/job-queue"
        className="block py-3 px-4 rounded-lg hover:bg-gray-700 text-gray-300 hover:text-white transition-colors min-h-[44px] flex items-center"
        onClick={() => setIsSidebarOpen(false)}
      >
        ⚙️ Job Queue
      </Link>
    </li>
    <li>
      <Link
        to="/admin/scheduler"
        className="block py-3 px-4 rounded-lg hover:bg-gray-700 text-gray-300 hover:text-white transition-colors min-h-[44px] flex items-center"
        onClick={() => setIsSidebarOpen(false)}
      >
        ⏰ Scheduled Jobs
      </Link>
    </li>
    <li>
      <Link
        to="/admin/mailing-list"
        className="block py-3 px-4 rounded-lg hover:bg-gray-700 text-gray-300 hover:text-white transition-colors min-h-[44px] flex items-center"
        onClick={() => setIsSidebarOpen(false)}
      >
        📧 Mailing List Subscribers
      </Link>
    </li>
    <li>
      <Link
        to="/admin/mailing-list/send"
        className="block py-3 px-4 rounded-lg hover:bg-gray-700 text-gray-300 hover:text-white transition-colors min-h-[44px] flex items-center"
        onClick={() => setIsSidebarOpen(false)}
      >
        ✉️ Send Message
      </Link>
    </li>
    <li>
      <Link
        to="/admin/mailing-list/history"
        className="block py-3 px-4 rounded-lg hover:bg-gray-700 text-gray-300 hover:text-white transition-colors min-h-[44px] flex items-center"
        onClick={() => setIsSidebarOpen(false)}
      >
        📜 Message History
      </Link>
    </li>
    <li className="pt-4 border-t border-gray-700 mt-4">
      <Link
        to="/app"
        className="block py-3 px-4 rounded-lg hover:bg-gray-700 text-gray-400 hover:text-white transition-colors min-h-[44px] flex items-center"
        onClick={() => setIsSidebarOpen(false)}
      >
        ← Back to App
      </Link>
    </li>
  </ul>
</nav>
```

**Step 2: Commit**

```bash
git add zettelkasten-front/src/pages/admin/AdminPage.tsx
git commit -m "feat(admin): update sidebar with Dashboard and All Users links"
```

---

### Task 4: Test the implementation

**Step 1: Start the frontend dev server**

```bash
cd zettelkasten-front
npm start
```

**Step 2: Verify functionality**

1. Navigate to `/admin` - should see dashboard with top 10 users by last_seen
2. Click "Dashboard" in sidebar - should stay on dashboard
3. Click "All Users" in sidebar - should go to `/admin/users` with full user list
4. Click on a user name - should navigate to user detail page
5. Use search/filter - should work within the top 10 users
6. Sort by columns - should work within the top 10 users

**Step 3: Edge case checks**

- If you have fewer than 10 users, all should be displayed
- Users with `null` last_seen should appear at the bottom
- Loading and error states should display correctly

**Step 4: Commit any fixes**

If issues found and fixed:

```bash
git add zettelkasten-front/src/pages/admin/
git commit -m "fix(admin): address issues found during testing"
```

---

## Summary

After implementation:
- `/admin` → Dashboard showing top 10 users by last_seen
- `/admin/users` → Full paginated user list
- Sidebar has both "Dashboard" and "All Users" links
- Same table styling and functionality across both views
