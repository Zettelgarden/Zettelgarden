/**
 * Smoke test: NotificationInboxPage renders the inbox with notifications (a5q.6).
 */

import { describe, it, expect, vi, beforeEach } from 'vitest';
import { render, screen, waitFor } from '@testing-library/react';
import { renderWithProviders } from '../../tests/utils';
import { NotificationInboxPage } from '../NotificationInboxPage';

vi.mock('../../api/notifications', () => ({
  listNotifications: vi.fn(),
  markAsRead: vi.fn(),
  getUnreadCount: vi.fn(),
}));

vi.mock('../../api/tasks', () => ({
  fetchTask: vi.fn(),
  saveExistingTask: vi.fn(),
}));

vi.mock('../../api/rss', () => ({
  getArticle: vi.fn(),
  getUnreadCounts: vi.fn(async () => ({ feeds: {} })),
}));

const { listNotifications, getUnreadCount } = await import(
  '../../api/notifications'
);

const notifications: import('../../api/notifications').Notification[] = [
  {
    id: 1,
    user_id: 1,
    source_type: 'rss',
    source_id: 11,
    title: 'New article from feed',
    preview: 'An interesting read',
    timestamp: '2024-01-01T00:00:00Z',
    importance_score: 0.8,
    is_read: false,
    is_archived: false,
    filter_tags: [],
  },
  {
    id: 2,
    user_id: 1,
    source_type: 'task',
    source_id: 22,
    title: 'Task reminder: submit report',
    preview: 'Due today',
    timestamp: '2024-01-01T00:00:00Z',
    importance_score: 0.5,
    is_read: false,
    is_archived: false,
    filter_tags: [],
  },
];

describe('NotificationInboxPage smoke', () => {
  beforeEach(() => {
    vi.clearAllMocks();
  });

  it('renders the inbox with notifications from both sources', async () => {
    vi.mocked(listNotifications).mockResolvedValue({
      notifications,
      total: 2,
      unread_count: 2,
    });
    vi.mocked(getUnreadCount).mockResolvedValue({ unread_count: 2 });

    renderWithProviders(<NotificationInboxPage />);

    // The "Notifications" header renders during loading too; wait for rows.
    await waitFor(() =>
      expect(screen.getByText('New article from feed')).toBeInTheDocument(),
    );
    expect(
      screen.getByText('Task reminder: submit report'),
    ).toBeInTheDocument();
    // Tab buttons
    expect(screen.getByText('All')).toBeInTheDocument();
    expect(screen.getByText('RSS')).toBeInTheDocument();
    expect(screen.getByText('Tasks')).toBeInTheDocument();
  });

  it('renders the empty state when there are no notifications', async () => {
    vi.mocked(listNotifications).mockResolvedValue({
      notifications: [],
      total: 0,
      unread_count: 0,
    });
    vi.mocked(getUnreadCount).mockResolvedValue({ unread_count: 0 });

    renderWithProviders(<NotificationInboxPage />);

    await waitFor(() =>
      expect(screen.getByText('No notifications')).toBeInTheDocument(),
    );
  });
});
