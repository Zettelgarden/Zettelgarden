/**
 * Tests for auth hooks (src/hooks/queries/useAuth.ts)
 */

import { describe, it, expect, beforeEach, vi } from 'vitest';
import { renderHook, waitFor } from '@testing-library/react';
import { QueryClient, QueryClientProvider } from '@tanstack/react-query';
import {
  useCurrentUser,
  useIsAdmin,
  useUserSubscription,
  useUpdateUser,
} from './useAuth';
import {
  getCurrentUser,
  checkAdmin,
  getUserSubscription,
  updateUser,
} from '../../api/users';
import { User, UserSubscription } from '../../models/User';

vi.mock('../../api/users', () => ({
  getCurrentUser: vi.fn(),
  checkAdmin: vi.fn(),
  getUserSubscription: vi.fn(),
  updateUser: vi.fn(),
}));

function createTestQueryClient() {
  return new QueryClient({
    defaultOptions: {
      queries: { retry: false, staleTime: 0 },
      mutations: { retry: false },
    },
  });
}

function wrapper({ children }: { children: React.ReactNode }) {
  return (
    <QueryClientProvider client={createTestQueryClient()}>
      {children}
    </QueryClientProvider>
  );
}

function makeUser(overrides: Partial<User> = {}): User {
  return {
    id: 1,
    username: 'tester',
    email: 'tester@example.com',
    password: '',
    created_at: '2024-01-01T00:00:00Z',
    updated_at: '2024-01-01T00:00:00Z',
    is_admin: false,
    last_login: '2024-01-01T00:00:00Z',
    last_seen: '2024-01-01T00:00:00Z',
    email_validated: true,
    can_upload_files: true,
    max_file_storage: 100,
    stripe_customer_id: '',
    stripe_subscription_id: '',
    stripe_subscription_status: '',
    stripe_subscription_frequency: '',
    stripe_current_plan: '',
    is_active: true,
    dashboard_card_pk: 0,
    card_count: 0,
    task_count: 0,
    file_count: 0,
    chat_message_count: 0,
    llm_cost: 0,
    revenue: 0,
    has_seen_getting_started: true,
    timezone: 'UTC',
    show_tasks: true,
    show_rss: true,
    ...overrides,
  };
}

describe('useCurrentUser', () => {
  beforeEach(() => {
    vi.clearAllMocks();
  });

  it('fetches the current user', async () => {
    const user = makeUser();
    vi.mocked(getCurrentUser).mockResolvedValue(user);

    const { result } = renderHook(() => useCurrentUser(), { wrapper });

    expect(result.current.isLoading).toBe(true);
    await waitFor(() => expect(result.current.isLoading).toBe(false));
    expect(result.current.data).toEqual(user);
    expect(getCurrentUser).toHaveBeenCalled();
  });

  it('surfaces fetch errors without retrying', async () => {
    const error = new Error('Not authenticated');
    vi.mocked(getCurrentUser).mockRejectedValue(error);

    const { result } = renderHook(() => useCurrentUser(), { wrapper });

    await waitFor(() => expect(result.current.isError).toBe(true));
    expect(result.current.error).toEqual(error);
  });
});

describe('useIsAdmin', () => {
  beforeEach(() => {
    vi.clearAllMocks();
  });

  it('fetches admin status', async () => {
    vi.mocked(checkAdmin).mockResolvedValue(true);

    const { result } = renderHook(() => useIsAdmin(), { wrapper });

    await waitFor(() => expect(result.current.isSuccess).toBe(true));
    expect(result.current.data).toBe(true);
  });
});

describe('useUserSubscription', () => {
  beforeEach(() => {
    vi.clearAllMocks();
  });

  it('fetches subscription for a user id', async () => {
    const subscription: UserSubscription = {
      id: 1,
      stripe_subscription_id: 'sub_123',
      stripe_subscription_status: 'active',
      stripe_customer_id: 'cus_123',
      stripe_current_plan: 'pro',
      stripe_subscription_frequency: 'monthly',
      isActive: true,
    };
    vi.mocked(getUserSubscription).mockResolvedValue(subscription);

    const { result } = renderHook(() => useUserSubscription(1), { wrapper });

    await waitFor(() => expect(result.current.isSuccess).toBe(true));
    expect(getUserSubscription).toHaveBeenCalledWith(1);
    expect(result.current.data).toEqual(subscription);
  });

  it('does not fetch without a user id', async () => {
    const { result } = renderHook(() => useUserSubscription(0), { wrapper });

    await waitFor(() => expect(result.current.isFetching).toBe(false));
    expect(getUserSubscription).not.toHaveBeenCalled();
  });
});

describe('useUpdateUser', () => {
  beforeEach(() => {
    vi.clearAllMocks();
  });

  it('updates the user and writes the result into the current-user cache', async () => {
    const updated = makeUser({ username: 'new-name' });
    vi.mocked(updateUser).mockResolvedValue(updated);

    const queryClient = createTestQueryClient();
    const { result } = renderHook(() => useUpdateUser(), {
      wrapper: ({ children }) => (
        <QueryClientProvider client={queryClient}>
          {children}
        </QueryClientProvider>
      ),
    });

    result.current.mutate(updated);

    await waitFor(() => expect(result.current.isSuccess).toBe(true));
    expect(updateUser).toHaveBeenCalledWith(updated);
    expect(queryClient.getQueryData(['auth', 'current'])).toEqual(updated);
  });

  it('handles update errors', async () => {
    const error = new Error('Update failed');
    vi.mocked(updateUser).mockRejectedValue(error);

    const { result } = renderHook(() => useUpdateUser(), { wrapper });

    result.current.mutate(makeUser());

    await waitFor(() => expect(result.current.isError).toBe(true));
    expect(result.current.error).toEqual(error);
  });
});
