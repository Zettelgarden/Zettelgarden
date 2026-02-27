import React, { useState, useEffect, useCallback } from "react";
import { useNavigate, useSearchParams } from "react-router-dom";
import {
  listEmails,
  listEmailAccounts,
  createEmailAccount,
  syncEmailAccount,
  updateEmailStatus,
  batchArchiveEmails,
  batchConvertEmails,
  batchCreateTasks,
  getTopSenders,
  searchEmails,
  SenderInfo,
  Email,
  EmailAccount,
  EmailSearchResult
} from "../api/email";
import { EmailList } from "../components/email/EmailList";
import { CreateTaskWindow } from "../components/tasks/CreateTaskWindow";
import { setDocumentTitle } from "../utils/title";

type StatusFilter = "all" | "unprocessed" | "triaged" | "archived";
type ViewMode = "flat" | "threaded";

/**
 * Email Inbox Page
 *
 * Displays a list of emails with filtering by status.
 * Default view shows unprocessed emails that need attention.
 * Supports filtering by:
 * - All: Shows all emails
 * - Unprocessed: Shows emails that need attention (default)
 * - Triaged: Shows emails that have been reviewed
 * - Archived: Shows archived emails
 */
export function EmailInboxPage() {
  const navigate = useNavigate();
  const [searchParams, setSearchParams] = useSearchParams();

  // State management
  const [emails, setEmails] = useState<Email[]>([]);
  const [emailAccounts, setEmailAccounts] = useState<EmailAccount[]>([]);
  const [loading, setLoading] = useState<boolean>(true);
  const [statusFilter, setStatusFilter] = useState<StatusFilter>("unprocessed");
  const [viewMode, setViewMode] = useState<ViewMode>("threaded");
  const [total, setTotal] = useState<number>(0);
  const [topSenders, setTopSenders] = useState<SenderInfo[]>([]);
  const [selectedSender, setSelectedSender] = useState<string | null>(null);
  const [showSenderDropdown, setShowSenderDropdown] = useState(false);
  const [loadingSenders, setLoadingSenders] = useState(false);
  const [accountEmail, setAccountEmail] = useState("");
  const [accountPassword, setAccountPassword] = useState("");
  const [accountError, setAccountError] = useState("");
  const [isAddingAccount, setIsAddingAccount] = useState(false);
  const [isSyncing, setIsSyncing] = useState(false);
  const [showCreateTaskWindow, setShowCreateTaskWindow] = useState(false);
  const [emailForTask, setEmailForTask] = useState<Email | null>(null);

  // Batch operations state
  const [selectedEmailIds, setSelectedEmailIds] = useState<Set<number>>(new Set());
  const [isAllSelected, setIsAllSelected] = useState(false);
  const [batchOperationLoading, setBatchOperationLoading] = useState(false);
  const [batchError, setBatchError] = useState("");
  const [showBatchConfirm, setShowBatchConfirm] = useState(false);
  const [pendingBatchOperation, setPendingBatchOperation] = useState<string | null>(null);

  // Search state
  const [searchQuery, setSearchQuery] = useState("");
  const [searchResults, setSearchResults] = useState<EmailSearchResult[]>([]);
  const [isSearching, setIsSearching] = useState(false);
  const [showSearchResults, setShowSearchResults] = useState(false);

  /**
   * Fetch email accounts
   */
  const fetchEmailAccounts = useCallback(async () => {
    try {
      const accounts = await listEmailAccounts();
      setEmailAccounts(accounts ?? []);
    } catch (error) {
      console.error("Failed to fetch email accounts:", error);
      setEmailAccounts([]);
    }
  }, []);

  /**
   * Fetch top senders
   */
  const fetchTopSenders = useCallback(async () => {
    setLoadingSenders(true);
    try {
      const response = await getTopSenders(statusFilter === "all" ? undefined : statusFilter, 10);
      setTopSenders(response.senders ?? []);
    } catch (error) {
      console.error("Failed to fetch top senders:", error);
      setTopSenders([]);
    } finally {
      setLoadingSenders(false);
    }
  }, [statusFilter]);

  /**
   * Fetch emails from the API with the current status filter
   */
  const fetchEmails = useCallback(async () => {
    setLoading(true);
    try {
      const response = await listEmails({
        status: statusFilter === "all" ? undefined : statusFilter,
        from_address: selectedSender || undefined,
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
  }, [statusFilter, selectedSender]);

  // Initialize from URL params on mount
  useEffect(() => {
    const senderParam = searchParams.get("from_address");
    if (senderParam) {
      setSelectedSender(senderParam);
    }
  }, [searchParams]);

  // Fetch accounts and emails when component mounts
  useEffect(() => {
    fetchEmailAccounts();
    fetchEmails();
  }, [fetchEmails, fetchEmailAccounts]);

  // Fetch top senders when filter changes
  useEffect(() => {
    fetchTopSenders();
  }, [fetchTopSenders]);

  // Fetch emails when filter changes
  useEffect(() => {
    if (emailAccounts && emailAccounts.length > 0) {
      fetchEmails();
    }
  }, [fetchEmails, emailAccounts]);

  // Update document title based on filter
  useEffect(() => {
    const titleMap: Record<StatusFilter, string> = {
      all: "Email Inbox",
      unprocessed: "Unprocessed Emails",
      triaged: "Triaged Emails",
      archived: "Archived Emails",
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
   * Handle thread click - navigate to thread detail page
   */
  const handleThreadClick = (threadId: string) => {
    navigate(`/app/emails/threads/${threadId}`);
  };

  /**
   * Handle filter button click
   */
  const handleFilterChange = (filter: StatusFilter) => {
    setStatusFilter(filter);
  };

  /**
   * Handle sender selection
   */
  const handleSenderSelect = useCallback((senderAddress: string) => {
    setSelectedSender(senderAddress);
    setShowSenderDropdown(false);
    // Update URL params
    if (senderAddress) {
      searchParams.set("from_address", senderAddress);
    } else {
      searchParams.delete("from_address");
    }
    setSearchParams(searchParams);
  }, [searchParams, setSearchParams]);

  /**
   * Handle clear sender filter
   */
  const handleClearSenderFilter = useCallback(() => {
    setSelectedSender(null);
    searchParams.delete("from_address");
    setSearchParams(searchParams);
  }, [searchParams, setSearchParams]);

  /**
   * Handle adding a new email account
   */
  const handleAddAccount = async () => {
    setIsAddingAccount(true);
    setAccountError("");

    try {
      const account = await createEmailAccount({
        email_address: accountEmail,
        app_password: accountPassword,
      });

      // Clear form
      setAccountEmail("");
      setAccountPassword("");

      // Trigger initial sync
      console.log("Triggering initial sync for account", account.id);
      await syncEmailAccount(account.id);

      // Reload accounts and emails
      fetchEmailAccounts();
      fetchEmails();
    } catch (error: any) {
      setAccountError(error.message || "Failed to add account");
    } finally {
      setIsAddingAccount(false);
    }
  };

  /**
   * Handle manual sync of all email accounts
   */
  const handleManualSync = async () => {
    setIsSyncing(true);
    try {
      // Sync all active accounts
      for (const account of (emailAccounts ?? [])) {
        if (account.is_active) {
          console.log("Syncing account", account.id);
          await syncEmailAccount(account.id);
        }
      }
      // Refresh emails after sync
      await fetchEmails();
    } catch (error) {
      console.error("Failed to sync emails:", error);
    } finally {
      setIsSyncing(false);
    }
  };

  /**
   * Handle quick archive from the email list
   */
  const handleQuickArchive = async (email: Email) => {
    const newStatus = email.status === 'archived' ? 'unprocessed' : 'archived';
    try {
      await updateEmailStatus(email.id, newStatus);
      // Refresh the email list
      await fetchEmails();
    } catch (error) {
      console.error("Failed to archive email:", error);
    }
  };

  /**
   * Handle create task from email
   */
  const handleCreateTaskFromEmail = (email: Email) => {
    setEmailForTask(email);
    setShowCreateTaskWindow(true);
  };

  /**
   * Handle close create task window
   */
  const handleCloseTaskWindow = () => {
    setShowCreateTaskWindow(false);
    setEmailForTask(null);
  };

  /**
   * Toggle email selection
   */
  const handleToggleSelect = useCallback((email: Email) => {
    setSelectedEmailIds((prev) => {
      const next = new Set(prev);
      if (next.has(email.id)) {
        next.delete(email.id);
      } else {
        next.add(email.id);
      }
      // Update isAllSelected state
      setIsAllSelected(next.size === emails.length && emails.length > 0);
      return next;
    });
  }, [emails.length]);

  /**
   * Toggle select all emails
   */
  const handleToggleSelectAll = useCallback(() => {
    if (isAllSelected) {
      setSelectedEmailIds(new Set());
      setIsAllSelected(false);
    } else {
      setSelectedEmailIds(new Set(emails.map(e => e.id)));
      setIsAllSelected(true);
    }
  }, [isAllSelected, emails]);

  /**
   * Clear selection
   */
  const handleClearSelection = useCallback(() => {
    setSelectedEmailIds(new Set());
    setIsAllSelected(false);
  }, []);

  /**
   * Handle batch archive
   */
  const handleBatchArchive = useCallback(async (archive: boolean) => {
    if (selectedEmailIds.size === 0) return;

    const status = archive ? "archived" : "unprocessed";
    const action = archive ? "archive" : "unarchive";

    setBatchOperationLoading(true);
    setBatchError("");

    try {
      const emailIdsArray = Array.from(selectedEmailIds);
      await batchArchiveEmails({ email_ids: emailIdsArray, status });

      // Clear selection and refresh
      handleClearSelection();
      await fetchEmails();
    } catch (error: any) {
      setBatchError(error.message || `Failed to ${action} emails`);
      setTimeout(() => setBatchError(""), 3000);
    } finally {
      setBatchOperationLoading(false);
    }
  }, [selectedEmailIds, fetchEmails, handleClearSelection]);

  /**
   * Handle batch convert to cards
   */
  const handleBatchConvert = useCallback(async () => {
    if (selectedEmailIds.size === 0) return;

    setBatchOperationLoading(true);
    setBatchError("");

    try {
      const emailIdsArray = Array.from(selectedEmailIds);
      const result = await batchConvertEmails({
        email_ids: emailIdsArray,
        // Use default title/body conversion
      });

      // Clear selection and refresh
      handleClearSelection();
      await fetchEmails();

      // Show success message
      if (result.fail_count > 0) {
        setBatchError(`Converted ${result.success_count} emails, ${result.fail_count} failed`);
        setTimeout(() => setBatchError(""), 5000);
      }
    } catch (error: any) {
      setBatchError(error.message || "Failed to convert emails");
      setTimeout(() => setBatchError(""), 3000);
    } finally {
      setBatchOperationLoading(false);
    }
  }, [selectedEmailIds, fetchEmails, handleClearSelection]);

  /**
   * Handle batch create tasks
   */
  const handleBatchCreateTasks = useCallback(async () => {
    if (selectedEmailIds.size === 0) return;

    setBatchOperationLoading(true);
    setBatchError("");

    try {
      const emailIdsArray = Array.from(selectedEmailIds);
      const result = await batchCreateTasks({ email_ids: emailIdsArray });

      // Clear selection and refresh
      handleClearSelection();
      await fetchEmails();

      // Show success message
      if (result.fail_count > 0) {
        setBatchError(`Created ${result.success_count} tasks, ${result.fail_count} failed`);
        setTimeout(() => setBatchError(""), 5000);
      }
    } catch (error: any) {
      setBatchError(error.message || "Failed to create tasks");
      setTimeout(() => setBatchError(""), 3000);
    } finally {
      setBatchOperationLoading(false);
    }
  }, [selectedEmailIds, fetchEmails, handleClearSelection]);

  /**
   * Handle batch operation with confirmation
   */
  const handleBatchOperationWithConfirm = useCallback((operation: string) => {
    setPendingBatchOperation(operation);
    setShowBatchConfirm(true);
  }, []);

  /**
   * Confirm batch operation
   */
  const handleConfirmBatchOperation = useCallback(async () => {
    setShowBatchConfirm(false);

    switch (pendingBatchOperation) {
      case "archive":
        await handleBatchArchive(true);
        break;
      case "unarchive":
        await handleBatchArchive(false);
        break;
      case "convert":
        await handleBatchConvert();
        break;
      case "createTasks":
        await handleBatchCreateTasks();
        break;
    }

    setPendingBatchOperation(null);
  }, [pendingBatchOperation, handleBatchArchive, handleBatchConvert, handleBatchCreateTasks]);

  /**
   * Cancel batch operation
   */
  const handleCancelBatchOperation = useCallback(() => {
    setShowBatchConfirm(false);
    setPendingBatchOperation(null);
  }, []);

  /**
   * Handle email search
   */
  const handleEmailSearch = useCallback(async (query: string) => {
    if (!query.trim()) {
      setShowSearchResults(false);
      setSearchResults([]);
      return;
    }

    setIsSearching(true);
    setSearchQuery(query);
    setShowSearchResults(true);

    try {
      const response = await searchEmails({
        search_term: query,
        page: 1,
        per_page: 20,
      });
      setSearchResults(response.results ?? []);
    } catch (error) {
      console.error("Failed to search emails:", error);
      setSearchResults([]);
    } finally {
      setIsSearching(false);
    }
  }, []);

  /**
   * Clear search results
   */
  const handleClearSearch = useCallback(() => {
    setSearchQuery("");
    setSearchResults([]);
    setShowSearchResults(false);
  }, []);

  /**
   * Handle search result click
   */
  const handleSearchResultClick = useCallback((result: EmailSearchResult) => {
    setShowSearchResults(false);
    navigate(`/app/emails/${result.id}`);
  }, [navigate]);

  // Clear selection when filter changes
  useEffect(() => {
    handleClearSelection();
  }, [statusFilter, handleClearSelection]);

  // Keyboard shortcut for select all (Ctrl+A)
  useEffect(() => {
    const handleKeyDown = (e: KeyboardEvent) => {
      if (e.ctrlKey && e.key === "a") {
        e.preventDefault();
        handleToggleSelectAll();
      }
    };

    window.addEventListener("keydown", handleKeyDown);
    return () => window.removeEventListener("keydown", handleKeyDown);
  }, [handleToggleSelectAll]);

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
          <div
            style={{
              display: 'flex',
              alignItems: 'center',
              gap: '16px',
              flex: 1,
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

            {/* Search bar */}
            <div style={{ position: 'relative', maxWidth: '400px', flex: 1 }}>
              <input
                type="text"
                value={searchQuery}
                onChange={(e) => handleEmailSearch(e.target.value)}
                placeholder="Search emails..."
                style={{
                  width: '100%',
                  padding: '8px 12px 8px 36px',
                  fontSize: '14px',
                  borderRadius: '8px',
                  border: '1px solid #d1d5db',
                  backgroundColor: '#ffffff',
                  outline: 'none',
                  transition: 'all 0.15s ease',
                }}
                onFocus={(e) => {
                  e.currentTarget.style.borderColor = '#3b82f6';
                  e.currentTarget.style.boxShadow = '0 0 0 3px rgba(59, 130, 246, 0.1)';
                }}
                onBlur={(e) => {
                  e.currentTarget.style.borderColor = '#d1d5db';
                  e.currentTarget.style.boxShadow = 'none';
                }}
              />
              <span
                style={{
                  position: 'absolute',
                  left: '12px',
                  top: '50%',
                  transform: 'translateY(-50%)',
                  fontSize: '16px',
                  color: '#9ca3af',
                  pointerEvents: 'none',
                }}
              >
                🔍
              </span>
              {searchQuery && (
                <button
                  onClick={handleClearSearch}
                  style={{
                    position: 'absolute',
                    right: '8px',
                    top: '50%',
                    transform: 'translateY(-50%)',
                    background: 'none',
                    border: 'none',
                    color: '#9ca3af',
                    cursor: 'pointer',
                    padding: '4px',
                    fontSize: '16px',
                    display: 'flex',
                    alignItems: 'center',
                  }}
                  onMouseEnter={(e) => {
                    e.currentTarget.style.color = '#6b7280';
                  }}
                  onMouseLeave={(e) => {
                    e.currentTarget.style.color = '#9ca3af';
                  }}
                >
                  ✕
                </button>
              )}

              {/* Search results dropdown */}
              {showSearchResults && (searchQuery || searchResults.length > 0) && (
                <div
                  style={{
                    position: 'absolute',
                    top: '100%',
                    left: 0,
                    right: 0,
                    marginTop: '4px',
                    backgroundColor: '#ffffff',
                    border: '1px solid #d1d5db',
                    borderRadius: '8px',
                    boxShadow: '0 4px 6px rgba(0, 0, 0, 0.1)',
                    zIndex: 100,
                    maxHeight: '300px',
                    overflowY: 'auto',
                  }}
                >
                  {isSearching ? (
                    <div
                      style={{
                        padding: '12px 16px',
                        fontSize: '14px',
                        color: '#6b7280',
                        textAlign: 'center',
                      }}
                    >
                      Searching...
                    </div>
                  ) : searchResults.length === 0 ? (
                    <div
                      style={{
                        padding: '12px 16px',
                        fontSize: '14px',
                        color: '#6b7280',
                        textAlign: 'center',
                      }}
                    >
                      No emails found
                    </div>
                  ) : (
                    searchResults.map((result) => (
                      <div
                        key={result.id}
                        onClick={() => handleSearchResultClick(result)}
                        style={{
                          padding: '12px 16px',
                          cursor: 'pointer',
                          borderBottom: '1px solid #f3f4f6',
                          transition: 'background-color 0.15s ease',
                        }}
                        onMouseEnter={(e) => {
                          e.currentTarget.style.backgroundColor = '#f9fafb';
                        }}
                        onMouseLeave={(e) => {
                          e.currentTarget.style.backgroundColor = 'transparent';
                        }}
                      >
                        <div
                          style={{
                            fontSize: '14px',
                            fontWeight: '500',
                            color: '#111827',
                            marginBottom: '4px',
                          }}
                        >
                          {result.subject || '(No subject)'}
                        </div>
                        <div
                          style={{
                            fontSize: '12px',
                            color: '#6b7280',
                            marginBottom: '4px',
                          }}
                        >
                          From: {result.sender}
                        </div>
                        <div
                          style={{
                            fontSize: '12px',
                            color: '#9ca3af',
                            overflow: 'hidden',
                            textOverflow: 'ellipsis',
                            whiteSpace: 'nowrap',
                          }}
                        >
                          {result.preview}
                        </div>
                      </div>
                    ))
                  )}
                </div>
              )}
            </div>
          </div>
          <div
            style={{
              display: 'flex',
              alignItems: 'center',
              gap: '16px',
            }}
          >
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
            <button
              onClick={handleManualSync}
              disabled={isSyncing || !emailAccounts || emailAccounts.length === 0}
              style={{
                padding: '8px 16px',
                fontSize: '14px',
                fontWeight: '500',
                borderRadius: '8px',
                border: '1px solid #d1d5db',
                backgroundColor: isSyncing ? '#f3f4f6' : '#ffffff',
                color: isSyncing ? '#9ca3af' : '#374151',
                cursor: isSyncing || !emailAccounts || emailAccounts.length === 0 ? 'not-allowed' : 'pointer',
                opacity: isSyncing || !emailAccounts || emailAccounts.length === 0 ? 0.6 : 1,
                display: 'flex',
                alignItems: 'center',
                gap: '6px',
                transition: 'all 0.15s ease',
              }}
              onMouseEnter={(e) => {
                if (!isSyncing && emailAccounts && emailAccounts.length > 0) {
                  e.currentTarget.style.backgroundColor = '#f9fafb';
                  e.currentTarget.style.borderColor = '#9ca3af';
                }
              }}
              onMouseLeave={(e) => {
                if (!isSyncing && emailAccounts && emailAccounts.length > 0) {
                  e.currentTarget.style.backgroundColor = '#ffffff';
                  e.currentTarget.style.borderColor = '#d1d5db';
                }
              }}
            >
              <span style={{ fontSize: '16px' }}>
                {isSyncing ? '⟳' : '↻'}
              </span>
              {isSyncing ? 'Syncing...' : 'Sync'}
            </button>
          </div>
        </div>

        {/* Filter buttons */}
        <div
          style={{
            display: 'flex',
            gap: '8px',
            alignItems: 'center',
            flexWrap: 'wrap',
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
          <FilterButton
            active={statusFilter === "archived"}
            onClick={() => handleFilterChange("archived")}
          >
            Archived
          </FilterButton>

          {/* View mode toggle */}
          <div
            style={{
              display: 'flex',
              backgroundColor: '#f3f4f6',
              borderRadius: '8px',
              padding: '2px',
              marginLeft: '16px',
            }}
          >
            <button
              onClick={() => setViewMode('threaded')}
              style={{
                padding: '6px 12px',
                fontSize: '13px',
                fontWeight: '500',
                borderRadius: '6px',
                border: 'none',
                backgroundColor: viewMode === 'threaded' ? '#ffffff' : 'transparent',
                color: '#374151',
                cursor: 'pointer',
                transition: 'all 0.15s ease',
                boxShadow: viewMode === 'threaded' ? '0 1px 2px rgba(0, 0, 0, 0.05)' : 'none',
              }}
              onMouseEnter={(e) => {
                if (viewMode !== 'threaded') {
                  e.currentTarget.style.backgroundColor = '#e5e7eb';
                }
              }}
              onMouseLeave={(e) => {
                if (viewMode !== 'threaded') {
                  e.currentTarget.style.backgroundColor = 'transparent';
                }
              }}
              title="Group emails by conversation thread"
            >
              💬 Threads
            </button>
            <button
              onClick={() => setViewMode('flat')}
              style={{
                padding: '6px 12px',
                fontSize: '13px',
                fontWeight: '500',
                borderRadius: '6px',
                border: 'none',
                backgroundColor: viewMode === 'flat' ? '#ffffff' : 'transparent',
                color: '#374151',
                cursor: 'pointer',
                transition: 'all 0.15s ease',
                boxShadow: viewMode === 'flat' ? '0 1px 2px rgba(0, 0, 0, 0.05)' : 'none',
              }}
              onMouseEnter={(e) => {
                if (viewMode !== 'flat') {
                  e.currentTarget.style.backgroundColor = '#e5e7eb';
                }
              }}
              onMouseLeave={(e) => {
                if (viewMode !== 'flat') {
                  e.currentTarget.style.backgroundColor = 'transparent';
                }
              }}
              title="Show all emails individually"
            >
              📄 Flat
            </button>
          </div>
        </div>

        {/* Sender filter */}
        {(topSenders.length > 0 || selectedSender) && (
          <div
            style={{
              marginTop: '12px',
              display: 'flex',
              alignItems: 'center',
              gap: '12px',
              flexWrap: 'wrap',
            }}
          >
            {/* Sender filter dropdown */}
            <div style={{ position: 'relative' }}>
              <button
                onClick={() => setShowSenderDropdown(!showSenderDropdown)}
                disabled={loadingSenders}
                style={{
                  padding: '8px 16px',
                  fontSize: '14px',
                  fontWeight: '500',
                  borderRadius: '8px',
                  border: '1px solid #d1d5db',
                  backgroundColor: '#ffffff',
                  color: '#374151',
                  cursor: loadingSenders ? 'not-allowed' : 'pointer',
                  display: 'flex',
                  alignItems: 'center',
                  gap: '6px',
                  transition: 'all 0.15s ease',
                  opacity: loadingSenders ? 0.6 : 1,
                }}
                onMouseEnter={(e) => {
                  if (!loadingSenders) {
                    e.currentTarget.style.backgroundColor = '#f9fafb';
                    e.currentTarget.style.borderColor = '#9ca3af';
                  }
                }}
                onMouseLeave={(e) => {
                  if (!loadingSenders) {
                    e.currentTarget.style.backgroundColor = '#ffffff';
                    e.currentTarget.style.borderColor = '#d1d5db';
                  }
                }}
              >
                <span>👤</span>
                <span>Filter by sender</span>
                <span style={{ fontSize: '12px' }}>▼</span>
              </button>

              {/* Sender dropdown */}
              {showSenderDropdown && (
                <div
                  style={{
                    position: 'absolute',
                    top: '100%',
                    left: 0,
                    marginTop: '4px',
                    backgroundColor: '#ffffff',
                    border: '1px solid #d1d5db',
                    borderRadius: '8px',
                    boxShadow: '0 4px 6px rgba(0, 0, 0, 0.1)',
                    zIndex: 100,
                    minWidth: '250px',
                    maxHeight: '300px',
                    overflowY: 'auto',
                  }}
                  onMouseLeave={() => setShowSenderDropdown(false)}
                >
                  {loadingSenders ? (
                    <div
                      style={{
                        padding: '12px 16px',
                        fontSize: '14px',
                        color: '#6b7280',
                        textAlign: 'center',
                      }}
                    >
                      Loading...
                    </div>
                  ) : topSenders.length === 0 ? (
                    <div
                      style={{
                        padding: '12px 16px',
                        fontSize: '14px',
                        color: '#6b7280',
                        textAlign: 'center',
                      }}
                    >
                      No senders found
                    </div>
                  ) : (
                    topSenders.map((sender) => (
                      <div
                        key={sender.from_address}
                        onClick={() => handleSenderSelect(sender.from_address)}
                        style={{
                          padding: '10px 16px',
                          fontSize: '14px',
                          color: '#374151',
                          cursor: 'pointer',
                          borderBottom: '1px solid #f3f4f6',
                          display: 'flex',
                          alignItems: 'center',
                          justifyContent: 'space-between',
                          transition: 'background-color 0.15s ease',
                        }}
                        onMouseEnter={(e) => {
                          e.currentTarget.style.backgroundColor = '#f9fafb';
                        }}
                        onMouseLeave={(e) => {
                          e.currentTarget.style.backgroundColor = 'transparent';
                        }}
                      >
                        <div style={{ display: 'flex', flexDirection: 'column' }}>
                          <span style={{ fontWeight: '500' }}>
                            {sender.from_name || sender.from_address}
                          </span>
                          {sender.from_name && sender.from_name !== sender.from_address && (
                            <span style={{ fontSize: '12px', color: '#6b7280' }}>
                              {sender.from_address}
                            </span>
                          )}
                        </div>
                        <span
                          style={{
                            fontSize: '12px',
                            color: '#6b7280',
                            backgroundColor: '#f3f4f6',
                            padding: '2px 8px',
                            borderRadius: '10px',
                          }}
                        >
                          {sender.count}
                        </span>
                      </div>
                    ))
                  )}
                </div>
              )}
            </div>

            {/* Active sender filter chip */}
            {selectedSender && (
              <div
                style={{
                  display: 'flex',
                  alignItems: 'center',
                  gap: '8px',
                  padding: '6px 12px',
                  backgroundColor: '#eff6ff',
                  borderRadius: '20px',
                  border: '1px solid #bfdbfe',
                }}
              >
                <span style={{ fontSize: '14px', color: '#1d4ed8' }}>
                  From: {selectedSender}
                </span>
                <button
                  onClick={handleClearSenderFilter}
                  style={{
                    background: 'none',
                    border: 'none',
                    color: '#1d4ed8',
                    cursor: 'pointer',
                    padding: '0',
                    fontSize: '16px',
                    display: 'flex',
                    alignItems: 'center',
                  }}
                  onMouseEnter={(e) => {
                    e.currentTarget.style.color = '#1e40af';
                  }}
                  onMouseLeave={(e) => {
                    e.currentTarget.style.color = '#1d4ed8';
                  }}
                >
                  ✕
                </button>
              </div>
            )}
          </div>
        )}

        {/* Bulk action bar */}
        {selectedEmailIds.size > 0 && (
          <div
            style={{
              marginTop: '16px',
              padding: '12px 16px',
              backgroundColor: '#eff6ff',
              borderRadius: '8px',
              display: 'flex',
              alignItems: 'center',
              justifyContent: 'space-between',
              flexWrap: 'wrap',
              gap: '12px',
            }}
          >
            <div
              style={{
                display: 'flex',
                alignItems: 'center',
                gap: '12px',
              }}
            >
              <span
                style={{
                  fontSize: '14px',
                  fontWeight: '500',
                  color: '#1d4ed8',
                }}
              >
                {selectedEmailIds.size} {selectedEmailIds.size === 1 ? 'email' : 'emails'} selected
              </span>

              {/* Select all checkbox */}
              <label
                style={{
                  display: 'flex',
                  alignItems: 'center',
                  gap: '6px',
                  fontSize: '13px',
                  color: '#374151',
                  cursor: 'pointer',
                }}
              >
                <input
                  type="checkbox"
                  checked={isAllSelected}
                  onChange={handleToggleSelectAll}
                  style={{
                    width: '14px',
                    height: '14px',
                    cursor: 'pointer',
                    accentColor: '#3b82f6',
                  }}
                />
                Select all (Ctrl+A)
              </label>
            </div>

            <div
              style={{
                display: 'flex',
                gap: '8px',
                flexWrap: 'wrap',
              }}
            >
              {/* Archive button */}
              {statusFilter !== 'archived' ? (
                <button
                  onClick={() => handleBatchOperationWithConfirm("archive")}
                  disabled={batchOperationLoading}
                  style={{
                    padding: '6px 12px',
                    fontSize: '13px',
                    fontWeight: '500',
                    borderRadius: '6px',
                    border: 'none',
                    backgroundColor: '#f3f4f6',
                    color: '#374151',
                    cursor: batchOperationLoading ? 'not-allowed' : 'pointer',
                    display: 'flex',
                    alignItems: 'center',
                    gap: '4px',
                    transition: 'all 0.15s ease',
                    opacity: batchOperationLoading ? 0.6 : 1,
                  }}
                  onMouseEnter={(e) => {
                    if (!batchOperationLoading) {
                      e.currentTarget.style.backgroundColor = '#e5e7eb';
                    }
                  }}
                  onMouseLeave={(e) => {
                    e.currentTarget.style.backgroundColor = '#f3f4f6';
                  }}
                >
                  <span>📁</span>
                  <span>Archive</span>
                </button>
              ) : (
                <button
                  onClick={() => handleBatchArchive(false)}
                  disabled={batchOperationLoading}
                  style={{
                    padding: '6px 12px',
                    fontSize: '13px',
                    fontWeight: '500',
                    borderRadius: '6px',
                    border: 'none',
                    backgroundColor: '#fef3c7',
                    color: '#92400e',
                    cursor: batchOperationLoading ? 'not-allowed' : 'pointer',
                    display: 'flex',
                    alignItems: 'center',
                    gap: '4px',
                    transition: 'all 0.15s ease',
                    opacity: batchOperationLoading ? 0.6 : 1,
                  }}
                  onMouseEnter={(e) => {
                    if (!batchOperationLoading) {
                      e.currentTarget.style.backgroundColor = '#fde68a';
                    }
                  }}
                  onMouseLeave={(e) => {
                    e.currentTarget.style.backgroundColor = '#fef3c7';
                  }}
                >
                  <span>↱</span>
                  <span>Unarchive</span>
                </button>
              )}

              {/* Convert to cards button */}
              <button
                onClick={() => handleBatchOperationWithConfirm("convert")}
                disabled={batchOperationLoading}
                style={{
                  padding: '6px 12px',
                  fontSize: '13px',
                  fontWeight: '500',
                  borderRadius: '6px',
                  border: 'none',
                  backgroundColor: '#ecfdf5',
                  color: '#065f46',
                  cursor: batchOperationLoading ? 'not-allowed' : 'pointer',
                  display: 'flex',
                  alignItems: 'center',
                  gap: '4px',
                  transition: 'all 0.15s ease',
                  opacity: batchOperationLoading ? 0.6 : 1,
                }}
                onMouseEnter={(e) => {
                  if (!batchOperationLoading) {
                    e.currentTarget.style.backgroundColor = '#d1fae5';
                  }
                }}
                onMouseLeave={(e) => {
                  e.currentTarget.style.backgroundColor = '#ecfdf5';
                }}
              >
                <span>📝</span>
                <span>Convert to Cards</span>
              </button>

              {/* Create tasks button */}
              <button
                onClick={() => handleBatchOperationWithConfirm("createTasks")}
                disabled={batchOperationLoading}
                style={{
                  padding: '6px 12px',
                  fontSize: '13px',
                  fontWeight: '500',
                  borderRadius: '6px',
                  border: 'none',
                  backgroundColor: '#eff6ff',
                  color: '#1d4ed8',
                  cursor: batchOperationLoading ? 'not-allowed' : 'pointer',
                  display: 'flex',
                  alignItems: 'center',
                  gap: '4px',
                  transition: 'all 0.15s ease',
                  opacity: batchOperationLoading ? 0.6 : 1,
                }}
                onMouseEnter={(e) => {
                  if (!batchOperationLoading) {
                    e.currentTarget.style.backgroundColor = '#dbeafe';
                  }
                }}
                onMouseLeave={(e) => {
                  e.currentTarget.style.backgroundColor = '#eff6ff';
                }}
              >
                <span>✓</span>
                <span>Create Tasks</span>
              </button>

              {/* Clear selection button */}
              <button
                onClick={handleClearSelection}
                disabled={batchOperationLoading}
                style={{
                  padding: '6px 12px',
                  fontSize: '13px',
                  fontWeight: '500',
                  borderRadius: '6px',
                  border: 'none',
                  backgroundColor: '#fee2e2',
                  color: '#991b1b',
                  cursor: batchOperationLoading ? 'not-allowed' : 'pointer',
                  display: 'flex',
                  alignItems: 'center',
                  gap: '4px',
                  transition: 'all 0.15s ease',
                  opacity: batchOperationLoading ? 0.6 : 1,
                }}
                onMouseEnter={(e) => {
                  if (!batchOperationLoading) {
                    e.currentTarget.style.backgroundColor = '#fecaca';
                  }
                }}
                onMouseLeave={(e) => {
                  e.currentTarget.style.backgroundColor = '#fee2e2';
                }}
              >
                <span>✕</span>
                <span>Clear</span>
              </button>
            </div>
          </div>
        )}

        {/* Batch operation error message */}
        {batchError && (
          <div
            style={{
              marginTop: '12px',
              padding: '10px 14px',
              backgroundColor: '#fee2e2',
              borderRadius: '6px',
              color: '#991b1b',
              fontSize: '13px',
              display: 'flex',
              alignItems: 'center',
              gap: '8px',
            }}
          >
            <span>⚠</span>
            <span>{batchError}</span>
          </div>
        )}
      </div>

      {/* Email list */}
      <div
        style={{
          flex: 1,
          overflowY: 'auto',
          backgroundColor: '#ffffff',
        }}
      >
        {(!emailAccounts || emailAccounts.length === 0) && !loading ? (
          <div style={{ padding: '48px', textAlign: 'center' }}>
            <h2 style={{ fontSize: '20px', marginBottom: '16px' }}>Connect Your Email Account</h2>
            <p style={{ marginBottom: '24px' }}>Add your email account to start syncing emails</p>

            <div style={{ maxWidth: '400px', margin: '0 auto', textAlign: 'left' }}>
              <div style={{ marginBottom: '16px' }}>
                <label style={{ display: 'block', marginBottom: '8px', fontWeight: 'bold' }}>
                  Email Address
                </label>
                <input
                  type="email"
                  value={accountEmail}
                  onChange={(e) => setAccountEmail(e.target.value)}
                  placeholder="you@example.com"
                  style={{ width: '100%', padding: '8px', border: '1px solid #d1d5db', borderRadius: '4px' }}
                />
              </div>

              <div style={{ marginBottom: '16px' }}>
                <label style={{ display: 'block', marginBottom: '8px', fontWeight: 'bold' }}>
                  App Password
                </label>
                <input
                  type="password"
                  value={accountPassword}
                  onChange={(e) => setAccountPassword(e.target.value)}
                  placeholder="Your Fastmail app password"
                  style={{ width: '100%', padding: '8px', border: '1px solid #d1d5db', borderRadius: '4px' }}
                />
                <p style={{ fontSize: '12px', color: '#6b7280', marginTop: '4px' }}>
                  Create an app password in Fastmail → Settings → Password & Security
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
            onArchive={handleQuickArchive}
            onCreateTask={handleCreateTaskFromEmail}
            selectedEmailIds={selectedEmailIds}
            onToggleSelect={handleToggleSelect}
            viewThreads={viewMode === 'threaded'}
            onThreadClick={handleThreadClick}
          />
        )}
      </div>

      {/* Batch operation confirmation dialog */}
      {showBatchConfirm && (
        <div
          style={{
            position: 'fixed',
            top: 0,
            left: 0,
            right: 0,
            bottom: 0,
            backgroundColor: 'rgba(0, 0, 0, 0.5)',
            display: 'flex',
            alignItems: 'center',
            justifyContent: 'center',
            zIndex: 1000,
          }}
          onClick={handleCancelBatchOperation}
        >
          <div
            style={{
              backgroundColor: '#ffffff',
              borderRadius: '8px',
              padding: '24px',
              maxWidth: '400px',
              width: '90%',
              boxShadow: '0 4px 6px rgba(0, 0, 0, 0.1)',
            }}
            onClick={(e) => e.stopPropagation()}
          >
            <h3
              style={{
                fontSize: '18px',
                fontWeight: '600',
                color: '#111827',
                marginBottom: '12px',
              }}
            >
              Confirm Batch Operation
            </h3>
            <p
              style={{
                fontSize: '14px',
                color: '#6b7280',
                marginBottom: '20px',
              }}
            >
              {pendingBatchOperation === "archive" && `Archive ${selectedEmailIds.size} ${selectedEmailIds.size === 1 ? 'email' : 'emails'}?`}
              {pendingBatchOperation === "convert" && `Convert ${selectedEmailIds.size} ${selectedEmailIds.size === 1 ? 'email' : 'emails'} to cards?`}
              {pendingBatchOperation === "createTasks" && `Create tasks from ${selectedEmailIds.size} ${selectedEmailIds.size === 1 ? 'email' : 'emails'}?`}
            </p>
            <div
              style={{
                display: 'flex',
                gap: '12px',
                justifyContent: 'flex-end',
              }}
            >
              <button
                onClick={handleCancelBatchOperation}
                style={{
                  padding: '8px 16px',
                  fontSize: '14px',
                  fontWeight: '500',
                  borderRadius: '6px',
                  border: '1px solid #d1d5db',
                  backgroundColor: '#ffffff',
                  color: '#374151',
                  cursor: 'pointer',
                  transition: 'all 0.15s ease',
                }}
                onMouseEnter={(e) => {
                  e.currentTarget.style.backgroundColor = '#f9fafb';
                }}
                onMouseLeave={(e) => {
                  e.currentTarget.style.backgroundColor = '#ffffff';
                }}
              >
                Cancel
              </button>
              <button
                onClick={handleConfirmBatchOperation}
                disabled={batchOperationLoading}
                style={{
                  padding: '8px 16px',
                  fontSize: '14px',
                  fontWeight: '500',
                  borderRadius: '6px',
                  border: 'none',
                  backgroundColor: batchOperationLoading ? '#9ca3af' : '#3b82f6',
                  color: '#ffffff',
                  cursor: batchOperationLoading ? 'not-allowed' : 'pointer',
                  opacity: batchOperationLoading ? 0.6 : 1,
                  transition: 'all 0.15s ease',
                }}
                onMouseEnter={(e) => {
                  if (!batchOperationLoading) {
                    e.currentTarget.style.backgroundColor = '#2563eb';
                  }
                }}
                onMouseLeave={(e) => {
                  e.currentTarget.style.backgroundColor = batchOperationLoading ? '#9ca3af' : '#3b82f6';
                }}
              >
                {batchOperationLoading ? 'Processing...' : 'Confirm'}
              </button>
            </div>
          </div>
        </div>
      )}

      {/* Create Task Window */}
      {showCreateTaskWindow && (
        <CreateTaskWindow
          currentCard={null}
          setShowTaskWindow={handleCloseTaskWindow}
          currentFilter={emailForTask?.subject ? `Email: ${emailForTask.subject}` : undefined}
        />
      )}
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
