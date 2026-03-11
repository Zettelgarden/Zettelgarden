import React, { useState, KeyboardEvent } from 'react';

interface FileTagsProps {
  tags: string[];
  onAddTag: (tagName: string) => void;
  onRemoveTag: (tagName: string) => void;
  editable?: boolean;
}

export function FileTags({ tags, onAddTag, onRemoveTag, editable = true }: FileTagsProps) {
  const [inputValue, setInputValue] = useState('');

  const handleKeyPress = (e: KeyboardEvent<HTMLInputElement>) => {
    if (e.key === 'Enter' && inputValue.trim()) {
      e.preventDefault();
      onAddTag(inputValue.trim());
      setInputValue('');
    }
  };

  return (
    <div className="flex flex-wrap items-center gap-2">
      {tags.map((tag) => (
        <span
          key={tag}
          className="inline-flex items-center gap-1 px-2 py-1 text-sm bg-blue-100 text-blue-800 rounded-full"
        >
          #{tag}
          {editable && (
            <button
              onClick={() => onRemoveTag(tag)}
              className="text-blue-600 hover:text-blue-800"
              aria-label={`Remove tag ${tag}`}
            >
              ×
            </button>
          )}
        </span>
      ))}
      {editable && (
        <input
          type="text"
          value={inputValue}
          onChange={(e) => setInputValue(e.target.value)}
          onKeyPress={handleKeyPress}
          placeholder="Add tag..."
          className="px-2 py-1 text-sm border border-gray-300 rounded focus:outline-none focus:ring-2 focus:ring-blue-500"
        />
      )}
    </div>
  );
}
