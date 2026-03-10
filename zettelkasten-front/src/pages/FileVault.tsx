import React, { useState, ChangeEvent, useEffect, useCallback, useRef } from "react";
import { getAllFiles, FilesResponse, deleteFile, editFile, uploadFile } from "../api/files";
import { FileListItem } from "../components/files/FileListItem";
import { FileUpload } from "../components/files/FileUpload";
import { useUIState } from "../contexts/UIStateContext";
import { MobileTopBar } from "../components/layout/MobileTopBar";
import { useToast } from "../components/toast/ToastContext";

import { File } from "../models/File";
import { defaultCard, PartialCard } from "../models/Card";
import { HeaderSection } from "../components/Header";
import { setDocumentTitle } from "../utils/title";
import { BacklinkInput } from "../components/cards/BacklinkInput";

export function FileVault() {
  const { toggleMobileSidebar, refreshFiles, setRefreshFiles } = useUIState();
  const { showToast } = useToast();
  const [files, setFiles] = useState<File[]>([]);
  const [loading, setLoading] = useState(true);
  const [error, setError] = useState<string | null>(null);
  const [filterString, setFilterString] = useState<string>("");
  const [searchTerm, setSearchTerm] = useState<string>(""); // Actual search term sent to API
  const [filetypeFilter, setFiletypeFilter] = useState<string>("");
  const [unlinkedOnly, setUnlinkedOnly] = useState(false);
  const [currentPage, setCurrentPage] = useState(1);
  const [itemsPerPage] = useState(20);
  const [totalPages, setTotalPages] = useState(1);
  const [totalItems, setTotalItems] = useState(0);
  const [showFilterHelp, setShowFilterHelp] = useState<boolean>(false);
  const [storageUsed, setStorageUsed] = useState(0);
  const [maxStorage, setMaxStorage] = useState(0);

  // Bulk selection state
  const [selectedFiles, setSelectedFiles] = useState<Set<number>>(new Set());
  const [showBulkLinkInput, setShowBulkLinkInput] = useState(false);
  const [isBulkProcessing, setIsBulkProcessing] = useState(false);

  // Drag and drop state
  const [isDraggingOver, setIsDraggingOver] = useState(false);
  const [isUploading, setIsUploading] = useState(false);
  const dragCounterRef = useRef(0);

  // Sort state
  const [sortBy, setSortBy] = useState<string>("date");
  const [sortOrder, setSortOrder] = useState<string>("desc");

  // Sort options
  const sortOptions = [
    { value: "date", label: "Date uploaded" },
    { value: "name", label: "Name" },
    { value: "size", label: "File size" },
    { value: "type", label: "File type" },
  ];

  // File type filter options
  const fileTypeOptions = [
    { label: "All", value: "" },
    { label: "PDF", value: "pdf" },
    { label: "Images", value: "image/" },
    { label: "Documents", value: "document" },
    { label: "Spreadsheets", value: "spreadsheet" },
  ];

  // Helper to format bytes to human readable
  const formatBytes = (bytes: number): string => {
    if (bytes === 0) return '0 B';
    const k = 1024;
    const sizes = ['B', 'KB', 'MB', 'GB', 'TB'];
    const i = Math.floor(Math.log(bytes) / Math.log(k));
    return parseFloat((bytes / Math.pow(k, i)).toFixed(1)) + ' ' + sizes[i];
  };

  // Calculate storage percentage for progress bar
  const storagePercentage = maxStorage > 0 ? Math.min((storageUsed / maxStorage) * 100, 100) : 0;

  // Check if any filters are active
  const hasActiveFilters = searchTerm || filetypeFilter || unlinkedOnly;

  // Bulk selection helpers
  const isAllSelected = files.length > 0 && selectedFiles.size === files.length;
  const isSomeSelected = selectedFiles.size > 0;

  const toggleFileSelection = (fileId: number) => {
    setSelectedFiles((prev) => {
      const newSet = new Set(prev);
      if (newSet.has(fileId)) {
        newSet.delete(fileId);
      } else {
        newSet.add(fileId);
      }
      return newSet;
    });
  };

  const toggleSelectAll = () => {
    if (isAllSelected) {
      setSelectedFiles(new Set());
    } else {
      setSelectedFiles(new Set(files.map((f) => f.id)));
    }
  };

  const clearSelection = () => {
    setSelectedFiles(new Set());
    setShowBulkLinkInput(false);
  };

  // Bulk operations
  const handleBulkDelete = async () => {
    if (selectedFiles.size === 0) return;

    const confirmed = window.confirm(
      `Are you sure you want to delete ${selectedFiles.size} file(s)? This action cannot be undone.`
    );
    if (!confirmed) return;

    setIsBulkProcessing(true);
    let successCount = 0;
    let failCount = 0;

    for (const fileId of selectedFiles) {
      try {
        await deleteFile(fileId);
        successCount++;
      } catch (err) {
        failCount++;
        console.error(`Failed to delete file ${fileId}:`, err);
      }
    }

    setIsBulkProcessing(false);
    clearSelection();

    if (successCount > 0) {
      showToast("success", "Files Deleted", `${successCount} file(s) deleted successfully`);
    }
    if (failCount > 0) {
      showToast("error", "Delete Failed", `${failCount} file(s) could not be deleted`);
    }

    // Refresh the list
    fetchFiles(currentPage, searchTerm, filetypeFilter, unlinkedOnly, sortBy, sortOrder);
  };

  const handleBulkLink = async (card: PartialCard) => {
    if (selectedFiles.size === 0) return;

    setIsBulkProcessing(true);
    let successCount = 0;
    let failCount = 0;

    for (const fileId of selectedFiles) {
      const file = files.find((f) => f.id === fileId);
      if (!file) continue;

      try {
        await editFile(fileId.toString(), { name: file.name, card_pk: card.id });
        successCount++;
      } catch (err) {
        failCount++;
        console.error(`Failed to link file ${fileId}:`, err);
      }
    }

    setIsBulkProcessing(false);
    setShowBulkLinkInput(false);
    clearSelection();

    if (successCount > 0) {
      showToast("success", "Files Linked", `${successCount} file(s) linked to card`);
    }
    if (failCount > 0) {
      showToast("error", "Link Failed", `${failCount} file(s) could not be linked`);
    }

    fetchFiles(currentPage, searchTerm, filetypeFilter, unlinkedOnly, sortBy, sortOrder);
  };

  const handleBulkUnlink = async () => {
    if (selectedFiles.size === 0) return;

    setIsBulkProcessing(true);
    let successCount = 0;
    let failCount = 0;

    for (const fileId of selectedFiles) {
      const file = files.find((f) => f.id === fileId);
      if (!file) continue;

      try {
        await editFile(fileId.toString(), { name: file.name, card_pk: -1 });
        successCount++;
      } catch (err) {
        failCount++;
        console.error(`Failed to unlink file ${fileId}:`, err);
      }
    }

    setIsBulkProcessing(false);
    clearSelection();

    if (successCount > 0) {
      showToast("success", "Files Unlinked", `${successCount} file(s) unlinked`);
    }
    if (failCount > 0) {
      showToast("error", "Unlink Failed", `${failCount} file(s) could not be unlinked`);
    }

    fetchFiles(currentPage, searchTerm, filetypeFilter, unlinkedOnly, sortBy, sortOrder);
  };

  // Drag and drop handlers
  const handleDragEnter = useCallback((e: React.DragEvent) => {
    e.preventDefault();
    e.stopPropagation();
    dragCounterRef.current++;
    if (e.dataTransfer.types.includes('Files')) {
      setIsDraggingOver(true);
    }
  }, []);

  const handleDragLeave = useCallback((e: React.DragEvent) => {
    e.preventDefault();
    e.stopPropagation();
    dragCounterRef.current--;
    if (dragCounterRef.current === 0) {
      setIsDraggingOver(false);
    }
  }, []);

  const handleDragOver = useCallback((e: React.DragEvent) => {
    e.preventDefault();
    e.stopPropagation();
  }, []);

  const handleDrop = useCallback(async (e: React.DragEvent) => {
    e.preventDefault();
    e.stopPropagation();
    setIsDraggingOver(false);
    dragCounterRef.current = 0;

    const droppedFiles = Array.from(e.dataTransfer.files);
    if (droppedFiles.length === 0) return;

    setIsUploading(true);
    let successCount = 0;
    let failCount = 0;

    for (const file of droppedFiles) {
      try {
        const response = await uploadFile(file, -1); // Upload without linking to a card
        if ("error" in response) {
          failCount++;
        } else {
          successCount++;
        }
      } catch (err) {
        failCount++;
        console.error(`Failed to upload ${file.name}:`, err);
      }
    }

    setIsUploading(false);

    if (successCount > 0) {
      showToast("success", "Upload Complete", `${successCount} file(s) uploaded successfully`);
      setRefreshFiles(true);
    }
    if (failCount > 0) {
      showToast("error", "Upload Failed", `${failCount} file(s) could not be uploaded`);
    }
  }, [showToast, setRefreshFiles]);

  function onDelete(file_id: number) {
    setFiles(files.filter((file) => file.id !== file_id));
    // Refresh the current page to get updated counts
    fetchFiles(currentPage, searchTerm, filetypeFilter, unlinkedOnly, sortBy, sortOrder);
  }

  function handleFilter(
    e: ChangeEvent<HTMLInputElement | HTMLTextAreaElement | HTMLSelectElement>
  ) {
    setFilterString(e.target.value);
  }

  const fetchFiles = async (
    page: number = currentPage,
    search: string = searchTerm,
    filetype: string = filetypeFilter,
    unlinked: boolean = unlinkedOnly,
    sort: string = sortBy,
    order: string = sortOrder
  ) => {
    setLoading(true);
    try {
      const data = await getAllFiles(page, itemsPerPage, search, filetype, unlinked, sort, order);
      setFiles(data.files);
      setCurrentPage(data.page);
      setTotalPages(data.total_pages);
      setTotalItems(data.total);
      setStorageUsed(data.storage_used || 0);
      setMaxStorage(data.max_storage || 0);
      setError(null);
    } catch (err) {
      setError("Failed to load files");
      console.error("Error loading files:", err);
    } finally {
      setLoading(false);
    }
  };

  const handleFiletypeChange = (filetype: string) => {
    setFiletypeFilter(filetype);
    setCurrentPage(1);
  };

  const handleUnlinkedToggle = () => {
    setUnlinkedOnly(!unlinkedOnly);
    setCurrentPage(1);
  };

  const handleSortChange = (newSortBy: string) => {
    if (sortBy === newSortBy) {
      // Toggle order if clicking same sort field
      setSortOrder(sortOrder === "asc" ? "desc" : "asc");
    } else {
      setSortBy(newSortBy);
      setSortOrder("desc"); // Default to descending for new sort field
    }
    setCurrentPage(1);
  };

  const clearAllFilters = () => {
    setFilterString("");
    setSearchTerm("");
    setFiletypeFilter("");
    setUnlinkedOnly(false);
    setCurrentPage(1);
  };

  // Debounced search effect - triggers 300ms after user stops typing
  useEffect(() => {
    const debounceTimer = setTimeout(() => {
      setSearchTerm(filterString);
      setCurrentPage(1);
    }, 300);

    return () => clearTimeout(debounceTimer);
  }, [filterString]);

  useEffect(() => {
    if (refreshFiles) {
      fetchFiles(1, "", "", false, sortBy, sortOrder);
      setRefreshFiles(false);
    }
  }, [refreshFiles]);

  useEffect(() => {
    setDocumentTitle("Files");
    fetchFiles(1, "", "", false, sortBy, sortOrder);
  }, []);

  useEffect(() => {
    fetchFiles(currentPage, searchTerm, filetypeFilter, unlinkedOnly, sortBy, sortOrder);
  }, [currentPage, filetypeFilter, unlinkedOnly, sortBy, sortOrder]);

  useEffect(() => {
    // Fetch when searchTerm changes (triggered by debounce)
    setCurrentPage(1);
    fetchFiles(1, searchTerm, filetypeFilter, unlinkedOnly, sortBy, sortOrder);
  }, [searchTerm]);

  return (
    <div
      onDragEnter={handleDragEnter}
      onDragLeave={handleDragLeave}
      onDragOver={handleDragOver}
      onDrop={handleDrop}
      className="relative"
    >
      {/* Drag and Drop Overlay */}
      {isDraggingOver && (
        <div className="fixed inset-0 bg-blue-500 bg-opacity-10 z-50 pointer-events-none">
          <div className="absolute inset-4 border-4 border-dashed border-blue-500 rounded-lg flex items-center justify-center bg-white bg-opacity-90">
            <div className="text-center">
              <svg className="w-16 h-16 mx-auto text-blue-500 mb-4" fill="none" stroke="currentColor" viewBox="0 0 24 24">
                <path strokeLinecap="round" strokeLinejoin="round" strokeWidth={2} d="M7 16a4 4 0 01-.88-7.903A5 5 0 1115.9 6L16 6a5 5 0 011 9.9M15 13l-3-3m0 0l-3 3m3-3v12" />
              </svg>
              <p className="text-xl font-semibold text-blue-600">Drop files here to upload</p>
              <p className="text-sm text-gray-500 mt-1">Files will be uploaded without linking to a card</p>
            </div>
          </div>
        </div>
      )}

      {/* Upload in progress indicator */}
      {isUploading && (
        <div className="fixed inset-0 bg-black bg-opacity-30 z-50 flex items-center justify-center">
          <div className="bg-white rounded-lg p-6 shadow-xl">
            <div className="flex items-center gap-3">
              <svg className="animate-spin h-5 w-5 text-blue-600" xmlns="http://www.w3.org/2000/svg" fill="none" viewBox="0 0 24 24">
                <circle className="opacity-25" cx="12" cy="12" r="10" stroke="currentColor" strokeWidth="4"></circle>
                <path className="opacity-75" fill="currentColor" d="M4 12a8 8 0 018-8V0C5.373 0 0 5.373 0 12h4zm2 5.291A7.962 7.962 0 014 12H0c0 3.042 1.135 5.824 3 7.938l3-2.647z"></path>
              </svg>
              <span className="text-gray-700 font-medium">Uploading files...</span>
            </div>
          </div>
        </div>
      )}

      <MobileTopBar
        title="Files"
        onMenuClick={toggleMobileSidebar}
      />
      <div className="p-4">
        <HeaderSection text="Files" />

      {/* Storage Quota Indicator */}
      {maxStorage > 0 && (
        <div className="mb-4 bg-white border border-gray-200 rounded-md p-3">
          <div className="flex items-center justify-between mb-2">
            <span className="text-sm font-medium text-gray-700">Storage</span>
            <span className="text-sm text-gray-600">
              {formatBytes(storageUsed)} / {formatBytes(maxStorage)}
            </span>
          </div>
          <div className="w-full bg-gray-200 rounded-full h-2">
            <div
              className={`h-2 rounded-full transition-all duration-300 ${
                storagePercentage >= 90
                  ? 'bg-red-500'
                  : storagePercentage >= 70
                  ? 'bg-yellow-500'
                  : 'bg-blue-500'
              }`}
              style={{ width: `${storagePercentage}%` }}
            />
          </div>
          {storagePercentage >= 90 && (
            <p className="mt-2 text-xs text-red-600">
              Storage almost full. Consider deleting unused files.
            </p>
          )}
        </div>
      )}

      {/* Search and Upload Section */}
      <div className="bg-slate-100 p-4 border-b border-slate-300 mb-4">
        <div className="flex flex-col sm:flex-row items-stretch sm:items-center gap-3">
          {/* Search input section */}
          <div className="flex-grow flex items-center gap-2">
            <input
              type="text"
              value={filterString}
              onChange={handleFilter}
              placeholder="Search files..."
              className="flex-grow h-9 px-3 border border-slate-300 rounded-md text-sm focus:outline-none focus:ring-2 focus:ring-blue-500 focus:border-blue-500"
            />
            <div className="relative">
              <span
                className="h-9 flex items-center justify-center w-8 cursor-pointer text-slate-500 hover:text-slate-700 rounded-md hover:bg-slate-200"
                onMouseEnter={() => setShowFilterHelp(true)}
                onMouseLeave={() => setShowFilterHelp(false)}
                aria-label="Search help"
              >
                ?
              </span>
              {showFilterHelp && (
                <div className="absolute top-full mt-2 right-0 bg-white p-3 border border-slate-300 rounded-md shadow-lg z-20 w-80">
                  <h4 className="font-semibold mb-2 text-slate-700">Search Options:</h4>
                  <ul className="list-none space-y-1 text-sm text-slate-600">
                    <li><strong>Filename:</strong> Searches file names (e.g., <code className="bg-slate-100 px-1 rounded">annual-report</code>)</li>
                    <li><strong>Filetype:</strong> <code className="bg-slate-100 px-1 rounded">filetype:TYPE</code> (e.g., <code className="bg-slate-100 px-1 rounded">filetype:pdf</code>)</li>
                    <li>Combine: <code className="bg-slate-100 px-1 rounded">filetype:pdf quarterly</code></li>
                  </ul>
                </div>
              )}
            </div>
          </div>

          {/* Upload section */}
          <div className="flex-shrink-0">
            <FileUpload
              card={defaultCard}
            >
              <button className="h-9 bg-green-600 hover:bg-green-700 text-white rounded-md px-4 text-sm font-medium">
                Upload File
              </button>
            </FileUpload>
          </div>
        </div>
      </div>

      {/* Filter Chips */}
      <div className="flex flex-wrap items-center gap-2 mb-4">
        {/* File type filter chips */}
        {fileTypeOptions.map((option) => (
          <button
            key={option.value}
            onClick={() => handleFiletypeChange(option.value)}
            className={`px-3 py-1.5 text-sm rounded-full border transition-colors ${
              filetypeFilter === option.value
                ? 'bg-blue-600 text-white border-blue-600'
                : 'bg-white text-gray-700 border-gray-300 hover:border-blue-400 hover:text-blue-600'
            }`}
          >
            {option.label}
          </button>
        ))}

        {/* Unlinked filter toggle */}
        <button
          onClick={handleUnlinkedToggle}
          className={`px-3 py-1.5 text-sm rounded-full border transition-colors ${
            unlinkedOnly
              ? 'bg-orange-500 text-white border-orange-500'
              : 'bg-white text-gray-700 border-gray-300 hover:border-orange-400 hover:text-orange-600'
          }`}
        >
          Unlinked Only
        </button>

        {/* Clear filters button */}
        {hasActiveFilters && (
          <button
            onClick={clearAllFilters}
            className="px-3 py-1.5 text-sm text-gray-500 hover:text-gray-700 underline"
          >
            Clear all
          </button>
        )}

        {/* Sort dropdown */}
        <div className="ml-auto flex items-center gap-2">
          <span className="text-sm text-gray-500">Sort by:</span>
          <select
            value={sortBy}
            onChange={(e) => handleSortChange(e.target.value)}
            className="h-8 px-2 text-sm border border-gray-300 rounded-md bg-white focus:outline-none focus:ring-2 focus:ring-blue-500 focus:border-blue-500"
          >
            {sortOptions.map((option) => (
              <option key={option.value} value={option.value}>
                {option.label} {sortBy === option.value && (sortOrder === 'asc' ? '↑' : '↓')}
              </option>
            ))}
          </select>
          <button
            onClick={() => setSortOrder(sortOrder === 'asc' ? 'desc' : 'asc')}
            className="h-8 w-8 flex items-center justify-center text-gray-500 hover:text-gray-700 border border-gray-300 rounded-md hover:bg-gray-50"
            title={sortOrder === 'asc' ? 'Ascending' : 'Descending'}
          >
            {sortOrder === 'asc' ? '↑' : '↓'}
          </button>
        </div>
      </div>

      {/* Loading State */}
      {loading && (
        <div className="flex justify-center w-full py-20">
          <div className="text-gray-600">Loading files...</div>
        </div>
      )}

      {/* Error State */}
      {error && (
        <div className="p-4 mb-4 bg-red-50 border border-red-200 rounded-md">
          <div className="text-red-800">{error}</div>
        </div>
      )}

      {/* Content */}
      {!loading && !error && (
        <>
          {files && files.length > 0 ? (
            <>
              {/* Results Info */}
              <div className="flex items-center justify-between mb-4 px-4">
                <div className="text-sm text-gray-600">
                  Showing {files.length} files (Page {currentPage} of {totalPages}, {totalItems} total)
                </div>
                {isSomeSelected && (
                  <div className="text-sm text-blue-600 font-medium">
                    {selectedFiles.size} selected
                  </div>
                )}
              </div>

              {/* Bulk Actions Toolbar */}
              {isSomeSelected && (
                <div className="mb-4 bg-blue-50 border border-blue-200 rounded-md p-3 flex items-center gap-3 flex-wrap">
                  <span className="text-sm font-medium text-blue-800">
                    {selectedFiles.size} file(s) selected
                  </span>
                  <div className="flex items-center gap-2 flex-wrap">
                    {showBulkLinkInput ? (
                      <div className="flex items-center gap-2">
                        <BacklinkInput addBacklink={handleBulkLink} />
                        <button
                          onClick={() => setShowBulkLinkInput(false)}
                          className="text-sm text-gray-500 hover:text-gray-700"
                        >
                          Cancel
                        </button>
                      </div>
                    ) : (
                      <>
                        <button
                          onClick={() => setShowBulkLinkInput(true)}
                          disabled={isBulkProcessing}
                          className="px-3 py-1.5 text-sm bg-white border border-gray-300 rounded hover:bg-gray-50 disabled:opacity-50"
                        >
                          Link to Card
                        </button>
                        <button
                          onClick={handleBulkUnlink}
                          disabled={isBulkProcessing}
                          className="px-3 py-1.5 text-sm bg-white border border-gray-300 rounded hover:bg-gray-50 disabled:opacity-50"
                        >
                          Unlink
                        </button>
                        <button
                          onClick={handleBulkDelete}
                          disabled={isBulkProcessing}
                          className="px-3 py-1.5 text-sm bg-red-500 text-white rounded hover:bg-red-600 disabled:opacity-50"
                        >
                          Delete
                        </button>
                      </>
                    )}
                    <button
                      onClick={clearSelection}
                      disabled={isBulkProcessing}
                      className="px-3 py-1.5 text-sm text-gray-500 hover:text-gray-700"
                    >
                      Clear selection
                    </button>
                  </div>
                </div>
              )}

              {/* Files List */}
              <div className="bg-white border border-gray-200 rounded-md overflow-hidden">
                {/* Select All Header */}
                <div className="flex items-center px-3 py-2 bg-gray-50 border-b border-gray-200">
                  <input
                    type="checkbox"
                    checked={isAllSelected}
                    onChange={toggleSelectAll}
                    className="w-4 h-4 rounded border-gray-300 text-blue-600 focus:ring-blue-500"
                    aria-label="Select all files"
                  />
                  <span className="ml-3 text-xs text-gray-500">
                    {isAllSelected ? "Deselect all" : "Select all"}
                  </span>
                </div>

                <ul className="divide-y divide-gray-200">
                  {files.map((file) => (
                    <li key={file.id} className={`hover:bg-gray-50 ${selectedFiles.has(file.id) ? 'bg-blue-50' : ''}`}>
                      <div className="flex items-center">
                        <input
                          type="checkbox"
                          checked={selectedFiles.has(file.id)}
                          onChange={() => toggleFileSelection(file.id)}
                          className="ml-3 w-4 h-4 rounded border-gray-300 text-blue-600 focus:ring-blue-500"
                          aria-label={`Select ${file.name}`}
                        />
                        <div className="flex-1">
                          <FileListItem
                            file={file}
                            onDelete={onDelete}
                            setRefreshFiles={setRefreshFiles}
                            filterString={filterString}
                            setFilterString={setFilterString}
                          />
                        </div>
                      </div>
                    </li>
                  ))}
                </ul>
              </div>

              {/* Pagination */}
              {totalItems > 0 && (
                <div className="flex justify-center items-center gap-4 mt-6 p-4">
                  <button
                    onClick={() => setCurrentPage(currentPage - 1)}
                    disabled={currentPage === 1}
                    className="px-4 py-2 text-sm font-medium text-gray-700 bg-white border border-gray-300 rounded-md shadow-sm hover:bg-gray-50 focus:outline-none focus:ring-2 focus:ring-offset-2 focus:ring-blue-500 disabled:opacity-50 disabled:cursor-not-allowed"
                  >
                    Previous
                  </button>
                  <span className="flex items-center text-sm text-gray-600">
                    Page {currentPage} of {totalPages} ({totalItems} total files)
                  </span>
                  <button
                    onClick={() => setCurrentPage(currentPage + 1)}
                    disabled={currentPage === totalPages}
                    className="px-4 py-2 text-sm font-medium text-gray-700 bg-white border border-gray-300 rounded-md shadow-sm hover:bg-gray-50 focus:outline-none focus:ring-2 focus:ring-offset-2 focus:ring-blue-500 disabled:opacity-50 disabled:cursor-not-allowed"
                  >
                    Next
                  </button>
                </div>
              )}
            </>
          ) : (
            <div className="text-center py-12">
              <div className="text-gray-500 text-lg mb-2">No files found</div>
              {hasActiveFilters ? (
                <div className="text-gray-400 text-sm">
                  Try adjusting your filters or{" "}
                  <button
                    onClick={clearAllFilters}
                    className="text-blue-600 hover:text-blue-700 underline"
                  >
                    clear all filters
                  </button>
                </div>
              ) : (
                <div className="text-gray-400 text-sm">
                  Upload your first file to get started
                </div>
              )}
            </div>
          )}
        </>
      )}
      </div>
    </div>
  );
}
