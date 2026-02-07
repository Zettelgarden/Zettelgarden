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
}

export interface ArticleFilters {
  folder?: string;
  unread?: boolean;
  feed_id?: number;
  limit?: number;
}

export interface UnreadCounts {
  folders: Record<string, number>;
  feeds: Record<number, number>;
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

export function refreshFeeds(): Promise<{ fetched: number }> {
  return getData(apiClient.post<{ fetched: number }>("/rss/feeds/fetch", {}));
}

// Article API
export function listArticles(filters?: ArticleFilters): Promise<RSSArticle[]> {
  const params = new URLSearchParams();
  if (filters?.folder) params.set("folder", filters.folder);
  if (filters?.unread) params.set("unread", "true");
  if (filters?.feed_id) params.set("feed_id", filters.feed_id.toString());
  if (filters?.limit) params.set("limit", filters.limit.toString());

  const query = params.toString();
  return getData(apiClient.get<RSSArticle[]>(`/rss/articles${query ? `?${query}` : ""}`)).then(data => data ?? []);
}

export function getArticle(id: number): Promise<RSSArticle> {
  return getData(apiClient.get<RSSArticle>(`/rss/articles/${id}`));
}

export function markAsRead(id: number, read: boolean = true): Promise<void> {
  return getData(apiClient.post<void>(`/rss/articles/${id}/read`, { read }));
}

export function convertToCard(id: number, params?: ConvertArticleParams): Promise<any> {
  return getData(apiClient.post<any>(`/rss/articles/${id}/convert`, params));
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

// Unread Counts API
export function getUnreadCounts(): Promise<UnreadCounts> {
  return getData(apiClient.get<UnreadCounts>("/rss/unread-counts"));
}
