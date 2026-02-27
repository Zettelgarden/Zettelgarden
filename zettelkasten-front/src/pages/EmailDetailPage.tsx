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

  // Fact extraction state
  const [extractedFacts, setExtractedFacts] = useState<string[]>([]);
  const [isExtractingFacts, setIsExtractingFacts] = useState(false);
  const [showFactDialog, setShowFactDialog] = useState(false);
  const [factExtractionError, setFactExtractionError] = useState<string | null>(null);

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

          <button
            onClick={handleExtractFacts}
            disabled={isExtractingFacts}
            title={isProUser ? "Extract facts from email using AI" : "PRO feature: Extract facts from email using AI"}
            className={`px-4 py-2 text-sm font-medium rounded-lg border flex items-center gap-1.5 transition-all duration-150 ${
              isExtractingFacts
                ? "bg-gray-100 text-gray-400 cursor-not-allowed opacity-60 border-gray-300"
                : isProUser
                  ? "bg-white text-gray-700 border-gray-300 hover:bg-gray-50 hover:border-gray-400 cursor-pointer"
                  : "bg-yellow-50 text-yellow-800 border-yellow-400 hover:bg-yellow-100 hover:border-yellow-500 cursor-pointer"
            }`}
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
      <div className="flex-1 overflow-y-auto px-6 py-8 max-w-2xl mx-auto w-full">
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
              {formatDate(email.received_at)}
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
                      {attachment.is_saved_to_vault && " • Saved to vault"}
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
                    {!attachment.is_saved_to_vault && (
                      <button
                        onClick={() => handleSaveAttachmentToVault(attachment.id)}
                        title="Save to file vault"
                        className="px-3 py-1.5 text-xs font-medium rounded-md border border-gray-300 bg-white text-gray-700 cursor-pointer transition-all duration-150 hover:bg-blue-50 hover:border-blue-500"
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
          className="fixed inset-0 bg-black bg-opacity-50 flex items-center justify-center z-50"
          onClick={() => setShowFactDialog(false)}
        >
          <div
            className="bg-white rounded-xl p-6 max-w-lg w-[90%] max-h-[80vh] overflow-auto"
            onClick={(e) => e.stopPropagation()}
          >
            <h2 className="text-xl font-semibold mb-4">
              Extracted Facts from Email
            </h2>
            <p className="text-gray-500 mb-4">
              Review the AI-extracted facts below. Uncheck any facts you don't want to save.
            </p>

            {factExtractionError && (
              <div className="p-3 bg-red-50 border border-red-500 rounded-lg text-red-700 mb-4">
                {factExtractionError}
              </div>
            )}

            <div className="mb-4">
              {extractedFacts.length === 0 ? (
                <p className="text-gray-500 italic">
                  No facts were extracted from this email.
                </p>
              ) : (
                extractedFacts.map((fact, index) => (
                  <div
                    key={index}
                    id={`fact-item-${index}`}
                    className="p-3 border rounded-lg mb-2 bg-gray-50"
                  >
                    <label className="flex items-start gap-2 cursor-pointer">
                      <input
                        type="checkbox"
                        defaultChecked={true}
                        className="mt-1"
                        data-fact-index={index}
                      />
                      <span className="text-sm text-gray-800">
                        {fact}
                      </span>
                    </label>
                  </div>
                ))
              )}
            </div>

            <div className="flex justify-end gap-3 pt-4 border-t border-gray-200">
              <button
                onClick={() => {
                  setShowFactDialog(false);
                  setExtractedFacts([]);
                }}
                className="px-4 py-2 text-sm font-medium rounded-lg border border-gray-300 bg-white text-gray-700 cursor-pointer hover:bg-gray-50"
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
                className="px-4 py-2 text-sm font-medium rounded-lg border-none bg-blue-600 text-white cursor-pointer hover:bg-blue-700"
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
