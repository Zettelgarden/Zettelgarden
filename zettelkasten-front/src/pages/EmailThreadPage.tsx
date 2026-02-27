import React, { useState, useEffect, useCallback } from "react";
import { useParams, useNavigate } from "react-router-dom";
import {
  getEmailThread,
  markThreadAsRead,
  archiveThread,
  updateEmailStatus,
  EmailThread,
  Email,
} from "../api/email";
import { setDocumentTitle } from "../utils/title";

/**
 * Email Thread Page
 *
 * Displays a full email conversation thread with all messages in chronological order.
 * Features:
 * - View all messages in the thread
 * - Expand/collapse individual messages
 * - Thread-level actions (archive all, mark all read)
 * - Navigation between messages
 */
export function EmailThreadPage() {
  const { threadId } = useParams<{ threadId: string }>();
  const navigate = useNavigate();

  const [thread, setThread] = useState<EmailThread | null>(null);
  const [loading, setLoading] = useState<boolean>(true);
  const [error, setError] = useState<string>("");
  const [expandedMessages, setExpandedMessages] = useState<Set<number>>(new Set());
  const [archiving, setArchiving] = useState(false);
  const [markingRead, setMarkingRead] = useState(false);

  /**
   * Fetch thread data
   */
  const fetchThread = useCallback(async () => {
    if (!threadId) return;

    setLoading(true);
    setError("");
    try {
      const threadData = await getEmailThread(threadId);
      setThread(threadData);

      // Set document title
      setDocumentTitle(threadData.subject || "Email Thread");

      // Expand all messages by default
      if (threadData.messages && threadData.messages.length > 0) {
        setExpandedMessages(new Set(threadData.messages.map(e => e.id)));
      }
    } catch (err: any) {
      console.error("Failed to fetch email thread:", err);
      setError(err.message || "Failed to load email thread");
    } finally {
      setLoading(false);
    }
  }, [threadId]);

  useEffect(() => {
    fetchThread();
  }, [fetchThread]);

  /**
   * Toggle message expansion
   */
  const toggleMessage = useCallback((emailId: number) => {
    setExpandedMessages(prev => {
      const next = new Set(prev);
      if (next.has(emailId)) {
        next.delete(emailId);
      } else {
        next.add(emailId);
      }
      return next;
    });
  }, []);

  /**
   * Handle clicking on an individual email to view details
   */
  const handleEmailClick = useCallback((emailId: number) => {
    navigate(`/app/emails/${emailId}`);
  }, [navigate]);

  /**
   * Handle archive entire thread
   */
  const handleArchiveThread = async () => {
    if (!threadId || archiving) return;

    setArchiving(true);
    setError("");
    try {
      await archiveThread(threadId);
      // Navigate back to inbox
      navigate("/app/emails");
    } catch (err: any) {
      console.error("Failed to archive thread:", err);
      setError(err.message || "Failed to archive thread");
    } finally {
      setArchiving(false);
    }
  };

  /**
   * Handle mark all as read
   */
  const handleMarkAllRead = async () => {
    if (!threadId || markingRead) return;

    setMarkingRead(true);
    setError("");
    try {
      await markThreadAsRead(threadId);
      // Refresh thread data
      await fetchThread();
    } catch (err: any) {
      console.error("Failed to mark thread as read:", err);
      setError(err.message || "Failed to mark thread as read");
    } finally {
      setMarkingRead(false);
    }
  };

  /**
   * Handle archive individual email
   */
  const handleArchiveEmail = async (email: Email) => {
    const newStatus = email.status === 'archived' ? 'unprocessed' : 'archived';
    try {
      await updateEmailStatus(email.id, newStatus);
      // Refresh thread data
      await fetchThread();
    } catch (err: any) {
      console.error("Failed to archive email:", err);
      setError(err.message || "Failed to archive email");
    }
  };

  /**
   * Format date for display
   */
  const formatDate = (dateString: string | undefined): string => {
    if (!dateString) return "";
    const date = new Date(dateString);
    return date.toLocaleString();
  };

  /**
   * Get avatar letter
   */
  const getAvatarLetter = (name: string | undefined, address: string | undefined): string => {
    if (name) return name.charAt(0).toUpperCase();
    if (address) return address.charAt(0).toUpperCase();
    return "?";
  };

  /**
   * Get display name
   */
  const getDisplayName = (name: string | undefined, address: string | undefined): string => {
    return name || address || "Unknown";
  };

  if (loading) {
    return (
      <div style={{ padding: '24px', textAlign: 'center' }}>
        <div style={{ fontSize: '14px', color: '#6b7280' }}>Loading thread...</div>
      </div>
    );
  }

  if (error && !thread) {
    return (
      <div style={{ padding: '24px' }}>
        <div
          style={{
            padding: '16px',
            backgroundColor: '#fee2e2',
            borderRadius: '8px',
            color: '#991b1b',
            marginBottom: '16px',
          }}
        >
          {error}
        </div>
        <button
          onClick={() => navigate("/app/emails")}
          style={{
            padding: '8px 16px',
            borderRadius: '6px',
            border: '1px solid #d1d5db',
            backgroundColor: '#ffffff',
            color: '#374151',
            cursor: 'pointer',
          }}
        >
          Back to Inbox
        </button>
      </div>
    );
  }

  if (!thread) {
    return null;
  }

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
        {/* Navigation and actions */}
        <div
          style={{
            display: 'flex',
            alignItems: 'center',
            justifyContent: 'space-between',
            marginBottom: '16px',
          }}
        >
          <button
            onClick={() => navigate("/app/emails")}
            style={{
              padding: '8px 16px',
              fontSize: '14px',
              fontWeight: '500',
              borderRadius: '8px',
              border: '1px solid #d1d5db',
              backgroundColor: '#ffffff',
              color: '#374151',
              cursor: 'pointer',
              display: 'flex',
              alignItems: 'center',
              gap: '6px',
              transition: 'all 0.15s ease',
            }}
            onMouseEnter={(e) => {
              e.currentTarget.style.backgroundColor = '#f9fafb';
              e.currentTarget.style.borderColor = '#9ca3af';
            }}
            onMouseLeave={(e) => {
              e.currentTarget.style.backgroundColor = '#ffffff';
              e.currentTarget.style.borderColor = '#d1d5db';
            }}
          >
            <span>←</span>
            <span>Back to Inbox</span>
          </button>

          <div style={{ display: 'flex', gap: '8px' }}>
            <button
              onClick={handleMarkAllRead}
              disabled={markingRead || thread.unread_count === 0}
              style={{
                padding: '8px 16px',
                fontSize: '14px',
                fontWeight: '500',
                borderRadius: '8px',
                border: '1px solid #d1d5db',
                backgroundColor: '#ffffff',
                color: '#374151',
                cursor: (markingRead || thread.unread_count === 0) ? 'not-allowed' : 'pointer',
                opacity: (markingRead || thread.unread_count === 0) ? 0.5 : 1,
                transition: 'all 0.15s ease',
              }}
              onMouseEnter={(e) => {
                if (!markingRead && thread.unread_count > 0) {
                  e.currentTarget.style.backgroundColor = '#f9fafb';
                  e.currentTarget.style.borderColor = '#9ca3af';
                }
              }}
              onMouseLeave={(e) => {
                e.currentTarget.style.backgroundColor = '#ffffff';
                e.currentTarget.style.borderColor = '#d1d5db';
              }}
            >
              {markingRead ? 'Marking...' : `Mark All Read (${thread.unread_count})`}
            </button>
            <button
              onClick={handleArchiveThread}
              disabled={archiving}
              style={{
                padding: '8px 16px',
                fontSize: '14px',
                fontWeight: '500',
                borderRadius: '8px',
                border: 'none',
                backgroundColor: archiving ? '#9ca3af' : '#ef4444',
                color: '#ffffff',
                cursor: archiving ? 'not-allowed' : 'pointer',
                opacity: archiving ? 0.7 : 1,
                transition: 'all 0.15s ease',
              }}
              onMouseEnter={(e) => {
                if (!archiving) {
                  e.currentTarget.style.backgroundColor = '#dc2626';
                }
              }}
              onMouseLeave={(e) => {
                e.currentTarget.style.backgroundColor = archiving ? '#9ca3af' : '#ef4444';
              }}
            >
              {archiving ? 'Archiving...' : 'Archive Thread'}
            </button>
          </div>
        </div>

        {/* Thread subject */}
        <h1
          style={{
            fontSize: '24px',
            fontWeight: '700',
            color: '#111827',
            margin: 0,
            marginBottom: '8px',
          }}
        >
          {thread.subject || '(No subject)'}
        </h1>

        {/* Thread metadata */}
        <div
          style={{
            fontSize: '14px',
            color: '#6b7280',
            display: 'flex',
            gap: '16px',
          }}
        >
          <span>{thread.message_count} {thread.message_count === 1 ? 'message' : 'messages'}</span>
          <span>{thread.participant_count} {thread.participant_count === 1 ? 'participant' : 'participants'}</span>
          {thread.unread_count > 0 && (
            <span style={{ color: '#3b82f6', fontWeight: '500' }}>
              {thread.unread_count} unread
            </span>
          )}
        </div>
      </div>

      {/* Error message */}
      {error && (
        <div
          style={{
            margin: '16px 24px',
            padding: '12px 16px',
            backgroundColor: '#fee2e2',
            borderRadius: '8px',
            color: '#991b1b',
            fontSize: '14px',
            display: 'flex',
            alignItems: 'center',
            gap: '8px',
          }}
        >
          <span>⚠</span>
          <span>{error}</span>
        </div>
      )}

      {/* Messages list */}
      <div
        style={{
          flex: 1,
          overflowY: 'auto',
          padding: '16px 24px',
          backgroundColor: '#f9fafb',
        }}
      >
        <div style={{ maxWidth: '900px', margin: '0 auto' }}>
          {thread.messages && thread.messages.length > 0 ? (
            thread.messages.map((email, index) => {
              const isExpanded = expandedMessages.has(email.id);
              const avatarLetter = getAvatarLetter(email.from_name, email.from_address);
              const displayName = getDisplayName(email.from_name, email.from_address);

              return (
                <div
                  key={email.id}
                  style={{
                    marginBottom: '16px',
                    backgroundColor: '#ffffff',
                    borderRadius: '8px',
                    border: '1px solid #e5e7eb',
                    overflow: 'hidden',
                  }}
                >
                  {/* Message header */}
                  <div
                    onClick={() => toggleMessage(email.id)}
                    style={{
                      padding: '16px',
                      cursor: 'pointer',
                      borderBottom: isExpanded ? '1px solid #e5e7eb' : 'none',
                      transition: 'background-color 0.15s ease',
                    }}
                    onMouseEnter={(e) => {
                      e.currentTarget.style.backgroundColor = '#f9fafb';
                    }}
                    onMouseLeave={(e) => {
                      e.currentTarget.style.backgroundColor = '#transparent';
                    }}
                  >
                    <div style={{ display: 'flex', alignItems: 'flex-start', gap: '12px' }}>
                      {/* Avatar */}
                      <div
                        style={{
                          width: '40px',
                          height: '40px',
                          borderRadius: '50%',
                          backgroundColor: email.is_read ? '#9ca3af' : '#3b82f6',
                          color: '#ffffff',
                          display: 'flex',
                          alignItems: 'center',
                          justifyContent: 'center',
                          fontSize: '16px',
                          fontWeight: '600',
                          flexShrink: 0,
                        }}
                      >
                        {avatarLetter}
                      </div>

                      {/* Header content */}
                      <div style={{ flex: 1, minWidth: 0 }}>
                        <div
                          style={{
                            display: 'flex',
                            alignItems: 'center',
                            justifyContent: 'space-between',
                            marginBottom: '4px',
                          }}
                        >
                          <div style={{ display: 'flex', alignItems: 'center', gap: '8px' }}>
                            {!email.is_read && (
                              <div
                                style={{
                                  width: '8px',
                                  height: '8px',
                                  borderRadius: '50%',
                                  backgroundColor: '#3b82f6',
                                }}
                              />
                            )}
                            <span
                              style={{
                                fontSize: '14px',
                                fontWeight: email.is_read ? '500' : '600',
                                color: '#111827',
                              }}
                            >
                              {displayName}
                            </span>
                            {email.from_name && email.from_address && email.from_name !== email.from_address && (
                              <span
                                style={{
                                  fontSize: '13px',
                                  color: '#6b7280',
                                }}
                              >
                                &lt;{email.from_address}&gt;
                              </span>
                            )}
                          </div>
                          <span
                            style={{
                              fontSize: '12px',
                              color: '#9ca3af',
                            }}
                          >
                            {formatDate(email.received_at)}
                          </span>
                        </div>

                        {/* Subject (if different from thread subject) */}
                        {email.subject && email.subject !== thread.subject && (
                          <div
                            style={{
                              fontSize: '14px',
                              color: '#374151',
                              marginBottom: '4px',
                            }}
                          >
                            {email.subject}
                          </div>
                        )}

                        {/* Preview */}
                        <div
                          style={{
                            fontSize: '13px',
                            color: '#6b7280',
                            overflow: 'hidden',
                            textOverflow: 'ellipsis',
                            whiteSpace: 'nowrap',
                          }}
                        >
                          {email.body_text ? (
                            email.body_text.substring(0, 100) + (email.body_text.length > 100 ? '...' : '')
                          ) : (
                            '(No content)'
                          )}
                        </div>
                      </div>

                      {/* Expand/collapse indicator */}
                      <div
                        style={{
                          color: '#9ca3af',
                          fontSize: '12px',
                          transition: 'transform 0.15s ease',
                          transform: isExpanded ? 'rotate(180deg)' : 'rotate(0deg)',
                        }}
                      >
                        ▼
                      </div>
                    </div>
                  </div>

                  {/* Expanded message body */}
                  {isExpanded && (
                    <div style={{ padding: '16px' }}>
                      {/* Message body */}
                      <div
                        style={{
                          fontSize: '14px',
                          lineHeight: '1.6',
                          color: '#374151',
                          whiteSpace: 'pre-wrap',
                          marginBottom: '16px',
                        }}
                      >
                        {email.body_text || <em>(No text content)</em>}
                      </div>

                      {/* Message actions */}
                      <div
                        style={{
                          display: 'flex',
                          gap: '8px',
                          paddingTop: '12px',
                          borderTop: '1px solid #f3f4f6',
                        }}
                      >
                        <button
                                                          onClick={(e) => {
                            e.stopPropagation();
                            handleEmailClick(email.id);
                          }}
                                                          style={{
                            padding: '6px 12px',
                                                            fontSize: '13px',
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
                                                            e.currentTarget.style.borderColor = '#9ca3af';
                                                          }}
                                                          onMouseLeave={(e) => {
                                                            e.currentTarget.style.backgroundColor = '#ffffff';
                                                            e.currentTarget.style.borderColor = '#d1d5db';
                                                          }}
                                                        >
                          View Details
                        </button>
                        <button
                          onClick={(e) => {
                            e.stopPropagation();
                            handleArchiveEmail(email);
                          }}
                          style={{
                            padding: '6px 12px',
                            fontSize: '13px',
                            fontWeight: '500',
                            borderRadius: '6px',
                            border: 'none',
                            backgroundColor: email.status === 'archived' ? '#fef3c7' : '#f3f4f6',
                            color: email.status === 'archived' ? '#92400e' : '#6b7280',
                            cursor: 'pointer',
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
                        >
                          {email.status === 'archived' ? 'Unarchive' : 'Archive'}
                        </button>
                      </div>
                    </div>
                  )}
                </div>
              );
            })
          ) : (
            <div
              style={{
                padding: '48px',
                textAlign: 'center',
                color: '#6b7280',
              }}
            >
              No messages in this thread
            </div>
          )}
        </div>
      </div>
    </div>
  );
}
