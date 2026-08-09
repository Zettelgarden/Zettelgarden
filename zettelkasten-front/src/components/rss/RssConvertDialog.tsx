import React, { useState, useEffect } from 'react';
import { Dialog, Transition } from '@headlessui/react';
import { Fragment } from 'react';
import { Spinner } from '../ui/Spinner';
import {
  convertToCard,
  ConvertArticleParams,
  RSSArticle,
  ConvertCardResponse,
} from '../../api/rss';
import { safeHtmlToMarkdown } from '../../utils/markdown';
import { CardIdDiscoveryDialog } from '../cards/CardIdDiscoveryDialog';
import { SearchTagDropdown } from '../tags/SearchTagDropdown';
import { useTagContext } from '../../contexts/TagContext';

interface RssConvertDialogProps {
  isOpen: boolean;
  onClose: () => void;
  article: RSSArticle | null;
  onConverted: (cardId: number) => void;
}

export function RssConvertDialog({
  isOpen,
  onClose,
  article,
  onConverted,
}: RssConvertDialogProps) {
  const { tags: allTags } = useTagContext();
  const [title, setTitle] = useState('');
  const [body, setBody] = useState('');
  const [selectedTags, setSelectedTags] = useState<string[]>([]);
  const [cardId, setCardId] = useState('');
  const [showCardIdDiscovery, setShowCardIdDiscovery] = useState(false);
  const [loading, setLoading] = useState(false);
  const [error, setError] = useState<string>('');

  // Initialize title and body from article when it changes
  useEffect(() => {
    if (article) {
      setTitle(article.title);
      // Convert HTML content to markdown for editing
      if (article.content) {
        setBody(safeHtmlToMarkdown(article.content));
      } else {
        setBody('');
      }
      setSelectedTags([]);
      setCardId('');
    }
  }, [article]);

  const handleSubmit = async (e: React.FormEvent) => {
    e.preventDefault();
    if (!article) return;

    if (!title.trim()) {
      setError('Title is required');
      return;
    }

    setLoading(true);
    setError('');

    try {
      const params: ConvertArticleParams = {
        title: title.trim(),
      };

      if (body.trim()) {
        params.body = body.trim();
      }

      if (selectedTags.length > 0) {
        params.tags = selectedTags
          .map((t) => `#${t.replace(/^#/, '')}`)
          .join(' ');
      }

      if (cardId.trim()) {
        params.card_id = cardId.trim();
      }

      const result: ConvertCardResponse = await convertToCard(
        article.id,
        params,
      );

      if (result.id) {
        onConverted(result.id);
        handleClose();
      } else {
        setError('Failed to convert article to card');
      }
    } catch (err) {
      console.error('Failed to convert article:', err);
      setError(
        err instanceof Error
          ? err.message
          : 'Failed to convert article. Please try again.',
      );
    } finally {
      setLoading(false);
    }
  };

  const handleClose = () => {
    setTitle('');
    setBody('');
    setSelectedTags([]);
    setCardId('');
    setError('');
    onClose();
  };

  const handleCardIdSelected = (selectedCardId: string) => {
    setCardId(selectedCardId);
    setShowCardIdDiscovery(false);
  };

  const handleTagClick = (tagName: string) => {
    const cleanTag = tagName.replace(/^#/, '');
    if (!selectedTags.includes(cleanTag)) {
      setSelectedTags([...selectedTags, cleanTag]);
    }
  };

  const handleRemoveTag = (tagName: string) => {
    setSelectedTags(selectedTags.filter((t) => t !== tagName));
  };

  return (
    <>
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
                <Dialog.Panel className="w-full max-w-2xl transform overflow-hidden rounded-2xl bg-white p-6 text-left align-middle shadow-xl transition-all">
                  <Dialog.Title
                    as="h3"
                    className="text-lg font-medium leading-6 text-gray-900 mb-4"
                  >
                    Convert to Card
                  </Dialog.Title>

                  <form onSubmit={handleSubmit} className="space-y-4">
                    {/* Title - Required */}
                    <div>
                      <label
                        htmlFor="card-title"
                        className="block text-sm font-medium text-gray-700 mb-1"
                      >
                        Title <span className="text-red-500">*</span>
                      </label>
                      <input
                        id="card-title"
                        type="text"
                        value={title}
                        onChange={(e) => setTitle(e.target.value)}
                        placeholder="Article title"
                        className="w-full px-3 py-2 border border-gray-300 rounded-md shadow-sm focus:outline-none focus:ring-2 focus:ring-blue-500 focus:border-blue-500 sm:text-sm"
                        required
                        autoFocus
                      />
                    </div>

                    {/* Content - Optional, monospace font, 10 rows */}
                    <div>
                      <label
                        htmlFor="card-content"
                        className="block text-sm font-medium text-gray-700 mb-1"
                      >
                        Content{' '}
                        <span className="text-gray-400">(optional)</span>
                      </label>
                      <textarea
                        id="card-content"
                        value={body}
                        onChange={(e) => setBody(e.target.value)}
                        placeholder="Article content in markdown format..."
                        rows={10}
                        className="w-full px-3 py-2 border border-gray-300 rounded-md shadow-sm focus:outline-none focus:ring-2 focus:ring-blue-500 focus:border-blue-500 sm:text-sm font-mono text-sm resize-y"
                      />
                    </div>

                    {/* Tags - Optional */}
                    <div>
                      <div className="flex items-center justify-between mb-1">
                        <label className="block text-sm font-medium text-gray-700">
                          Tags <span className="text-gray-400">(optional)</span>
                        </label>
                        <SearchTagDropdown
                          tags={allTags}
                          handleTagClick={handleTagClick}
                        />
                      </div>
                      {selectedTags.length > 0 && (
                        <div className="flex flex-wrap gap-1.5 mt-2">
                          {selectedTags.map((tag) => (
                            <span
                              key={tag}
                              className="inline-flex items-center px-1.5 py-0.5 bg-purple-50 text-purple-600 text-xs rounded-full"
                            >
                              <span className="cursor-pointer hover:bg-purple-100">
                                #{tag}
                              </span>
                              <button
                                onClick={() => handleRemoveTag(tag)}
                                className="ml-1.5 text-purple-400 hover:text-purple-600"
                              >
                                &times;
                              </button>
                            </span>
                          ))}
                        </div>
                      )}
                      {selectedTags.length === 0 && (
                        <p className="mt-1 text-xs text-gray-500">
                          Select tags from dropdown to add to the card
                        </p>
                      )}
                    </div>

                    {/* Card ID - Optional */}
                    <div>
                      <label
                        htmlFor="card-id"
                        className="block text-sm font-medium text-gray-700 mb-1"
                      >
                        Card ID{' '}
                        <span className="text-gray-400">(optional)</span>
                      </label>
                      <div className="flex gap-2">
                        <input
                          id="card-id"
                          type="text"
                          value={cardId}
                          onChange={(e) => setCardId(e.target.value)}
                          className="flex-1 px-3 py-2 border border-gray-300 rounded-md shadow-sm focus:outline-none focus:ring-2 focus:ring-blue-500 focus:border-blue-500 sm:text-sm font-mono text-sm"
                        />
                        <button
                          type="button"
                          onClick={() => setShowCardIdDiscovery(true)}
                          className="px-4 py-2 bg-gray-100 hover:bg-gray-200 border border-gray-300 rounded-md text-sm font-medium transition-colors"
                          title="Discover card ID"
                        >
                          Discover
                        </button>
                      </div>
                    </div>

                    {/* Article Link */}
                    {article?.url && (
                      <div className="text-sm text-gray-500">
                        <span>Source: </span>
                        <a
                          href={article.url}
                          target="_blank"
                          rel="noopener noreferrer"
                          className="text-blue-600 hover:text-blue-800 hover:underline"
                        >
                          {article.url}
                        </a>
                      </div>
                    )}

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
                        disabled={loading || !title.trim()}
                        className="px-4 py-2 min-h-[44px] bg-blue-600 text-white hover:bg-blue-700 rounded-md font-medium transition-colors disabled:opacity-50 disabled:cursor-not-allowed flex items-center gap-2"
                      >
                        {loading ? (
                          <>
                            <Spinner size="sm" className="text-white" />
                            Converting...
                          </>
                        ) : (
                          <>
                            <svg
                              className="w-4 h-4"
                              fill="currentColor"
                              viewBox="0 0 20 20"
                            >
                              <path d="M7 3a1 1 0 000 2h6a1 1 0 100-2H7zM4 7a1 1 0 011-1h10a1 1 0 110 2H5a1 1 0 01-1-1zM2 11a2 2 0 012-2h12a2 2 0 012 2v4a2 2 0 01-2 2H4a2 2 0 01-2-2v-4z" />
                            </svg>
                            Convert to Card
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

      {/* Card ID Discovery Dialog */}
      {showCardIdDiscovery && (
        <CardIdDiscoveryDialog
          onClose={() => setShowCardIdDiscovery(false)}
          onSelectId={handleCardIdSelected}
        />
      )}
    </>
  );
}
