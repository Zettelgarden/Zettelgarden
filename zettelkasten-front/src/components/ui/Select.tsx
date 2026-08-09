import React from 'react';

export interface SelectProps
  extends React.SelectHTMLAttributes<HTMLSelectElement> {
  /** Red border + focus ring when the field has a validation error */
  hasError?: boolean;
}

const baseClasses =
  'block w-full rounded-md border px-3 py-2 shadow-sm focus:outline-none focus:ring-1 sm:text-sm';

/**
 * Select — the shared dropdown styling, with an error state.
 */
export function Select({
  hasError = false,
  className = '',
  children,
  ...props
}: SelectProps) {
  const stateClasses = hasError
    ? 'border-red-500 focus:border-red-500 focus:ring-red-500'
    : 'border-gray-300 focus:border-blue-500 focus:ring-blue-500';
  return (
    <select
      className={`${baseClasses} ${stateClasses} ${className}`}
      {...props}
    >
      {children}
    </select>
  );
}
