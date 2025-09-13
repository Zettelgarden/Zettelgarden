import React, { useState } from "react";
import { useTaskContext } from "../../contexts/TaskContext";
import { SearchTagDropdown } from "../../components/tags/SearchTagDropdown";
import { Tag } from "../../models/Tags";
import { BulkTaskDateDisplay } from "./BulkTaskDateDisplay";
import { Task } from "../../models/Task";
import { Menu } from "@headlessui/react";

interface TaskPageOptionsMenu {
  tags: Tag[];
  handleTagClick: (tag: string) => void;
  tasks: Task[];
}

export function TaskPageOptionsMenu({
  tags,
  handleTagClick,
  tasks,
}: TaskPageOptionsMenu) {
  const { setRefreshTasks } = useTaskContext();
  const [showTagMenu, setShowTagMenu] = useState<boolean>(false);
  const [showBulkEdit, setShowBulkEdit] = useState<boolean>(false);

  function toggleTagMenu() {
    setShowTagMenu(true);
    setShowBulkEdit(false);
  }

  function toggleBulkEdit() {
    setShowTagMenu(false);
    setShowBulkEdit(true);
  }

  return (
    <div className="relative">
      <Menu as="div" className="relative inline-block text-left">
        <div>
          <Menu.Button className="font-semibold rounded focus:outline-none focus:ring-2 focus:ring-offset-2 bg-palette-dark text-white hover:bg-palette-darkest focus:ring-blue-500 px-4 py-2">
            Actions
          </Menu.Button>
        </div>
        <Menu.Items className="origin-top-right absolute right-0 mt-2 w-40 rounded-md shadow-lg bg-white ring-1 ring-black ring-opacity-5 focus:outline-none z-10">
          <div className="py-1">
            <Menu.Item>
              {({ active }) => (
                <button
                  onClick={toggleTagMenu}
                  className={`${
                    active ? 'bg-gray-100 text-gray-900' : 'text-gray-700'
                  } group flex rounded-md items-center w-full px-2 py-2 text-sm`}
                >
                  Add Tags
                </button>
              )}
            </Menu.Item>
            <Menu.Item>
              {({ active }) => (
                <button
                  onClick={toggleBulkEdit}
                  className={`${
                    active ? 'bg-gray-100 text-gray-900' : 'text-gray-700'
                  } group flex rounded-md items-center w-full px-2 py-2 text-sm`}
                >
                  Bulk Edit Date
                </button>
              )}
            </Menu.Item>
          </div>
        </Menu.Items>
      </Menu>

      {showTagMenu && (
        <SearchTagDropdown
          tags={tags}
          handleTagClick={handleTagClick}
          setShowTagMenu={setShowTagMenu}
        />
      )}
      {showBulkEdit && (
        <BulkTaskDateDisplay tasks={tasks} setShowBulkEdit={setShowBulkEdit} />
      )}
    </div>
  );
}
