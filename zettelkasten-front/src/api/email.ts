import { apiClient, getData } from "./client";

// Types
export interface EmailAccount {
  id: number;
  user_id: number;
  email_address: string;
  imap_server?: string;
  imap_server_type?: string;
  is_active: boolean;
  last_sync_at?: string;
  sync_status: string;
  imap_uid?: number;
  imap_uid_validity?: number;
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
  is_read: boolean;
  card_id?: number; // Link to card if email was converted
  created_at: string;
  updated_at: string;
}

export interface CreateEmailAccountParams {
  email_address: string;
  app_password: string;
}

export interface EmailListFilters {
  status?: string;
  folder?: string;
  is_read?: boolean;
  from_address?: string;
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
  if (filters?.is_read !== undefined) params.set("is_read", filters.is_read.toString());
  if (filters?.from_address) params.set("from_address", filters.from_address);
  if (filters?.limit) params.set("limit", filters.limit.toString());
  if (filters?.offset) params.set("offset", filters.offset.toString());

  const query = params.toString();
  return getData(apiClient.get<EmailListResponse>(`/emails${query ? `?${query}` : ""}`));
}

export function getEmail(id: number): Promise<Email> {
  return getData(apiClient.get<Email>(`/emails/${id}`));
}

export function updateEmailStatus(id: number, status: string): Promise<Email> {
  return getData(apiClient.patch<Email>(`/emails/${id}/status`, { status }));
}

export function getEmailStats(): Promise<Record<string, number>> {
  return getData(apiClient.get<Record<string, number>>("/emails/stats"));
}

export interface SenderInfo {
  from_address: string;
  from_name?: string;
  count: number;
}

export interface TopSendersResponse {
  senders: SenderInfo[];
}

export function getTopSenders(status?: string, limit?: number): Promise<TopSendersResponse> {
  const params = new URLSearchParams();
  if (status) params.set("status", status);
  if (limit) params.set("limit", limit.toString());

  const query = params.toString();
  return getData(apiClient.get<TopSendersResponse>(`/emails/top-senders${query ? `?${query}` : ""}`));
}

// Email Conversion API
export interface ConvertEmailParams {
  title?: string;
  body?: string;
  tags?: string;
  card_id?: string;
}

export interface ConvertCardResponse {
  id: number;
}

export function convertEmailToCard(id: number, params?: ConvertEmailParams): Promise<ConvertCardResponse> {
  return getData(apiClient.post<ConvertCardResponse>(`/emails/${id}/convert`, params));
}

// Batch operation types
export interface BatchEmailParams {
  email_ids: number[];
}

export interface BatchArchiveParams extends BatchEmailParams {
  status?: "archived" | "unprocessed";
}

export interface BatchConvertParams extends BatchEmailParams {
  title?: string;
  body?: string;
  tags?: string;
}

export interface BatchOperationResponse {
  success: boolean;
  total: number;
  success_count: number;
  fail_count: number;
  results: Array<{
    email_id: number;
    success: boolean;
    card_id?: string;
    task_id?: number;
    error?: string;
  }>;
}

export interface BatchArchiveResponse {
  success: boolean;
  count: number;
  emails: Email[];
}

// Batch operation API functions
export function batchArchiveEmails(params: BatchArchiveParams): Promise<BatchArchiveResponse> {
  return getData(apiClient.post<BatchArchiveResponse>("/emails/batch-archive", params));
}

export function batchConvertEmails(params: BatchConvertParams): Promise<BatchOperationResponse> {
  return getData(apiClient.post<BatchOperationResponse>("/emails/batch-convert", params));
}

export function batchCreateTasks(params: BatchEmailParams): Promise<BatchOperationResponse> {
  return getData(apiClient.post<BatchOperationResponse>("/emails/batch-create-tasks", params));
}

// Email Search types
export interface EmailSearchParams {
  search_term: string;
  page?: number;
  per_page?: number;
}

export interface EmailSearchResult {
  id: number;
  subject: string;
  sender: string;
  preview: string;
  metadata?: Record<string, unknown>;
}

export interface EmailSearchResponse {
  results: EmailSearchResult[];
  page: number;
  per_page: number;
  total: number;
  total_pages: number;
}

// Email Search API
export function searchEmails(params: EmailSearchParams): Promise<EmailSearchResponse> {
  return getData(apiClient.post<EmailSearchResponse>("/emails/search", params));
}

// Thread types
export interface EmailThread {
  thread_id: string;
  subject: string;
  participant_count: number;
  message_count: number;
  unread_count: number;
  oldest_date?: string;
  newest_date?: string;
  messages?: Email[];
}

export interface EmailThreadListFilters {
  status?: string;
  folder?: string;
  limit?: number;
  offset?: number;
}

export interface EmailThreadListResponse {
  threads: EmailThread[];
  total: number;
}

// Thread API functions
export function getEmailThread(threadId: string): Promise<EmailThread> {
  return getData(apiClient.get<EmailThread>(`/emails/threads/${threadId}`));
}

export function markThreadAsRead(threadId: string): Promise<{ success: boolean }> {
  return getData(apiClient.patch<{ success: boolean }>(`/emails/threads/${threadId}/read`, {}));
}

export function archiveThread(threadId: string): Promise<{ success: boolean }> {
  return getData(apiClient.patch<{ success: boolean }>(`/emails/threads/${threadId}/archive`, {}));
}

// Email Fact Extraction types (PRO feature)
export interface ExtractFactsResponse {
  email_id: number;
  facts: string[];
  count: number;
}

export interface SaveFactsRequest {
  facts: string[];
}

export interface SaveFactsResponse {
  success: boolean;
  email_id: number;
  saved_count: number;
  facts: Array<{
    id: number;
    user_id: number;
    card_pk: number;
    fact: string;
  }>;
}

export interface EmailFactsResponse {
  email_id: number;
  facts: Array<{
    id: number;
    user_id: number;
    card_pk: number;
    fact: string;
    created_at: string;
    updated_at: string;
  }>;
  count: number;
}

// Email Fact Extraction API functions (PRO feature)
export function extractFactsFromEmail(id: number): Promise<ExtractFactsResponse> {
  return getData(apiClient.post<ExtractFactsResponse>(`/emails/${id}/extract-facts`, {}));
}

export function saveFactsFromEmail(id: number, facts: string[]): Promise<SaveFactsResponse> {
  return getData(apiClient.post<SaveFactsResponse>(`/emails/${id}/save-facts`, { facts }));
}

export function getEmailFacts(id: number): Promise<EmailFactsResponse> {
  return getData(apiClient.get<EmailFactsResponse>(`/emails/${id}/facts`));
}

// Email Attachment types
export interface EmailAttachment {
  id: number;
  user_id: number;
  email_id: number;
  file_id?: number;
  filename: string;
  content_type?: string;
  size?: number;
  s3_key?: string;
  thumbnail_path?: string;
  content_id?: string;
  is_inline: boolean;
  created_at: string;
  updated_at: string;
}

export interface EmailAttachmentWithDownloadURL extends EmailAttachment {
  download_url: string;
  thumbnail_url?: string;
  is_image: boolean;
  is_saved_to_vault: boolean;
}

export interface EmailAttachmentsResponse {
  attachments: EmailAttachmentWithDownloadURL[];
  count: number;
}

export interface SaveAttachmentToVaultParams {
  card_pk?: number;
}

// Email Attachment API functions
export function getEmailAttachments(id: number): Promise<EmailAttachmentsResponse> {
  return getData(apiClient.get<EmailAttachmentsResponse>(`/emails/${id}/attachments`));
}

export function downloadEmailAttachment(id: number): string {
  return `/api/emails/attachments/${id}/download`;
}

export function getAttachmentThumbnail(id: number): string {
  return `/api/emails/attachments/${id}/thumbnail`;
}

export function saveAttachmentToVault(id: number, params: SaveAttachmentToVaultParams): Promise<EmailAttachment> {
  return getData(apiClient.post<EmailAttachment>(`/emails/attachments/${id}/save-to-vault`, params));
}

export function deleteEmailAttachment(id: number): Promise<void> {
  return getData(apiClient.delete<void>(`/emails/attachments/${id}`));
}
