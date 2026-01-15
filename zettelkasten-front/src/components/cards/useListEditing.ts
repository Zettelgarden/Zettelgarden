import { RefObject } from "react";
import { Card } from "../../models/Card";

interface UseListEditingOptions {
  editingCard: Card;
  textareaRef: RefObject<HTMLTextAreaElement>;
  setEditingCard: (card: Card) => void;
}

export function useListEditing({
  editingCard,
  textareaRef,
  setEditingCard
}: UseListEditingOptions) {
  const handleListKeyDown = (event: React.KeyboardEvent<HTMLTextAreaElement>) => {
    const textarea = event.currentTarget;
    const { value, selectionStart, selectionEnd } = textarea;

    if (event.key === 'Tab') {
      // Find the start and end of the affected lines
      const lineStart = value.lastIndexOf('\n', selectionStart - 1) + 1;
      const lineEnd = value.indexOf('\n', selectionEnd);
      const actualLineEnd = lineEnd === -1 ? value.length : lineEnd;

      // Get all selected lines
      const selectedLines = value.substring(lineStart, actualLineEnd);
      const lines = selectedLines.split('\n');

      // Check if any line starts with a bullet (- or *)
      const hasBullets = lines.some(line => {
        const trimmed = line.trim();
        return trimmed.startsWith('-') || trimmed.startsWith('*');
      });

      // Only handle Tab if we're on bullet list lines
      if (!hasBullets) {
        return; // Allow default Tab behavior
      }

      event.preventDefault();

      const beforeSelection = value.substring(0, lineStart);
      const afterSelection = value.substring(actualLineEnd);

      let newLines: string[];
      let cursorOffset = 0;

      if (event.shiftKey) {
        // Shift+Tab: Unindent (remove up to 2 spaces)
        newLines = lines.map(line => {
          // Only unindent lines that have bullets or are already indented
          if (line.match(/^\s*[-*]/) || line.match(/^\s+/)) {
            // Remove up to 2 spaces from the beginning
            if (line.startsWith('  ')) {
              return line.substring(2);
            } else if (line.startsWith(' ')) {
              return line.substring(1);
            }
          }
          return line;
        });

        // Calculate cursor offset (negative for unindent)
        const originalLength = selectedLines.length;
        const newLength = newLines.join('\n').length;
        cursorOffset = newLength - originalLength;
      } else {
        // Tab: Indent (add 2 spaces)
        newLines = lines.map(line => {
          // Only indent lines that have content or bullets
          if (line.trim().length > 0) {
            return '  ' + line;
          }
          return line;
        });

        // Calculate cursor offset (positive for indent)
        const addedSpaces = newLines.filter(line => line.trim().length > 0).length * 2;
        cursorOffset = addedSpaces;
      }

      const newBody = beforeSelection + newLines.join('\n') + afterSelection;
      setEditingCard({ ...editingCard, body: newBody });

      setTimeout(() => {
        const newStart = selectionStart + (lineStart === selectionStart ? (event.shiftKey ? Math.max(cursorOffset, -2) : 2) : 0);
        const newEnd = selectionEnd + cursorOffset;
        textarea.setSelectionRange(newStart, newEnd);
      }, 0);

      return;
    }

    if (event.key === 'Enter') {
      // Find the start of the current line
      const currentLineStart = value.lastIndexOf('\n', selectionStart - 1) + 1;
      const currentLine = value.substring(currentLineStart, selectionStart);

      // Check if the current line starts with a bullet point (possibly with indentation)
      const bulletMatch = currentLine.match(/^(\s*)-\s+/);

      if (bulletMatch) {
        event.preventDefault();

        // Extract the indentation and bullet format
        const indentation = bulletMatch[1] || '';
        const bulletFormat = '- ';

        // Case 1: Empty bullet -> remove bullet and just insert newline (exit list)
        if (currentLine.trim() === '-') {
          const newText = `\n`;

          const newBody =
            value.substring(0, currentLineStart) +
            newText +
            value.substring(selectionStart);

          setEditingCard({ ...editingCard, body: newBody });

          setTimeout(() => {
            const cursorPos = currentLineStart + newText.length;
            textarea.setSelectionRange(cursorPos, cursorPos);
          }, 0);
        } else {
          // Case 2: Non-empty bullet -> continue list with new bullet
          const newText = `\n${indentation}${bulletFormat}`;

          const newBody =
            value.substring(0, selectionStart) +
            newText +
            value.substring(selectionStart);

          setEditingCard({ ...editingCard, body: newBody });

          setTimeout(() => {
            textarea.setSelectionRange(
              selectionStart + newText.length,
              selectionStart + newText.length
            );
          }, 0);
        }
      }
    }
  };

  return { handleListKeyDown };
}