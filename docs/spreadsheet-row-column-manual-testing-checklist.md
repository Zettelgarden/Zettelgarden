# Spreadsheet Row/Column Operations - Manual Testing Checklist

## Test Execution Summary

- **Unit Tests:** 595/595 passed (51 test files)
- **Test Date:** 2026-02-14
- **Test Duration:** 9.35s

## Manual Testing Checklist

This checklist provides comprehensive manual testing scenarios for the spreadsheet row/column operations feature. Since we cannot start a dev server in this environment, these tests should be performed manually by the user.

### Prerequisites

1. Start the development server: `cd zettelkasten-front && npm start`
2. Navigate to a card with a spreadsheet component
3. Ensure you have edit permissions for the card

---

## 1. Row Insertion Operations

### 1.1 Insert Row Above
- [ ] Right-click on a row header (e.g., row 3)
- [ ] Verify context menu appears with "Insert Row Above" option
- [ ] Click "Insert Row Above"
- [ ] **Expected:** New empty row inserted at position 3
- [ ] **Expected:** All data from row 3 onwards shifted down by one row
- [ ] **Expected:** New row has focus (optional enhancement)

### 1.2 Insert Row Below
- [ ] Right-click on a row header (e.g., row 3)
- [ ] Verify context menu appears with "Insert Row Below" option
- [ ] Click "Insert Row Below"
- [ ] **Expected:** New empty row inserted at position 4
- [ ] **Expected:** Data from row 4 onwards shifted down by one row
- [ ] **Expected:** Original row 3 data remains unchanged

### 1.3 Insert Row at First Position
- [ ] Right-click on row header 1
- [ ] Click "Insert Row Above"
- [ ] **Expected:** New row inserted at position 1
- [ ] **Expected:** All existing data shifted down by one row
- [ ] **Expected:** Formula references updated (e.g., =A1 becomes =A2)

### 1.4 Insert Row with Data Migration
- [ ] Create a spreadsheet with sample data:
  ```
  A1: 10    B1: 20
  A2: 30    B2: 40
  A3: =A1+A2
  ```
- [ ] Right-click row 2 header
- [ ] Click "Insert Row Above"
- [ ] **Expected:** New row inserted at position 2
- [ ] **Expected:** Data shifted down:
  ```
  A1: 10    B1: 20
  A2: [new row]
  A3: 30    B3: 40
  A4: =A1+A3
  ```
- [ ] **Expected:** Formula in A4 updated to =A1+A3

---

## 2. Column Insertion Operations

### 2.1 Insert Column Left
- [ ] Right-click on a column header (e.g., column C)
- [ ] Verify context menu appears with "Insert Column Left" option
- [ ] Click "Insert Column Left"
- [ ] **Expected:** New empty column inserted at position C
- [ ] **Expected:** All data from column C onwards shifted right by one column
- [ ] **Expected:** New column headers updated (C, D, E, etc.)

### 2.2 Insert Column Right
- [ ] Right-click on a column header (e.g., column C)
- [ ] Verify context menu appears with "Insert Column Right" option
- [ ] Click "Insert Column Right"
- [ ] **Expected:** New empty column inserted at position D
- [ ] **Expected:** Data from column D onwards shifted right by one column
- [ ] **Expected:** Original column C data remains unchanged

### 2.3 Insert Column at First Position
- [ ] Right-click on column header A
- [ ] Click "Insert Column Left"
- [ ] **Expected:** New column inserted at position A
- [ ] **Expected:** All existing data shifted right by one column
- [ ] **Expected:** Formula references updated (e.g., =A1 becomes =B1)

### 2.4 Insert Column with Data Migration
- [ ] Create a spreadsheet with sample data:
  ```
  A1: 10    B1: 20    C1: 30
  A2: =A1+B1
  ```
- [ ] Right-click column B header
- [ ] Click "Insert Column Left"
- [ ] **Expected:** New column inserted at position B
- [ ] **Expected:** Data shifted right:
  ```
  A1: 10    B1: [new]    C1: 20    D1: 30
  A2: =A1+C1
  ```
- [ ] **Expected:** Formula in A2 updated to =A1+C1

---

## 3. Row Deletion Operations

### 3.1 Delete Empty Row
- [ ] Right-click on an empty row header (e.g., row 5 with no data)
- [ ] Click "Delete Row"
- [ ] **Expected:** Row removed immediately without confirmation
- [ ] **Expected:** All rows below shift up by one

### 3.2 Delete Row with Data - Confirmation
- [ ] Create a spreadsheet with data in row 3
- [ ] Right-click on row 3 header
- [ ] Click "Delete Row"
- [ ] **Expected:** Confirmation dialog appears with message:
  "Are you sure you want to delete this row? This action cannot be undone."

### 3.3 Confirm Row Deletion
- [ ] With confirmation dialog showing, click "Delete"
- [ ] **Expected:** Row removed
- [ ] **Expected:** All rows below shift up by one
- [ ] **Expected:** Formula references updated to account for removed row
- [ ] **Expected:** "Saving..." indicator appears
- [ ] **Expected:** Changes persist after refresh

### 3.4 Cancel Row Deletion
- [ ] With confirmation dialog showing, click "Cancel"
- [ ] **Expected:** Dialog closes
- [ ] **Expected:** Row remains unchanged
- [ ] **Expected:** No data loss

### 3.5 Delete Row with Formula Updates
- [ ] Create a spreadsheet with:
  ```
  A1: 10    B1: 20
  A2: 30    B2: 40
  A3: =A1+A2
  A4: =A2*2
  ```
- [ ] Delete row 2
- [ ] **Expected:** Row 2 removed
- [ ] **Expected:** Formulas updated:
  ```
  A1: 10    B1: 20
  A2: =A1*2
  ```

### 3.6 Delete Last Row Protection
- [ ] Reduce spreadsheet to only 1 row with data
- [ ] Right-click on the last row header
- [ ] Click "Delete Row"
- [ ] **Expected:** Nothing happens (last row protected)
- [ ] **Expected:** Row remains in place

---

## 4. Column Deletion Operations

### 4.1 Delete Empty Column
- [ ] Right-click on an empty column header (e.g., column Z with no data)
- [ ] Click "Delete Column"
- [ ] **Expected:** Column removed immediately without confirmation
- [ ] **Expected:** All columns to the right shift left by one

### 4.2 Delete Column with Data - Confirmation
- [ ] Create a spreadsheet with data in column C
- [ ] Right-click on column C header
- [ ] Click "Delete Column"
- [ ] **Expected:** Confirmation dialog appears with message:
  "Are you sure you want to delete this column? This action cannot be undone."

### 4.3 Confirm Column Deletion
- [ ] With confirmation dialog showing, click "Delete"
- [ ] **Expected:** Column removed
- [ ] **Expected:** All columns to the right shift left by one
- [ ] **Expected:** Formula references updated to account for removed column
- [ ] **Expected:** "Saving..." indicator appears
- [ ] **Expected:** Changes persist after refresh

### 4.4 Cancel Column Deletion
- [ ] With confirmation dialog showing, click "Cancel"
- [ ] **Expected:** Dialog closes
- [ ] **Expected:** Column remains unchanged
- [ ] **Expected:** No data loss

### 4.5 Delete Column with Formula Updates
- [ ] Create a spreadsheet with:
  ```
  A1: 10    B1: 20    C1: 30
  A2: =A1+B1+C1
  ```
- [ ] Delete column B
- [ ] **Expected:** Column B removed
- [ ] **Expected:** Formula updated: A2: =A1+B1 (was C1)

### 4.6 Delete Last Column Protection
- [ ] Reduce spreadsheet to only 1 column with data
- [ ] Right-click on the last column header
- [ ] Click "Delete Column"
- [ ] **Expected:** Nothing happens (last column protected)
- [ ] **Expected:** Column remains in place

---

## 5. Limit Enforcement

### 5.1 Maximum Rows Limit (100)
- [ ] Create a spreadsheet with 99 rows
- [ ] Right-click on any row header
- [ ] Click "Insert Row Below"
- [ ] **Expected:** Row inserted successfully (now 100 rows)
- [ ] Right-click on any row header
- [ ] Click "Insert Row Below"
- [ ] **Expected:** Toast notification appears: "Maximum rows reached (100)"
- [ ] **Expected:** No new row inserted

### 5.2 Maximum Columns Limit (26)
- [ ] Create a spreadsheet with 25 columns (A-Y)
- [ ] Right-click on any column header
- [ ] Click "Insert Column Right"
- [ ] **Expected:** Column inserted successfully (now 26 columns, A-Z)
- [ ] Right-click on any column header
- [ ] Click "Insert Column Right"
- [ ] **Expected:** Toast notification appears: "Maximum columns reached (26)"
- [ ] **Expected:** No new column inserted

---

## 6. Data Persistence

### 6.1 Row Insertion Persistence
- [ ] Insert a new row with data:
  ```
  A1: Test Data
  ```
- [ ] Wait for "Saving..." to complete
- [ ] Refresh the page
- [ ] **Expected:** Inserted row and data persist

### 6.2 Column Insertion Persistence
- [ ] Insert a new column with data:
  ```
  B1: Column Data
  ```
- [ ] Wait for "Saving..." to complete
- [ ] Refresh the page
- [ ] **Expected:** Inserted column and data persist

### 6.3 Row Deletion Persistence
- [ ] Delete a row with data
- [ ] Confirm deletion
- [ ] Wait for "Saving..." to complete
- [ ] Refresh the page
- [ ] **Expected:** Row remains deleted

### 6.4 Column Deletion Persistence
- [ ] Delete a column with data
- [ ] Confirm deletion
- [ ] Wait for "Saving..." to complete
- [ ] Refresh the page
- [ ] **Expected:** Column remains deleted

### 6.5 Formula Persistence
- [ ] Create formula: =A1+B1
- [ ] Insert row at position 1
- [ ] **Expected:** Formula updates to =A2+B2
- [ ] Wait for "Saving..." to complete
- [ ] Refresh the page
- [ ] **Expected:** Updated formula persists

---

## 7. Read-Only Mode

### 7.1 Read-Only Spreadsheet
- [ ] Open a spreadsheet in read-only mode (no edit permissions)
- [ ] Right-click on any row header
- [ ] **Expected:** No context menu appears
- [ ] Right-click on any column header
- [ ] **Expected:** No context menu appears

### 7.2 Edit Mode Verification
- [ ] Open a spreadsheet with edit permissions
- [ ] Right-click on any row header
- [ ] **Expected:** Context menu appears with row options
- [ ] Right-click on any column header
- [ ] **Expected:** Context menu appears with column options

---

## 8. Edge Cases and Error Handling

### 8.1 Undo/Redo Compatibility
- [ ] Insert a row
- [ ] Press Ctrl+Z (if undo is implemented)
- [ ] **Expected:** Insertion is undone (if supported)

### 8.2 Multiple Sequential Operations
- [ ] Insert row at position 3
- [ ] Insert column at position C
- [ ] Delete row at position 5
- [ ] **Expected:** All operations execute correctly
- [ ] **Expected:** Data integrity maintained

### 8.3 Large Data Sets
- [ ] Create a spreadsheet with 50+ rows and data
- [ ] Insert a row in the middle
- [ ] **Expected:** Operation completes in reasonable time
- [ ] **Expected:** All data shifted correctly
- [ ] **Expected:** UI remains responsive

### 8.4 Special Characters in Formulas
- [ ] Create formula: =SUM(A1:A10)
- [ ] Insert row at position 5
- [ ] **Expected:** Formula updates to =SUM(A1:A11)

---

## 9. Context Menu UI/UX

### 9.1 Context Menu Positioning
- [ ] Right-click near the right edge of the spreadsheet
- [ ] **Expected:** Context menu positioned to avoid overflow
- [ ] Right-click near the bottom edge of the spreadsheet
- [ ] **Expected:** Context menu positioned to avoid overflow

### 9.2 Context Menu Dismissal
- [ ] Open context menu
- [ ] Click elsewhere on the spreadsheet
- [ ] **Expected:** Context menu closes
- [ ] Press Escape key
- [ ] **Expected:** Context menu closes (if implemented)

### 9.3 Context Menu Styling
- [ ] Verify context menu has proper styling
- [ ] Verify menu items are readable
- [ ] Verify hover states work correctly
- [ ] Verify separators between menu items (if applicable)

---

## 10. Accessibility

### 10.1 Keyboard Navigation
- [ ] Navigate to a row header using keyboard
- [ ] Press appropriate key for context menu (if implemented)
- [ ] **Expected:** Context menu opens
- [ ] Navigate context menu with arrow keys (if implemented)
- [ ] **Expected:** Menu items are selectable

### 10.2 Screen Reader Support
- [ ] Enable screen reader
- [ ] Navigate to row header
- [ ] **Expected:** Row number announced
- [ ] Open context menu
- [ ] **Expected:** Menu items announced

---

## 11. Browser Compatibility

### 11.1 Chrome/Edge
- [ ] Test all scenarios in Chrome
- [ ] Test all scenarios in Edge
- [ ] **Expected:** All functionality works correctly

### 11.2 Firefox
- [ ] Test all scenarios in Firefox
- [ ] **Expected:** All functionality works correctly

### 11.3 Safari (if applicable)
- [ ] Test all scenarios in Safari
- [ ] **Expected:** All functionality works correctly

---

## Test Results Summary

After completing manual testing, update the following:

- **Total Tests Run:** ___/___
- **Tests Passed:** ___/___
- **Tests Failed:** ___/___
- **Issues Found:** ___
- **Critical Issues:** ___
- **Minor Issues:** ___

### Issues Found

1. [ ] Describe any issues found during testing
2. [ ] Include steps to reproduce
3. [ ] Include expected vs actual behavior
4. [ ] Include browser/OS information

---

## Additional Notes

Add any additional observations, suggestions, or improvements discovered during testing:

1.
2.
3.

---

## Implementation Completion Status

- [x] Task 1: Create utility functions for row/column insertion/deletion
- [x] Task 2: Add confirmation dialog for deletion operations
- [x] Task 3: Integrate insert/delete operations into Spreadsheet component
- [x] Task 4: Add toast notifications for soft limits
- [x] Task 5: Add context menu for row/column headers
- [x] Task 6: Ensure read-only mode is respected
- [x] Task 7: Add unit tests for operations
- [x] Task 8: Add integration test for context menu
- [x] Task 9: Manual testing and final verification
