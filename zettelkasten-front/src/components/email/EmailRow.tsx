import React from "react";
import { Email } from "../../api/email";

interface EmailRowProps {
  email: Email;
  onClick: () => void;
  onArchive?: (email: Email) => void;
  onCreateTask?: (email: Email) => void;
  isSelected?: boolean;
  onToggleSelect?: (email: Email) => void;
}

/**
 * Formats an email date for display
 * - Shows time if today
 * - Shows "Yesterday" if yesterday
 * - Shows weekday if within past 6 days
 * - Shows date otherwise
 */
function formatDate(dateString: string | undefined): string {
  if (!dateString) return "";

  const date = new Date(dateString);
  const now = new Date();
  const today = new Date(now.getFullYear(), now.getMonth(), now.getDate());
  const emailDate = new Date(date.getFullYear(), date.getMonth(), date.getDate());

  const diffTime = today.getTime() - emailDate.getTime();
  const diffDays = Math.floor(diffTime / (1000 * 60 * 60 * 24));

  if (diffDays === 0) {
    // Today - show time
    return date.toLocaleTimeString([], { hour: '2-digit', minute: '2-digit' });
  } else if (diffDays === 1) {
    // Yesterday
    return "Yesterday";
  } else if (diffDays < 7) {
    // Within past week - show weekday
    return date.toLocaleDateString([], { weekday: 'short' });
  } else {
    // Older - show date
    return date.toLocaleDateString([], { month: 'short', day: 'numeric' });
  }
}

/**
 * Get status badge color based on email status
 */
function getStatusBadgeColor(status: string): string {
  switch (status.toLowerCase()) {
    case 'unprocessed':
      return '#fef3c7'; // amber-100
    case 'triaged':
      return '#dbeafe'; // blue-100
    case 'processed':
      return '#d1fae5'; // green-100
    case 'archived':
      return '#e5e7eb'; // gray-200
    default:
      return '#f3f4f6'; // gray-100
  }
}

/**
 * Get status badge text color based on email status
 */
function getStatusBadgeTextColor(status: string): string {
  switch (status.toLowerCase()) {
    case 'unprocessed':
      return '#92400e'; // amber-800
    case 'triaged':
      return '#1e40af'; // blue-800
    case 'processed':
      return '#065f46'; // green-800
    case 'archived':
      return '#374151'; // gray-700
    default:
      return '#374151'; // gray-700
  }
}

/**
 * Individual email row component displaying:
 * - Checkbox for selection
 * - Unread indicator (blue dot)
 * - Sender avatar (first letter in circle)
 * - From name
 * - Subject (bold if unread)
 * - Date (formatted)
 * - Status badge
 * - Quick action buttons (archive, create task)
 */
export function EmailRow({ email, onClick, onArchive, onCreateTask, isSelected, onToggleSelect }: EmailRowProps) {
  // Get first letter of sender name or email address for avatar
  const avatarLetter = email.from_name
    ? email.from_name.charAt(0).toUpperCase()
    : email.from_address
      ? email.from_address.charAt(0).toUpperCase()
      : "?";

  const displayName = email.from_name || email.from_address || "Unknown";
  const displaySubject = email.subject || "(No subject)";
  const isUnread = !email.is_read;

  const handleRowClick = (e: React.MouseEvent) => {
    // Don't trigger onClick if clicking the checkbox or action buttons
    if ((e.target as HTMLElement).closest('input[type="checkbox"]') ||
        (e.target as HTMLElement).closest('button')) {
      return;
    }
    onClick();
  };

  const handleCheckboxChange = (e: React.ChangeEvent<HTMLInputElement>) => {
    e.stopPropagation();
    if (onToggleSelect) {
      onToggleSelect(email);
    }
  };

  return (
    <div
      onClick={handleRowClick}
      style={{
        display: 'flex',
        alignItems: 'center',
        padding: '12px 16px',
        borderBottom: '1px solid #e5e7eb',
        cursor: 'pointer',
        transition: 'background-color 0.15s ease',
        backgroundColor: isSelected ? '#eff6ff' : (isUnread ? '#ffffff' : '#f9fafb'),
      }}
      onMouseEnter={(e) => {
        if (!isSelected) {
          e.currentTarget.style.backgroundColor = '#f3f4f6';
        }
      }}
      onMouseLeave={(e) => {
        if (!isSelected) {
          e.currentTarget.style.backgroundColor = isUnread ? '#ffffff' : '#f9fafb';
        }
      }}
    >
      {/* Checkbox for selection */}
      {onToggleSelect && (
        <input
          type="checkbox"
          checked={isSelected || false}
          onChange={handleCheckboxChange}
          onClick={(e) => e.stopPropagation()}
          style={{
            width: '16px',
            height: '16px',
            marginRight: '12px',
            cursor: 'pointer',
            accentColor: '#3b82f6',
          }}
        />
      )}

      {/* Unread indicator - blue dot */}
      {isUnread && (
        <div
          style={{
            width: '8px',
            height: '8px',
            borderRadius: '50%',
            backgroundColor: '#3b82f6',
            flexShrink: 0,
            marginRight: '8px',
          }}
        />
      )}

      {/* Spacer for read emails to maintain alignment */}
      {!isUnread && (
        <div style={{ width: '8px', marginRight: '8px' }} />
      )}

      {/* Sender avatar */}
      <div
        style={{
          width: '40px',
          height: '40px',
          borderRadius: '50%',
          backgroundColor: isUnread ? '#3b82f6' : '#9ca3af',
          color: '#ffffff',
          display: 'flex',
          alignItems: 'center',
          justifyContent: 'center',
          fontSize: '16px',
          fontWeight: '600',
          flexShrink: 0,
          marginRight: '12px',
        }}
      >
        {avatarLetter}
      </div>

      {/* Email content */}
      <div style={{ flex: 1, minWidth: 0 }}>
        {/* From name */}
        <div
          style={{
            fontSize: '14px',
            fontWeight: isUnread ? '600' : '500',
            color: '#111827',
            overflow: 'hidden',
            textOverflow: 'ellipsis',
            whiteSpace: 'nowrap',
            marginBottom: '2px',
          }}
        >
          {displayName}
        </div>

        {/* Subject */}
        <div
          style={{
            fontSize: '13px',
            fontWeight: isUnread ? '500' : '400',
            color: isUnread ? '#1f2937' : '#6b7280',
            overflow: 'hidden',
            textOverflow: 'ellipsis',
            whiteSpace: 'nowrap',
          }}
        >
          {displaySubject}
        </div>
      </div>

      {/* Quick action buttons */}
      {(onArchive || onCreateTask) && (
        <div
          style={{
            display: 'flex',
            gap: '8px',
            marginRight: '12px',
          }}
        >
          {onCreateTask && (
            <button
              onClick={(e) => {
                e.stopPropagation();
                onCreateTask(email);
              }}
              style={{
                padding: '6px 10px',
                fontSize: '13px',
                fontWeight: '500',
                borderRadius: '6px',
                border: 'none',
                backgroundColor: '#eff6ff',
                color: '#1d4ed8',
                cursor: 'pointer',
                display: 'flex',
                alignItems: 'center',
                gap: '4px',
                transition: 'all 0.15s ease',
              }}
              onMouseEnter={(e) => {
                e.currentTarget.style.backgroundColor = '#dbeafe';
              }}
              onMouseLeave={(e) => {
                e.currentTarget.style.backgroundColor = '#eff6ff';
              }}
              title="Create task from this email"
            >
              <span>✓</span>
              <span>Task</span>
            </button>
          )}
          {onArchive && (
            <button
              onClick={(e) => {
                e.stopPropagation();
                onArchive(email);
              }}
              style={{
                padding: '6px 10px',
                fontSize: '13px',
                fontWeight: '500',
                borderRadius: '6px',
                border: 'none',
                backgroundColor: email.status === 'archived' ? '#fef3c7' : '#f3f4f6',
                color: email.status === 'archived' ? '#92400e' : '#6b7280',
                cursor: 'pointer',
                display: 'flex',
                alignItems: 'center',
                gap: '4px',
                transition: 'all 0.15s ease',
              }}
              onMouseEnter={(e) => {
                if (email.status !== 'archived') {
                  e.currentTarget.style.backgroundColor = '#e5e7eb';
                } else {
                  e.currentTarget.style.backgroundColor = '#fde68a';
                }
              }}
              onMouseLeave={(e) => {
                e.currentTarget.style.backgroundColor = email.status === 'archived' ? '#fef3c7' : '#f3f4f6';
              }}
              title={email.status === 'archived' ? 'Unarchive' : 'Archive'}
            >
              <span>{email.status === 'archived' ? '↱' : '📁'}</span>
              <span>{email.status === 'archived' ? 'Unarchive' : 'Archive'}</span>
            </button>
          )}
        </div>
      )}

      {/* Date and status */}
      <div
        style={{
          display: 'flex',
          flexDirection: 'column',
          alignItems: 'flex-end',
          gap: '4px',
          marginLeft: '12px',
          flexShrink: 0,
        }}
      >
        {/* Date */}
        <div
          style={{
            fontSize: '12px',
            color: '#9ca3af',
            fontWeight: '500',
          }}
        >
          {formatDate(email.received_at)}
        </div>

        {/* Status badge */}
        <div
          style={{
            backgroundColor: getStatusBadgeColor(email.status),
            color: getStatusBadgeTextColor(email.status),
            fontSize: '11px',
            fontWeight: '600',
            padding: '2px 8px',
            borderRadius: '9999px',
            textTransform: 'capitalize',
            whiteSpace: 'nowrap',
          }}
        >
          {email.status}
        </div>
      </div>
    </div>
  );
}
