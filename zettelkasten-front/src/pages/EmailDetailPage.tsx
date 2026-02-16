import React, { useState, useEffect } from "react";
import { useParams, useNavigate } from "react-router-dom";
import { getEmail, Email } from "../api/email";
import { setDocumentTitle } from "../utils/title";

/**
 * Email Detail Page
 *
 * Displays a single email with its full content.
 */
export function EmailDetailPage() {
  const { id } = useParams<{ id: string }>();
  const navigate = useNavigate();

  const [email, setEmail] = useState<Email | null>(null);
  const [loading, setLoading] = useState(true);
  const [error, setError] = useState<string | null>(null);

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

  const handleBack = () => {
    navigate("/app/emails");
  };

  const formatDate = (dateString?: string) => {
    if (!dateString) return "";
    return new Date(dateString).toLocaleString();
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
          {email.body_html ? (
            <div
              dangerouslySetInnerHTML={{ __html: email.body_html }}
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
    </div>
  );
}
