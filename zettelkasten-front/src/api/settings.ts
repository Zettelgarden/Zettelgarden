import { apiClient, getData } from './client';

/**
 * Runtime admin settings exposed publicly by GET /api/settings (config.yaml
 * on the server). Values are hot-reloaded server-side, so admin UI edits
 * apply without a restart. All keys here are non-secret by construction;
 * admin_email is deliberately not exposed.
 */
export interface AppSettings {
  siteName: string;
  signupsEnabled: boolean;
  oidcAutoProvision: boolean;
  mailEnabled: boolean;
  emailAutoValidate: boolean;
  supportEmail: string;
}

/** Raw wire shape: the backend returns the settings map as strings. */
interface RawSettings {
  site_name?: string;
  signups_enabled?: string;
  oidc_auto_provision?: string;
  mail_enabled?: string;
  email_auto_validate?: string;
  support_email?: string;
}

function parseSettings(raw: RawSettings): AppSettings {
  return {
    siteName: raw.site_name ?? 'Zettelgarden',
    signupsEnabled: raw.signups_enabled !== 'false',
    oidcAutoProvision: raw.oidc_auto_provision !== 'false',
    mailEnabled: raw.mail_enabled !== 'false',
    emailAutoValidate: raw.email_auto_validate !== 'false',
    supportEmail: raw.support_email ?? '',
  };
}

const DEFAULT_SETTINGS: AppSettings = {
  siteName: 'Zettelgarden',
  signupsEnabled: true,
  oidcAutoProvision: true,
  mailEnabled: true,
  emailAutoValidate: true,
  supportEmail: '',
};

let settingsPromise: Promise<AppSettings> | null = null;

/**
 * Fetches the public runtime settings once and caches them for the page
 * lifetime (same pattern as getBillingStatus). On failure we fall back to
 * defaults — signups/mail shown as enabled — rather than locking anyone out.
 */
export function getSettings(): Promise<AppSettings> {
  if (!settingsPromise) {
    settingsPromise = getData(apiClient.get<RawSettings>('/settings'))
      .then(parseSettings)
      .catch(() => DEFAULT_SETTINGS);
  }
  return settingsPromise;
}
