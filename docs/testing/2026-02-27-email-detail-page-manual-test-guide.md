# EmailDetailPage Manual Testing Guide

**Date**: 2026-02-27
**Component**: EmailDetailPage
**Refactor**: Tailwind CSS conversion
**Server URL**: http://localhost:5176/

## Prerequisites

1. Development server running on http://localhost:5176/
2. Valid user account with login credentials
3. At least one email in the system (for testing)
4. For PRO features: PRO user account

## Test Environment Setup

### Starting the Server
```bash
cd /home/nick/code/Zettelgarden/zettelkasten-front
npm start
```

The server should display:
```
VITE v5.3.4  ready in 179 ms
➜  Local:   http://localhost:5176/
```

## Manual Test Cases

### TC-001: Page Load and Display
**Priority**: High
**Preconditions**: User is logged in

**Steps**:
1. Navigate to http://localhost:5176/
2. Click on "Emails" in the sidebar
3. Click on any email in the email list

**Expected Results**:
- Email detail page loads without errors
- No console errors in browser DevTools
- Page displays with proper layout
- All sections are visible (header, metadata, body, attachments if present)

**Actual Results**: _______________________

**Status**: [ ] Pass [ ] Fail

---

### TC-002: Header Buttons - Back Button
**Priority**: High

**Steps**:
1. Open any email detail page
2. Locate the "← Back to Inbox" button
3. Hover over the button
4. Click the button

**Expected Results**:
- Button displays with white background, gray border
- Icon (←) and text aligned with proper spacing (gap-1.5)
- Hover state: background changes to gray-50, border darkens
- Smooth transition (150ms)
- Click navigates back to email list

**Actual Results**: _______________________

**Status**: [ ] Pass [ ] Fail

---

### TC-003: Header Buttons - Archive Button (Unarchived Email)
**Priority**: High

**Steps**:
1. Open an unarchived email (status: unprocessed or triaged)
2. Locate the "📁 Archive" button
3. Click the button

**Expected Results**:
- Button displays with white background, gray border
- Icon (📁) and text aligned properly
- Hover state works correctly
- On click: button shows loading state (gray background, "..." text)
- After completion: button changes to "↱ Unarchive" with yellow styling
- Email status updates to "archived"

**Actual Results**: _______________________

**Status**: [ ] Pass [ ] Fail

---

### TC-004: Header Buttons - Unarchive Button (Archived Email)
**Priority**: High

**Steps**:
1. Open an archived email
2. Locate the "↱ Unarchive" button
3. Click the button

**Expected Results**:
- Button displays with yellow background (bg-yellow-50)
- Yellow text (text-yellow-800)
- Yellow border (border-yellow-200)
- On click: shows loading state
- After completion: button changes to "📁 Archive" with default styling

**Actual Results**: _______________________

**Status**: [ ] Pass [ ] Fail

---

### TC-005: Header Buttons - Convert to Card (Not Converted)
**Priority**: High

**Steps**:
1. Open an email that hasn't been converted to a card
2. Locate the "Convert to Card" button
3. Click the button

**Expected Results**:
- Button displays with white background, gray border
- SVG icon displays correctly (card icon)
- Hover state: background changes to gray-50
- On click: EmailConvertDialog opens
- After conversion: button changes to "View Card" with green styling

**Actual Results**: _______________________

**Status**: [ ] Pass [ ] Fail

---

### TC-006: Header Buttons - View Card (Already Converted)
**Priority**: Medium

**Steps**:
1. Open an email that has been converted to a card
2. Locate the "View Card" button
3. Click the button

**Expected Results**:
- Button displays with green background (bg-green-100)
- Green text (text-green-800)
- Green border (border-green-200)
- On click: navigates to the card detail page

**Actual Results**: _______________________

**Status**: [ ] Pass [ ] Fail

---

### TC-007: Header Buttons - Create Task
**Priority**: High

**Steps**:
1. Open any email
2. Locate the "✚ Create Task" button
3. Click the button

**Expected Results**:
- Button displays with white background, gray border
- Icon (✚) and text aligned properly
- Hover state works correctly
- On click: CreateTaskWindow dialog opens
- Task is pre-filled with email subject in title/description

**Actual Results**: _______________________

**Status**: [ ] Pass [ ] Fail

---

### TC-008: Header Buttons - Extract Facts (PRO User)
**Priority**: High (for PRO users)
**Preconditions**: User has PRO subscription

**Steps**:
1. Open any email as a PRO user
2. Locate the "Extract Facts" button
3. Click the button

**Expected Results**:
- Button displays with white background, gray border
- SVG icon and text "Extract Facts"
- Hover state works correctly
- On click: shows loading state ("..." text)
- After completion: Fact extraction dialog opens with extracted facts

**Actual Results**: _______________________

**Status**: [ ] Pass [ ] Fail [ ] N/A (not PRO user)

---

### TC-009: Header Buttons - Extract Facts (Non-PRO User)
**Priority**: Medium
**Preconditions**: User does NOT have PRO subscription

**Steps**:
1. Open any email as a non-PRO user
2. Locate the "👑 Extract Facts" button
3. Hover over the button

**Expected Results**:
- Button displays with yellow background (bg-yellow-50)
- Yellow text (text-yellow-800)
- Yellow border (border-yellow-400)
- Crown icon (👑) visible
- Title attribute shows "PRO feature: Extract facts from email using AI"
- On click: alert shows "Fact extraction is a PRO feature..."

**Actual Results**: _______________________

**Status**: [ ] Pass [ ] Fail [ ] N/A (PRO user)

---

### TC-010: Email Subject Display
**Priority**: High

**Steps**:
1. Open an email with a subject
2. Open an email without a subject

**Expected Results**:
- Subject displays as h1 (text-2xl, font-bold)
- Color: text-gray-900
- Bottom margin: mb-6
- Line height: leading-tight
- Emails without subject show "(No subject)"

**Actual Results**: _______________________

**Status**: [ ] Pass [ ] Fail

---

### TC-011: Email Metadata - From Field
**Priority**: High

**Steps**:
1. Open any email
2. Check the "From:" field

**Expected Results**:
- Label: "FROM:" in small uppercase gray text
- Value: email address in larger gray-800 text
- Format: "Name <address>" or just "address"
- Proper spacing between label and value (ml-2)

**Actual Results**: _______________________

**Status**: [ ] Pass [ ] Fail

---

### TC-012: Email Metadata - To Field
**Priority**: Medium

**Steps**:
1. Open an email with to addresses
2. Check the "To:" field

**Expected Results**:
- Same styling as From field
- Only displays when email.to_addresses exists
- Shows all recipient addresses

**Actual Results**: _______________________

**Status**: [ ] Pass [ ] Fail

---

### TC-013: Email Metadata - Status Badge
**Priority**: High

**Steps**:
1. Open emails with different statuses (unprocessed, triaged, archived)
2. Check the status badge

**Expected Results**:
- Unprocessed: bg-yellow-100 text-yellow-800
- Triaged: bg-green-100 text-green-800
- Archived: bg-gray-100 text-gray-700
- Rounded: rounded
- Small padding: px-2 py-0.5
- Text matches email status

**Actual Results**: _______________________

**Status**: [ ] Pass [ ] Fail

---

### TC-014: Email Metadata - Read Badge
**Priority**: High

**Steps**:
1. Open read and unread emails
2. Check the "Read:" badge

**Expected Results**:
- Yes: bg-green-100 text-green-800
- No: bg-blue-100 text-blue-800
- Same styling as status badge (rounded, padding)
- Text matches email.is_read state

**Actual Results**: _______________________

**Status**: [ ] Pass [ ] Fail

---

### TC-015: Email Body - HTML Content
**Priority**: High

**Steps**:
1. Open an email with HTML body
2. Check the rendered content
3. Test links in the email
4. Check images (if present)

**Expected Results**:
- HTML renders correctly with EmailContent.module.css styles
- Links are blue and underlined
- Images are responsive (max-width: 100%)
- Tables render with borders
- Code blocks have dark background
- No layout issues or overflow

**Actual Results**: _______________________

**Status**: [ ] Pass [ ] Fail

---

### TC-016: Email Body - Text Content
**Priority**: High

**Steps**:
1. Open an email with only text body (no HTML)
2. Check the rendered content

**Expected Results**:
- Text displays in <pre> tag
- Font inherits from parent (font-inherit)
- Size: text-base
- Line height: leading-relaxed
- White space preserved: whitespace-pre-wrap
- Text wraps: break-word
- Color: text-gray-800

**Actual Results**: _______________________

**Status**: [ ] Pass [ ] Fail

---

### TC-017: Attachments Section - Display
**Priority**: High
**Preconditions**: Email has attachments

**Steps**:
1. Open an email with attachments
2. Check the attachments section

**Expected Results**:
- Section header shows "Attachments (N)" where N is count
- Header: text-base font-semibold text-gray-900
- Border top separates from email body
- Attachment cards display in vertical stack (flex-col gap-3)

**Actual Results**: _______________________

**Status**: [ ] Pass [ ] Fail [ ] N/A (no attachments)

---

### TC-018: Attachment Card - Image Attachment
**Priority**: Medium
**Preconditions**: Email has image attachment

**Steps**:
1. Open an email with an image attachment
2. Check the attachment card

**Expected Results**:
- Thumbnail displays (w-12 h-12)
- Image: object-cover rounded border border-gray-200
- Filename displays with truncation if long
- File size shows in human-readable format
- Content type shows (e.g., "PNG", "PDF")

**Actual Results**: _______________________

**Status**: [ ] Pass [ ] Fail [ ] N/A (no image attachments)

---

### TC-019: Attachment Card - Non-Image Attachment
**Priority**: Medium
**Preconditions**: Email has non-image attachment

**Steps**:
1. Open an email with a non-image attachment
2. Check the attachment card

**Expected Results**:
- Paperclip icon (📎) displays in gray box (w-12 h-12)
- Box background: bg-gray-100
- Filename and metadata display correctly

**Actual Results**: _______________________

**Status**: [ ] Pass [ ] Fail [ ] N/A (all attachments are images)

---

### TC-020: Attachment Actions - Download
**Priority**: High
**Preconditions**: Email has attachments

**Steps**:
1. Open an email with attachments
2. Click the "Download" button on an attachment

**Expected Results**:
- Button displays with border, white background
- Text: "Download"
- Hover state: background changes to gray-50
- On click: attachment downloads or opens in new tab

**Actual Results**: _______________________

**Status**: [ ] Pass [ ] Fail [ ] N/A (no attachments)

---

### TC-021: Attachment Actions - Save to Vault
**Priority**: High
**Preconditions**: Email has unsaved attachments

**Steps**:
1. Open an email with unsaved attachments
2. Click the "Save to Vault" button

**Expected Results**:
- Button displays next to Download button
- Hover state: background changes to blue-50, border to blue-500
- On click: attachment saves to file vault
- Button disappears after saving
- Metadata updates to show "Saved to vault"

**Actual Results**: _______________________

**Status**: [ ] Pass [ ] Fail [ ] N/A (no attachments or all saved)

---

### TC-022: Fact Extraction Dialog - Display
**Priority**: High
**Preconditions**: PRO user, facts extracted

**Steps**:
1. Extract facts from an email (as PRO user)
2. Check the dialog display

**Expected Results**:
- Modal overlay covers entire screen (fixed inset-0)
- Overlay: semi-transparent black (bg-black bg-opacity-50)
- Dialog centered (flex items-center justify-center)
- Dialog: white background, rounded corners (rounded-xl)
- Title: "Extracted Facts from Email"
- Description shows below title

**Actual Results**: _______________________

**Status**: [ ] Pass [ ] Fail [ ] N/A (not PRO or no facts)

---

### TC-023: Fact Extraction Dialog - Checkbox Interaction
**Priority**: High
**Preconditions**: Dialog is open with facts

**Steps**:
1. Open fact extraction dialog
2. Click checkboxes to toggle selection
3. Verify visual state

**Expected Results**:
- Each fact has a checkbox (default checked)
- Clicking checkbox toggles state
- Label has cursor-pointer
- Checkboxes are aligned with fact text

**Actual Results**: _______________________

**Status**: [ ] Pass [ ] Fail [ ] N/A (dialog not open)

---

### TC-024: Fact Extraction Dialog - Save Action
**Priority**: High
**Preconditions**: Dialog is open with facts

**Steps**:
1. Open fact extraction dialog
2. Uncheck some facts
3. Click "Save Selected Facts" button

**Expected Results**:
- Button shows count of selected facts
- Only checked facts are saved
- Success alert shows number of facts saved
- Dialog closes after saving
- Facts list is cleared

**Actual Results**: _______________________

**Status**: [ ] Pass [ ] Fail [ ] N/A (dialog not open)

---

### TC-025: Fact Extraction Dialog - Cancel Action
**Priority**: Medium
**Preconditions**: Dialog is open

**Steps**:
1. Open fact extraction dialog
2. Click "Cancel" button

**Expected Results**:
- Dialog closes
- Facts list is cleared
- No facts are saved
- Page returns to normal state

**Actual Results**: _______________________

**Status**: [ ] Pass [ ] Fail [ ] N/A (dialog not open)

---

### TC-026: Fact Extraction Dialog - Overlay Click
**Priority**: Medium
**Preconditions**: Dialog is open

**Steps**:
1. Open fact extraction dialog
2. Click outside the dialog (on the overlay)

**Expected Results**:
- Dialog closes
- Clicking on dialog content does NOT close it (stopPropagation)

**Actual Results**: _______________________

**Status**: [ ] Pass [ ] Fail [ ] N/A (dialog not open)

---

### TC-027: Loading State
**Priority**: High

**Steps**:
1. Navigate to an email detail page
2. Observe initial load
3. Refresh the page

**Expected Results**:
- Loading message displays: "Loading email..."
- Text: text-base text-gray-500
- Centered horizontally (text-center)
- Padding: px-12
- Brief display before content loads

**Actual Results**: _______________________

**Status**: [ ] Pass [ ] Fail

---

### TC-028: Error State
**Priority**: High

**Steps**:
1. Navigate to an invalid email ID (e.g., /app/emails/999999)
2. Observe error display

**Expected Results**:
- Error message displays in red (text-red-600)
- Message: "Email not found" or error details
- "Back to Inbox" button displays below error
- Button: blue background (bg-blue-600), white text
- Hover state: hover:bg-blue-700

**Actual Results**: _______________________

**Status**: [ ] Pass [ ] Fail

---

### TC-029: Responsive Design - Desktop (1920x1080)
**Priority**: High

**Steps**:
1. Open browser DevTools
2. Set viewport to 1920x1080
3. Navigate to email detail page
4. Check layout

**Expected Results**:
- Full width layout works correctly
- Max width container centers content (max-w-2xl mx-auto)
- All header buttons visible and accessible
- Email content readable with good line length
- No horizontal scrollbars

**Actual Results**: _______________________

**Status**: [ ] Pass [ ] Fail

---

### TC-030: Responsive Design - Tablet (768x1024)
**Priority**: High

**Steps**:
1. Open browser DevTools
2. Set viewport to 768x1024
3. Navigate to email detail page
4. Check layout

**Expected Results**:
- Layout adapts without horizontal scroll
- Header buttons remain accessible
- Email content remains readable
- Modal fits within viewport
- Touch targets are sufficient size

**Actual Results**: _______________________

**Status**: [ ] Pass [ ] Fail

---

### TC-031: Responsive Design - Mobile (375x667)
**Priority**: High

**Steps**:
1. Open browser DevTools
2. Set viewport to 375x667
3. Navigate to email detail page
4. Check layout

**Expected Results**:
- Single column layout
- Touch targets are sufficient size (min 44px)
- Modal uses w-[90%] for mobile
- Text remains readable at small sizes
- No horizontal scrolling
- Header buttons may wrap if needed

**Actual Results**: _______________________

**Status**: [ ] Pass [ ] Fail

---

### TC-032: Console Errors Check
**Priority**: Critical

**Steps**:
1. Open browser DevTools Console tab
2. Navigate to email detail page
3. Interact with all elements
4. Monitor console for errors

**Expected Results**:
- No CSS-related errors
- No missing class warnings
- No JavaScript errors
- No TypeScript errors
- All network requests succeed

**Actual Results**: _______________________

**Status**: [ ] Pass [ ] Fail

---

### TC-033: Accessibility - Keyboard Navigation
**Priority**: Medium

**Steps**:
1. Navigate to email detail page
2. Use Tab key to navigate
3. Use Enter/Space to activate buttons
4. Use Escape to close dialogs

**Expected Results**:
- Tab order is logical
- Focus indicators are visible
- Buttons activate with keyboard
- Escape closes dialogs
- No keyboard traps

**Actual Results**: _______________________

**Status**: [ ] Pass [ ] Fail

---

### TC-034: Accessibility - Screen Reader
**Priority**: Low (requires screen reader)

**Steps**:
1. Enable screen reader (NVDA, JAWS, VoiceOver)
2. Navigate to email detail page
3. Listen to element announcements

**Expected Results**:
- Buttons are announced as buttons
- States (disabled, active) are announced
- Form elements have proper labels
- Modal announcements are clear

**Actual Results**: _______________________

**Status**: [ ] Pass [ ] Fail [ ] N/A (no screen reader)

---

## Test Summary

### Test Execution
- **Total Test Cases**: 34
- **Passed**: _____
- **Failed**: _____
- **N/A**: _____
- **Pass Rate**: _____%

### Critical Issues Found
1.
2.
3.

### Non-Critical Issues Found
1.
2.
3.

### Overall Assessment
[ ] All tests passed - Ready for deployment
[ ] Minor issues found - Can be deployed with follow-up
[ ] Major issues found - Requires fixes before deployment

### Tester Notes



### Sign-off
**Tester**: _______________________
**Date**: _______________________
**Build/Commit**: _______________________

---

## Additional Notes

### Browser Compatibility
Tested on:
- [ ] Chrome/Edge (Chromium)
- [ ] Firefox
- [ ] Safari
- [ ] Mobile Safari (iOS)
- [ ] Mobile Chrome (Android)

### Performance Observations
- Page load time: _____
- Time to interactive: _____
- Any perceived lag or jank: _____

### Visual Regression
- [ ] Screenshots captured for comparison
- [ ] Visual regression test run (if available)

### Known Limitations
1. Email content styling uses CSS module for security
2. Some email client quirks may affect rendering
3. PRO features require active subscription
