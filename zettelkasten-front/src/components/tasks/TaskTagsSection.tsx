import React from "react";
import { Task } from "../../models/Task";

interface TaskTag {
  id: number;
  name: string;
}

interface TaskTagsSectionProps {
  task: Task;
  showTagEditor: boolean;
  newTagInput: string;
  setNewTagInput: (input: string) => void;
  allTags: TaskTag[];
  onAddTag: (tagName: string) => Promise<void>;
  onRemoveTag: (tagName: string) => Promise<void>;
  getCurrentTaskTags: () => Set<string>;
}

export function TaskTagsSection({
  task,
  showTagEditor,
  newTagInput,
  setNewTagInput,
  allTags,
  onAddTag,
  onRemoveTag,
  getCurrentTaskTags,
}: TaskTagsSectionProps) {
  function handleAddNewTag() {
    if (newTagInput.trim()) {
      onAddTag(newTagInput);
      setNewTagInput("");
    }
  }

  return showTagEditor ? (
    <div className="border border-gray-200 rounded-lg p-4 bg-gray-50 space-y-3">
      {/* New tag input */}
      <div>
        <label className="block text-sm font-medium text-gray-700 mb-2">
          Add New Tag
        </label>
        <div className="flex gap-2">
          <input
            type="text"
            value={newTagInput}
            onChange={(e) => setNewTagInput(e.target.value)}
            onKeyPress={(e) => {
              if (e.key === "Enter") {
                handleAddNewTag();
              }
            }}
            placeholder="Enter tag name"
            className="flex-1 px-3 py-2 border border-gray-300 rounded-md focus:outline-none focus:ring-2 focus:ring-blue-500"
          />
          <button
            onClick={handleAddNewTag}
            disabled={!newTagInput.trim()}
            className="px-4 py-2 bg-blue-600 text-white rounded-md hover:bg-blue-700 disabled:opacity-50 disabled:cursor-not-allowed"
          >
            Add
          </button>
        </div>
      </div>

      {/* Available tags */}
      <div>
        <label className="block text-sm font-medium text-gray-700 mb-2">
          Available Tags (
          {
            allTags.filter(
              (tag) =>
                newTagInput.trim() === "" ||
                tag.name
                  .replace(/^#/, "")
                  .toLowerCase()
                  .includes(newTagInput.trim().toLowerCase())
            ).length
          }
          )
        </label>
        <div className="max-h-48 overflow-y-auto border border-gray-200 rounded-md p-3 bg-white">
          {(() => {
            const filteredTags = allTags.filter(
              (tag) =>
                newTagInput.trim() === "" ||
                tag.name
                  .replace(/^#/, "")
                  .toLowerCase()
                  .includes(newTagInput.trim().toLowerCase())
            );

            if (filteredTags.length === 0) {
              return newTagInput.trim() ? (
                <p className="text-gray-500 text-sm text-center py-2">
                  No tags match "{newTagInput.trim()}"
                </p>
              ) : (
                <p className="text-gray-500 text-sm text-center py-2">
                  No tags available
                </p>
              );
            }

            return (
              <div className="flex flex-wrap gap-2">
                {filteredTags.map((tag) => {
                  const cleanTagName = tag.name.replace(/^#/, "");
                  const isSelected = getCurrentTaskTags().has(cleanTagName);
                  return (
                    <button
                      key={tag.id}
                      onClick={() => {
                        if (isSelected) {
                          onRemoveTag(cleanTagName);
                        } else {
                          onAddTag(cleanTagName);
                        }
                      }}
                      className={`px-3 py-1 rounded-full text-sm transition-colors ${
                        isSelected
                          ? "bg-purple-600 text-white"
                          : "bg-gray-200 text-gray-700 hover:bg-gray-300"
                      }`}
                    >
                      #{cleanTagName}
                      {isSelected && " ✓"}
                    </button>
                  );
                })}
              </div>
            );
          })()}
        </div>
      </div>
    </div>
  ) : null;
}