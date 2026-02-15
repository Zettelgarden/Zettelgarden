import React, { useState, useEffect, useCallback } from "react";
import { useNavigate } from "react-router-dom";
import { listEmails, Email } from "../api/email";
import { EmailList } from "../components/email/EmailList";
import { setDocumentTitle } from "../utils/title";

type StatusFilter = "all" | "unprocessed" | "triaged";

/**
 * Email Inbox Page
 *
 * Displays a list of emails with filtering by status.
 * Supports filtering by:
 * - All: Shows all emails
 * - Unprocessed: Shows emails that need attention
 * - Triaged: Shows emails that have been reviewed
 */
export function EmailInboxPage() {
  const navigate = useNavigate();

  // State management
  const [emails, setEmails] = useState<Email[]>([]);
  const [loading, setLoading] = useState<boolean>(true);
  const [statusFilter, setStatusFilter] = useState<StatusFilter>("all");
  const [total, setTotal] = useState<number>(0);

  /**
   * Fetch emails from the API with the current status filter
   */
  const fetchEmails = useCallback(async () => {
    setLoading(true);
    try {
      const response = await listEmails({
        status: statusFilter === "all" ? undefined : statusFilter,
        limit: 50,
      });
      setEmails(response.emails);
      setTotal(response.total);
    } catch (error) {
      console.error("Failed to fetch emails:", error);
      setEmails([]);
      setTotal(0);
    } finally {
      setLoading(false);
    }
  }, [statusFilter]);

  // Fetch emails when component mounts or filter changes
  useEffect(() => {
    fetchEmails();
  }, [fetchEmails]);

  // Update document title based on filter
  useEffect(() => {
    const titleMap: Record<StatusFilter, string> = {
      all: "Email Inbox",
      unprocessed: "Unprocessed Emails",
      triaged: "Triaged Emails",
    };
    setDocumentTitle(titleMap[statusFilter]);
  }, [statusFilter]);

  /**
   * Handle email click - navigate to email detail page
   */
  const handleEmailClick = (email: Email) => {
    navigate(`/app/emails/${email.id}`);
  };

  /**
   * Handle filter button click
   */
  const handleFilterChange = (filter: StatusFilter) => {
    setStatusFilter(filter);
  };

  return (
    <div style={{ display: 'flex', flexDirection: 'column', height: '100vh' }}>
      {/* Header */}
      <div
        style={{
          borderBottom: '1px solid #e5e7eb',
          backgroundColor: '#ffffff',
          padding: '16px 24px',
        }}
      >
        {/* Title and total count */}
        <div
          style={{
            display: 'flex',
            alignItems: 'center',
            justifyContent: 'space-between',
            marginBottom: '16px',
          }}
        >
          <h1
            style={{
              fontSize: '24px',
              fontWeight: '700',
              color: '#111827',
              margin: 0,
            }}
          >
            Email Inbox
          </h1>
          {total > 0 && (
            <div
              style={{
                fontSize: '14px',
                color: '#6b7280',
              }}
            >
              {total} {total === 1 ? 'email' : 'emails'}
            </div>
          )}
        </div>

        {/* Filter buttons */}
        <div
          style={{
            display: 'flex',
            gap: '8px',
          }}
        >
          <FilterButton
            active={statusFilter === "all"}
            onClick={() => handleFilterChange("all")}
          >
            All
          </FilterButton>
          <FilterButton
            active={statusFilter === "unprocessed"}
            onClick={() => handleFilterChange("unprocessed")}
          >
            Unprocessed
          </FilterButton>
          <FilterButton
            active={statusFilter === "triaged"}
            onClick={() => handleFilterChange("triaged")}
          >
            Triaged
          </FilterButton>
        </div>
      </div>

      {/* Email list */}
      <div
        style={{
          flex: 1,
          overflowY: 'auto',
          backgroundColor: '#ffffff',
        }}
      >
        <EmailList
          emails={emails}
          loading={loading}
          onEmailClick={handleEmailClick}
        />
      </div>
    </div>
  );
}

/**
 * Filter button component
 */
interface FilterButtonProps {
  active: boolean;
  onClick: () => void;
  children: React.ReactNode;
}

function FilterButton({ active, onClick, children }: FilterButtonProps) {
  return (
    <button
      onClick={onClick}
      style={{
        padding: '8px 16px',
        fontSize: '14px',
        fontWeight: '500',
        borderRadius: '8px',
        border: 'none',
        cursor: 'pointer',
        transition: 'all 0.15s ease',
        backgroundColor: active ? '#3b82f6' : '#f3f4f6',
        color: active ? '#ffffff' : '#374151',
      }}
      onMouseEnter={(e) => {
        if (!active) {
          e.currentTarget.style.backgroundColor = '#e5e7eb';
        }
      }}
      onMouseLeave={(e) => {
        if (!active) {
          e.currentTarget.style.backgroundColor = '#f3f4f6';
        }
      }}
    >
      {children}
    </button>
  );
}
