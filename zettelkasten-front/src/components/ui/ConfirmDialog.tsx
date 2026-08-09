import React, { useEffect, useRef, useState } from 'react';
import { Modal } from './Modal';
import { Button } from './Button';

export interface ConfirmDialogProps {
  isOpen: boolean;
  onClose: () => void;
  onConfirm: () => void;
  title: string;
  message: string;
  /** Optional detail text shown under the message */
  details?: string;
  confirmText?: string;
  cancelText?: string;
  /** Severity styling for the confirm button/icon (default: danger) */
  variant?: 'info' | 'warning' | 'danger';
  /** Require a checkbox before the confirm button is enabled */
  requireCheckbox?: boolean;
  checkboxLabel?: string;
  /** Disables both buttons while an async confirm is in flight */
  isLoading?: boolean;
}

const variantStyles = {
  danger: {
    icon: 'text-red-600',
    confirmButton:
      'bg-red-600 hover:bg-red-700 focus:ring-red-500 disabled:hover:bg-red-600',
  },
  warning: {
    icon: 'text-yellow-600',
    confirmButton:
      'bg-yellow-600 hover:bg-yellow-700 focus:ring-yellow-500 disabled:hover:bg-yellow-600',
  },
  info: {
    icon: 'text-blue-600',
    confirmButton:
      'bg-blue-600 hover:bg-blue-700 focus:ring-blue-500 disabled:hover:bg-blue-600',
  },
} as const;

/**
 * ConfirmDialog — the single shared confirm dialog, built on ui/Modal.
 *
 * API is derived from the previous tasks/ConfirmDialog (isOpen/onClose/
 * onConfirm, variant) with admin/ConfirmDialog's a11y and safety features
 * (Escape, scroll-lock, optional checkbox) folded in via ui/Modal.
 */
export function ConfirmDialog({
  isOpen,
  onClose,
  onConfirm,
  title,
  message,
  details,
  confirmText = 'Confirm',
  cancelText = 'Cancel',
  variant = 'danger',
  requireCheckbox = false,
  checkboxLabel = 'I understand this action cannot be undone',
  isLoading = false,
}: ConfirmDialogProps) {
  const [checkboxChecked, setCheckboxChecked] = useState(false);

  // Reset checkbox each time the dialog opens
  useEffect(() => {
    if (isOpen) {
      setCheckboxChecked(false);
    }
  }, [isOpen]);

  const styles = variantStyles[variant];
  const canConfirm = !requireCheckbox || checkboxChecked;

  const handleConfirm = () => {
    if (isLoading) return;
    onConfirm();
    onClose();
  };

  const isInfo = variant === 'info';
  const iconPath = isInfo
    ? 'M13 16h-1v-4h-1m1-4h.01M21 12a9 9 0 11-18 0 9 9 0 0118 0z'
    : 'M12 9v2m0 4h.01m-6.938 4h13.856c1.54 0 2.502-1.667 1.732-3L13.732 4c-.77-1.333-2.694-1.333-3.464 0L3.34 16c-.77 1.333.192 3 1.732 3z';

  return (
    <Modal open={isOpen} onClose={onClose}>
      {/* Header with icon */}
      <div className="flex items-center mb-4">
        <div className="flex-shrink-0 mr-3">
          <svg
            className={`w-6 h-6 ${styles.icon}`}
            fill="none"
            viewBox="0 0 24 24"
            stroke="currentColor"
          >
            <path
              strokeLinecap="round"
              strokeLinejoin="round"
              strokeWidth={2}
              d={iconPath}
            />
          </svg>
        </div>
        <h3 className="text-lg font-semibold text-gray-900">{title}</h3>
      </div>

      {/* Content */}
      <div className="mb-6">
        <p className="text-sm text-gray-600">{message}</p>
        {details && (
          <p className="mt-2 text-xs text-gray-600 bg-black/5 p-2 rounded">
            {details}
          </p>
        )}
      </div>

      {/* Checkbox (if required) */}
      {requireCheckbox && (
        <div className="mb-4">
          <label className="flex items-start gap-2 cursor-pointer">
            <input
              type="checkbox"
              checked={checkboxChecked}
              onChange={(e) => setCheckboxChecked(e.target.checked)}
              className="mt-1 w-4 h-4 text-gray-600 rounded focus:ring-2 focus:ring-offset-2"
            />
            <span className="text-sm text-gray-700">{checkboxLabel}</span>
          </label>
        </div>
      )}

      {/* Footer buttons */}
      <div className="flex justify-end space-x-3">
        <Button
          variant="outline"
          onClick={onClose}
          disabled={isLoading}
          className="text-sm"
        >
          {cancelText}
        </Button>
        <button
          type="button"
          onClick={handleConfirm}
          disabled={!canConfirm || isLoading}
          className={`px-4 py-3 min-h-[44px] text-sm font-medium text-white border border-transparent rounded-md focus:outline-none focus:ring-2 focus:ring-offset-2 disabled:opacity-50 disabled:cursor-not-allowed ${styles.confirmButton}`}
        >
          {confirmText}
        </button>
      </div>
    </Modal>
  );
}

type ConfirmConfig = Omit<ConfirmDialogProps, 'isOpen' | 'onClose'>;

/**
 * useConfirmDialog — promise-based confirm helper.
 *
 * `await confirm(...)` resolves `true` on confirm and `false` on cancel/
 * Escape/backdrop-dismiss, so callers never hang.
 */
export function useConfirmDialog() {
  const [isOpen, setIsOpen] = useState(false);
  const [config, setConfig] = useState<ConfirmConfig>({
    onConfirm: () => {},
    title: '',
    message: '',
  });
  const resolveRef = useRef<((value: boolean) => void) | null>(null);

  const confirm = (newConfig: ConfirmConfig) => {
    return new Promise<boolean>((resolve) => {
      // Resolve any previous pending confirm before opening a new one
      resolveRef.current?.(false);
      resolveRef.current = resolve;
      setConfig(newConfig);
      setIsOpen(true);
    });
  };

  const handleClose = () => {
    resolveRef.current?.(false);
    resolveRef.current = null;
    setIsOpen(false);
  };

  const handleConfirm = () => {
    config.onConfirm();
    resolveRef.current?.(true);
    resolveRef.current = null;
    setIsOpen(false);
  };

  const Dialog = () => (
    <ConfirmDialog
      isOpen={isOpen}
      onClose={handleClose}
      onConfirm={handleConfirm}
      title={config.title}
      message={config.message}
      details={config.details}
      confirmText={config.confirmText}
      cancelText={config.cancelText}
      variant={config.variant}
      requireCheckbox={config.requireCheckbox}
      checkboxLabel={config.checkboxLabel}
    />
  );

  return { confirm, Dialog, isOpen };
}
