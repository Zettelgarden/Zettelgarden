import React from 'react';
import { Label } from './Label';

export interface FieldProps {
  /** Label text; pass htmlFor to associate it with the control's id */
  label?: React.ReactNode;
  /** id of the control this field's label points to */
  htmlFor?: string;
  /** Validation error text (turns the field red and replaces help) */
  error?: string;
  /** Helper text shown under the control when there is no error */
  help?: React.ReactNode;
  /** Renders a red asterisk next to the label */
  required?: boolean;
  /** The control (Input/Select/…) */
  children: React.ReactNode;
  className?: string;
}

/**
 * Field — label + control + error/help slot in one block, keeping the
 * label/control association consistent across forms.
 */
export function Field({
  label,
  htmlFor,
  error,
  help,
  required = false,
  children,
  className = '',
}: FieldProps) {
  return (
    <div className={`mb-4 ${className}`}>
      {label !== undefined && (
        <Label htmlFor={htmlFor}>
          {label}
          {required && <span className="text-red-500"> *</span>}
        </Label>
      )}
      {children}
      {error ? (
        <p className="text-sm text-red-600 mt-1" role="alert">
          {error}
        </p>
      ) : help !== undefined ? (
        <p className="text-sm text-gray-500 mt-1">{help}</p>
      ) : null}
    </div>
  );
}
