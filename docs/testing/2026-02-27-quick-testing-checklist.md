# EmailDetailPage Quick Testing Checklist

**Date**: 2026-02-27
**Server**: http://localhost:5176/
**Component**: EmailDetailPage (Tailwind Refactor)

## Quick Start

1. **Open browser**: http://localhost:5176/
2. **Login** to your account
3. **Navigate to Emails** section
4. **Click on an email** to open detail page

## Visual Checks (5 minutes)

### Header Buttons
- [ ] Back button (← Back to Inbox) - white background, gray border
- [ ] Archive button (📁) - white when unarchived, yellow when archived
- [ ] Convert button - white if not converted, green if converted
- [ ] Create Task button (✚) - white background
- [ ] Extract Facts button - white for PRO, yellow for non-PRO

### Email Content
- [ ] Subject heading - large, bold, dark gray
- [ ] From/To/Date fields - small gray labels, larger gray values
- [ ] Status badges - colored backgrounds (yellow/green/gray)
- [ ] Email body - properly formatted with good spacing
- [ ] Attachments (if any) - cards with thumbnails and buttons

### States
- [ ] Hover states work on all buttons (background color change)
- [ ] Disabled states show correctly (gray, opacity reduced)
- [ ] Loading state displays briefly
- [ ] Error state shows red error message

## Interactive Tests (5 minutes)

### Button Actions
- [ ] Back button navigates to email list
- [ ] Archive button changes state (↱ Unarchive)
- [ ] Convert button opens dialog
- [ ] Create Task opens task dialog
- [ ] Extract Facts shows alert (non-PRO) or extracts (PRO)

### Attachments (if available)
- [ ] Download button works
- [ ] Save to Vault button works and disappears after saving
- [ ] Attachment cards have hover effect

### Dialogs
- [ ] Email convert dialog displays correctly
- [ ] Create task dialog displays correctly
- [ ] Fact extraction dialog displays correctly (PRO)
- [ ] Dialogs close on overlay click
- [ ] Dialogs stay open when clicking content

## Responsive Tests (3 minutes)

### Desktop (1920x1080)
- [ ] Layout looks correct
- [ ] Content centered with max width
- [ ] No horizontal scroll

### Tablet (768x1024)
- [ ] Layout adapts correctly
- [ ] All buttons accessible
- [ ] Content readable

### Mobile (375x667)
- [ ] Single column layout
- [ ] Buttons may wrap
- [ ] Text remains readable
- [ ] No horizontal scroll

## Console Check (1 minute)

### Browser DevTools
- [ ] Open Console tab (F12)
- [ ] No red errors
- [ ] No missing class warnings
- [ ] No CSS errors

## Final Decision

### All Checks Passed?
- ✅ **Yes**: Create verification commit and deploy
- ❌ **No**: Document issues and fix before proceeding

---

**Total Time**: ~15 minutes
**Test Cases**: 34 detailed cases available in manual test guide
**Automated Checks**: 19/19 passed

For detailed testing instructions, see:
- `docs/testing/2026-02-27-email-detail-page-manual-test-guide.md`

For verification report, see:
- `docs/testing/2026-02-27-email-detail-page-tailwind-verification.md`
