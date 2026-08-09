import React from 'react';
import { Menu, MenuItem, MenuRawItem } from '../ui/Menu';
import { SearchTagDropdown } from '../tags/SearchTagDropdown';
import { Tag } from '../../models/Tags';

interface CardListMenuProps {
  cardId: number;
  onEditClick: () => void;
  onAddTag: (tagName: string) => void;
  onRecategoryClick?: () => void;
  showRecategory?: boolean;
  tags?: Tag[];
  isStarred?: boolean;
  onToggleStar?: () => void;
}

export function CardListMenu({
  cardId,
  onEditClick,
  onAddTag,
  onRecategoryClick,
  showRecategory = false,
  tags = [],
  isStarred = false,
  onToggleStar,
}: CardListMenuProps) {
  return (
    <Menu
      panelClassName="w-32"
      button={
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
      }
    >
      <MenuItem onClick={onEditClick}>
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
      </MenuItem>

      {onToggleStar && (
        <MenuItem onClick={onToggleStar}>
          <svg
            className="w-4 h-4 mr-2"
            fill={isStarred ? 'currentColor' : 'none'}
            stroke="currentColor"
            viewBox="0 0 24 24"
          >
            <path
              strokeLinecap="round"
              strokeLinejoin="round"
              strokeWidth={2}
              d="M11.049 2.927c.3-.921 1.603-.921 1.902 0l1.519 4.674a1 1 0 00.95.69h4.915c.969 0 1.371 1.24.588 1.81l-3.976 2.888a1 1 0 00-.363 1.118l1.518 4.674c.3.922-.755 1.688-1.538 1.118l-3.976-2.888a1 1 0 00-1.176 0l-3.976 2.888c-.783.57-1.838-.197-1.538-1.118l1.518-4.674a1 1 0 00-.363-1.118l-3.976-2.888c-.784-.57-.38-1.81.588-1.81h4.914a1 1 0 00.951-.69l1.519-4.674z"
            />
          </svg>
          {isStarred ? 'Unstar' : 'Star'}
        </MenuItem>
      )}

      <MenuRawItem>
        {({ active }) => (
          <div
            className={`${
              active ? 'bg-gray-100' : ''
            } flex w-full items-center px-4 py-3 min-h-[44px] text-sm text-gray-700 hover:bg-gray-100 border-t border-gray-100`}
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
            <span className="mr-2">Add Tag</span>
            <SearchTagDropdown tags={tags} handleTagClick={onAddTag} />
          </div>
        )}
      </MenuRawItem>

      {showRecategory && onRecategoryClick && (
        <MenuItem onClick={onRecategoryClick}>
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
        </MenuItem>
      )}
    </Menu>
  );
}
