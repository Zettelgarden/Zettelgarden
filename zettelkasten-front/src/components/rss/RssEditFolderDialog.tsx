import React, { useState } from "react";
import { Dialog, Transition } from "@headlessui/react";
import { Fragment } from "react";
import { updateFolder, UpdateRSSFolderParams, RSSFolder } from "../../api/rss";

interface RssEditFolderDialogProps {
  isOpen: boolean;
  onClose: () => void;
  folder: RSSFolder | null;
  onFolderUpdated: (folder: RSSFolder) => void;
}

export function RssEditFolderDialog({
  isOpen,
  onClose,
  folder,
  onFolderUpdated,
}: RssEditFolderDialogProps) {
  const [name, setName] = useState("");
  const [loading, setLoading] = useState(false);
  const [error, setError] = useState<string>("");

  // Populate form when folder changes
  React.useEffect(() => {
    if (folder) {
      setName(folder.name);
    }
  }, [folder]);

  const handleSubmit = async (e: React.FormEvent) => {
    e.preventDefault();
    if (!folder) {
      setError("No folder selected");
      return;
    }

    if (!name.trim()) {
      setError("Folder name is required");
      return;
    }

    setLoading(true);
    setError("");

    try {
      const folderParams: UpdateRSSFolderParams = {};

      if (name.trim() !== folder.name) {
        folderParams.name = name.trim();
      }

      const updatedFolder = await updateFolder(folder.id, folderParams);
      onFolderUpdated(updatedFolder);
      handleClose();
    } catch (err) {
      console.error("Failed to update folder:", err);
      setError(err instanceof Error ? err.message : "Failed to update folder. Please try again.");
    } finally {
      setLoading(false);
    }
  };

  const handleClose = () => {
    setName("");
    setError("");
    onClose();
  };

  if (!folder) return null;

  return (
    <Transition appear show={isOpen} as={Fragment}>
      <Dialog as="div" className="relative z-[80]" onClose={handleClose}>
        <Transition.Child
          as={Fragment}
          enter="ease-out duration-300"
          enterFrom="opacity-0"
          enterTo="opacity-100"
          leave="ease-in duration-200"
          leaveFrom="opacity-100"
          leaveTo="opacity-0"
        >
          <div className="fixed inset-0 bg-black bg-opacity-30" />
        </Transition.Child>

        <div className="fixed inset-0 overflow-y-auto">
          <div className="flex min-h-full items-center justify-center p-4 text-center">
            <Transition.Child
              as={Fragment}
              enter="ease-out duration-300"
              enterFrom="opacity-0 scale-95"
              enterTo="opacity-100 scale-100"
              leave="ease-in duration-200"
              leaveFrom="opacity-100 scale-100"
              leaveTo="opacity-0 scale-95"
            >
              <Dialog.Panel className="w-full max-w-md transform overflow-hidden rounded-2xl bg-white p-6 text-left align-middle shadow-xl transition-all">
                <Dialog.Title as="h3" className="text-lg font-medium leading-6 text-gray-900 mb-4">
                  Rename Folder
                </Dialog.Title>

                <form onSubmit={handleSubmit} className="space-y-4">
                  {/* Folder Name */}
                  <div>
                    <label htmlFor="folder-name" className="block text-sm font-medium text-gray-700 mb-1">
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
                          <svg className="w-4 h-4 animate-spin" fill="none" viewBox="0 0 24 24">
                            <circle className="opacity-25" cx="12" cy="12" r="10" stroke="currentColor" strokeWidth="4" />
                            <path className="opacity-75" fill="currentColor" d="M4 12a8 8 0 018-8V0C5.373 0 0 5.373 0 12h4zm2 5.291A7.962 7.962 0 014 12H0c0 3.042 1.135 5.824 3 7.938l3-2.647z" />
                          </svg>
                          Saving...
                        </>
                      ) : (
                        "Save"
                      )}
                    </button>
                  </div>
                </form>
              </Dialog.Panel>
            </Transition.Child>
          </div>
        </div>
      </Dialog>
    </Transition>
  );
}
