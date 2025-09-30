import React from "react";
import { Menu } from "@headlessui/react";
import { SearchTagDropdown } from "../tags/SearchTagDropdown";
import { Tag } from "../../models/Tags";

interface CardListMenuProps {
  cardId: number;
  onEditClick: () => void;
  onAddTag: (tagName: string) => void;
  onRecategoryClick?: () => void;
  showRecategory?: boolean;
  tags?: Tag[];
}

export function CardListMenu({
  cardId,
  onEditClick,
  onAddTag,
  onRecategoryClick,
  showRecategory = false,
  tags = [],
}: CardListMenuProps) {
  return (
    <Menu as="div" className="relative flex-shrink-0 w-6">
      <Menu.Button className="rounded hover:bg-gray-100 transition-colors">
        <svg
          className="w-4 h-4 text-gray-500"
          fill="none"
          stroke="currentColor"
          viewBox="0 0 24 24"
        >
          <path
            strokeLinecap="round"
            strokeLinejoin="round"
            strokeWidth={2}
            d="M12 5v.01M12 12v.01M12 19v.01M12 6a1 1 0 110-2 1 1 0 010 2zm0 7a1 1 0 110-2 1 1 0 010 2zm0 7a1 1 0 110-2 1 1 0 010 2z"
          />
        </svg>
      </Menu.Button>

      <Menu.Items className="absolute right-0 z-10 mt-1 w-32 origin-top-right bg-white border border-gray-200 rounded-md shadow-lg focus:outline-none">
        <div className="py-1">
          <Menu.Item>
            {({ active }) => (
              <button
                onClick={onEditClick}
                className={`${active ? "bg-gray-100" : ""
                  } flex w-full items-center px-3 py-2 text-sm text-gray-700 hover:bg-gray-100`}
              >
                <svg
                  className="w-4 h-4 mr-2"
                  fill="none"
                  stroke="currentColor"
                  viewBox="0 0 24 24"
                >
                  <path
                    strokeLinecap="round"
                    strokeLinejoin="round"
                    strokeWidth={2}
                    d="M11 5H6a2 2 0 00-2 2v11a2 2 0 002 2h11a2 2 0 002-2v-5m-1.414-9.414a2 2 0 112.828 2.828L11.828 15H9v-2.828l8.586-8.586z"
                  />
                </svg>
                Edit
              </button>
            )}
          </Menu.Item>

          <Menu.Item>
            {({ active }) => (
              <div
                className={`${active ? "bg-gray-100" : ""
                  } flex w-full items-center px-3 py-2 text-sm text-gray-700 hover:bg-gray-100 border-t border-gray-100`}
              >
                <svg
                  className="w-4 h-4 mr-2"
                  fill="none"
                  stroke="currentColor"
                  viewBox="0 0 24 24"
                >
                  <path
                    strokeLinecap="round"
                    strokeLinejoin="round"
                    strokeWidth={2}
                    d="M7 7h.01M7 3h5c.512 0 1.024.195 1.414.586l7 7a2 2 0 010 2.828l-7 7a2 2 0 01-2.828 0l-7-7A1.994 1.994 0 713 12V7a4 4 0 014-4z"
                  />
                </svg>
                <span className="mr-2">Add Tag</span>
                <SearchTagDropdown tags={tags} handleTagClick={onAddTag} />
              </div>
            )}
          </Menu.Item>

          {showRecategory && onRecategoryClick && (
            <Menu.Item>
              {({ active }) => (
                <button
                  onClick={onRecategoryClick}
                  className={`${active ? "bg-gray-100" : ""
                    } flex w-full items-center px-3 py-2 text-sm text-gray-700 hover:bg-gray-100`}
                >
                  <svg
                    className="w-4 h-4 mr-2"
                    fill="none"
                    stroke="currentColor"
                    viewBox="0 0 24 24"
                  >
                    <path
                      strokeLinecap="round"
                      strokeLinejoin="round"
                      strokeWidth={2}
                      d="M7 7h.01M7 3h5c.512 0 1.024.195 1.414.586l7 7a2 2 0 010 2.828l-7 7a2 2 0 01-2.828 0l-7-7A1.994 1.994 0 013 12V7a4 4 0 014-4z"
                    />
                  </svg>
                  Recategorize
                </button>
              )}
            </Menu.Item>
          )}
        </div>
      </Menu.Items>
    </Menu>
  );
}