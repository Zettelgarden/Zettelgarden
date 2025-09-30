import React from "react";
import { useNavigate } from "react-router-dom";
import { Menu } from "@headlessui/react";
import { Tag } from "../../models/Tags";
import { deleteTag } from "../../api/tags";
import { useTagContext } from "../../contexts/TagContext";

interface TagListItemInterface {
  tag: Tag;
}

export function TagListItem({ tag }: TagListItemInterface) {
  const { setRefreshTags } = useTagContext();
  const navigate = useNavigate();

  function handleViewCards() {
    let searchTerm = "#" + tag.name;
    navigate(`/app/search?term=${encodeURIComponent(searchTerm)}`);
  }

  function handleViewTasks() {
    let searchTerm = "#" + tag.name;
    navigate(`/app/tasks?term=${encodeURIComponent(searchTerm)}`);
  }

  async function handleDelete() {
    let _ = await deleteTag(tag.id)
      .then((data) => {
        setRefreshTags(true);
      })
      .catch((error) =>
        alert("Unable to delete tag.")
      );
  }

  return (
    <div className="bg-white rounded-lg shadow-md p-4 hover:shadow-lg transition-all relative">
      <div className="flex justify-between items-center mb-2">
        <h3 className="text-lg font-semibold">{tag.name}</h3>
        <Menu as="div" className="relative">
          <Menu.Button className="text-gray-600 hover:text-blue-600 cursor-pointer p-1">
            <svg
              className="w-5 h-5"
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

          <Menu.Items className="absolute right-0 z-10 mt-1 w-48 origin-top-right bg-white border border-gray-200 rounded-md shadow-lg focus:outline-none">
            <div className="py-1">
              <Menu.Item>
                {({ active }) => (
                  <button
                    onClick={handleViewCards}
                    className={`${
                      active ? "bg-gray-100" : ""
                    } flex w-full items-center px-4 py-2 text-sm text-gray-700 hover:bg-gray-100`}
                  >
                    View Cards
                  </button>
                )}
              </Menu.Item>

              <Menu.Item>
                {({ active }) => (
                  <button
                    onClick={handleViewTasks}
                    className={`${
                      active ? "bg-gray-100" : ""
                    } flex w-full items-center px-4 py-2 text-sm text-gray-700 hover:bg-gray-100`}
                  >
                    View Tasks
                  </button>
                )}
              </Menu.Item>

              <Menu.Item>
                {({ active }) => (
                  <button
                    onClick={handleDelete}
                    className={`${
                      active ? "bg-gray-100" : ""
                    } flex w-full items-center px-4 py-2 text-sm text-red-600 hover:bg-gray-100`}
                  >
                    Delete
                  </button>
                )}
              </Menu.Item>
            </div>
          </Menu.Items>
        </Menu>
      </div>
    </div>
  );
}
