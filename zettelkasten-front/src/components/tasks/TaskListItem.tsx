import React, { useState, useEffect, KeyboardEvent } from "react";
import { deleteTask, saveExistingTask } from "../../api/tasks";
import { getTomorrow } from "../../utils/dates";

import { TaskDateDisplay } from "./TaskDateDisplay";
import { TaskPriorityDisplay } from "./TaskPriorityDisplay";
import { TaskStatusDisplay } from "./TaskStatusDisplay";
import { Task } from "../../models/Task";
import { Tag } from "../../models/Tags";
import { Link } from "react-router-dom";
import { PartialCard } from "../../models/Card";
import { BacklinkInput } from "../cards/BacklinkInput";
import { linkifyWithDefaultOptions } from "../../utils/strings";
import { TaskClosedIcon } from "../../assets/icons/TaskClosedIcon";
import { TaskOpenIcon } from "../../assets/icons/TaskOpenIcon";
import { TaskTagDisplay } from "./TaskTagDisplay";
import { removeTagsFromTitle, parseTags } from "../../utils/tasks";
import { useTaskContext } from "../../contexts/TaskContext";
import { useShortcutContext } from "../../contexts/ShortcutContext";
import { useStatus } from "../../contexts/StatusContext";

interface TaskListItemProps {
  task: Task;
  onTagClick: (tag: string) => void;
  hideMatrixTags?: boolean;
  selectMode?: boolean;
  isSelected?: boolean;
  onSelect?: () => void;
}

export function TaskListItem({
  task,
  onTagClick,
  hideMatrixTags = false,
  selectMode = false,
  isSelected = false,
  onSelect,
}: TaskListItemProps) {
  const [editTitle, setEditTitle] = useState<boolean>(false);
  const [newTitle, setNewTitle] = useState<string>("");
  const [showCardLink, setShowCardLink] = useState<boolean>(false);
  const [tags, setTags] = useState<Tag[]>([]);
  const { setRefreshTasks, updateTask } = useTaskContext();
  const { setShowTaskDialog, setSelectedTaskId } = useShortcutContext();
  const { getDefaultStatus, getCompleteStatus } = useStatus();

  async function handleTitleClick() {
    setSelectedTaskId(task.id);
    setShowTaskDialog(true);
  }

  async function handleBacklink(card: PartialCard) {
    const editedTask = { ...task, card_pk: card.id, card: card };

    // Optimistic update: update UI immediately
    updateTask(editedTask);
    setShowCardLink(false);

    // Send update to server in background
    try {
      const response = await saveExistingTask(editedTask);
      if ("error" in response) {
        // Rollback on error
        updateTask(task);
        setShowCardLink(true);
        console.error("Failed to link card to task:", response.error);
      }
    } catch (error) {
      // Rollback on network error
      updateTask(task);
      setShowCardLink(true);
      console.error("Failed to link card to task:", error);
    }
  }

  async function handleTitleEdit() {
    let editedTask = { ...task, title: newTitle };
    let response = await saveExistingTask(editedTask);
    if (!("error" in response)) {
      setRefreshTasks(true);
      setEditTitle(false);
      setNewTitle("");
    }
  }

  async function handleToggleComplete() {
    // Determine the target status based on current completion state
    const targetStatus = task.is_complete ? getDefaultStatus() : getCompleteStatus();

    if (!targetStatus) {
      console.error("Could not find appropriate status for toggle");
      return;
    }

    const editedTask = {
      ...task,
      status: targetStatus.name,
      is_complete: targetStatus.is_complete_state,
    };

    // Optimistic update: update UI immediately
    updateTask(editedTask);

    // Send update to server in background
    try {
      const response = await saveExistingTask(editedTask);
      if ("error" in response) {
        // Rollback on error
        updateTask(task);
        console.error("Failed to toggle task completion:", response.error);
      }
    } catch (error) {
      // Rollback on network error
      updateTask(task);
      console.error("Failed to toggle task completion:", error);
    }
  }

  async function handleRemoveTag(tagName: string) {
    // Remove # prefix if present
    const cleanTagName = tagName.replace(/^#/, '');
    const tagRegex = new RegExp(`\\n*#${cleanTagName}\\b`, 'g');

    const updatedTask = {
      ...task,
      title: task.title.replace(tagRegex, '').trim(),
      tags: task.tags.filter(tag => tag.name.replace(/^#/, '') !== cleanTagName)
    };

    // Optimistic update: update UI immediately
    updateTask(updatedTask);

    // Send update to server in background
    try {
      const response = await saveExistingTask(updatedTask);
      if ("error" in response) {
        // Rollback on error
        updateTask(task);
        console.error("Failed to remove tag:", response.error);
      }
    } catch (error) {
      // Rollback on network error
      updateTask(task);
      console.error("Failed to remove tag:", error);
    }
  }

  useEffect(() => {
    setTags(task.tags);
  }, [task]);

  return (
    <div className="task-list-item">
      <div className="task-list-item-checkbox">
        {selectMode ? (
          <input
            type="checkbox"
            checked={isSelected}
            onChange={onSelect}
            className="w-5 h-5 cursor-pointer"
            onClick={(e) => e.stopPropagation()}
          />
        ) : (
          <span onClick={handleToggleComplete}>
            {task.is_complete ? <TaskClosedIcon /> : <TaskOpenIcon />}
          </span>
        )}
      </div>
      <div className="task-list-item-middle-container">
        <div className="task-list-item-title">
          <span
            onClick={handleTitleClick}
            className={task.is_complete ? "task-completed" : "task-title"}
            dangerouslySetInnerHTML={{
              __html: linkifyWithDefaultOptions(
                removeTagsFromTitle(task.title),
              ),
            }}
          />
        </div>
        <div className="task-list-item-details inline-block">
          <TaskStatusDisplay
            task={task}
            setTask={(task: Task) => { }}
            saveOnChange={true}
          />
          <TaskDateDisplay
            task={task}
            setTask={(task: Task) => { }}
            saveOnChange={true}
          />
          <TaskPriorityDisplay
            task={task}
            setTask={(task: Task) => { }}
            saveOnChange={true}
          />
          {task.reminder_time && (
            <span
              className="text-xs mr-1"
              style={{
                color: task.reminder_sent ? '#9ca3af' : '#3b82f6',
                cursor: 'default'
              }}
              title={task.reminder_sent
                ? `Reminder sent: ${new Date(task.reminder_time).toLocaleString()}`
                : `Reminder set for: ${new Date(task.reminder_time).toLocaleString()}`
              }
            >
              🔔
            </span>
          )}
          <TaskTagDisplay task={task} tags={tags} onTagClick={onTagClick} onRemoveTag={handleRemoveTag} hideMatrixTags={hideMatrixTags} />
        </div>
      </div>
      <div className="task-list-item-card">
        {task.card && task.card.id > 0 && (
          <Link
            to={`/app/card/${task.card.id}`}
            style={{ textDecoration: "none", color: "inherit" }}
          >
            <span className="card-id">[{task.card.card_id}]</span>
          </Link>
        )}
        {!task.card ||
          (task.card.id == 0 && (
            <div>
              {showCardLink && <BacklinkInput addBacklink={handleBacklink} />}
            </div>
          ))}
      </div>
      <button onClick={() => {
        setSelectedTaskId(task.id);
        setShowTaskDialog(true);
      }} className="menu-button">
        ⋮
      </button>
    </div>
  );
}
