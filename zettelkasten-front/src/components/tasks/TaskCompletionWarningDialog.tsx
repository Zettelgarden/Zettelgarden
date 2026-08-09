import React, { useState } from 'react';
import { Modal } from '../ui/Modal';
import { Task } from '../../models/Task';

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

  const incompleteSubtasks = (task.subtasks || []).filter(
    (s) => !s.is_complete,
  );

  const handleForceComplete = async () => {
    setIsCompleting(true);
    try {
      await onForceComplete();
    } finally {
      setIsCompleting(false);
    }
  };

  return (
    <Modal open={visible} onClose={onCancel} size="md" dialogClassName="z-[80]">
      {/* Header */}
      <div className="flex items-center gap-3 mb-4">
        <svg
          className="w-6 h-6 text-amber-500"
          fill="none"
          viewBox="0 0 24 24"
          stroke="currentColor"
          strokeWidth={2}
        >
          <path
            strokeLinecap="round"
            strokeLinejoin="round"
            d="M12 9v2m0 4h.01m-6.938 4h13.856c1.54 0 2.502-1.667 1.732-3L13.732 4c-.77-1.333-2.694-1.333-3.464 0L3.34 16c-.77 1.333.192 3 1.732 3z"
          />
        </svg>
        <h3 className="text-lg font-semibold text-gray-900">
          Incomplete Subtasks
        </h3>
      </div>

      {/* Content */}
      <div className="mb-6">
        <p className="text-sm text-gray-600 mb-3">
          <strong className="text-gray-800">"{task.title}"</strong> has{' '}
          {incompleteCount} incomplete{' '}
          {incompleteCount === 1 ? 'subtask' : 'subtasks'}:
        </p>

        <ul className="space-y-1 max-h-40 overflow-y-auto">
          {incompleteSubtasks.slice(0, 5).map((subtask) => (
            <li
              key={subtask.id}
              className="text-sm text-gray-700 flex items-center gap-2"
            >
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
          type="button"
          onClick={onCancel}
          disabled={isCompleting}
          className="px-4 py-2 text-sm text-gray-700 hover:bg-gray-100 rounded-lg transition-colors disabled:opacity-50"
        >
          Cancel
        </button>
        <button
          type="button"
          onClick={handleForceComplete}
          disabled={isCompleting}
          className="px-4 py-2 text-sm bg-red-600 text-white rounded-lg hover:bg-red-700 transition-colors disabled:opacity-50 disabled:cursor-not-allowed"
        >
          {isCompleting ? 'Completing...' : 'Force Complete Anyway'}
        </button>
      </div>
    </Modal>
  );
}
