// @vitest-environment happy-dom

import React from 'react';
import { describe, it, expect, vi, beforeEach, afterEach } from 'vitest';
import { render, screen, waitFor, cleanup } from '@testing-library/react';
import userEvent from '@testing-library/user-event';
import { BrowserRouter } from 'react-router-dom';
import { RssPage } from '../RssPage';
import { renderWithProviders } from '../../tests/utils';

// Mock the RSS API
vi.mock('../../api/rss', () => ({
  listFeeds: vi.fn(() => Promise.resolve([])),
  listArticles: vi.fn(() => Promise.resolve([])),
  listFolders: vi.fn(() => Promise.resolve([])),
  markAsRead: vi.fn(() => Promise.resolve()),
  convertToCard: vi.fn(() => Promise.resolve({ id: 1 })),
  refreshFeeds: vi.fn(() => Promise.resolve({ fetched: 0 })),
  getUnreadCounts: vi.fn(() => Promise.resolve({ folders: {}, feeds: {} })),
}));

// Mock the RSS components to avoid complex dependencies
vi.mock('../../components/rss/RssAddFeedDialog', () => ({
  RssAddFeedDialog: ({ isOpen, onClose, onFeedAdded }: any) =>
    isOpen ? (
      <div data-testid="add-feed-dialog">
        <button onClick={() => onFeedAdded({ id: 1, name: 'Test Feed' })}>
          Add Feed
        </button>
        <button onClick={onClose}>Cancel</button>
      </div>
    ) : null,
}));

vi.mock('../../components/rss/RssEditFeedDialog', () => ({
  RssEditFeedDialog: ({ isOpen, onClose }: any) =>
    isOpen ? (
      <div data-testid="edit-feed-dialog">
        <button onClick={onClose}>Cancel</button>
      </div>
    ) : null,
}));

vi.mock('../../components/rss/RssEditFolderDialog', () => ({
  RssEditFolderDialog: ({ isOpen, onClose }: any) =>
    isOpen ? (
      <div data-testid="edit-folder-dialog">
        <button onClick={onClose}>Cancel</button>
      </div>
    ) : null,
}));

vi.mock('../../components/rss/RssCreateFolderDialog', () => ({
  RssCreateFolderDialog: ({ isOpen, onClose }: any) =>
    isOpen ? (
      <div data-testid="create-folder-dialog">
        <button onClick={onClose}>Cancel</button>
      </div>
    ) : null,
}));

vi.mock('../../components/rss/RssConfirmDialog', () => ({
  RssConfirmDialog: ({ isOpen, onClose }: any) =>
    isOpen ? (
      <div data-testid="confirm-dialog">
        <button onClick={onClose}>Cancel</button>
      </div>
    ) : null,
}));

vi.mock('../../components/rss/RssConvertDialog', () => ({
  RssConvertDialog: ({ isOpen, onClose, onConverted }: any) =>
    isOpen ? (
      <div data-testid="convert-dialog">
        <button onClick={() => onConverted(1)}>Convert</button>
        <button onClick={onClose}>Cancel</button>
      </div>
    ) : null,
}));

vi.mock('../../components/rss/RssImportDialog', () => ({
  RssImportDialog: ({ isOpen, onClose }: any) =>
    isOpen ? (
      <div data-testid="import-dialog">
        <button onClick={onClose}>Cancel</button>
      </div>
    ) : null,
}));

vi.mock('../../components/rss/RssDesktopLayout', () => {
  return {
    RssDesktopLayout: ({ ...props }: any) => (
      <div data-testid="desktop-layout">
        <div>RSS Feeds</div>
        <div>Articles</div>
        <button onClick={() => props.onAddFeed && props.onAddFeed()}>
          Add Feed
        </button>
        <button onClick={() => props.onRefresh && props.onRefresh()}>
          Refresh All
        </button>
        <label>
          <input
            type="checkbox"
            checked={props.showUnreadOnly || false}
            onChange={() =>
              props.onToggleShowUnreadOnly && props.onToggleShowUnreadOnly()
            }
          />
          Unread only
        </label>
        <div>
          {props.selectedFolder
            ? 'Folder: ' + props.selectedFolder
            : 'All Feeds (' + (props.feeds?.length || 0) + ')'}
        </div>
        <div>
          {(props.articles?.length || 0) === 0
            ? 'No articles found'
            : 'Articles loaded'}
        </div>
        <div>
          {props.selectedArticle
            ? 'Article content'
            : 'Select an article to read'}
        </div>
      </div>
    ),
  };
});

vi.mock('../../components/rss/RssMobileLayout', () => ({
  RssMobileLayout: ({ ...props }: any) => (
    <div data-testid="mobile-layout">
      <div>Mobile RSS</div>
    </div>
  ),
}));

// Mock the title utility
vi.mock('../../utils/title', () => ({
  setDocumentTitle: vi.fn(),
}));

describe('RssPage', () => {
  beforeEach(() => {
    vi.clearAllMocks();
  });

  afterEach(() => {
    cleanup();
  });

  it('renders the RSS page', async () => {
    renderWithProviders(<RssPage />);

    await waitFor(() => {
      expect(screen.getByText('RSS Feeds')).toBeInTheDocument();
    });
  });

  it('shows refresh button', async () => {
    renderWithProviders(<RssPage />);

    await waitFor(() => {
      expect(screen.getByText('Refresh All')).toBeInTheDocument();
    });
  });

  it('shows add feed button', async () => {
    renderWithProviders(<RssPage />);

    await waitFor(() => {
      expect(screen.getByText('Add Feed')).toBeInTheDocument();
    });
  });

  it('shows articles panel', async () => {
    renderWithProviders(<RssPage />);

    await waitFor(() => {
      expect(screen.getByText('Articles')).toBeInTheDocument();
    });
  });

  it('shows empty state when no articles', async () => {
    renderWithProviders(<RssPage />);

    await waitFor(() => {
      expect(screen.getByText('No articles found')).toBeInTheDocument();
    });
  });

  it('shows select article message when no article selected', async () => {
    renderWithProviders(<RssPage />);

    await waitFor(() => {
      expect(screen.getByText('Select an article to read')).toBeInTheDocument();
    });
  });

  it('shows all feeds button with count', async () => {
    renderWithProviders(<RssPage />);

    await waitFor(() => {
      expect(screen.getByText(/All Feeds/)).toBeInTheDocument();
    });
  });

  it('shows unread only checkbox', async () => {
    renderWithProviders(<RssPage />);

    await waitFor(() => {
      const checkbox = screen.getByRole('checkbox', { name: /unread only/i });
      expect(checkbox).toBeInTheDocument();
    });
  });

  it('shows loading state initially', () => {
    renderWithProviders(<RssPage />);

    expect(screen.getByText('Loading RSS feeds...')).toBeInTheDocument();
  });
});
