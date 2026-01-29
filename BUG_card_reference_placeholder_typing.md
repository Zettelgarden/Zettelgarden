# Bug Report: Card Reference Popover Blocks Empty Bracket Typing

**Bug ID**: ZETTG-2025-001
**Reported Date**: 2026-01-29
**Status**: Open
**Severity**: Medium
**Priority**: P2
**Component**: Card Editor - Inline References

---

## Executive Summary

Users cannot type `[ ]` (empty brackets) as placeholders in the card body editor because the inline card reference dialog immediately captures input focus upon typing `[`. This breaks a common workflow where users want to mark sections for later completion.

**User Impact**: Medium - Users work around this but experience friction during normal typing workflows.

---

## Problem Statement

When a user types the `[` character in the card body editor, the inline card reference autocomplete dialog appears immediately and captures keyboard input. This prevents users from creating empty brackets `[ ]` as placeholders or future references. The dialog has no intuitive dismissal mechanism for users who want to continue typing without selecting a card.

### Current Behavior
1. User types `[` in card body editor
2. Card reference dialog appears immediately and captures focus
3. User cannot type `]` to complete empty brackets
4. User must either:
   - Select a card (unwanted behavior)
   - Press Escape (undiscoverable)
   - Click away (disrupts typing flow)
   - Use backspace to delete and manually workaround

### Expected Behavior
Users should be able to type `[ ]` naturally without being forced into card selection mode. The reference dialog should enhance workflow, not interrupt it.

---

## Affected Components

| Component | File | Lines | Notes |
|-----------|------|-------|-------|
| Inline Card Reference Dialog | `zettelkasten-front/src/components/cards/InlineCardReferenceDialog.tsx` | 135-140 | Escape handling exists but undiscoverable |
| Card Reference Hook | `zettelkasten-front/src/components/cards/useCardReference.ts` | - | Manages trigger logic |
| Card Body Text Area | `zettelkasten-front/src/components/cards/CardBodyTextArea.tsx` | - | Input handler |

---

## User Stories

### Primary User Story
**As a** knowledge worker using Zettelgarden
**I want to** type empty brackets `[ ]` as placeholders in my notes
**So that** I can mark sections to fill in later without interrupting my typing flow

### Acceptance Criteria
- [ ] Typing `[` followed immediately by `]` creates empty brackets without showing dialog
- [ ] Dialog does not appear when typing `[ ]` quickly
- [ ] Users can still trigger dialog intentionally (e.g., typing `[` + pause, or `[` + character)
- [ ] Escape key dismisses dialog (already implemented, ensure documentation)
- [ ] Clicking outside dialog dismisses it (if not already implemented)

### Secondary User Story
**As a** new Zettelgarden user
**I want to** have an intuitive way to dismiss the card reference dialog
**So that** I don't feel trapped when I accidentally trigger it

### Acceptance Criteria
- [ ] Dialog dismissal is discoverable (visible hint or obvious interaction)
- [ ] Dialog dismissal doesn't lose cursor position
- [ ] Dialog can be dismissed without selecting a card

---

## Steps to Reproduce

1. Navigate to any card edit page
2. Click in the card body text area
3. Type `[`
4. **Observe**: Card reference dialog appears immediately
5. Type `]`
6. **Observe**: Character is captured by dialog filter, not inserted into text
7. **Result**: Cannot create `[ ]` without pressing Escape first (undiscoverable)

**Environment**:
- OS: Any
- Browser: Any
- Zettelgarden Version: Current (development build)

---

## Potential Solutions

### Solution 1: Smart Dismiss on Closing Bracket (Recommended)
**Implementation**: Detect when user types `]` as the next character after `[`

```typescript
// In useCardReference or CardBodyTextArea
const handleBracketInput = (char: string) => {
  if (char === ']' && lastChar === '[' && dialogVisible) {
    // User typed [] quickly - dismiss dialog and insert []
    closeDialog();
    insertText('[]');
    moveCursorAfterBrackets();
  }
}
```

**Pros**:
- Intuitive - matches user expectation
- Minimal behavior change
- Preserves fast workflow

**Cons**:
- Requires careful timing logic
- Edge cases with rapid typing

**Estimate**: 2-4 hours

---

### Solution 2: Delayed Trigger
**Implementation**: Only show dialog after a delay (e.g., 300ms) or after typing additional characters

```typescript
// Debounce dialog appearance
const showDialogAfterDelay = useRef<number>();

const triggerDialog = () => {
  clearTimeout(showDialogAfterDelay.current);
  showDialogAfterDelay.current = setTimeout(() => {
    setShowDialog(true);
  }, 300);
}
```

**Pros**:
- Allows quick `[ ]` typing naturally
- Users who pause see dialog (helpful)

**Cons**:
- Perceived lag for legitimate reference creation
- Delay might feel sluggish

**Estimate**: 2-3 hours

---

### Solution 3: Explicit Activation Only
**Implementation**: Remove auto-trigger on `[`, require explicit keyboard shortcut (e.g., Ctrl+Space)

**Pros**:
- Never interrupts typing
- Explicit user intent

**Cons**:
- Breaks existing user workflow
- Loses discoverability of feature
- Higher interaction cost for common case

**Estimate**: 1-2 hours (but high UX cost)

---

### Solution 4: Improved Visual Dismissal Hint (Complementary)
**Implementation**: Add visible hint showing Escape key dismisses dialog

```typescript
// In InlineCardReferenceDialog
<div className="dialog-hint">
  Press <kbd>Escape</kbd> to close
</div>
```

**Pros**:
- Makes existing Escape key discoverable
- Low implementation effort
- Complements other solutions

**Cons**:
- Doesn't solve the root problem
- Adds UI clutter

**Estimate**: 1 hour

---

## Recommended Approach

**Primary Solution**: Implement Solution 1 (Smart Dismiss on Closing Bracket)
- Natural typing behavior preserved
- Minimal code changes
- Backward compatible

**Secondary Enhancement**: Add Solution 4 (Visual Hint) temporarily
- Makes Escape discoverable until Solution 1 is implemented
- Can be removed once Solution 1 is proven effective

**Out of Scope**: Solutions 2 and 3 introduce UX issues or breaking changes

---

## Dependencies

| Dependency | Type | Status |
|------------|------|--------|
| `useCardReference.ts` refactoring | Internal | Required for Solution 1 |
| `CardBodyTextArea.tsx` input handling | Internal | Required for Solution 1 |
| UI/UX review of dialog hint | External | Recommended for Solution 4 |

---

## Risk Assessment

| Risk | Likelihood | Impact | Mitigation |
|------|------------|--------|------------|
| Solution 1 timing edge cases | Medium | Low | Add unit tests for typing patterns |
| Breaking existing user workflows | Low | Medium | Beta test with power users |
| Performance regression from delay logic | Low | Low | Profile debounce implementation |

---

## Success Metrics

### Definition of Done
- [ ] Unit tests added for bracket typing patterns
- [ ] Manual testing confirms `[ ]` typing works
- [ ] Card reference dialog still functions normally for actual references
- [ ] No regression in existing card link creation workflow
- [ ] Browser compatibility verified (Chrome, Firefox, Safari, Edge)

### Success Criteria
- Users can type `[ ]` without seeing dialog (measured via manual testing)
- Time to create empty brackets < 1 second
- Card reference creation time unchanged (≤ 200ms difference)
- Zero bugs reported related to card linking in follow-up week

---

## Related Issues

- **Blocks**: None
- **Blocked By**: None
- **Duplicates**: None known
- **References**: Similar UX patterns in Notion, Obsidian (both allow empty brackets)

---

## Open Questions

1. Should the delay timing be configurable? (No, hardcode 300ms for consistency)
2. Do we want to track how often users create empty brackets vs actual references? (Yes, future analytics consideration)
3. Should Solution 4 be permanent or temporary? (Temporary - remove after Solution 1 proves effective)

---

## Implementation Notes

**Technical Considerations**:
- Use React `useRef` to track last typed character
- Leverage existing `handleKeyDown` in `InlineCardReferenceDialog.tsx`
- Ensure cursor position management is precise
- Test with international keyboard layouts

**Testing Strategy**:
- Unit tests for input pattern detection
- Integration tests for dialog open/close behavior
- Manual E2E testing with various typing speeds
- Accessibility testing with screen readers

---

## Changelog Entry

```
### Fixed
- Card reference dialog now allows typing empty brackets [ ] without interruption
- Added smart dismissal when closing bracket is typed immediately after opening bracket
- Improved discoverability of Escape key to dismiss reference dialog
```

---

## Reporter Notes

This bug was identified during normal usage when attempting to create placeholder brackets in notes. The current behavior forces a context switch that disrupts the "flow state" of writing. The recommended Solution 1 maintains the quick reference workflow while supporting the placeholder use case.

---

**Last Updated**: 2026-01-29
**Assigned To**: Unassigned
**Sprint**: TBD
**Story Points**: TBD (awaiting developer estimation)