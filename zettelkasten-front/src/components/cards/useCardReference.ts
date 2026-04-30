import { useState } from "react";
import { RefObject } from "react";
import { PartialCard, Card } from "../../models/Card";
import { getCaretCoordinates } from "../../utils/cursor";

interface UseCardReferenceOptions {
  editingCard: Card;
  textareaRef: RefObject<HTMLTextAreaElement>;
  setEditingCard: (card: Card) => void;
}

interface DialogPosition {
  top: number;
  left: number;
  height: number;
}

export function useCardReference({
  editingCard,
  textareaRef,
  setEditingCard
}: UseCardReferenceOptions) {
  const [showReferenceDialog, setShowReferenceDialog] = useState(false);
  const [dialogPosition, setDialogPosition] = useState<DialogPosition>({ top: 0, left: 0, height: 0 });
  // The position of the FIRST [ in the [[ trigger sequence
  const [triggerIndex, setTriggerIndex] = useState<number | null>(null);

  const handleReferenceSelect = (card: PartialCard) => {
    const textarea = textareaRef.current;
    if (!textarea || triggerIndex === null || !card) return;

    const value = textarea.value;
    // Replace the [[ with [[card_id]] — the [[ is already in the textarea
    const newText = "[[" + card.card_id + "]]";

    // triggerIndex points to the first [, so replace from there through the second ]
    const before = value.substring(0, triggerIndex);
    const after = value.substring(triggerIndex + 2); // skip the two [ characters
    const newBody = before + newText + after;

    setEditingCard({ ...editingCard, body: newBody });
    setShowReferenceDialog(false);
    setTriggerIndex(null);

    // Restore focus and cursor after the closing ]]
    setTimeout(() => {
      textarea.focus();
      const newCursorPos = triggerIndex + newText.length;
      textarea.setSelectionRange(newCursorPos, newCursorPos);
    }, 0);
  };

  const handleBracketKey = (event: React.KeyboardEvent<HTMLTextAreaElement>) => {
    if (event.key !== '[') return;

    const textarea = event.currentTarget;
    const { selectionStart } = textarea;

    // Check if the character before the cursor (before this newly-typed [) is also [
    // The [ hasn't been inserted into the value yet at keydown time
    const charBefore = selectionStart > 0 ? textarea.value[selectionStart - 1] : '';

    if (charBefore === '[') {
      // This is the second [, forming [[ — show the reference dialog
      const caret = getCaretCoordinates(textarea, selectionStart);
      const textareaRect = textarea.getBoundingClientRect();

      const viewportCaret = {
        top: caret.top + textareaRect.top + window.scrollY,
        left: caret.left + textareaRect.left + window.scrollX,
        height: caret.height
      };

      setDialogPosition(viewportCaret);
      // triggerIndex is the position of the first [, which is selectionStart - 1
      setTriggerIndex(selectionStart - 1);
      setShowReferenceDialog(true);
      // We do NOT prevent default, allowing the second [ to be typed
    }
  };

  const handleCloseReferenceDialog = () => {
    setShowReferenceDialog(false);
    // Remove the second [ that was typed, leaving just the first one
    if (triggerIndex !== null && textareaRef.current) {
      const value = textareaRef.current.value;
      // Remove the character at triggerIndex + 1 (the second [)
      const newBody = value.substring(0, triggerIndex + 1) + value.substring(triggerIndex + 2);
      setEditingCard({ ...editingCard, body: newBody });
      setTriggerIndex(null);

      setTimeout(() => {
        textareaRef.current?.focus();
        const pos = triggerIndex + 1;
        textareaRef.current?.setSelectionRange(pos, pos);
      }, 0);
    } else {
      setTriggerIndex(null);
      textareaRef.current?.focus();
    }
  };

  return {
    showReferenceDialog,
    dialogPosition,
    handleReferenceSelect,
    handleBracketKey,
    handleCloseReferenceDialog
  };
}
