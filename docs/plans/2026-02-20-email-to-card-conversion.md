# Email to Card Conversion Implementation Plan

> **For Claude:** REQUIRED SUB-SKILL: Use superpowers:executing-plans to implement this plan task-by-task.

**Goal:** Add ability to convert emails to cards, mirroring the existing RSS article conversion feature.

**Architecture:** Create backend endpoint for email-to-card conversion, frontend dialog component, and integrate into EmailDetailPage with visual indicator.

**Tech Stack:** Go (backend), React/TypeScript (frontend), PostgreSQL (existing email_card_links table)

---

## Task 1: Add card_id to Email model

**Files:**
- Modify: `go-backend/models/email_sync.go:23-42`

**Step 1: Add CardID field to Email struct**

Add the `CardID` field to the `Email` struct after `IsRead`:

```go
type Email struct {
	ID             int        `json:"id"`
	UserID         int        `json:"user_id"`
	EmailAccountID *int       `json:"email_account_id,omitempty"`
	MessageID      string     `json:"message_id"`
	ThreadID       *string    `json:"thread_id,omitempty"`
	Subject        *string    `json:"subject,omitempty"`
	FromAddress    *string    `json:"from_address,omitempty"`
	FromName       *string    `json:"from_name,omitempty"`
	ToAddresses    *string    `json:"to_addresses,omitempty"`
	BodyText       *string    `json:"body_text,omitempty"`
	BodyHTML       *string    `json:"body_html,omitempty"`
	ReceivedAt     *time.Time `json:"received_at,omitempty"`
	Folder         *string    `json:"folder,omitempty"`
	IMAPUID        *int64     `json:"imap_uid,omitempty"`
	Status         string     `json:"status"`
	IsRead         bool       `json:"is_read"`
	CardID         *int       `json:"card_id,omitempty"` // Add this line
	CreatedAt      time.Time  `json:"created_at"`
	UpdatedAt      time.Time  `json:"updated_at"`
}
```

**Step 2: Commit**

```bash
cd go-backend
git add models/email_sync.go
git commit -m "feat: add card_id to Email model"
```

---

## Task 2: Add ConvertEmailParams struct

**Files:**
- Modify: `go-backend/models/email_sync.go:63-83` (after EmailListFilters)

**Step 1: Add ConvertEmailParams struct**

Add after the `EmailListFilters` struct:

```go
// ConvertEmailParams represents parameters for converting an email to a card
type ConvertEmailParams struct {
	Title   string  `json:"title"`
	Body    *string `json:"body,omitempty"`
	Tags    *string `json:"tags,omitempty"`
	CardID  *string `json:"card_id,omitempty"` // For linking to existing card
}
```

**Step 2: Commit**

```bash
cd go-backend
git add models/email_sync.go
git commit -m "feat: add ConvertEmailParams struct"
```

---

## Task 3: Add conversion route

**Files:**
- Modify: `go-backend/routes/email.go:8-21`

**Step 1: Add convert route**

Add the route after the UpdateEmailStatusRoute:

```go
func RegisterEmailRoutes(r *mux.Router, h *handlers.Handler) {
	// Email account management routes
	addProtectedRoute(r, h, "/api/email/accounts", h.ListEmailAccountsRoute, "GET")
	addProtectedRoute(r, h, "/api/email/accounts", h.CreateEmailAccountRoute, "POST")
	addProtectedRoute(r, h, "/api/email/accounts/{id}", h.GetEmailAccountRoute, "GET")
	addProtectedRoute(r, h, "/api/email/accounts/{id}", h.DeleteEmailAccountRoute, "DELETE")
	addProtectedRoute(r, h, "/api/email/accounts/{id}/sync", h.SyncEmailAccountRoute, "POST")

	// Email message routes
	addProtectedRoute(r, h, "/api/emails", h.ListEmailsRoute, "GET")
	addProtectedRoute(r, h, "/api/emails/{id}", h.GetEmailRoute, "GET")
	addProtectedRoute(r, h, "/api/emails/{id}/status", h.UpdateEmailStatusRoute, "PATCH")
	addProtectedRoute(r, h, "/api/emails/{id}/convert", h.ConvertEmailToCardRoute, "POST") // Add this line
	addProtectedRoute(r, h, "/api/emails/stats", h.GetEmailStatsRoute, "GET")
}
```

**Step 2: Commit**

```bash
cd go-backend
git add routes/email.go
git commit -m "feat: add email convert route"
```

---

## Task 4: Add conversion handler

**Files:**
- Modify: `go-backend/handlers/email_sync.go` (at end of file, after UpdateEmailStatusRoute)

**Step 1: Add ConvertEmailToCardRoute handler**

Add this function at the end of the file:

```go
// ConvertEmailToCardRoute handles POST /api/emails/{id}/convert
// Converts an email to a card, optionally linking to an existing card
func (h *Handler) ConvertEmailToCardRoute(w http.ResponseWriter, r *http.Request) {
	userID := r.Context().Value("current_user").(int)

	// Get email ID from path
	vars := mux.Vars(r)
	idStr := vars["id"]
	emailID, err := strconv.Atoi(idStr)
	if err != nil {
		http.Error(w, "Invalid email ID", http.StatusBadRequest)
		return
	}

	// Parse request body
	var params models.ConvertEmailParams
	decoder := json.NewDecoder(r.Body)
	if err := decoder.Decode(&params); err != nil {
		http.Error(w, "Invalid request body", http.StatusBadRequest)
		return
	}

	// Validate title
	if params.Title == "" {
		http.Error(w, "Title is required", http.StatusBadRequest)
		return
	}

	emailService := services.NewEmailService(h.GetDB())
	cardService := services.NewCardService(h.GetDB())
	db := h.GetDB()

	// Get the email to verify ownership
	email, err := emailService.GetEmailByID(context.Background(), userID, emailID)
	if err != nil {
		if err.Error() == "email not found" {
			http.Error(w, "Email not found", http.StatusNotFound)
			return
		}
		log.Printf("[email] failed to get email for conversion: %v", err)
		http.Error(w, "Failed to get email", http.StatusInternalServerError)
		return
	}

	var cardID int

	// If card_id provided, update existing card; otherwise create new
	if params.CardID != nil && *params.CardID != "" {
		// Parse card ID - may be in format like "20250131123456"
		parsedCardID, err := services.ParseCardID(*params.CardID)
		if err != nil {
			http.Error(w, "Invalid card ID format", http.StatusBadRequest)
			return
		}

		// Verify card belongs to user and update it
		card, err := cardService.GetCardByID(context.Background(), userID, parsedCardID)
		if err != nil {
			http.Error(w, "Card not found", http.StatusNotFound)
			return
		}

		// Update card with new content
		updateParams := services.UpdateCardParams{
			Title: &params.Title,
		}
		if params.Body != nil {
			updateParams.Body = params.Body
		}
		if params.Tags != nil {
			updateParams.RawTags = params.Tags
		}

		_, err = cardService.UpdateCard(context.Background(), userID, card.ID, updateParams)
		if err != nil {
			log.Printf("[email] failed to update card: %v", err)
			http.Error(w, "Failed to update card", http.StatusInternalServerError)
			return
		}

		cardID = card.ID
	} else {
		// Create new card
		createParams := services.CreateCardParams{
			Title:   params.Title,
			Content: params.Body,
		}
		if params.Tags != nil {
			createParams.RawTags = params.Tags
		}

		card, err := cardService.CreateCard(context.Background(), userID, createParams)
		if err != nil {
			log.Printf("[email] failed to create card: %v", err)
			http.Error(w, "Failed to create card", http.StatusInternalServerError)
			return
		}

		cardID = card.ID
	}

	// Create email_card_link record
	// First check if link already exists
	var existingLinkID int
	err = db.QueryRowContext(context.Background(),
		"SELECT id FROM email_card_links WHERE email_id = $1",
		emailID).Scan(&existingLinkID)

	if err == sql.ErrNoRows {
		// No existing link, create one
		_, err = db.ExecContext(context.Background(),
			"INSERT INTO email_card_links (email_id, card_id) VALUES ($1, $2)",
			emailID, cardID)
		if err != nil {
			log.Printf("[email] failed to create email_card_link: %v", err)
			// Don't fail the request - card was created/updated successfully
		}
	} else if err != nil {
		log.Printf("[email] error checking email_card_link: %v", err)
		// Don't fail the request
	}

	// Update email's card_id
	_, err = db.ExecContext(context.Background(),
		"UPDATE emails SET card_id = $1 WHERE id = $2",
		cardID, emailID)
	if err != nil {
		log.Printf("[email] failed to update email card_id: %v", err)
		// Don't fail the request
	}

	log.Printf("[email] converted email %d to card %d", emailID, cardID)

	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusOK)
	json.NewEncoder(w).Encode(map[string]interface{}{
		"id": cardID,
	})
}
```

**Step 2: Commit**

```bash
cd go-backend
git add handlers/email_sync.go
git commit -m "feat: add email to card conversion handler"
```

---

## Task 5: Update Email service to include card_id in queries

**Files:**
- Modify: `go-backend/services/emails.go`

**Step 1: Find and update GetEmailByID query**

Look for the SELECT query in `GetEmailByID` and add `card_id` to the selected columns. The query should include:

```go
query := `
	SELECT id, user_id, email_account_id, message_id, thread_id, subject,
	       from_address, from_name, to_addresses, body_text, body_html,
	       received_at, folder, imap_uid, status, is_read, card_id,
	       created_at, updated_at
	FROM emails
	WHERE id = $1 AND user_id = $2
`
```

**Step 2: Update the scan to include CardID**

Make sure the scan includes `&email.CardID`:

```go
err := row.Scan(
	&email.ID, &email.UserID, &email.EmailAccountID, &email.MessageID,
	&email.ThreadID, &email.Subject, &email.FromAddress, &email.FromName,
	&email.ToAddresses, &email.BodyText, &email.BodyHTML, &email.ReceivedAt,
	&email.Folder, &email.IMAPUID, &email.Status, &email.IsRead,
	&email.CardID, &email.CreatedAt, &email.UpdatedAt,
)
```

**Step 3: Commit**

```bash
cd go-backend
git add services/emails.go
git commit -m "feat: include card_id in email queries"
```

---

## Task 6: Add card_id to frontend Email interface

**Files:**
- Modify: `zettelkasten-front/src/api/email.ts:19-37`

**Step 1: Add card_id to Email interface**

Add `card_id?: number;` after `is_read`:

```typescript
export interface Email {
  id: number;
  user_id: number;
  email_account_id?: number;
  message_id: string;
  thread_id?: string;
  subject?: string;
  from_address?: string;
  from_name?: string;
  to_addresses?: string;
  body_text?: string;
  body_html?: string;
  received_at?: string;
  folder?: string;
  status: string;
  is_read: boolean;
  card_id?: number; // Add this line
  created_at: string;
  updated_at: string;
}
```

**Step 2: Commit**

```bash
cd zettelkasten-front
git add src/api/email.ts
git commit -m "feat: add card_id to Email interface"
```

---

## Task 7: Add conversion API function

**Files:**
- Modify: `zettelkasten-front/src/api/email.ts` (at end of file, before export)

**Step 1: Add ConvertEmailParams interface and convertEmailToCard function**

Add at the end of the file:

```typescript
export interface ConvertEmailParams {
  title?: string;
  body?: string;
  tags?: string;
  card_id?: string;
}

export interface ConvertCardResponse {
  id: number;
}

export function convertEmailToCard(id: number, params?: ConvertEmailParams): Promise<ConvertCardResponse> {
  return getData(apiClient.post<ConvertCardResponse>(`/emails/${id}/convert`, params));
}
```

**Step 2: Commit**

```bash
cd zettelkasten-front
git add src/api/email.ts
git commit -m "feat: add convertEmailToCard API function"
```

---

## Task 8: Create EmailConvertDialog component

**Files:**
- Create: `zettelkasten-front/src/components/email/EmailConvertDialog.tsx`

**Step 1: Create the component file**

Create the new component:

```typescript
import React, { useState, useEffect } from "react";
import { Dialog, Transition } from "@headlessui/react";
import { Fragment } from "react";
import { convertEmailToCard, ConvertEmailParams, Email, ConvertCardResponse } from "../../api/email";
import { safeHtmlToMarkdown } from "../../utils/markdown";
import { CardIdDiscoveryDialog } from "../cards/CardIdDiscoveryDialog";
import { SearchTagDropdown } from "../tags/SearchTagDropdown";
import { useTagContext } from "../../contexts/TagContext";

interface EmailConvertDialogProps {
  isOpen: boolean;
  onClose: () => void;
  email: Email | null;
  onConverted: (cardId: number) => void;
}

export function EmailConvertDialog({
  isOpen,
  onClose,
  email,
  onConverted,
}: EmailConvertDialogProps) {
  const { tags: allTags } = useTagContext();
  const [title, setTitle] = useState("");
  const [body, setBody] = useState("");
  const [selectedTags, setSelectedTags] = useState<string[]>([]);
  const [cardId, setCardId] = useState("");
  const [showCardIdDiscovery, setShowCardIdDiscovery] = useState(false);
  const [loading, setLoading] = useState(false);
  const [error, setError] = useState<string>("");

  // Initialize title and body from email when it changes
  useEffect(() => {
    if (email) {
      setTitle(email.subject || "");
      // Convert HTML content to markdown for editing
      if (email.body_html) {
        setBody(safeHtmlToMarkdown(email.body_html));
      } else if (email.body_text) {
        setBody(email.body_text);
      } else {
        setBody("");
      }
      setSelectedTags([]);
      setCardId("");
    }
  }, [email]);

  const handleSubmit = async (e: React.FormEvent) => {
    e.preventDefault();
    if (!email) return;

    if (!title.trim()) {
      setError("Title is required");
      return;
    }

    setLoading(true);
    setError("");

    try {
      const params: ConvertEmailParams = {
        title: title.trim(),
      };

      if (body.trim()) {
        params.body = body.trim();
      }

      if (selectedTags.length > 0) {
        params.tags = selectedTags.map(t => `#${t.replace(/^#/, '')}`).join(' ');
      }

      if (cardId.trim()) {
        params.card_id = cardId.trim();
      }

      const result: ConvertCardResponse = await convertEmailToCard(email.id, params);

      if (result.id) {
        onConverted(result.id);
        handleClose();
      } else {
        setError("Failed to convert email to card");
      }
    } catch (err) {
      console.error("Failed to convert email:", err);
      setError(err instanceof Error ? err.message : "Failed to convert email. Please try again.");
    } finally {
      setLoading(false);
    }
  };

  const handleClose = () => {
    setTitle("");
    setBody("");
    setSelectedTags([]);
    setCardId("");
    setError("");
    onClose();
  };

  const handleCardIdSelected = (selectedCardId: string) => {
    setCardId(selectedCardId);
    setShowCardIdDiscovery(false);
  };

  const handleTagClick = (tagName: string) => {
    const cleanTag = tagName.replace(/^#/, '');
    if (!selectedTags.includes(cleanTag)) {
      setSelectedTags([...selectedTags, cleanTag]);
    }
  };

  const handleRemoveTag = (tagName: string) => {
    setSelectedTags(selectedTags.filter(t => t !== tagName));
  };

  return (
    <>
      <Transition appear show={isOpen} as={Fragment}>
        <Dialog as="div" className="relative z-[80]" onClose={handleClose}>
          <Transition.Child
            as={Fragment}
            enter="ease-out duration-300"
            enterFrom="opacity-0"
            enterTo="opacity-100"
            leave="ease-in duration-200"
            leaveFrom="opacity-100"
            leaveTo="opacity-0"
          >
            <div className="fixed inset-0 bg-black bg-opacity-30" />
          </Transition.Child>

          <div className="fixed inset-0 overflow-y-auto">
            <div className="flex min-h-full items-center justify-center p-4 text-center">
              <Transition.Child
                as={Fragment}
                enter="ease-out duration-300"
                enterFrom="opacity-0 scale-95"
                enterTo="opacity-100 scale-100"
                leave="ease-in duration-200"
                leaveFrom="opacity-100 scale-100"
                leaveTo="opacity-0 scale-95"
              >
                <Dialog.Panel className="w-full max-w-2xl transform overflow-hidden rounded-2xl bg-white p-6 text-left align-middle shadow-xl transition-all">
                  <Dialog.Title as="h3" className="text-lg font-medium leading-6 text-gray-900 mb-4">
                    Convert to Card
                  </Dialog.Title>

                  <form onSubmit={handleSubmit} className="space-y-4">
                    {/* Title - Required */}
                    <div>
                      <label htmlFor="card-title" className="block text-sm font-medium text-gray-700 mb-1">
                        Title <span className="text-red-500">*</span>
                      </label>
                      <input
                        id="card-title"
                        type="text"
                        value={title}
                        onChange={(e) => setTitle(e.target.value)}
                        placeholder="Email subject"
                        className="w-full px-3 py-2 border border-gray-300 rounded-md shadow-sm focus:outline-none focus:ring-2 focus:ring-blue-500 focus:border-blue-500 sm:text-sm"
                        required
                        autoFocus
                      />
                    </div>

                    {/* Content - Optional */}
                    <div>
                      <label htmlFor="card-content" className="block text-sm font-medium text-gray-700 mb-1">
                        Content <span className="text-gray-400">(optional)</span>
                      </label>
                      <textarea
                        id="card-content"
                        value={body}
                        onChange={(e) => setBody(e.target.value)}
                        placeholder="Email content in markdown format..."
                        rows={10}
                        className="w-full px-3 py-2 border border-gray-300 rounded-md shadow-sm focus:outline-none focus:ring-2 focus:ring-blue-500 focus:border-blue-500 sm:text-sm font-mono text-sm resize-y"
                      />
                    </div>

                    {/* Tags - Optional */}
                    <div>
                      <div className="flex items-center justify-between mb-1">
                        <label className="block text-sm font-medium text-gray-700">
                          Tags <span className="text-gray-400">(optional)</span>
                        </label>
                        <SearchTagDropdown
                          tags={allTags}
                          handleTagClick={handleTagClick}
                        />
                      </div>
                      {selectedTags.length > 0 && (
                        <div className="flex flex-wrap gap-1.5 mt-2">
                          {selectedTags.map((tag) => (
                            <span
                              key={tag}
                              className="inline-flex items-center px-1.5 py-0.5 bg-purple-50 text-purple-600 text-xs rounded-full"
                            >
                              <span className="cursor-pointer hover:bg-purple-100">
                                #{tag}
                              </span>
                              <button
                                onClick={() => handleRemoveTag(tag)}
                                className="ml-1.5 text-purple-400 hover:text-purple-600"
                              >
                                &times;
                              </button>
                            </span>
                          ))}
                        </div>
                      )}
                      {selectedTags.length === 0 && (
                        <p className="mt-1 text-xs text-gray-500">
                          Select tags from dropdown to add to the card
                        </p>
                      )}
                    </div>

                    {/* Card ID - Optional */}
                    <div>
                      <label htmlFor="card-id" className="block text-sm font-medium text-gray-700 mb-1">
                        Card ID <span className="text-gray-400">(optional)</span>
                      </label>
                      <div className="flex gap-2">
                        <input
                          id="card-id"
                          type="text"
                          value={cardId}
                          onChange={(e) => setCardId(e.target.value)}
                          className="flex-1 px-3 py-2 border border-gray-300 rounded-md shadow-sm focus:outline-none focus:ring-2 focus:ring-blue-500 focus:border-blue-500 sm:text-sm font-mono text-sm"
                        />
                        <button
                          type="button"
                          onClick={() => setShowCardIdDiscovery(true)}
                          className="px-4 py-2 bg-gray-100 hover:bg-gray-200 border border-gray-300 rounded-md text-sm font-medium transition-colors"
                          title="Discover card ID"
                        >
                          Discover
                        </button>
                      </div>
                    </div>

                    {/* Email metadata */}
                    {email?.from_address && (
                      <div className="text-sm text-gray-500">
                        <span>From: </span>
                        <span className="font-medium">{email.from_name || email.from_address}</span>
                        {email.from_name && email.from_address && <> <{email.from_address}></>}
                      </div>
                    )}

                    {/* Error Message */}
                    {error && (
                      <div className="rounded-md bg-red-50 p-3">
                        <p className="text-sm text-red-800">{error}</p>
                      </div>
                    )}

                    {/* Action Buttons */}
                    <div className="flex justify-end space-x-2 pt-2">
                      <button
                        type="button"
                        onClick={handleClose}
                        disabled={loading}
                        className="px-4 py-2 min-h-[44px] text-gray-700 bg-gray-200 hover:bg-gray-300 rounded-md font-medium transition-colors disabled:opacity-50 disabled:cursor-not-allowed"
                      >
                        Cancel
                      </button>
                      <button
                        type="submit"
                        disabled={loading || !title.trim()}
                        className="px-4 py-2 min-h-[44px] bg-blue-600 text-white hover:bg-blue-700 rounded-md font-medium transition-colors disabled:opacity-50 disabled:cursor-not-allowed flex items-center gap-2"
                      >
                        {loading ? (
                          <>
                            <svg className="w-4 h-4 animate-spin" fill="none" viewBox="0 0 24 24">
                              <circle className="opacity-25" cx="12" cy="12" r="10" stroke="currentColor" strokeWidth="4" />
                              <path className="opacity-75" fill="currentColor" d="M4 12a8 8 0 018-8V0C5.373 0 0 5.373 0 12h4zm2 5.291A7.962 7.962 0 014 12H0c0 3.042 1.135 5.824 3 7.938l3-2.647z" />
                            </svg>
                            Converting...
                          </>
                        ) : (
                          <>
                            <svg className="w-4 h-4" fill="currentColor" viewBox="0 0 20 20">
                              <path d="M7 3a1 1 0 000 2h6a1 1 0 100-2H7zM4 7a1 1 0 011-1h10a1 1 0 110 2H5a1 1 0 01-1-1zM2 11a2 2 0 012-2h12a2 2 0 012 2v4a2 2 0 01-2 2H4a2 2 0 01-2-2v-4z" />
                            </svg>
                            Convert to Card
                          </>
                        )}
                      </button>
                    </div>
                  </form>
                </Dialog.Panel>
              </Transition.Child>
            </div>
          </div>
        </Dialog>
      </Transition>

      {/* Card ID Discovery Dialog */}
      {showCardIdDiscovery && (
        <CardIdDiscoveryDialog
          onClose={() => setShowCardIdDiscovery(false)}
          onSelectId={handleCardIdSelected}
        />
      )}
    </>
  );
}
```

**Step 2: Create email components directory if needed**

```bash
mkdir -p zettelkasten-front/src/components/email
```

**Step 3: Commit**

```bash
cd zettelkasten-front
git add src/components/email/EmailConvertDialog.tsx
git commit -m "feat: add EmailConvertDialog component"
```

---

## Task 9: Update EmailDetailPage with convert button and dialog

**Files:**
- Modify: `zettelkasten-front/src/pages/EmailDetailPage.tsx`

**Step 1: Add import for EmailConvertDialog**

Add to imports at top:

```typescript
import { EmailConvertDialog } from "../components/email/EmailConvertDialog";
```

**Step 2: Add state for convert dialog**

Add to state declarations (after `showCreateTaskWindow`):

```typescript
const [showConvertDialog, setShowConvertDialog] = useState(false);
```

**Step 3: Add handleConvertEmail function**

Add after `handleCreateTaskFromEmail`:

```typescript
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
```

**Step 4: Add Convert to Card button in header**

Add button between Archive and Create Task buttons (after line 227, before Create Task button):

```typescript
<button
  onClick={handleConvertEmail}
  style={{
    padding: "8px 16px",
    fontSize: "14px",
    fontWeight: "500",
    borderRadius: "8px",
    border: email?.card_id ? "1px solid #22c55e" : "1px solid #d1d5db",
    backgroundColor: email?.card_id ? "#dcfce7" : "#ffffff",
    color: email?.card_id ? "#15803d" : "#374151",
    cursor: "pointer",
    display: "flex",
    alignItems: "center",
    gap: "6px",
    transition: "all 0.15s ease",
  }}
  onMouseEnter={(e) => {
    e.currentTarget.style.backgroundColor = email?.card_id ? "#d1fae5" : "#f9fafb";
    e.currentTarget.style.borderColor = email?.card_id ? "#16a34a" : "#9ca3af";
  }}
  onMouseLeave={(e) => {
    e.currentTarget.style.backgroundColor = email?.card_id ? "#dcfce7" : "#ffffff";
    e.currentTarget.style.borderColor = email?.card_id ? "#22c55e" : "#d1d5db";
  }}
>
  {email?.card_id ? (
    <>
      <svg style={{ width: "16px", height: "16px" }} fill="currentColor" viewBox="0 0 20 20">
        <path d="M7 3a1 1 0 000 2h6a1 1 0 100-2H7zM4 7a1 1 0 011-1h10a1 1 0 110 2H5a1 1 0 01-1-1zM2 11a2 2 0 012-2h12a2 2 0 012 2v4a2 2 0 01-2 2H4a2 2 0 01-2-2v-4z" />
      </svg>
      View Card
    </>
  ) : (
    <>
      <svg style={{ width: "16px", height: "16px" }} fill="currentColor" viewBox="0 0 20 20">
        <path d="M7 3a1 1 0 000 2h6a1 1 0 100-2H7zM4 7a1 1 0 011-1h10a1 1 0 110 2H5a1 1 0 01-1-1zM2 11a2 2 0 012-2h12a2 2 0 012 2v4a2 2 0 01-2 2H4a2 2 0 01-2-2v-4z" />
      </svg>
      Convert to Card
    </>
  )}
</button>
```

**Step 5: Add EmailConvertDialog component at end**

Add before closing `</div>` (after CreateTaskWindow):

```typescript
{/* Convert to Card Dialog */}
{showConvertDialog && (
  <EmailConvertDialog
    isOpen={showConvertDialog}
    email={email}
    onClose={handleCloseConvertDialog}
    onConverted={handleEmailConverted}
  />
)}
```

**Step 6: Commit**

```bash
cd zettelkasten-front
git add src/pages/EmailDetailPage.tsx
git commit -m "feat: add convert to card button and dialog to EmailDetailPage"
```

---

## Task 10: Test the implementation

**Step 1: Start backend**

```bash
cd go-backend
go run main.go
```

**Step 2: Start frontend**

```bash
cd zettelkasten-front
npm start
```

**Step 3: Manual testing**

1. Navigate to an email in the inbox
2. Click "Convert to Card" button
3. Verify dialog opens with pre-filled title and content
4. Edit title/content if desired
5. Add a tag
6. Click "Convert to Card"
7. Verify button changes to "View Card" with green styling
8. Refresh page and verify card_id persists
9. Click "View Card" - navigate to the created card

**Step 4: Test linking to existing card**

1. Open another email
2. Click "Convert to Card"
3. Click "Discover" button to find an existing card
4. Select a card
5. Convert and verify it updates the existing card

**Step 5: Final commit if any tweaks needed**

```bash
git add -A
git commit -m "fix: any tweaks from testing"
```

---

## Summary

This implementation adds email-to-card conversion mirroring the existing RSS article conversion:

1. **Backend**: New route and handler for conversion, updates Email model with card_id
2. **Frontend API**: New convertEmailToCard function
3. **Frontend Component**: EmailConvertDialog for user interaction
4. **UI Integration**: Button in EmailDetailPage header with visual status indicator
5. **Database**: Uses existing email_card_links table

The feature maintains parity with RSS conversion while handling email-specific content (HTML-to-markdown conversion, from/to addresses in metadata).
