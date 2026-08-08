import React, { useState, useMemo } from 'react';
import { RSSFeed, RSSFolder, UnreadCounts } from '../../api/rss';
import { RssBulkMoveDialog } from './RssBulkMoveDialog';
import { RssBulkTagsDialog } from './RssBulkTagsDialog';

interface RssManageFeedsTableProps {
  feeds: RSSFeed[];
  folders: RSSFolder[];
  unreadCounts: UnreadCounts;
  onUpdateFeed: (id: number, params: any) => Promise<RSSFeed>;
  onDeleteFeed: (id: number) => Promise<void>;
  onBulkUpdate: (feedIds: number[], params: any) => Promise<void>;
  onBulkDelete: (feedIds: number[]) => Promise<void>;
  refreshing: boolean;
  isMobile: boolean;
}

type SortField = 'name' | 'url' | 'folder' | 'last_fetched' | 'unread';
type SortOrder = 'asc' | 'desc';

export function RssManageFeedsTable({
  feeds,
  folders,
  unreadCounts,
  onUpdateFeed,
  onDeleteFeed,
  onBulkUpdate,
  onBulkDelete,
  refreshing,
  isMobile,
}: RssManageFeedsTableProps) {
  const [selectedIds, setSelectedIds] = useState<Set<number>>(new Set());
  const [sortField, setSortField] = useState<SortField>('name');
  const [sortOrder, setSortOrder] = useState<SortOrder>('asc');
  const [searchQuery, setSearchQuery] = useState('');
  const [editingFeed, setEditingFeed] = useState<RSSFeed | null>(null);
  const [showBulkMove, setShowBulkMove] = useState(false);
  const [showBulkTags, setShowBulkTags] = useState(false);
  const [showBulkDelete, setShowBulkDelete] = useState(false);

  // Filter and sort feeds
  const filteredAndSortedFeeds = useMemo(() => {
    let result = [...feeds];

    // Filter by search
    if (searchQuery) {
      const query = searchQuery.toLowerCase();
      result = result.filter(
        (f) =>
          f.name.toLowerCase().includes(query) ||
          f.url.toLowerCase().includes(query),
      );
    }

    // Sort
    result.sort((a, b) => {
      let aVal: any, bVal: any;

      switch (sortField) {
        case 'name':
          aVal = a.name.toLowerCase();
          bVal = b.name.toLowerCase();
          break;
        case 'url':
          aVal = a.url.toLowerCase();
          bVal = b.url.toLowerCase();
          break;
        case 'folder':
          aVal = a.folder || '';
          bVal = b.folder || '';
          break;
        case 'last_fetched':
          aVal = a.last_fetched_at ? new Date(a.last_fetched_at).getTime() : 0;
          bVal = b.last_fetched_at ? new Date(b.last_fetched_at).getTime() : 0;
          break;
        case 'unread':
          aVal = unreadCounts.feeds[a.id] || 0;
          bVal = unreadCounts.feeds[b.id] || 0;
          break;
        default:
          return 0;
      }

      if (aVal < bVal) return sortOrder === 'asc' ? -1 : 1;
      if (aVal > bVal) return sortOrder === 'asc' ? 1 : -1;
      return 0;
    });

    return result;
  }, [feeds, searchQuery, sortField, sortOrder, unreadCounts]);

  const handleSort = (field: SortField) => {
    if (sortField === field) {
      setSortOrder(sortOrder === 'asc' ? 'desc' : 'asc');
    } else {
      setSortField(field);
      setSortOrder('asc');
    }
  };

  const handleSelectAll = () => {
    if (selectedIds.size === filteredAndSortedFeeds.length) {
      setSelectedIds(new Set());
    } else {
      setSelectedIds(new Set(filteredAndSortedFeeds.map((f) => f.id)));
    }
  };

  const handleSelectOne = (id: number) => {
    const next = new Set(selectedIds);
    if (next.has(id)) {
      next.delete(id);
    } else {
      next.add(id);
    }
    setSelectedIds(next);
  };

  const handleBulkEnable = async () => {
    await onBulkUpdate(Array.from(selectedIds), { enabled: true });
    setSelectedIds(new Set());
  };

  const handleBulkDisable = async () => {
    await onBulkUpdate(Array.from(selectedIds), { enabled: false });
    setSelectedIds(new Set());
  };

  const handleBulkDeleteConfirm = async () => {
    await onBulkDelete(Array.from(selectedIds));
    setSelectedIds(new Set());
    setShowBulkDelete(false);
  };

  const handleBulkMove = async (folder: string | null) => {
    await onBulkUpdate(Array.from(selectedIds), {
      folder: folder || undefined,
    });
    setSelectedIds(new Set());
    setShowBulkMove(false);
  };

  const handleBulkTags = async (tags: string, mode: 'replace' | 'append') => {
    const feedUpdates = Array.from(selectedIds).map((id) => {
      const feed = feeds.find((f) => f.id === id);
      let finalTags = tags;
      if (mode === 'append' && feed?.auto_tags) {
        const existing = feed.auto_tags
          .split(',')
          .map((t) => t.trim())
          .filter((t) => t);
        const newTags = tags
          .split(',')
          .map((t) => t.trim())
          .filter((t) => t);
        const combined = [...new Set([...existing, ...newTags])];
        finalTags = combined.join(', ');
      }
      return { id, params: { auto_tags: finalTags } };
    });

    for (const { id, params } of feedUpdates) {
      await onUpdateFeed(id, params);
    }

    setSelectedIds(new Set());
    setShowBulkTags(false);
  };

  const formatRelativeTime = (dateStr?: string) => {
    if (!dateStr) return 'Never';
    const date = new Date(dateStr);
    const now = new Date();
    const seconds = Math.floor((now.getTime() - date.getTime()) / 1000);

    if (seconds < 60) return 'Just now';
    if (seconds < 3600) return `${Math.floor(seconds / 60)}m ago`;
    if (seconds < 86400) return `${Math.floor(seconds / 3600)}h ago`;
    if (seconds < 604800) return `${Math.floor(seconds / 86400)}d ago`;
    return date.toLocaleDateString();
  };

  const getUnreadCount = (feedId: number) => {
    return unreadCounts.feeds[feedId] || 0;
  };

  return (
    <div className="flex-1 overflow-hidden flex flex-col">
      {/* Bulk Actions Toolbar */}
      {selectedIds.size > 0 && (
        <div className="bg-blue-50 border-b border-blue-200 px-4 py-3 flex items-center gap-3">
          <span className="text-sm font-medium text-blue-900">
            {selectedIds.size} feed{selectedIds.size !== 1 ? 's' : ''} selected
          </span>
          <div className="h-4 w-px bg-blue-300" />
          <button
            onClick={handleBulkEnable}
            className="px-3 py-1 text-sm bg-white border border-gray-300 rounded-md hover:bg-gray-50"
          >
            Enable
          </button>
          <button
            onClick={handleBulkDisable}
            className="px-3 py-1 text-sm bg-white border border-gray-300 rounded-md hover:bg-gray-50"
          >
            Disable
          </button>
          <button
            onClick={() => setShowBulkMove(true)}
            className="px-3 py-1 text-sm bg-white border border-gray-300 rounded-md hover:bg-gray-50"
          >
            Move to folder...
          </button>
          <button
            onClick={() => setShowBulkTags(true)}
            className="px-3 py-1 text-sm bg-white border border-gray-300 rounded-md hover:bg-gray-50"
          >
            Set tags...
          </button>
          <button
            onClick={() => setShowBulkDelete(true)}
            className="px-3 py-1 text-sm bg-white border border-red-300 text-red-600 rounded-md hover:bg-red-50"
          >
            Delete
          </button>
          <button
            onClick={() => setSelectedIds(new Set())}
            className="ml-auto text-sm text-blue-600 hover:underline"
          >
            Clear selection
          </button>
        </div>
      )}

      {/* Search and Filter Bar */}
      <div className="bg-white border-b border-gray-200 px-4 py-3 flex items-center gap-4">
        <div className="relative flex-1">
          <input
            type="text"
            value={searchQuery}
            onChange={(e) => setSearchQuery(e.target.value)}
            placeholder="Search feeds by name or URL..."
            className="w-full pl-10 pr-4 py-2 border border-gray-300 rounded-md focus:ring-2 focus:ring-blue-500 focus:border-blue-500"
          />
          <svg
            className="w-5 h-5 absolute left-3 top-2.5 text-gray-400"
            fill="none"
            viewBox="0 0 24 24"
            stroke="currentColor"
          >
            <path
              strokeLinecap="round"
              strokeLinejoin="round"
              strokeWidth={2}
              d="M21 21l-6-6m2-5a7 7 0 11-14 0 7 7 0 0114 0z"
            />
          </svg>
        </div>
        <div className="text-sm text-gray-500">
          {filteredAndSortedFeeds.length} feed
          {filteredAndSortedFeeds.length !== 1 ? 's' : ''}
        </div>
      </div>

      {/* Table */}
      <div className="flex-1 overflow-auto">
        <div className="min-w-[800px]">
          <table className="w-full">
            <thead className="bg-gray-50 sticky top-0">
              <tr>
                <th className="px-4 py-3 text-left">
                  <input
                    type="checkbox"
                    checked={
                      selectedIds.size === filteredAndSortedFeeds.length &&
                      filteredAndSortedFeeds.length > 0
                    }
                    onChange={handleSelectAll}
                    className="rounded border-gray-300"
                  />
                </th>
                <th className="px-4 py-3 text-left">
                  <button
                    onClick={() => handleSort('name')}
                    className="flex items-center gap-1 font-medium text-gray-700 hover:text-gray-900"
                  >
                    Name
                    {sortField === 'name' && (
                      <span className="text-gray-400">
                        {sortOrder === 'asc' ? '↑' : '↓'}
                      </span>
                    )}
                  </button>
                </th>
                <th className="px-4 py-3 text-left">
                  <button
                    onClick={() => handleSort('url')}
                    className="flex items-center gap-1 font-medium text-gray-700 hover:text-gray-900"
                  >
                    URL
                    {sortField === 'url' && (
                      <span className="text-gray-400">
                        {sortOrder === 'asc' ? '↑' : '↓'}
                      </span>
                    )}
                  </button>
                </th>
                <th className="px-4 py-3 text-left">
                  <button
                    onClick={() => handleSort('folder')}
                    className="flex items-center gap-1 font-medium text-gray-700 hover:text-gray-900"
                  >
                    Folder
                    {sortField === 'folder' && (
                      <span className="text-gray-400">
                        {sortOrder === 'asc' ? '↑' : '↓'}
                      </span>
                    )}
                  </button>
                </th>
                <th className="px-4 py-3 text-left font-medium text-gray-700">
                  Tags
                </th>
                <th className="px-4 py-3 text-left font-medium text-gray-700">
                  Status
                </th>
                <th className="px-4 py-3 text-left">
                  <button
                    onClick={() => handleSort('last_fetched')}
                    className="flex items-center gap-1 font-medium text-gray-700 hover:text-gray-900"
                  >
                    Last Fetched
                    {sortField === 'last_fetched' && (
                      <span className="text-gray-400">
                        {sortOrder === 'asc' ? '↑' : '↓'}
                      </span>
                    )}
                  </button>
                </th>
                <th className="px-4 py-3 text-left font-medium text-gray-700">
                  Error
                </th>
                <th className="px-4 py-3 text-left">
                  <button
                    onClick={() => handleSort('unread')}
                    className="flex items-center gap-1 font-medium text-gray-700 hover:text-gray-900"
                  >
                    Unread
                    {sortField === 'unread' && (
                      <span className="text-gray-400">
                        {sortOrder === 'asc' ? '↑' : '↓'}
                      </span>
                    )}
                  </button>
                </th>
                <th className="px-4 py-3 text-left font-medium text-gray-700">
                  Actions
                </th>
              </tr>
            </thead>
            <tbody className="divide-y divide-gray-200">
              {filteredAndSortedFeeds.length === 0 ? (
                <tr>
                  <td
                    colSpan={10}
                    className="px-4 py-12 text-center text-gray-500"
                  >
                    {searchQuery
                      ? 'No feeds match your search.'
                      : 'No feeds yet.'}
                  </td>
                </tr>
              ) : (
                filteredAndSortedFeeds.map((feed) => (
                  <tr
                    key={feed.id}
                    className={`hover:bg-gray-50 ${
                      selectedIds.has(feed.id) ? 'bg-blue-50' : ''
                    }`}
                  >
                    <td className="px-4 py-3">
                      <input
                        type="checkbox"
                        checked={selectedIds.has(feed.id)}
                        onChange={() => handleSelectOne(feed.id)}
                        className="rounded border-gray-300"
                      />
                    </td>
                    <td className="px-4 py-3">
                      <div className="flex items-center gap-2">
                        <span className="font-medium">{feed.name}</span>
                        {feed.priority && (
                          <span
                            className="text-amber-500"
                            title="Priority feed"
                          >
                            <svg
                              className="w-4 h-4"
                              fill="currentColor"
                              viewBox="0 0 20 20"
                            >
                              <path d="M9.049 2.927c.3-.921 1.603-.921 1.902 0l1.07 3.292a1 1 0 00.95.69h3.462c.969 0 1.371 1.24.588 1.81l-2.8 2.034a1 1 0 00-.364 1.118l1.07 3.292c.3.921-.755 1.688-1.54 1.118l-2.8-2.034a1 1 0 00-1.175 0l-2.8 2.034c-.784.57-1.838-.197-1.539-1.118l1.07-3.292a1 1 0 00-.364-1.118L2.98 8.72c-.783-.57-.38-1.81.588-1.81h3.461a1 1 0 00.951-.69l1.07-3.292z" />
                            </svg>
                          </span>
                        )}
                      </div>
                    </td>
                    <td className="px-4 py-3">
                      <div
                        className="max-w-xs truncate text-gray-500 text-sm"
                        title={feed.url}
                      >
                        {feed.url}
                      </div>
                    </td>
                    <td className="px-4 py-3">
                      {feed.folder ? (
                        <span className="px-2 py-1 bg-gray-100 text-gray-700 text-xs rounded-full">
                          {feed.folder}
                        </span>
                      ) : (
                        <span className="text-gray-400 text-sm">
                          Uncategorized
                        </span>
                      )}
                    </td>
                    <td className="px-4 py-3">
                      {feed.auto_tags ? (
                        <div className="flex flex-wrap gap-1 max-w-xs">
                          {feed.auto_tags
                            .split(',')
                            .slice(0, 3)
                            .map((tag, i) => (
                              <span
                                key={i}
                                className="px-2 py-0.5 bg-blue-100 text-blue-700 text-xs rounded-full"
                              >
                                {tag.trim()}
                              </span>
                            ))}
                          {feed.auto_tags.split(',').length > 3 && (
                            <span className="text-xs text-gray-400">
                              +{feed.auto_tags.split(',').length - 3}
                            </span>
                          )}
                        </div>
                      ) : (
                        <span className="text-gray-400 text-sm">None</span>
                      )}
                    </td>
                    <td className="px-4 py-3">
                      <div className="flex items-center gap-2">
                        {feed.enabled ? (
                          <span className="text-green-600" title="Enabled">
                            <svg
                              className="w-4 h-4"
                              fill="currentColor"
                              viewBox="0 0 20 20"
                            >
                              <path
                                fillRule="evenodd"
                                d="M16.707 5.293a1 1 0 010 1.414l-8 8a1 1 0 01-1.414 0l-4-4a1 1 0 011.414-1.414L8 12.586l7.293-7.293a1 1 0 011.414 0z"
                                clipRule="evenodd"
                              />
                            </svg>
                          </span>
                        ) : (
                          <span className="text-gray-400" title="Disabled">
                            <svg
                              className="w-4 h-4"
                              fill="currentColor"
                              viewBox="0 0 20 20"
                            >
                              <path
                                fillRule="evenodd"
                                d="M4.293 4.293a1 1 0 011.414 0L10 8.586l4.293-4.293a1 1 0 111.414 1.414L11.414 10l4.293 4.293a1 1 0 01-1.414 1.414L10 11.414l-4.293 4.293a1 1 0 01-1.414-1.414L8.586 10 4.293 5.707a1 1 0 010-1.414z"
                                clipRule="evenodd"
                              />
                            </svg>
                          </span>
                        )}
                      </div>
                    </td>
                    <td className="px-4 py-3 text-sm text-gray-500">
                      {formatRelativeTime(feed.last_fetched_at)}
                    </td>
                    <td className="px-4 py-3">
                      {feed.last_error ? (
                        <span className="text-red-500" title={feed.last_error}>
                          <svg
                            className="w-4 h-4"
                            fill="currentColor"
                            viewBox="0 0 20 20"
                          >
                            <path
                              fillRule="evenodd"
                              d="M18 10a8 8 0 11-16 0 8 8 0 0116 0zm-7 4a1 1 0 11-2 0 1 1 0 012 0zm-1-9a1 1 0 00-1 1v4a1 1 0 102 0V6a1 1 0 00-1-1z"
                              clipRule="evenodd"
                            />
                          </svg>
                        </span>
                      ) : (
                        <span className="text-gray-300">
                          <svg
                            className="w-4 h-4"
                            fill="currentColor"
                            viewBox="0 0 20 20"
                          >
                            <path
                              fillRule="evenodd"
                              d="M10 18a8 8 0 100-16 8 8 0 000 16zm3.707-9.293a1 1 0 00-1.414-1.414L9 10.586 7.707 9.293a1 1 0 00-1.414 1.414l2 2a1 1 0 001.414 0l4-4z"
                              clipRule="evenodd"
                            />
                          </svg>
                        </span>
                      )}
                    </td>
                    <td className="px-4 py-3">
                      {getUnreadCount(feed.id) > 0 && (
                        <span className="bg-red-500 text-white text-xs font-bold px-2 py-0.5 rounded-full">
                          {getUnreadCount(feed.id)}
                        </span>
                      )}
                    </td>
                    <td className="px-4 py-3">
                      <div className="flex items-center gap-2">
                        <button
                          onClick={() => setEditingFeed(feed)}
                          className="p-1 text-gray-400 hover:text-blue-600"
                          title="Edit feed"
                        >
                          <svg
                            className="w-4 h-4"
                            fill="none"
                            viewBox="0 0 24 24"
                            stroke="currentColor"
                          >
                            <path
                              strokeLinecap="round"
                              strokeLinejoin="round"
                              strokeWidth={2}
                              d="M11 5H6a2 2 0 00-2 2v11a2 2 0 002 2h11a2 2 0 002-2v-5m-1.414-9.414a2 2 0 112.828 2.828L11.828 15H9v-2.828l8.586-8.586z"
                            />
                          </svg>
                        </button>
                        <button
                          onClick={() => onDeleteFeed(feed.id)}
                          className="p-1 text-gray-400 hover:text-red-600"
                          title="Delete feed"
                        >
                          <svg
                            className="w-4 h-4"
                            fill="none"
                            viewBox="0 0 24 24"
                            stroke="currentColor"
                          >
                            <path
                              strokeLinecap="round"
                              strokeLinejoin="round"
                              strokeWidth={2}
                              d="M19 7l-.867 12.142A2 2 0 0116.138 21H7.862a2 2 0 01-1.995-1.858L5 7m5 4v6m4-6v6m1-10V4a1 1 0 00-1-1h-4a1 1 0 00-1 1v3M4 7h16"
                            />
                          </svg>
                        </button>
                      </div>
                    </td>
                  </tr>
                ))
              )}
            </tbody>
          </table>
        </div>
      </div>

      {/* Edit Feed Dialog - Reuse existing dialog */}
      {editingFeed && (
        <div className="fixed inset-0 bg-black/50 flex items-center justify-center z-50">
          <div className="bg-white rounded-lg p-6 w-[500px]">
            <h3 className="text-lg font-semibold mb-4">Edit Feed</h3>
            <form
              onSubmit={async (e) => {
                e.preventDefault();
                const formData = new FormData(e.currentTarget);
                await onUpdateFeed(editingFeed.id, {
                  name: formData.get('name') as string,
                  url: formData.get('url') as string,
                  folder: (formData.get('folder') as string) || undefined,
                  auto_tags: formData.get('tags') as string,
                  fetch_interval:
                    parseInt(formData.get('interval') as string) || undefined,
                  enabled: formData.get('enabled') === 'on',
                  priority: formData.get('priority') === 'on',
                });
                setEditingFeed(null);
              }}
            >
              <div className="space-y-4">
                <div>
                  <label className="block text-sm font-medium text-gray-700 mb-1">
                    Name
                  </label>
                  <input
                    name="name"
                    defaultValue={editingFeed.name}
                    className="w-full px-3 py-2 border border-gray-300 rounded-md"
                    required
                  />
                </div>
                <div>
                  <label className="block text-sm font-medium text-gray-700 mb-1">
                    URL
                  </label>
                  <input
                    name="url"
                    defaultValue={editingFeed.url}
                    type="url"
                    className="w-full px-3 py-2 border border-gray-300 rounded-md"
                    required
                  />
                </div>
                <div>
                  <label className="block text-sm font-medium text-gray-700 mb-1">
                    Folder
                  </label>
                  <select
                    name="folder"
                    defaultValue={editingFeed.folder || ''}
                    className="w-full px-3 py-2 border border-gray-300 rounded-md"
                  >
                    <option value="">Uncategorized</option>
                    {folders.map((f) => (
                      <option key={f.id} value={f.name}>
                        {f.name}
                      </option>
                    ))}
                  </select>
                </div>
                <div>
                  <label className="block text-sm font-medium text-gray-700 mb-1">
                    Auto Tags (comma-separated)
                  </label>
                  <input
                    name="tags"
                    defaultValue={editingFeed.auto_tags}
                    className="w-full px-3 py-2 border border-gray-300 rounded-md"
                  />
                </div>
                <div>
                  <label className="block text-sm font-medium text-gray-700 mb-1">
                    Fetch Interval (minutes)
                  </label>
                  <input
                    name="interval"
                    defaultValue={editingFeed.fetch_interval}
                    type="number"
                    min="1"
                    className="w-full px-3 py-2 border border-gray-300 rounded-md"
                  />
                </div>
                <div className="flex gap-4">
                  <label className="flex items-center gap-2">
                    <input
                      name="enabled"
                      type="checkbox"
                      defaultChecked={editingFeed.enabled}
                    />
                    <span className="text-sm">Enabled</span>
                  </label>
                  <label className="flex items-center gap-2">
                    <input
                      name="priority"
                      type="checkbox"
                      defaultChecked={editingFeed.priority}
                    />
                    <span className="text-sm">Priority</span>
                  </label>
                </div>
              </div>
              <div className="flex justify-end gap-2 mt-6">
                <button
                  type="button"
                  onClick={() => setEditingFeed(null)}
                  className="px-4 py-2 bg-gray-200 text-gray-700 rounded-md hover:bg-gray-300"
                >
                  Cancel
                </button>
                <button
                  type="submit"
                  className="px-4 py-2 bg-blue-600 text-white rounded-md hover:bg-blue-700"
                >
                  Save
                </button>
              </div>
            </form>
          </div>
        </div>
      )}

      {/* Bulk Delete Confirmation */}
      {showBulkDelete && (
        <div className="fixed inset-0 bg-black/50 flex items-center justify-center z-50">
          <div className="bg-white rounded-lg p-6 w-96">
            <h3 className="text-lg font-semibold mb-2">Delete Feeds</h3>
            <p className="text-gray-600 mb-4">
              Are you sure you want to delete {selectedIds.size} feed
              {selectedIds.size !== 1 ? 's' : ''}? This will also delete all
              articles from these feeds. This action cannot be undone.
            </p>
            <div className="flex justify-end gap-2">
              <button
                onClick={() => setShowBulkDelete(false)}
                className="px-4 py-2 bg-gray-200 text-gray-700 rounded-md hover:bg-gray-300"
              >
                Cancel
              </button>
              <button
                onClick={handleBulkDeleteConfirm}
                className="px-4 py-2 bg-red-600 text-white rounded-md hover:bg-red-700"
              >
                Delete
              </button>
            </div>
          </div>
        </div>
      )}

      {/* Bulk Move Dialog */}
      <RssBulkMoveDialog
        isOpen={showBulkMove}
        onClose={() => setShowBulkMove(false)}
        onConfirm={handleBulkMove}
        folders={folders}
        feedCount={selectedIds.size}
      />

      {/* Bulk Tags Dialog */}
      <RssBulkTagsDialog
        isOpen={showBulkTags}
        onClose={() => setShowBulkTags(false)}
        onConfirm={handleBulkTags}
        feedCount={selectedIds.size}
      />

      {/* Mobile message when table is too wide */}
      {isMobile && (
        <div className="md:hidden fixed bottom-0 left-0 right-0 bg-yellow-100 border-t border-yellow-200 p-3 text-sm text-yellow-800">
          Table view is best viewed on larger screens.
        </div>
      )}
    </div>
  );
}
