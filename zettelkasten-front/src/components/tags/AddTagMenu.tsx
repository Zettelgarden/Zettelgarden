import React, { useState, ChangeEvent } from "react";
import { Task } from "../../models/Task";
import { Tag } from "../../models/Tags";
import { useTagContext } from "../../contexts/TagContext";
import { Menu } from "@headlessui/react";

interface AddTagMenuProps {
  task: Task;
  handleAddTag: (tagName: string) => void;
}

export function AddTagMenu({ task, handleAddTag }: AddTagMenuProps) {
  const [textInput, setTextInput] = useState<string>("");
  const { tags, setRefreshTags } = useTagContext();

  function handleExistingTagClick(tag: Tag) {
    handleAddTag("#" + tag.name);
    setRefreshTags(true);
  }

  function handleInput(
    e: ChangeEvent<HTMLInputElement | HTMLTextAreaElement | HTMLSelectElement>,
  ) {
    setTextInput(e.target.value);
  }

  async function handleEnter() {
    handleAddTag("#" + textInput);
    setRefreshTags(true);
  }

  console.log("tags", tags)

  return (
    <Menu as="div" className="relative inline-block text-left">
      <div>
        <Menu.Button className="inline-flex justify-center items-center w-full rounded-md border border-gray-300 shadow-sm px-2 py-2 min-w-[44px] min-h-[44px] bg-white text-sm font-medium text-gray-700 hover:bg-gray-50 focus:outline-none focus:ring-2 focus:ring-offset-2 focus:ring-offset-gray-100 focus:ring-indigo-500">
          <svg xmlns="http://www.w3.org/2000/svg" className="h-5 w-5" viewBox="0 0 20 20" fill="currentColor">
            <path fillRule="evenodd" d="M10 3a1 1 0 011 1v5h5a1 1 0 110 2h-5v5a1 1 0 11-2 0v-5H4a1 1 0 110-2h5V4a1 1 0 011-1z" clipRule="evenodd" />
          </svg>
        </Menu.Button>
      </div>
      <Menu.Items className="origin-top-right absolute right-0 mt-2 w-48 rounded-md shadow-lg bg-white ring-1 ring-black ring-opacity-5 focus:outline-none z-10">
        <div className="p-2">
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
          <div className="max-h-32 overflow-y-auto">
            {tags &&
              tags
                .filter((tag) =>
                  tag.name.toLowerCase().includes(textInput.toLowerCase()),
                )
                .map((tag) =>
                  task.title.includes("#" + tag.name) ? (
                    <div key={tag.id}></div>
                  ) : (
                    <Menu.Item key={tag.id}>
                      {({ active }) => (
                        <button
                          onClick={() => handleExistingTagClick(tag)}
                          className={`${
                            active ? 'bg-gray-100 text-gray-900' : 'text-gray-700'
                          } group flex rounded-md items-center w-full px-4 py-3 min-h-[44px] text-sm`}
                        >
                          {"#" + tag.name}
                        </button>
                      )}
                    </Menu.Item>
                  ),
                )}
          </div>
        </div>
      </Menu.Items>
    </Menu>
  );
}
