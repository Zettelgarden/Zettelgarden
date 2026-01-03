import React, { useState, useEffect, useContext } from "react";
import { saveNewTask } from "../..//api/tasks";
import { useTaskContext } from "../../contexts/TaskContext";
import { Task, emptyTask } from "../../models/Task";
import { Card, PartialCard } from "../..//models/Card";
import { BacklinkInput } from "../cards/BacklinkInput";
import { TaskDateDisplay } from "./TaskDateDisplay";
import { TaskPriorityDisplay } from "./TaskPriorityDisplay";
import { TaskReminderDisplay } from "./TaskReminderDisplay";
import { TaskStatusDisplay } from "./TaskStatusDisplay";
import { TaskTagDisplay } from "./TaskTagDisplay";
import { Button } from "../Button";
import { AddTagMenu } from "../../components/tags/AddTagMenu";

interface CreateTaskWindowProps {
  currentCard: Card | PartialCard | null;
  setShowTaskWindow: (showTaskWindow: boolean) => void;
  currentFilter?: string;
}

export function CreateTaskWindow({
  currentCard,
  setShowTaskWindow,
  currentFilter,
}: CreateTaskWindowProps) {
  const [newTask, setNewTask] = useState<Task>({
    ...emptyTask,
    scheduled_date: new Date(),
  });
  const [selectedCard, setSelectedCard] = useState<PartialCard | null>(null);

  const [showMenu, setShowMenu] = useState<boolean>(false);
  const [showTagMenu, setShowTagMenu] = useState<boolean>(false);
  const { setRefreshTasks } = useTaskContext();

  // Function to detect and extract priority from text
  function detectAndSetPriority(text: string) {
    // Regex to match "priority:a", "priority:b", or "priority:c" (case insensitive)
    const priorityRegex = /priority:\s*([abc])/i;
    const match = text.match(priorityRegex);
    
    if (match) {
      const detectedPriority = match[1].toUpperCase();
      // Remove the priority text from the title
      const cleanedTitle = text.replace(/priority:\s*[abc]/i, '').trim();
      
      // Update the task with both cleaned title and detected priority
      setNewTask({ ...newTask, title: cleanedTitle, priority: detectedPriority });
    } else {
      // No priority detected, just update the title
      setNewTask({ ...newTask, title: text });
    }
  }

  async function handleSaveTask() {
    let response;

    // Make a copy of the task to ensure all properties are included
    let task = { ...newTask };

    if (currentCard) {
      task.card_pk = currentCard.id;
    }

    // Log the task to verify priority is included
    console.log("Saving task with priority:", task.priority);

    response = await saveNewTask(task);
    if (!("error" in response)) {
      setShowTaskWindow(false);
      setSelectedCard(null);
      setRefreshTasks(true);
      setNewTask({ ...emptyTask, scheduled_date: new Date() });
      if (currentCard) {
        setNewTask({ ...emptyTask, card_pk: currentCard.id });
      }
    }
  }
  function handleBacklink(card: PartialCard) {
    setSelectedCard(card);
    setNewTask({ ...newTask, card_pk: card.id });
  }

  function toggleMenu() {
    if (showTagMenu) {
      setShowTagMenu(false);
    }
    setShowMenu(!showMenu);
  }

  async function handleAddTagClick() {
    setShowMenu(false);
    setShowTagMenu(true);
  }
  async function handleAddRecurring(tag: string) {
    let editedTask = { ...newTask, title: newTask.title + " " + tag };
    setNewTask(editedTask);
    setShowTagMenu(false);
    setShowMenu(false);
  }

  function handleRemoveTag(tagName: string) {
    // Remove # prefix if present
    const cleanTagName = tagName.replace(/^#/, '');
    const tagRegex = new RegExp(`\\n*#${cleanTagName}\\b`, 'g');

    const updatedTask = {
      ...newTask,
      title: newTask.title.replace(tagRegex, '').trim(),
      tags: newTask.tags.filter(tag => tag.name.replace(/^#/, '') !== cleanTagName)
    };

    setNewTask(updatedTask);
  }
  const handleKeyPress = (event: KeyboardEvent) => {
    // if this is true, the user is using a system shortcut, don't do anything with it
    if (event.metaKey) {
      return;
    }
    if (event.key === "Escape") {
      event.preventDefault();
      return;
    }
  };

  useEffect(() => {
    const keyDownListener = (event: KeyboardEvent) => handleKeyPress(event);
    document.addEventListener("keydown", keyDownListener);
    return () => {
      document.removeEventListener("keydown", keyDownListener);
    };
  }, []);

  useEffect(() => {
    if (currentFilter === undefined) {
      setNewTask({ ...newTask, title: "" });
    } else {
      detectAndSetPriority(currentFilter + " ");
    }
  }, [currentFilter]);

  // console.log("create task", currentFilter);

  return (
    <div
      className="create-task-popup-overlay"
      onClick={() => setShowTaskWindow(false)}
    >
      <div
        className="create-task-popup-content"
        onClick={(e) => e.stopPropagation()}
      >
        <div className="flex flex-col h-full">
          <div className="mb-4">
            <span className="block mb-3 font-bold text-gray-700 text-lg">
              {"New Task"}
            </span>
            <div className="flex gap-2 mb-4">
              <input
                className="flex-1 px-3 py-2.5 text-base border rounded-md focus:outline-none focus:ring-2 focus:ring-blue-500 border-gray-300 focus:border-blue-500"
                placeholder="Enter new task"
                value={newTask.title}
                onChange={(e) =>
                  detectAndSetPriority(e.target.value)
                }
                onKeyPress={(event: React.KeyboardEvent<HTMLInputElement>) => {
                  if (event.key === "Enter") {
                    handleSaveTask();
                  }
                }}
                autoFocus
              />
              <div className="dropdown flex-shrink-0">
                <button onClick={toggleMenu} className="menu-button h-full px-3">
                  ⋮
                </button>
                {showMenu && (
                  <div className="absolute right-0 top-full bg-white border border-gray-200 rounded-md shadow-lg z-50 min-w-[200px]">
                    <button
                      onClick={() => handleAddTagClick()}
                      className="w-full px-4 py-2 text-left hover:bg-gray-50"
                    >
                      Add Tag
                    </button>
                    <div className="h-px bg-gray-200 my-1"></div>
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
                        Tip: You can type "every X days/weeks/months" for custom intervals
                      </div>
                    </div>
                  </div>
                )}
                {showTagMenu && (
                  <div className="popup-menu">
                    <AddTagMenu task={newTask} handleAddTag={handleAddRecurring} />
                  </div>
                )}
              </div>
            </div>
          </div>
          <div className="flex flex-col gap-4">
            {!currentCard && (
              <div className="w-full">
                <BacklinkInput addBacklink={handleBacklink} />
              </div>
            )}

            <div className="flex flex-col gap-3">
              {selectedCard && (
                <span className="font-bold text-blue-600">
                  [{selectedCard?.card_id}]
                </span>
              )}
              <div className="flex flex-col sm:flex-row items-stretch sm:items-center gap-3 flex-wrap">
                <TaskStatusDisplay
                  task={newTask}
                  setTask={setNewTask}
                  saveOnChange={false}
                />
                <TaskDateDisplay
                  task={newTask}
                  setTask={setNewTask}
                  saveOnChange={false}
                />
                <TaskPriorityDisplay
                  task={newTask}
                  setTask={setNewTask}
                  saveOnChange={false}
                />
                <TaskReminderDisplay
                  task={newTask}
                  setTask={setNewTask}
                  saveOnChange={false}
                />
                <TaskTagDisplay
                  task={newTask}
                  tags={newTask.tags}
                  onTagClick={() => {}}
                  onRemoveTag={handleRemoveTag}
                />
              </div>
            </div>

            <div className="flex justify-end mt-2">
              <Button
                onClick={handleSaveTask}
                className="w-full sm:w-auto px-6 py-2.5 text-base"
              >
                Save
              </Button>
            </div>
          </div>
        </div>
      </div>
    </div>
  );
}
