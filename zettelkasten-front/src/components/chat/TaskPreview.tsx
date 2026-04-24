import React from 'react';
import { Task } from '../../models/Task';

interface TaskPreviewProps {
  task: Task;
  onTaskClick?: (taskId: number) => void;
}

export function TaskPreview({ task, onTaskClick }: TaskPreviewProps) {
  const handleClick = () => {
    if (onTaskClick) {
      onTaskClick(task.id);
    }
  };

  const getPriorityColor = (priority: string | null) => {
    switch (priority?.toLowerCase()) {
      case 'high':
        return 'text-red-600 bg-red-50';
      case 'medium':
        return 'text-yellow-600 bg-yellow-50';
      case 'low':
        return 'text-green-600 bg-green-50';
      default:
        return 'text-gray-600 bg-gray-50';
    }
  };

  const formatDate = (date: Date | null) => {
    if (!date) return null;
    const d = new Date(date);
    return d.toLocaleDateString('en-US', { month: 'short', day: 'numeric' });
  };

  return (
    <li
      onClick={handleClick}
      className="p-2 bg-white border border-blue-200 rounded-lg hover:bg-blue-50 hover:border-blue-300 transition-all cursor-pointer group"
    >
      <div className="flex items-start gap-2">
        <div className={`mt-0.5 w-4 h-4 rounded border-2 flex items-center justify-center flex-shrink-0 ${
          task.is_complete ? 'bg-blue-500 border-blue-500' : 'border-blue-300'
        }`}>
          {task.is_complete && (
            <svg className="w-3 h-3 text-white" fill="none" stroke="currentColor" viewBox="0 0 24 24">
              <path strokeLinecap="round" strokeLinejoin="round" strokeWidth={3} d="M5 13l4 4L19 7" />
            </svg>
          )}
        </div>

        <div className="flex-1 min-w-0">
          <div className="flex items-center gap-2 mb-1">
            <span className={`text-sm font-medium ${task.is_complete ? 'line-through text-gray-500' : 'text-gray-800'}`}>
              {task.title}
            </span>
            {task.priority && (
              <span className={`px-1.5 py-0.5 text-xs font-medium rounded ${getPriorityColor(task.priority)}`}>
                {task.priority}
              </span>
            )}
          </div>

          <div className="flex items-center gap-3 text-xs text-gray-500">
            {task.scheduled_date && (
              <div className="flex items-center gap-1">
                <svg className="w-3 h-3" fill="none" stroke="currentColor" viewBox="0 0 24 24">
                  <path strokeLinecap="round" strokeLinejoin="round" strokeWidth={2} d="M8 7V3m8 4V3m-9 8h10M5 21h14a2 2 0 002-2V7a2 2 0 00-2-2H5a2 2 0 00-2 2v12a2 2 0 002 2z" />
                </svg>
                <span>{formatDate(task.scheduled_date)}</span>
              </div>
            )}
            {task.card && (
              <div className="flex items-center gap-1">
                <svg className="w-3 h-3" fill="none" stroke="currentColor" viewBox="0 0 24 24">
                  <path strokeLinecap="round" strokeLinejoin="round" strokeWidth={2} d="M7 21h10a2 2 0 002-2V9.414a1 1 0 00-.293-.707l-5.414-5.414A1 1 0 0012.586 3H7a2 2 0 00-2 2v14a2 2 0 002 2z" />
                </svg>
                <span>{task.card.card_id || `Card ${task.card_pk}`}</span>
              </div>
            )}
          </div>
        </div>
      </div>
    </li>
  );
}