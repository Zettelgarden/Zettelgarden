import React from "react";
import { Task } from "../../models/Task";

interface TaskTitleSectionProps {
  task: Task;
  setTask: (task: Task) => void;
  mode: "create" | "edit";
  isEditingTitle: boolean;
  setIsEditingTitle: (editing: boolean) => void;
  showRecurringMenu: boolean;
  setShowRecurringMenu: (show: boolean) => void;
  onTitleSubmit?: () => void;
  saveOnChange: boolean;
  onSaveTitle: () => Promise<void>;
}

export function TaskTitleSection({
  task,
  setTask,
  mode,
  isEditingTitle,
  setIsEditingTitle,
  showRecurringMenu,
  setShowRecurringMenu,
  onTitleSubmit,
  saveOnChange,
  onSaveTitle,
}: TaskTitleSectionProps) {
  // Priority detection from text input (e.g., "priority:a" -> Priority A)
  function detectAndSetPriority(text: string) {
    const priorityRegex = /priority:\s*([abc])/i;
    const match = text.match(priorityRegex);

    if (match) {
      const detectedPriority = match[1].toUpperCase();
      const cleanedTitle = text.replace(/priority:\s*[abc]/i, "").trim();
      setTask({ ...task, title: cleanedTitle, priority: detectedPriority });
    } else {
      setTask({ ...task, title: text });
    }
  }

  function handleTitleChange(e: React.ChangeEvent<HTMLInputElement>) {
    detectAndSetPriority(e.target.value);
  }

  function handleTitleKeyPress(e: React.KeyboardEvent<HTMLInputElement>) {
    if (e.key === "Enter") {
      if (mode === "create" && onTitleSubmit) {
        onTitleSubmit();
      } else if (mode === "edit") {
        handleTitleSave();
      }
    }
  }

  async function handleTitleSave() {
    if (mode === "edit" && saveOnChange) {
      await onSaveTitle();
      setIsEditingTitle(false);
    }
  }

  // Recurring task options (create mode only)
  function handleAddRecurring(interval: string) {
    setTask({ ...task, title: task.title + " " + interval });
    setShowRecurringMenu(false);
  }

  return (
    <div className="flex gap-2">
      {mode === "create" || isEditingTitle ? (
        <input
          className="flex-1 px-3 py-2.5 text-base border rounded-md focus:outline-none focus:ring-2 focus:ring-blue-500 border-gray-300 focus:border-blue-500"
          placeholder="Enter task title"
          value={task.title}
          onChange={handleTitleChange}
          onKeyPress={handleTitleKeyPress}
          onBlur={mode === "edit" ? handleTitleSave : undefined}
          autoFocus
        />
      ) : (
        <div
          className={`flex-1 text-lg cursor-pointer hover:bg-gray-50 p-2 rounded ${
            task.is_complete ? "line-through text-gray-500" : ""
          }`}
          onClick={() => setIsEditingTitle(true)}
        >
          {task.title}
        </div>
      )}

      {/* Recurring Options Menu (create mode only) */}
      {mode === "create" && (
        <div className="relative flex-shrink-0">
          <button
            onClick={() => setShowRecurringMenu(!showRecurringMenu)}
            className="menu-button h-full px-3"
          >
            ...
          </button>
          {showRecurringMenu && (
            <div className="absolute right-0 top-full bg-white border border-gray-200 rounded-md shadow-lg z-50 min-w-[200px]">
              <div className="py-1">
                <div className="px-4 py-1 text-sm font-medium text-gray-600">
                  Recurring Task
                </div>
                <button
                  onClick={() => handleAddRecurring("every day")}
                  className="w-full px-4 py-2 text-left hover:bg-gray-50"
                >
                  Daily
                </button>
                <button
                  onClick={() => handleAddRecurring("every week")}
                  className="w-full px-4 py-2 text-left hover:bg-gray-50"
                >
                  Weekly
                </button>
                <button
                  onClick={() => handleAddRecurring("every month")}
                  className="w-full px-4 py-2 text-left hover:bg-gray-50"
                >
                  Monthly
                </button>
                <div className="px-4 py-2 text-xs text-gray-600 border-t border-gray-100 bg-gray-50">
                  Tip: You can type "every X days/weeks/months" for custom
                  intervals
                </div>
              </div>
            </div>
          )}
        </div>
      )}
    </div>
  );
}