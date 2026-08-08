import React from 'react';
import { Task } from '../../models/Task';

interface TaskDescriptionSectionProps {
  task: Task;
  setTask: (task: Task) => void;
  mode: 'create' | 'edit';
  isEditingDescription: boolean;
  setIsEditingDescription: (editing: boolean) => void;
  onSaveDescription: () => Promise<void>;
}

export function TaskDescriptionSection({
  task,
  setTask,
  mode,
  isEditingDescription,
  setIsEditingDescription,
  onSaveDescription,
}: TaskDescriptionSectionProps) {
  function handleDescriptionChange(e: React.ChangeEvent<HTMLTextAreaElement>) {
    setTask({ ...task, description: e.target.value || null });
  }

  async function handleDescriptionSave() {
    await onSaveDescription();
  }

  return mode === 'create' || isEditingDescription ? (
    <div className="space-y-2">
      <textarea
        placeholder="Add a description..."
        value={task.description || ''}
        onChange={handleDescriptionChange}
        className="w-full px-3 py-2 border rounded-md focus:outline-none focus:ring-2 focus:ring-blue-500 border-gray-300 min-h-[80px] max-h-[30vh] resize-y"
        autoFocus={mode === 'edit'}
      />
      {mode === 'edit' && (
        <div className="flex gap-2">
          <button
            onClick={handleDescriptionSave}
            className="px-4 py-3 min-h-[44px] bg-blue-600 text-white rounded-md hover:bg-blue-700 text-sm"
          >
            Save
          </button>
          <button
            onClick={() => setIsEditingDescription(false)}
            className="px-4 py-3 min-h-[44px] bg-gray-200 text-gray-700 rounded-md hover:bg-gray-300 text-sm"
          >
            Cancel
          </button>
        </div>
      )}
    </div>
  ) : (
    <div
      className="text-gray-600 cursor-pointer hover:bg-gray-50 p-2 rounded min-h-[44px] flex items-center"
      onClick={() => setIsEditingDescription(true)}
    >
      {task.description ? (
        <p className="whitespace-pre-wrap">{task.description}</p>
      ) : (
        <p className="text-gray-400 italic">Add a description...</p>
      )}
    </div>
  );
}
