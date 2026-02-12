# Spreadsheet Creation Card Body Update - Implementation Plan

**Date:** 2026-02-11
**Status:** Ready for Implementation
**Design:** [2026-02-11-spreadsheet-creation-card-body-update-design.md](./2026-02-11-spreadsheet-creation-card-body-update-design.md)

## Overview

Fix the gap where creating a spreadsheet via SpreadsheetsTab doesn't add the markdown reference to the card body, causing the spreadsheet to not render in the card view.

## Tasks

### Task 1: Update SpreadsheetsTab to Append Reference

**File:** `zettelkasten-front/src/components/tabs/SpreadsheetsTab.tsx`

**Change:** Modify `handleCreateSpreadsheet` function to append `{{spreadsheet:ID}}` to card body after successful creation.

```typescript
// After line 61 (newSpreadsheet creation), add:
// Append spreadsheet reference to card body
const updatedCard = {
  ...viewingCard,
  body: viewingCard.body.trim() + `\n\n{{spreadsheet:${newSpreadsheet.id}}}\n`
};
setViewCard(updatedCard);
```

**Steps:**
1. Read the current `handleCreateSpreadsheet` function
2. Add the card body update logic after `setSelectedSpreadsheet(newSpreadsheet)`
3. Ensure proper trimming and newline formatting

### Task 2: Add Unit Test

**File:** `zettelkasten-front/src/components/tabs/SpreadsheetsTab.test.tsx`

**Change:** Add a new test case verifying card body is updated after spreadsheet creation.

**Steps:**
1. Add new `describe` block for `handleCreateSpreadsheet` behavior
2. Test case: should append spreadsheet reference to card body after creation
3. Mock `createSpreadsheet` to return a spreadsheet with known ID
4. Assert `setViewCard` was called with updated body containing `{{spreadsheet:ID}}`

### Task 3: Run Tests and Verify

**Steps:**
1. Run unit tests: `cd zettelkasten-front && npm test -- SpreadsheetsTab.test.tsx`
2. Verify new test passes
3. Verify no existing tests are broken

### Task 4: Manual Testing

**Steps:**
1. Start frontend dev server
2. Open a card with existing markdown content
3. Navigate to Spreadsheets tab
4. Click "Add Spreadsheet"
5. Navigate back to Content tab
6. Verify spreadsheet renders at bottom of card
7. Test edge cases:
   - Empty card body
   - Card with only whitespace
   - Creating multiple spreadsheets in sequence

## Files Modified

| File | Change |
|------|--------|
| `zettelkasten-front/src/components/tabs/SpreadsheetsTab.tsx` | Add card body update logic |
| `zettelkasten-front/src/components/tabs/SpreadsheetsTab.test.tsx` | Add unit test |

## Definition of Done

- [ ] Card body is updated with `{{spreadsheet:ID}}` reference after creation
- [ ] Unit test added and passing
- [ ] Manual testing confirms spreadsheet renders in card body
- [ ] No existing tests broken
- [ ] Changes committed to git

## Risk Assessment

**Risk Level:** Low

- Single file modification
- No backend changes required
- Isolated to SpreadsheetsTab component
- Easy to rollback if issues arise

## Estimated Effort

- Task 1: 5 minutes (simple state update)
- Task 2: 10 minutes (unit test)
- Task 3: 5 minutes (run tests)
- Task 4: 10 minutes (manual testing)

**Total:** ~30 minutes
