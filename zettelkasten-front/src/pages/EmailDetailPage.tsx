import React, { useState, useEffect, useMemo } from "react";
import { useParams, useNavigate } from "react-router-dom";
import { getEmail, updateEmailStatus, Email } from "../api/email";
import { setDocumentTitle } from "../utils/title";
import { CreateTaskWindow } from "../components/tasks/CreateTaskWindow";
import { EmailConvertDialog } from "../components/email/EmailConvertDialog";
import { processEmailHtml } from "../utils/emailHtml";

/**
 * Email Detail Page
 *
 * Displays a single email with its full content.
 * HTML content is sanitized using DOMPurify for security.
 * All links in email bodies open in a new tab for security.
 */

// Responsive CSS for email content
const emailStyles = `
  /* Base styles */
  .email-content {
    font-family: -apple-system, BlinkMacSystemFont, 'Segoe UI', Roboto, 'Helvetica Neue', Arial, sans-serif;
    font-size: 15px;
    line-height: 1.6;
    color: #1f2937;
    word-wrap: break-word;
    overflow-wrap: break-word;
  }

  /* Headings */
  .email-content h1, .email-content h2, .email-content h3,
  .email-content h4, .email-content h5, .email-content h6 {
    margin-top: 1.5em;
    margin-bottom: 0.5em;
    font-weight: 600;
    line-height: 1.3;
  }

  .email-content h1 { font-size: 1.75em; }
  .email-content h2 { font-size: 1.5em; }
  .email-content h3 { font-size: 1.25em; }
  .email-content h4 { font-size: 1.1em; }
  .email-content h5 { font-size: 1em; }
  .email-content h6 { font-size: 0.9em; }

  /* Paragraphs and lists */
  .email-content p {
    margin-top: 0;
    margin-bottom: 1em;
  }

  .email-content ul, .email-content ol {
    margin-top: 0;
    margin-bottom: 1em;
    padding-left: 2em;
  }

  .email-content li {
    margin-bottom: 0.25em;
  }

  /* Links */
  .email-content a {
    color: #2563eb;
    text-decoration: underline;
  }

  .email-content a:hover {
    color: #1d4ed8;
  }

  /* Tables - responsive and styled */
  .email-content table {
    max-width: 100%;
    border-collapse: collapse;
    margin: 1em 0;
    font-size: 14px;
    overflow: hidden;
    table-layout: auto;
  }

  .email-content table[width] {
    width: auto !important;
    max-width: 100%;
  }

  .email-content th,
  .email-content td {
    padding: 0.5em;
    border: 1px solid #d1d5db;
    vertical-align: top;
    text-align: left;
    word-wrap: break-word;
    overflow-wrap: break-word;
    min-width: 50px;
  }

  .email-content th {
    background-color: #f3f4f6;
    font-weight: 600;
  }

  .email-content tr:nth-child(even) {
    background-color: #f9fafb;
  }

  /* Images - responsive */
  .email-content img {
    max-width: 100%;
    height: auto;
    display: block;
    margin: 0.5em 0;
  }

  .email-content img[width] {
    width: auto !important;
    max-width: 100%;
  }

  .email-content img[height] {
    height: auto !important;
  }

  /* Blockquotes */
  .email-content blockquote {
    margin: 1em 0;
    padding: 0.5em 1em;
    border-left: 4px solid #d1d5db;
    background-color: #f9fafb;
    color: #6b7280;
  }

  /* Code blocks */
  .email-content pre {
    background-color: #1f2937;
    color: #f3f4f6;
    padding: 1em;
    border-radius: 0.375em;
    overflow-x: auto;
    margin: 1em 0;
  }

  .email-content code {
    font-family: 'Courier New', Courier, monospace;
    font-size: 0.9em;
    background-color: #f3f4f6;
    padding: 0.125em 0.25em;
    border-radius: 0.125em;
  }

  .email-content pre code {
    background-color: transparent;
    padding: 0;
  }

  /* Horizontal rule */
  .email-content hr {
    border: none;
    border-top: 1px solid #e5e7eb;
    margin: 1.5em 0;
  }

  /* Text formatting */
  .email-content strong, .email-content b {
    font-weight: 600;
  }

  .email-content em, .email-content i {
    font-style: italic;
  }

  /* Remove margins from first/last elements */
  .email-content > *:first-child {
    margin-top: 0;
  }

  .email-content > *:last-child {
    margin-bottom: 0;
  }

  /* Fix for email clients that add weird styles */
  .email-content div[style*="font-size"] {
    font-size: inherit !important;
  }
`;
export function EmailDetailPage() {
  const { id } = useParams<{ id: string }>();
  const navigate = useNavigate();

  const [email, setEmail] = useState<Email | null>(null);
  const [loading, setLoading] = useState(true);
  const [error, setError] = useState<string | null>(null);
  const [isArchiving, setIsArchiving] = useState(false);
  const [showCreateTaskWindow, setShowCreateTaskWindow] = useState(false);
  const [showConvertDialog, setShowConvertDialog] = useState(false);

  // Inject email styles on mount
  useEffect(() => {
    const styleId = 'email-content-styles';
    let styleElement = document.getElementById(styleId) as HTMLStyleElement;

    if (!styleElement) {
      styleElement = document.createElement('style');
      styleElement.id = styleId;
      styleElement.textContent = emailStyles;
      document.head.appendChild(styleElement);
    }

    return () => {
      // Clean up styles when component unmounts
      // Only remove if this is the last EmailDetailPage instance
      const remainingElements = document.querySelectorAll('.email-content-wrapper');
      if (remainingElements.length === 0) {
        styleElement.remove();
      }
    };
  }, []);

  useEffect(() => {
    const fetchEmail = async () => {
      if (!id) return;

      setLoading(true);
      setError(null);

      try {
        const data = await getEmail(parseInt(id));
        setEmail(data);
        setDocumentTitle(data.subject || "Email");
      } catch (err: any) {
        console.error("Failed to fetch email:", err);
        setError(err.message || "Failed to load email");
      } finally {
        setLoading(false);
      }
    };

    fetchEmail();
  }, [id]);

  // Process and sanitize HTML content
  const processedHtml = useMemo(() => {
    if (!email?.body_html) {
      return "";
    }

    return processEmailHtml(email.body_html);
  }, [email?.body_html]);

  const handleBack = () => {
    navigate("/app/emails");
  };

  const formatDate = (dateString?: string) => {
    if (!dateString) return "";
    return new Date(dateString).toLocaleString();
  };

  const handleArchive = async () => {
    if (!email || !id) return;

    setIsArchiving(true);
    try {
      const updated = await updateEmailStatus(email.id, "archived");
      setEmail(updated);
    } catch (err: any) {
      console.error("Failed to archive email:", err);
    } finally {
      setIsArchiving(false);
    }
  };

  const handleUnarchive = async () => {
    if (!email || !id) return;

    setIsArchiving(true);
    try {
      const updated = await updateEmailStatus(email.id, "unprocessed");
      setEmail(updated);
    } catch (err: any) {
      console.error("Failed to unarchive email:", err);
    } finally {
      setIsArchiving(false);
    }
  };

  const handleCreateTaskFromEmail = () => {
    if (!email) return;
    setShowCreateTaskWindow(true);
  };

  const handleCloseTaskWindow = () => {
    setShowCreateTaskWindow(false);
  };

  const handleConvertEmail = () => {
    if (!email) return;
    if (email.card_id) {
      // Navigate to existing card
      navigate(`/app/card/${email.card_id}`);
    } else {
      // Open convert dialog
      setShowConvertDialog(true);
    }
  };

  const handleCloseConvertDialog = () => {
    setShowConvertDialog(false);
  };

  const handleEmailConverted = (cardId: number) => {
    // Update email with card_id to show conversion status
    if (email) {
      setEmail({ ...email, card_id: cardId });
    }
  };

  if (loading) {
    return (
      <div style={{ padding: "48px", textAlign: "center" }}>
        <div style={{ fontSize: "16px", color: "#6b7280" }}>Loading email...</div>
      </div>
    );
  }

  if (error || !email) {
    return (
      <div style={{ padding: "48px", textAlign: "center" }}>
        <div style={{ fontSize: "18px", color: "#dc2626", marginBottom: "16px" }}>
          {error || "Email not found"}
        </div>
        <button
          onClick={handleBack}
          style={{
            padding: "10px 20px",
            backgroundColor: "#3b82f6",
            color: "white",
            border: "none",
            borderRadius: "4px",
            cursor: "pointer",
          }}
        >
          Back to Inbox
        </button>
      </div>
    );
  }

  return (
    <div style={{ display: "flex", flexDirection: "column", height: "100vh", backgroundColor: "#ffffff" }}>
      {/* Header */}
      <div
        style={{
          borderBottom: "1px solid #e5e7eb",
          backgroundColor: "#ffffff",
          padding: "16px 24px",
        }}
      >
        <div
          style={{
            display: "flex",
            alignItems: "center",
            justifyContent: "space-between",
            width: "100%",
          }}
        >
          <button
            onClick={handleBack}
            style={{
              padding: "8px 16px",
              fontSize: "14px",
              fontWeight: "500",
              borderRadius: "8px",
              border: "1px solid #d1d5db",
              backgroundColor: "#ffffff",
              color: "#374151",
              cursor: "pointer",
              display: "flex",
              alignItems: "center",
              gap: "6px",
              transition: "all 0.15s ease",
            }}
            onMouseEnter={(e) => {
              e.currentTarget.style.backgroundColor = "#f9fafb";
              e.currentTarget.style.borderColor = "#9ca3af";
            }}
            onMouseLeave={(e) => {
              e.currentTarget.style.backgroundColor = "#ffffff";
              e.currentTarget.style.borderColor = "#d1d5db";
            }}
          >
            ← Back to Inbox
          </button>

          <button
            onClick={email.status === "archived" ? handleUnarchive : handleArchive}
            disabled={isArchiving}
            style={{
              padding: "8px 16px",
              fontSize: "14px",
              fontWeight: "500",
              borderRadius: "8px",
              border: "1px solid #d1d5db",
              backgroundColor: isArchiving ? "#f3f4f6" : email.status === "archived" ? "#fef3c7" : "#ffffff",
              color: isArchiving ? "#9ca3af" : email.status === "archived" ? "#92400e" : "#374151",
              cursor: isArchiving ? "not-allowed" : "pointer",
              opacity: isArchiving ? 0.6 : 1,
              display: "flex",
              alignItems: "center",
              gap: "6px",
              transition: "all 0.15s ease",
            }}
            onMouseEnter={(e) => {
              if (!isArchiving) {
                e.currentTarget.style.backgroundColor = email.status === "archived" ? "#fde68a" : "#f9fafb";
                e.currentTarget.style.borderColor = "#9ca3af";
              }
            }}
            onMouseLeave={(e) => {
              if (!isArchiving) {
                e.currentTarget.style.backgroundColor = email.status === "archived" ? "#fef3c7" : "#ffffff";
                e.currentTarget.style.borderColor = "#d1d5db";
              }
            }}
          >
            {isArchiving ? (
              "..."
            ) : email.status === "archived" ? (
              <>↱ Unarchive</>
            ) : (
              <>📁 Archive</>
            )}
          </button>

          <button
            onClick={handleConvertEmail}
            style={{
              padding: "8px 16px",
              fontSize: "14px",
              fontWeight: "500",
              borderRadius: "8px",
              border: "1px solid #d1d5db",
              backgroundColor: email.card_id ? "#d1fae5" : "#ffffff",
              color: email.card_id ? "#065f46" : "#374151",
              cursor: "pointer",
              display: "flex",
              alignItems: "center",
              gap: "6px",
              transition: "all 0.15s ease",
            }}
            onMouseEnter={(e) => {
              e.currentTarget.style.backgroundColor = email.card_id ? "#a7f3d0" : "#f9fafb";
              e.currentTarget.style.borderColor = "#9ca3af";
            }}
            onMouseLeave={(e) => {
              e.currentTarget.style.backgroundColor = email.card_id ? "#d1fae5" : "#ffffff";
              e.currentTarget.style.borderColor = "#d1d5db";
            }}
          >
            <svg width="16" height="16" viewBox="0 0 24 24" fill="none" stroke="currentColor" strokeWidth="2">
              <rect x="3" y="3" width="18" height="18" rx="2" ry="2" />
              <line x1="9" y1="3" x2="9" y2="21" />
            </svg>
            {email.card_id ? "View Card" : "Convert to Card"}
          </button>

          <button
            onClick={handleCreateTaskFromEmail}
            style={{
              padding: "8px 16px",
              fontSize: "14px",
              fontWeight: "500",
              borderRadius: "8px",
              border: "1px solid #d1d5db",
              backgroundColor: "#ffffff",
              color: "#374151",
              cursor: "pointer",
              display: "flex",
              alignItems: "center",
              gap: "6px",
              transition: "all 0.15s ease",
            }}
            onMouseEnter={(e) => {
              e.currentTarget.style.backgroundColor = "#f9fafb";
              e.currentTarget.style.borderColor = "#9ca3af";
            }}
            onMouseLeave={(e) => {
              e.currentTarget.style.backgroundColor = "#ffffff";
              e.currentTarget.style.borderColor = "#d1d5db";
            }}
          >
            ✚ Create Task
          </button>
        </div>
      </div>

      {/* Email content */}
      <div
        style={{
          flex: 1,
          overflowY: "auto",
          padding: "32px 24px",
          maxWidth: "900px",
          margin: "0 auto",
          width: "100%",
        }}
      >
        {/* Subject */}
        <h1
          style={{
            fontSize: "28px",
            fontWeight: "700",
            color: "#111827",
            marginBottom: "24px",
            lineHeight: "1.3",
          }}
        >
          {email.subject || "(No subject)"}
        </h1>

        {/* Email metadata */}
        <div
          style={{
            marginBottom: "32px",
            paddingBottom: "24px",
            borderBottom: "1px solid #e5e7eb",
          }}
        >
          <div style={{ marginBottom: "12px" }}>
            <span style={{ fontSize: "13px", color: "#6b7280", fontWeight: "600", textTransform: "uppercase" }}>
              From:
            </span>
            <span style={{ marginLeft: "8px", fontSize: "15px", color: "#1f2937" }}>
              {email.from_name && email.from_address ? `${email.from_name} <${email.from_address}>` : email.from_address || "Unknown"}
            </span>
          </div>

          {email.to_addresses && (
            <div style={{ marginBottom: "12px" }}>
              <span style={{ fontSize: "13px", color: "#6b7280", fontWeight: "600", textTransform: "uppercase" }}>
                To:
              </span>
              <span style={{ marginLeft: "8px", fontSize: "15px", color: "#1f2937" }}>
                {email.to_addresses}
              </span>
            </div>
          )}

          <div style={{ marginBottom: "12px" }}>
            <span style={{ fontSize: "13px", color: "#6b7280", fontWeight: "600", textTransform: "uppercase" }}>
              Date:
            </span>
            <span style={{ marginLeft: "8px", fontSize: "15px", color: "#1f2937" }}>
              {formatDate(email.received_at)}
            </span>
          </div>

          {email.folder && (
            <div style={{ marginBottom: "12px" }}>
              <span style={{ fontSize: "13px", color: "#6b7280", fontWeight: "600", textTransform: "uppercase" }}>
                Folder:
              </span>
              <span style={{ marginLeft: "8px", fontSize: "15px", color: "#1f2937" }}>
                {email.folder}
              </span>
            </div>
          )}

          <div>
            <span style={{ fontSize: "13px", color: "#6b7280", fontWeight: "600", textTransform: "uppercase" }}>
              Status:
            </span>
            <span
              style={{
                marginLeft: "8px",
                fontSize: "13px",
                padding: "2px 8px",
                borderRadius: "4px",
                backgroundColor:
                  email.status === "unprocessed"
                    ? "#fef3c7"
                    : email.status === "triaged"
                    ? "#d1fae5"
                    : "#f3f4f6",
                color:
                  email.status === "unprocessed"
                    ? "#92400e"
                    : email.status === "triaged"
                    ? "#065f46"
                    : "#6b7280",
              }}
            >
              {email.status}
            </span>
            <span
              style={{
                marginLeft: "12px",
                fontSize: "13px",
                color: "#6b7280",
                fontWeight: "600",
                textTransform: "uppercase",
              }}
            >
              Read:
            </span>
            <span
              style={{
                marginLeft: "8px",
                fontSize: "13px",
                padding: "2px 8px",
                borderRadius: "4px",
                backgroundColor: email.is_read ? "#d1fae5" : "#dbeafe",
                color: email.is_read ? "#065f46" : "#1e40af",
              }}
            >
              {email.is_read ? "Yes" : "No"}
            </span>
          </div>
        </div>

        {/* Email body */}
        <div>
          {processedHtml ? (
            <div
              className="email-content email-content-wrapper"
              dangerouslySetInnerHTML={{ __html: processedHtml }}
            />
          ) : email.body_text ? (
            <pre
              style={{
                fontFamily: "inherit",
                fontSize: "15px",
                lineHeight: "1.6",
                whiteSpace: "pre-wrap",
                wordWrap: "break-word",
                margin: 0,
                color: "#1f2937",
              }}
            >
              {email.body_text}
            </pre>
          ) : (
            <div style={{ color: "#9ca3af", fontStyle: "italic" }}>No content available</div>
          )}
        </div>
      </div>

      {/* Create Task Window */}
      {showCreateTaskWindow && (
        <CreateTaskWindow
          currentCard={null}
          setShowTaskWindow={handleCloseTaskWindow}
          currentFilter={email?.subject ? `Email: ${email.subject}` : undefined}
        />
      )}

      {/* Convert to Card Dialog */}
      {showConvertDialog && (
        <EmailConvertDialog
          isOpen={showConvertDialog}
          email={email}
          onClose={handleCloseConvertDialog}
          onConverted={handleEmailConverted}
        />
      )}
    </div>
  );
}
