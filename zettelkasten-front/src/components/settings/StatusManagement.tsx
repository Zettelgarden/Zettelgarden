import React, { useState } from 'react';
import { Modal } from '../ui/Modal';
import { useStatus } from '../../contexts/StatusContext';
import {
  TaskStatus,
  CreateTaskStatusParams,
  UpdateTaskStatusParams,
} from '../../models/TaskStatus';
import {
  createTaskStatus,
  updateTaskStatus,
  deleteTaskStatus,
  reorderTaskStatuses,
} from '../../api/taskStatuses';
import {
  DragDropContext,
  Droppable,
  Draggable,
  DropResult,
} from '@hello-pangea/dnd';

interface StatusItemProps {
  status: TaskStatus;
  onEdit: (status: TaskStatus) => void;
  onDelete: (status: TaskStatus) => void;
  index: number;
}

const StatusItem: React.FC<StatusItemProps> = ({
  status,
  onEdit,
  onDelete,
  index,
}) => {
  return (
    <Draggable draggableId={status.id.toString()} index={index}>
      {(provided, snapshot) => (
        <div
          ref={provided.innerRef}
          {...provided.draggableProps}
          {...provided.dragHandleProps}
          className={`bg-white rounded-lg border p-4 mb-3 transition-shadow ${
            snapshot.isDragging
              ? 'shadow-lg border-blue-400'
              : 'shadow-sm border-gray-200'
          }`}
        >
          <div className="flex items-center justify-between">
            <div className="flex items-center gap-3 flex-1">
              {/* Drag handle icon */}
              <div className="text-gray-400 cursor-grab active:cursor-grabbing">
                <svg
                  className="w-5 h-5"
                  fill="currentColor"
                  viewBox="0 0 20 20"
                >
                  <path d="M7 2a2 2 0 1 0 .001 4.001A2 2 0 0 0 7 2zm0 6a2 2 0 1 0 .001 4.001A2 2 0 0 0 7 8zm0 6a2 2 0 1 0 .001 4.001A2 2 0 0 0 7 14zm6-8a2 2 0 1 0-.001-4.001A2 2 0 0 0 13 6zm0 2a2 2 0 1 0 .001 4.001A2 2 0 0 0 13 8zm0 6a2 2 0 1 0 .001 4.001A2 2 0 0 0 13 14z" />
                </svg>
              </div>

              {/* Status icon and color */}
              <div
                className="w-10 h-10 rounded-full flex items-center justify-center text-xl"
                style={{
                  backgroundColor: status.color + '20',
                  border: `2px solid ${status.color}`,
                }}
              >
                {status.icon}
              </div>

              {/* Status info */}
              <div className="flex-1">
                <div className="flex items-center gap-2">
                  <h3 className="font-semibold text-gray-900">
                    {status.display_name}
                  </h3>
                  <span className="text-xs text-gray-500">({status.name})</span>
                </div>
                <div className="flex gap-2 mt-1">
                  {status.is_default && (
                    <span className="inline-flex items-center px-2 py-0.5 rounded text-xs font-medium bg-blue-100 text-blue-800">
                      Default
                    </span>
                  )}
                  {status.is_complete_state && (
                    <span className="inline-flex items-center px-2 py-0.5 rounded text-xs font-medium bg-green-100 text-green-800">
                      Complete Status
                    </span>
                  )}
                </div>
              </div>
            </div>

            {/* Action buttons */}
            <div className="flex gap-2">
              <button
                onClick={() => onEdit(status)}
                className="px-4 py-3 min-h-[44px] text-sm text-blue-600 hover:bg-blue-50 rounded-md transition-colors"
              >
                Edit
              </button>
              <button
                onClick={() => onDelete(status)}
                className="px-4 py-3 min-h-[44px] text-sm text-red-600 hover:bg-red-50 rounded-md transition-colors"
              >
                Delete
              </button>
            </div>
          </div>
        </div>
      )}
    </Draggable>
  );
};

const commonColors = [
  '#6B7280', // Gray
  '#EF4444', // Red
  '#F59E0B', // Amber
  '#10B981', // Green
  '#3B82F6', // Blue
  '#8B5CF6', // Purple
  '#EC4899', // Pink
  '#14B8A6', // Teal
];

const commonIcons = [
  '⭕',
  '🔄',
  '🚫',
  '✅',
  '⏸️',
  '🎯',
  '⚡',
  '🔥',
  '💡',
  '🚀',
  '⭐',
  '📌',
];

export const StatusManagement: React.FC = () => {
  const { statuses, refreshStatuses } = useStatus();
  const [editingStatus, setEditingStatus] = useState<TaskStatus | null>(null);
  const [isCreating, setIsCreating] = useState(false);
  const [deleteConfirm, setDeleteConfirm] = useState<TaskStatus | null>(null);
  const [error, setError] = useState<string | null>(null);
  const [success, setSuccess] = useState<string | null>(null);

  // Form state for create/edit
  const [formData, setFormData] = useState({
    name: '',
    display_name: '',
    color: '#6B7280',
    icon: '⭕',
    is_default: false,
    is_complete_state: false,
  });

  const resetForm = () => {
    setFormData({
      name: '',
      display_name: '',
      color: '#6B7280',
      icon: '⭕',
      is_default: false,
      is_complete_state: false,
    });
    setEditingStatus(null);
    setIsCreating(false);
    setError(null);
  };

  const handleEdit = (status: TaskStatus) => {
    setFormData({
      name: status.name,
      display_name: status.display_name,
      color: status.color,
      icon: status.icon,
      is_default: status.is_default,
      is_complete_state: status.is_complete_state,
    });
    setEditingStatus(status);
    setIsCreating(false);
  };

  const handleCreate = () => {
    resetForm();
    setIsCreating(true);
  };

  const handleSubmit = async (e: React.FormEvent) => {
    e.preventDefault();
    setError(null);
    setSuccess(null);

    try {
      if (editingStatus) {
        // Update existing status
        const updates: UpdateTaskStatusParams = {
          display_name: formData.display_name,
          color: formData.color,
          icon: formData.icon,
          is_default: formData.is_default,
          is_complete_state: formData.is_complete_state,
        };
        await updateTaskStatus(editingStatus.id, updates);
        setSuccess('Status updated successfully');
      } else {
        // Create new status
        const params: CreateTaskStatusParams = {
          ...formData,
          position: statuses.length, // Add to end
        };
        await createTaskStatus(params);
        setSuccess('Status created successfully');
      }

      await refreshStatuses();
      resetForm();
      setTimeout(() => setSuccess(null), 3000);
    } catch (err) {
      setError(err instanceof Error ? err.message : 'Failed to save status');
    }
  };

  const handleDelete = async (status: TaskStatus) => {
    setDeleteConfirm(status);
  };

  const confirmDelete = async () => {
    if (!deleteConfirm) return;

    setError(null);
    setSuccess(null);

    try {
      await deleteTaskStatus(deleteConfirm.id);
      setSuccess('Status deleted successfully');
      await refreshStatuses();
      setDeleteConfirm(null);
      setTimeout(() => setSuccess(null), 3000);
    } catch (err) {
      setError(err instanceof Error ? err.message : 'Failed to delete status');
      setDeleteConfirm(null);
    }
  };

  const handleDragEnd = async (result: DropResult) => {
    if (!result.destination) return;

    const items = Array.from(statuses);
    const [reorderedItem] = items.splice(result.source.index, 1);
    items.splice(result.destination.index, 0, reorderedItem);

    try {
      await reorderTaskStatuses({ status_ids: items.map((s) => s.id) });
      await refreshStatuses();
      setSuccess('Statuses reordered successfully');
      setTimeout(() => setSuccess(null), 2000);
    } catch (err) {
      setError('Failed to reorder statuses');
    }
  };

  return (
    <div className="max-w-4xl mx-auto p-6">
      <div className="mb-6">
        <h2 className="text-2xl font-bold text-gray-900 mb-2">
          Task Status Management
        </h2>
        <p className="text-gray-600">
          Customize your workflow by managing task statuses. Drag to reorder.
        </p>
      </div>

      {/* Success/Error messages */}
      {success && (
        <div className="mb-4 p-4 bg-green-50 border border-green-200 rounded-lg text-green-800">
          {success}
        </div>
      )}
      {error && (
        <div className="mb-4 p-4 bg-red-50 border border-red-200 rounded-lg text-red-800">
          {error}
        </div>
      )}

      {/* Create/Edit Form */}
      {(isCreating || editingStatus) && (
        <div className="mb-6 bg-gray-50 border border-gray-200 rounded-lg p-6">
          <h3 className="text-lg font-semibold mb-4">
            {editingStatus ? 'Edit Status' : 'Create New Status'}
          </h3>

          <form onSubmit={handleSubmit} className="space-y-4">
            <div className="grid grid-cols-2 gap-4">
              <div>
                <label className="block text-sm font-medium text-gray-700 mb-1">
                  Name (identifier)
                </label>
                <input
                  type="text"
                  value={formData.name}
                  onChange={(e) =>
                    setFormData({ ...formData, name: e.target.value })
                  }
                  disabled={!!editingStatus} // Can't change name when editing
                  className="w-full px-3 py-2 border border-gray-300 rounded-md focus:outline-none focus:ring-2 focus:ring-blue-500 disabled:bg-gray-100"
                  placeholder="e.g., in_progress"
                  required
                />
                {!editingStatus && (
                  <p className="text-xs text-gray-500 mt-1">
                    Lowercase, underscores only. Cannot be changed later.
                  </p>
                )}
              </div>

              <div>
                <label className="block text-sm font-medium text-gray-700 mb-1">
                  Display Name
                </label>
                <input
                  type="text"
                  value={formData.display_name}
                  onChange={(e) =>
                    setFormData({ ...formData, display_name: e.target.value })
                  }
                  className="w-full px-3 py-2 border border-gray-300 rounded-md focus:outline-none focus:ring-2 focus:ring-blue-500"
                  placeholder="e.g., In Progress"
                  required
                />
              </div>
            </div>

            <div>
              <label className="block text-sm font-medium text-gray-700 mb-2">
                Color
              </label>
              <div className="flex gap-2 flex-wrap">
                {commonColors.map((color) => (
                  <button
                    key={color}
                    type="button"
                    onClick={() => setFormData({ ...formData, color })}
                    className={`w-10 h-10 rounded-full border-2 transition-all ${
                      formData.color === color
                        ? 'border-gray-900 scale-110'
                        : 'border-gray-300 hover:scale-105'
                    }`}
                    style={{ backgroundColor: color }}
                  />
                ))}
                <input
                  type="color"
                  value={formData.color}
                  onChange={(e) =>
                    setFormData({ ...formData, color: e.target.value })
                  }
                  className="w-10 h-10 rounded-full border-2 border-gray-300 cursor-pointer"
                />
              </div>
            </div>

            <div>
              <label className="block text-sm font-medium text-gray-700 mb-2">
                Icon
              </label>
              <div className="flex gap-2 flex-wrap mb-2">
                {commonIcons.map((icon) => (
                  <button
                    key={icon}
                    type="button"
                    onClick={() => setFormData({ ...formData, icon })}
                    className={`w-12 h-12 rounded-lg border-2 text-xl transition-all ${
                      formData.icon === icon
                        ? 'border-blue-500 bg-blue-50 scale-110'
                        : 'border-gray-300 hover:bg-gray-50 hover:scale-105'
                    }`}
                  >
                    {icon}
                  </button>
                ))}
              </div>
              <input
                type="text"
                value={formData.icon}
                onChange={(e) =>
                  setFormData({ ...formData, icon: e.target.value })
                }
                className="w-full px-3 py-2 border border-gray-300 rounded-md focus:outline-none focus:ring-2 focus:ring-blue-500"
                placeholder="Or enter any emoji"
                maxLength={10}
              />
            </div>

            <div className="space-y-2">
              <label className="flex items-center gap-2">
                <input
                  type="checkbox"
                  checked={formData.is_default}
                  onChange={(e) =>
                    setFormData({ ...formData, is_default: e.target.checked })
                  }
                  className="w-4 h-4 text-blue-600 rounded focus:ring-blue-500"
                />
                <span className="text-sm text-gray-700">
                  Set as default status (used for new tasks)
                </span>
              </label>

              <label className="flex items-center gap-2">
                <input
                  type="checkbox"
                  checked={formData.is_complete_state}
                  onChange={(e) =>
                    setFormData({
                      ...formData,
                      is_complete_state: e.target.checked,
                    })
                  }
                  className="w-4 h-4 text-blue-600 rounded focus:ring-blue-500"
                />
                <span className="text-sm text-gray-700">
                  Mark as complete status (tasks with this status are considered
                  done)
                </span>
              </label>
            </div>

            <div className="flex gap-2 pt-2">
              <button
                type="submit"
                className="px-4 py-2 bg-blue-600 text-white rounded-md hover:bg-blue-700 transition-colors"
              >
                {editingStatus ? 'Update Status' : 'Create Status'}
              </button>
              <button
                type="button"
                onClick={resetForm}
                className="px-4 py-2 bg-gray-200 text-gray-700 rounded-md hover:bg-gray-300 transition-colors"
              >
                Cancel
              </button>
            </div>
          </form>
        </div>
      )}

      {/* Add Status Button */}
      {!isCreating && !editingStatus && (
        <button
          onClick={handleCreate}
          className="mb-6 w-full py-3 border-2 border-dashed border-gray-300 rounded-lg text-gray-600 hover:border-blue-400 hover:text-blue-600 transition-colors"
        >
          + Add New Status
        </button>
      )}

      {/* Status List */}
      <DragDropContext onDragEnd={handleDragEnd}>
        <Droppable droppableId="statuses">
          {(provided) => (
            <div {...provided.droppableProps} ref={provided.innerRef}>
              {statuses.map((status, index) => (
                <StatusItem
                  key={status.id}
                  status={status}
                  index={index}
                  onEdit={handleEdit}
                  onDelete={handleDelete}
                />
              ))}
              {provided.placeholder}
            </div>
          )}
        </Droppable>
      </DragDropContext>

      {/* Delete Confirmation Modal */}
      {deleteConfirm && (
        <Modal
          open
          onClose={() => setDeleteConfirm(null)}
          size="md"
          dialogClassName="z-50"
        >
          <h3 className="text-lg font-semibold mb-2">Delete Status</h3>
          <p className="text-gray-600 mb-4">
            Are you sure you want to delete "{deleteConfirm.display_name}"? All
            tasks with this status will be reassigned to your default status.
          </p>
          {(deleteConfirm.is_default || deleteConfirm.is_complete_state) && (
            <p className="text-amber-600 text-sm mb-4">
              ⚠️ Warning: This is a{' '}
              {deleteConfirm.is_default ? 'default' : 'complete'} status. Make
              sure you have another one configured.
            </p>
          )}
          <div className="flex gap-2 justify-end">
            <button
              onClick={() => setDeleteConfirm(null)}
              className="px-4 py-2 bg-gray-200 text-gray-700 rounded-md hover:bg-gray-300 transition-colors"
            >
              Cancel
            </button>
            <button
              onClick={confirmDelete}
              className="px-4 py-2 bg-red-600 text-white rounded-md hover:bg-red-700 transition-colors"
            >
              Delete
            </button>
          </div>
        </Modal>
      )}
    </div>
  );
};
