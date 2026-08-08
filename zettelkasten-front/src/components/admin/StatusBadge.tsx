import React from 'react';

/**
 * Status badge type with predefined styles
 */
export type StatusType = 'success' | 'warning' | 'error' | 'info' | 'neutral';

/**
 * Props for the StatusBadge component
 */
export interface StatusBadgeProps {
  /** The value to display (shown if no label is provided) */
  value?: boolean | string | number;
  /** The status type that determines badge color */
  type: StatusType;
  /** Optional custom label (overrides default label from value) */
  label?: string;
  /** Additional CSS classes */
  className?: string;
}

/**
 * Maps status types to their Tailwind CSS classes
 */
const statusStyles: Record<StatusType, string> = {
  success: 'bg-green-100 text-green-800',
  warning: 'bg-yellow-100 text-yellow-800',
  error: 'bg-red-100 text-red-800',
  info: 'bg-blue-100 text-blue-800',
  neutral: 'bg-gray-100 text-gray-800',
};

/**
 * Default labels for boolean values
 */
const booleanLabels: Record<string, { true: string; false: string }> = {
  default: { true: 'Yes', false: 'No' },
  verified: { true: 'Verified', false: 'Pending' },
  subscribed: { true: 'Subscribed', false: 'Unsubscribed' },
  sent: { true: 'Sent', false: 'Pending' },
  hasAccount: { true: 'Has Account', false: 'No Account' },
};

/**
 * StatusBadge component for displaying status indicators with consistent styling.
 *
 * @example
 * ```tsx
 * // With boolean value
 * <StatusBadge type="success" value={user.emailValidated} labelType="verified" />
 *
 * // With custom label
 * <StatusBadge type="info" label="Pro" />
 *
 * // With string value
 * <StatusBadge type="warning" value={subscription.status} />
 * ```
 */
export function StatusBadge({
  value,
  type,
  label,
  className = '',
}: StatusBadgeProps) {
  const classes =
    `px-2 py-1 rounded text-sm ${statusStyles[type]} ${className}`.trim();

  // If label is provided, use it
  if (label !== undefined) {
    return <span className={classes}>{label}</span>;
  }

  // If value is a boolean, use default labels
  if (typeof value === 'boolean') {
    const displayLabel = value
      ? booleanLabels.default.true
      : booleanLabels.default.false;
    return <span className={classes}>{displayLabel}</span>;
  }

  // If value is a string or number, display it
  if (value !== undefined) {
    return <span className={classes}>{String(value)}</span>;
  }

  // Fallback: no value or label provided
  return null;
}

/**
 * Helper function to get subscription status badge props
 */
export function getSubscriptionStatusBadge(status: string): {
  type: StatusType;
  label: string;
} {
  switch (status) {
    case 'active':
      return { type: 'success', label: 'Active' };
    case 'trialing':
      return { type: 'info', label: 'Trial' };
    case 'past_due':
    case 'canceled':
    case 'incomplete':
    case 'incomplete_expired':
      return { type: 'error', label: status };
    default:
      return { type: 'neutral', label: 'Free' };
  }
}

/**
 * Helper function to get boolean status badge props
 */
export function getBooleanStatusBadge(
  value: boolean,
  options: { trueLabel?: string; falseLabel?: string } = {},
): { type: StatusType; label: string } {
  const { trueLabel = 'Yes', falseLabel = 'No' } = options;
  return {
    type: value ? 'success' : 'neutral',
    label: value ? trueLabel : falseLabel,
  };
}
