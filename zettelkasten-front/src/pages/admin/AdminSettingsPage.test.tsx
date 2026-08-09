import { describe, it, expect, vi, beforeEach } from 'vitest';
import { render, screen, fireEvent, waitFor } from '@testing-library/react';
import { AdminSettingsPage } from './AdminSettingsPage';
import * as adminSettingsApi from '../../api/adminSettings';
import { setDocumentTitle } from '../../utils/title';

vi.mock('../../api/adminSettings', () => ({
  getAdminSettings: vi.fn(),
  updateAdminSettings: vi.fn(),
}));

vi.mock('../../utils/title', () => ({
  setDocumentTitle: vi.fn(),
}));

const mockGet = vi.mocked(adminSettingsApi.getAdminSettings);
const mockUpdate = vi.mocked(adminSettingsApi.updateAdminSettings);

function mockSettings(): adminSettingsApi.AdminSettings {
  return {
    admin_email: 'admin@example.com',
    site_name: 'Zettelgarden',
    signups_enabled: 'true',
    oidc_auto_provision: 'true',
    mail_enabled: 'true',
    email_auto_validate: 'true',
    support_email: 'support@example.com',
    job_retention_days: '30',
    rss_article_retention_days: '30',
  };
}

describe('AdminSettingsPage (6er.16)', () => {
  beforeEach(() => {
    mockGet.mockReset();
    mockUpdate.mockReset();
    mockGet.mockResolvedValue(mockSettings());
    mockUpdate.mockResolvedValue(mockSettings());
  });

  it('renders all settings sections including the migrated keys', async () => {
    render(<AdminSettingsPage />);

    // Existing fields.
    expect(await screen.findByLabelText('Site name')).toBeTruthy();
    expect(screen.getByLabelText('Admin email')).toBeTruthy();
    expect(screen.getByLabelText('Support email')).toBeTruthy();

    // Migrated OIDC toggle + retention days.
    expect(
      screen.getByLabelText(/Auto-provision accounts via OIDC\/SSO/),
    ).toBeTruthy();
    expect(screen.getByLabelText('LLM job history (days)')).toBeTruthy();
    expect(screen.getByLabelText('RSS article retention (days)')).toBeTruthy();
  });

  it('loads the OIDC toggle and retention values into the form', async () => {
    mockGet.mockResolvedValue({
      ...mockSettings(),
      oidc_auto_provision: 'false',
      job_retention_days: '90',
      rss_article_retention_days: '7',
    });
    render(<AdminSettingsPage />);

    const oidcToggle = (await screen.findByLabelText(
      /Auto-provision accounts via OIDC\/SSO/,
    )) as HTMLInputElement;
    expect(oidcToggle.checked).toBe(false);

    const jobDays = screen.getByLabelText(
      'LLM job history (days)',
    ) as HTMLInputElement;
    expect(jobDays.value).toBe('90');
    expect(
      (
        screen.getByLabelText(
          'RSS article retention (days)',
        ) as HTMLInputElement
      ).value,
    ).toBe('7');
  });

  it('submits the migrated keys in the partial update', async () => {
    render(<AdminSettingsPage />);
    await screen.findByLabelText('Site name');

    const jobDays = screen.getByLabelText(
      'LLM job history (days)',
    ) as HTMLInputElement;
    fireEvent.change(jobDays, { target: { value: '60' } });

    fireEvent.click(screen.getByRole('button', { name: 'Save Settings' }));

    await waitFor(() => expect(mockUpdate).toHaveBeenCalledTimes(1));
    const payload = mockUpdate.mock.calls[0][0];
    expect(payload.oidc_auto_provision).toBe('true');
    expect(payload.job_retention_days).toBe('60');
    expect(payload.rss_article_retention_days).toBe('30');

    expect(await screen.findByText(/applied immediately/i)).toBeTruthy();
    expect(setDocumentTitle).toHaveBeenCalledWith('Admin Settings');
  });
});
