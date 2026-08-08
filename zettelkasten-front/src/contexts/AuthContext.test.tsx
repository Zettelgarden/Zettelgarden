/**
 * Tests for AuthContext (src/contexts/AuthContext.tsx)
 *
 * Covers the auth lifecycle: token storage/clearing, login, logout,
 * OAuth token login, subscription status, and admin state.
 *
 * Note: assertions read values rendered into the DOM by a probe component
 * (rather than capturing hook return values in a JS closure, which proved
 * unreliable with async effects in this test environment).
 */

import { describe, it, expect, beforeEach, vi } from 'vitest';
import { render, screen, act, waitFor } from '@testing-library/react';
import { AuthProvider, useAuth } from './AuthContext';
import {
  checkAdmin,
  updateUser as apiUpdateUser,
  getUserSubscription,
  getCurrentUser,
} from '../api/users';
import { getBillingStatus } from '../api/billing';
import { User } from '../models/User';
import { LoginResponse } from '../models/Auth';

vi.mock('../api/users', () => ({
  checkAdmin: vi.fn(),
  updateUser: vi.fn(),
  getUserSubscription: vi.fn(),
  getCurrentUser: vi.fn(),
}));

vi.mock('../api/billing', () => ({
  getBillingStatus: vi.fn(),
}));

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

const subscription = {
  id: 1,
  user_id: 1,
  stripe_subscription_id: 'sub_123',
  stripe_subscription_status: 'active',
  stripe_customer_id: 'cus_123',
  current_plan: 'pro',
  frequency: 'monthly',
};

beforeEach(() => {
  localStorage.clear();
  vi.clearAllMocks();
});

interface ProbeState {
  isAuthenticated: boolean;
  isAdmin: boolean;
  hasSubscription: boolean;
  isLoading: boolean;
  username: string;
}

/**
 * Renders the provider with a probe that serializes the auth state into
 * the DOM, and exposes the context's action functions for tests.
 */
function renderAuth() {
  let actions: ReturnType<typeof useAuth>;

  function Probe() {
    const auth = useAuth();
    actions = auth;
    const state: ProbeState = {
      isAuthenticated: auth.isAuthenticated,
      isAdmin: auth.isAdmin,
      hasSubscription: auth.hasSubscription,
      isLoading: auth.isLoading,
      username: auth.currentUser?.username ?? '',
    };
    return <div data-testid="probe">{JSON.stringify(state)}</div>;
  }

  render(
    <AuthProvider>
      <Probe />
    </AuthProvider>,
  );

  function readState(): ProbeState {
    return JSON.parse(screen.getByTestId('probe').textContent as string);
  }

  async function waitForState(predicate: (s: ProbeState) => boolean) {
    await waitFor(() => expect(predicate(readState())).toBe(true));
    return readState();
  }

  return {
    readState,
    waitForState,
    get actions() {
      return actions!;
    },
  };
}

describe('AuthContext', () => {
  it('initializes as unauthenticated when no token is stored', async () => {
    const auth = renderAuth();

    const state = await auth.waitForState((s) => !s.isLoading);
    expect(state.isAuthenticated).toBe(false);
    expect(getCurrentUser).not.toHaveBeenCalled();
  });

  it('restores the session from a stored token on mount', async () => {
    localStorage.setItem('token', 'stored-token');
    const user = makeUser();
    vi.mocked(checkAdmin).mockResolvedValue(true);
    vi.mocked(getCurrentUser).mockResolvedValue(user);
    vi.mocked(getUserSubscription).mockResolvedValue(subscription as any);
    vi.mocked(getBillingStatus).mockResolvedValue({ enabled: true });

    const auth = renderAuth();

    const state = await auth.waitForState(
      (s) => s.isAuthenticated && s.isAdmin && s.hasSubscription && s.username === 'tester' && !s.isLoading
    );
    expect(checkAdmin).toHaveBeenCalled();
    expect(getCurrentUser).toHaveBeenCalled();
  });

  it('treats billing-disabled instances as fully subscribed', async () => {
    localStorage.setItem('token', 'stored-token');
    vi.mocked(checkAdmin).mockResolvedValue(false);
    vi.mocked(getCurrentUser).mockResolvedValue(makeUser());
    vi.mocked(getUserSubscription).mockResolvedValue(null as any);
    vi.mocked(getBillingStatus).mockResolvedValue({ enabled: false });

    const auth = renderAuth();

    const state = await auth.waitForState((s) => s.isAuthenticated && s.hasSubscription && !s.isLoading);
    expect(state.hasSubscription).toBe(true);
  });

  it('logs out when session restoration fails', async () => {
    localStorage.setItem('token', 'bad-token');
    const consoleSpy = vi.spyOn(console, 'error').mockImplementation(() => {});
    vi.mocked(checkAdmin).mockRejectedValue(new Error('invalid token'));

    const auth = renderAuth();

    const state = await auth.waitForState(
      (s) => !s.isAuthenticated && !s.isLoading,
    );
    expect(localStorage.getItem('token')).toBeNull();
    expect(consoleSpy).toHaveBeenCalled();

    consoleSpy.mockRestore();
  });

  it('loginUser stores the token and marks the user authenticated', async () => {
    const user = makeUser({ stripe_subscription_status: 'trialing' });
    vi.mocked(getBillingStatus).mockResolvedValue({ enabled: true });

    const auth = renderAuth();
    await auth.waitForState((s) => !s.isLoading);

    const loginResponse: LoginResponse = {
      access_token: 'fresh-token',
      user,
      message: 'ok',
    };

    await act(async () => {
      await auth.actions.loginUser(loginResponse);
    });

    const state = auth.readState();
    expect(localStorage.getItem('token')).toBe('fresh-token');
    expect(localStorage.getItem('username')).toBe('tester');
    expect(state.isAuthenticated).toBe(true);
    expect(state.hasSubscription).toBe(true); // trialing counts as subscribed
  });

  it('loginUserFromToken restores user data after OAuth', async () => {
    const user = makeUser({ username: 'oauth-user' });
    vi.mocked(checkAdmin).mockResolvedValue(false);
    vi.mocked(getCurrentUser).mockResolvedValue(user);
    vi.mocked(getUserSubscription).mockResolvedValue(subscription as any);
    vi.mocked(getBillingStatus).mockResolvedValue({ enabled: true });

    const auth = renderAuth();
    await auth.waitForState((s) => !s.isLoading);

    await act(async () => {
      await auth.actions.loginUserFromToken('oauth-token');
    });

    const state = auth.readState();
    expect(localStorage.getItem('token')).toBe('oauth-token');
    expect(localStorage.getItem('username')).toBe('oauth-user');
    expect(state.isAuthenticated).toBe(true);
    expect(state.username).toBe('oauth-user');
    expect(state.hasSubscription).toBe(true);
  });

  it('logoutUser clears the token and resets auth state', async () => {
    localStorage.setItem('token', 'stored-token');
    vi.mocked(checkAdmin).mockResolvedValue(true);
    vi.mocked(getCurrentUser).mockResolvedValue(makeUser());
    vi.mocked(getUserSubscription).mockResolvedValue(subscription as any);
    vi.mocked(getBillingStatus).mockResolvedValue({ enabled: true });

    const auth = renderAuth();
    await auth.waitForState((s) => s.isAuthenticated && !s.isLoading);

    act(() => {
      auth.actions.logoutUser();
    });

    const state = auth.readState();
    expect(localStorage.getItem('token')).toBeNull();
    expect(state.isAuthenticated).toBe(false);
    expect(state.isAdmin).toBe(false);
  });

  it('updateUser updates both user and currentUser', async () => {
    const updated = makeUser({ username: 'renamed' });
    vi.mocked(apiUpdateUser).mockResolvedValue(updated);

    const auth = renderAuth();
    await auth.waitForState((s) => !s.isLoading);

    await act(async () => {
      await auth.actions.updateUser(updated);
    });

    const state = auth.readState();
    expect(apiUpdateUser).toHaveBeenCalledWith(updated);
    expect(state.username).toBe('renamed');
  });

  it('refreshSubscription reflects the current billing state', async () => {
    vi.mocked(getCurrentUser).mockResolvedValue(makeUser());
    vi.mocked(getUserSubscription).mockResolvedValue(subscription as any);
    vi.mocked(getBillingStatus).mockResolvedValue({ enabled: true });

    const auth = renderAuth();
    await auth.waitForState((s) => !s.isLoading);

    let result: boolean | undefined;
    await act(async () => {
      result = await auth.actions.refreshSubscription();
    });

    expect(result).toBe(true);
    expect(auth.readState().hasSubscription).toBe(true);
  });
});
