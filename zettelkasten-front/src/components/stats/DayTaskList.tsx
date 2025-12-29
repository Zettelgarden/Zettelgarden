import React from "react";
import { Task } from "../../models/Task";
import { Link } from "react-router-dom";

interface DayTaskListProps {
  tasks: Task[];
  date: Date;
  onClose: () => void;
}

function formatTime(date: Date): string {
  return date.toLocaleTimeString("en-US", {
    hour: "numeric",
    minute: "2-digit",
    hour12: true,
  });
}

function formatDate(date: Date): string {
  return date.toLocaleDateString("en-US", {
    weekday: "long",
    year: "numeric",
    month: "long",
    day: "numeric",
  });
}

export function DayTaskList({ tasks, date, onClose }: DayTaskListProps) {
  return (
    <div className="bg-white rounded-lg shadow p-4 mt-4">
      <div className="flex justify-between items-center mb-3">
        <h2 className="text-lg font-semibold text-gray-800">
          {formatDate(date)} ({tasks.length} task{tasks.length !== 1 ? 's' : ''})
        </h2>
        <button
          onClick={onClose}
          className="text-gray-400 hover:text-gray-600 text-xl leading-none"
          aria-label="Close"
        >
          &times;
        </button>
      </div>

      {tasks.length === 0 ? (
        <div className="text-center py-4 text-sm text-gray-500">
          No tasks were completed on this day.
        </div>
      ) : (
        <div className="divide-y divide-gray-100">
          {tasks.map((task) => (
            <div
              key={task.id}
              className="py-2 hover:bg-gray-50 -mx-4 px-4 transition-colors"
            >
              <div className="flex items-baseline gap-2 flex-wrap">
                <span className="text-sm font-medium text-gray-900">{task.title}</span>
                {task.completed_at && (
                  <span className="text-xs text-gray-500">
                    {formatTime(task.completed_at)}
                  </span>
                )}
                {task.priority && (
                  <span className="px-1.5 py-0.5 bg-blue-100 text-blue-700 rounded text-xs">
                    {task.priority}
                  </span>
                )}
              </div>
              <div className="flex items-center gap-3 mt-1 flex-wrap">
                {task.card && task.card.card_id && (
                  <Link
                    to={`/app/card/${task.card.card_id}`}
                    className="text-xs text-blue-600 hover:text-blue-800 hover:underline"
                  >
                    {task.card.title || task.card.card_id}
                  </Link>
                )}
                {task.tags && task.tags.length > 0 && (
                  <div className="flex flex-wrap gap-1">
                    {task.tags.map((tag) => (
                      <span
                        key={tag.id}
                        className="px-1.5 py-0.5 bg-gray-100 text-gray-600 rounded text-xs"
                      >
                        #{tag.name}
                      </span>
                    ))}
                  </div>
                )}
              </div>
            </div>
          ))}
        </div>
      )}
    </div>
  );
}
