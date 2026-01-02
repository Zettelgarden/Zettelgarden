import React, { useState, useEffect } from "react";
import { Dialog } from "@headlessui/react";
import { Task, TaskAuditEvent } from "../../models/Task";
import { TaskDateDisplay } from "./TaskDateDisplay";
import { TaskPriorityDisplay } from "./TaskPriorityDisplay";
import { TaskStatusDisplay } from "./TaskStatusDisplay";
import { TaskReminderDisplay } from "./TaskReminderDisplay";
import { BacklinkInput } from "../cards/BacklinkInput";
import { PartialCard } from "../../models/Card";
import { Link } from "react-router-dom";
import { TaskTagDisplay } from "./TaskTagDisplay";
import { saveExistingTask, deleteTask, fetchTaskAuditEvents, fetchTask } from "../../api/tasks";
import { useTaskContext } from "../../contexts/TaskContext";
import { Button } from "../../components/Button";
import { TaskListOptionsMenu } from "./TaskListOptionsMenu";
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

    // Title changes
    if (changeDetails.Title) {
      changes.push(`Changed title from "${changeDetails.Title.from}" to "${changeDetails.Title.to}"`);
    }

    // Completion status changes
    if (changeDetails.IsComplete) {
      changes.push(changeDetails.IsComplete.to ? "Marked as complete" : "Marked as incomplete");
    }

    // Scheduled date changes
    if (changeDetails.ScheduledDate) {
      const newDate = changeDetails.ScheduledDate.to ?
        format(new Date(changeDetails.ScheduledDate.to), 'MMM d, yyyy') :
        'none';
      changes.push(`Changed scheduled date to ${newDate}`);
    }

    // Card link changes
    if (changeDetails.CardPK) {
      if (changeDetails.CardPK.from === 0 && changeDetails.CardPK.to > 0) {
        changes.push(`Linked to card [${changeDetails.CardPK.to}]`);
      } else if (changeDetails.CardPK.from > 0 && changeDetails.CardPK.to === 0) {
        changes.push(`Unlinked from card [${changeDetails.CardPK.from}]`);
      } else {
        changes.push(`Changed linked card from [${changeDetails.CardPK.from}] to [${changeDetails.CardPK.to}]`);
      }
    }

    // Priority changes
    if (changeDetails.Priority) {
      const fromPriority = changeDetails.Priority.from ? `Priority ${changeDetails.Priority.from}` : "No Priority";
      const toPriority = changeDetails.Priority.to ? `Priority ${changeDetails.Priority.to}` : "No Priority";
      changes.push(`Changed priority from ${fromPriority} to ${toPriority}`);
    }

    // Reminder changes
    if (changeDetails.ReminderTime) {
      if (!changeDetails.ReminderTime.from && changeDetails.ReminderTime.to) {
        const newReminder = format(new Date(changeDetails.ReminderTime.to), 'MMM d, yyyy h:mm a');
        changes.push(`Set reminder for ${newReminder}`);
      } else if (changeDetails.ReminderTime.from && !changeDetails.ReminderTime.to) {
        changes.push(`Removed reminder`);
      } else if (changeDetails.ReminderTime.from && changeDetails.ReminderTime.to) {
        const newReminder = format(new Date(changeDetails.ReminderTime.to), 'MMM d, yyyy h:mm a');
        changes.push(`Changed reminder to ${newReminder}`);
      }
    }

    // If no specific changes were detected
    if (changes.length === 0) {
      return "Task updated";
    }

    return changes.join("; ");
  }

  return "Unknown change";
}

export function TaskDialog({ taskId, isOpen, onClose, onTagClick }: TaskDialogProps) {
  const [editedTask, setEditedTask] = useState<Task | null>(null);
  const [isEditing, setIsEditing] = useState(false);
  const [showCardLink, setShowCardLink] = useState<boolean>(false);
  const [auditEvents, setAuditEvents] = useState<TaskAuditEvent[]>([]);
  const [isLoading, setIsLoading] = useState(false);
  const { setRefreshTasks } = useTaskContext();

  useEffect(() => {
    if (taskId && isOpen) {
      setIsLoading(true);
      fetchTask(taskId.toString())
        .then(task => {
          // Convert date strings to Date objects
          const processedTask = {
            ...task,
            scheduled_date: task.scheduled_date ? new Date(task.scheduled_date) : null,
            dueDate: task.dueDate ? new Date(task.dueDate) : null,
            created_at: new Date(task.created_at),
            updated_at: new Date(task.updated_at),
            completed_at: task.completed_at ? new Date(task.completed_at) : null,
            reminder_time: task.reminder_time ? new Date(task.reminder_time) : null,
            tags: task.tags || []
          };
          setEditedTask(processedTask);
          return fetchTaskAuditEvents(taskId);
        })
        .then(events => setAuditEvents(events))
        .catch(error => console.error("Error fetching task:", error))
        .finally(() => setIsLoading(false));
    }
  }, [taskId, isOpen]);

  // Return null if task is not loaded yet
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

  const handleTitleChange = (e: React.ChangeEvent<HTMLInputElement>) => {
    if (editedTask) {
      setEditedTask({ ...editedTask, title: e.target.value });
    }
  };

  const handleSave = async () => {
    if (!editedTask) return;

    // Log the task to verify priority is included
    console.log("Saving edited task with priority:", editedTask.priority);

    const response = await saveExistingTask(editedTask);
    if (!("error" in response)) {
      setRefreshTasks(true);
      setIsEditing(false);
    }
  };

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

    // Make a complete copy of the task to ensure all properties are included
    const updatedTask = {
      ...editedTask,
      is_complete: !editedTask.is_complete
    };

    // Log the task to verify priority is included
    console.log("Toggling completion with priority:", updatedTask.priority);

    const response = await saveExistingTask(updatedTask);
    if (!("error" in response)) {
      setEditedTask(updatedTask);
      setRefreshTasks(true);
    }
  };

  const handleRemoveTag = async (tagName: string) => {
    if (!editedTask) return;

    // Remove # prefix if present
    const cleanTagName = tagName.replace(/^#/, '');
    const tagRegex = new RegExp(`\\n*#${cleanTagName}\\b`, 'g');

    const updatedTask = {
      ...editedTask,
      title: editedTask.title.replace(tagRegex, '').trim(),
      tags: editedTask.tags.filter(tag => tag.name.replace(/^#/, '') !== cleanTagName)
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
        <Dialog.Panel className={`w-full max-w-2xl transform overflow-hidden rounded-2xl p-6 shadow-xl transition-all ${
          editedTask.is_complete ? 'bg-green-50 border-2 border-green-300' : 'bg-white'
        }`}>
          <div className="flex justify-between items-start mb-4">
            <div className="flex items-center gap-4">
              <span
                onClick={handleToggleComplete}
                className="cursor-pointer hover:scale-110 transition-transform"
              >
                {editedTask.is_complete ? <TaskClosedIcon /> : <TaskOpenIcon />}
              </span>
              <Dialog.Title className={`text-lg font-medium leading-6 ${
                editedTask.is_complete ? 'text-green-800' : 'text-gray-900'
              }`}>
                {editedTask.is_complete ? '✓ Task Completed' : 'Task Details'}
              </Dialog.Title>
            </div>
            <TaskListOptionsMenu
              task={editedTask}
              tags={editedTask.tags}
              showCardLink={showCardLink}
              setShowCardLink={setShowCardLink}
            />
          </div>

          <div className="space-y-4">
            <div className="flex items-center justify-between">
              {isEditing ? (
                <input
                  type="text"
                  value={editedTask.title}
                  onChange={handleTitleChange}
                  className="w-full px-3 py-2 border rounded-md focus:outline-none focus:ring-2 focus:ring-blue-500 border-gray-300"
                  autoFocus
                />
              ) : (
                <div
                  className={`text-lg cursor-pointer hover:bg-gray-50 p-2 rounded flex-grow ${
                    editedTask.is_complete ? 'line-through text-gray-500' : ''
                  }`}
                  onClick={() => setIsEditing(true)}
                >
                  {editedTask.title}
                </div>
              )}
              {editedTask.card && editedTask.card.id > 0 && (
                <Link
                  to={`/app/card/${editedTask.card.id}`}
                  className="ml-2 text-blue-600 hover:text-blue-800"
                  style={{ textDecoration: "none" }}
                >
                  <span className="card-id">[{editedTask.card.card_id}]</span>
                </Link>
              )}
            </div>

            <div className="flex items-center gap-4">
              <TaskStatusDisplay
                task={editedTask}
                setTask={setEditedTask}
                saveOnChange={true}
              />
              <TaskDateDisplay
                task={editedTask}
                setTask={setEditedTask}
                saveOnChange={true}
              />
              <TaskPriorityDisplay
                task={editedTask}
                setTask={setEditedTask}
                saveOnChange={true}
              />
              <TaskReminderDisplay
                task={editedTask}
                setTask={setEditedTask}
                saveOnChange={true}
              />
              <TaskTagDisplay task={editedTask} tags={editedTask.tags} onTagClick={onTagClick} onRemoveTag={handleRemoveTag} />
            </div>

            {showCardLink && (
              <div className="border-t pt-4">
                <BacklinkInput addBacklink={handleBacklink} />
              </div>
            )}
          </div>

          <div className="mt-6 border-t pt-4">
            <h3 className="text-lg font-medium text-gray-900 mb-4">Task History</h3>
            <div className="space-y-3 max-h-[200px] overflow-y-auto">
              {auditEvents.length > 0 ? (
                auditEvents.map((event) => (
                  <div key={event.id} className="flex items-start space-x-3 text-sm hover:bg-gray-50 p-2 rounded">
                    <div className="text-gray-500 min-w-[120px] font-medium">
                      {format(event.created_at, 'MMM d, HH:mm')}
                    </div>
                    <div className="flex-grow text-gray-700">
                      {formatAuditEvent(event)}
                    </div>
                  </div>
                ))
              ) : (
                <div className="text-sm text-gray-500 text-center py-4">
                  No history available
                </div>
              )}
            </div>
          </div>

          <div className="mt-6 flex justify-between">
            <Button
              onClick={handleDelete}
              className="bg-red-500 hover:bg-red-600 text-white"
            >
              Delete Task
            </Button>
            <div className="flex gap-2">
              {isEditing && (
                <Button onClick={handleSave}>
                  Save Changes
                </Button>
              )}
              <Button onClick={onClose}>
                Close
              </Button>
            </div>
          </div>
        </Dialog.Panel>
      </div>
    </Dialog>
  );
}
