import React from "react";
import { Task } from "../../models/Task";

import { saveExistingTask } from "../../api/tasks";

import { useTaskContext } from "../../contexts/TaskContext";
import { Menu } from "@headlessui/react";

interface TaskListOptionsMenuProps {
  task: Task;
  showCardLink: boolean;
  setShowCardLink: (show: boolean) => void;
}

export function TaskListOptionsMenu({
  task,
  showCardLink,
  setShowCardLink,
}: TaskListOptionsMenuProps) {
  const { setRefreshTasks } = useTaskContext();

  function toggleCardLink() {
    setShowCardLink(!showCardLink);
  }

  async function handleCardUnlink() {
    let editedTask = { ...task, card_pk: 0 };
    let response = await saveExistingTask(editedTask);
    if (!("error" in response)) {
      setRefreshTasks(true);
    }
  }

  return (
    <Menu as="div" className="relative inline-block text-left">
      <div>
        <Menu.Button className="inline-flex justify-center w-full rounded-md border border-gray-300 shadow-sm px-2 py-1 bg-white text-sm font-medium text-gray-700 hover:bg-gray-50 focus:outline-none focus:ring-2 focus:ring-offset-2 focus:ring-offset-gray-100 focus:ring-indigo-500">
          ⋮
        </Menu.Button>
      </div>
      <Menu.Items className="origin-top-right absolute right-0 mt-2 w-48 rounded-md shadow-lg bg-white ring-1 ring-black ring-opacity-5 focus:outline-none z-10">
        <div className="py-1">
          <Menu.Item>
            {({ active }) => (
              <button
                onClick={task.card_pk === 0 ? toggleCardLink : handleCardUnlink}
                className={`${
                  active ? 'bg-gray-100 text-gray-900' : 'text-gray-700'
                } group flex rounded-md items-center w-full px-2 py-2 text-sm`}
              >
                {task.card_pk === 0 ? 'Link Card' : 'Unlink Card'}
              </button>
            )}
          </Menu.Item>
        </div>
      </Menu.Items>
    </Menu>
  );
}
