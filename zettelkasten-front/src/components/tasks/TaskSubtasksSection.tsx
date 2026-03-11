import React, { useState } from "react";
import { Task } from "../../models/Task";
import { ConfirmDialog } from "./ConfirmDialog";
import { useToast } from "../toast/ToastContext";

interface TaskSubtasksSectionProps {
  task: Task;
  onCreateSubtask: (title: string) => Promise<void>;
  onToggleSubtask: (subtaskId: number, isComplete: boolean) => Promise<void>;
  onDeleteSubtask: (subtaskId: number) => Promise<void>;
  disabled?: boolean;
}

export function TaskSubtasksSection({
  task,
  onCreateSubtask,
  onToggleSubtask,
  onDeleteSubtask,
  disabled = false,
}: TaskSubtasksSectionProps) {
  const [newSubtaskTitle, setNewSubtaskTitle] = useState("");
  const [isAdding, setIsAdding] = useState(false);
  const [loadingId, setLoadingId] = useState<number | null>(null);
  const [showAddInput, setShowAddInput] = useState(false);
  const [showDeleteConfirm, setShowDeleteConfirm] = useState(false);
  const [subtaskToDelete, setSubtaskToDelete] = useState<number | null>(null);
  const { showToast } = useToast();

  const subtasks = task.subtasks || [];
  const completeCount = subtasks.filter((s) => s.is_complete).length;

  const handleAddSubtask = async () => {
    if (!newSubtaskTitle.trim()) return;

    setIsAdding(true);
    try {
      await onCreateSubtask(newSubtaskTitle.trim());
      setNewSubtaskTitle("");
      setShowAddInput(false);
    } catch (error) {
      console.error("Failed to create subtask:", error);
      showToast("error", "Failed to Create Subtask", "Could not create the subtask. Please try again.");
    } finally {
      setIsAdding(false);
    }
  };

  const handleToggle = async (subtaskId: number, currentComplete: boolean) => {
    setLoadingId(subtaskId);
    try {
      await onToggleSubtask(subtaskId, !currentComplete);
    } catch (error) {
      console.error("Failed to toggle subtask:", error);
      showToast("error", "Failed to Update Subtask", "Could not update the subtask status. Please try again.");
    } finally {
      setLoadingId(null);
    }
  };

  const handleDeleteClick = (subtaskId: number) => {
    setSubtaskToDelete(subtaskId);
    setShowDeleteConfirm(true);
  };

  const confirmDelete = async () => {
    if (subtaskToDelete === null) return;
    
    setLoadingId(subtaskToDelete);
    try {
      await onDeleteSubtask(subtaskToDelete);
      setShowDeleteConfirm(false);
      setSubtaskToDelete(null);
    } catch (error) {
      console.error("Failed to delete subtask:", error);
      showToast("error", "Failed to Delete Subtask", "Could not delete the subtask. Please try again.");
    } finally {
      setLoadingId(null);
    }
  };

  const handleKeyDown = (e: React.KeyboardEvent) => {
    if (e.key === "Enter") {
      e.preventDefault();
      handleAddSubtask();
    }
    if (e.key === "Escape") {
      setNewSubtaskTitle("");
      setShowAddInput(false);
    }
  };

  return (
    <>
      <div className="space-y-2">
        {/* Header */}
        <div className="flex items-center justify-between">
          <h4 className="text-sm font-medium text-gray-700">
            Subtasks ({completeCount}/{subtasks.length})
          </h4>
          {!disabled && !showAddInput && (
            <button
              type="button"
              onClick={() => setShowAddInput(true)}
              className="text-xs text-blue-600 hover:text-blue-800 font-medium"
            >
              + Add Subtask
            </button>
          )}
        </div>

        {/* Subtask List */}
        {subtasks.length > 0 ? (
          <ul className="space-y-1 border border-gray-200 rounded-md divide-y divide-gray-100">
            {subtasks.map((subtask) => (
              <li
                key={subtask.id}
                className="flex items-center gap-2 p-2 hover:bg-gray-50 group"
              >
                <input
                  type="checkbox"
                  checked={subtask.is_complete}
                  onChange={() => handleToggle(subtask.id, subtask.is_complete)}
                  disabled={disabled || loadingId === subtask.id}
                  className="w-4 h-4 rounded border-gray-300 text-blue-600 focus:ring-blue-500 cursor-pointer"
                />
                <span
                  className={`flex-1 text-sm ${
                    subtask.is_complete ? "line-through text-gray-400" : "text-gray-700"
                  }`}
                >
                  {subtask.title}
                </span>
                {!disabled && (
                  <button
                    type="button"
                    onClick={() => handleDeleteClick(subtask.id)}
                    disabled={loadingId === subtask.id}
                    className="text-gray-300 hover:text-red-600 p-1 opacity-0 group-hover:opacity-100 transition-opacity"
                    title="Delete subtask"
                  >
                    <svg className="w-4 h-4" fill="none" stroke="currentColor" viewBox="0 0 24 24">
                      <path strokeLinecap="round" strokeLinejoin="round" strokeWidth={2} d="M6 18L18 6M6 6l12 12" />
                    </svg>
                  </button>
                )}
              </li>
            ))}
          </ul>
        ) : !showAddInput ? (
          <p className="text-sm text-gray-400 italic py-2">No subtasks yet</p>
        ) : null}

        {/* Add Subtask Input */}
        {showAddInput && !disabled && (
          <div className="flex gap-2">
            <input
              type="text"
              value={newSubtaskTitle}
              onChange={(e) => setNewSubtaskTitle(e.target.value)}
              onKeyDown={handleKeyDown}
              placeholder="Subtask title..."
              className="flex-1 px-3 py-2 text-sm border border-gray-300 rounded-md focus:outline-none focus:ring-2 focus:ring-blue-500 focus:border-transparent"
              autoFocus
            />
            <button
              type="button"
              onClick={handleAddSubtask}
              disabled={isAdding || !newSubtaskTitle.trim()}
              className="px-3 py-2 text-sm bg-blue-600 text-white rounded-md hover:bg-blue-700 disabled:opacity-50 disabled:cursor-not-allowed transition-colors"
            >
              {isAdding ? "Adding..." : "Add"}
            </button>
            <button
              type="button"
              onClick={() => {
                setNewSubtaskTitle("");
                setShowAddInput(false);
              }}
              className="px-3 py-2 text-sm text-gray-600 border border-gray-300 rounded-md hover:bg-gray-50 transition-colors"
            >
              Cancel
            </button>
          </div>
        )}
      </div>

      {/* Delete Confirmation Dialog */}
      <ConfirmDialog
        isOpen={showDeleteConfirm}
        onClose={() => {
          setShowDeleteConfirm(false);
          setSubtaskToDelete(null);
        }}
        onConfirm={confirmDelete}
        title="Delete Subtask"
        message="Are you sure you want to delete this subtask? This cannot be undone."
        confirmText="Delete"
        cancelText="Cancel"
        variant="danger"
      />
    </>
  );
}
