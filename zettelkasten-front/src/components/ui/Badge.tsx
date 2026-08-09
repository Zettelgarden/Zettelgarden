import React from 'react';

export type BadgeColor = 'success' | 'warning' | 'error' | 'info' | 'neutral';

export interface BadgeProps {
  /** Color of the pill (default: neutral) */
  color?: BadgeColor;
  children: React.ReactNode;
  /** Show a small colored dot before the label */
  dot?: boolean;
  /** Pulse the dot (for running/in-progress states) */
  pulse?: boolean;
  className?: string;
}

const colorClasses: Record<BadgeColor, string> = {
  success: 'bg-green-100 text-green-800',
  warning: 'bg-yellow-100 text-yellow-800',
  error: 'bg-red-100 text-red-800',
  info: 'bg-blue-100 text-blue-800',
  neutral: 'bg-gray-100 text-gray-800',
};

const dotColorClasses: Record<BadgeColor, string> = {
  success: 'bg-green-500',
  warning: 'bg-yellow-500',
  error: 'bg-red-500',
  info: 'bg-blue-500',
  neutral: 'bg-gray-500',
};

/**
 * Badge — the single shared colored pill, absorbing admin/StatusBadge and
 * scheduler/JobStatusBadge. Standardizes on the rounded-full pill look with
 * an optional status dot (pulse for running states).
 */
export function Badge({
  color = 'neutral',
  children,
  dot = false,
  pulse = false,
  className = '',
}: BadgeProps) {
  return (
    <span
      className={`inline-flex items-center px-3 py-1 rounded-full text-sm font-medium ${colorClasses[color]} ${className}`}
    >
      {(dot || pulse) && (
        <span
          className={`w-2 h-2 rounded-full mr-2 ${
            pulse ? 'animate-pulse' : ''
          } ${dotColorClasses[color]}`}
        />
      )}
      {children}
    </span>
  );
}

/**
 * Badge props for a subscription status string (from admin/StatusBadge).
 */
export function getSubscriptionStatusBadge(status: string): {
  color: BadgeColor;
  label: string;
} {
  switch (status) {
    case 'active':
      return { color: 'success', label: 'Active' };
    case 'trialing':
      return { color: 'info', label: 'Trial' };
    case 'past_due':
    case 'canceled':
    case 'incomplete':
    case 'incomplete_expired':
      return { color: 'error', label: status };
    default:
      return { color: 'neutral', label: 'Free' };
  }
}

/**
 * Badge props for a boolean status (from admin/StatusBadge).
 */
export function getBooleanStatusBadge(
  value: boolean,
  options: { trueLabel?: string; falseLabel?: string } = {},
): { color: BadgeColor; label: string } {
  const { trueLabel = 'Yes', falseLabel = 'No' } = options;
  return {
    color: value ? 'success' : 'neutral',
    label: value ? trueLabel : falseLabel,
  };
}
