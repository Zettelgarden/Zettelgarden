# EmailDetailPage Tailwind Refactor Implementation Plan

> **For Claude:** REQUIRED SUB-SKILL: Use superpowers:executing-plans to implement this plan task-by-task.

**Goal:** Refactor EmailDetailPage.tsx from inline styles to Tailwind utility classes, matching existing codebase patterns.

**Architecture:** Convert all inline `style={{}}` props to Tailwind `className` props. Extract email body CSS to a minimal scoped CSS module. Follow patterns from EmailConvertDialog and ViewPage.

**Tech Stack:** React, TypeScript, Tailwind CSS, CSS Modules

---

## Task 1: Create EmailContent CSS Module

**Files:**
- Create: `zettelkasten-front/src/components/email/EmailContent.module.css`

**Step 1: Create the CSS module file**

Create a minimal CSS module for email content styling. This preserves the essential email HTML sanitization while keeping it scoped.

```css
/* EmailContent.module.css - Minimal styles for email body content */
.emailContentWrapper {
  @apply font-sans text-base leading-relaxed text-gray-800 break-words;
}

/* Email body content styles - sanitized HTML from external emails */
.emailContent {
  font-family: -apple-system, BlinkMacSystemFont, 'Segoe UI', Roboto, 'Helvetica Neue', Arial, sans-serif;
  font-size: 15px;
  line-height: 1.6;
  color: #1f2937;
  word-wrap: break-word;
  overflow-wrap: break-word;
}

/* Headings */
.emailContent h1,
.emailContent h2,
.emailContent h3,
.emailContent h4,
.emailContent h5,
.emailContent h6 {
  margin-top: 1.5em;
  margin-bottom: 0.5em;
  font-weight: 600;
  line-height: 1.3;
}

.emailContent h1 { font-size: 1.75em; }
.emailContent h2 { font-size: 1.5em; }
.emailContent h3 { font-size: 1.25em; }
.emailContent h4 { font-size: 1.1em; }
.emailContent h5 { font-size: 1em; }
.emailContent h6 { font-size: 0.9em; }

/* Paragraphs and lists */
.emailContent p {
  margin-top: 0;
  margin-bottom: 1em;
}

.emailContent ul,
.emailContent ol {
  margin-top: 0;
  margin-bottom: 1em;
  padding-left: 2em;
}

.emailContent li {
  margin-bottom: 0.25em;
}

/* Links - open in new tab for security */
.emailContent a {
  color: #2563eb;
  text-decoration: underline;
}

.emailContent a:hover {
  color: #1d4ed8;
}

/* Tables */
.emailContent table {
  max-width: 100%;
  border-collapse: collapse;
  margin: 1em 0;
  font-size: 14px;
  table-layout: auto;
}

.emailContent table[width] {
  width: auto !important;
  max-width: 100%;
}

.emailContent th,
.emailContent td {
  padding: 0.5em;
  border: 1px solid #d1d5db;
  vertical-align: top;
  text-align: left;
  word-wrap: break-word;
  overflow-wrap: break-word;
  min-width: 50px;
}

.emailContent th {
  background-color: #f3f4f6;
  font-weight: 600;
}

.emailContent tr:nth-child(even) {
  background-color: #f9fafb;
}

/* Images - responsive */
.emailContent img {
  max-width: 100%;
  height: auto;
  display: block;
  margin: 0.5em 0;
}

.emailContent img[width] {
  width: auto !important;
  max-width: 100%;
}

.emailContent img[height] {
  height: auto !important;
}

/* Blockquotes */
.emailContent blockquote {
  margin: 1em 0;
  padding: 0.5em 1em;
  border-left: 4px solid #d1d5db;
  background-color: #f9fafb;
  color: #6b7280;
}

/* Code blocks */
.emailContent pre {
  background-color: #1f2937;
  color: #f3f4f6;
  padding: 1em;
  border-radius: 0.375em;
  overflow-x: auto;
  margin: 1em 0;
}

.emailContent code {
  font-family: 'Courier New', Courier, monospace;
  font-size: 0.9em;
  background-color: #f3f4f6;
  padding: 0.125em 0.25em;
  border-radius: 0.125em;
}

.emailContent pre code {
  background-color: transparent;
  padding: 0;
}

/* Horizontal rule */
.emailContent hr {
  border: none;
  border-top: 1px solid #e5e7eb;
  margin: 1.5em 0;
}

/* Text formatting */
.emailContent strong,
.emailContent b {
  font-weight: 600;
}

.emailContent em,
.emailContent i {
  font-style: italic;
}

/* Remove margins from first/last elements */
.emailContent > *:first-child {
  margin-top: 0;
}

.emailContent > *:last-child {
  margin-bottom: 0;
}

/* Fix for email clients that add weird styles */
.emailContent div[style*="font-size"] {
  font-size: inherit !important;
}
```

**Step 2: Commit the CSS module**

```bash
git add zettelkasten-front/src/components/email/EmailContent.module.css
git commit -m "feat: add EmailContent CSS module for email body styling"
```

---

## Task 2: Update Imports and Remove Style Injection

**Files:**
- Modify: `zettelkasten-front/src/pages/EmailDetailPage.tsx:1-239`

**Step 1: Add CSS module import**

Add the import for the new CSS module at the top of the file (after existing imports, around line 18):

```tsx
import { useAuth } from "../contexts/AuthContext";
import styles from "../components/email/EmailContent.module.css";
```

**Step 2: Remove emailStyles constant**

Delete the entire `emailStyles` constant (lines 29-195). Find and remove:

```tsx
// DELETE these lines (29-195):
const emailStyles = `
  /* Base styles */
  .email-content {
    ...
  }
`;
```

**Step 3: Remove style injection useEffect**

Remove the useEffect that injects styles (lines 219-239). Find and remove:

```tsx
// DELETE these lines (219-239):
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
    const remainingElements = document.querySelectorAll('.email-content-wrapper');
    if (remainingElements.length === 0) {
      styleElement.remove();
    }
  };
}, []);
```

**Step 4: Verify imports look correct**

The imports section should now look like this (approximately lines 1-19):

```tsx
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
```

**Step 5: Commit**

```bash
git add zettelkasten-front/src/pages/EmailDetailPage.tsx
git commit -m "refactor: remove inline emailStyles constant and style injection, add CSS module import"
```

---

## Task 3: Convert Main Container and Header to Tailwind

**Files:**
- Modify: `zettelkasten-front/src/pages/EmailDetailPage.tsx:479-673`

**Step 1: Convert main container div**

Find the main container div (around line 479-485) and convert:

```tsx
// BEFORE:
<div style={{ display: "flex", flexDirection: "column", height: "100vh", backgroundColor: "#ffffff" }}>

// AFTER:
<div className="flex flex-col h-screen bg-white">
```

**Step 2: Convert header div**

Find the header div (around line 482-488) and convert:

```tsx
// BEFORE:
<div
  style={{
    borderBottom: "1px solid #e5e7eb",
    backgroundColor: "#ffffff",
    padding: "16px 24px",
  }}
>

// AFTER:
<div className="border-b border-gray-200 bg-white px-6 py-4">
```

**Step 3: Convert header button container**

Find the header button container div (around line 489-496) and convert:

```tsx
// BEFORE:
<div
  style={{
    display: "flex",
    alignItems: "center",
    justifyContent: "space-between",
    width: "100%",
  }}
>

// AFTER:
<div className="flex items-center justify-between w-full">
```

**Step 4: Convert Back button**

Find the Back button (around line 497-523) and convert:

```tsx
// BEFORE:
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

// AFTER:
<button
  onClick={handleBack}
  className="px-4 py-2 text-sm font-medium rounded-lg border border-gray-300 bg-white text-gray-700 cursor-pointer flex items-center gap-1.5 transition-all duration-150 hover:bg-gray-50 hover:border-gray-400"
>
  ← Back to Inbox
</button>
```

**Step 5: Convert Archive/Unarchive button**

Find the Archive button (around line 525-563) and convert:

```tsx
// BEFORE:
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

// AFTER:
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
```

**Step 6: Convert Convert to Card button**

Find the Convert button (around line 565-595) and convert:

```tsx
// BEFORE:
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

// AFTER:
<button
  onClick={handleConvertEmail}
  className={`px-4 py-2 text-sm font-medium rounded-lg border flex items-center gap-1.5 transition-all duration-150 cursor-pointer ${
    email.card_id
      ? "bg-green-100 text-green-800 border-green-200 hover:bg-green-200 hover:border-green-300"
      : "bg-white text-gray-700 border-gray-300 hover:bg-gray-50 hover:border-gray-400"
  }`}
>
```

**Step 7: Convert Create Task button**

Find the Create Task button (around line 597-623) and convert:

```tsx
// BEFORE:
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

// AFTER:
<button
  onClick={handleCreateTaskFromEmail}
  className="px-4 py-2 text-sm font-medium rounded-lg border border-gray-300 bg-white text-gray-700 cursor-pointer flex items-center gap-1.5 transition-all duration-150 hover:bg-gray-50 hover:border-gray-400"
>
```

**Step 8: Convert Extract Facts button**

Find the Extract Facts button (around line 625-671) and convert:

```tsx
// BEFORE:
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

// AFTER:
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
```

**Step 9: Commit**

```bash
git add zettelkasten-front/src/pages/EmailDetailPage.tsx
git commit -m "refactor: convert header section to Tailwind classes"
```

---

## Task 4: Convert Email Content Section to Tailwind

**Files:**
- Modify: `zettelkasten-front/src/pages/EmailDetailPage.tsx:675-823`

**Step 1: Convert email content wrapper div**

Find the email content wrapper div (around line 675-685) and convert:

```tsx
// BEFORE:
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

// AFTER:
<div className="flex-1 overflow-y-auto px-6 py-8 max-w-2xl mx-auto w-full">
```

**Step 2: Convert subject heading**

Find the subject heading (around line 686-697) and convert:

```tsx
// BEFORE:
<h1
  style={{
    fontSize: "28px",
    fontWeight: "700",
    color: "#111827",
    marginBottom: "24px",
    lineHeight: "1.3",
  }}
>

// AFTER:
<h1 className="text-2xl font-bold text-gray-900 mb-6 leading-tight">
```

**Step 3: Convert email metadata container**

Find the email metadata container div (around line 699-705) and convert:

```tsx
// BEFORE:
<div
  style={{
    marginBottom: "32px",
    paddingBottom: "24px",
    borderBottom: "1px solid #e5e7eb",
  }}
>

// AFTER:
<div className="mb-8 pb-6 border-b border-gray-200">
```

**Step 4: Convert metadata row divs**

Find all the metadata row divs (around lines 707-745) and convert. Each row follows the same pattern:

```tsx
// BEFORE (for each metadata row):
<div style={{ marginBottom: "12px" }}>
  <span style={{ fontSize: "13px", color: "#6b7280", fontWeight: "600", textTransform: "uppercase" }}>
    From:
  </span>
  <span style={{ marginLeft: "8px", fontSize: "15px", color: "#1f2937" }}>
    ...
  </span>
</div>

// AFTER:
<div className="mb-3">
  <span className="text-xs font-semibold text-gray-500 uppercase">
    From:
  </span>
  <span className="ml-2 text-base text-gray-800">
    ...
  </span>
</div>
```

**Step 5: Convert status and read badges**

Find the status and read badges section (around lines 747-796) and convert:

```tsx
// BEFORE:
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
          ? "##065f46"
          : "#6b7280",
  }}
>

// AFTER:
<span
  className={`ml-2 text-xs px-2 py-0.5 rounded ${
    email.status === "unprocessed"
      ? "bg-yellow-100 text-yellow-800"
      : email.status === "triaged"
        ? "bg-green-100 text-green-800"
        : "bg-gray-100 text-gray-700"
  }`}
>
```

Convert the read badge similarly:

```tsx
// BEFORE:
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

// AFTER:
<span
  className={`ml-2 text-xs px-2 py-0.5 rounded ${
    email.is_read ? "bg-green-100 text-green-800" : "bg-blue-100 text-blue-800"
  }`}
>
```

**Step 6: Convert email body div**

Find the email body div (around line 799-822) and convert to use the CSS module:

```tsx
// BEFORE:
<div>
  {processedHtml ? (
    <div
      dangerouslySetInnerHTML={{ __html: processedHtml }}
    />
  ) : ...

// AFTER:
<div>
  {processedHtml ? (
    <div
      className={styles.emailContent}
      dangerouslySetInnerHTML={{ __html: processedHtml }}
    />
  ) : ...
```

**Step 7: Commit**

```bash
git add zettelkasten-front/src/pages/EmailDetailPage.tsx
git commit -m "refactor: convert email content section to Tailwind classes"
```

---

## Task 5: Convert Attachments Section to Tailwind

**Files:**
- Modify: `zettelkasten-front/src/pages/EmailDetailPage.tsx:824-955`

**Step 1: Convert attachments section wrapper**

Find the attachments section wrapper (around line 824-826) and convert:

```tsx
// BEFORE:
{attachments.length > 0 && (
  <div style={{ marginTop: "32px", paddingTop: "24px", borderTop: "1px solid #e5e7eb" }}>

// AFTER:
{attachments.length > 0 && (
  <div className="mt-8 pt-6 border-t border-gray-200">
```

**Step 2: Convert attachments heading**

Find the attachments heading (around line 827-829) and convert:

```tsx
// BEFORE:
<h3 style={{ fontSize: "16px", fontWeight: "600", color: "#111827", marginBottom: "16px" }}>

// AFTER:
<h3 className="text-base font-semibold text-gray-900 mb-4">
```

**Step 3: Convert attachments list container**

Find the attachments list container div (around line 830) and convert:

```tsx
// BEFORE:
<div style={{ display: "flex", flexDirection: "column", gap: "12px" }}>

// AFTER:
<div className="flex flex-col gap-3">
```

**Step 4: Convert individual attachment card**

Find the attachment card div (around line 831-851) and convert:

```tsx
// BEFORE:
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

// AFTER:
<div
  key={attachment.id}
  className="flex items-center px-4 py-3 border rounded-lg bg-white transition-all duration-150 hover:bg-gray-50 hover:border-gray-300"
>
```

**Step 5: Convert thumbnail/icon container**

Find the thumbnail container div (around line 852-882) and convert:

```tsx
// BEFORE:
<div style={{ marginRight: "12px", flexShrink: 0 }}>

// AFTER:
<div className="mr-3 flex-shrink-0">
```

Convert the thumbnail image:

```tsx
// BEFORE:
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

// AFTER:
<img
  src={attachment.thumbnail_url}
  alt={attachment.filename}
  className="w-12 h-12 object-cover rounded border border-gray-200"
/>
```

Convert the icon placeholder div:

```tsx
// BEFORE:
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

// AFTER:
<div className="w-12 h-12 flex items-center justify-center bg-gray-100 rounded text-xl">
```

**Step 6: Convert file info container**

Find the file info container div (around line 884-894) and convert:

```tsx
// BEFORE:
<div style={{ flex: 1, minWidth: 0 }}>

// AFTER:
<div className="flex-1 min-w-0">
```

Convert the filename div:

```tsx
// BEFORE:
<div style={{ fontSize: "14px", fontWeight: "500", color: "#111827", marginBottom: "2px", overflow: "hidden", textOverflow: "ellipsis", whiteSpace: "nowrap" }}>

// AFTER:
<div className="text-sm font-medium text-gray-900 mb-0.5 truncate">
```

Convert the file metadata div:

```tsx
// BEFORE:
<div style={{ fontSize: "12px", color: "#6b7280" }}>

// AFTER:
<div className="text-xs text-gray-500">
```

**Step 7: Convert actions container**

Find the actions container div (around line 896-951) and convert:

```tsx
// BEFORE:
<div style={{ display: "flex", gap: "8px", flexShrink: 0 }}>

// AFTER:
<div className="flex gap-2 flex-shrink-0">
```

Convert the Download button:

```tsx
// BEFORE:
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

// AFTER:
<button
  onClick={() => handleDownloadAttachment(attachment)}
  title="Download attachment"
  className="px-3 py-1.5 text-xs font-medium rounded-md border border-gray-300 bg-white text-gray-700 cursor-pointer transition-all duration-150 hover:bg-gray-50 hover:border-gray-400"
>
```

Convert the Save to Vault button similarly:

```tsx
// BEFORE:
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

// AFTER:
<button
  onClick={() => handleSaveAttachmentToVault(attachment.id)}
  title="Save to file vault"
  className="px-3 py-1.5 text-xs font-medium rounded-md border border-gray-300 bg-white text-gray-700 cursor-pointer transition-all duration-150 hover:bg-blue-50 hover:border-blue-500"
>
```

**Step 8: Commit**

```bash
git add zettelkasten-front/src/pages/EmailDetailPage.tsx
git commit -m "refactor: convert attachments section to Tailwind classes"
```

---

## Task 6: Convert Loading and Error States to Tailwind

**Files:**
- Modify: `zettelkasten-front/src/pages/EmailDetailPage.tsx:448-477`

**Step 1: Convert loading state**

Find the loading state div (around line 448-454) and convert:

```tsx
// BEFORE:
if (loading) {
  return (
    <div style={{ padding: "48px", textAlign: "center" }}>
      <div style={{ fontSize: "16px", color: "#6b7280" }}>Loading email...</div>
    </div>
  );
}

// AFTER:
if (loading) {
  return (
    <div className="px-12 text-center">
      <div className="text-base text-gray-500">Loading email...</div>
    </div>
  );
}
```

**Step 2: Convert error state**

Find the error state div (around line 456-477) and convert:

```tsx
// BEFORE:
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

// AFTER:
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
```

**Step 3: Commit**

```bash
git add zettelkasten-front/src/pages/EmailDetailPage.tsx
git commit -m "refactor: convert loading and error states to Tailwind classes"
```

---

## Task 7: Convert Fact Extraction Dialog to Tailwind

**Files:**
- Modify: `zettelkasten-front/src/pages/EmailDetailPage.tsx:977-1119`

**Step 1: Convert dialog overlay**

Find the fact extraction dialog overlay div (around line 978-989) and convert:

```tsx
// BEFORE:
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

// AFTER:
{showFactDialog && (
  <div
    className="fixed inset-0 bg-black bg-opacity-50 flex items-center justify-center z-50"
    onClick={() => setShowFactDialog(false)}
  >
```

**Step 2: Convert dialog panel**

Find the dialog panel div (around line 990-1002) and convert:

```tsx
// BEFORE:
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

// AFTER:
<div
  className="bg-white rounded-xl p-6 max-w-lg w-[90%] max-h-[80vh] overflow-auto"
  onClick={(e) => e.stopPropagation()}
>
```

**Step 3: Convert dialog heading**

Find the dialog heading (around line 1003-1005) and convert:

```tsx
// BEFORE:
<h2 style={{ fontSize: "20px", fontWeight: "600", marginBottom: "16px" }}>

// AFTER:
<h2 className="text-xl font-semibold mb-4">
```

**Step 4: Convert dialog description**

Find the dialog description paragraph (around line 1006-1008) and convert:

```tsx
// BEFORE:
<p style={{ color: "#6b7280", marginBottom: "16px" }}>

// AFTER:
<p className="text-gray-500 mb-4">
```

**Step 5: Convert error message div**

Find the error message div (around line 1010-1021) and convert:

```tsx
// BEFORE:
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

// AFTER:
{factExtractionError && (
  <div className="p-3 bg-red-50 border border-red-500 rounded-lg text-red-700 mb-4">
    {factExtractionError}
  </div>
)}
```

**Step 6: Convert facts container**

Find the facts container div (around line 1023-1055) and convert:

```tsx
// BEFORE:
<div style={{ marginBottom: "16px" }}>

// AFTER:
<div className="mb-4">
```

Convert the no facts message:

```tsx
// BEFORE:
<p style={{ color: "#6b7280", fontStyle: "italic" }}>

// AFTER:
<p className="text-gray-500 italic">
```

**Step 7: Convert individual fact item**

Find the fact item div (around line 1029-1052) and convert:

```tsx
// BEFORE:
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

// AFTER:
<div
  key={index}
  id={`fact-item-${index}`}
  className="p-3 border rounded-lg mb-2 bg-gray-50"
>
```

Convert the label:

```tsx
// BEFORE:
<label style={{ display: "flex", alignItems: "flex-start", gap: "8px", cursor: "pointer" }}>

// AFTER:
<label className="flex items-start gap-2 cursor-pointer">
```

Convert the checkbox:

```tsx
// BEFORE:
<input
  type="checkbox"
  defaultChecked={true}
  style={{ marginTop: "4px" }}
  data-fact-index={index}
/>

// AFTER:
<input
  type="checkbox"
  defaultChecked={true}
  className="mt-1"
  data-fact-index={index}
/>
```

Convert the fact text span:

```tsx
// BEFORE:
<span style={{ fontSize: "14px", color: "#1f2937" }}>

// AFTER:
<span className="text-sm text-gray-800">
```

**Step 8: Convert dialog action buttons**

Find the action buttons container (around line 1057-1116) and convert:

```tsx
// BEFORE:
<div style={{ display: "flex", justifyContent: "flex-end", gap: "12px", paddingTop: "16px", borderTop: "1px solid #e5e7eb" }}>

// AFTER:
<div className="flex justify-end gap-3 pt-4 border-t border-gray-200">
```

Convert the Cancel button:

```tsx
// BEFORE:
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

// AFTER:
<button
  onClick={() => {
    setShowFactDialog(false);
    setExtractedFacts([]);
  }}
  className="px-4 py-2 text-sm font-medium rounded-lg border border-gray-300 bg-white text-gray-700 cursor-pointer hover:bg-gray-50"
>
```

Convert the Save button:

```tsx
// BEFORE:
<button
  onClick={() => {
    // Get all checked facts
    ...
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

// AFTER:
<button
  onClick={() => {
    // Get all checked facts
    ...
  }}
  className="px-4 py-2 text-sm font-medium rounded-lg border-none bg-blue-600 text-white cursor-pointer hover:bg-blue-700"
>
```

**Step 9: Commit**

```bash
git add zettelkasten-front/src/pages/EmailDetailPage.tsx
git commit -m "refactor: convert fact extraction dialog to Tailwind classes"
```

---

## Task 8: Verify and Test

**Files:**
- Test: Visual inspection and interaction testing

**Step 1: Start the frontend development server**

```bash
cd zettelkasten-front
npm start
```

**Step 2: Navigate to an email detail page**

1. Go to http://localhost:5173
2. Navigate to the Emails section
3. Click on an email to open the detail page

**Step 3: Verify visual appearance**

Check the following elements match the original design:
- [ ] Header buttons (Back, Archive, Convert, Create Task, Extract Facts) - correct spacing, colors, borders
- [ ] Email subject heading - correct size and weight
- [ ] Email metadata (From, To, Date, Folder, Status) - correct layout and styling
- [ ] Email body content - displays correctly with proper spacing
- [ ] Attachments section - cards display correctly with icons and buttons
- [ ] Loading state - centered text displays correctly
- [ ] Error state - error message and button display correctly
- [ ] Fact extraction dialog - modal displays correctly

**Step 4: Verify interactive states**

Test the following interactive behaviors:
- [ ] Hover states on header buttons work (background color change)
- [ ] Archive/Unarchive button - disabled state shows correctly
- [ ] Extract Facts button - PRO user styling works correctly
- [ ] Attachment cards - hover state works
- [ ] Fact extraction dialog checkboxes - can toggle
- [ ] Dialog close on overlay click works

**Step 5: Verify responsive behavior**

Test at different viewport sizes:
- [ ] Full desktop (1920x1080) - layout looks correct
- [ ] Tablet (768x1024) - layout adapts correctly
- [ ] Mobile (375x667) - layout adapts correctly

**Step 6: Check console for errors**

Open browser DevTools and verify:
- [ ] No CSS-related errors in console
- [ ] No missing class warnings
- [ ] Email content styles are applied correctly

**Step 7: Final commit**

If all tests pass:

```bash
git add .
git commit -m "test: verify EmailDetailPage Tailwind refactor - all visual and interactive states working"
```

---

## Summary of Changes

**Files Created:**
- `zettelkasten-front/src/components/email/EmailContent.module.css` - Email body content styles

**Files Modified:**
- `zettelkasten-front/src/pages/EmailDetailPage.tsx` - Converted from inline styles to Tailwind

**Lines Removed:**
- ~165 lines of `emailStyles` CSS constant
- ~20 lines of style injection useEffect
- ~200+ lines of inline `style={{}}` props

**Lines Added:**
- ~200 lines in `EmailContent.module.css` (minimal, scoped email content styles)
- Tailwind `className` props replacing inline styles

**Benefits:**
- Consistent with rest of codebase (ViewPage, EmailConvertDialog)
- No runtime style injection overhead
- Better performance (no JavaScript style manipulation)
- Easier to maintain and modify
- Smaller bundle size (shared Tailwind utilities)
