import React, { useState } from "react";
import { useTaskContext } from "../../contexts/TaskContext";
import { SearchTagDropdown } from "../../components/tags/SearchTagDropdown";
import { Tag } from "../../models/Tags";
import { BulkTaskDateDisplay } from "./BulkTaskDateDisplay";
import { BulkTaskTagEditor } from "./BulkTaskTagEditor";
import { Task } from "../../models/Task";
import { Menu } from "@headlessui/react";

interface TaskPageOptionsMenu {
  tags: Tag[];
  handleTagClick: (tag: string) => void;
  tasks: Task[];
  selectMode: boolean;
  selectedTaskIds: Set<number>;
  onSelectAll: () => void;
  onClearSelection: () => void;
  onToggleSelectMode: () => void;
}

export function TaskPageOptionsMenu({
  tags,
  handleTagClick,
  tasks,
  selectMode,
  selectedTaskIds,
  onSelectAll,
  onClearSelection,
  onToggleSelectMode,
}: TaskPageOptionsMenu) {
  const { setRefreshTasks } = useTaskContext();
  const [showBulkEdit, setShowBulkEdit] = useState<boolean>(false);
  const [showBulkTagEdit, setShowBulkTagEdit] = useState<boolean>(false);

  function toggleBulkEdit() {
    setShowBulkEdit(!showBulkEdit);
  }

  return (
    <div className="relative">
      <Menu as="div" className="relative inline-block text-left">
        <div>
          <Menu.Button className="font-semibold rounded focus:outline-none focus:ring-2 focus:ring-offset-2 bg-palette-dark text-white hover:bg-palette-darkest focus:ring-blue-500 px-4 py-2">
            Actions
          </Menu.Button>
        </div>
        <Menu.Items className="origin-top-right absolute right-0 mt-2 w-48 rounded-md shadow-lg bg-white ring-1 ring-black ring-opacity-5 focus:outline-none z-10">
          <div className="py-1">
            {selectMode ? (
              <>
                <Menu.Item>
                  {({ active }) => (
                    <button
                      onClick={() => setShowBulkTagEdit(true)}
                      disabled={selectedTaskIds.size === 0}
                      className={`${
                        active ? 'bg-gray-100 text-gray-900' : 'text-gray-700'
                      } group flex rounded-md items-center w-full px-2 py-2 text-sm disabled:opacity-50 disabled:cursor-not-allowed`}
                    >
                      Edit Tags
                    </button>
                  )}
                </Menu.Item>
                <Menu.Item>
                  {({ active }) => (
                    <button
                      onClick={() => setShowBulkEdit(true)}
                      disabled={selectedTaskIds.size === 0}
                      className={`${
                        active ? 'bg-gray-100 text-gray-900' : 'text-gray-700'
                      } group flex rounded-md items-center w-full px-2 py-2 text-sm disabled:opacity-50 disabled:cursor-not-allowed`}
                    >
                      Edit Date
                    </button>
                  )}
                </Menu.Item>
                <hr className="my-1" />
                <Menu.Item>
                  {({ active }) => (
                    <button
                      onClick={onSelectAll}
                      className={`${
                        active ? 'bg-gray-100 text-gray-900' : 'text-gray-700'
                      } group flex rounded-md items-center w-full px-2 py-2 text-sm`}
                    >
                      Select All
                    </button>
                  )}
                </Menu.Item>
                <Menu.Item>
                  {({ active }) => (
                    <button
                      onClick={onClearSelection}
                      className={`${
                        active ? 'bg-gray-100 text-gray-900' : 'text-gray-700'
                      } group flex rounded-md items-center w-full px-2 py-2 text-sm`}
                    >
                      Clear Selection
                    </button>
                  )}
                </Menu.Item>
                <hr className="my-1" />
                <Menu.Item>
                  {({ active }) => (
                    <button
                      onClick={onToggleSelectMode}
                      className={`${
                        active ? 'bg-gray-100 text-gray-900' : 'text-gray-700'
                      } group flex rounded-md items-center w-full px-2 py-2 text-sm`}
                    >
                      Exit Select Mode
                    </button>
                  )}
                </Menu.Item>
              </>
            ) : (
              <Menu.Item>
                {({ active }) => (
                  <button
                    onClick={onToggleSelectMode}
                    className={`${
                      active ? 'bg-gray-100 text-gray-900' : 'text-gray-700'
                    } group flex rounded-md items-center w-full px-2 py-2 text-sm`}
                  >
                    Enter Select Mode
                  </button>
                )}
              </Menu.Item>
            )}
          </div>
        </Menu.Items>
      </Menu>

      <div className="absolute top-full left-0 mt-2 flex gap-2">
        <SearchTagDropdown
          tags={tags}
          handleTagClick={handleTagClick}
        />
      </div>
      {showBulkEdit && (
        <BulkTaskDateDisplay
          tasks={tasks.filter((t) => selectedTaskIds.has(t.id))}
          setShowBulkEdit={setShowBulkEdit}
        />
      )}
      {showBulkTagEdit && (
        <BulkTaskTagEditor
          tasks={tasks.filter((t) => selectedTaskIds.has(t.id))}
          setShowBulkTagEdit={setShowBulkTagEdit}
        />
      )}
    </div>
  );
}
