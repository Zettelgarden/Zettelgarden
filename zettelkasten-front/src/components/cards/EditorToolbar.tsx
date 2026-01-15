import React from "react";
import { Menu } from "@headlessui/react";
import { Card } from "../../models/Card";

interface EditorToolbarProps {
  newCard: boolean;
  originalCard: Card;
  editingCard: Card;
  setEditingCard: (card: Card) => void;
  setShowSaveAsTemplate: (show: boolean) => void;
}

export function EditorToolbar({
  newCard,
  originalCard,
  editingCard,
  setEditingCard,
  setShowSaveAsTemplate,
}: EditorToolbarProps) {
  return (
    <div className="flex flex-col md:flex-row items-start md:items-center justify-between bg-white rounded-lg p-3 shadow-sm">
      <div className="flex-grow">
        <div className="flex items-center flex-wrap gap-2">
          <span className="font-bold text-gray-600">
            Editing:
          </span>
          {newCard ? (
            <div>
              <span className="text-gray-600">{"New Card"}</span>
            </div>
          ) : (
            <div>
              <span className="text-blue-600">[{originalCard.card_id}]</span>
              <span className="text-gray-600">{" - "}{originalCard.title}</span>
            </div>
          )}
        </div>
      </div>
      <div className="mt-2 md:mt-0 md:ml-4 flex gap-2">
        <Menu as="div" className="relative inline-block text-left">
          <div>
            <Menu.Button className="inline-flex justify-center w-full rounded-md border border-gray-300 shadow-sm px-2 py-2 bg-white text-sm font-medium text-gray-700 hover:bg-gray-50 focus:outline-none focus:ring-2 focus:ring-offset-2 focus:ring-offset-gray-100 focus:ring-indigo-500">
              <svg
                xmlns="http://www.w3.org/2000/svg"
                className="h-5"
                viewBox="0 0 20 20"
                fill="currentColor"
              >
                <path d="M10 6a2 2 0 110-4 2 2 0 010 4zM10 12a2 2 0 110-4 2 2 0 010 4zM10 18a2 2 0 110-4 2 2 0 010 4z" />
              </svg>
            </Menu.Button>
            <Menu.Items className="origin-top-left md:origin-top-right absolute z-10 right-0 md:right-0 left-0 md:left-auto mt-2 w-56 divide-y divide-gray-100 rounded-md bg-white shadow-lg ring-1 ring-black ring-opacity-5 focus:outline-none">
              <div className="px-1 py-1 ">
                {!newCard && (
                  <Menu.Item>
                    <div className="flex items-center gap-2 p-2">
                      <input
                        type="checkbox"
                        id="process_entities_and_facts"
                        checked={editingCard.process_entities_and_facts || false}
                        onChange={(e) =>
                          setEditingCard({
                            ...editingCard,
                            process_entities_and_facts: e.target.checked,
                          })
                        }
                        className="rounded border-gray-300 text-blue-600 focus:ring-blue-500"
                      />
                      <label
                        htmlFor="process_entities_and_facts"
                        className="text-sm text-gray-700"
                      >
                        Process Entities & Facts
                      </label>
                    </div>
                  </Menu.Item>
                )}
                <Menu.Item>
                  {({ active }) => (
                    <button
                      onClick={() => setShowSaveAsTemplate(true)}
                      className={`${
                        active ? "bg-gray-100 text-gray-900" : "text-gray-700"
                      } group flex rounded-md items-center w-full px-2 py-2 text-sm`}
                    >
                      <span className="flex-grow text-left">Save as Template</span>
                    </button>
                  )}
                </Menu.Item>
              </div>
            </Menu.Items>
          </div>
        </Menu>
      </div>
    </div>
  );
}