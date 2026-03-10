import React, { useState, ChangeEvent, useEffect } from "react";
import { getAllFiles, FilesResponse } from "../api/files";
import { FileListItem } from "../components/files/FileListItem";
import { FileUpload } from "../components/files/FileUpload";
import { useUIState } from "../contexts/UIStateContext";
import { MobileTopBar } from "../components/layout/MobileTopBar";

import { File } from "../models/File";
import { defaultCard } from "../models/Card";
import { HeaderSection } from "../components/Header";
import { setDocumentTitle } from "../utils/title";

export function FileVault() {
  const { toggleMobileSidebar, refreshFiles, setRefreshFiles } = useUIState();
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

  function onDelete(file_id: number) {
    setFiles(files.filter((file) => file.id !== file_id));
    // Refresh the current page to get updated counts
    fetchFiles(currentPage, searchTerm, filetypeFilter, unlinkedOnly);
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
    unlinked: boolean = unlinkedOnly
  ) => {
    setLoading(true);
    try {
      const data = await getAllFiles(page, itemsPerPage, search, filetype, unlinked);
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

  const handleSearch = () => {
    setSearchTerm(filterString);
    setCurrentPage(1);
  };

  const handleKeyPress = (e: React.KeyboardEvent) => {
    if (e.key === 'Enter') {
      handleSearch();
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

  const clearAllFilters = () => {
    setFilterString("");
    setSearchTerm("");
    setFiletypeFilter("");
    setUnlinkedOnly(false);
    setCurrentPage(1);
  };

  useEffect(() => {
    if (refreshFiles) {
      fetchFiles(1, "", "", false);
      setRefreshFiles(false);
    }
  }, [refreshFiles]);

  useEffect(() => {
    setDocumentTitle("Files");
    fetchFiles(1, "", "", false);
  }, []);

  useEffect(() => {
    fetchFiles(currentPage, searchTerm, filetypeFilter, unlinkedOnly);
  }, [currentPage, filetypeFilter, unlinkedOnly]);

  useEffect(() => {
    // Fetch when searchTerm changes (triggered by Enter press)
    setCurrentPage(1);
    fetchFiles(1, searchTerm, filetypeFilter, unlinkedOnly);
  }, [searchTerm]);

  return (
    <div>
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
              onKeyPress={handleKeyPress}
              placeholder="Search files... (Press Enter to search)"
              className="flex-grow h-9 px-3 border border-slate-300 rounded-md text-sm focus:outline-none focus:ring-2 focus:ring-blue-500 focus:border-blue-500"
            />
            <button
              onClick={handleSearch}
              className="h-9 bg-blue-600 hover:bg-blue-700 text-white rounded-md px-4 text-sm flex-shrink-0"
            >
              Search
            </button>
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
              <div className="text-sm text-gray-600 mb-4 px-4">
                Showing {files.length} files (Page {currentPage} of {totalPages}, {totalItems} total)
              </div>

              {/* Files List */}
              <div className="bg-white border border-gray-200 rounded-md overflow-hidden">
                <ul className="divide-y divide-gray-200">
                  {files.map((file, index) => (
                    <li key={file.id} className="hover:bg-gray-50">
                      <FileListItem
                        file={file}
                        onDelete={onDelete}
                        setRefreshFiles={setRefreshFiles}
                        filterString={filterString}
                        setFilterString={setFilterString}
                      />
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
