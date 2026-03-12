import React, { useState, useEffect, useCallback } from 'react';
import { Link } from 'react-router-dom';
import { File } from '../../models/File';
import { editFile, tagFile, untagFile } from '../../api/files';
import { FileTags } from './FileTags';
import { BacklinkInput } from '../cards/BacklinkInput';
import { PartialCard } from '../../models/Card';
import { useToast } from '../toast/ToastContext';

const UNLINKED_CARD_PK = -1;
const SAVE_DEBOUNCE_MS = 500;

interface FileMetadataEditorProps {
  file: File;
  onUpdate: () => void;
  onClose: () => void;
}

export function FileMetadataEditor({ file, onUpdate, onClose }: FileMetadataEditorProps) {
  const { showToast } = useToast();
  const [description, setDescription] = useState(file.description || '');
  const [tags, setTags] = useState<string[]>(file.tags || []);
  const [saving, setSaving] = useState(false);
  const [cardPk, setCardPk] = useState(file.card_pk ?? UNLINKED_CARD_PK);
  const [linkingCard, setLinkingCard] = useState(false);
  const [tagOperation, setTagOperation] = useState<string | null>(null);

  // Sync state when file prop changes (e.g., after parent re-fetches)
  useEffect(() => {
    setDescription(file.description || '');
    setTags(file.tags || []);
    setCardPk(file.card_pk ?? UNLINKED_CARD_PK);
  }, [file.id, file.description, file.tags, file.card_pk]);

  // Auto-save description with debounce
  const saveDescription = useCallback(async (newDescription: string) => {
    setSaving(true);
    try {
      await editFile(file.id.toString(), { 
        name: file.name, 
        card_pk: cardPk, 
        description: newDescription || undefined 
      });
      onUpdate();
    } catch (error) {
      console.error('Failed to save description:', error);
      showToast('error', 'Save Failed', 'Could not save description');
    } finally {
      setSaving(false);
    }
  }, [file.id, file.name, cardPk, onUpdate, showToast]);

  // Debounce description changes
  useEffect(() => {
    const timeoutId = setTimeout(() => {
      // Only save if description differs from original
      const originalDescription = file.description || '';
      if (description !== originalDescription) {
        saveDescription(description);
      }
    }, SAVE_DEBOUNCE_MS);

    return () => clearTimeout(timeoutId);
  }, [description, file.description, saveDescription]);

  const handleDescriptionChange = (value: string) => {
    setDescription(value);
  };

  const handleAddTag = async (tagName: string) => {
    setTagOperation(`adding:${tagName}`);
    try {
      await tagFile(file.id, [tagName]);
      setTags([...tags, tagName]);
      showToast('success', 'Tag Added', `Added tag: ${tagName}`);
      onUpdate();
    } catch (error) {
      console.error('Failed to add tag:', error);
      showToast('error', 'Tag Failed', `Could not add tag: ${tagName}`);
    } finally {
      setTagOperation(null);
    }
  };

  const handleRemoveTag = async (tagName: string) => {
    setTagOperation(`removing:${tagName}`);
    try {
      await untagFile(file.id, tagName);
      setTags(tags.filter((t) => t !== tagName));
      showToast('success', 'Tag Removed', `Removed tag: ${tagName}`);
      onUpdate();
    } catch (error) {
      console.error('Failed to remove tag:', error);
      showToast('error', 'Tag Failed', `Could not remove tag: ${tagName}`);
    } finally {
      setTagOperation(null);
    }
  };

  const handleLinkCard = async (card: PartialCard) => {
    setLinkingCard(true);
    try {
      await editFile(file.id.toString(), { name: file.name, card_pk: card.id });
      setCardPk(card.id);
      showToast('success', 'Card Linked', `Linked to card ${card.card_id}`);
      onUpdate();
    } catch (error) {
      console.error('Failed to link card:', error);
      showToast('error', 'Link Failed', 'Could not link card');
    } finally {
      setLinkingCard(false);
    }
  };

  const handleUnlinkCard = async () => {
    setLinkingCard(true);
    try {
      await editFile(file.id.toString(), { name: file.name, card_pk: UNLINKED_CARD_PK });
      setCardPk(UNLINKED_CARD_PK);
      showToast('success', 'Card Unlinked', 'File unlinked from card');
      onUpdate();
    } catch (error) {
      console.error('Failed to unlink card:', error);
      showToast('error', 'Unlink Failed', 'Could not unlink card');
    } finally {
      setLinkingCard(false);
    }
  };

  const handleClose = () => {
    onClose();
  };

  // Derive linked state from local cardPk (which syncs with props via useEffect)
  const isLinked = cardPk > 0 && file.card;

  return (
    <div className="p-4">
      <div className="flex justify-between items-center mb-4">
        <div className="min-w-0 flex-1 mr-2">
          <h3 className="text-lg font-semibold">Edit File Details</h3>
          <p className="text-sm text-gray-500 truncate" title={file.name}>{file.name}</p>
        </div>
        <button onClick={handleClose} className="text-gray-400 hover:text-gray-600 text-2xl leading-none flex-shrink-0">
          ×
        </button>
      </div>

      {/* Description */}
      <div className="mb-4">
        <label className="block text-sm font-medium text-gray-700 mb-1">
          Description
          {saving && <span className="ml-2 text-xs text-gray-400">Saving...</span>}
        </label>
        <textarea
          value={description}
          onChange={(e) => handleDescriptionChange(e.target.value)}
          placeholder="Add notes about this file..."
          className="w-full p-2 border border-gray-300 rounded focus:outline-none focus:ring-2 focus:ring-blue-500"
          rows={3}
        />
      </div>

      {/* Tags */}
      <div className="mb-4">
        <label className="block text-sm font-medium text-gray-700 mb-1">Tags</label>
        <div className="relative">
          {tagOperation && (
            <div className="absolute inset-0 bg-white bg-opacity-75 flex items-center justify-center rounded">
              <span className="text-sm text-gray-500">Updating...</span>
            </div>
          )}
          <FileTags
            tags={tags}
            onAddTag={handleAddTag}
            onRemoveTag={handleRemoveTag}
            editable={!tagOperation}
          />
        </div>
      </div>

      {/* Card Link */}
      <div className="mb-4">
        <label className="block text-sm font-medium text-gray-700 mb-1">Linked Card</label>
        {isLinked ? (
          <div className="flex items-center gap-2">
            <Link
              to={`/app/card/${file.card.id}`}
              className="text-sm text-blue-600 hover:text-blue-700 bg-blue-50 px-2 py-1 rounded"
              onClick={onClose}
            >
              [{file.card.card_id}] {file.card.title || 'Untitled'}
            </Link>
            <button
              type="button"
              onClick={handleUnlinkCard}
              disabled={linkingCard}
              className="text-xs text-gray-500 hover:text-red-600 underline disabled:opacity-50"
            >
              {linkingCard ? 'Unlinking...' : 'Unlink'}
            </button>
          </div>
        ) : (
          <div className="relative">
            {linkingCard && (
              <div className="absolute inset-0 bg-white bg-opacity-75 flex items-center justify-center rounded">
                <span className="text-sm text-gray-500">Linking...</span>
              </div>
            )}
            <BacklinkInput addBacklink={handleLinkCard} />
          </div>
        )}
      </div>

      {/* Footer */}
      <div className="flex justify-end items-center pt-3 border-t">
        <button
          type="button"
          onClick={handleClose}
          className="px-4 py-2 text-sm font-medium text-gray-700 bg-gray-100 hover:bg-gray-200 rounded-md"
        >
          Done
        </button>
      </div>
    </div>
  );
}
