import React, { useState } from 'react';
import { Dialog, Transition } from '@headlessui/react';
import { Fragment } from 'react';
import { Spinner } from '../ui/Spinner';
import {
  createFeed,
  CreateRSSFeedParams,
  RSSFeed,
  RSSFolder,
  discoverFeed,
  DiscoverFeedResponse,
} from '../../api/rss';

interface RssAddFeedDialogProps {
  isOpen: boolean;
  onClose: () => void;
  folders: RSSFolder[];
  onFeedAdded: (feed: RSSFeed) => void;
}

export function RssAddFeedDialog({
  isOpen,
  onClose,
  folders,
  onFeedAdded,
}: RssAddFeedDialogProps) {
  const [url, setUrl] = useState('');
  const [name, setName] = useState('');
  const [folder, setFolder] = useState('');
  const [autoTags, setAutoTags] = useState('');
  const [priority, setPriority] = useState(false);
  const [loading, setLoading] = useState(false);
  const [discovering, setDiscovering] = useState(false);
  const [error, setError] = useState<string>('');

  const handleSubmit = async (e: React.FormEvent) => {
    e.preventDefault();
    if (!url.trim()) {
      setError('Feed URL is required');
      return;
    }

    setLoading(true);
    setError('');

    try {
      const feedParams: CreateRSSFeedParams = {
        url: url.trim(),
      };

      if (name.trim()) {
        feedParams.name = name.trim();
      }
      if (folder.trim()) {
        feedParams.folder = folder.trim();
      }
      if (autoTags.trim()) {
        feedParams.auto_tags = autoTags.trim();
      }
      if (priority) {
        feedParams.priority = priority;
      }

      const newFeed = await createFeed(feedParams);
      onFeedAdded(newFeed);
      handleClose();
    } catch (err) {
      console.error('Failed to create feed:', err);
      setError(
        err instanceof Error
          ? err.message
          : 'Failed to add feed. Please check the URL and try again.',
      );
    } finally {
      setLoading(false);
    }
  };

  const handleClose = () => {
    setUrl('');
    setName('');
    setFolder('');
    setAutoTags('');
    setPriority(false);
    setError('');
    onClose();
  };

  const handleDiscover = async () => {
    const trimmedUrl = url.trim();
    if (!trimmedUrl) {
      setError('Please enter a URL to discover the feed');
      return;
    }

    setDiscovering(true);
    setError('');

    try {
      const discovered: DiscoverFeedResponse = await discoverFeed(trimmedUrl);
      setUrl(discovered.feed_url);
      setName(discovered.title);
    } catch (err) {
      console.error('Failed to discover feed:', err);
      const errorMessage =
        err instanceof Error
          ? err.message
          : 'Failed to discover feed. Please check the URL and try again.';
      setError(errorMessage);
    } finally {
      setDiscovering(false);
    }
  };

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
                  Add RSS Feed
                </Dialog.Title>

                <form onSubmit={handleSubmit} className="space-y-4">
                  {/* Feed URL - Required */}
                  <div>
                    <label
                      htmlFor="feed-url"
                      className="block text-sm font-medium text-gray-700 mb-1"
                    >
                      Feed URL <span className="text-red-500">*</span>
                    </label>
                    <div className="flex gap-2">
                      <input
                        id="feed-url"
                        type="url"
                        value={url}
                        onChange={(e) => setUrl(e.target.value)}
                        placeholder="https://example.com/feed"
                        className="flex-1 px-3 py-2 border border-gray-300 rounded-md shadow-sm focus:outline-none focus:ring-2 focus:ring-blue-500 focus:border-blue-500 sm:text-sm"
                        required
                        autoFocus
                        disabled={loading || discovering}
                      />
                      <button
                        type="button"
                        onClick={handleDiscover}
                        disabled={!url.trim() || loading || discovering}
                        className="px-3 py-2 bg-blue-600 text-white hover:bg-blue-700 rounded-md font-medium transition-colors disabled:opacity-50 disabled:cursor-not-allowed flex items-center gap-2 whitespace-nowrap"
                      >
                        {discovering ? (
                          <>
                            <Spinner size="sm" className="text-white" />
                            Discovering...
                          </>
                        ) : (
                          <>
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
                                d="M21 21l-6-6m2-5a7 7 0 11-14 0 7 7 0 0114 0z"
                              />
                            </svg>
                            Discover Feed
                          </>
                        )}
                      </button>
                    </div>
                  </div>

                  {/* Name - Optional */}
                  <div>
                    <label
                      htmlFor="feed-name"
                      className="block text-sm font-medium text-gray-700 mb-1"
                    >
                      Name{' '}
                      <span className="text-gray-400">
                        (optional - auto-detected from feed if blank)
                      </span>
                    </label>
                    <input
                      id="feed-name"
                      type="text"
                      value={name}
                      onChange={(e) => setName(e.target.value)}
                      placeholder="My Awesome Blog"
                      className="w-full px-3 py-2 border border-gray-300 rounded-md shadow-sm focus:outline-none focus:ring-2 focus:ring-blue-500 focus:border-blue-500 sm:text-sm"
                    />
                  </div>

                  {/* Folder - Optional */}
                  <div>
                    <label
                      htmlFor="feed-folder"
                      className="block text-sm font-medium text-gray-700 mb-1"
                    >
                      Folder <span className="text-gray-400">(optional)</span>
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

                  {/* Auto Tags - Optional */}
                  <div>
                    <label
                      htmlFor="feed-tags"
                      className="block text-sm font-medium text-gray-700 mb-1"
                    >
                      Auto Tags{' '}
                      <span className="text-gray-400">(optional)</span>
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

                  {/* Priority - Optional */}
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
                      disabled={loading || discovering}
                      className="px-4 py-2 min-h-[44px] text-gray-700 bg-gray-200 hover:bg-gray-300 rounded-md font-medium transition-colors disabled:opacity-50 disabled:cursor-not-allowed"
                    >
                      Cancel
                    </button>
                    <button
                      type="submit"
                      disabled={loading || discovering || !url.trim()}
                      className="px-4 py-2 min-h-[44px] bg-blue-600 text-white hover:bg-blue-700 rounded-md font-medium transition-colors disabled:opacity-50 disabled:cursor-not-allowed flex items-center gap-2"
                    >
                      {loading ? (
                        <>
                          <Spinner size="sm" className="text-white" />
                          Adding...
                        </>
                      ) : (
                        <>
                          <svg
                            className="w-4 h-4"
                            fill="currentColor"
                            viewBox="0 0 20 20"
                          >
                            <path
                              fillRule="evenodd"
                              d="M10 3a1 1 0 011 1v5h5a1 1 0 110 2h-5v5a1 1 0 11-2 0v-5H4a1 1 0 110-2h5V4a1 1 0 011-1z"
                              clipRule="evenodd"
                            />
                          </svg>
                          Add Feed
                        </>
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
