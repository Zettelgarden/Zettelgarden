/**
 * Admin panel entry on the profile settings page (6er.16 follow-up):
 * admins see an "Open Admin Panel" button, regular users do not.
 */

import { describe, it, expect, vi, beforeEach } from 'vitest';
import { render, screen, waitFor } from '@testing-library/react';
import { MemoryRouter } from 'react-router-dom';
import { UserSettingsPage } from '../UserSettings';
import { useAuth } from '../../contexts/AuthContext';

vi.mock('../../contexts/AuthContext', () => ({
  useAuth: vi.fn(),
}));

vi.mock('../../api/billing', () => ({
  getBillingPortalUrl: vi.fn().mockResolvedValue({ url: '' }),
  getBillingStatus: vi.fn().mockResolvedValue({ enabled: false }),
}));

vi.mock('../../api/account', () => ({
  exportUserData: vi.fn(),
  deleteAccount: vi.fn(),
}));

vi.mock('../../api/auth', () => ({
  requestPasswordReset: vi.fn(),
}));

const mockUseAuth = vi.mocked(useAuth);

function baseUser(overrides = {}) {
  return {
    id: 1,
    username: 'admin',
    email: 'admin@example.com',
    is_admin: false,
    ...overrides,
  };
}

function renderPage(user: any) {
  mockUseAuth.mockReturnValue({
    user,
    hasSubscription: true,
    updateUser: vi.fn(),
    logoutUser: vi.fn(),
  } as any);
  return render(
    <MemoryRouter>
      <UserSettingsPage />
    </MemoryRouter>,
  );
}

describe('UserSettingsPage admin entry (profile tab)', () => {
  beforeEach(() => {
    mockUseAuth.mockReset();
  });

  it('hides the Administration panel for regular users', async () => {
    renderPage(baseUser());
    await waitFor(() =>
      expect(screen.queryByText('Profile Settings')).not.toBeNull(),
    );
    expect(screen.queryByText('Administration')).not.toBeInTheDocument();
    expect(
      screen.queryByRole('button', { name: 'Open Admin Panel' }),
    ).not.toBeInTheDocument();
  });

  it('shows the Administration panel with an /admin link for admins', async () => {
    renderPage(baseUser({ is_admin: true }));
    await waitFor(() =>
      expect(screen.getByText('Administration')).toBeInTheDocument(),
    );
    expect(
      screen.getByRole('button', { name: 'Open Admin Panel' }),
    ).toBeInTheDocument();
  });
});
