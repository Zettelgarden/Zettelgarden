import React, { useState, useEffect, useMemo } from "react";
import { useParams, useNavigate } from "react-router-dom";
import {
  getEmail,
  updateEmailStatus,
  Email,
  getEmailAttachments,
  EmailAttachmentWithDownloadURL,
  deleteEmailAttachment,
} from "../api/email";
import { setDocumentTitle } from "../utils/title";
import { CreateTaskWindow } from "../components/tasks/CreateTaskWindow";
import { EmailConvertDialog } from "../components/email/EmailConvertDialog";
import { processEmailHtml } from "../utils/emailHtml";
import { useAuth } from "../contexts/AuthContext";
import styles from "../components/email/EmailContent.module.css";

/**
 * Email Detail Page
 *
 * Displays a single email with its full content.
 * HTML content is sanitized using DOMPurify for security.
 * All links in email bodies open in a new tab for security.
 */
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

  // Attachments state
  const [attachments, setAttachments] = useState<EmailAttachmentWithDownloadURL[]>([]);
  const [attachmentsLoading, setAttachmentsLoading] = useState(false);
  const [attachmentsError, setAttachmentsError] = useState<string | null>(null);

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

  const handleDownloadAttachment = (attachment: EmailAttachmentWithDownloadURL) => {
    window.open(attachment.download_url, '_blank');
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

  if (loading) {
    return (
      <div className="px-12 text-center">
        <div className="text-base text-gray-500">Loading email...</div>
      </div>
    );
  }

  if (error || !email) {
    return (
      <div className="px-12 text-center">
        <div className="text-lg text-red-600 mb-4">
          {error || "Email not found"}
        </div>
        <button
          onClick={handleBack}
          className="px-5 py-2.5 bg-blue-600 text-white border-none rounded cursor-pointer hover:bg-blue-700"
        >
          Back to Inbox
        </button>
      </div>
    );
  }

  return (
    <div className="flex flex-col h-screen bg-white">
      {/* Header */}
      <div className="border-b border-gray-200 bg-white px-6 py-4">
        <div className="flex items-center justify-between w-full">
          <button
            onClick={handleBack}
            className="px-4 py-2 text-sm font-medium rounded-lg border border-gray-300 bg-white text-gray-700 cursor-pointer flex items-center gap-1.5 transition-all duration-150 hover:bg-gray-50 hover:border-gray-400"
          >
            ← Back to Inbox
          </button>

          <button
            onClick={email.status === "archived" ? handleUnarchive : handleArchive}
            disabled={isArchiving}
            className={`px-4 py-2 text-sm font-medium rounded-lg border flex items-center gap-1.5 transition-all duration-150 ${
              isArchiving
                ? "bg-gray-100 text-gray-400 cursor-not-allowed opacity-60 border-gray-300"
                : email.status === "archived"
                  ? "bg-yellow-50 text-yellow-800 border-yellow-200 hover:bg-yellow-100 hover:border-yellow-300 cursor-pointer"
                  : "bg-white text-gray-700 border-gray-300 hover:bg-gray-50 hover:border-gray-400 cursor-pointer"
            }`}
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
            className={`px-4 py-2 text-sm font-medium rounded-lg border flex items-center gap-1.5 transition-all duration-150 cursor-pointer ${
              email.card_id
                ? "bg-green-100 text-green-800 border-green-200 hover:bg-green-200 hover:border-green-300"
                : "bg-white text-gray-700 border-gray-300 hover:bg-gray-50 hover:border-gray-400"
            }`}
          >
            <svg width="16" height="16" viewBox="0 0 24 24" fill="none" stroke="currentColor" strokeWidth="2">
              <rect x="3" y="3" width="18" height="18" rx="2" ry="2" />
              <line x1="9" y1="3" x2="9" y2="21" />
            </svg>
            {email.card_id ? "View Card" : "Convert to Card"}
          </button>

          <button
            onClick={handleCreateTaskFromEmail}
            className="px-4 py-2 text-sm font-medium rounded-lg border border-gray-300 bg-white text-gray-700 cursor-pointer flex items-center gap-1.5 transition-all duration-150 hover:bg-gray-50 hover:border-gray-400"
          >
            ✚ Create Task
          </button>
        </div>
      </div>

      {/* Email content */}
      <div className="flex-1 overflow-y-auto px-6 py-8 max-w-4xl mx-auto w-full">
        {/* Subject */}
        <h1 className="text-2xl font-bold text-gray-900 mb-6 leading-tight">
          {email.subject || "(No subject)"}
        </h1>

        {/* Email metadata */}
        <div className="mb-8 pb-6 border-b border-gray-200">
          <div className="mb-3">
            <span className="text-xs font-semibold text-gray-500 uppercase">
              From:
            </span>
            <span className="ml-2 text-base text-gray-800">
              {email.from_name && email.from_address ? `${email.from_name} <${email.from_address}>` : email.from_address || "Unknown"}
            </span>
          </div>

          {email.to_addresses && (
            <div className="mb-3">
              <span className="text-xs font-semibold text-gray-500 uppercase">
                To:
              </span>
              <span className="ml-2 text-base text-gray-800">
                {email.to_addresses}
              </span>
            </div>
          )}

          <div className="mb-3">
            <span className="text-xs font-semibold text-gray-500 uppercase">
              Date:
            </span>
            <span className="ml-2 text-base text-gray-800">
              {formatDate(email.received_at || email.created_at)}
            </span>
          </div>

          {email.folder && (
            <div className="mb-3">
              <span className="text-xs font-semibold text-gray-500 uppercase">
                Folder:
              </span>
              <span className="ml-2 text-base text-gray-800">
                {email.folder}
              </span>
            </div>
          )}

          <div>
            <span className="text-xs font-semibold text-gray-500 uppercase">
              Status:
            </span>
            <span
              className={`ml-2 text-xs px-2 py-0.5 rounded ${
                email.status === "unprocessed"
                  ? "bg-yellow-100 text-yellow-800"
                  : email.status === "triaged"
                    ? "bg-green-100 text-green-800"
                    : "bg-gray-100 text-gray-700"
              }`}
            >
              {email.status}
            </span>
            <span
              className="ml-3 text-xs font-semibold text-gray-500 uppercase"
            >
              Read:
            </span>
            <span
              className={`ml-2 text-xs px-2 py-0.5 rounded ${
                email.is_read ? "bg-green-100 text-green-800" : "bg-blue-100 text-blue-800"
              }`}
            >
              {email.is_read ? "Yes" : "No"}
            </span>
          </div>
        </div>

        {/* Email body */}
        <div>
          {processedHtml ? (
            <div
              className={styles.emailContent}
              dangerouslySetInnerHTML={{ __html: processedHtml }}
            />
          ) : email.body_text ? (
            <pre className="font-inherit text-base leading-relaxed whitespace-pre-wrap break-word m-0 text-gray-800">
              {email.body_text}
            </pre>
          ) : (
            <div className="text-gray-400 italic">No content available</div>
          )}
        </div>

        {/* Attachments section */}
        {attachments.length > 0 && (
          <div className="mt-8 pt-6 border-t border-gray-200">
            <h3 className="text-base font-semibold text-gray-900 mb-4">
              Attachments ({attachments.length})
            </h3>
            <div className="flex flex-col gap-3">
              {attachments.map((attachment) => (
                <div
                  key={attachment.id}
                  className="flex items-center px-4 py-3 border rounded-lg bg-white transition-all duration-150 hover:bg-gray-50 hover:border-gray-300"
                >
                  {/* Thumbnail or icon */}
                  <div className="mr-3 flex-shrink-0">
                    {attachment.is_image && attachment.thumbnail_url ? (
                      <img
                        src={attachment.thumbnail_url}
                        alt={attachment.filename}
                        className="w-12 h-12 object-cover rounded border border-gray-200"
                      />
                    ) : (
                      <div className="w-12 h-12 flex items-center justify-center bg-gray-100 rounded text-xl">
                        📎
                      </div>
                    )}
                  </div>

                  {/* File info */}
                  <div className="flex-1 min-w-0">
                    <div className="text-sm font-medium text-gray-900 mb-0.5 truncate">
                      {attachment.filename}
                    </div>
                    <div className="text-xs text-gray-500">
                      {formatFileSize(attachment.size)}
                      {attachment.content_type && ` • ${attachment.content_type.split("/")[1]?.toUpperCase() || "FILE"}`}
                    </div>
                  </div>

                  {/* Actions */}
                  <div className="flex gap-2 flex-shrink-0">
                    <button
                      onClick={() => handleDownloadAttachment(attachment)}
                      title="Download attachment"
                      className="px-3 py-1.5 text-xs font-medium rounded-md border border-gray-300 bg-white text-gray-700 cursor-pointer transition-all duration-150 hover:bg-gray-50 hover:border-gray-400"
                    >
                      Download
                    </button>
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
    </div>
  );
}
