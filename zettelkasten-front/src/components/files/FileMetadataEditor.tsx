import React, { useState } from 'react';
import { Link } from 'react-router-dom';
import { File } from '../../models/File';
import { editFile, tagFile, untagFile } from '../../api/files';
import { FileTags } from './FileTags';
import { BacklinkInput } from '../cards/BacklinkInput';
import { PartialCard } from '../../models/Card';

interface FileMetadataEditorProps {
  file: File;
  onUpdate: () => void;
  onClose: () => void;
}

export function FileMetadataEditor({ file, onUpdate, onClose }: FileMetadataEditorProps) {
  const [description, setDescription] = useState(file.description || '');
  const [tags, setTags] = useState<string[]>(file.tags || []);
  const [saving, setSaving] = useState(false);
  const [cardPk, setCardPk] = useState(file.card_pk);
  const [hasUnsavedChanges, setHasUnsavedChanges] = useState(false);

  const handleSaveDescription = async () => {
    if (!hasUnsavedChanges) return;
    setSaving(true);
    try {
      await editFile(file.id.toString(), { name: file.name, card_pk: cardPk, description });
      setHasUnsavedChanges(false);
      onUpdate();
    } catch (error) {
      console.error('Failed to save description:', error);
    } finally {
      setSaving(false);
    }
  };

  const handleDescriptionChange = (value: string) => {
    setDescription(value);
    setHasUnsavedChanges(true);
  };

  const handleAddTag = async (tagName: string) => {
    try {
      await tagFile(file.id, [tagName]);
      setTags([...tags, tagName]);
      onUpdate();
    } catch (error) {
      console.error('Failed to add tag:', error);
    }
  };

  const handleRemoveTag = async (tagName: string) => {
    try {
      await untagFile(file.id, tagName);
      setTags(tags.filter((t) => t !== tagName));
      onUpdate();
    } catch (error) {
      console.error('Failed to remove tag:', error);
    }
  };

  const handleLinkCard = async (card: PartialCard) => {
    try {
      await editFile(file.id.toString(), { name: file.name, card_pk: card.id });
      setCardPk(card.id);
      onUpdate();
    } catch (error) {
      console.error('Failed to link card:', error);
    }
  };

  const handleUnlinkCard = async () => {
    try {
      await editFile(file.id.toString(), { name: file.name, card_pk: -1 });
      setCardPk(-1);
      onUpdate();
    } catch (error) {
      console.error('Failed to unlink card:', error);
    }
  };

  const handleClose = () => {
    if (hasUnsavedChanges) {
      if (!window.confirm('You have unsaved changes. Discard them?')) {
        return;
      }
    }
    onClose();
  };

  const isLinked = cardPk && cardPk > 0 && file.card;

  return (
    <div className="p-4">
      <div className="flex justify-between items-center mb-4">
        <div>
          <h3 className="text-lg font-semibold">Edit File Details</h3>
          <p className="text-sm text-gray-500 truncate max-w-[400px]">{file.name}</p>
        </div>
        <button onClick={handleClose} className="text-gray-400 hover:text-gray-600 text-2xl leading-none">
          ×
        </button>
      </div>

      {/* Description */}
      <div className="mb-4">
        <label className="block text-sm font-medium text-gray-700 mb-1">Description</label>
        <textarea
          value={description}
          onChange={(e) => handleDescriptionChange(e.target.value)}
          placeholder="Add notes about this file..."
          className="w-full p-2 border border-gray-300 rounded focus:outline-none focus:ring-2 focus:ring-blue-500"
          rows={3}
        />
        {hasUnsavedChanges && (
          <button
            onClick={handleSaveDescription}
            disabled={saving}
            className="mt-2 px-3 py-1.5 text-sm font-medium text-white bg-blue-600 hover:bg-blue-700 rounded-md disabled:opacity-50"
          >
            {saving ? 'Saving...' : 'Save Description'}
          </button>
        )}
      </div>

      {/* Tags */}
      <div className="mb-4">
        <label className="block text-sm font-medium text-gray-700 mb-1">Tags</label>
        <FileTags tags={tags} onAddTag={handleAddTag} onRemoveTag={handleRemoveTag} />
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
              onClick={handleUnlinkCard}
              className="text-xs text-gray-500 hover:text-red-600 underline"
            >
              Unlink
            </button>
          </div>
        ) : (
          <BacklinkInput addBacklink={handleLinkCard} />
        )}
      </div>

      {/* Footer */}
      <div className="flex justify-between items-center pt-3 border-t">
        <span className="text-sm text-gray-500">
          {hasUnsavedChanges ? 'Unsaved changes' : ''}
        </span>
        <button
          onClick={handleClose}
          className="px-4 py-2 text-sm font-medium text-gray-700 bg-gray-100 hover:bg-gray-200 rounded-md"
        >
          Done
        </button>
      </div>
    </div>
  );
}
