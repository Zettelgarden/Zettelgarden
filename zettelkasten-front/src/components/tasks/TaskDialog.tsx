import React, { useState, useEffect, useCallback } from 'react';
import { Modal } from '../ui/Modal';
import { Task, TaskAuditEvent } from '../../models/Task';
import { PartialCard } from '../../models/Card';
import { Link } from 'react-router-dom';
import { Spinner } from '../ui/Spinner';
import {
  saveExistingTask,
  deleteTask,
  fetchTaskAuditEvents,
  fetchTask,
  createSubtask,
} from '../../api/tasks';
import { useTaskContext } from '../../contexts/TaskContext';
import { useStatus } from '../../contexts/StatusContext';
import { useAuth } from '../../contexts/AuthContext';
import { Button } from '../Button';
import { TaskListOptionsMenu } from './TaskListOptionsMenu';
import { TaskForm } from './TaskForm';
import { TaskAuditHistory } from './TaskAuditHistory';
import { ConfirmDialog } from '../ui/ConfirmDialog';
import { TaskClosedIcon } from '../../assets/icons/TaskClosedIcon';
import { TaskOpenIcon } from '../../assets/icons/TaskOpenIcon';
import { getToday, getTomorrow } from '../../utils/dates';
import { TaskSubtasksSection } from './TaskSubtasksSection';
import { TaskCompletionWarningDialog } from './TaskCompletionWarningDialog';

interface TaskDialogProps {
  taskId: number | null;
  isOpen: boolean;
  onClose: () => void;
}

function LoadingSpinner() {
  return (
    <div className="flex items-center justify-center py-8">
      <Spinner size="lg" className="text-gray-900" />
    </div>
  );
}

export function TaskDialog({ taskId, isOpen, onClose }: TaskDialogProps) {
  const [editedTask, setEditedTask] = useState<Task | null>(null);
  const [showCardLink, setShowCardLink] = useState<boolean>(false);
  const [auditEvents, setAuditEvents] = useState<TaskAuditEvent[]>([]);
  const [showDeleteConfirm, setShowDeleteConfirm] = useState(false);
  const [showHistory, setShowHistory] = useState(false);
  const [isLoading, setIsLoading] = useState(false);
  const [showCompletionWarning, setShowCompletionWarning] = useState(false);
  const [completionError, setCompletionError] = useState<{
    incomplete_count: number;
  } | null>(null);
  const { setRefreshTasks } = useTaskContext();
  const { getDefaultStatus, getCompleteStatus } = useStatus();
  const { user } = useAuth();
  const userTimezone = user?.timezone || 'UTC';

  useEffect(() => {
    if (taskId && isOpen) {
      setIsLoading(true);
      fetchTask(taskId.toString())
        .then((task) => {
          setEditedTask(task);
          return fetchTaskAuditEvents(taskId);
        })
        .then((events) => setAuditEvents(events))
        .catch((error) => console.error('Error fetching task:', error))
        .finally(() => setIsLoading(false));
    }
  }, [taskId, isOpen]);

  const handleDelete = useCallback(async () => {
    if (!editedTask) return;
    setShowDeleteConfirm(true);
  }, [editedTask]);

  const confirmDelete = useCallback(async () => {
    if (!editedTask) return;
    await deleteTask(editedTask.id);
    setRefreshTasks(true);
    onClose();
  }, [editedTask, setRefreshTasks, onClose]);

  const handleBacklink = useCallback(
    async (card: PartialCard) => {
      if (!editedTask) return;

      const updatedTask = { ...editedTask, card_pk: card.id };
      const response = await saveExistingTask(updatedTask);
      if (!('error' in response)) {
        setEditedTask(updatedTask);
        setRefreshTasks(true);
        setShowCardLink(false);
      }
    },
    [editedTask, setRefreshTasks],
  );

  const handleToggleComplete = useCallback(async () => {
    if (!editedTask) return;

    // If trying to complete, check for incomplete subtasks
    if (!editedTask.is_complete) {
      const incompleteSubtasks = (editedTask.subtasks || []).filter(
        (s) => !s.is_complete,
      );
      if (incompleteSubtasks.length > 0) {
        setCompletionError({ incomplete_count: incompleteSubtasks.length });
        setShowCompletionWarning(true);
        return;
      }
    }

    const targetStatus = editedTask.is_complete
      ? getDefaultStatus()
      : getCompleteStatus();

    if (!targetStatus) {
      console.error('Could not find appropriate status for toggle');
      return;
    }

    const updatedTask = {
      ...editedTask,
      status: targetStatus.name,
      is_complete: targetStatus.is_complete_state,
    };

    const response = await saveExistingTask(updatedTask);
    if (!('error' in response)) {
      setEditedTask(updatedTask);
      setRefreshTasks(true);
    }
  }, [editedTask, getDefaultStatus, getCompleteStatus, setRefreshTasks]);

  const handleForceComplete = useCallback(async () => {
    if (!editedTask) return;

    const completeStatus = getCompleteStatus();
    if (!completeStatus) return;

    const updatedTask = {
      ...editedTask,
      status: completeStatus.name,
      is_complete: true,
    };

    try {
      const response = await saveExistingTask(updatedTask);
      if (!('error' in response)) {
        setEditedTask(updatedTask);
        setRefreshTasks(true);
        setShowCompletionWarning(false);
        setCompletionError(null);
      }
    } catch (error) {
      console.error('Failed to force complete:', error);
    }
  }, [editedTask, getCompleteStatus, setRefreshTasks]);

  // Subtask handlers
  const handleCreateSubtask = useCallback(
    async (title: string) => {
      if (!editedTask) return;

      await createSubtask(editedTask.id, {
        title,
        user_id: editedTask.user_id,
      });
      // Refresh task to get updated subtasks
      const updated = await fetchTask(editedTask.id.toString());
      setEditedTask(updated);
      setRefreshTasks(true);
    },
    [editedTask, setRefreshTasks],
  );

  const handleToggleSubtask = useCallback(
    async (subtaskId: number, isComplete: boolean) => {
      if (!editedTask) return;

      const subtask = editedTask.subtasks?.find((s) => s.id === subtaskId);
      if (!subtask) return;

      const completeStatus = getCompleteStatus();
      const defaultStatus = getDefaultStatus();
      const newStatus = isComplete
        ? completeStatus?.name || 'done'
        : defaultStatus?.name || 'todo';

      await saveExistingTask({
        ...subtask,
        is_complete: isComplete,
        status: newStatus,
      });

      // Refresh task
      const updated = await fetchTask(editedTask.id.toString());
      setEditedTask(updated);
      setRefreshTasks(true);
    },
    [editedTask, getCompleteStatus, getDefaultStatus, setRefreshTasks],
  );

  const handleDeleteSubtask = useCallback(
    async (subtaskId: number) => {
      if (!editedTask) return;

      await deleteTask(subtaskId);

      // Refresh task
      const updated = await fetchTask(editedTask.id.toString());
      setEditedTask(updated);
      setRefreshTasks(true);
    },
    [editedTask, setRefreshTasks],
  );

  if (!editedTask || isLoading) {
    return (
      <Modal open={isOpen} onClose={onClose} size="2xl" dialogClassName="z-50">
        <LoadingSpinner />
      </Modal>
    );
  }

  return (
    <>
      <Modal
        open={isOpen}
        onClose={onClose}
        size="2xl"
        dialogClassName="z-50"
        className={`!px-0 max-h-[80vh] flex flex-col ${
          editedTask.is_complete
            ? 'border-2 border-green-200'
            : 'border-2 border-transparent'
        }`}
      >
        {/* Header */}
        <div className="px-6 flex justify-between items-start mb-4">
          <div className="flex items-center gap-4">
            <span
              onClick={handleToggleComplete}
              className="cursor-pointer hover:scale-110 transition-transform"
            >
              {editedTask.is_complete ? <TaskClosedIcon /> : <TaskOpenIcon />}
            </span>
            <h3
              className={`text-lg font-medium leading-6 ${
                editedTask.is_complete ? 'text-green-700' : 'text-gray-900'
              }`}
            >
              {editedTask.is_complete ? 'Task Completed' : 'Task Details'}
            </h3>
            {editedTask.card && editedTask.card.id > 0 && (
              <Link
                to={`/app/card/${editedTask.card.id}`}
                className="text-blue-600 hover:text-blue-800"
                style={{ textDecoration: 'none' }}
              >
                <span className="card-id">[{editedTask.card.card_id}]</span>
              </Link>
            )}
          </div>
          <TaskListOptionsMenu
            task={editedTask}
            showCardLink={showCardLink}
            setShowCardLink={setShowCardLink}
            onDelete={handleDelete}
            onToggleComplete={handleToggleComplete}
            onRefresh={() => setRefreshTasks(true)}
            onClose={onClose}
            showHistory={showHistory}
            onToggleHistory={() => setShowHistory(!showHistory)}
          />
        </div>

        {/* Scrollable Content */}
        <div className="flex-1 overflow-y-auto px-6">
          <div className="space-y-4">
            <TaskForm
              task={editedTask}
              setTask={setEditedTask}
              mode="edit"
              saveOnChange={true}
              showCardLink={showCardLink}
              onBacklink={handleBacklink}
            />

            {/* Subtasks Section */}
            {editedTask.id > 0 && (
              <div className="border-t border-gray-200 pt-4">
                <TaskSubtasksSection
                  task={editedTask}
                  onCreateSubtask={handleCreateSubtask}
                  onToggleSubtask={handleToggleSubtask}
                  onDeleteSubtask={handleDeleteSubtask}
                  disabled={editedTask.is_complete}
                />
              </div>
            )}

            {showHistory && <TaskAuditHistory events={auditEvents} />}
          </div>
        </div>

        {/* Footer */}
        <div className="px-6 mt-6 flex justify-end">
          <Button onClick={onClose}>Close</Button>
        </div>
      </Modal>

      <ConfirmDialog
        isOpen={showDeleteConfirm}
        onClose={() => setShowDeleteConfirm(false)}
        onConfirm={confirmDelete}
        title="Delete Task"
        message="Are you sure you want to delete this task? This cannot be undone."
        confirmText="Delete"
        cancelText="Cancel"
      />

      {/* Completion Warning Dialog */}
      {editedTask && (
        <TaskCompletionWarningDialog
          visible={showCompletionWarning}
          task={editedTask}
          incompleteCount={completionError?.incomplete_count || 0}
          onForceComplete={handleForceComplete}
          onCancel={() => {
            setShowCompletionWarning(false);
            setCompletionError(null);
          }}
        />
      )}
    </>
  );
}
