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
  const [triggerIndex, setTriggerIndex] = useState<number | null>(null);

  const handleReferenceSelect = (card: PartialCard) => {
    const textarea = textareaRef.current;
    if (!textarea || triggerIndex === null || !card) return;

    // Use textarea.value instead of editingCard.body because editingCard.body might be stale
    // (React state update from the '[' insertion might not have propagated yet to this callback closure)
    const value = textarea.value;
    const newText = "[" + card.card_id + "]";

    // Insert at triggerIndex
    const newBody = value.substring(0, triggerIndex) + newText + value.substring(triggerIndex);

    setEditingCard({ ...editingCard, body: newBody });
    setShowReferenceDialog(false);
    setTriggerIndex(null);

    // Restore focus and cursor
    setTimeout(() => {
      textarea.focus();
      const newCursorPos = triggerIndex + newText.length;
      textarea.setSelectionRange(newCursorPos, newCursorPos);
    }, 0);
  };

  const handleBracketKey = (event: React.KeyboardEvent<HTMLTextAreaElement>) => {
    const { selectionStart } = event.currentTarget;

    if (event.key === '[') {
      const caret = getCaretCoordinates(event.currentTarget, selectionStart);
      const textareaRect = event.currentTarget.getBoundingClientRect();

      // Convert container-relative coordinates to viewport-relative coordinates
      // The dialog uses position: fixed, which expects viewport coordinates
      const viewportCaret = {
        top: caret.top + textareaRect.top + window.scrollY,
        left: caret.left + textareaRect.left + window.scrollX,
        height: caret.height
      };

      setDialogPosition(viewportCaret);
      setTriggerIndex(selectionStart); // +1 because [ will be inserted
      setShowReferenceDialog(true);
      // We do NOT prevent default, allowing [ to be typed
    }
  };

  const handleCloseReferenceDialog = () => {
    setShowReferenceDialog(false);
    setTriggerIndex(null);
    textareaRef.current?.focus();
  };

  return {
    showReferenceDialog,
    dialogPosition,
    handleReferenceSelect,
    handleBracketKey,
    handleCloseReferenceDialog
  };
}