import React, { useState, useEffect, KeyboardEvent } from 'react';
import { deleteTask, saveExistingTask } from '../../api/tasks';
import { getTomorrow } from '../../utils/dates';

import { TaskDateDisplay } from './TaskDateDisplay';
import { TaskPriorityDisplay } from './TaskPriorityDisplay';
import { TaskStatusDisplay } from './TaskStatusDisplay';
import { TaskListOptionsMenu } from './TaskListOptionsMenu';
import { ConfirmDialog } from '../ui/ConfirmDialog';
import { Task } from '../../models/Task';
import { Tag } from '../../models/Tags';
import { Link } from 'react-router-dom';
import { PartialCard } from '../../models/Card';
import { BacklinkInput } from '../cards/BacklinkInput';
import { linkifyWithDefaultOptions } from '../../utils/strings';
import { TaskClosedIcon } from '../../assets/icons/TaskClosedIcon';
import { TaskOpenIcon } from '../../assets/icons/TaskOpenIcon';
import { TaskTagDisplay } from './TaskTagDisplay';
import { removeTagsFromTitle, parseTags } from '../../utils/tasks';
import { useTaskContext } from '../../contexts/TaskContext';
import { useDialogState } from '../../contexts/DialogStateContext';
import { useStatus } from '../../contexts/StatusContext';
import { useAuth } from '../../contexts/AuthContext';
import { format } from 'date-fns-tz';

interface TaskListItemProps {
  task: Task;
  onTagClick: (tag: string) => void;
  hideMatrixTags?: boolean;
  selectMode?: boolean;
  isSelected?: boolean;
  onSelect?: () => void;
  parentTask?: Task;
}

export function TaskListItem({
  task,
  onTagClick,
  hideMatrixTags = false,
  selectMode = false,
  isSelected = false,
  onSelect,
  parentTask,
}: TaskListItemProps) {
  const [editTitle, setEditTitle] = useState<boolean>(false);
  const [newTitle, setNewTitle] = useState<string>('');
  const [showCardLink, setShowCardLink] = useState<boolean>(false);
  const [showDeleteConfirm, setShowDeleteConfirm] = useState(false);
  const [showHistory, setShowHistory] = useState(false);
  const [tags, setTags] = useState<Tag[]>([]);
  const { setRefreshTasks, updateTask } = useTaskContext();
  const { setShowTaskDialog, setSelectedTaskId } = useDialogState();
  const { getDefaultStatus, getCompleteStatus } = useStatus();
  const { user } = useAuth();
  const userTimezone = user?.timezone || 'UTC';

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
      if ('error' in response) {
        // Rollback on error
        updateTask(task);
        setShowCardLink(true);
        console.error('Failed to link card to task:', response.error);
      }
    } catch (error) {
      // Rollback on network error
      updateTask(task);
      setShowCardLink(true);
      console.error('Failed to link card to task:', error);
    }
  }

  async function handleTitleEdit() {
    let editedTask = { ...task, title: newTitle };
    let response = await saveExistingTask(editedTask);
    if (!('error' in response)) {
      setRefreshTasks(true);
      setEditTitle(false);
      setNewTitle('');
    }
  }

  async function handleToggleComplete() {
    // Determine the target status based on current completion state
    const targetStatus = task.is_complete
      ? getDefaultStatus()
      : getCompleteStatus();

    if (!targetStatus) {
      console.error('Could not find appropriate status for toggle');
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
      if ('error' in response) {
        // Rollback on error
        updateTask(task);
        console.error('Failed to toggle task completion:', response.error);
      }
    } catch (error) {
      // Rollback on network error
      updateTask(task);
      console.error('Failed to toggle task completion:', error);
    }
  }

  async function handleRemoveTag(tagName: string) {
    // Remove # prefix if present
    const cleanTagName = tagName.replace(/^#/, '');
    const tagRegex = new RegExp(`\\n*#${cleanTagName}\\b`, 'g');

    const updatedTask = {
      ...task,
      title: task.title.replace(tagRegex, '').trim(),
      tags: task.tags.filter(
        (tag) => tag.name.replace(/^#/, '') !== cleanTagName,
      ),
    };

    // Optimistic update: update UI immediately
    updateTask(updatedTask);

    // Send update to server in background
    try {
      const response = await saveExistingTask(updatedTask);
      if ('error' in response) {
        // Rollback on error
        updateTask(task);
        console.error('Failed to remove tag:', response.error);
      }
    } catch (error) {
      // Rollback on network error
      updateTask(task);
      console.error('Failed to remove tag:', error);
    }
  }

  // Menu handlers
  const handleDelete = () => {
    setShowDeleteConfirm(true);
  };

  const confirmDelete = async () => {
    await deleteTask(task.id);
    setRefreshTasks(true);
  };

  const handleRefresh = () => {
    setRefreshTasks(true);
  };

  const handleClose = () => {
    // Close any open menus - the menu handles this itself
  };

  useEffect(() => {
    setTags(task.tags);
  }, [task]);

  return (
    <>
      <div className="flex items-center bg-white py-0.5 md:py-0">
        <div className="mr-2.5">
          {selectMode ? (
            <div className="min-w-[36px] md:min-w-[24px] min-h-[36px] md:min-h-[24px] flex items-center justify-center">
              <input
                type="checkbox"
                checked={isSelected}
                onChange={onSelect}
                className="w-5 h-5 cursor-pointer"
                onClick={(e) => e.stopPropagation()}
              />
            </div>
          ) : (
            <span
              onClick={handleToggleComplete}
              className="min-w-[36px] md:min-w-[24px] min-h-[36px] md:min-h-[24px] flex items-center justify-center cursor-pointer"
            >
              {task.is_complete ? <TaskClosedIcon /> : <TaskOpenIcon />}
            </span>
          )}
        </div>
        <div className="flex-grow min-w-0">
          {/* Parent task badge */}
          {parentTask && (
            <div className="mb-1">
              <button
                onClick={(e) => {
                  e.stopPropagation();
                  setSelectedTaskId(parentTask.id);
                  setShowTaskDialog(true);
                }}
                className="inline-flex items-center gap-1 px-2 py-0.5 text-xs text-gray-600 bg-gray-100 hover:bg-gray-200 rounded-full transition-colors focus-visible:ring-2 focus-visible:ring-blue-500"
                title={`Parent: ${parentTask.title}`}
                aria-label={`View parent task: ${parentTask.title}`}
              >
                <span>↳</span>
                <span className="max-w-[150px] truncate">
                  {parentTask.title || 'Untitled'}
                </span>
              </button>
            </div>
          )}
          <div className="whitespace-nowrap overflow-hidden text-ellipsis">
            <span
              onClick={handleTitleClick}
              className={task.is_complete ? 'line-through' : 'cursor-pointer'}
              dangerouslySetInnerHTML={{
                __html: linkifyWithDefaultOptions(
                  removeTagsFromTitle(task.title),
                ),
              }}
            />
          </div>
          <div className="flex text-sm inline-block">
            <TaskStatusDisplay
              task={task}
              setTask={(task: Task) => {}}
              saveOnChange={true}
            />
            <TaskDateDisplay
              task={task}
              setTask={(task: Task) => {}}
              saveOnChange={true}
            />
            <TaskPriorityDisplay
              task={task}
              setTask={(task: Task) => {}}
              saveOnChange={true}
            />
            {task.reminder_time && (
              <span
                className="text-xs mr-1"
                style={{
                  color: task.reminder_sent ? '#9ca3af' : '#3b82f6',
                  cursor: 'default',
                }}
                title={
                  task.reminder_sent
                    ? `Reminder sent: ${format(
                        new Date(task.reminder_time),
                        'MMM d, yyyy h:mm a',
                        { timeZone: userTimezone },
                      )}`
                    : `Reminder set for: ${format(
                        new Date(task.reminder_time),
                        'MMM d, yyyy h:mm a',
                        { timeZone: userTimezone },
                      )}`
                }
              >
                🔔
              </span>
            )}
            {task.blocked_by &&
              task.blocked_by.filter((bt) => !bt.is_complete).length > 0 && (
                <span
                  className="text-xs mr-1 px-1 rounded"
                  style={{
                    backgroundColor: '#fed7aa',
                    color: '#9a3412',
                    cursor: 'default',
                  }}
                  title={`Blocked by: ${task.blocked_by
                    .filter((bt) => !bt.is_complete)
                    .map((bt) => bt.title)
                    .join(', ')}`}
                >
                  🚧
                </span>
              )}
            <TaskTagDisplay
              task={task}
              tags={tags}
              onTagClick={onTagClick}
              onRemoveTag={handleRemoveTag}
              hideMatrixTags={hideMatrixTags}
            />
          </div>
        </div>
        <div className="ml-2.5">
          {task.card && task.card.id > 0 && (
            <Link
              to={`/app/card/${task.card.id}`}
              className="text-blue-600 hover:text-blue-800"
              style={{ textDecoration: 'none' }}
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
        <TaskListOptionsMenu
          task={task}
          showCardLink={showCardLink}
          setShowCardLink={setShowCardLink}
          onDelete={handleDelete}
          onToggleComplete={handleToggleComplete}
          onRefresh={handleRefresh}
          onClose={handleClose}
          showHistory={showHistory}
          onToggleHistory={() => setShowHistory(!showHistory)}
        />
      </div>
      <ConfirmDialog
        isOpen={showDeleteConfirm}
        onClose={() => setShowDeleteConfirm(false)}
        onConfirm={confirmDelete}
        title="Delete Task"
        message="Are you sure you want to delete this task? This cannot be undone."
        confirmText="Delete"
        cancelText="Cancel"
      />
    </>
  );
}
