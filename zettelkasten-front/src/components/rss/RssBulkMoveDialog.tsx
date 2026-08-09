import React, { useState } from 'react';
import { Modal } from '../ui/Modal';
import { RSSFolder } from '../../api/rss';

interface RssBulkMoveDialogProps {
  isOpen: boolean;
  onClose: () => void;
  onConfirm: (folder: string | null) => void;
  folders: RSSFolder[];
  feedCount: number;
}

export function RssBulkMoveDialog({
  isOpen,
  onClose,
  onConfirm,
  folders,
  feedCount,
}: RssBulkMoveDialogProps) {
  const [selectedFolder, setSelectedFolder] = useState<string | null>(null);
  const [newFolderName, setNewFolderName] = useState('');
  const [showNewFolderInput, setShowNewFolderInput] = useState(false);

  const handleMove = () => {
    onConfirm(showNewFolderInput ? newFolderName : selectedFolder);
    setNewFolderName('');
    setSelectedFolder(null);
    setShowNewFolderInput(false);
  };

  const handleClose = () => {
    setNewFolderName('');
    setSelectedFolder(null);
    setShowNewFolderInput(false);
    onClose();
  };

  const handleCreateNewFolder = () => {
    setNewFolderName('');
    setSelectedFolder(null);
    setShowNewFolderInput(true);
  };

  const handleSelectFolder = (folderName: string | null) => {
    setSelectedFolder(folderName);
    setShowNewFolderInput(false);
  };

  if (!isOpen) return null;

  return (
    <Modal
      open={isOpen}
      onClose={onClose}
      size="md"
      dialogClassName="z-50"
      className="!max-w-[500px]"
    >
      <h3 className="text-lg font-semibold mb-4">
        Move {feedCount} feed{feedCount !== 1 ? 's' : ''} to folder
      </h3>

      {!showNewFolderInput ? (
        <div className="space-y-4">
          <div>
            <button
              onClick={handleCreateNewFolder}
              className="w-full px-4 py-2 text-left text-blue-600 hover:bg-blue-50 rounded-md border border-transparent hover:border-blue-300"
            >
              <svg
                className="w-5 h-5 inline mr-2"
                fill="none"
                viewBox="0 0 24 24"
                stroke="currentColor"
              >
                <path
                  strokeLinecap="round"
                  strokeLinejoin="round"
                  strokeWidth={2}
                  d="M12 4v16m8-8H4"
                />
              </svg>
              Create new folder...
            </button>
          </div>

          {folders.length > 0 && (
            <div>
              <p className="text-sm text-gray-600 mb-2">
                Or select existing folder:
              </p>
              <div className="space-y-1 max-h-60 overflow-y-auto">
                {folders.map((folder) => (
                  <button
                    key={folder.id}
                    onClick={() => handleSelectFolder(folder.name)}
                    className={`w-full px-4 py-2 text-left text-sm hover:bg-gray-50 rounded-md ${
                      selectedFolder === folder.name
                        ? 'bg-blue-50 border border-blue-300'
                        : 'border border-transparent'
                    }`}
                  >
                    {folder.name}
                  </button>
                ))}
              </div>
            </div>
          )}

          <div>
            <button
              onClick={() => handleSelectFolder(null)}
              className={`w-full px-4 py-2 text-left text-sm hover:bg-gray-50 rounded-md ${
                selectedFolder === null
                  ? 'bg-blue-50 border border-blue-300'
                  : 'border border-transparent'
              }`}
            >
              Uncategorized
            </button>
          </div>
        </div>
      ) : (
        <div className="space-y-4">
          <p className="text-sm text-gray-600">
            Enter a name for the new folder:
          </p>
          <input
            type="text"
            value={newFolderName}
            onChange={(e) => setNewFolderName(e.target.value)}
            placeholder="Folder name"
            className="w-full px-3 py-2 border border-gray-300 rounded-md focus:ring-2 focus:ring-blue-500 focus:border-blue-500"
            autoFocus
          />
          <div className="flex gap-2">
            <button
              onClick={() => setShowNewFolderInput(false)}
              className="px-4 py-2 bg-gray-200 text-gray-700 rounded-md hover:bg-gray-300 text-sm"
            >
              Back
            </button>
          </div>
        </div>
      )}

      <div className="flex justify-end gap-2 mt-6 pt-4 border-t border-gray-200">
        <button
          onClick={handleClose}
          className="px-4 py-2 bg-gray-200 text-gray-700 rounded-md hover:bg-gray-300"
        >
          Cancel
        </button>
        <button
          onClick={handleMove}
          disabled={!selectedFolder && !newFolderName}
          className={`px-4 py-2 rounded-md ${
            selectedFolder || newFolderName
              ? 'bg-blue-600 text-white hover:bg-blue-700'
              : 'bg-gray-300 text-gray-500 cursor-not-allowed'
          }`}
        >
          Move
        </button>
      </div>
    </Modal>
  );
}
