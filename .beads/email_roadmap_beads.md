# Email Feature Roadmap Beads

## Phase 1: Quick Wins

### [feature] Keyboard Shortcuts for Email
Add 'e' keyboard shortcut to navigate to email inbox from anywhere in the app. This aligns with Zettelgarden's existing keyboard shortcut culture (c for cards, t for tasks, s for search).

**Implementation:**
- Add 'e' key to `useKeyboardShortcuts.ts` hook
- Navigate to `/app/emails` on press
- Ensure shortcut doesn't trigger when input/textarea focused

**Files:**
- `zettelkasten-front/src/hooks/useKeyboardShortcuts.ts`
- `zettelkasten-front/src/components/Sidebar.tsx`

**Acceptance Criteria:**
- Pressing 'e' navigates to email inbox from anywhere in app
- Shortcut doesn't trigger when input/textarea focused
- Works consistently with existing 'c', 't', 's' shortcuts

**Priority:** High | **Effort:** Low | **Tier:** Free

---

### [feature] Batch Operations for Emails
Add checkbox selection and bulk actions to the email list. Users can select multiple emails and perform bulk archive, convert to cards, or create tasks.

**Features:**
- Checkbox selection in email list
- Bulk archive/unarchive
- Bulk convert to cards
- Bulk create tasks
- "Select all" option (Ctrl+A)

**API Endpoints:**
- POST /api/emails/batch-archive
- POST /api/emails/batch-convert
- POST /api/emails/batch-create-tasks

**Files:**
- Frontend: `EmailInboxPage.tsx`, `EmailRow.tsx`, `email.ts`
- Backend: `handlers/email_sync.go`, `routes/email.go`

**Acceptance Criteria:**
- Select multiple emails with checkboxes
- Bulk actions apply to all selected
- Confirmation for destructive operations
- Clear selection after action
- Keyboard: Ctrl+A to select all

**Priority:** High | **Effort:** Low-Medium | **Tier:** Free

**Blocks:** None

---

### [feature] Sender-Based Filtering for Emails
Add ability to filter emails by sender. Shows top senders by count with quick filter toggle.

**Implementation:**
- Add "Filter by sender" dropdown to inbox
- Shows top senders by count
- Quick filter toggle
- Backend: Add `from_address` filter to existing list endpoint

**Files:**
- Frontend: `EmailInboxPage.tsx`, `email.ts`
- Backend: `services/emails.go`

**Acceptance Criteria:**
- Click sender name to filter by that sender
- Show sender filter chips
- Clear filter to return to full list
- Persist filter in URL state

**Priority:** Medium-High | **Effort:** Low | **Tier:** Free

---

### [feature] Improved HTML Email Rendering
Improve HTML email rendering with proper sanitization and responsive CSS.

**Implementation:**
- Use proper HTML sanitization library (dompurify)
- Add responsive email CSS
- Better handling of tables, images
- Improve link security

**Files:**
- Frontend: `EmailDetailPage.tsx`
- Consider: `dompurify`, `juice` for inline CSS

**Acceptance Criteria:**
- HTML emails render correctly
- Sanitized content (no XSS)
- Responsive layout
- Images load correctly
- Tables don't overflow

**Priority:** Medium | **Effort:** Low-Medium | **Tier:** Free

---

## Phase 2: Productivity Enhancements

### [feature] Email Search
Add full-text search across email bodies, subjects, and senders using Typesense.

**Implementation:**
- Add emails to Typesense index on sync
- Search across subject, from, body
- Integrate with existing quick search (s key)
- Add search bar to EmailInboxPage

**Files:**
- Backend: New `services/email_search.go`
- Frontend: `EmailInboxPage.tsx`, integrate with `SearchForm.tsx`
- Typesense schema for emails

**Acceptance Criteria:**
- Full-text search across emails
- Quick search from anywhere (s + search query)
- Filter results by sender, date range
- Highlight matches in results

**Priority:** High | **Effort:** Medium | **Tier:** Free

---

### [feature] Email Thread View
Group emails by conversation/thread to show full conversation context.

**Implementation:**
- Group emails by `thread_id` (already in schema)
- Thread detail view showing all messages
- Collapse/expand threads
- Thread-level actions (archive entire thread)

**Database:**
- `thread_id` already exists in emails table

**Files:**
- Frontend: New `EmailThreadPage.tsx`, update `EmailRow.tsx`
- Backend: `services/emails.go` - add thread listing methods
- API: GET /api/emails/threads/{thread_id}

**Acceptance Criteria:**
- Group related emails in inbox
- Thread view shows chronological messages
- Expand/collapse threads
- Archive/mark read entire thread

**Priority:** Medium-High | **Effort:** High | **Tier:** Free

---

### [feature] Email-to-Fact Integration
Add ability to extract facts from emails using AI. PRO feature.

**Implementation:**
- Add "Extract Fact" action to email detail
- Use LLM to identify factual statements
- Pre-fill fact creation dialog
- Link fact to original email

**Files:**
- Frontend: `EmailDetailPage.tsx`, integrate with fact dialog
- Backend: LLM service for fact extraction
- Similar to existing entity extraction

**Acceptance Criteria:**
- AI suggests facts from email content
- User reviews and saves facts
- Facts linked to source email
- PRO-gated feature

**Priority:** Medium | **Effort:** Medium | **Tier:** PRO

---

### [feature] Email Attachment Handling
Parse and store email attachments, display in email detail with download/save to file vault options.

**Implementation:**
- Parse attachments during IMAP sync
- Store in S3 (existing infrastructure)
- Display attachment list in email detail
- Download / save to file vault actions

**Database:**
- New `email_attachments` table

**Files:**
- Backend: `services/imap_client.go` - attachment parsing
- Frontend: `EmailDetailPage.tsx` - attachment display
- Integration with file vault

**Acceptance Criteria:**
- Show attachments in email detail
- Download attachments
- Save to file vault
- Image thumbnails

**Priority:** Medium | **Effort:** Medium-High | **Tier:** Free

---

## Phase 3: Advanced Features

### [feature] AI-Powered Email Triage
Implement AI-powered email triage that automatically suggests actions for new emails. PRO feature.

**Implementation:**
- Background job analyzes new emails
- Categories: archive, delete, keep, convert
- Confidence scoring
- Graduated trust: high confidence = auto-action
- Review queue for low-confidence decisions

**Database:**
- `email_triage_decisions` table (designed but not implemented)

**Files:**
- Backend: `services/email_triage.go`, job scheduler
- Frontend: Review UI, trust settings
- LLM integration

**Acceptance Criteria:**
- AI suggests actions for new emails
- Confidence scores displayed
- High-confidence actions auto-execute
- User can override and train system
- PRO-gated feature

**Priority:** Medium-High | **Effort:** High | **Tier:** PRO

---

### [feature] Smart Email Suggestions
Proactively suggest cards to create, tags to add, and facts to extract from email content. PRO feature.

**Implementation:**
- Analyze email for card-worthy content
- Suggest related cards to link
- Suggest tags based on content
- Suggest facts extraction
- Learning from user behavior

**Files:**
- Backend: LLM analysis service
- Frontend: Suggestions panel in email detail
- Integration with search for related cards

**Acceptance Criteria:**
- AI suggests when email should be card
- Shows related existing cards
- Suggests relevant tags
- Suggests facts to extract
- PRO-gated feature

**Priority:** Medium | **Effort:** High | **Tier:** PRO

---

### [feature] Email Analytics Dashboard
Add analytics dashboard showing email patterns and metrics. PRO feature.

**Features:**
- Email volume over time
- Top senders
- Response time metrics
- Conversion rate (emails to cards)
- Archive vs. keep ratio

**Files:**
- Frontend: New `EmailAnalyticsPage.tsx`
- Backend: Analytics queries
- Charts: integrate with existing charting library

**Acceptance Criteria:**
- Visual dashboard of email stats
- Date range filtering
- Export data
- PRO-gated feature

**Priority:** Low-Medium | **Effort:** Medium | **Tier:** PRO

---

## Phase 4: Polish & Performance

### [feature] Email Performance Optimizations
Optimize email feature performance for large inboxes.

**Implementation:**
- Add database indexes on common queries
- Implement pagination for large lists
- Optimize IMAP sync (incremental)
- Cache frequently accessed data
- Lazy loading for email list

**Database:**
- Indexes on `received_at`, `from_address`, `status`

**Files:**
- Backend: `services/emails.go`, `services/imap_client.go`
- Database: migration for indexes

**Acceptance Criteria:**
- Inbox loads in <1s for 1000 emails
- Sync completes quickly
- Smooth scrolling
- No memory leaks

**Priority:** Medium | **Effort:** Medium | **Tier:** Free

---

### [feature] Mobile Email Experience
Optimize email experience for mobile devices with swipe gestures and responsive layout.

**Implementation:**
- Responsive email list
- Swipe gestures (archive, delete)
- Mobile-optimized email detail
- Bottom sheet for actions
- Better touch targets

**Files:**
- Frontend: `EmailRow.tsx`, `EmailDetailPage.tsx`
- Pattern: Copy from RSS mobile implementation

**Acceptance Criteria:**
- Swipe to archive
- Swipe to convert
- Responsive layout
- Touch-friendly buttons

**Priority:** Medium | **Effort:** Medium | **Tier:** Free

---

### [feature] Email Settings & Preferences
Add email preferences section for customization.

**Features:**
- Default sync interval
- Auto-archive rules
- Notification preferences
- Display settings (density, preview)

**Database:**
- `email_preferences` table (similar to notification_preferences)

**Files:**
- Frontend: Settings section in user settings
- Backend: Preferences API

**Acceptance Criteria:**
- User can set preferences
- Preferences persist
- Applied consistently

**Priority:** Low-Medium | **Effort:** Low | **Tier:** Free

---

### [feature] Email Error Handling & Edge Cases
Improve error handling and add retry logic for email sync failures.

**Implementation:**
- Better sync error messages
- Retry logic with exponential backoff
- Account status indicators
- Graceful degradation
- Error recovery actions

**Files:**
- Frontend: Error states, toast notifications
- Backend: Error handling in sync job

**Acceptance Criteria:**
- Clear error messages
- Retry failures automatically
- Manual retry option
- Account status visible

**Priority:** Medium | **Effort:** Low-Medium | **Tier:** Free

---

## Phase 5: PRO Feature Differentiation

### [feature] Email Sender-Based Rules
Add automation rules for email processing based on sender. PRO feature.

**Features:**
- Auto-archive newsletters
- Auto-convert from specific senders
- Priority senders (never archive)
- Custom routing rules

**Database:**
- `email_rules` table

**Acceptance Criteria:**
- Create rules per sender
- Rules apply automatically
- Test rules before applying
- PRO-gated

**Priority:** Medium | **Effort:** Medium-High | **Tier:** PRO

---

### [feature] Email Templates & Quick Actions
Add templates for quick email-to-card conversions. PRO feature.

**Features:**
- Save conversion templates
- Quick tag presets
- Bulk action presets
- Custom card templates per sender

**Acceptance Criteria:**
- Create templates from email
- Apply templates on conversion
- PRO-gated

**Priority:** Low-Medium | **Effort:** Medium | **Tier:** PRO
