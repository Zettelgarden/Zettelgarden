import React, { DragEventHandler, useState, useRef, forwardRef, useImperativeHandle } from "react";
import ReactMarkdown from 'react-markdown';
import remarkGfm from 'remark-gfm';
import { PartialCard, Card } from "../../models/Card";
import { File } from "../../models/File";
import { uploadFile } from "../../api/files";

interface CardBodyTextAreaProps {
  editingCard: Card;
  setEditingCard: (card: Card) => void;
  setMessage: (message: string) => void;
  newCard: boolean;
  filesToUpdate: File[];
  setFilesToUpdate: (files: File[]) => void;
}

export interface CardBodyTextAreaHandle {
  formatText: (formatType: string) => void;
  togglePreviewMode: () => void;
}

export const CardBodyTextArea = forwardRef<CardBodyTextAreaHandle, CardBodyTextAreaProps>(({
  editingCard,
  setEditingCard,
  setMessage,
  newCard,
  filesToUpdate,
  setFilesToUpdate,
}: CardBodyTextAreaProps, ref) => {
  const textareaRef = useRef<HTMLTextAreaElement>(null);
  const [isPreviewMode, setIsPreviewMode] = useState<boolean>(false);

  const handleKeyDown = (event: React.KeyboardEvent<HTMLTextAreaElement>) => {
    const textarea = event.currentTarget;
    const { value, selectionStart, selectionEnd } = textarea;

    // Handle Tab and Shift+Tab for indentation on bullet lists
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

  function handleBodyChange(event: React.ChangeEvent<HTMLTextAreaElement>) {
    setEditingCard({ ...editingCard, body: event.target.value });
  }

  const handleDragOver = (event: React.DragEvent<HTMLTextAreaElement>) => {
    event.preventDefault();
  };

  const handleDrop = async (event: React.DragEvent<HTMLTextAreaElement>) => {
    event.preventDefault();
    if (event.dataTransfer.files && event.dataTransfer.files.length > 0) {
      for (let i = 0; i < event.dataTransfer.files.length; i++) {
        const file = event.dataTransfer.files[i];
        if (file.type.startsWith("image/")) {
          try {
            // Create a sanitized filename based on card title
            const sanitizedTitle = editingCard.title
              .replace(/[^a-zA-Z0-9]/g, '-') // Replace non-alphanumeric chars with dashes
              .replace(/-+/g, '-') // Replace multiple dashes with single dash
              .trim();

            // Add timestamp and index to ensure uniqueness
            const timestamp = new Date().getTime();
            const customFilename = `${sanitizedTitle}-${timestamp}-${i}`;

            const response = await uploadFile(file, editingCard.id, customFilename);

            if ("error" in response) {
              setMessage("Error uploading file: " + response["message"]);
            } else {
              setMessage(
                "File uploaded successfully: " + response["file"]["name"],
              );
              setFilesToUpdate([...filesToUpdate, response.file]);
            }
          } catch (error) {
            setMessage("Error uploading file: " + error);
          }
        }
      }
    }
  };

  const handlePaste = async (
    event: React.ClipboardEvent<HTMLTextAreaElement>,
  ) => {
    if (event.clipboardData && event.clipboardData.items) {
      const items = Array.from(event.clipboardData.items);
      for (const item of items) {
        if (item.type.indexOf("image") !== -1) {
          event.preventDefault(); // Prevent default only for images
          const file = item.getAsFile();

          try {
            // Use title or default to "image" if title is blank
            const baseTitle = editingCard.title.trim() || "image";

            // Create a sanitized filename based on card title
            const sanitizedTitle = baseTitle
              .replace(/[^a-zA-Z0-9]/g, '-') // Replace non-alphanumeric chars with dashes
              .replace(/-+/g, '-') // Replace multiple dashes with single dash
              .trim();

            // Add timestamp to ensure uniqueness
            const timestamp = new Date().getTime();
            const customFilename = `${sanitizedTitle}-${timestamp}`;

            const response = await uploadFile(file!, editingCard.id, customFilename);

            if ("error" in response) {
              setMessage("Error uploading file: " + response["message"]);
            } else {
              setFilesToUpdate([...filesToUpdate, response.file]);
              let append_text = "\n\n![](" + response["file"]["id"] + ")";
              setMessage(
                `File uploaded successfully: ${response["file"]["name"]}`,
              );

              let prevEditingCard = {
                ...editingCard,
                body: editingCard.body + append_text,
              };
              setEditingCard(prevEditingCard);
            }
          } catch (error) {
            setMessage(`Error uploading file: ${error}`);
          }
        }
      }
    }
  };

  // Format text function for the markdown toolbar
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

      default:
        return;
    }

    if (selectedText) {
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

  // Expose methods to parent component
  useImperativeHandle(ref, () => ({
    formatText,
    togglePreviewMode: () => {
      setIsPreviewMode(prev => !prev);
    }
  }));

  return (
    <div>
      {isPreviewMode ? (
        <div className="prose block w-full h-48 p-2 overflow-y-auto">
          <ReactMarkdown remarkPlugins={[remarkGfm]}>
            {editingCard.body}
          </ReactMarkdown>
        </div>
      ) : (
        <textarea
          ref={textareaRef}
          className="block w-full h-96 p-2 border border-gray-200"
          id="body"
          value={editingCard.body}
          onChange={handleBodyChange}
          onKeyDown={handleKeyDown}
          onDrop={handleDrop}
          onDragOver={handleDragOver}
          onPaste={handlePaste}
          placeholder="Body"
        />
      )}
    </div>
  );
});
