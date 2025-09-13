import React, { useState, ChangeEvent } from "react";
import { Task } from "../../models/Task";
import { Tag } from "../../models/Tags";

import { saveExistingTask } from "../../api/tasks";

import { useTaskContext } from "../../contexts/TaskContext";
import { useTagContext } from "../../contexts/TagContext";
import { Menu } from "@headlessui/react";

interface TaskListOptionsMenuProps {
  task: Task;
  tags: Tag[];
  showCardLink: boolean;
  setShowCardLink: (show: boolean) => void;
}

export function TaskListOptionsMenu({
  task,
  tags,
  showCardLink,
  setShowCardLink,
}: TaskListOptionsMenuProps) {
  const [textInput, setTextInput] = useState<string>("");
  const { setRefreshTasks } = useTaskContext();
  const { tags: allTags, setRefreshTags } = useTagContext();

  function handleInput(
    e: ChangeEvent<HTMLInputElement | HTMLTextAreaElement | HTMLSelectElement>,
  ) {
    setTextInput(e.target.value);
  }

  function handleExistingTagClick(tag: Tag) {
    handleAddTag("#" + tag.name);
    setRefreshTags(true);
  }

  async function handleEnter() {
    handleAddTag("#" + textInput);
    setRefreshTags(true);
    setTextInput("");
  }

  async function handleAddTag(tagName: string) {
    let editedTask = { ...task, title: task.title + " " + tagName };
    let response = await saveExistingTask(editedTask);
    if (!("error" in response)) {
      setRefreshTasks(true);
    }
  }

  async function handleRemoveTag(tag: string) {
    const tagRegex = new RegExp(`(?:^|\\s)${tag}(?=\\s|$)`, "g");
    let editedTitle = task.title.replace(tagRegex, "").trim();
    editedTitle = editedTitle.replace(/\s+/g, " ");
    let editedTask = { ...task, title: editedTitle };

    let response = await saveExistingTask(editedTask);
    if (!("error" in response)) {
      setRefreshTasks(true);
    }
  }

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

          <div className="border-t border-gray-100 my-1"></div>

          {/* Add Tag Section */}
          <div className="p-2">
            <div className="text-xs text-gray-500 mb-2 font-semibold">Add Tag</div>
            <input
              type="text"
              value={textInput}
              placeholder="Tag"
              onChange={handleInput}
              onKeyPress={(event: React.KeyboardEvent<HTMLInputElement>) => {
                if (event.key === "Enter") {
                  handleEnter();
                }
              }}
              className="w-full px-2 py-1 border border-gray-300 rounded-md text-sm mb-2 focus:outline-none focus:ring-2 focus:ring-indigo-500"
            />
            <div className="max-h-24 overflow-y-auto">
              {allTags &&
                allTags
                  .filter((tag) =>
                    tag.name.toLowerCase().includes(textInput.toLowerCase()),
                  )
                  .map((tag) =>
                    task.title.includes("#" + tag.name) ? null : (
                      <Menu.Item key={tag.id}>
                        {({ active }) => (
                          <button
                            onClick={() => handleExistingTagClick(tag)}
                            className={`${
                              active ? 'bg-gray-100 text-gray-900' : 'text-gray-700'
                            } group flex rounded-md items-center w-full px-2 py-1 text-xs mb-1`}
                          >
                            {"#" + tag.name}
                          </button>
                        )}
                      </Menu.Item>
                    ),
                  )}
            </div>
          </div>

          {tags.length > 0 && (
            <>
              <div className="border-t border-gray-100 my-1"></div>
              <div className="text-xs text-gray-500 mb-1 px-2 font-semibold">Remove Tags</div>
              {tags.map((tag) => (
                <Menu.Item key={tag.id}>
                  {({ active }) => (
                    <button
                      onClick={() => handleRemoveTag("#" + tag.name)}
                      className={`${
                        active ? 'bg-gray-100 text-gray-900' : 'text-gray-700'
                      } group flex rounded-md items-center w-full px-2 py-2 text-sm`}
                    >
                      Remove #{tag.name}
                    </button>
                  )}
                </Menu.Item>
              ))}
            </>
          )}
        </div>
      </Menu.Items>
    </Menu>
  );
}
