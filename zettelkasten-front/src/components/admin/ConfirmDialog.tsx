import React from 'react';

/**
 * Props for the ConfirmDialog component
 */
export interface ConfirmDialogProps {
  /** Whether the dialog is visible */
  isOpen: boolean;
  /** Callback when user confirms */
  onConfirm: () => void;
  /** Callback when user cancels or dismisses */
  onCancel: () => void;
  /** Dialog title */
  title: string;
  /** Main message/question to display */
  message: string;
  /** Optional detail text */
  details?: string;
  /** Text for confirm button (default: "Confirm") */
  confirmText?: string;
  /** Text for cancel button (default: "Cancel") */
  cancelText?: string;
  /** Severity level for styling (default: "warning") */
  severity?: 'info' | 'warning' | 'danger';
  /** Whether to require checkbox confirmation (default: false) */
  requireCheckbox?: boolean;
  /** Checkbox label text (when requireCheckbox is true) */
  checkboxLabel?: string;
}

/**
 * Maps severity to Tailwind CSS classes
 */
const severityStyles = {
  info: {
    container: 'border-blue-200 bg-blue-50',
    title: 'text-blue-900',
    icon: 'ℹ️',
    confirmButton: 'bg-blue-600 hover:bg-blue-700 text-white',
  },
  warning: {
    container: 'border-yellow-200 bg-yellow-50',
    title: 'text-yellow-900',
    icon: '⚠️',
    confirmButton: 'bg-yellow-600 hover:bg-yellow-700 text-white',
  },
  danger: {
    container: 'border-red-200 bg-red-50',
    title: 'text-red-900',
    icon: '🚨',
    confirmButton: 'bg-red-600 hover:bg-red-700 text-white',
  },
};

/**
 * ConfirmDialog - A reusable confirmation dialog for admin actions.
 *
 * Provides:
 * - Consistent styling for all admin confirmations
 * - Severity-based visual cues (info/warning/danger)
 * - Optional checkbox requirement for particularly dangerous actions
 * - Escape key and backdrop click to cancel
 *
 * @example
 * ```tsx
 * <ConfirmDialog
 *   isOpen={showDialog}
 *   title="Unsubscribe User"
 *   message={`Are you sure you want to unsubscribe ${email}?`}
 *   onConfirm={handleConfirm}
 *   onCancel={() => setShowDialog(false)}
 *   severity="danger"
 *   confirmText="Yes, unsubscribe"
 * />
 * ```
 */
export function ConfirmDialog({
  isOpen,
  onConfirm,
  onCancel,
  title,
  message,
  details,
  confirmText = 'Confirm',
  cancelText = 'Cancel',
  severity = 'warning',
  requireCheckbox = false,
  checkboxLabel = 'I understand this action cannot be undone',
}: ConfirmDialogProps) {
  const [checkboxChecked, setCheckboxChecked] = React.useState(false);
  const [isClosing, setIsClosing] = React.useState(false);

  // Reset checkbox when dialog opens
  React.useEffect(() => {
    if (isOpen) {
      setCheckboxChecked(false);
      setIsClosing(false);
    }
  }, [isOpen]);

  // Handle escape key
  React.useEffect(() => {
    if (!isOpen) return;

    const handleEscape = (e: KeyboardEvent) => {
      if (e.key === 'Escape' && !isClosing) {
        handleClose();
      }
    };

    document.addEventListener('keydown', handleEscape);
    return () => document.removeEventListener('keydown', handleEscape);
  }, [isOpen, isClosing]);

  // Prevent body scroll when dialog is open
  React.useEffect(() => {
    if (isOpen) {
      document.body.style.overflow = 'hidden';
    } else {
      document.body.style.overflow = '';
    }
    return () => {
      document.body.style.overflow = '';
    };
  }, [isOpen]);

  const handleConfirm = () => {
    setIsClosing(true);
    onConfirm();
  };

  const handleClose = () => {
    if (isClosing) return;
    setIsClosing(true);
    // Small delay for animation
    setTimeout(() => {
      onCancel();
    }, 150);
  };

  if (!isOpen) return null;

  const styles = severityStyles[severity];
  const canConfirm = !requireCheckbox || checkboxChecked;

  return (
    <div
      className="fixed inset-0 z-50 flex items-center justify-center p-4"
      onClick={handleClose}
    >
      {/* Backdrop */}
      <div
        className={`absolute inset-0 bg-black/50 transition-opacity duration-300 ${
          isOpen ? 'opacity-100' : 'opacity-0'
        }`}
      />

      {/* Dialog */}
      <div
        className={`relative bg-white rounded-lg shadow-xl border-2 max-w-md w-full p-6 transition-all duration-300 ${
          styles.container
        } ${isClosing ? 'scale-95 opacity-0' : 'scale-100 opacity-100'}`}
        onClick={(e) => e.stopPropagation()}
      >
        {/* Header */}
        <div className="flex items-start gap-4">
          <div className="text-2xl flex-shrink-0">{styles.icon}</div>
          <div className="flex-1">
            <h3 className={`text-lg font-semibold ${styles.title}`}>{title}</h3>
            <p className="mt-2 text-sm text-gray-700">{message}</p>
            {details && (
              <p className="mt-2 text-xs text-gray-600 bg-black/5 p-2 rounded">
                {details}
              </p>
            )}
          </div>
        </div>

        {/* Checkbox (if required) */}
        {requireCheckbox && (
          <div className="mt-4">
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

        {/* Actions */}
        <div className="mt-6 flex justify-end gap-3">
          <button
            onClick={handleClose}
            disabled={isClosing}
            className="px-4 py-3 min-h-[44px] text-sm font-medium text-gray-700 bg-white border border-gray-300 rounded hover:bg-gray-50 disabled:opacity-50 disabled:cursor-not-allowed transition-colors"
          >
            {cancelText}
          </button>
          <button
            onClick={handleConfirm}
            disabled={!canConfirm || isClosing}
            className={`px-4 py-3 min-h-[44px] text-sm font-medium rounded disabled:opacity-50 disabled:cursor-not-allowed transition-colors ${styles.confirmButton}`}
          >
            {confirmText}
          </button>
        </div>
      </div>
    </div>
  );
}

/**
 * Hook for managing confirm dialog state
 */
export function useConfirmDialog() {
  const [isOpen, setIsOpen] = React.useState(false);
  const [config, setConfig] = React.useState<
    Omit<ConfirmDialogProps, 'isOpen' | 'onCancel'>
  >({
    onConfirm: () => {},
    title: '',
    message: '',
  });

  const confirm = (
    newConfig: Omit<ConfirmDialogProps, 'isOpen' | 'onCancel'>,
  ) => {
    return new Promise<boolean>((resolve) => {
      setConfig({
        ...newConfig,
        onConfirm: () => {
          newConfig.onConfirm();
          resolve(true);
          setIsOpen(false);
        },
      });
      setIsOpen(true);
    });
  };

  const Dialog = () => (
    <ConfirmDialog
      isOpen={isOpen}
      onConfirm={config.onConfirm}
      onCancel={() => setIsOpen(false)}
      title={config.title}
      message={config.message}
      details={config.details}
      confirmText={config.confirmText}
      cancelText={config.cancelText}
      severity={config.severity}
      requireCheckbox={config.requireCheckbox}
      checkboxLabel={config.checkboxLabel}
    />
  );

  return { confirm, Dialog, isOpen };
}
