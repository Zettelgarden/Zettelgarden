import React from 'react';
import { Spinner } from '../ui/Spinner';

/**
 * Error severity levels
 */
export type ErrorSeverity = 'info' | 'warning' | 'error' | 'critical';

/**
 * Error display props
 */
export interface AdminErrorDisplayProps {
  /** The error message to display */
  message: string;
  /** Error severity for styling */
  severity?: ErrorSeverity;
  /** Optional title/headline for the error */
  title?: string;
  /** Optional error details (stack trace, technical info, etc.) */
  details?: string;
  /** Callback when dismiss button is clicked */
  onDismiss?: () => void;
  /** Optional retry action */
  onRetry?: () => void;
  /** Whether error can be dismissed */
  dismissible?: boolean;
  /** Additional CSS classes */
  className?: string;
}

/**
 * Maps error severity to Tailwind CSS classes
 */
const severityStyles: Record<ErrorSeverity, string> = {
  info: 'bg-blue-50 border-blue-200 text-blue-800',
  warning: 'bg-yellow-50 border-yellow-200 text-yellow-800',
  error: 'bg-red-50 border-red-200 text-red-800',
  critical: 'bg-red-100 border-red-300 text-red-900',
};

const iconMap: Record<ErrorSeverity, string> = {
  info: 'ℹ️',
  warning: '⚠️',
  error: '❌',
  critical: '🚨',
};

/**
 * AdminErrorDisplay component - Consistent error display for admin pages.
 *
 * Provides dismissible error alerts with:
 * - Severity-based styling
 * - Optional title and details
 * - Retry action support
 * - Dismissible with close button
 *
 * @example
 * ```tsx
 * <AdminErrorDisplay
 *   message="Failed to load users"
 *   severity="error"
 *   details={error.message}
 *   onRetry={() => refetch()}
 *   onDismiss={() => setError(null)}
 * />
 * ```
 */
export function AdminErrorDisplay({
  message,
  severity = 'error',
  title,
  details,
  onDismiss,
  onRetry,
  dismissible = true,
  className = '',
}: AdminErrorDisplayProps) {
  const baseClasses = 'rounded-lg border p-4 mb-4';
  const severityClasses = severityStyles[severity];

  return (
    <div className={`${baseClasses} ${severityClasses} ${className}`}>
      <div className="flex items-start">
        <div className="flex-shrink-0 text-xl mr-3">{iconMap[severity]}</div>
        <div className="flex-1">
          {title && <h3 className="text-lg font-semibold mb-1">{title}</h3>}
          <p className="text-sm">{message}</p>
          {details && (
            <details className="mt-2">
              <summary className="cursor-pointer text-sm opacity-75 hover:opacity-100">
                Technical details
              </summary>
              <pre className="mt-2 text-xs bg-black/5 p-2 rounded overflow-auto max-h-32">
                {details}
              </pre>
            </details>
          )}
          {(onRetry || dismissible) && (
            <div className="mt-3 flex gap-2">
              {onRetry && (
                <button
                  onClick={onRetry}
                  className="px-3 py-1 text-sm font-medium bg-white/50 hover:bg-white/75 rounded transition-colors"
                >
                  Retry
                </button>
              )}
              {dismissible && onDismiss && (
                <button
                  onClick={onDismiss}
                  className="px-3 py-1 text-sm font-medium bg-black/5 hover:bg-black/10 rounded transition-colors"
                >
                  Dismiss
                </button>
              )}
            </div>
          )}
        </div>
        {dismissible && onDismiss && (
          <button
            onClick={onDismiss}
            className="ml-4 text-lg opacity-50 hover:opacity-100 transition-opacity"
            aria-label="Dismiss"
          >
            ×
          </button>
        )}
      </div>
    </div>
  );
}

/**
 * Loading skeleton component for admin pages
 */
export interface AdminLoadingStateProps {
  /** Number of skeleton rows to show */
  rows?: number;
  /** Optional loading message */
  message?: string;
  /** Additional CSS classes */
  className?: string;
}

/**
 * AdminLoadingState - Skeleton loading state for admin tables.
 *
 * @example
 * ```tsx
 * <AdminLoadingState rows={5} message="Loading users..." />
 * ```
 */
export function AdminLoadingState({
  rows = 5,
  message,
  className = '',
}: AdminLoadingStateProps) {
  return (
    <div className={`space-y-4 ${className}`}>
      {message && (
        <div className="text-center text-gray-500 py-4">
          <div className="animate-pulse inline-flex items-center gap-2">
            <Spinner size="sm" className="text-gray-600" />
            {message}
          </div>
        </div>
      )}
      <div className="overflow-x-auto">
        <table className="min-w-full bg-white shadow-md rounded">
          <thead className="bg-gray-800 text-white">
            <tr>
              <th className="py-2 px-4 text-left">
                <div className="h-4 bg-gray-600 rounded animate-pulse" />
              </th>
              <th className="py-2 px-4 text-left">
                <div className="h-4 bg-gray-600 rounded animate-pulse" />
              </th>
              <th className="py-2 px-4 text-left">
                <div className="h-4 bg-gray-600 rounded animate-pulse" />
              </th>
            </tr>
          </thead>
          <tbody>
            {Array.from({ length: rows }).map((_, index) => (
              <tr key={index} className="border-b">
                <td className="py-2 px-4">
                  <div className="h-4 bg-gray-200 rounded animate-pulse" />
                </td>
                <td className="py-2 px-4">
                  <div className="h-4 bg-gray-200 rounded animate-pulse" />
                </td>
                <td className="py-2 px-4">
                  <div className="h-4 bg-gray-200 rounded animate-pulse" />
                </td>
              </tr>
            ))}
          </tbody>
        </table>
      </div>
    </div>
  );
}
