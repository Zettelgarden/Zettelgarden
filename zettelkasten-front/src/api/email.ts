import { apiClient, getData } from "./client";

// Types
export interface EmailAccount {
  id: number;
  user_id: number;
  email_address: string;
  jmap_server_url: string;
  is_active: boolean;
  last_sync_at?: string;
  sync_status: string;
  jmap_state?: string;
  created_at: string;
  updated_at: string;
}

export interface Email {
  id: number;
  user_id: number;
  email_account_id?: number;
  message_id: string;
  thread_id?: string;
  subject?: string;
  from_address?: string;
  from_name?: string;
  to_addresses?: string;
  body_text?: string;
  body_html?: string;
  received_at?: string;
  folder?: string;
  status: string;
  created_at: string;
  updated_at: string;
}

export interface CreateEmailAccountParams {
  email_address: string;
  api_token: string;
}

export interface EmailListFilters {
  status?: string;
  folder?: string;
  limit?: number;
  offset?: number;
}

export interface EmailListResponse {
  emails: Email[];
  total: number;
  limit: number;
  offset: number;
}

export interface SyncEmailAccountResponse {
  message: string;
  account_id: number;
  emails_fetched?: number;
  emails_stored?: number;
}

// Email Account API
export function listEmailAccounts(): Promise<EmailAccount[]> {
  return getData(apiClient.get<EmailAccount[]>("/email/accounts"));
}

export function createEmailAccount(params: CreateEmailAccountParams): Promise<EmailAccount> {
  return getData(apiClient.post<EmailAccount>("/email/accounts", params));
}

export function getEmailAccount(id: number): Promise<EmailAccount> {
  return getData(apiClient.get<EmailAccount>(`/email/accounts/${id}`));
}

export function deleteEmailAccount(id: number): Promise<void> {
  return getData(apiClient.delete<void>(`/email/accounts/${id}`));
}

export function syncEmailAccount(id: number): Promise<SyncEmailAccountResponse> {
  return getData(apiClient.post<SyncEmailAccountResponse>(`/email/accounts/${id}/sync`, {}));
}

// Email API
export function listEmails(filters?: EmailListFilters): Promise<EmailListResponse> {
  const params = new URLSearchParams();
  if (filters?.status) params.set("status", filters.status);
  if (filters?.folder) params.set("folder", filters.folder);
  if (filters?.limit) params.set("limit", filters.limit.toString());
  if (filters?.offset) params.set("offset", filters.offset.toString());

  const query = params.toString();
  return getData(apiClient.get<EmailListResponse>(`/emails${query ? `?${query}` : ""}`));
}

export function getEmail(id: number): Promise<Email> {
  return getData(apiClient.get<Email>(`/emails/${id}`));
}

export function getEmailStats(): Promise<Record<string, number>> {
  return getData(apiClient.get<Record<string, number>>("/emails/stats"));
}
