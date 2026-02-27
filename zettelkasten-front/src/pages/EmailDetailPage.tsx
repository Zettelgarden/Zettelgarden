import React, { useState, useEffect, useMemo } from "react";
import { useParams, useNavigate } from "react-router-dom";
import {
  getEmail,
  updateEmailStatus,
  extractFactsFromEmail,
  saveFactsFromEmail,
  Email,
  getEmailAttachments,
  EmailAttachmentWithDownloadURL,
  saveAttachmentToVault,
  deleteEmailAttachment,
} from "../api/email";
import { setDocumentTitle } from "../utils/title";
import { CreateTaskWindow } from "../components/tasks/CreateTaskWindow";
import { EmailConvertDialog } from "../components/email/EmailConvertDialog";
import { processEmailHtml } from "../utils/emailHtml";
import { useAuth } from "../contexts/AuthContext";

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
  const { user } = useAuth();

  const [email, setEmail] = useState<Email | null>(null);
  const [loading, setLoading] = useState(true);
  const [error, setError] = useState<string | null>(null);
  const [isArchiving, setIsArchiving] = useState(false);
  const [showCreateTaskWindow, setShowCreateTaskWindow] = useState(false);
  const [showConvertDialog, setShowConvertDialog] = useState(false);

  // Fact extraction state
  const [extractedFacts, setExtractedFacts] = useState<string[]>([]);
  const [isExtractingFacts, setIsExtractingFacts] = useState(false);
  const [showFactDialog, setShowFactDialog] = useState(false);
  const [factExtractionError, setFactExtractionError] = useState<string | null>(null);

  // Attachments state
  const [attachments, setAttachments] = useState<EmailAttachmentWithDownloadURL[]>([]);
  const [attachmentsLoading, setAttachmentsLoading] = useState(false);
  const [attachmentsError, setAttachmentsError] = useState<string | null>(null);

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

  // Fetch attachments when email is loaded
  useEffect(() => {
    const fetchAttachments = async () => {
      if (!id) return;

      setAttachmentsLoading(true);
      setAttachmentsError(null);

      try {
        const data = await getEmailAttachments(parseInt(id));
        setAttachments(data.attachments);
      } catch (err: any) {
        console.error("Failed to fetch attachments:", err);
        setAttachmentsError(err.message || "Failed to load attachments");
      } finally {
        setAttachmentsLoading(false);
      }
    };

    fetchAttachments();
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

  const handleExtractFacts = async () => {
    if (!email || !id) return;

    // Check if user is PRO
    const isPro = user?.stripe_subscription_status === "active" || user?.stripe_subscription_status === "trialing";
    if (!isPro) {
      alert("Fact extraction is a PRO feature. Please upgrade your subscription to access this feature.");
      return;
    }

    setIsExtractingFacts(true);
    setFactExtractionError(null);
    try {
      const response = await extractFactsFromEmail(parseInt(id));
      setExtractedFacts(response.facts);
      if (response.facts.length > 0) {
        setShowFactDialog(true);
      } else {
        alert("No facts were extracted from this email. The email may not contain enough factual information.");
      }
    } catch (err: any) {
      console.error("Failed to extract facts:", err);
      setFactExtractionError(err.message || "Failed to extract facts");
    } finally {
      setIsExtractingFacts(false);
    }
  };

  const handleSaveFacts = async (factsToSave: string[]) => {
    if (!email || !id) return;

    try {
      const response = await saveFactsFromEmail(parseInt(id), factsToSave);
      alert(`Successfully saved ${response.saved_count} facts from this email.`);
      setShowFactDialog(false);
      setExtractedFacts([]);
    } catch (err: any) {
      console.error("Failed to save facts:", err);
      alert("Failed to save facts: " + (err.message || "Unknown error"));
    }
  };

  const handleDownloadAttachment = (attachment: EmailAttachmentWithDownloadURL) => {
    window.open(attachment.download_url, '_blank');
  };

  const handleSaveAttachmentToVault = async (attachmentId: number) => {
    try {
      const updated = await saveAttachmentToVault(attachmentId, {});
      // Update the attachment in the list
      setAttachments(attachments.map(a =>
        a.id === attachmentId
          ? { ...a, is_saved_to_vault: true, file_id: updated.file_id }
          : a
      ));
      alert("Attachment saved to file vault!");
    } catch (err: any) {
      console.error("Failed to save attachment to vault:", err);
      alert("Failed to save attachment: " + (err.message || "Unknown error"));
    }
  };

  const handleDeleteAttachment = async (attachmentId: number) => {
    if (!confirm("Are you sure you want to delete this attachment?")) {
      return;
    }

    try {
      await deleteEmailAttachment(attachmentId);
      // Remove from list
      setAttachments(attachments.filter(a => a.id !== attachmentId));
    } catch (err: any) {
      console.error("Failed to delete attachment:", err);
      alert("Failed to delete attachment: " + (err.message || "Unknown error"));
    }
  };

  const formatFileSize = (bytes?: number) => {
    if (!bytes) return "";
    if (bytes < 1024) return bytes + " B";
    if (bytes < 1024 * 1024) return (bytes / 1024).toFixed(1) + " KB";
    return (bytes / (1024 * 1024)).toFixed(1) + " MB";
  };

  const isProUser = user?.stripe_subscription_status === "active" || user?.stripe_subscription_status === "trialing";

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

          <button
            onClick={handleExtractFacts}
            disabled={isExtractingFacts}
            style={{
              padding: "8px 16px",
              fontSize: "14px",
              fontWeight: "500",
              borderRadius: "8px",
              border: isProUser ? "1px solid #d1d5db" : "1px solid #fbbf24",
              backgroundColor: isExtractingFacts ? "#f3f4f6" : isProUser ? "#ffffff" : "#fef3c7",
              color: isExtractingFacts ? "#9ca3af" : isProUser ? "#374151" : "#92400e",
              cursor: isExtractingFacts ? "not-allowed" : "pointer",
              opacity: isExtractingFacts ? 0.6 : 1,
              display: "flex",
              alignItems: "center",
              gap: "6px",
              transition: "all 0.15s ease",
            }}
            onMouseEnter={(e) => {
              if (!isExtractingFacts) {
                e.currentTarget.style.backgroundColor = isProUser ? "#f9fafb" : "#fde68a";
                e.currentTarget.style.borderColor = isProUser ? "#9ca3af" : "#f59e0b";
              }
            }}
            onMouseLeave={(e) => {
              if (!isExtractingFacts) {
                e.currentTarget.style.backgroundColor = isProUser ? "#ffffff" : "#fef3c7";
                e.currentTarget.style.borderColor = isProUser ? "#d1d5db" : "#fbbf24";
              }
            }}
            title={isProUser ? "Extract facts from email using AI" : "PRO feature: Extract facts from email using AI"}
          >
            {isExtractingFacts ? (
              "..."
            ) : isProUser ? (
              <>
                <svg width="16" height="16" viewBox="0 0 24 24" fill="none" stroke="currentColor" strokeWidth="2">
                  <path d="M2 12h20M2 12l5-5m-5 5l5 5" />
                </svg>
                Extract Facts
              </>
            ) : (
              <>
                👑 Extract Facts
              </>
            )}
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

        {/* Attachments section */}
        {attachments.length > 0 && (
          <div style={{ marginTop: "32px", paddingTop: "24px", borderTop: "1px solid #e5e7eb" }}>
            <h3 style={{ fontSize: "16px", fontWeight: "600", color: "#111827", marginBottom: "16px" }}>
              Attachments ({attachments.length})
            </h3>
            <div style={{ display: "flex", flexDirection: "column", gap: "12px" }}>
              {attachments.map((attachment) => (
                <div
                  key={attachment.id}
                  style={{
                    display: "flex",
                    alignItems: "center",
                    padding: "12px 16px",
                    border: "1px solid #e5e7eb",
                    borderRadius: "8px",
                    backgroundColor: "#ffffff",
                    transition: "all 0.15s ease",
                  }}
                  onMouseEnter={(e) => {
                    e.currentTarget.style.backgroundColor = "#f9fafb";
                    e.currentTarget.style.borderColor = "#d1d5db";
                  }}
                  onMouseLeave={(e) => {
                    e.currentTarget.style.backgroundColor = "#ffffff";
                    e.currentTarget.style.borderColor = "#e5e7eb";
                  }}
                >
                  {/* Thumbnail or icon */}
                  <div style={{ marginRight: "12px", flexShrink: 0 }}>
                    {attachment.is_image && attachment.thumbnail_url ? (
                      <img
                        src={attachment.thumbnail_url}
                        alt={attachment.filename}
                        style={{
                          width: "48px",
                          height: "48px",
                          objectFit: "cover",
                          borderRadius: "4px",
                          border: "1px solid #e5e7eb",
                        }}
                      />
                    ) : (
                      <div
                        style={{
                          width: "48px",
                          height: "48px",
                          display: "flex",
                          alignItems: "center",
                          justifyContent: "center",
                          backgroundColor: "#f3f4f6",
                          borderRadius: "4px",
                          fontSize: "20px",
                        }}
                      >
                        📎
                      </div>
                    )}
                  </div>

                  {/* File info */}
                  <div style={{ flex: 1, minWidth: 0 }}>
                    <div style={{ fontSize: "14px", fontWeight: "500", color: "#111827", marginBottom: "2px", overflow: "hidden", textOverflow: "ellipsis", whiteSpace: "nowrap" }}>
                      {attachment.filename}
                    </div>
                    <div style={{ fontSize: "12px", color: "#6b7280" }}>
                      {formatFileSize(attachment.size)}
                      {attachment.content_type && ` • ${attachment.content_type.split("/")[1]?.toUpperCase() || "FILE"}`}
                      {attachment.is_saved_to_vault && " • Saved to vault"}
                    </div>
                  </div>

                  {/* Actions */}
                  <div style={{ display: "flex", gap: "8px", flexShrink: 0 }}>
                    <button
                      onClick={() => handleDownloadAttachment(attachment)}
                      title="Download attachment"
                      style={{
                        padding: "6px 12px",
                        fontSize: "12px",
                        fontWeight: "500",
                        borderRadius: "6px",
                        border: "1px solid #d1d5db",
                        backgroundColor: "#ffffff",
                        color: "#374151",
                        cursor: "pointer",
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
                      Download
                    </button>
                    {!attachment.is_saved_to_vault && (
                      <button
                        onClick={() => handleSaveAttachmentToVault(attachment.id)}
                        title="Save to file vault"
                                                        style={{
                                                          padding: "6px 12px",
                                                          fontSize: "12px",
                                                          fontWeight: "500",
                                                          borderRadius: "6px",
                                                          border: "1px solid #d1d5db",
                                                          backgroundColor: "#ffffff",
                                                          color: "#374151",
                                                          cursor: "pointer",
                                                          transition: "all 0.15s ease",
                                                        }}
                                                        onMouseEnter={(e) => {
                                                          e.currentTarget.style.backgroundColor = "#eff6ff";
                                                          e.currentTarget.style.borderColor = "#3b82f6";
                                                        }}
                                                        onMouseLeave={(e) => {
                                                          e.currentTarget.style.backgroundColor = "#ffffff";
                                                          e.currentTarget.style.borderColor = "#d1d5db";
                                                        }}
                                                      >
                                                        Save to Vault
                                                      </button>
                                                    )}
                                                  </div>
                                                </div>
                                              ))}
                                            </div>
                                          </div>
                                        )}
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

      {/* Fact Extraction Dialog */}
      {showFactDialog && (
        <div
          style={{
            position: "fixed",
            inset: 0,
            backgroundColor: "rgba(0, 0, 0, 0.5)",
            display: "flex",
            alignItems: "center",
            justifyContent: "center",
            zIndex: 50,
          }}
          onClick={() => setShowFactDialog(false)}
        >
          <div
            style={{
              backgroundColor: "white",
              borderRadius: "12px",
              padding: "24px",
              maxWidth: "600px",
              width: "90%",
              maxHeight: "80vh",
              overflow: "auto",
            }}
            onClick={(e) => e.stopPropagation()}
          >
            <h2 style={{ fontSize: "20px", fontWeight: "600", marginBottom: "16px" }}>
              Extracted Facts from Email
            </h2>
            <p style={{ color: "#6b7280", marginBottom: "16px" }}>
              Review the AI-extracted facts below. Uncheck any facts you don't want to save.
            </p>

            {factExtractionError && (
              <div style={{
                padding: "12px",
                backgroundColor: "#fee2e2",
                border: "1px solid #ef4444",
                borderRadius: "8px",
                color: "#b91c1c",
                marginBottom: "16px"
              }}>
                {factExtractionError}
              </div>
            )}

            <div style={{ marginBottom: "16px" }}>
              {extractedFacts.length === 0 ? (
                <p style={{ color: "#6b7280", fontStyle: "italic" }}>
                  No facts were extracted from this email.
                </p>
              ) : (
                extractedFacts.map((fact, index) => (
                  <div
                    key={index}
                    id={`fact-item-${index}`}
                    style={{
                      padding: "12px",
                      border: "1px solid #e5e7eb",
                      borderRadius: "8px",
                      marginBottom: "8px",
                      backgroundColor: "#f9fafb",
                    }}
                  >
                    <label style={{ display: "flex", alignItems: "flex-start", gap: "8px", cursor: "pointer" }}>
                      <input
                        type="checkbox"
                        defaultChecked={true}
                        style={{ marginTop: "4px" }}
                        data-fact-index={index}
                      />
                      <span style={{ fontSize: "14px", color: "#1f2937" }}>
                        {fact}
                      </span>
                    </label>
                  </div>
                ))
              )}
            </div>

            <div style={{ display: "flex", justifyContent: "flex-end", gap: "12px", paddingTop: "16px", borderTop: "1px solid #e5e7eb" }}>
              <button
                onClick={() => {
                  setShowFactDialog(false);
                  setExtractedFacts([]);
                }}
                style={{
                  padding: "8px 16px",
                  fontSize: "14px",
                  fontWeight: "500",
                  borderRadius: "8px",
                  border: "1px solid #d1d5db",
                  backgroundColor: "#ffffff",
                  color: "#374151",
                  cursor: "pointer",
                }}
              >
                Cancel
              </button>
              <button
                onClick={() => {
                  // Get all checked facts
                  const checkboxes = document.querySelectorAll('input[type="checkbox"][data-fact-index]');
                  const checkedFacts: string[] = [];
                  checkboxes.forEach((cb) => {
                    if (cb instanceof HTMLInputElement && cb.checked) {
                      const factSpan = cb.parentElement?.querySelector("span");
                      const factText = factSpan?.textContent || "";
                      if (factText.trim()) {
                        checkedFacts.push(factText.trim());
                      }
                    }
                  });

                  if (checkedFacts.length > 0) {
                    handleSaveFacts(checkedFacts);
                  } else {
                    alert("Please select at least one fact to save.");
                  }
                }}
                style={{
                  padding: "8px 16px",
                  fontSize: "14px",
                  fontWeight: "500",
                  borderRadius: "8px",
                  border: "none",
                  backgroundColor: "#3b82f6",
                  color: "white",
                  cursor: "pointer",
                }}
                onMouseEnter={(e) => {
                  e.currentTarget.style.backgroundColor = "#2563eb";
                }}
                onMouseLeave={(e) => {
                  e.currentTarget.style.backgroundColor = "#3b82f6";
                }}
              >
                Save Selected Facts ({extractedFacts.length})
              </button>
            </div>
          </div>
        </div>
      )}
    </div>
  );
}
