import { apiClient, getData } from "./client";

// Types
export interface Notification {
  id: number;
  user_id: number;
  source_type: 'rss' | 'task';
  source_id: number;
  title: string;
  preview: string;
  timestamp: string;
  importance_score: number;
  is_read: boolean;
  is_archived: boolean;
  filter_tags: string[];
}

export interface NotificationPreferences {
  user_id: number;
  show_starred_articles: boolean;
  show_priority_tasks: boolean;
  show_priority_feeds: boolean;
  items_per_page: number;
}

export interface NotificationListFilters {
  source_type?: string;
  unread_only?: boolean;
  limit?: number;
  offset?: number;
}

export interface NotificationListResponse {
  notifications: Notification[];
  total: number;
  unread_count: number;
}

// Notification API
export function listNotifications(filters?: NotificationListFilters): Promise<NotificationListResponse> {
  const params = new URLSearchParams();
  if (filters?.source_type) params.set("source_type", filters.source_type);
  if (filters?.unread_only) params.set("unread_only", "true");
  if (filters?.limit) params.set("limit", filters.limit.toString());
  if (filters?.offset) params.set("offset", filters.offset.toString());

  const query = params.toString();
  return getData(apiClient.get<NotificationListResponse>(`/notifications${query ? `?${query}` : ""}`));
}

export function getUnreadCount(): Promise<{ unread_count: number }> {
  return getData(apiClient.get<{ unread_count: number }>("/notifications/unread-count"));
}

export function markAsRead(id: number, read: boolean): Promise<void> {
  return getData(apiClient.patch<void>(`/notifications/${id}/read`, { read }));
}

export function archiveNotification(id: number): Promise<void> {
  return getData(apiClient.patch<void>(`/notifications/${id}/archive`, {}));
}

export function getPreferences(): Promise<NotificationPreferences> {
  return getData(apiClient.get<NotificationPreferences>("/notifications/preferences"));
}

export function updatePreferences(prefs: Partial<NotificationPreferences>): Promise<NotificationPreferences> {
  return getData(apiClient.patch<NotificationPreferences>("/notifications/preferences", prefs));
}
