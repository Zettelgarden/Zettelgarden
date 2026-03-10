import React, { useState } from "react";
import { Task } from "../../models/Task";

interface TaskCompletionWarningDialogProps {
  visible: boolean;
  task: Task;
  incompleteCount: number;
  onForceComplete: () => Promise<void>;
  onCancel: () => void;
}

export function TaskCompletionWarningDialog({
  visible,
  task,
  incompleteCount,
  onForceComplete,
  onCancel,
}: TaskCompletionWarningDialogProps) {
  const [isCompleting, setIsCompleting] = useState(false);

  if (!visible) return null;

  const incompleteSubtasks = (task.subtasks || []).filter((s) => !s.is_complete);

  const handleForceComplete = async () => {
    setIsCompleting(true);
    try {
      await onForceComplete();
    } finally {
      setIsCompleting(false);
    }
  };

  return (
    <div className="fixed inset-0 bg-black bg-opacity-50 flex items-center justify-center z-50 p-4">
      <div className="bg-white rounded-lg w-full max-w-md p-6 shadow-xl">
        {/* Header */}
        <div className="flex items-center gap-3 mb-4">
          <span className="text-2xl">⚠️</span>
          <h3 className="text-lg font-semibold text-gray-900">
            Incomplete Subtasks
          </h3>
        </div>

        {/* Content */}
        <div className="mb-6">
          <p className="text-sm text-gray-600 mb-3">
            <strong className="text-gray-800">"{task.title}"</strong> has {incompleteCount} incomplete{" "}
            {incompleteCount === 1 ? "subtask" : "subtasks"}:
          </p>

          <ul className="space-y-1 max-h-40 overflow-y-auto">
            {incompleteSubtasks.slice(0, 5).map((subtask) => (
              <li key={subtask.id} className="text-sm text-gray-700 flex items-center gap-2">
                <span className="w-2 h-2 rounded-full bg-orange-400 flex-shrink-0" />
                <span className="truncate">{subtask.title}</span>
              </li>
            ))}
            {incompleteSubtasks.length > 5 && (
              <li className="text-sm text-gray-400 italic">
                +{incompleteSubtasks.length - 5} more...
              </li>
            )}
          </ul>
        </div>

        {/* Actions */}
        <div className="flex justify-end gap-3">
          <button
            onClick={onCancel}
            disabled={isCompleting}
            className="px-4 py-2 text-sm text-gray-700 hover:bg-gray-100 rounded-lg transition-colors disabled:opacity-50"
          >
            Cancel
          </button>
          <button
            onClick={handleForceComplete}
            disabled={isCompleting}
            className="px-4 py-2 text-sm bg-red-600 text-white rounded-lg hover:bg-red-700 transition-colors disabled:opacity-50 disabled:cursor-not-allowed"
          >
            {isCompleting ? "Completing..." : "Force Complete Anyway"}
          </button>
        </div>
      </div>
    </div>
  );
}
