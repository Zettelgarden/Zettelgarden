import React from "react";
import { Menu } from "@headlessui/react";
import { Card } from "../../models/Card";
import { Button } from "../Button";
import { PinButton } from "./PinButton";

interface ViewPageHeaderProps {
  viewingCard: Card;
  isPinned: boolean;
  onTogglePin: () => void;
  onOpenChatSidebar: () => void;
  onEditCard: () => void;
  onToggleStar: () => void;
  toggleCreateTaskWindow: () => void;
  onResummarize: () => void;
  onRecategorize: () => void;
  showIdDiscovery: boolean;
  viewMode: 'normal' | 'tree';
  onToggleViewMode: () => void;
  onNavigateParent?: () => void;
}

export function ViewPageHeader({
  viewingCard,
  isPinned,
  onTogglePin,
  onOpenChatSidebar,
  onEditCard,
  onToggleStar,
  toggleCreateTaskWindow,
  onResummarize,
  onRecategorize,
  showIdDiscovery,
  viewMode,
  onToggleViewMode,
}: ViewPageHeaderProps) {
  return (
    <div className="flex flex-col md:flex-row items-start md:items-center justify-between bg-white rounded-lg p-3 shadow-sm">
      <div className="flex-grow">
        <div className="flex items-center flex-wrap md:flex-nowrap gap-2">
          <span className="font-bold text-gray-600">
            Viewing:
          </span>

          <span className="text-blue-600">
            [{viewingCard.card_id}]
          </span>
          <span className="text-gray-600 md:truncate">{" - "}
            {viewingCard.title}
          </span>
        </div>
      </div>
      <div className="mt-2 md:mt-0 flex justify-end gap-2 flex-shrink">
        <PinButton
          card={viewingCard}
          isPinned={isPinned}
          onTogglePin={onTogglePin}
        />
        <Button
          onClick={onOpenChatSidebar}
          className="bg-green-500 hover:bg-green-600 text-white"
        >
          💬 Chat
        </Button>
        <Button onClick={onEditCard}>Edit</Button>
        <Menu as="div" className="relative inline-block text-right">
          <div>
            <Menu.Button className="inline-flex justify-center w-full rounded-md border border-gray-300 shadow-sm px-2 py-2 bg-white text-sm font-medium text-gray-700 hover:bg-gray-50 focus:outline-none focus:ring-2 focus:ring-offset-2 focus:ring-offset-gray-100 focus:ring-indigo-500">
              <svg xmlns="http://www.w3.org/2000/svg" className="h-5" viewBox="0 0 20 20" fill="currentColor">
                <path d="M10 6a2 2 0 110-4 2 2 0 010 4zM10 12a2 2 0 110-4 2 2 0 010 4zM10 18a2 2 0 110-4 2 2 0 010 4z" />
              </svg>
            </Menu.Button>
          </div>
          <Menu.Items className="origin-top-left md:origin-top-right absolute right-0 md:right-0 left-0 md:left-auto mt-2 w-56 rounded-md shadow-lg bg-white ring-1 ring-black ring-opacity-5 focus:outline-none z-10">
            <div className="py-1">
              <Menu.Item>
                {({ active }) => (
                  <button
                    onClick={onToggleViewMode}
                    className={`${active ? 'bg-gray-100 text-gray-900' : 'text-gray-700'
                      } group flex rounded-md items-center w-full px-2 py-2 text-sm`}
                  >
                    {viewMode === 'tree' ? '📋 Switch to Normal View' : '📂 Switch to Tree View'}
                  </button>
                )}
              </Menu.Item>
              <Menu.Item>
                {({ active }) => (
                  <button
                    onClick={toggleCreateTaskWindow}
                    className={`${active ? 'bg-gray-100 text-gray-900' : 'text-gray-700'
                      } group flex rounded-md items-center w-full px-2 py-2 text-sm`}
                  >
                    Add Task
                  </button>
                )}
              </Menu.Item>
              <Menu.Item>
                {({ active }) => (
                  <button
                    onClick={onToggleStar}
                    className={`${active ? 'bg-gray-100 text-gray-900' : 'text-gray-700'
                      } group flex rounded-md items-center w-full px-2 py-2 text-sm`}
                  >
                    {viewingCard.is_starred ? 'Unstar Card' : 'Star Card'}
                  </button>
                )}
              </Menu.Item>
              <Menu.Item>
                {({ active }) => (
                  <button
                    onClick={onResummarize}
                    className={`${active ? 'bg-gray-100 text-gray-900' : 'text-gray-700'
                      } group flex rounded-md items-center w-full px-2 py-2 text-sm`}
                  >
                    Resummarize Card
                  </button>
                )}
              </Menu.Item>
              {viewingCard.card_id === "" && (
                <Menu.Item>
                  {({ active }) => (
                    <button
                      onClick={onRecategorize}
                      className={`${active ? 'bg-gray-100 text-gray-900' : 'text-gray-700'
                        } group flex rounded-md items-center w-full px-2 py-2 text-sm`}
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
  );
}