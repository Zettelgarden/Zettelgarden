import React, { useState, useRef, forwardRef, useImperativeHandle } from "react";
import ReactMarkdown from 'react-markdown';
import remarkGfm from 'remark-gfm';
import { PartialCard, Card } from "../../models/Card";
import { File } from "../../models/File";
import { InlineCardReferenceDialog } from "./InlineCardReferenceDialog";
import { useTextFormatting } from "./useTextFormatting";
import { useDragDrop } from "./useDragDrop";
import { useCardReference } from "./useCardReference";
import { useListEditing } from "./useListEditing";

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

  // Use extracted hooks
  const { formatText } = useTextFormatting({ textareaRef, editingCard, setEditingCard });
  const { handleDragOver, handleDrop, handlePaste } = useDragDrop({ editingCard, setEditingCard, setMessage, filesToUpdate, setFilesToUpdate });
  const { showReferenceDialog, dialogPosition, handleReferenceSelect, handleBracketKey, handleCloseReferenceDialog } = useCardReference({ editingCard, textareaRef, setEditingCard });
  const { handleListKeyDown } = useListEditing({ editingCard, textareaRef, setEditingCard });


  const handleKeyDown = (event: React.KeyboardEvent<HTMLTextAreaElement>) => {
    // Handle ESC key to close reference dialog
    if (event.key === 'Escape' && showReferenceDialog) {
      event.preventDefault();
      handleCloseReferenceDialog();
      return;
    }

    // Handle card reference bracket activation
    handleBracketKey(event);

    // Handle list editing (Tab/Enter for bullet lists)
    handleListKeyDown(event);
  };

  function handleBodyChange(event: React.ChangeEvent<HTMLTextAreaElement>) {
    setEditingCard({ ...editingCard, body: event.target.value });
  }


  // Expose methods to parent component
  useImperativeHandle(ref, () => ({
    formatText,
    togglePreviewMode: () => {
      setIsPreviewMode(prev => !prev);
    }
  }));

  return (
    <div className="relative">
      {isPreviewMode ? (
        <div className="prose sm:text-sm block w-full min-h-[150px] max-h-[50vh] sm:max-h-[60vh] lg:h-48 p-2 overflow-y-auto">
          <ReactMarkdown remarkPlugins={[remarkGfm]}>
            {editingCard.body}
          </ReactMarkdown>
        </div>
      ) : (
        <textarea
          ref={textareaRef}
          className="block w-full min-h-[200px] max-h-[50vh] sm:max-h-[60vh] lg:min-h-[384px] p-2 border border-gray-200 sm:text-sm resize-y"
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
      {showReferenceDialog && (
        <InlineCardReferenceDialog
          position={dialogPosition}
          onSelect={handleReferenceSelect}
          onClose={handleCloseReferenceDialog}
          excludeCardId={editingCard.id}
        />
      )}
    </div>
  );
});
