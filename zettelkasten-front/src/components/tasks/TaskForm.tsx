import React, { useState, useRef, useEffect } from "react";
import { Task } from "../../models/Task";
import { PartialCard } from "../../models/Card";
import { BacklinkInput } from "../cards/BacklinkInput";
import { TaskTitleSection } from "./TaskTitleSection";
import { TaskDescriptionSection } from "./TaskDescriptionSection";
import { TaskScheduleSection } from "./TaskScheduleSection";
import { TaskDependenciesSection } from "./TaskDependenciesSection";
import { TaskTagsSection } from "./TaskTagsSection";
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

  // Debounce timer for auto-save to avoid excessive API calls
  const debounceTimerRef = useRef<NodeJS.Timeout | null>(null);
  const pendingSaveRef = useRef<Task | null>(null);

  // Debounced save function
  const debouncedSave = useRef((taskToSave: Task) => {
    if (debounceTimerRef.current) {
      clearTimeout(debounceTimerRef.current);
    }
    pendingSaveRef.current = taskToSave;
    debounceTimerRef.current = setTimeout(async () => {
      if (pendingSaveRef.current && mode === "edit" && saveOnChange) {
        const response = await saveExistingTask(pendingSaveRef.current);
        if (!("error" in response)) {
          setRefreshTasks(true);
          setRefreshTags(true);
        }
      }
      pendingSaveRef.current = null;
    }, 500); // 500ms debounce delay
  }).current;

  // Cleanup debounce timer on unmount
  useEffect(() => {
    return () => {
      if (debounceTimerRef.current) {
        clearTimeout(debounceTimerRef.current);
      }
    };
  }, []);

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
      debouncedSave(task);
      setIsEditingTitle(false);
    }
  }

  function handleDescriptionChange(e: React.ChangeEvent<HTMLTextAreaElement>) {
    setTask({ ...task, description: e.target.value || null });
  }

  async function handleDescriptionSave() {
    if (mode === "edit" && saveOnChange) {
      debouncedSave(task);
      setIsEditingDescription(false);
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
      <TaskTitleSection
        task={task}
        setTask={setTask}
        mode={mode}
        isEditingTitle={isEditingTitle}
        setIsEditingTitle={setIsEditingTitle}
        showRecurringMenu={showRecurringMenu}
        setShowRecurringMenu={setShowRecurringMenu}
        onTitleSubmit={onTitleSubmit}
        saveOnChange={saveOnChange}
        onSaveTitle={handleTitleSave}
      />

      <TaskDescriptionSection
        task={task}
        setTask={setTask}
        mode={mode}
        isEditingDescription={isEditingDescription}
        setIsEditingDescription={setIsEditingDescription}
        onSaveDescription={handleDescriptionSave}
      />

      {/* Card Link */}
      {(showCardLink || (!currentCard && mode === "create")) && (
        <div className="w-full">
          <BacklinkInput addBacklink={handleBacklink} />
        </div>
      )}

      <TaskScheduleSection
        task={task}
        setTask={setTask}
        saveOnChange={saveOnChange}
        showTagEditor={showTagEditor}
        setShowTagEditor={setShowTagEditor}
        onRemoveTag={handleRemoveTag}
      />

      <TaskDependenciesSection
        task={task}
        mode={mode}
        showDependencyEditor={showDependencyEditor}
        setShowDependencyEditor={setShowDependencyEditor}
        dependencyFilter={dependencyFilter}
        setDependencyFilter={setDependencyFilter}
        tasks={tasks}
        onAddDependency={handleAddDependency}
        onRemoveDependency={handleRemoveDependency}
      />

      <TaskTagsSection
        task={task}
        showTagEditor={showTagEditor}
        newTagInput={newTagInput}
        setNewTagInput={setNewTagInput}
        allTags={allTags}
        onAddTag={handleAddTag}
        onRemoveTag={handleRemoveTag}
        getCurrentTaskTags={getCurrentTaskTags}
      />
    </div>
  );
}
