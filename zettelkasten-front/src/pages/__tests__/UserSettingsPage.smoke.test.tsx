/**
 * Smoke test: UserSettingsPage renders the profile tab without crashing (a5q.6).
 */

import { describe, it, expect, vi, beforeEach } from 'vitest';
import { render, screen, waitFor } from '@testing-library/react';
import { renderWithProviders } from '../../tests/utils';
import { UserSettingsPage } from '../UserSettings';

vi.mock('../../api/billing', () => ({
  getBillingPortalUrl: vi.fn(),
  getBillingStatus: vi.fn(),
}));

vi.mock('../../api/account', () => ({
  exportUserData: vi.fn(),
  deleteAccount: vi.fn(),
}));

vi.mock('../../api/auth', () => ({
  requestPasswordReset: vi.fn(),
}));

const { getBillingStatus, getBillingPortalUrl } = await import(
  '../../api/billing'
);

describe('UserSettingsPage smoke', () => {
  beforeEach(() => {
    vi.clearAllMocks();
  });

  it('renders the profile settings form and tab navigation', async () => {
    vi.mocked(getBillingStatus).mockResolvedValue({ enabled: true });
    vi.mocked(getBillingPortalUrl).mockResolvedValue({
      url: 'https://billing.example.com/portal',
    });

    renderWithProviders(<UserSettingsPage />);

    await waitFor(() =>
      expect(screen.getByText('Profile Settings')).toBeInTheDocument(),
    );

    expect(screen.getByText('Save Changes')).toBeInTheDocument();
    expect(screen.getByText('Password Settings')).toBeInTheDocument();
    for (const tab of ['Profile', 'Templates', 'Tags', 'Task Statuses']) {
      expect(screen.getByText(tab)).toBeInTheDocument();
    }
  });

  it('does not crash when billing is disabled', async () => {
    vi.mocked(getBillingStatus).mockResolvedValue({ enabled: false });

    renderWithProviders(<UserSettingsPage />);

    await waitFor(() =>
      expect(screen.getByText('Profile Settings')).toBeInTheDocument(),
    );
    expect(getBillingPortalUrl).not.toHaveBeenCalled();
  });
});
