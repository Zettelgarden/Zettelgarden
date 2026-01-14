import React, { useState, useEffect, useCallback } from "react";
import { Dialog } from "@headlessui/react";
import { Task, TaskAuditEvent } from "../../models/Task";
import { PartialCard } from "../../models/Card";
import { Link } from "react-router-dom";
import { saveExistingTask, deleteTask, fetchTaskAuditEvents, fetchTask } from "../../api/tasks";
import { useTaskContext } from "../../contexts/TaskContext";
import { useStatus } from "../../contexts/StatusContext";
import { Button } from "../Button";
import { TaskListOptionsMenu } from "./TaskListOptionsMenu";
import { TaskForm } from "./TaskForm";
import { TaskAuditHistory } from "./TaskAuditHistory";
import { TaskClosedIcon } from "../../assets/icons/TaskClosedIcon";
import { TaskOpenIcon } from "../../assets/icons/TaskOpenIcon";

interface TaskDialogProps {
  taskId: number | null;
  isOpen: boolean;
  onClose: () => void;
}

function LoadingSpinner() {
  return (
    <div className="flex items-center justify-center py-8">
      <div className="animate-spin rounded-full h-8 w-8 border-b-2 border-gray-900" />
    </div>
  );
}

export function TaskDialog({ taskId, isOpen, onClose }: TaskDialogProps) {
  const [editedTask, setEditedTask] = useState<Task | null>(null);
  const [showCardLink, setShowCardLink] = useState<boolean>(false);
  const [auditEvents, setAuditEvents] = useState<TaskAuditEvent[]>([]);
  const [isLoading, setIsLoading] = useState(false);
  const { setRefreshTasks } = useTaskContext();
  const { getDefaultStatus, getCompleteStatus } = useStatus();

  useEffect(() => {
    if (taskId && isOpen) {
      setIsLoading(true);
      fetchTask(taskId.toString())
        .then((task) => {
          setEditedTask(task);
          return fetchTaskAuditEvents(taskId);
        })
        .then((events) => setAuditEvents(events))
        .catch((error) => console.error("Error fetching task:", error))
        .finally(() => setIsLoading(false));
    }
  }, [taskId, isOpen]);

  const handleDelete = useCallback(async () => {
    if (!editedTask) return;
    if (window.confirm("Are you sure you want to delete this task? This cannot be undone.")) {
      await deleteTask(editedTask.id);
      setRefreshTasks(true);
      onClose();
    }
  }, [editedTask, setRefreshTasks, onClose]);

  const handleBacklink = useCallback(async (card: PartialCard) => {
    if (!editedTask) return;

    const updatedTask = { ...editedTask, card_pk: card.id };
    const response = await saveExistingTask(updatedTask);
    if (!("error" in response)) {
      setEditedTask(updatedTask);
      setRefreshTasks(true);
      setShowCardLink(false);
    }
  }, [editedTask, setRefreshTasks]);

  const handleToggleComplete = useCallback(async () => {
    if (!editedTask) return;

    const targetStatus = editedTask.is_complete ? getDefaultStatus() : getCompleteStatus();

    if (!targetStatus) {
      console.error("Could not find appropriate status for toggle");
      return;
    }

    const updatedTask = {
      ...editedTask,
      status: targetStatus.name,
      is_complete: targetStatus.is_complete_state,
    };

    const response = await saveExistingTask(updatedTask);
    if (!("error" in response)) {
      setEditedTask(updatedTask);
      setRefreshTasks(true);
    }
  }, [editedTask, getDefaultStatus, getCompleteStatus, setRefreshTasks]);

  if (!editedTask || isLoading) {
    return (
      <Dialog open={isOpen} onClose={onClose} className="relative z-50">
        <div className="fixed inset-0 bg-black/30" aria-hidden="true" />
        <div className="fixed inset-0 flex items-center justify-center p-4">
          <Dialog.Panel className="w-full max-w-2xl transform overflow-hidden rounded-2xl bg-white p-6 shadow-xl transition-all">
            <LoadingSpinner />
          </Dialog.Panel>
        </div>
      </Dialog>
    );
  }

  return (
    <Dialog open={isOpen} onClose={onClose} className="relative z-50">
      <div className="fixed inset-0 bg-black/30" aria-hidden="true" />

      <div className="fixed inset-0 flex items-center justify-center p-4">
        <Dialog.Panel
          className={`w-full max-w-2xl transform overflow-hidden rounded-2xl py-6 shadow-xl transition-all max-h-[80vh] flex flex-col ${
            editedTask.is_complete ? "bg-green-50 border-2 border-green-300" : "bg-white"
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
              <Dialog.Title
                className={`text-lg font-medium leading-6 ${
                  editedTask.is_complete ? "text-green-800" : "text-gray-900"
                }`}
              >
                {editedTask.is_complete ? "Task Completed" : "Task Details"}
              </Dialog.Title>
              {editedTask.card && editedTask.card.id > 0 && (
                <Link
                  to={`/app/card/${editedTask.card.id}`}
                  className="text-blue-600 hover:text-blue-800"
                  style={{ textDecoration: "none" }}
                >
                  <span className="card-id">[{editedTask.card.card_id}]</span>
                </Link>
              )}
            </div>
            <TaskListOptionsMenu
              task={editedTask}
              showCardLink={showCardLink}
              setShowCardLink={setShowCardLink}
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

              <TaskAuditHistory events={auditEvents} />
            </div>
          </div>

          {/* Footer */}
          <div className="px-6 mt-6 flex justify-between">
            <Button onClick={handleDelete} className="bg-red-500 hover:bg-red-600 text-white">
              Delete Task
            </Button>
            <Button onClick={onClose}>Close</Button>
          </div>
        </Dialog.Panel>
      </div>
    </Dialog>
  );
}
