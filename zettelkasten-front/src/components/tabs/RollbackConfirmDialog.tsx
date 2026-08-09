import React from 'react';
import { ConfirmDialog } from '../ui/ConfirmDialog';
import { Spinner } from '../ui/Spinner';
import { AuditChange, parseAuditEvent, formatDate } from '../../utils/audit';

interface RollbackConfirmDialogProps {
  isOpen: boolean;
  onClose: () => void;
  onConfirm: () => void;
  cardTitle: string;
  auditEvent: any;
  isLoading?: boolean;
}

/**
 * RollbackConfirmDialog — thin specialization on ui/ConfirmDialog: shared
 * shell/API from the primitive, custom change-preview body here.
 */
export function RollbackConfirmDialog({
  isOpen,
  onClose,
  onConfirm,
  cardTitle,
  auditEvent,
  isLoading = false,
}: RollbackConfirmDialogProps) {
  if (!isOpen || !auditEvent) return null;

  const changes: AuditChange[] = parseAuditEvent(auditEvent);
  const eventType = auditEvent.details?.change_type || 'unknown';

  // Generate a summary of what will be restored
  const getRestoreSummary = () => {
    if (eventType.toLowerCase() === 'create') {
      return 'restore to the initial state when this card was created';
    }
    if (changes.length === 0) {
      return 'restore with no changes';
    }
    const fieldNames = changes.map((c) => {
      switch (c.field.toLowerCase()) {
        case 'title':
          return 'title';
        case 'body':
          return 'content';
        case 'cardid':
          return 'card ID';
        case 'link':
          return 'link';
        default:
          return c.field.toLowerCase();
      }
    });
    if (fieldNames.length === 1) {
      return `restore the ${fieldNames[0]}`;
    }
    if (fieldNames.length === 2) {
      return `restore the ${fieldNames[0]} and ${fieldNames[1]}`;
    }
    return `restore ${changes.length} fields`;
  };

  return (
    <ConfirmDialog
      isOpen={isOpen}
      onClose={onClose}
      onConfirm={onConfirm}
      title="Confirm Restore"
      confirmText={
        isLoading ? (
          <>
            <Spinner size="sm" className="text-white -ml-1 mr-2" />
            Restoring...
          </>
        ) : (
          'Restore'
        )
      }
      cancelText="Cancel"
      variant="info"
      isLoading={isLoading}
    >
      <p className="text-sm text-gray-600 mb-4">
        You are about to <strong>{getRestoreSummary()}</strong> for card:
      </p>

      <div className="bg-gray-50 border border-gray-200 rounded-md p-3 mb-4">
        <p className="font-medium text-gray-900">{cardTitle}</p>
      </div>

      <div className="text-sm text-gray-600 mb-3">
        <p className="font-medium text-gray-700 mb-1">Target version:</p>
        <p className="text-gray-500">
          {eventType.toLowerCase() === 'create'
            ? 'Initial state'
            : formatDate(auditEvent.created_at)}
        </p>
      </div>

      {/* Warning */}
      <div className="bg-yellow-50 border border-yellow-200 rounded-md p-3">
        <div className="flex">
          <svg
            className="w-5 h-5 text-yellow-400 mr-2 flex-shrink-0"
            fill="currentColor"
            viewBox="0 0 20 20"
          >
            <path
              fillRule="evenodd"
              d="M8.257 3.099c.765-1.36 2.722-1.36 3.486 0l5.58 9.92c.75 1.334-.213 2.98-1.742 2.98H4.42c-1.53 0-2.493-1.646-1.743-2.98l5.58-9.92zM11 13a1 1 0 11-2 0 1 1 0 012 0zm-1-8a1 1 0 00-1 1v3a1 1 0 002 0V6a1 1 0 00-1-1z"
              clipRule="evenodd"
            />
          </svg>
          <div className="text-sm text-yellow-800">
            <p className="font-medium mb-1">Important notes:</p>
            <ul className="list-disc list-inside space-y-1 text-yellow-700">
              <li>This will create a new version (a new audit event)</li>
              <li>Current changes will be preserved in history</li>
              <li>You can undo this restore by restoring again</li>
            </ul>
          </div>
        </div>
      </div>

      {/* Changes preview */}
      {changes.length > 0 && eventType.toLowerCase() !== 'create' && (
        <div className="mt-4">
          <p className="text-sm font-medium text-gray-700 mb-2">
            Changes that will be made:
          </p>
          <div className="bg-gray-50 border border-gray-200 rounded-md p-3 space-y-2">
            {changes.map((change, idx) => {
              const fieldName = change.field.toLowerCase();
              const displayField =
                fieldName === 'cardid' ? 'card ID' : fieldName;
              return (
                <div key={idx} className="text-sm">
                  <span className="font-medium text-gray-700">
                    {displayField}:
                  </span>
                  <span className="text-gray-600 ml-2">
                    will revert to &quot;{String(change.from || '(empty)')}
                    &quot;
                  </span>
                </div>
              );
            })}
          </div>
        </div>
      )}
    </ConfirmDialog>
  );
}
