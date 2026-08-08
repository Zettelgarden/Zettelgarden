import { apiClient, getData } from "./client";

/**
 * Admin-managed settings backed by config.yaml on the server (admin-only
 * endpoints). Includes admin_email, which the public /api/settings endpoint
 * deliberately hides. Values are hot-reloaded server-side: saves apply
 * immediately without a restart.
 */
export interface AdminSettings {
  admin_email: string;
  site_name: string;
  signups_enabled: string;
  mail_enabled: string;
  email_auto_validate: string;
  support_email: string;
}

export function getAdminSettings(): Promise<AdminSettings> {
  return getData(apiClient.get<AdminSettings>("/admin/settings"));
}

/**
 * Partial update — only the keys present in `updates` are changed. The
 * response is the full new settings map.
 */
export function updateAdminSettings(
  updates: Partial<AdminSettings>,
): Promise<AdminSettings> {
  return getData(apiClient.put<AdminSettings>("/admin/settings", updates));
}
