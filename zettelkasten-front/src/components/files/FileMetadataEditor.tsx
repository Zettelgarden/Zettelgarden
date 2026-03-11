import React, { useState } from 'react';
import { File } from '../../models/File';
import { editFile, tagFile, untagFile } from '../../api/files';
import { FileTags } from './FileTags';
import { BacklinkInput } from '../cards/BacklinkInput';
import { defaultCard, PartialCard } from '../../models/Card';

interface FileMetadataEditorProps {
  file: File;
  onUpdate: () => void;
  onClose: () => void;
}

export function FileMetadataEditor({ file, onUpdate, onClose }: FileMetadataEditorProps) {
  const [description, setDescription] = useState(file.description || '');
  const [tags, setTags] = useState<string[]>(file.tags || []);
  const [saving, setSaving] = useState(false);

  const handleSaveDescription = async () => {
    setSaving(true);
    try {
      await editFile(file.id.toString(), { name: file.name, card_pk: file.card_pk, description });
      onUpdate();
    } catch (error) {
      console.error('Failed to save description:', error);
    } finally {
      setSaving(false);
    }
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
      onUpdate();
    } catch (error) {
      console.error('Failed to link card:', error);
    }
  };

  return (
    <div className="p-4 bg-white border border-gray-200 rounded-lg shadow-lg">
      <div className="flex justify-between items-center mb-4">
        <h3 className="text-lg font-semibold">Edit File Details</h3>
        <button onClick={onClose} className="text-gray-500 hover:text-gray-700 text-2xl">
          ×
        </button>
      </div>

      {/* Description */}
      <div className="mb-4">
        <label className="block text-sm font-medium text-gray-700 mb-1">Description</label>
        <textarea
          value={description}
          onChange={(e) => setDescription(e.target.value)}
          onBlur={handleSaveDescription}
          placeholder="Add notes about this file..."
          className="w-full p-2 border border-gray-300 rounded focus:outline-none focus:ring-2 focus:ring-blue-500"
          rows={3}
        />
      </div>

      {/* Tags */}
      <div className="mb-4">
        <label className="block text-sm font-medium text-gray-700 mb-1">Tags</label>
        <FileTags tags={tags} onAddTag={handleAddTag} onRemoveTag={handleRemoveTag} />
      </div>

      {/* Card Link */}
      <div className="mb-4">
        <label className="block text-sm font-medium text-gray-700 mb-1">Linked Card</label>
        {file.card_pk && file.card_pk > 0 ? (
          <div className="text-sm text-gray-600">Linked to card #{file.card_pk}</div>
        ) : (
          <BacklinkInput addBacklink={handleLinkCard} />
        )}
      </div>

      {saving && <div className="text-sm text-gray-500">Saving...</div>}
    </div>
  );
}
