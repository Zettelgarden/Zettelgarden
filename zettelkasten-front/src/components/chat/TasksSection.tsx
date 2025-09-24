import React, { useState } from 'react';

import { Task } from '../../models/Task';
import { TaskPreview } from './TaskPreview';

interface TasksSectionProps {
  tasks: Task[];
  onTaskClick?: (taskId: number) => void;
}

export function TasksSection({ tasks, onTaskClick }: TasksSectionProps) {
  const [isExpanded, setIsExpanded] = useState(true);

  // Filter out tasks with null or empty titles
  const validTasks = tasks.filter(task => task.title && task.title.trim() !== '');

  if (validTasks.length === 0) return null;

  const toggleExpanded = () => {
    setIsExpanded(!isExpanded);
  };

  return (
    <div className="mt-4 p-3 bg-blue-50 border border-blue-200 rounded-lg">
      <button
        onClick={toggleExpanded}
        className="w-full flex items-center justify-between mb-2 hover:bg-blue-100 p-1 rounded transition-colors"
      >
        <div className="flex items-center gap-2">
          <svg className="w-4 h-4 text-blue-600" fill="none" stroke="currentColor" viewBox="0 0 24 24">
            <path strokeLinecap="round" strokeLinejoin="round" strokeWidth={2} d="M9 5H7a2 2 0 00-2 2v12a2 2 0 002 2h10a2 2 0 002-2V7a2 2 0 00-2-2h-2M9 5a2 2 0 002 2h2a2 2 0 002-2M9 5a2 2 0 012-2h2a2 2 0 012 2m-6 9l2 2 4-4" />
          </svg>
          <h4 className="text-sm font-medium text-blue-800">
            {validTasks.length === 1 ? 'Referenced Task' : `Referenced Tasks (${validTasks.length})`}
          </h4>
        </div>
        <svg
          className={`w-4 h-4 text-blue-500 transition-transform duration-200 ${isExpanded ? 'rotate-180' : ''}`}
          fill="none"
          stroke="currentColor"
          viewBox="0 0 24 24"
        >
          <path strokeLinecap="round" strokeLinejoin="round" strokeWidth={2} d="M19 9l-7 7-7-7" />
        </svg>
      </button>

      {isExpanded && (
        <ul className="space-y-1 animate-fadeIn">
          {validTasks.map((task) => (
            <TaskPreview
              key={task.id}
              task={task}
              onTaskClick={onTaskClick}
            />
          ))}
        </ul>
      )}
    </div>
  );
}