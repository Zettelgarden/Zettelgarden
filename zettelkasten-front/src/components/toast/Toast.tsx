import React, { useEffect } from 'react';

export type ToastType = 'success' | 'error' | 'info' | 'warning';

export interface ToastMessage {
  id: string;
  type: ToastType;
  title: string;
  description?: string;
  duration?: number;
}

interface ToastProps extends ToastMessage {
  onRemove: (id: string) => void;
}

const toastStyles = {
  success: 'bg-green-500 border-green-600',
  error: 'bg-red-500 border-red-600',
  info: 'bg-blue-500 border-blue-600',
  warning: 'bg-yellow-500 border-yellow-600',
};

const iconStyles = {
  success: '✅',
  error: '❌',
  info: 'ℹ️',
  warning: '⚠️',
};

export function Toast({
  id,
  type,
  title,
  description,
  duration = 4000,
  onRemove,
}: ToastProps) {
  useEffect(() => {
    if (duration > 0) {
      const timer = setTimeout(() => {
        onRemove(id);
      }, duration);

      return () => clearTimeout(timer);
    }
  }, [id, duration, onRemove]);

  return (
    <div
      className={`${toastStyles[type]} text-white px-4 py-3 rounded-lg border-l-4 shadow-lg transform transition-all duration-300 ease-in-out flex items-start max-w-sm`}
    >
      <div className="flex items-start">
        <span className="text-lg mr-3 mt-0.5">{iconStyles[type]}</span>
        <div className="flex-1 min-w-0">
          <p className="font-semibold text-sm">{title}</p>
          {description && (
            <p className="text-sm opacity-90 mt-1 leading-relaxed">
              {description}
            </p>
          )}
        </div>
        <button
          onClick={() => onRemove(id)}
          className="ml-3 text-white hover:opacity-75 transition-opacity p-1"
          aria-label="Dismiss"
        >
          <svg
            className="w-4 h-4"
            fill="none"
            stroke="currentColor"
            viewBox="0 0 24 24"
          >
            <path
              strokeLinecap="round"
              strokeLinejoin="round"
              strokeWidth={2}
              d="M6 18L18 6M6 6l12 12"
            />
          </svg>
        </button>
      </div>
    </div>
  );
}
