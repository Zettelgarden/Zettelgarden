import React, { useState, useEffect, FormEvent } from 'react';
import {
  getAdminSettings,
  updateAdminSettings,
  AdminSettings,
} from '../../api/adminSettings';
import { setDocumentTitle } from '../../utils/title';

interface SettingsFormState {
  admin_email: string;
  site_name: string;
  support_email: string;
  signups_enabled: boolean;
  oidc_auto_provision: boolean;
  mail_enabled: boolean;
  email_auto_validate: boolean;
  job_retention_days: string;
  rss_article_retention_days: string;
}

const EMPTY_FORM: SettingsFormState = {
  admin_email: '',
  site_name: 'Zettelgarden',
  support_email: '',
  signups_enabled: true,
  oidc_auto_provision: true,
  mail_enabled: true,
  email_auto_validate: true,
  job_retention_days: '30',
  rss_article_retention_days: '30',
};

export function AdminSettingsPage() {
  const [form, setForm] = useState<SettingsFormState>(EMPTY_FORM);
  const [isLoading, setIsLoading] = useState(true);
  const [error, setError] = useState<string | null>(null);
  const [success, setSuccess] = useState<string | null>(null);

  useEffect(() => {
    setDocumentTitle('Admin Settings');
    getAdminSettings()
      .then((s: AdminSettings) =>
        setForm({
          admin_email: s.admin_email,
          site_name: s.site_name,
          support_email: s.support_email,
          signups_enabled: s.signups_enabled !== 'false',
          oidc_auto_provision: s.oidc_auto_provision !== 'false',
          mail_enabled: s.mail_enabled !== 'false',
          email_auto_validate: s.email_auto_validate !== 'false',
          job_retention_days: s.job_retention_days || '30',
          rss_article_retention_days: s.rss_article_retention_days || '30',
        }),
      )
      .catch((e) => setError(e.message))
      .finally(() => setIsLoading(false));
  }, []);

  const handleText =
    (name: keyof SettingsFormState) =>
    (e: React.ChangeEvent<HTMLInputElement>) =>
      setForm({ ...form, [name]: e.target.value });

  const handleBool =
    (name: keyof SettingsFormState) =>
    (e: React.ChangeEvent<HTMLInputElement>) =>
      setForm({ ...form, [name]: e.target.checked });

  async function handleSubmit(event: FormEvent) {
    event.preventDefault();
    setError(null);
    setSuccess(null);
    try {
      await updateAdminSettings({
        admin_email: form.admin_email,
        site_name: form.site_name,
        support_email: form.support_email,
        signups_enabled: String(form.signups_enabled),
        oidc_auto_provision: String(form.oidc_auto_provision),
        mail_enabled: String(form.mail_enabled),
        email_auto_validate: String(form.email_auto_validate),
        job_retention_days: form.job_retention_days,
        rss_article_retention_days: form.rss_article_retention_days,
      });
      setSuccess('Settings saved and applied immediately.');
    } catch (e: any) {
      setError(e.message);
    }
  }

  if (isLoading) {
    return (
      <div className="animate-pulse bg-gray-100 rounded-lg p-8">
        Loading settings...
      </div>
    );
  }

  const textField = (
    name: keyof SettingsFormState,
    label: string,
    hint: string,
  ) => (
    <div>
      <label
        htmlFor={name}
        className="block text-sm font-medium text-gray-700 mb-1"
      >
        {label}
      </label>
      <input
        id={name}
        name={name}
        type="text"
        value={String(form[name])}
        onChange={handleText(name)}
        className="w-full px-3 py-2 border border-gray-300 rounded-md focus:ring-2 focus:ring-indigo-500"
      />
      <p className="text-xs text-gray-500 mt-1">{hint}</p>
    </div>
  );

  const numberField = (
    name: 'job_retention_days' | 'rss_article_retention_days',
    label: string,
    hint: string,
  ) => (
    <div>
      <label
        htmlFor={name}
        className="block text-sm font-medium text-gray-700 mb-1"
      >
        {label}
      </label>
      <input
        id={name}
        name={name}
        type="number"
        min={1}
        value={String(form[name])}
        onChange={handleText(name)}
        className="w-full px-3 py-2 border border-gray-300 rounded-md focus:ring-2 focus:ring-indigo-500"
      />
      <p className="text-xs text-gray-500 mt-1">{hint}</p>
    </div>
  );

  const boolField = (
    name: keyof SettingsFormState,
    label: string,
    hint: string,
  ) => (
    <label
      htmlFor={name}
      className="flex items-start gap-3 py-2 cursor-pointer"
    >
      <input
        id={name}
        name={name}
        type="checkbox"
        checked={Boolean(form[name])}
        onChange={handleBool(name)}
        className="mt-1 h-4 w-4 text-indigo-600 border-gray-300 rounded"
      />
      <span>
        <span className="block text-sm font-medium text-gray-700">{label}</span>
        <span className="block text-xs text-gray-500">{hint}</span>
      </span>
    </label>
  );

  return (
    <div className="max-w-2xl">
      <h1 className="text-2xl font-semibold mb-1">Admin Settings</h1>
      <p className="text-sm text-gray-600 mb-6">
        Stored in <code>config.yaml</code> next to the database; saves apply
        immediately (no restart).
      </p>

      {error && (
        <div className="mb-4 border border-red-200 bg-red-50 text-red-700 text-sm rounded-lg p-3">
          {error}
        </div>
      )}
      {success && (
        <div className="mb-4 border border-green-200 bg-green-50 text-green-700 text-sm rounded-lg p-3">
          {success}
        </div>
      )}

      <form onSubmit={handleSubmit} className="space-y-5">
        {textField(
          'site_name',
          'Site name',
          'Shown in the browser tab and email templates.',
        )}
        {textField(
          'admin_email',
          'Admin email',
          'Notification recipient (e.g. new subscriptions) and the email that grants admin on registration.',
        )}
        {textField(
          'support_email',
          'Support email',
          'Shown to users in settings for account/export help. Leave empty to hide.',
        )}

        <div className="border-t border-gray-200 pt-4 space-y-2">
          <h2 className="text-sm font-semibold text-gray-800">
            Registration &amp; SSO
          </h2>
          {boolField(
            'signups_enabled',
            'Allow new user registration',
            'When off, the register link is hidden and public signup is rejected.',
          )}
          {boolField(
            'oidc_auto_provision',
            'Auto-provision accounts via OIDC/SSO',
            'When off, OIDC only logs in accounts that already exist (same email or linked); unknown IdP users get an error.',
          )}
        </div>

        <div className="border-t border-gray-200 pt-4 space-y-2">
          <h2 className="text-sm font-semibold text-gray-800">Email</h2>
          {boolField(
            'mail_enabled',
            'Send transactional email',
            'When off, no SMTP mail is sent and the email-validation banner is hidden. Applies immediately — no restart needed.',
          )}
          {boolField(
            'email_auto_validate',
            'Auto-validate new accounts',
            'Treat new accounts as email-validated without a confirmation email.',
          )}
        </div>

        <div className="border-t border-gray-200 pt-4 space-y-3">
          <h2 className="text-sm font-semibold text-gray-800">
            Data retention
          </h2>
          {numberField(
            'job_retention_days',
            'LLM job history (days)',
            'Completed/failed background jobs (summaries, imports) are purged after this many days. Applied on the next daily cleanup.',
          )}
          {numberField(
            'rss_article_retention_days',
            'RSS article retention (days)',
            'Unstarred RSS articles not converted to cards are deleted after this many days. Applied on the next daily cleanup.',
          )}
        </div>

        <div className="border-t border-gray-200 pt-4">
          <button
            type="submit"
            className="px-4 py-2 bg-indigo-600 text-white rounded-md hover:bg-indigo-700"
          >
            Save Settings
          </button>
        </div>
      </form>

      <p className="text-xs text-gray-500 mt-6">
        Secrets and boot-time values (SECRET_KEY, SMTP_PASSWORD, Stripe /
        Typesense / LLM keys, port, URL, storage dir) are intentionally not
        manageable here — they stay in the environment config.
      </p>
    </div>
  );
}
