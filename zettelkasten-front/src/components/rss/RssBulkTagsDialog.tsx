import React from 'react';

interface RssBulkTagsDialogProps {
  isOpen: boolean;
  onClose: () => void;
  onConfirm: (tags: string, mode: 'replace' | 'append') => void;
  feedCount: number;
}

export function RssBulkTagsDialog({
  isOpen,
  onClose,
  onConfirm,
  feedCount,
}: RssBulkTagsDialogProps) {
  const [tags, setTags] = React.useState('');
  const [mode, setMode] = React.useState<'replace' | 'append'>('replace');

  const handleApply = () => {
    if (tags.trim()) {
      onConfirm(tags.trim(), mode);
    }
  };

  if (!isOpen) return null;

  return (
    <div className="fixed inset-0 bg-black/50 flex items-center justify-center z-50">
      <div className="bg-white rounded-lg p-6 w-[500px]">
        <h3 className="text-lg font-semibold mb-4">
          Set Tags for {feedCount} Feed{feedCount !== 1 ? 's' : ''}
        </h3>

        <div className="space-y-4">
          <div>
            <label
              htmlFor="tags"
              className="block text-sm font-medium text-gray-700 mb-1"
            >
              Tags
            </label>
            <input
              id="tags"
              type="text"
              placeholder="tech, news, ai (comma-separated)"
              value={tags}
              onChange={(e) => setTags(e.target.value)}
              className="w-full px-3 py-2 border border-gray-300 rounded-md focus:ring-2 focus:ring-blue-500 focus:border-blue-500"
            />
            <p className="text-sm text-gray-500 mt-1">
              Enter comma-separated tags. Spaces will be trimmed.
            </p>
          </div>

          <div>
            <label className="block text-sm font-medium text-gray-700 mb-2">
              Tag Mode
            </label>
            <div className="space-y-2">
              <div className="flex items-center space-x-2">
                <input
                  type="radio"
                  id="replace"
                  name="mode"
                  value="replace"
                  checked={mode === 'replace'}
                  onChange={(e) =>
                    setMode(e.target.value as 'replace' | 'append')
                  }
                  className="text-blue-600 focus:ring-blue-500"
                />
                <label htmlFor="replace" className="text-sm text-gray-700">
                  Replace existing tags
                </label>
              </div>
              <div className="flex items-center space-x-2">
                <input
                  type="radio"
                  id="append"
                  name="mode"
                  value="append"
                  checked={mode === 'append'}
                  onChange={(e) =>
                    setMode(e.target.value as 'replace' | 'append')
                  }
                  className="text-blue-600 focus:ring-blue-500"
                />
                <label htmlFor="append" className="text-sm text-gray-700">
                  Append to existing tags
                </label>
              </div>
            </div>
            <p className="text-sm text-gray-500 mt-1">
              {mode === 'replace'
                ? 'All existing tags will be removed and replaced with the new ones.'
                : 'New tags will be added to any existing tags, avoiding duplicates.'}
            </p>
          </div>
        </div>

        <div className="flex justify-end gap-2 mt-6 pt-4 border-t border-gray-200">
          <button
            onClick={onClose}
            className="px-4 py-2 bg-gray-200 text-gray-700 rounded-md hover:bg-gray-300"
          >
            Cancel
          </button>
          <button
            onClick={handleApply}
            disabled={!tags.trim()}
            className={`px-4 py-2 rounded-md ${
              tags.trim()
                ? 'bg-blue-600 text-white hover:bg-blue-700'
                : 'bg-gray-300 text-gray-500 cursor-not-allowed'
            }`}
          >
            Apply Tags
          </button>
        </div>
      </div>
    </div>
  );
}
