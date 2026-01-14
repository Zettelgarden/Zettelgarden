import React, { useState, useEffect } from "react";
import { Dialog } from "@headlessui/react";
import { Task, TaskAuditEvent } from "../../models/Task";
import { PartialCard } from "../../models/Card";
import { Link } from "react-router-dom";
import { saveExistingTask, deleteTask, fetchTaskAuditEvents, fetchTask } from "../../api/tasks";
import { useTaskContext } from "../../contexts/TaskContext";
import { useStatus } from "../../contexts/StatusContext";
import { Button } from "../../components/Button";
import { TaskListOptionsMenu } from "./TaskListOptionsMenu";
import { TaskForm } from "./TaskForm";
import { format } from "date-fns";
import { TaskClosedIcon } from "../../assets/icons/TaskClosedIcon";
import { TaskOpenIcon } from "../../assets/icons/TaskOpenIcon";

interface TaskDialogProps {
  taskId: number | null;
  isOpen: boolean;
  onClose: () => void;
  onTagClick: (tag: string) => void;
}

function formatAuditEvent(event: TaskAuditEvent): string {
  if (event.action === "create") {
    return "Task created";
  }

  if (event.action === "delete") {
    return "Task deleted";
  }

  if (event.action === "update" && event.details.change_type === "update") {
    const changes: string[] = [];
    const changeDetails = event.details.changes;

    if (changeDetails.Title) {
      changes.push(`Changed title from "${changeDetails.Title.from}" to "${changeDetails.Title.to}"`);
    }

    if (changeDetails.IsComplete) {
      changes.push(changeDetails.IsComplete.to ? "Marked as complete" : "Marked as incomplete");
    }

    if (changeDetails.ScheduledDate) {
      const newDate = changeDetails.ScheduledDate.to
        ? format(new Date(changeDetails.ScheduledDate.to), "MMM d, yyyy")
        : "none";
      changes.push(`Changed scheduled date to ${newDate}`);
    }

    if (changeDetails.CardPK) {
      if (changeDetails.CardPK.from === 0 && changeDetails.CardPK.to > 0) {
        changes.push(`Linked to card [${changeDetails.CardPK.to}]`);
      } else if (changeDetails.CardPK.from > 0 && changeDetails.CardPK.to === 0) {
        changes.push(`Unlinked from card [${changeDetails.CardPK.from}]`);
      } else {
        changes.push(`Changed linked card from [${changeDetails.CardPK.from}] to [${changeDetails.CardPK.to}]`);
      }
    }

    if (changeDetails.Priority) {
      const fromPriority = changeDetails.Priority.from ? `Priority ${changeDetails.Priority.from}` : "No Priority";
      const toPriority = changeDetails.Priority.to ? `Priority ${changeDetails.Priority.to}` : "No Priority";
      changes.push(`Changed priority from ${fromPriority} to ${toPriority}`);
    }

    if (changeDetails.Description) {
      if (!changeDetails.Description.from && changeDetails.Description.to) {
        changes.push(`Added description`);
      } else if (changeDetails.Description.from && !changeDetails.Description.to) {
        changes.push(`Removed description`);
      } else {
        changes.push(`Updated description`);
      }
    }

    if (changeDetails.ReminderTime) {
      if (!changeDetails.ReminderTime.from && changeDetails.ReminderTime.to) {
        const newReminder = format(new Date(changeDetails.ReminderTime.to), "MMM d, yyyy h:mm a");
        changes.push(`Set reminder for ${newReminder}`);
      } else if (changeDetails.ReminderTime.from && !changeDetails.ReminderTime.to) {
        changes.push(`Removed reminder`);
      } else if (changeDetails.ReminderTime.from && changeDetails.ReminderTime.to) {
        const newReminder = format(new Date(changeDetails.ReminderTime.to), "MMM d, yyyy h:mm a");
        changes.push(`Changed reminder to ${newReminder}`);
      }
    }

    if (changes.length === 0) {
      return "Task updated";
    }

    return changes.join("; ");
  }

  return "Unknown change";
}

export function TaskDialog({ taskId, isOpen, onClose, onTagClick }: TaskDialogProps) {
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

  if (!editedTask || isLoading) {
    return (
      <Dialog open={isOpen} onClose={onClose} className="relative z-50">
        <div className="fixed inset-0 bg-black/30" aria-hidden="true" />
        <div className="fixed inset-0 flex items-center justify-center p-4">
          <Dialog.Panel className="w-full max-w-2xl transform overflow-hidden rounded-2xl bg-white p-6 shadow-xl transition-all">
            <div className="flex items-center justify-center py-8">
              <div className="animate-spin rounded-full h-8 w-8 border-b-2 border-gray-900"></div>
            </div>
          </Dialog.Panel>
        </div>
      </Dialog>
    );
  }

  const handleDelete = async () => {
    if (!editedTask) return;
    if (window.confirm("Are you sure you want to delete this task? This cannot be undone.")) {
      await deleteTask(editedTask.id);
      setRefreshTasks(true);
      onClose();
    }
  };

  const handleBacklink = async (card: PartialCard) => {
    if (!editedTask) return;

    const updatedTask = { ...editedTask, card_pk: card.id };
    const response = await saveExistingTask(updatedTask);
    if (!("error" in response)) {
      setEditedTask(updatedTask);
      setRefreshTasks(true);
      setShowCardLink(false);
    }
  };

  const handleToggleComplete = async () => {
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
  };

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

              {/* Audit History */}
              <div className="mt-6 border-t pt-4">
                <h3 className="text-lg font-medium text-gray-900 mb-4">Task History</h3>
                <div className="space-y-3 max-h-[200px] overflow-y-auto">
                  {auditEvents.length > 0 ? (
                    auditEvents.map((event) => (
                      <div
                        key={event.id}
                        className="flex items-start space-x-3 text-sm hover:bg-gray-50 p-2 rounded"
                      >
                        <div className="text-gray-500 min-w-[120px] font-medium">
                          {format(event.created_at, "MMM d, HH:mm")}
                        </div>
                        <div className="flex-grow text-gray-700">{formatAuditEvent(event)}</div>
                      </div>
                    ))
                  ) : (
                    <div className="text-sm text-gray-500 text-center py-4">No history available</div>
                  )}
                </div>
              </div>
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
