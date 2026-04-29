import { RefObject } from "react";
import { Card } from "../../models/Card";

interface UseTextFormattingOptions {
  textareaRef: RefObject<HTMLTextAreaElement>;
  editingCard: Card;
  setEditingCard: (card: Card) => void;
}

export function useTextFormatting({
  textareaRef,
  editingCard,
  setEditingCard
}: UseTextFormattingOptions) {
  const formatText = (formatType: string) => {
    const textarea = textareaRef.current;
    if (!textarea) return;

    const start = textarea.selectionStart;
    const end = textarea.selectionEnd;
    const selectedText = editingCard.body.substring(start, end);
    const currentLineStart = editingCard.body.lastIndexOf('\n', start - 1) + 1;
    const currentLineEnd = editingCard.body.indexOf('\n', end);
    const currentLine = editingCard.body.substring(
      currentLineStart,
      currentLineEnd === -1 ? editingCard.body.length : currentLineEnd
    );

    let formattedText = selectedText;
    let newCursorStart = start;
    let newCursorEnd = end;
    let newBody = editingCard.body;

    switch (formatType) {
      case 'bold':
        if (selectedText) {
          formattedText = `**${selectedText}**`;
          newBody =
            editingCard.body.substring(0, start) +
            formattedText +
            editingCard.body.substring(end);
          newCursorStart = start + 2;
          newCursorEnd = end + 2;
        }
        break;

      case 'italic':
        if (selectedText) {
          formattedText = `*${selectedText}*`;
          newBody =
            editingCard.body.substring(0, start) +
            formattedText +
            editingCard.body.substring(end);
          newCursorStart = start + 1;
          newCursorEnd = end + 1;
        }
        break;

      case 'strikethrough':
        if (selectedText) {
          formattedText = `~~${selectedText}~~`;
          newBody =
            editingCard.body.substring(0, start) +
            formattedText +
            editingCard.body.substring(end);
          newCursorStart = start + 2;
          newCursorEnd = end + 2;
        }
        break;

      case 'h1':
        if (currentLine.trim() === selectedText.trim()) {
          // If the selected text is the entire line, prepend with heading
          formattedText = `# ${selectedText}`;
          newBody =
            editingCard.body.substring(0, currentLineStart) +
            formattedText +
            editingCard.body.substring(end);
          newCursorStart = start + 2;
          newCursorEnd = end + 2;
        } else if (selectedText) {
          // Add a newline and heading before selection
          formattedText = `\n# ${selectedText}`;
          newBody =
            editingCard.body.substring(0, start) +
            formattedText +
            editingCard.body.substring(end);
          newCursorStart = start + 3;
          newCursorEnd = end + 3;
        }
        break;

      case 'h2':
        if (currentLine.trim() === selectedText.trim()) {
          formattedText = `## ${selectedText}`;
          newBody =
            editingCard.body.substring(0, currentLineStart) +
            formattedText +
            editingCard.body.substring(end);
          newCursorStart = start + 3;
          newCursorEnd = end + 3;
        } else if (selectedText) {
          formattedText = `\n## ${selectedText}`;
          newBody =
            editingCard.body.substring(0, start) +
            formattedText +
            editingCard.body.substring(end);
          newCursorStart = start + 4;
          newCursorEnd = end + 4;
        }
        break;

      case 'h3':
        if (currentLine.trim() === selectedText.trim()) {
          formattedText = `### ${selectedText}`;
          newBody =
            editingCard.body.substring(0, currentLineStart) +
            formattedText +
            editingCard.body.substring(end);
          newCursorStart = start + 4;
          newCursorEnd = end + 4;
        } else if (selectedText) {
          formattedText = `\n### ${selectedText}`;
          newBody =
            editingCard.body.substring(0, start) +
            formattedText +
            editingCard.body.substring(end);
          newCursorStart = start + 5;
          newCursorEnd = end + 5;
        }
        break;

      case 'bulletList':
        if (selectedText.includes('\n')) {
          // Multi-line selection: add bullet to each line
          const lines = selectedText.split('\n');
          formattedText = lines.map(line => `- ${line}`).join('\n');
          newBody =
            editingCard.body.substring(0, start) +
            formattedText +
            editingCard.body.substring(end);
          newCursorStart = start + 2;
          newCursorEnd = start + formattedText.length;
        } else if (selectedText) {
          formattedText = `- ${selectedText}`;
          newBody =
            editingCard.body.substring(0, start) +
            formattedText +
            editingCard.body.substring(end);
          newCursorStart = start + 2;
          newCursorEnd = end + 2;
        }
        break;

      case 'numberList':
        if (selectedText.includes('\n')) {
          // Multi-line selection: add numbers to each line
          const lines = selectedText.split('\n');
          formattedText = lines.map((line, index) => `${index + 1}. ${line}`).join('\n');
          newBody =
            editingCard.body.substring(0, start) +
            formattedText +
            editingCard.body.substring(end);
          newCursorStart = start + 3; // "1. " is 3 characters
          newCursorEnd = start + formattedText.length;
        } else if (selectedText) {
          formattedText = `1. ${selectedText}`;
          newBody =
            editingCard.body.substring(0, start) +
            formattedText +
            editingCard.body.substring(end);
          newCursorStart = start + 3;
          newCursorEnd = end + 3;
        }
        break;

      case 'code':
        if (selectedText.includes('\n')) {
          // Multi-line selection: add code block
          formattedText = "```\n" + selectedText + "\n```";
          newBody =
            editingCard.body.substring(0, start) +
            formattedText +
            editingCard.body.substring(end);
          newCursorStart = start + 4; // "```\n" is 4 characters
          newCursorEnd = end + 4;
        } else if (selectedText) {
          // Single line: inline code
          formattedText = "`" + selectedText + "`";
          newBody =
            editingCard.body.substring(0, start) +
            formattedText +
            editingCard.body.substring(end);
          newCursorStart = start + 1;
          newCursorEnd = end + 1;
        }
        break;

      case 'quote':
        if (selectedText.includes('\n')) {
          // Multi-line selection: add quote to each line
          const lines = selectedText.split('\n');
          formattedText = lines.map(line => `> ${line}`).join('\n');
          newBody =
            editingCard.body.substring(0, start) +
            formattedText +
            editingCard.body.substring(end);
          newCursorStart = start + 2;
          newCursorEnd = start + formattedText.length;
        } else if (selectedText) {
          formattedText = `> ${selectedText}`;
          newBody =
            editingCard.body.substring(0, start) +
            formattedText +
            editingCard.body.substring(end);
          newCursorStart = start + 2;
          newCursorEnd = end + 2;
        }
        break;

      case 'table':
        // Insert a basic 3-column table with one data row
        formattedText = `|  |  |  |\n|---|---|---|\n|  |  |  |`;
        // Add newline before table if not at beginning of line
        const beforeTable = start > 0 && editingCard.body[start - 1] !== '\n' ? '\n' : '';
        // Add newline after table if not at end of content
        const afterTable = end < editingCard.body.length && editingCard.body[end] !== '\n' ? '\n' : '';
        formattedText = beforeTable + formattedText + afterTable;

        newBody =
          editingCard.body.substring(0, start) +
          formattedText +
          editingCard.body.substring(end);
        // Adjust cursor position based on whether we added a newline
        newCursorStart = start + (beforeTable ? 3 : 2); // Place cursor in first header cell
        newCursorEnd = start + (beforeTable ? 3 : 2);
        break;

      default:
        return;
    }

    // Update the body and cursor position if we have either selected text or if it's a structural format (like table)
    if (selectedText || formatType === 'table') {
      setEditingCard({ ...editingCard, body: newBody });

      // Re-focus and set selection to maintain cursor position after the formatting
      setTimeout(() => {
        textarea.focus();
        textarea.setSelectionRange(
          newCursorStart,
          newCursorEnd
        );
      }, 0);
    }
  };

  return { formatText };
}