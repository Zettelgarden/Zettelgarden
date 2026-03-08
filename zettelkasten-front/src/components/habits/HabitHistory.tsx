import React from 'react';
import { format, formatDistanceToNow, isToday, isYesterday } from 'date-fns';
import { HabitLog } from '../../models/habit';

interface HabitHistoryProps {
  logs: HabitLog[];
  onUndoCheckin?: (logId: number) => void;
  maxItems?: number;
}

export const HabitHistory: React.FC<HabitHistoryProps> = ({ logs, onUndoCheckin, maxItems = 10 }) => {
  const displayLogs = logs.slice(0, maxItems);

  const formatDateLabel = (date: Date): string => {
    if (isToday(date)) return 'Today';
    if (isYesterday(date)) return 'Yesterday';
    return format(date, 'EEEE, MMM d');
  };

  const formatTimeAgo = (date: Date): string => {
    return formatDistanceToNow(date, { addSuffix: true });
  };

  if (logs.length === 0) {
    return (
      <div className="text-center py-6 text-gray-500">
        <p className="text-sm">No check-ins yet</p>
        <p className="text-xs mt-1">Your check-in history will appear here</p>
      </div>
    );
  }

  return (
    <div className="space-y-2">
      {displayLogs.map((log) => {
        const completedAt = new Date(log.completed_at);
        const canUndo = isToday(completedAt);

        return (
          <div
            key={log.id}
            className="flex items-start justify-between p-2 bg-gray-50 rounded-lg hover:bg-gray-100 transition-colors"
          >
            <div className="flex items-start gap-2">
              <div className="mt-0.5">
                <svg className="w-4 h-4 text-green-500" fill="currentColor" viewBox="0 0 20 20">
                  <path fillRule="evenodd" d="M10 18a8 8 0 100-16 8 8 0 000 16zm3.707-9.293a1 1 0 00-1.414-1.414L9 10.586 7.707 9.293a1 1 0 00-1.414 1.414l2 2a1 1 0 001.414 0l4-4z" clipRule="evenodd" />
                </svg>
              </div>
              <div>
                <p className="text-sm font-medium text-gray-800">
                  {formatDateLabel(completedAt)}
                </p>
                <p className="text-xs text-gray-500">
                  {formatTimeAgo(completedAt)}
                </p>
                {log.notes && (
                  <p className="text-xs text-gray-600 mt-1 italic">
                    "{log.notes}"
                  </p>
                )}
              </div>
            </div>
            {canUndo && onUndoCheckin && (
              <button
                onClick={() => onUndoCheckin(log.id)}
                className="text-xs text-red-500 hover:text-red-700 px-2 py-1 hover:bg-red-50 rounded"
                title="Undo this check-in"
              >
                Undo
              </button>
            )}
          </div>
        );
      })}

      {logs.length > maxItems && (
        <p className="text-xs text-center text-gray-400 pt-2">
          Showing {maxItems} of {logs.length} check-ins
        </p>
      )}
    </div>
  );
};
