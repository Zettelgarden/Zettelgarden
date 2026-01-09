import React from "react";
import { SearchTagDropdown } from "../../components/tags/SearchTagDropdown";
import { Tag } from "../../models/Tags";
import { Menu } from "@headlessui/react";

interface TaskPageOptionsMenuProps {
  tags: Tag[];
  handleTagClick: (tag: string) => void;
  selectMode: boolean;
  onToggleSelectMode: () => void;
}

export function TaskPageOptionsMenu({
  tags,
  handleTagClick,
  selectMode,
  onToggleSelectMode,
}: TaskPageOptionsMenuProps) {
  return (
    <div className="relative">
      <Menu as="div" className="relative inline-block text-left">
        <div>
          <Menu.Button className="font-semibold rounded focus:outline-none focus:ring-2 focus:ring-offset-2 bg-slate-700 text-white hover:bg-slate-800 focus:ring-blue-500 px-4 py-2 text-sm h-9 flex items-center">
            Actions
          </Menu.Button>
        </div>
        <Menu.Items className="origin-top-right absolute right-0 mt-2 w-48 rounded-md shadow-lg bg-white ring-1 ring-black ring-opacity-5 focus:outline-none z-10 text-left">
          <div className="py-1">
            <Menu.Item>
              {({ active }) => (
                <button
                  onClick={onToggleSelectMode}
                  className={`${active ? 'bg-gray-100 text-gray-900' : 'text-gray-700'
                    } group flex rounded-md items-center w-full px-2 py-2 text-sm`}
                >
                  {selectMode ? "Exit Select Mode" : "Select Tasks"}
                </button>
              )}
            </Menu.Item>
          </div>
        </Menu.Items>
      </Menu>

      <div className="absolute top-full left-0 mt-2 flex gap-2">
        <SearchTagDropdown
          tags={tags}
          handleTagClick={handleTagClick}
        />
      </div>
    </div>
  );
}
