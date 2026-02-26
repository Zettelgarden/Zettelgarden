import React from "react";
import { Email } from "../../api/email";
import { EmailRow } from "./EmailRow";

interface EmailListProps {
  emails: Email[];
  loading: boolean;
  onEmailClick: (email: Email) => void;
  onArchive?: (email: Email) => void;
  onCreateTask?: (email: Email) => void;
  selectedEmailIds?: Set<number>;
  onToggleSelect?: (email: Email) => void;
}

/**
 * Email list component that displays:
 * - Loading state when fetching emails
 * - Empty state when no emails
 * - List of EmailRow components for each email
 * - Quick action buttons for each email (archive, create task)
 * - Selection checkboxes when selection mode is enabled
 */
export function EmailList({ emails, loading, onEmailClick, onArchive, onCreateTask, selectedEmailIds, onToggleSelect }: EmailListProps) {
  if (loading) {
    return (
      <div
        style={{
          display: 'flex',
          alignItems: 'center',
          justifyContent: 'center',
          padding: '48px 16px',
        }}
      >
        <div
          style={{
            display: 'flex',
            flexDirection: 'column',
            alignItems: 'center',
            gap: '16px',
          }}
        >
          {/* Loading spinner */}
          <div
            style={{
              width: '40px',
              height: '40px',
              border: '3px solid #e5e7eb',
              borderTopColor: '#3b82f6',
              borderRadius: '50%',
              animation: 'spin 1s linear infinite',
            }}
          />
          <div
            style={{
              fontSize: '14px',
              color: '#6b7280',
            }}
          >
            Loading emails...
          </div>
        </div>
        <style>{`
          @keyframes spin {
            to { transform: rotate(360deg); }
          }
        `}</style>
      </div>
    );
  }

  if (!emails || emails.length === 0) {
    return (
      <div
        style={{
          display: 'flex',
          alignItems: 'center',
          justifyContent: 'center',
          padding: '48px 16px',
        }}
      >
        <div
          style={{
            textAlign: 'center',
          }}
        >
          {/* Empty state icon */}
          <div
            style={{
              fontSize: '48px',
              marginBottom: '16px',
            }}
          >
            📧
          </div>
          <div
            style={{
              fontSize: '16px',
              fontWeight: '500',
              color: '#374151',
              marginBottom: '8px',
            }}
          >
            No emails found
          </div>
          <div
            style={{
              fontSize: '14px',
              color: '#6b7280',
            }}
          >
            Emails will appear here once you sync your email account
          </div>
        </div>
      </div>
    );
  }

  return (
    <div>
      {emails.map((email) => (
        <EmailRow
          key={email.id}
          email={email}
          onClick={() => onEmailClick(email)}
          onArchive={onArchive}
          onCreateTask={onCreateTask}
          isSelected={selectedEmailIds?.has(email.id)}
          onToggleSelect={onToggleSelect}
        />
      ))}
    </div>
  );
}
