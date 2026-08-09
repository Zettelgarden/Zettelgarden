import React from 'react';

export interface LabelProps {
  htmlFor?: string;
  children: React.ReactNode;
  className?: string;
}

/**
 * Label — the shared form-label styling with optional htmlFor association.
 */
export function Label({ htmlFor, children, className = '' }: LabelProps) {
  return (
    <label
      htmlFor={htmlFor}
      className={`block text-sm font-medium text-gray-700 mb-1 ${className}`}
    >
      {children}
    </label>
  );
}
