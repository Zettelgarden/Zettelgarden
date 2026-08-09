import React, { useState } from 'react';
import { Modal } from '../ui/Modal';
import { Spinner } from '../ui/Spinner';
import { createFolder, CreateRSSFolderParams, RSSFolder } from '../../api/rss';

interface RssCreateFolderDialogProps {
  isOpen: boolean;
  onClose: () => void;
  onFolderCreated: (folder: RSSFolder) => void;
}

export function RssCreateFolderDialog({
  isOpen,
  onClose,
  onFolderCreated,
}: RssCreateFolderDialogProps) {
  const [name, setName] = useState('');
  const [loading, setLoading] = useState(false);
  const [error, setError] = useState<string>('');

  const handleSubmit = async (e: React.FormEvent) => {
    e.preventDefault();

    if (!name.trim()) {
      setError('Folder name is required');
      return;
    }

    setLoading(true);
    setError('');

    try {
      const folderParams: CreateRSSFolderParams = {
        name: name.trim(),
      };

      const newFolder = await createFolder(folderParams);
      onFolderCreated(newFolder);
      handleClose();
    } catch (err) {
      console.error('Failed to create folder:', err);
      setError(
        err instanceof Error
          ? err.message
          : 'Failed to create folder. Please try again.',
      );
    } finally {
      setLoading(false);
    }
  };

  const handleClose = () => {
    setName('');
    setError('');
    onClose();
  };

  return (
    <Modal
      open={isOpen}
      onClose={handleClose}
      size="md"
      dialogClassName="z-[80]"
    >
      <h3 className="text-lg font-medium leading-6 text-gray-900 mb-4">
        Create Folder
      </h3>

      <form onSubmit={handleSubmit} className="space-y-4">
        {/* Folder Name */}
        <div>
          <label
            htmlFor="folder-name"
            className="block text-sm font-medium text-gray-700 mb-1"
          >
            Folder Name
          </label>
          <input
            id="folder-name"
            type="text"
            value={name}
            onChange={(e) => setName(e.target.value)}
            placeholder="My Folder"
            className="w-full px-3 py-2 border border-gray-300 rounded-md shadow-sm focus:outline-none focus:ring-2 focus:ring-blue-500 focus:border-blue-500 sm:text-sm"
            autoFocus
          />
        </div>

        {/* Error Message */}
        {error && (
          <div className="rounded-md bg-red-50 p-3">
            <p className="text-sm text-red-800">{error}</p>
          </div>
        )}

        {/* Action Buttons */}
        <div className="flex justify-end space-x-2 pt-2">
          <button
            type="button"
            onClick={handleClose}
            disabled={loading}
            className="px-4 py-2 min-h-[44px] text-gray-700 bg-gray-200 hover:bg-gray-300 rounded-md font-medium transition-colors disabled:opacity-50 disabled:cursor-not-allowed"
          >
            Cancel
          </button>
          <button
            type="submit"
            disabled={loading || !name.trim()}
            className="px-4 py-2 min-h-[44px] bg-blue-600 text-white hover:bg-blue-700 rounded-md font-medium transition-colors disabled:opacity-50 disabled:cursor-not-allowed flex items-center gap-2"
          >
            {loading ? (
              <>
                <Spinner size="sm" className="text-white" />
                Creating...
              </>
            ) : (
              'Create'
            )}
          </button>
        </div>
      </form>
    </Modal>
  );
}
