# Spreadsheet Creation Card Body Update

**Date:** 2026-02-11
**Status:** Design Approved

## Problem

When a user creates a spreadsheet via the SpreadsheetsTab, the spreadsheet is successfully created in the database but the card body never receives the `{{spreadsheet:ID}}` markdown reference. This means the newly created spreadsheet won't render in the card body unless the user manually types the reference.

## Solution

When a spreadsheet is created via SpreadsheetsTab, automatically append the markdown reference to the card body.

### Data Flow

1. User clicks "Add Spreadsheet" in SpreadsheetsTab
2. `createSpreadsheet()` API call creates the database record
3. Backend returns the new spreadsheet with its ID
4. SpreadsheetsTab appends `\n\n{{spreadsheet:ID}}\n` to the card body via `setViewCard`
5. CardBody re-renders and displays the new spreadsheet

### Implementation

**File to modify:** `zettelkasten-front/src/components/tabs/SpreadsheetsTab.tsx`

**Updated `handleCreateSpreadsheet` function:**

```typescript
const handleCreateSpreadsheet = async () => {
  try {
    setIsCreating(true);

    // Generate a unique name
    const existingNames = new Set(spreadsheets.map(s => s.name));
    let newName = 'sheet1';
    let counter = 1;
    while (existingNames.has(newName)) {
      counter++;
      newName = `sheet${counter}`;
    }

    const newSpreadsheet = await createSpreadsheet(viewingCard.id, newName);

    // Update local state
    setSpreadsheets(prev => [...prev, newSpreadsheet]);
    setSelectedSpreadsheet(newSpreadsheet);

    // Append spreadsheet reference to card body
    const updatedCard = {
      ...viewingCard,
      body: viewingCard.body.trim() + `\n\n{{spreadsheet:${newSpreadsheet.id}}}\n`
    };
    setViewCard(updatedCard);
  } catch (err) {
    if (err instanceof Error) {
      setError(err.message);
    } else {
      setError('Failed to create spreadsheet');
    }
  } finally {
    setIsCreating(false);
  }
};
```

### Edge Cases

| Scenario | Behavior |
|----------|----------|
| Empty card body | Results in just `{{spreadsheet:123}}` |
| Whitespace-only body | Trimmed, then `{{spreadsheet:123}}` appended |
| Existing content | Content + `\n\n{{spreadsheet:123}}\n` |
| API failure | `setViewCard` not called (transactional) |

### Testing

**Unit test addition:** `SpreadsheetsTab.test.tsx`

```typescript
it('should append spreadsheet reference to card body after creation', async () => {
  const mockCard: Card = {
    id: 1,
    body: '# My Card\nExisting content.',
    // ... other props
  };

  const newSpreadsheet: Spreadsheet = {
    id: 123,
    name: 'sheet1',
    // ... other props
  };

  vi.mocked(spreadsheetsApi.createSpreadsheet).mockResolvedValue(newSpreadsheet);

  render(<SpreadsheetsTab viewingCard={mockCard} setViewCard={mockSetViewCard} setError={vi.fn()} />);

  const addButton = screen.getByText('Add Spreadsheet');
  await userEvent.click(addButton);

  expect(mockSetViewCard).toHaveBeenCalledWith(
    expect.objectContaining({
      body: '# My Card\nExisting content.\n\n{{spreadsheet:123}}\n'
    })
  );
});
```

**Manual testing:**
1. Open a card with existing content
2. Go to Spreadsheets tab
3. Click "Add Spreadsheet"
4. Switch back to Content tab
5. Verify: The spreadsheet appears at the bottom of the card body

## Benefits

- New spreadsheets immediately visible in card body
- No manual markdown reference typing required
- Consistent user experience between SpreadsheetsTab and card body rendering

## Scope

**Single file change:** `SpreadsheetsTab.tsx` (~4 lines added)

**No backend changes required** - Backend already returns complete spreadsheet object

**Risk:** Low - Local React state update only

## Future Enhancements (Out of Scope)

- Checkbox option: "Add to card body: [✓]" for user control
- Multiple insertion location options (beginning, after heading, cursor position)
