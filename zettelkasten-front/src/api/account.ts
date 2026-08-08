import { apiClient, getData, buildURL } from './client';

const BASE_URL = import.meta.env.VITE_URL;

export interface DeleteAccountResponse {
  message: string;
  user_id: number;
}

/**
 * Downloads the current user's full data export (zip) and triggers the
 * browser download, mirroring the file-download pattern in api/files.ts.
 */
export async function exportUserData(): Promise<void> {
  const token = localStorage.getItem('token');
  const url = buildURL(BASE_URL, '/user/export');

  const response = await fetch(url, {
    method: 'GET',
    headers: {
      ...(token ? { Authorization: `Bearer ${token}` } : {}),
    },
  });
  if (!response.ok) {
    throw new Error(`Export failed (${response.status})`);
  }
  const blob = await response.blob();
  const localUrl = window.URL.createObjectURL(blob);

  // Prefer the server-provided filename (zettelgarden-export-<id>-<date>.zip),
  // fall back to a generic name.
  let filename = 'zettelgarden-export.zip';
  const disposition = response.headers.get('Content-Disposition');
  const match = disposition && disposition.match(/filename="?([^";]+)"?/);
  if (match) filename = match[1];

  const a = document.createElement('a');
  a.href = localUrl;
  a.download = filename;
  document.body.appendChild(a);
  a.click();
  window.URL.revokeObjectURL(localUrl);
  a.remove();
}

/**
 * Self-serve account deletion. Requires the account password for local users;
 * the server rejects it if billing/data checks fail.
 */
export function deleteAccount(
  password: string,
): Promise<DeleteAccountResponse> {
  return getData(
    apiClient.delete<DeleteAccountResponse>('/user', {
      body: JSON.stringify({ password }),
    }),
  );
}

/**
 * Admin-only: delete any user's account (cascades their data).
 */
export function adminDeleteUser(id: number): Promise<DeleteAccountResponse> {
  return getData(apiClient.delete<DeleteAccountResponse>(`/users/${id}`));
}
