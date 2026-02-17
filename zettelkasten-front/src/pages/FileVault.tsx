import React, { useState, ChangeEvent, useEffect } from "react";
import { getAllFiles, FilesResponse } from "../api/files";
import { sortCards } from "../utils/cards";
import { FileRenameModal } from "../components/files/FileRenameModal";
import { FileListItem } from "../components/files/FileListItem";
import { FileUpload } from "../components/files/FileUpload";
import { Button } from "../components/Button";
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
  const [isRenameModalOpen, setIsRenameModalOpen] = useState(false);
  const [fileToRename, setFileToRename] = useState<File | null>(null);
  const [filterString, setFilterString] = useState<string>("");
  const [searchTerm, setSearchTerm] = useState<string>(""); // Actual search term sent to API
  const [currentPage, setCurrentPage] = useState(1);
  const [itemsPerPage] = useState(20);
  const [totalPages, setTotalPages] = useState(1);
  const [totalItems, setTotalItems] = useState(0);
  const [showFilterHelp, setShowFilterHelp] = useState<boolean>(false);

  function onDelete(file_id: number) {
    setFiles(files.filter((file) => file.id !== file_id));
    // Refresh the current page to get updated counts
    fetchFiles(currentPage, searchTerm);
  }

  function handleFilter(
    e: ChangeEvent<HTMLInputElement | HTMLTextAreaElement | HTMLSelectElement>
  ) {
    setFilterString(e.target.value);
  }

  function onRename(fileId: number, updatedFile: File) {
    setFiles((prevFiles) =>
      prevFiles.map((f) => (f.id === updatedFile.id ? updatedFile : f)),
    );
    setIsRenameModalOpen(false);
  }

  const fetchFiles = async (page: number = currentPage, search: string = searchTerm) => {
    setLoading(true);
    try {
      const data = await getAllFiles(page, itemsPerPage, search);
      setFiles(data.files);
      setCurrentPage(data.page);
      setTotalPages(data.total_pages);
      setTotalItems(data.total);
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

  useEffect(() => {
    if (refreshFiles) {
      fetchFiles(1, "");
      setRefreshFiles(false);
    }
  }, [refreshFiles]);

  useEffect(() => {
    setDocumentTitle("Files");
    fetchFiles(1, "");
  }, []);

  useEffect(() => {
    fetchFiles(currentPage, searchTerm);
  }, [currentPage]);

  useEffect(() => {
    // Fetch when searchTerm changes (triggered by Enter press)
    setCurrentPage(1);
    fetchFiles(1, searchTerm);
  }, [searchTerm]);

  return (
    <div>
      <MobileTopBar
        title="Files"
        onMenuClick={toggleMobileSidebar}
      />
      <div className="p-4">
        <HeaderSection text="Files" />

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
      <FileRenameModal
        isOpen={isRenameModalOpen}
        onClose={() => setIsRenameModalOpen(false)}
        onRename={onRename}
        file={fileToRename}
      />
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
              {searchTerm && (
                <div className="text-gray-400 text-sm">
                  Try adjusting your search criteria or{" "}
                  <button
                    onClick={() => {
                      setFilterString("");
                      setSearchTerm("");
                    }}
                    className="text-blue-600 hover:text-blue-700 underline"
                  >
                    clear search
                  </button>
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
