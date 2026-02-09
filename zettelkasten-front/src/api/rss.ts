import { apiClient, getData } from "./client";

// Types
export interface RSSFeed {
  id: number;
  user_id: number;
  url: string;
  name: string;
  folder?: string;
  auto_tags: string;
  fetch_interval: number;
  enabled: boolean;
  last_fetched_at?: string;
  last_error?: string;
  created_at: string;
  updated_at: string;
}

export interface RSSArticle {
  id: number;
  user_id: number;
  feed_id: number;
  title: string;
  content?: string;
  author?: string;
  url: string;
  published_at?: string;
  fetched_at: string;
  read: boolean;
  card_id?: number;
}

export interface RSSFolder {
  id: number;
  user_id: number;
  name: string;
  order_index: number;
}

export interface CreateRSSFolderParams {
  name: string;
  order_index?: number;
}

export interface UpdateRSSFolderParams {
  name?: string;
  order_index?: number;
}

export interface CreateRSSFeedParams {
  url: string;
  name?: string;
  folder?: string;
  auto_tags?: string;
  fetch_interval?: number;
  enabled?: boolean;
}

export interface UpdateRSSFeedParams {
  name?: string;
  folder?: string;
  auto_tags?: string;
  fetch_interval?: number;
  enabled?: boolean;
}

export interface ConvertArticleParams {
  title?: string;
  body?: string;
  tags?: string;
  card_id?: string;
}

export interface ConvertCardResponse {
  id: number;
}

export interface ArticleFilters {
  folder?: string;
  unread?: boolean;
  feed_id?: number;
  limit?: number;
  offset?: number;
}

export interface PaginatedArticlesResponse {
  articles: RSSArticle[];
  total: number;
}

export interface UnreadCounts {
  folders: Record<string, number>;
  feeds: Record<number, number>;
}

export interface OPMLImportResult {
  created_feeds: number;
  skipped_feeds: number;
  created_folders: number;
  errors?: string[];
}

export interface DiscoverFeedRequest {
  url: string;
}

export interface DiscoverFeedResponse {
  feed_url: string;
  title: string;
}

// Feed API
export function createFeed(feed: CreateRSSFeedParams): Promise<RSSFeed> {
  return getData(apiClient.post<RSSFeed>("/rss/feeds", feed));
}

export function listFeeds(): Promise<RSSFeed[]> {
  return getData(apiClient.get<RSSFeed[]>("/rss/feeds")).then(data => data ?? []);
}

export function getFeed(id: number): Promise<RSSFeed> {
  return getData(apiClient.get<RSSFeed>(`/rss/feeds/${id}`));
}

export function updateFeed(id: number, params: UpdateRSSFeedParams): Promise<RSSFeed> {
  return getData(apiClient.put<RSSFeed>(`/rss/feeds/${id}`, params));
}

export function deleteFeed(id: number): Promise<void> {
  return getData(apiClient.delete<void>(`/rss/feeds/${id}`));
}

export function markFeedAsRead(id: number): Promise<void> {
  return getData(apiClient.post<void>(`/rss/feeds/${id}/read`, {}));
}

export function refreshFeeds(): Promise<{ fetched: number }> {
  return getData(apiClient.post<{ fetched: number }>("/rss/feeds/fetch", {}));
}

// Feed Discovery API
export function discoverFeed(url: string): Promise<DiscoverFeedResponse> {
  return getData(apiClient.post<DiscoverFeedResponse>("/rss/discover", { url }));
}

// Article API
export function listArticles(filters?: ArticleFilters): Promise<PaginatedArticlesResponse> {
  const params = new URLSearchParams();
  if (filters?.folder) params.set("folder", filters.folder);
  if (filters?.unread) params.set("unread", "true");
  if (filters?.feed_id) params.set("feed_id", filters.feed_id.toString());
  if (filters?.limit) params.set("limit", filters.limit.toString());
  if (filters?.offset) params.set("offset", filters.offset.toString());

  const query = params.toString();
  return getData(apiClient.get<PaginatedArticlesResponse>(`/rss/articles${query ? `?${query}` : ""}`));
}

export function getArticle(id: number): Promise<RSSArticle> {
  return getData(apiClient.get<RSSArticle>(`/rss/articles/${id}`));
}

export function markAsRead(id: number, read: boolean = true): Promise<void> {
  return getData(apiClient.post<void>(`/rss/articles/${id}/read`, { read }));
}

export function convertToCard(id: number, params?: ConvertArticleParams): Promise<ConvertCardResponse> {
  return getData(apiClient.post<ConvertCardResponse>(`/rss/articles/${id}/convert`, params));
}

// Folder API
export function listFolders(): Promise<RSSFolder[]> {
  return getData(apiClient.get<RSSFolder[]>("/rss/folders")).then(data => data ?? []);
}

export function getFolder(id: number): Promise<RSSFolder> {
  return getData(apiClient.get<RSSFolder>(`/rss/folders/${id}`));
}

export function createFolder(params: CreateRSSFolderParams): Promise<RSSFolder> {
  return getData(apiClient.post<RSSFolder>("/rss/folders", params));
}

export function updateFolder(id: number, params: UpdateRSSFolderParams): Promise<RSSFolder> {
  return getData(apiClient.put<RSSFolder>(`/rss/folders/${id}`, params));
}

export function deleteFolder(id: number): Promise<void> {
  return getData(apiClient.delete<void>(`/rss/folders/${id}`));
}

export function markFolderAsRead(id: number): Promise<void> {
  return getData(apiClient.post<void>(`/rss/folders/${id}/read`, {}));
}

// Unread Counts API
export function getUnreadCounts(): Promise<UnreadCounts> {
  return getData(apiClient.get<UnreadCounts>("/rss/unread-counts"));
}

// OPML Export/Import API
export async function exportOPML(): Promise<Blob> {
  const BASE_URL = import.meta.env.VITE_URL;
  const url = BASE_URL.endsWith("/")
    ? BASE_URL + "rss/opml/export"
    : BASE_URL + "/rss/opml/export";

  const token = localStorage.getItem("token");
  const response = await fetch(url, {
    method: "GET",
    headers: {
      Authorization: token ? `Bearer ${token}` : "",
    },
  });

  if (!response.ok) {
    const text = await response.text();
    throw new Error(text || `Export failed with status: ${response.status}`);
  }

  return response.blob();
}

export async function importOPML(file: File): Promise<OPMLImportResult> {
  const BASE_URL = import.meta.env.VITE_URL;
  const url = BASE_URL.endsWith("/")
    ? BASE_URL + "rss/opml/import"
    : BASE_URL + "/rss/opml/import";

  const formData = new FormData();
  formData.append("file", file);

  const token = localStorage.getItem("token");
  const response = await fetch(url, {
    method: "POST",
    headers: {
      Authorization: token ? `Bearer ${token}` : "",
    },
    body: formData,
  });

  if (!response.ok) {
    const text = await response.text();
    throw new Error(text || `Import failed with status: ${response.status}`);
  }

  return response.json();
}
