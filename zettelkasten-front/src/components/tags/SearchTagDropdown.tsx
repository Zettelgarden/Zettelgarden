import React, { useState, ChangeEvent } from 'react';
import { Tag } from '../../models/Tags';
import { Menu, MenuItem } from '../ui/Menu';

interface SearchTagDropdownProps {
  tags: Tag[];
  handleTagClick: (tag: string) => void;
}

export function SearchTagDropdown({
  tags,
  handleTagClick,
}: SearchTagDropdownProps) {
  const [textInput, setTextInput] = useState<string>('');

  function handleTagClickHook(tag: Tag) {
    handleTagClick(tag.name);
    setTextInput('');
  }

  function handleInput(
    e: ChangeEvent<HTMLInputElement | HTMLTextAreaElement | HTMLSelectElement>,
  ) {
    setTextInput(e.target.value);
  }

  function handleEnter() {
    if (textInput.trim()) {
      handleTagClick(textInput.trim());
      setTextInput('');
    }
  }

  return (
    <Menu
      panelClassName="w-40 sm:w-48 z-50"
      buttonClassName="!text-blue-500 hover:!text-blue-700 hover:!bg-blue-50 !p-2"
      button={
        <svg
          xmlns="http://www.w3.org/2000/svg"
          className="h-5 w-5"
          viewBox="0 0 20 20"
          fill="currentColor"
        >
          <path
            fillRule="evenodd"
            d="M10 3a1 1 0 011 1v5h5a1 1 0 110 2h-5v5a1 1 0 11-2 0v-5H4a1 1 0 110-2h5V4a1 1 0 011-1z"
            clipRule="evenodd"
          />
        </svg>
      }
    >
      <div className="p-2">
        <input
          type="text"
          value={textInput}
          placeholder="Tag name"
          onChange={handleInput}
          onKeyPress={(event: React.KeyboardEvent<HTMLInputElement>) => {
            if (event.key === 'Enter') {
              handleEnter();
            }
          }}
          className="w-full px-2 py-1 border border-gray-300 rounded-md text-sm mb-2 focus:outline-none focus:ring-2 focus:ring-indigo-500"
          autoFocus
        />
        <div className="max-h-32 overflow-y-auto">
          {tags &&
            tags
              .filter((tag) =>
                tag.name.toLowerCase().includes(textInput.toLowerCase()),
              )
              .map((tag) => (
                <MenuItem key={tag.id} onClick={() => handleTagClickHook(tag)}>
                  #{tag.name}
                </MenuItem>
              ))}
          {textInput.trim() && (
            <MenuItem
              onClick={handleEnter}
              className="!text-blue-600 hover:!bg-blue-100 !mt-1 !border-t !border-gray-100"
            >
              + Create "#{textInput.trim()}"
            </MenuItem>
          )}
        </div>
      </div>
    </Menu>
  );
}
