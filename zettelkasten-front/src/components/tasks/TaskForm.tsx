import React, { useState } from "react";
import { Task } from "../../models/Task";
import { PartialCard } from "../../models/Card";
import { BacklinkInput } from "../cards/BacklinkInput";
import { TaskDateDisplay } from "./TaskDateDisplay";
import { TaskDueDateDisplay } from "./TaskDueDateDisplay";
import { TaskPriorityDisplay } from "./TaskPriorityDisplay";
import { TaskReminderDisplay } from "./TaskReminderDisplay";
import { TaskStatusDisplay } from "./TaskStatusDisplay";
import { TaskTagDisplay } from "./TaskTagDisplay";
import { useTagContext } from "../../contexts/TagContext";
import { useTaskContext } from "../../contexts/TaskContext";
import { saveExistingTask, fetchTask, addTaskDependency, removeTaskDependency } from "../../api/tasks";

interface TaskFormProps {
  task: Task;
  setTask: (task: Task) => void;
  mode: "create" | "edit";
  saveOnChange: boolean;
  onTitleSubmit?: () => void;
  showCardLink?: boolean;
  currentCard?: PartialCard | null;
  onBacklink?: (card: PartialCard) => void;
}

export function TaskForm({
  task,
  setTask,
  mode,
  saveOnChange,
  onTitleSubmit,
  showCardLink = false,
  currentCard,
  onBacklink,
}: TaskFormProps) {
  const [isEditingTitle, setIsEditingTitle] = useState(mode === "create");
  const [isEditingDescription, setIsEditingDescription] = useState(false);
  const [showTagEditor, setShowTagEditor] = useState(false);
  const [newTagInput, setNewTagInput] = useState("");
  const [showDependencyEditor, setShowDependencyEditor] = useState(false);
  const [dependencyFilter, setDependencyFilter] = useState("");
  const [showRecurringMenu, setShowRecurringMenu] = useState(false);

  const { tags: allTags, setRefreshTags } = useTagContext();
  const { setRefreshTasks, tasks } = useTaskContext();

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
      const response = await saveExistingTask(task);
      if (!("error" in response)) {
        setRefreshTasks(true);
        setRefreshTags(true);
        setIsEditingTitle(false);
      }
    }
  }

  function handleDescriptionChange(e: React.ChangeEvent<HTMLTextAreaElement>) {
    setTask({ ...task, description: e.target.value || null });
  }

  async function handleDescriptionSave() {
    if (mode === "edit" && saveOnChange) {
      const response = await saveExistingTask(task);
      if (!("error" in response)) {
        setRefreshTasks(true);
        setIsEditingDescription(false);
      }
    } else {
      setIsEditingDescription(false);
    }
  }

  function handleBacklink(card: PartialCard) {
    setTask({ ...task, card_pk: card.id });
    if (onBacklink) {
      onBacklink(card);
    }
  }

  // Tag management
  async function handleAddTag(tagName: string) {
    const cleanTag = tagName.replace(/^#/, "").trim();
    if (!cleanTag || task.title.includes(`#${cleanTag}`)) {
      return;
    }

    const updatedTask = {
      ...task,
      title: `${task.title} #${cleanTag}`.trim(),
    };

    if (mode === "edit" && saveOnChange) {
      const response = await saveExistingTask(updatedTask);
      if (!("error" in response)) {
        const refreshedTask = await fetchTask(task.id.toString());
        setTask(refreshedTask);
        setRefreshTasks(true);
        setRefreshTags(true);
      }
    } else {
      setTask(updatedTask);
    }
  }

  async function handleRemoveTag(tagName: string) {
    const cleanTagName = tagName.replace(/^#/, "");
    const tagRegex = new RegExp(`\\s*#${cleanTagName}\\b`, "g");

    const updatedTask = {
      ...task,
      title: task.title.replace(tagRegex, "").trim(),
      tags: task.tags.filter(
        (tag) => tag.name.replace(/^#/, "") !== cleanTagName
      ),
    };

    if (mode === "edit" && saveOnChange) {
      const response = await saveExistingTask(updatedTask);
      if (!("error" in response)) {
        setTask(updatedTask);
        setRefreshTasks(true);
      }
    } else {
      setTask(updatedTask);
    }
  }

  function handleAddNewTag() {
    if (newTagInput.trim()) {
      handleAddTag(newTagInput);
      setNewTagInput("");
    }
  }

  function getCurrentTaskTags(): Set<string> {
    return new Set(task.tags.map((tag) => tag.name.replace(/^#/, "")));
  }

  // Recurring task options (create mode only)
  function handleAddRecurring(interval: string) {
    setTask({ ...task, title: task.title + " " + interval });
    setShowRecurringMenu(false);
  }

  // Dependency management (edit mode only)
  async function handleAddDependency(blockingTaskId: number) {
    if (mode !== "edit" || !task.id) return;

    try {
      await addTaskDependency(task.id, blockingTaskId);
      const updatedTask = await fetchTask(task.id.toString());
      setTask(updatedTask);
      setRefreshTasks(true);
    } catch (error) {
      console.error("Error adding dependency:", error);
    }
  }

  async function handleRemoveDependency(blockingTaskId: number) {
    if (mode !== "edit" || !task.id) return;

    try {
      await removeTaskDependency(task.id, blockingTaskId);
      const updatedTask = await fetchTask(task.id.toString());
      setTask(updatedTask);
      setRefreshTasks(true);
    } catch (error) {
      console.error("Error removing dependency:", error);
    }
  }

  return (
    <div className="flex flex-col gap-4">
      {/* Title Input */}
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

      {/* Description */}
      {mode === "create" || isEditingDescription ? (
        <div className="space-y-2">
          <textarea
            placeholder="Add a description..."
            value={task.description || ""}
            onChange={handleDescriptionChange}
            className="w-full px-3 py-2 border rounded-md focus:outline-none focus:ring-2 focus:ring-blue-500 border-gray-300 min-h-[80px] resize-y"
            autoFocus={mode === "edit"}
          />
          {mode === "edit" && (
            <div className="flex gap-2">
              <button
                onClick={handleDescriptionSave}
                className="px-3 py-1 bg-blue-600 text-white rounded-md hover:bg-blue-700 text-sm"
              >
                Save
              </button>
              <button
                onClick={() => setIsEditingDescription(false)}
                className="px-3 py-1 bg-gray-200 text-gray-700 rounded-md hover:bg-gray-300 text-sm"
              >
                Cancel
              </button>
            </div>
          )}
        </div>
      ) : (
        <div
          className="text-gray-600 cursor-pointer hover:bg-gray-50 p-2 rounded min-h-[40px]"
          onClick={() => setIsEditingDescription(true)}
        >
          {task.description ? (
            <p className="whitespace-pre-wrap">{task.description}</p>
          ) : (
            <p className="text-gray-400 italic">Add a description...</p>
          )}
        </div>
      )}

      {/* Card Link */}
      {(showCardLink || (!currentCard && mode === "create")) && (
        <div className="w-full">
          <BacklinkInput addBacklink={handleBacklink} />
        </div>
      )}

      {/* Display Components Row */}
      <div className="flex flex-col sm:flex-row items-stretch sm:items-center gap-3 flex-wrap">
        <TaskStatusDisplay
          task={task}
          setTask={setTask}
          saveOnChange={saveOnChange}
        />
        <TaskDateDisplay
          task={task}
          setTask={setTask}
          saveOnChange={saveOnChange}
        />
        <TaskDueDateDisplay
          task={task}
          setTask={setTask}
          saveOnChange={saveOnChange}
        />
        <TaskPriorityDisplay
          task={task}
          setTask={setTask}
          saveOnChange={saveOnChange}
        />
        <TaskReminderDisplay
          task={task}
          setTask={setTask}
          saveOnChange={saveOnChange}
        />
        <TaskTagDisplay
          task={task}
          tags={task.tags}
          onTagClick={() => {}}
          onRemoveTag={handleRemoveTag}
        />
        <button
          onClick={() => setShowTagEditor(!showTagEditor)}
          className="text-sm text-blue-600 hover:text-blue-800 font-medium"
        >
          {showTagEditor ? "- Hide Tags" : "+ Add Tags"}
        </button>
      </div>

      {/* Blocked By Section (edit mode only) */}
      {mode === "edit" && task.blocked_by && task.blocked_by.length > 0 && (
        <div className="flex items-center gap-2 flex-wrap">
          <span className="text-sm font-medium text-gray-700">Blocked by:</span>
          {task.blocked_by.map((blockingTask) => (
            <div
              key={blockingTask.id}
              className="inline-flex items-center gap-1 px-2 py-1 bg-orange-100 text-orange-800 rounded text-sm"
            >
              <span className={blockingTask.is_complete ? "line-through" : ""}>
                {blockingTask.title}
              </span>
              <button
                onClick={() => handleRemoveDependency(blockingTask.id)}
                className="ml-1 hover:text-orange-600"
                title="Remove blocker"
              >
                x
              </button>
            </div>
          ))}
        </div>
      )}

      {/* Add Blocker Button (edit mode only) */}
      {mode === "edit" && task.id > 0 && (
        <button
          onClick={() => {
            setShowDependencyEditor(!showDependencyEditor);
            if (showDependencyEditor) {
              setDependencyFilter("");
            }
          }}
          className="text-sm text-blue-600 hover:text-blue-800 font-medium w-fit"
        >
          {showDependencyEditor ? "- Hide Blockers" : "+ Add Blocker"}
        </button>
      )}

      {/* Dependency Editor (edit mode only) */}
      {mode === "edit" && showDependencyEditor && (
        <div className="border border-gray-200 rounded-lg p-4 bg-gray-50 space-y-3">
          <label className="block text-sm font-medium text-gray-700">
            Select tasks that block this task
          </label>
          <input
            type="text"
            value={dependencyFilter}
            onChange={(e) => setDependencyFilter(e.target.value)}
            placeholder="Filter tasks..."
            className="w-full px-3 py-2 border border-gray-300 rounded-md focus:outline-none focus:ring-2 focus:ring-blue-500"
          />
          <div className="max-h-48 overflow-y-auto border border-gray-200 rounded-md p-3 bg-white">
            {(() => {
              const availableTasks = tasks
                .filter((t) => t.id !== task.id && !t.is_complete)
                .filter(
                  (t) =>
                    dependencyFilter === "" ||
                    t.title.toLowerCase().includes(dependencyFilter.toLowerCase())
                );

              if (availableTasks.length === 0) {
                return (
                  <p className="text-gray-500 text-sm text-center py-2">
                    {dependencyFilter
                      ? "No tasks match your search"
                      : "No available tasks"}
                  </p>
                );
              }

              return (
                <div className="space-y-2">
                  {availableTasks.map((t) => {
                    const isBlocking =
                      task.blocked_by?.some((bt) => bt.id === t.id) || false;
                    return (
                      <button
                        key={t.id}
                        onClick={() => {
                          if (isBlocking) {
                            handleRemoveDependency(t.id);
                          } else {
                            handleAddDependency(t.id);
                          }
                        }}
                        className={`w-full text-left px-3 py-2 rounded transition-colors ${
                          isBlocking
                            ? "bg-orange-100 text-orange-800 border border-orange-300"
                            : "bg-gray-100 text-gray-700 hover:bg-gray-200"
                        }`}
                      >
                        <div className="flex items-center justify-between">
                          <span>{t.title}</span>
                          {isBlocking && (
                            <span className="text-xs">Blocking</span>
                          )}
                        </div>
                      </button>
                    );
                  })}
                </div>
              );
            })()}
          </div>
        </div>
      )}

      {/* Tag Editor */}
      {showTagEditor && (
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
                              handleRemoveTag(cleanTagName);
                            } else {
                              handleAddTag(cleanTagName);
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
      )}
    </div>
  );
}
