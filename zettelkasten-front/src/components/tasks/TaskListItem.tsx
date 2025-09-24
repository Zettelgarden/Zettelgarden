import React, { useState, useEffect, KeyboardEvent } from "react";
import { deleteTask, saveExistingTask } from "../../api/tasks";
import { getTomorrow } from "../../utils/dates";

import { TaskDateDisplay } from "./TaskDateDisplay";
import { TaskPriorityDisplay } from "./TaskPriorityDisplay";
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

interface TaskListItemProps {
  task: Task;
  onTagClick: (tag: string) => void;
}

export function TaskListItem({
  task,
  onTagClick,
}: TaskListItemProps) {
  const [editTitle, setEditTitle] = useState<boolean>(false);
  const [newTitle, setNewTitle] = useState<string>("");
  const [showCardLink, setShowCardLink] = useState<boolean>(false);
  const [tags, setTags] = useState<Tag[]>([]);
  const { setRefreshTasks } = useTaskContext();
  const { setShowTaskDialog, setSelectedTaskId } = useShortcutContext();

  async function handleTitleClick() {
    setSelectedTaskId(task.id);
    setShowTaskDialog(true);
  }

  async function handleBacklink(card: PartialCard) {
    let editedTask = { ...task, card_pk: card.id };
    let response = await saveExistingTask(editedTask);
    if (!("error" in response)) {
      setRefreshTasks(true);
      setShowCardLink(false);
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
    let editedTask = { ...task, is_complete: task.is_complete ? false : true };
    let response = await saveExistingTask(editedTask);
    if (!("error" in response)) {
      setRefreshTasks(true);
    }
  }

  useEffect(() => {
    setTags(task.tags);
  }, [task]);

  return (
    <div className="task-list-item">
      <div className="task-list-item-checkbox">
        <span onClick={handleToggleComplete}>
          {task.is_complete ? <TaskClosedIcon /> : <TaskOpenIcon />}
        </span>
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
          <TaskTagDisplay task={task} tags={tags} onTagClick={onTagClick} />
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
