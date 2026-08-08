import { Card } from '../../models/Card';
import { File } from '../../models/File';
import { uploadFile } from '../../api/files';

interface UseDragDropOptions {
  editingCard: Card;
  setEditingCard: (card: Card) => void;
  setMessage: (message: string) => void;
  filesToUpdate: File[];
  setFilesToUpdate: (files: File[]) => void;
}

export function useDragDrop({
  editingCard,
  setEditingCard,
  setMessage,
  filesToUpdate,
  setFilesToUpdate,
}: UseDragDropOptions) {
  const handleDragOver = (event: React.DragEvent<HTMLTextAreaElement>) => {
    event.preventDefault();
  };

  const handleDrop = async (event: React.DragEvent<HTMLTextAreaElement>) => {
    event.preventDefault();
    if (event.dataTransfer.files && event.dataTransfer.files.length > 0) {
      for (let i = 0; i < event.dataTransfer.files.length; i++) {
        const file = event.dataTransfer.files[i];
        if (file.type.startsWith('image/')) {
          try {
            // Create a sanitized filename based on card title
            const sanitizedTitle = editingCard.title
              .replace(/[^a-zA-Z0-9]/g, '-') // Replace non-alphanumeric chars with dashes
              .replace(/-+/g, '-') // Replace multiple dashes with single dash
              .trim();

            // Add timestamp and index to ensure uniqueness
            const timestamp = new Date().getTime();
            const customFilename = `${sanitizedTitle}-${timestamp}-${i}`;

            const response = await uploadFile(
              file,
              editingCard.id,
              customFilename,
            );

            if ('error' in response) {
              setMessage('Error uploading file: ' + response['message']);
            } else {
              setMessage(
                'File uploaded successfully: ' + response['file']['name'],
              );
              setFilesToUpdate([...filesToUpdate, response.file]);
            }
          } catch (error) {
            setMessage('Error uploading file: ' + error);
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
        if (item.type.indexOf('image') !== -1) {
          event.preventDefault(); // Prevent default only for images
          const file = item.getAsFile();

          try {
            // Use title or default to "image" if title is blank
            const baseTitle = editingCard.title.trim() || 'image';

            // Create a sanitized filename based on card title
            const sanitizedTitle = baseTitle
              .replace(/[^a-zA-Z0-9]/g, '-') // Replace non-alphanumeric chars with dashes
              .replace(/-+/g, '-') // Replace multiple dashes with single dash
              .trim();

            // Add timestamp to ensure uniqueness
            const timestamp = new Date().getTime();
            const customFilename = `${sanitizedTitle}-${timestamp}`;

            const response = await uploadFile(
              file!,
              editingCard.id,
              customFilename,
            );

            if ('error' in response) {
              setMessage('Error uploading file: ' + response['message']);
            } else {
              setFilesToUpdate([...filesToUpdate, response.file]);
              let append_text = '\n\n![](' + response['file']['id'] + ')';
              setMessage(
                `File uploaded successfully: ${response['file']['name']}`,
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

  return {
    handleDragOver,
    handleDrop,
    handlePaste,
  };
}
