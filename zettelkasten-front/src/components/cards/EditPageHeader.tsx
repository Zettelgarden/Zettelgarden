import React from 'react';
import { Menu } from '@headlessui/react';
import { Card } from '../../models/Card';
import { deleteCard } from '../../api/cards';
import { Button } from '../ui/Button';
import { Spinner } from '../ui/Spinner';
import { useUIState } from '../../contexts/UIStateContext';
import {
  useCardEditorContext,
  useEditorUIContext,
  useEditorMessagesContext,
} from '../../contexts/editor';

interface EditPageHeaderProps {
  newCard: boolean;
  originalCard: Card;
  suggestingTitle: boolean;
  handleSuggestTitle: () => void;
  handleSaveCard: () => void;
  handleCancelButtonClick: () => void;
  onDeleteSuccess: () => void;
}

/**
 * Obsidian-style header for the edit page, mirroring `ViewPageHeader`'s shape:
 * breadcrumb `[card_id]` + large editable title + quiet action row + overflow menu.
 * Save/Cancel live here (always reachable when the body is long) alongside the
 * desktop-only rail toggle, which sets up the closable rail in a later PR.
 */
export function EditPageHeader({
  newCard,
  originalCard,
  suggestingTitle,
  handleSuggestTitle,
  handleSaveCard,
  handleCancelButtonClick,
  onDeleteSuccess,
}: EditPageHeaderProps) {
  const { editingCard, setEditingCard } = useCardEditorContext();
  const { setShowSaveAsTemplate } = useEditorUIContext();
  const { setMessage } = useEditorMessagesContext();
  const { rightPaneOpen, toggleRightPane } = useUIState();

  // Breadcrumb reads the proposed id for new cards, the persisted id otherwise.
  const breadcrumbId = newCard
    ? editingCard.card_id || 'new'
    : originalCard.card_id;

  return (
    <header className="pb-2">
      <div className="flex items-start justify-between gap-3">
        {/* Title block: breadcrumb + editable title */}
        <div className="min-w-0 flex-1">
          <div className="flex items-center gap-1.5 text-xs text-gray-400 mb-1">
            <span className="font-mono">[{breadcrumbId}]</span>
          </div>
          <div className="relative">
            <input
              type="text"
              aria-label="Title"
              value={editingCard.title}
              onChange={(e) =>
                setEditingCard({ ...editingCard, title: e.target.value })
              }
              placeholder="Untitled"
              className="block w-full text-2xl font-semibold text-gray-900 bg-transparent border border-transparent rounded-md px-2 py-1 -mx-2 focus:outline-none focus:ring-2 focus:ring-blue-500 pr-10"
            />
            <Button
              onClick={handleSuggestTitle}
              disabled={suggestingTitle || !editingCard.body.trim()}
              type="button"
              size="small"
              className="absolute right-2 top-1/2 -translate-y-1/2 !p-1 text-palette-dark hover:text-palette-darkest hover:bg-palette-lighest disabled:text-gray-400 disabled:cursor-not-allowed disabled:hover:bg-transparent"
              title={
                suggestingTitle
                  ? 'Suggesting title...'
                  : 'Suggest title from content'
              }
            >
              {suggestingTitle ? (
                <Spinner size="md" />
              ) : (
                <svg
                  aria-hidden="true"
                  xmlns="http://www.w3.org/2000/svg"
                  className="h-5 w-5"
                  viewBox="0 0 20 20"
                  fill="currentColor"
                >
                  <path d="M13.586 3.586a2 2 0 112.828 2.828l-.793.793-2.828-2.828.793-.793zM11.379 5.793L3 14.172V17h2.828l8.38-8.379-2.83-2.828z" />
                </svg>
              )}
            </Button>
          </div>
        </div>

        {/* Actions */}
        <div className="flex items-center gap-1 shrink-0">
          <Button onClick={handleSaveCard} variant="primary" size="small">
            Save
          </Button>
          <Button
            onClick={handleCancelButtonClick}
            variant="outline"
            size="small"
          >
            Cancel
          </Button>
          <Button
            type="button"
            onClick={toggleRightPane}
            title="Toggle info pane"
            aria-pressed={rightPaneOpen}
            size="small"
            className="hidden md:inline-flex text-gray-400 hover:text-gray-700 hover:bg-gray-100"
          >
            <svg
              aria-hidden="true"
              className="h-5 w-5"
              fill="none"
              viewBox="0 0 24 24"
              stroke="currentColor"
            >
              <path
                strokeLinecap="round"
                strokeLinejoin="round"
                strokeWidth={2}
                d="M5 4h14a1 1 0 011 1v14a1 1 0 01-1 1H5a1 1 0 01-1-1V5a1 1 0 011-1z"
              />
              <path
                strokeLinecap="round"
                strokeLinejoin="round"
                strokeWidth={2}
                d="M15 5v14"
              />
            </svg>
          </Button>
          <Menu as="div" className="relative inline-block">
            <Menu.Button
              className="p-2 rounded-md text-gray-400 hover:text-gray-700 hover:bg-gray-100 transition-colors"
              title="More actions"
            >
              <svg
                className="h-5 w-5"
                fill="none"
                viewBox="0 0 24 24"
                stroke="currentColor"
              >
                <path
                  strokeLinecap="round"
                  strokeLinejoin="round"
                  strokeWidth={2}
                  d="M12 5v.01M12 12v.01M12 19v.01M12 6a1 1 0 110-2 1 1 0 010 2zm0 7a1 1 0 110-2 1 1 0 010 2zm0 7a1 1 0 110-2 1 1 0 010 2z"
                />
              </svg>
            </Menu.Button>
            <Menu.Items className="absolute right-0 mt-2 w-56 rounded-md shadow-lg bg-white ring-1 ring-black ring-opacity-5 focus:outline-none z-10">
              <div className="py-1">
                {!newCard && (
                  <Menu.Item>
                    <div className="flex items-center gap-2 p-2">
                      <input
                        type="checkbox"
                        id="process_entities_and_facts"
                        checked={
                          editingCard.process_entities_and_facts || false
                        }
                        onChange={(e) =>
                          setEditingCard({
                            ...editingCard,
                            process_entities_and_facts: e.target.checked,
                          })
                        }
                        className="rounded border-gray-300 text-blue-600 focus:ring-blue-500"
                      />
                      <label
                        htmlFor="process_entities_and_facts"
                        className="text-sm text-gray-700"
                      >
                        Process Entities & Facts
                      </label>
                    </div>
                  </Menu.Item>
                )}
                <Menu.Item>
                  {({ active }) => (
                    <button
                      onClick={() => setShowSaveAsTemplate(true)}
                      className={`${
                        active ? 'bg-gray-100 text-gray-900' : 'text-gray-700'
                      } group flex items-center w-full px-4 py-2 text-sm`}
                    >
                      Save as Template
                    </button>
                  )}
                </Menu.Item>
                {!newCard && (
                  <Menu.Item>
                    {({ active }) => (
                      <button
                        onClick={() => {
                          if (
                            window.confirm(
                              'Are you sure you want to delete this card? This cannot be reversed',
                            )
                          ) {
                            deleteCard(editingCard.id)
                              .then(() => {
                                setMessage('Card deleted successfully');
                                onDeleteSuccess();
                              })
                              .catch(() =>
                                setMessage(
                                  'Unable to delete card. Does it have backlinks, children or files?',
                                ),
                              );
                          }
                        }}
                        className={`${
                          active ? 'bg-red-50' : ''
                        } text-red-700 group flex items-center w-full px-4 py-2 text-sm hover:bg-red-50`}
                      >
                        Delete Card
                      </button>
                    )}
                  </Menu.Item>
                )}
              </div>
            </Menu.Items>
          </Menu>
        </div>
      </div>
    </header>
  );
}
