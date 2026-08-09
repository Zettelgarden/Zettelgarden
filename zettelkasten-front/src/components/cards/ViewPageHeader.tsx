import React from 'react';
import { Menu } from '@headlessui/react';
import { Card } from '../../models/Card';
import { Button } from '../ui/Button';
import { StarIcon } from '../../assets/icons/StarIcon';
import { useUIState } from '../../contexts/UIStateContext';
import { ViewMode } from '../../pages/cards/ViewPageContainer';

const VIEW_MODES: ViewMode[] = ['normal', 'summary'];

interface ViewPageHeaderProps {
  viewingCard: Card;
  onEditCard: () => void;
  onToggleStar: () => void;
  toggleCreateTaskWindow: () => void;
  onResummarize: () => void;
  onRecategorize: () => void;
  viewMode: ViewMode;
  onViewModeChange: (mode: ViewMode) => void;
  onNavigateParent?: () => void;
  onNavigatePrev?: () => void;
  onNavigateNext?: () => void;
  onCreateChildCard: () => void;
}

export function ViewPageHeader({
  viewingCard,
  onEditCard,
  onToggleStar,
  toggleCreateTaskWindow,
  onResummarize,
  onRecategorize,
  viewMode,
  onViewModeChange,
  onNavigateParent,
  onNavigatePrev,
  onNavigateNext,
  onCreateChildCard,
}: ViewPageHeaderProps) {
  const { rightPaneOpen, toggleRightPane, setRightPaneOpen, setRightPaneTab } =
    useUIState();
  const hasParent = !!(onNavigateParent && viewingCard.parent);
  // Lateral/up tree navigation cluster — shown only when there's somewhere
  // to go. Promoted out of the buried Links-tab Parent section so navigating
  // between siblings is always reachable at the top of the page.
  const showSiblingNav = hasParent || !!onNavigatePrev || !!onNavigateNext;

  // Opens the rail's Links tab — used by ＋ Child (Children + Linked
  // references + BacklinkInput live there).
  const openLinksTab = () => {
    setRightPaneOpen(true);
    setRightPaneTab('links');
  };

  return (
    <header className="pb-2">
      <div className="flex items-start justify-between gap-3">
        {/* Title block: breadcrumb + large title */}
        <div className="min-w-0 flex-1">
          <div className="flex items-center gap-1.5 text-xs text-gray-400 mb-1">
            {hasParent && (
              <>
                <button
                  type="button"
                  onClick={onNavigateParent}
                  className="font-mono hover:text-palette-dark transition-colors"
                  title={
                    viewingCard.parent?.title ||
                    `Go to parent [${viewingCard.parent?.card_id}]`
                  }
                >
                  [{viewingCard.parent!.card_id}]
                </button>
                <svg
                  className="h-3 w-3 shrink-0"
                  fill="none"
                  stroke="currentColor"
                  viewBox="0 0 24 24"
                >
                  <path
                    strokeLinecap="round"
                    strokeLinejoin="round"
                    strokeWidth={2}
                    d="M9 5l7 7-7 7"
                  />
                </svg>
              </>
            )}
            <span className="font-mono">[{viewingCard.card_id}]</span>
          </div>
          <h1 className="text-2xl font-semibold text-gray-900 truncate">
            {viewingCard.title || 'Untitled'}
          </h1>
        </div>

        {/* Actions */}
        <div className="flex items-center gap-1 shrink-0">
          <Button
            type="button"
            onClick={onToggleStar}
            title={viewingCard.is_starred ? 'Unstar card' : 'Star card'}
            size="small"
            className={`${
              viewingCard.is_starred
                ? 'text-yellow-500 hover:bg-yellow-50'
                : 'text-gray-400 hover:text-gray-700 hover:bg-gray-100'
            }`}
          >
            <StarIcon className="h-5 w-5" filled={!!viewingCard.is_starred} />
          </Button>
          <Button
            type="button"
            onClick={toggleRightPane}
            title="Toggle info pane"
            aria-pressed={rightPaneOpen}
            size="small"
            className={`hidden md:inline-flex ${
              rightPaneOpen
                ? 'text-gray-700 hover:bg-gray-100'
                : 'text-gray-400 hover:text-gray-700 hover:bg-gray-100'
            }`}
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
          <Button onClick={onEditCard} variant="outline" size="small">
            Edit
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
            <Menu.Items className="absolute right-0 mt-2 w-48 rounded-md shadow-lg bg-white ring-1 ring-black ring-opacity-5 focus:outline-none z-10">
              <div className="py-1">
                <Menu.Item>
                  {({ active }) => (
                    <button
                      onClick={toggleCreateTaskWindow}
                      className={`${
                        active ? 'bg-gray-100 text-gray-900' : 'text-gray-700'
                      } group flex items-center w-full px-4 py-2 text-sm`}
                    >
                      Add Task
                    </button>
                  )}
                </Menu.Item>
                <Menu.Item>
                  {({ active }) => (
                    <button
                      onClick={onResummarize}
                      className={`${
                        active ? 'bg-gray-100 text-gray-900' : 'text-gray-700'
                      } group flex items-center w-full px-4 py-2 text-sm`}
                    >
                      Resummarize
                    </button>
                  )}
                </Menu.Item>
                {viewingCard.card_id === '' && (
                  <Menu.Item>
                    {({ active }) => (
                      <button
                        onClick={onRecategorize}
                        className={`${
                          active ? 'bg-gray-100 text-gray-900' : 'text-gray-700'
                        } group flex items-center w-full px-4 py-2 text-sm`}
                      >
                        Recategorize
                      </button>
                    )}
                  </Menu.Item>
                )}
              </div>
            </Menu.Items>
          </Menu>
        </div>
      </div>

      {/* Sibling/parent navigation + view mode + card creation actions */}
      <div className="mt-4 flex items-center justify-between gap-3">
        {showSiblingNav ? (
          <div className="flex items-center gap-1 text-sm">
            {onNavigatePrev && (
              <Button
                type="button"
                onClick={onNavigatePrev}
                title="Previous sibling"
                size="small"
                className="text-gray-500 hover:text-gray-900 hover:bg-gray-100"
              >
                ‹ Prev
              </Button>
            )}
            {hasParent && (
              <Button
                type="button"
                onClick={onNavigateParent}
                title="Go to parent"
                size="small"
                className="text-gray-500 hover:text-gray-900 hover:bg-gray-100"
              >
                ↑ Up
              </Button>
            )}
            {onNavigateNext && (
              <Button
                type="button"
                onClick={onNavigateNext}
                title="Next sibling"
                size="small"
                className="text-gray-500 hover:text-gray-900 hover:bg-gray-100"
              >
                Next ›
              </Button>
            )}
          </div>
        ) : (
          <div />
        )}
        <div className="inline-flex items-center gap-1 bg-gray-100 rounded-lg p-1">
          {VIEW_MODES.map((mode) => (
            <button
              key={mode}
              type="button"
              onClick={() => onViewModeChange(mode)}
              className={`px-3 py-1.5 text-sm rounded-md transition-colors capitalize ${
                viewMode === mode
                  ? 'bg-white shadow-sm text-gray-900 font-medium'
                  : 'text-gray-500 hover:text-gray-700'
              }`}
            >
              {mode}
            </button>
          ))}
        </div>
        <div className="flex items-center gap-1">
          <Button
            type="button"
            onClick={() => {
              openLinksTab();
              onCreateChildCard();
            }}
            title="Create a child card"
            size="small"
            className="!px-2.5 !py-1 !min-h-0 md:!min-h-0 text-gray-500 hover:text-palette-dark rounded-md"
          >
            ＋ Child
          </Button>
        </div>
      </div>
    </header>
  );
}
