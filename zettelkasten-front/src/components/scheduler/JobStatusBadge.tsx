import React from 'react';

interface JobStatusBadgeProps {
  status: 'completed' | 'failed' | 'running' | 'never';
}

export function JobStatusBadge({ status }: JobStatusBadgeProps) {
  const styles = {
    completed: 'bg-green-100 text-green-800',
    failed: 'bg-red-100 text-red-800',
    running: 'bg-yellow-100 text-yellow-800',
    never: 'bg-gray-100 text-gray-800',
  };

  const labels = {
    completed: 'Completed',
    failed: 'Failed',
    running: 'Running',
    never: 'Never run',
  };

  const showPulse = status === 'running';

  return (
    <span
      className={`inline-flex items-center px-3 py-1 rounded-full text-sm font-medium ${styles[status]}`}
    >
      {showPulse && (
        <span className="w-2 h-2 bg-yellow-500 rounded-full mr-2 animate-pulse" />
      )}
      {!showPulse && status !== 'never' && (
        <span
          className={`w-2 h-2 rounded-full mr-2 ${
            status === 'completed'
              ? 'bg-green-500'
              : status === 'failed'
              ? 'bg-red-500'
              : 'bg-gray-500'
          }`}
        />
      )}
      {labels[status]}
    </span>
  );
}
