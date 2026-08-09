import React, { useState } from 'react';
import { Dialog, Transition } from '@headlessui/react';
import { Fragment } from 'react';
import { Spinner } from '../ui/Spinner';
import {
  updateFeed,
  UpdateRSSFeedParams,
  RSSFeed,
  RSSFolder,
} from '../../api/rss';

interface RssEditFeedDialogProps {
  isOpen: boolean;
  onClose: () => void;
  feed: RSSFeed | null;
  folders: RSSFolder[];
  onFeedUpdated: (feed: RSSFeed) => void;
}

export function RssEditFeedDialog({
  isOpen,
  onClose,
  feed,
  folders,
  onFeedUpdated,
}: RssEditFeedDialogProps) {
  const [name, setName] = useState('');
  const [folder, setFolder] = useState('');
  const [autoTags, setAutoTags] = useState('');
  const [enabled, setEnabled] = useState(true);
  const [priority, setPriority] = useState(false);
  const [loading, setLoading] = useState(false);
  const [error, setError] = useState<string>('');

  // Populate form when feed changes
  React.useEffect(() => {
    if (feed) {
      setName(feed.name);
      setFolder(feed.folder || '');
      setAutoTags(feed.auto_tags || '');
      setEnabled(feed.enabled);
      setPriority(feed.priority || false);
    }
  }, [feed]);

  const handleSubmit = async (e: React.FormEvent) => {
    e.preventDefault();
    if (!feed) {
      setError('No feed selected');
      return;
    }

    setLoading(true);
    setError('');

    try {
      const feedParams: UpdateRSSFeedParams = {};

      if (name.trim() !== feed.name) {
        feedParams.name = name.trim();
      }
      if (folder !== (feed.folder || '')) {
        feedParams.folder = folder.trim() || undefined;
      }
      if (autoTags !== feed.auto_tags) {
        feedParams.auto_tags = autoTags.trim();
      }
      if (enabled !== feed.enabled) {
        feedParams.enabled = enabled;
      }
      if (priority !== (feed.priority || false)) {
        feedParams.priority = priority;
      }

      const updatedFeed = await updateFeed(feed.id, feedParams);
      onFeedUpdated(updatedFeed);
      handleClose();
    } catch (err) {
      console.error('Failed to update feed:', err);
      setError(
        err instanceof Error
          ? err.message
          : 'Failed to update feed. Please try again.',
      );
    } finally {
      setLoading(false);
    }
  };

  const handleClose = () => {
    setName('');
    setFolder('');
    setAutoTags('');
    setEnabled(true);
    setPriority(false);
    setError('');
    onClose();
  };

  if (!feed) return null;

  return (
    <Transition appear show={isOpen} as={Fragment}>
      <Dialog as="div" className="relative z-[80]" onClose={handleClose}>
        <Transition.Child
          as={Fragment}
          enter="ease-out duration-300"
          enterFrom="opacity-0"
          enterTo="opacity-100"
          leave="ease-in duration-200"
          leaveFrom="opacity-100"
          leaveTo="opacity-0"
        >
          <div className="fixed inset-0 bg-black bg-opacity-30" />
        </Transition.Child>

        <div className="fixed inset-0 overflow-y-auto">
          <div className="flex min-h-full items-center justify-center p-4 text-center">
            <Transition.Child
              as={Fragment}
              enter="ease-out duration-300"
              enterFrom="opacity-0 scale-95"
              enterTo="opacity-100 scale-100"
              leave="ease-in duration-200"
              leaveFrom="opacity-100 scale-100"
              leaveTo="opacity-0 scale-95"
            >
              <Dialog.Panel className="w-full max-w-md transform overflow-hidden rounded-2xl bg-white p-6 text-left align-middle shadow-xl transition-all">
                <Dialog.Title
                  as="h3"
                  className="text-lg font-medium leading-6 text-gray-900 mb-4"
                >
                  Edit RSS Feed
                </Dialog.Title>

                <form onSubmit={handleSubmit} className="space-y-4">
                  {/* Feed URL - Read only */}
                  <div>
                    <label className="block text-sm font-medium text-gray-700 mb-1">
                      Feed URL
                    </label>
                    <input
                      type="text"
                      value={feed.url}
                      disabled
                      className="w-full px-3 py-2 border border-gray-300 rounded-md bg-gray-100 text-gray-500 sm:text-sm cursor-not-allowed"
                    />
                  </div>

                  {/* Name */}
                  <div>
                    <label
                      htmlFor="feed-name"
                      className="block text-sm font-medium text-gray-700 mb-1"
                    >
                      Name
                    </label>
                    <input
                      id="feed-name"
                      type="text"
                      value={name}
                      onChange={(e) => setName(e.target.value)}
                      placeholder="Feed Name"
                      className="w-full px-3 py-2 border border-gray-300 rounded-md shadow-sm focus:outline-none focus:ring-2 focus:ring-blue-500 focus:border-blue-500 sm:text-sm"
                    />
                  </div>

                  {/* Folder */}
                  <div>
                    <label
                      htmlFor="feed-folder"
                      className="block text-sm font-medium text-gray-700 mb-1"
                    >
                      Folder
                    </label>
                    <select
                      id="feed-folder"
                      value={folder}
                      onChange={(e) => setFolder(e.target.value)}
                      className="w-full px-3 py-2 border border-gray-300 rounded-md shadow-sm focus:outline-none focus:ring-2 focus:ring-blue-500 focus:border-blue-500 sm:text-sm"
                    >
                      <option value="">No folder</option>
                      {folders.map((f) => (
                        <option key={f.id} value={f.name}>
                          {f.name}
                        </option>
                      ))}
                    </select>
                  </div>

                  {/* Auto Tags */}
                  <div>
                    <label
                      htmlFor="feed-tags"
                      className="block text-sm font-medium text-gray-700 mb-1"
                    >
                      Auto Tags
                    </label>
                    <input
                      id="feed-tags"
                      type="text"
                      value={autoTags}
                      onChange={(e) => setAutoTags(e.target.value)}
                      placeholder="tech, ai, machine-learning"
                      className="w-full px-3 py-2 border border-gray-300 rounded-md shadow-sm focus:outline-none focus:ring-2 focus:ring-blue-500 focus:border-blue-500 sm:text-sm"
                    />
                    <p className="mt-1 text-xs text-gray-500">
                      Comma-separated tags to automatically apply to articles
                      from this feed
                    </p>
                  </div>

                  {/* Enabled */}
                  <div>
                    <label className="flex items-center gap-2 cursor-pointer">
                      <input
                        type="checkbox"
                        checked={enabled}
                        onChange={(e) => setEnabled(e.target.checked)}
                        className="rounded border-gray-300 text-blue-600 focus:ring-blue-500"
                      />
                      <span className="text-sm font-medium text-gray-700">
                        Enabled
                      </span>
                    </label>
                    <p className="mt-1 text-xs text-gray-500 ml-6">
                      When disabled, this feed won't be refreshed
                    </p>
                  </div>

                  {/* Priority */}
                  <div>
                    <label className="flex items-center gap-2 cursor-pointer">
                      <input
                        type="checkbox"
                        checked={priority}
                        onChange={(e) => setPriority(e.target.checked)}
                        className="rounded border-gray-300 text-blue-600 focus:ring-blue-500"
                      />
                      <span className="text-sm font-medium text-gray-700">
                        Priority feed
                      </span>
                    </label>
                    <p className="mt-1 text-xs text-gray-500 ml-6">
                      Priority feeds receive a +100 boost in the smart feed
                      ranking
                    </p>
                  </div>

                  {/* Error Message */}
                  {error && (
                    <div className="rounded-md bg-red-50 p-3">
                      <p className="text-sm text-red-800">{error}</p>
                    </div>
                  )}

                  {/* Action Buttons */}
                  <div className="flex justify-end space-x-2 pt-2">
                    <button
                      type="button"
                      onClick={handleClose}
                      disabled={loading}
                      className="px-4 py-2 min-h-[44px] text-gray-700 bg-gray-200 hover:bg-gray-300 rounded-md font-medium transition-colors disabled:opacity-50 disabled:cursor-not-allowed"
                    >
                      Cancel
                    </button>
                    <button
                      type="submit"
                      disabled={loading}
                      className="px-4 py-2 min-h-[44px] bg-blue-600 text-white hover:bg-blue-700 rounded-md font-medium transition-colors disabled:opacity-50 disabled:cursor-not-allowed flex items-center gap-2"
                    >
                      {loading ? (
                        <>
                          <Spinner size="sm" className="text-white" />
                          Saving...
                        </>
                      ) : (
                        'Save Changes'
                      )}
                    </button>
                  </div>
                </form>
              </Dialog.Panel>
            </Transition.Child>
          </div>
        </div>
      </Dialog>
    </Transition>
  );
}
