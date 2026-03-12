import { File } from "../../models/File";
import { PartialCard } from "../../models/Card";
import { renderFile, deleteFile, editFile, downloadThumbnail, downloadFile, importEpub } from "../../api/files";
import { Link } from "react-router-dom";
import React, { useState, KeyboardEvent, useEffect } from "react";
import { FileIcon } from "../../assets/icons/FileIcon";
import { FileRender } from "./FileRender";
import { FilePreview } from "./FilePreview";
import { Menu } from "@headlessui/react";

import { BacklinkInput } from "../cards/BacklinkInput";
import { CardIdDiscoveryDialog } from "../cards/CardIdDiscoveryDialog";
import { useToast } from "../toast/ToastContext";

interface FileListItemProps {
  file: File;
  onDelete: (file_id: number) => void;
  setRefreshFiles: (refresh: boolean) => void;
  displayFileOnCard?: (file: File) => void;
  filterString: string;
  setFilterString: (text: string) => void;
  onEditDetails?: (file: File) => void;
}

export function FileListItem({
  file,
  onDelete,
  setRefreshFiles,
  displayFileOnCard,
  filterString,
  setFilterString,
  onEditDetails,
}: FileListItemProps) {
  const [newName, setNewName] = useState<string>("");
  const [showEditName, setShowEditName] = useState<boolean>(false);
  const [renderImage, setRenderImage] = useState<boolean>(false);
  const [showCardLink, setShowCardLink] = useState<boolean>(false);
  const [thumbnailUrl, setThumbnailUrl] = useState<string | null>(null);
  const [showPdfPreview, setShowPdfPreview] = useState<boolean>(false);
  const [pdfUrl, setPdfUrl] = useState<string | null>(null);
  const [isImporting, setIsImporting] = useState<boolean>(false);
  const [showImportDialog, setShowImportDialog] = useState<boolean>(false);
  const { showToast } = useToast();

  function toggleEditName() {
    setNewName(file.name);
    setShowEditName(!showEditName);
  }
  function handleTitleEdit() {
    editFile(file["id"].toString(), { name: newName, card_pk: file.card_pk });
    toggleEditName();
    setRefreshFiles(true);
  }
  function closeRenderImage() {
    setRenderImage(false);
  }
  const handleFileDownload = (file: File, e: React.MouseEvent) => {
    e.preventDefault();
    const isImage = file.filetype === "image/png" ||
      file.filetype === "image/jpeg" ||
      file.filetype === "image/jpg" ||
      file.filetype === "image/gif" ||
      file.filetype === "image/webp";
    const isPDF = file.filetype === "application/pdf";

    if (isImage) {
      setRenderImage(true);
      return;
    }

    if (isPDF) {
      // Load PDF and show preview
      downloadFile(file.id.toString())
        .then((blobUrl) => {
          if (blobUrl) {
            setPdfUrl(blobUrl);
            setShowPdfPreview(true);
          }
        })
        .catch((error) => {
          console.error("Error loading PDF:", error);
        });
      return;
    }

    // For other file types, trigger download
    renderFile(file.id, file.name).catch((error) => {
      console.error("Error downloading file:", error);
    });
  };
  const handleFileDelete = (file_id: number) => {
    if (window.confirm("Are you sure you want to delete this file?")) {
      deleteFile(file_id)
        .then(() => {
          onDelete(file_id);
        })
        .catch((error) => {
          console.error("Error deleting file:", error);
        });
    }
  };

  async function handleBacklink(card: PartialCard) {
    editFile(file["id"].toString(), { name: file.name, card_pk: card.id }).then(
      (file) => {
        setShowCardLink(false);
        setRefreshFiles(true);
      },
    );
  }

  function toggleCardLink() {
    setShowCardLink(!showCardLink);
  }

  function onFileTypeClick(file: File) {
    setFilterString("filetype:" + file.filetype)
  }

  async function handleCardUnlink() {
    editFile(file["id"].toString(), { name: file.name, card_pk: -1 }).then(
      (file) => {
        setShowCardLink(false);
        setRefreshFiles(true);
      },
    );
  }

  async function handleDisplayCardClick() {
    if (!displayFileOnCard) {
      return
    }
    displayFileOnCard(file)
    setRefreshFiles(true);
  }

  function openImportDialog() {
    setShowImportDialog(true);
  }

  async function handleImportEpub(cardId: string) {
    setIsImporting(true);
    setShowImportDialog(false);
    try {
      const result = await importEpub(file.id, cardId);
      const totalCards = result.child_card_ids.length + 1;
      const title = result.metadata.title || file.name;
      showToast("success", "Epub Imported", `Created ${totalCards} cards from "${title}"`);
    } catch (error) {
      const message = error instanceof Error ? error.message : "Failed to import epub";
      showToast("error", "Import Failed", message);
    } finally {
      setIsImporting(false);
    }
  }

  useEffect(() => {
    // Load thumbnail if available
    if (file.thumbnail_path && file.filetype.startsWith("image/")) {
      downloadThumbnail(file.id.toString())
        .then((blobUrl) => {
          if (blobUrl) {
            setThumbnailUrl(blobUrl);
          }
        })
        .catch((error) => {
          console.error("Error loading thumbnail:", error);
        });
    }
  }, [file.id, file.thumbnail_path]);

  const isImage = file.filetype.startsWith("image/");

  return (
    <div className="px-3 py-2">
      <div className="flex items-center justify-between gap-3">
        {/* Thumbnail or Icon */}
        {isImage && thumbnailUrl ? (
          <div className="flex-shrink-0 w-16 h-16">
            <img
              src={thumbnailUrl}
              alt={file.name}
              className="w-full h-full object-cover rounded border border-gray-200"
              onError={(e) => {
                // Fallback to icon if thumbnail fails to load
                e.currentTarget.style.display = 'none';
              }}
            />
          </div>
        ) : (
          <div className="flex-shrink-0 text-gray-400 w-4 h-4">
            <FileIcon />
          </div>
        )}

        <div className="flex-grow min-w-0">
          <div className="flex items-center gap-2 mb-1">
            {showEditName ? (
              <input
                className="flex-grow px-2 py-1 text-sm border border-gray-300 rounded focus:outline-none focus:ring-2 focus:ring-blue-500 focus:border-blue-500"
                value={newName}
                onChange={(e) => setNewName(e.target.value)}
                onKeyPress={(event: KeyboardEvent<HTMLInputElement>) => {
                  if (event.key === "Enter") {
                    handleTitleEdit();
                  }
                }}
                autoFocus
              />
            ) : (
              <div className="flex-grow min-w-0">
                <a
                  href="#"
                  onClick={(e) => handleFileDownload(file, e)}
                  className="text-gray-900 hover:text-blue-600 font-medium truncate block text-sm"
                >
                  {file.name}
                </a>
                {renderImage && (
                  <div
                    onClick={closeRenderImage}
                    className="fixed inset-0 bg-black bg-opacity-50 flex items-center justify-center z-50 cursor-pointer"
                  >
                    <FileRender file={file} />
                  </div>
                )}
                {showPdfPreview && pdfUrl && (
                  <FilePreview
                    fileUrl={pdfUrl}
                    filename={file.name}
                    onClose={() => {
                      setShowPdfPreview(false);
                      URL.revokeObjectURL(pdfUrl);
                      setPdfUrl(null);
                    }}
                  />
                )}
              </div>
            )}
          </div>

          <div className="flex items-center gap-3 text-xs text-gray-500">
            <span>
              {new Date(file.created_at).toLocaleDateString()}
            </span>
            <span
              className="cursor-pointer hover:text-blue-600 px-1.5 py-0.5 rounded bg-gray-100 hover:bg-blue-50 text-xs"
              onClick={() => onFileTypeClick(file)}
              title="Click to filter by this file type"
            >
              {file.filetype}
            </span>
            <span>
              {(file.size / 1024).toFixed(1)} KB
            </span>
          </div>
        </div>

        <div className="flex items-center gap-2 ml-3">
          {file.card_pk > 0 && (
            <Link
              to={`/app/card/${file.card.id}`}
              className="text-blue-600 hover:text-blue-700 text-xs font-medium bg-blue-50 hover:bg-blue-100 px-1.5 py-0.5 rounded"
            >
              [{file.card.card_id}]
            </Link>
          )}

          {(!file.card || file.card.id == 0) && showCardLink && (
            <div className="min-w-0">
              <BacklinkInput addBacklink={handleBacklink} />
            </div>
          )}

          {/* Menu Dropdown */}
          <Menu as="div" className="relative">
            <Menu.Button className="p-1 text-gray-400 hover:text-gray-600 rounded hover:bg-gray-100">
              <svg className="w-4 h-4" fill="currentColor" viewBox="0 0 20 20">
                <path d="M10 6a2 2 0 110-4 2 2 0 010 4zM10 12a2 2 0 110-4 2 2 0 010 4zM10 18a2 2 0 110-4 2 2 0 010 4z" />
              </svg>
            </Menu.Button>
            <Menu.Items className="absolute right-0 mt-1 w-44 bg-white border border-gray-200 rounded-md shadow-lg z-10 py-1">
              <Menu.Item>
                {({ active }) => (
                  <button
                    onClick={() => toggleEditName()}
                    className={`block w-full px-3 py-1.5 text-left text-sm ${
                      active ? 'bg-gray-100 text-gray-900' : 'text-gray-700'
                    }`}
                  >
                    Rename
                  </button>
                )}
              </Menu.Item>

              {onEditDetails && (
                <Menu.Item>
                  {({ active }) => (
                    <button
                      onClick={() => onEditDetails(file)}
                      className={`block w-full px-3 py-1.5 text-left text-sm ${
                        active ? 'bg-gray-100 text-gray-900' : 'text-gray-700'
                      }`}
                    >
                      Edit Details
                    </button>
                  )}
                </Menu.Item>
              )}

              {file.card_pk <= 1 ? (
                <Menu.Item>
                  {({ active }) => (
                    <button
                      onClick={() => toggleCardLink()}
                      className={`block w-full px-3 py-1.5 text-left text-sm ${
                        active ? 'bg-gray-100 text-gray-900' : 'text-gray-700'
                      }`}
                    >
                      Link to Card
                    </button>
                  )}
                </Menu.Item>
              ) : (
                <Menu.Item>
                  {({ active }) => (
                    <button
                      onClick={() => handleCardUnlink()}
                      className={`block w-full px-3 py-1.5 text-left text-sm ${
                        active ? 'bg-gray-100 text-gray-900' : 'text-gray-700'
                      }`}
                    >
                      Unlink from Card
                    </button>
                  )}
                </Menu.Item>
              )}

              {displayFileOnCard && file.filetype.includes("image") && (
                <Menu.Item>
                  {({ active }) => (
                    <button
                      onClick={() => handleDisplayCardClick()}
                      className={`block w-full px-3 py-1.5 text-left text-sm ${
                        active ? 'bg-gray-100 text-gray-900' : 'text-gray-700'
                      }`}
                    >
                      Display on Card
                    </button>
                  )}
                </Menu.Item>
              )}

              {file.filetype === "application/epub+zip" && (
                <Menu.Item>
                  {({ active }) => (
                    <button
                      onClick={() => openImportDialog()}
                      disabled={isImporting}
                      className={`block w-full px-3 py-1.5 text-left text-sm ${
                        active ? 'bg-gray-100 text-gray-900' : 'text-gray-700'
                      } ${isImporting ? 'opacity-50 cursor-not-allowed' : ''}`}
                    >
                      {isImporting ? "Importing..." : "Import as Cards"}
                    </button>
                  )}
                </Menu.Item>
              )}

              <div className="border-t border-gray-100"></div>
              <Menu.Item>
                {({ active }) => (
                  <button
                    onClick={() => handleFileDelete(file.id)}
                    className={`block w-full px-3 py-1.5 text-left text-sm ${
                      active ? 'bg-red-50 text-red-700' : 'text-red-600'
                    }`}
                  >
                    Delete
                  </button>
                )}
              </Menu.Item>
            </Menu.Items>
          </Menu>
        </div>
      </div>

      {/* Import Epub Dialog */}
      {showImportDialog && (
        <CardIdDiscoveryDialog
          onClose={() => setShowImportDialog(false)}
          onSelectId={(cardId) => handleImportEpub(cardId)}
        />
      )}
    </div>
  );
}
