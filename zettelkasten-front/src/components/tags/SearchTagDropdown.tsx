import React, { useState, ChangeEvent } from "react";
import { Tag } from "../../models/Tags";
import { Menu } from "@headlessui/react";

interface SearchTagDropdownProps {
  tags: Tag[];
  handleTagClick: (tag: string) => void;
}

export function SearchTagDropdown({
  tags,
  handleTagClick,
}: SearchTagDropdownProps) {
  const [textInput, setTextInput] = useState<string>("");

  function handleTagClickHook(tag: Tag) {
    handleTagClick(tag.name);
    setTextInput("");
  }

  function handleInput(
    e: ChangeEvent<HTMLInputElement | HTMLTextAreaElement | HTMLSelectElement>,
  ) {
    setTextInput(e.target.value);
  }

  function handleEnter() {
    if (textInput.trim()) {
      handleTagClick(textInput.trim());
      setTextInput("");
    }
  }


  return (
    <Menu as="div" className="relative inline-block text-left">
      <Menu.Button className="text-blue-500 hover:text-blue-700 min-w-[44px] min-h-[44px] flex items-center justify-center p-2 rounded hover:bg-blue-50 transition-colors">
        <svg xmlns="http://www.w3.org/2000/svg" className="h-5 w-5" viewBox="0 0 20 20" fill="currentColor">
          <path fillRule="evenodd" d="M10 3a1 1 0 011 1v5h5a1 1 0 110 2h-5v5a1 1 0 11-2 0v-5H4a1 1 0 110-2h5V4a1 1 0 011-1z" clipRule="evenodd" />
        </svg>
      </Menu.Button>

      <Menu.Items className="absolute right-0 mt-2 w-40 sm:w-48 origin-top-right rounded-md shadow-lg bg-white ring-1 ring-black ring-opacity-5 focus:outline-none z-50 transform -translate-x-2 sm:translate-x-0">
        <div className="p-2">
          <input
            type="text"
            value={textInput}
            placeholder="Tag name"
            onChange={handleInput}
            onKeyPress={(event: React.KeyboardEvent<HTMLInputElement>) => {
              if (event.key === "Enter") {
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
                  <Menu.Item key={tag.id}>
                    {({ active }) => (
                      <button
                        onClick={() => handleTagClickHook(tag)}
                        className={`${
                          active ? 'bg-gray-100 text-gray-900' : 'text-gray-700'
                        } group flex rounded-md items-center w-full px-4 py-3 min-h-[44px] text-sm`}
                      >
                        #{tag.name}
                      </button>
                    )}
                  </Menu.Item>
                ))}
            {textInput.trim() && (
              <Menu.Item>
                {({ active }) => (
                  <button
                    onClick={handleEnter}
                    className={`${
                      active ? 'bg-blue-100 text-blue-900' : 'text-blue-600'
                    } group flex rounded-md items-center w-full px-4 py-3 min-h-[44px] text-sm border-t border-gray-100 mt-1 pt-3`}
                  >
                    + Create "#{textInput.trim()}"
                  </button>
                )}
              </Menu.Item>
            )}
          </div>
        </div>
      </Menu.Items>
    </Menu>
  );
}