import React, { useState, useEffect } from "react";
import { saveNewTask } from "../../api/tasks";
import { useTaskContext } from "../../contexts/TaskContext";
import { useAuth } from "../../contexts/AuthContext";
import { useUIState } from "../../contexts/UIStateContext";
import { Task, emptyTask } from "../../models/Task";
import { Card, PartialCard } from "../../models/Card";
import { Button } from "../Button";
import { TaskForm } from "./TaskForm";
import { stripSpecialFilters } from "../../utils/tasks";
import { getToday } from "../../utils/dates";

interface CreateTaskWindowProps {
  currentCard: Card | PartialCard | null;
  setShowTaskWindow: (showTaskWindow: boolean) => void;
  currentFilter?: string;
  initialStatus?: string;
  initialDate?: Date;
}

export function CreateTaskWindow({
  currentCard,
  setShowTaskWindow,
  currentFilter,
  initialStatus,
  initialDate,
}: CreateTaskWindowProps) {
  const { user } = useAuth();
  const userTimezone = user?.timezone || "UTC";

  const [newTask, setNewTask] = useState<Task>({
    ...emptyTask,
    scheduled_date: initialDate || getToday(userTimezone),
    status: initialStatus || emptyTask.status || "todo",
  });
  const [selectedCard, setSelectedCard] = useState<PartialCard | null>(null);

  const { setRefreshTasks } = useTaskContext();
  const { setRefreshTrigger } = useUIState();

  // Function to detect and extract priority from text
  function detectAndSetPriority(text: string) {
    const priorityRegex = /priority:\s*([abc])/i;
    const match = text.match(priorityRegex);

    if (match) {
      const detectedPriority = match[1].toUpperCase();
      const cleanedTitle = text.replace(/priority:\s*[abc]/i, "").trim();
      setNewTask({ ...newTask, title: cleanedTitle, priority: detectedPriority });
    } else {
      setNewTask({ ...newTask, title: text });
    }
  }

  async function handleSaveTask() {
    let task = { ...newTask };

    if (currentCard) {
      task.card_pk = currentCard.id;
    }

    const response = await saveNewTask(task);
    if (!("error" in response)) {
      setShowTaskWindow(false);
      setSelectedCard(null);
      setRefreshTasks(true);
      // If task is associated with a card, trigger card refresh to update displayed tasks
      if (currentCard) {
        setRefreshTrigger(currentCard.id.toString());
      }
      setNewTask({
        ...emptyTask,
        scheduled_date: getToday(userTimezone),
        status: initialStatus || emptyTask.status || "todo",
      });
      if (currentCard) {
        setNewTask({
          ...emptyTask,
          card_pk: currentCard.id,
          scheduled_date: getToday(userTimezone),
          status: initialStatus || emptyTask.status || "todo",
        });
      }
    }
  }

  function handleBacklink(card: PartialCard) {
    setSelectedCard(card);
    setNewTask({ ...newTask, card_pk: card.id });
  }

  const handleKeyPress = (event: KeyboardEvent) => {
    if (event.metaKey) {
      return;
    }
    if (event.key === "Escape") {
      event.preventDefault();
      setShowTaskWindow(false);
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
    if (initialStatus) {
      setNewTask((prev) => ({ ...prev, status: initialStatus }));
    }
  }, [initialStatus]);

  useEffect(() => {
    if (currentFilter === undefined) {
      setNewTask({ ...newTask, title: "" });
    } else {
      const cleanedFilter = stripSpecialFilters(currentFilter);
      detectAndSetPriority(cleanedFilter + " ");
    }
  }, [currentFilter]);

  return (
    <div
      className="fixed top-0 left-0 w-full h-full bg-black/50 flex justify-center items-center z-[1000]"
      onClick={() => setShowTaskWindow(false)}
    >
      <div
        className="bg-white p-4 sm:p-6 rounded-lg shadow-md max-w-[672px] w-[95%] sm:w-[90%] max-h-[90vh] overflow-y-visible"
        onClick={(e) => e.stopPropagation()}
      >
        <div className="flex flex-col h-full">
          <div className="mb-4">
            <span className="block mb-3 font-bold text-gray-700 text-lg">
              {"New Task"}
            </span>
            {selectedCard && (
              <span className="font-bold text-blue-600 mb-2 block">
                [{selectedCard?.card_id}]
              </span>
            )}
          </div>

          <TaskForm
            task={newTask}
            setTask={setNewTask}
            mode="create"
            saveOnChange={false}
            onTitleSubmit={handleSaveTask}
            currentCard={currentCard}
            onBacklink={handleBacklink}
          />

          <div className="flex justify-end mt-4">
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
  );
}
