import React, { useState, useEffect } from "react";
import { useParams, useNavigate } from "react-router-dom";
import { getEmail, updateEmailStatus, Email } from "../api/email";
import { setDocumentTitle } from "../utils/title";
import { CreateTaskWindow } from "../components/tasks/CreateTaskWindow";
import { EmailConvertDialog } from "../components/email/EmailConvertDialog";

/**
 * Email Detail Page
 *
 * Displays a single email with its full content.
 * All links in email bodies open in a new tab for security.
 */
export function EmailDetailPage() {
  const { id } = useParams<{ id: string }>();
  const navigate = useNavigate();

  const [email, setEmail] = useState<Email | null>(null);
  const [loading, setLoading] = useState(true);
  const [error, setError] = useState<string | null>(null);
  const [isArchiving, setIsArchiving] = useState(false);
  const [showCreateTaskWindow, setShowCreateTaskWindow] = useState(false);
  const [showConvertDialog, setShowConvertDialog] = useState(false);
  const [processedHtml, setProcessedHtml] = useState<string>("");

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

  // Process HTML to add target="_blank" and rel="noopener noreferrer" to all links
  useEffect(() => {
    if (!email?.body_html) {
      setProcessedHtml("");
      return;
    }

    const parser = new DOMParser();
    const doc = parser.parseFromString(email.body_html, "text/html");

    // Add target="_blank" and rel="noopener noreferrer" to all anchor tags
    const links = doc.querySelectorAll("a");
    links.forEach((link) => {
      link.setAttribute("target", "_blank");
      link.setAttribute("rel", "noopener noreferrer");
    });

    setProcessedHtml(doc.body.innerHTML);
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
    setShowConvertDialog(true);
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
        <div
          style={{
            fontSize: "15px",
            lineHeight: "1.6",
            color: "#1f2937",
          }}
        >
          {processedHtml ? (
            <div
              dangerouslySetInnerHTML={{ __html: processedHtml }}
              style={{
                overflowWrap: "break-word",
              }}
            />
          ) : email.body_text ? (
            <pre
              style={{
                fontFamily: "inherit",
                fontSize: "inherit",
                lineHeight: "inherit",
                whiteSpace: "pre-wrap",
                wordWrap: "break-word",
                margin: 0,
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
