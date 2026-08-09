import React from 'react';

export interface SpinnerProps {
  /** Size of the spinner (default: md) */
  size?: 'sm' | 'md' | 'lg' | 'xl';
  /** Extra classes, e.g. text color: text-blue-600 */
  className?: string;
  /** Accessible label announced to screen readers (default: "Loading") */
  label?: string;
}

const sizeClasses: Record<NonNullable<SpinnerProps['size']>, string> = {
  sm: 'h-4 w-4',
  md: 'h-5 w-5',
  lg: 'h-8 w-8',
  xl: 'h-12 w-12',
};

/**
 * Spinner — the single shared loading indicator.
 *
 * Uses the standard Tailwind SVG spinner, sized via the `size` prop and
 * colored via `className` (text-* utilities flow through currentColor).
 * Accessible via role="status" + a visually-hidden label.
 */
export function Spinner({
  size = 'md',
  className = '',
  label = 'Loading',
}: SpinnerProps) {
  return (
    <span role="status" className={`inline-flex ${className}`}>
      <svg
        className={`animate-spin ${sizeClasses[size]}`}
        viewBox="0 0 24 24"
        fill="none"
        aria-hidden="true"
      >
        <circle
          className="opacity-25"
          cx="12"
          cy="12"
          r="10"
          stroke="currentColor"
          strokeWidth="4"
        />
        <path
          className="opacity-75"
          fill="currentColor"
          d="M4 12a8 8 0 018-8V0C5.373 0 0 5.373 0 12h4zm2 5.291A7.962 7.962 0 014 12H0c0 3.042 1.135 5.824 3 7.938l3-2.647z"
        />
      </svg>
      <span className="sr-only">{label}</span>
    </span>
  );
}
