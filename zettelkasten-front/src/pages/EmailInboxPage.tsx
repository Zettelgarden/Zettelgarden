import React, { useState, useEffect, useCallback } from "react";
import { useNavigate } from "react-router-dom";
import { listEmails, createEmailAccount, syncEmailAccount, Email } from "../api/email";
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
  const [accountEmail, setAccountEmail] = useState("");
  const [accountPassword, setAccountPassword] = useState("");
  const [accountError, setAccountError] = useState("");
  const [isAddingAccount, setIsAddingAccount] = useState(false);

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
      setEmails(response.emails ?? []);
      setTotal(response.total ?? 0);
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

  /**
   * Handle adding a new email account
   */
  const handleAddAccount = async () => {
    setIsAddingAccount(true);
    setAccountError("");

    try {
      const account = await createEmailAccount({
        email_address: accountEmail,
        api_token: accountPassword,
      });

      // Clear form
      setAccountEmail("");
      setAccountPassword("");

      // Trigger initial sync
      console.log("Triggering initial sync for account", account.id);
      await syncEmailAccount(account.id);

      // Reload emails
      fetchEmails();
    } catch (error: any) {
      setAccountError(error.message || "Failed to add account");
    } finally {
      setIsAddingAccount(false);
    }
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
        {emails.length === 0 && !loading ? (
          <div style={{ padding: '48px', textAlign: 'center' }}>
            <h2 style={{ fontSize: '20px', marginBottom: '16px' }}>Connect Your Fastmail Account</h2>
            <p style={{ marginBottom: '24px' }}>Add your Fastmail account to start syncing emails</p>

            <div style={{ maxWidth: '400px', margin: '0 auto', textAlign: 'left' }}>
              <div style={{ marginBottom: '16px' }}>
                <label style={{ display: 'block', marginBottom: '8px', fontWeight: 'bold' }}>
                  Email Address
                </label>
                <input
                  type="email"
                  value={accountEmail}
                  onChange={(e) => setAccountEmail(e.target.value)}
                  placeholder="you@fastmail.com"
                  style={{ width: '100%', padding: '8px', border: '1px solid #d1d5db', borderRadius: '4px' }}
                />
              </div>

              <div style={{ marginBottom: '16px' }}>
                <label style={{ display: 'block', marginBottom: '8px', fontWeight: 'bold' }}>
                  API Token
                </label>
                <input
                  type="password"
                  value={accountPassword}
                  onChange={(e) => setAccountPassword(e.target.value)}
                  placeholder="Your Fastmail API token"
                  style={{ width: '100%', padding: '8px', border: '1px solid #d1d5db', borderRadius: '4px' }}
                />
                <p style={{ fontSize: '12px', color: '#6b7280', marginTop: '4px' }}>
                  Create an API token in Fastmail → Settings → API
                </p>
              </div>

              {accountError && (
                <div style={{ marginBottom: '16px', padding: '12px', backgroundColor: '#fee', borderRadius: '4px', color: '#c00' }}>
                  {accountError}
                </div>
              )}

              <button
                onClick={handleAddAccount}
                disabled={isAddingAccount || !accountEmail || !accountPassword}
                style={{
                  padding: '10px 20px',
                  backgroundColor: isAddingAccount ? '#9ca3af' : '#3b82f6',
                  color: 'white',
                  border: 'none',
                  borderRadius: '4px',
                  cursor: isAddingAccount || !accountEmail || !accountPassword ? 'not-allowed' : 'pointer',
                  opacity: isAddingAccount || !accountEmail || !accountPassword ? 0.6 : 1
                }}
              >
                {isAddingAccount ? 'Adding...' : 'Connect Account'}
              </button>
            </div>
          </div>
        ) : (
          <EmailList
            emails={emails}
            loading={loading}
            onEmailClick={handleEmailClick}
          />
        )}
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
