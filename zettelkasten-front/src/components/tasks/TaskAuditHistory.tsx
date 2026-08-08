import React, { useState } from 'react';
import { format } from 'date-fns-tz';
import { TaskAuditEvent } from '../../models/Task';
import { formatAuditEvent } from '../../utils/tasks';
import { useAuth } from '../../contexts/AuthContext';

interface TaskAuditHistoryProps {
  events: TaskAuditEvent[];
  defaultExpanded?: boolean;
}

export function TaskAuditHistory({
  events,
  defaultExpanded = false,
}: TaskAuditHistoryProps) {
  const [isExpanded, setIsExpanded] = useState(defaultExpanded);
  const { user } = useAuth();
  const userTimezone = user?.timezone || 'UTC';

  return (
    <div className="mt-6 border-t pt-4">
      <button
        onClick={() => setIsExpanded(!isExpanded)}
        className="flex items-center gap-2 w-full text-left"
      >
        <svg
          className={`w-4 h-4 text-gray-500 transition-transform ${
            isExpanded ? 'rotate-90' : ''
          }`}
          fill="none"
          viewBox="0 0 24 24"
          stroke="currentColor"
        >
          <path
            strokeLinecap="round"
            strokeLinejoin="round"
            strokeWidth={2}
            d="M9 5l7 7-7 7"
          />
        </svg>
        <h3 className="text-lg font-medium text-gray-900">Task History</h3>
        <span className="text-sm text-gray-500">({events.length})</span>
      </button>
      {isExpanded && (
        <div className="space-y-3 max-h-[200px] overflow-y-auto mt-4">
          {events.length > 0 ? (
            events.map((event) => (
              <div
                key={event.id}
                className="flex items-start space-x-3 text-sm hover:bg-gray-50 p-2 rounded"
              >
                <div className="text-gray-500 min-w-[120px] font-medium">
                  {format(event.created_at, 'MMM d, HH:mm', {
                    timeZone: userTimezone,
                  })}
                </div>
                <div className="flex-grow text-gray-700">
                  {formatAuditEvent(event, userTimezone)}
                </div>
              </div>
            ))
          ) : (
            <div className="text-sm text-gray-500 text-center py-4">
              No history available
            </div>
          )}
        </div>
      )}
    </div>
  );
}
