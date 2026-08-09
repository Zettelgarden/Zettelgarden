import React, { useState, useEffect } from 'react';
import { FactWithCard } from '../models/Fact';
import { Link } from 'react-router-dom';
import { CardIcon } from '../assets/icons/CardIcon';
import {
  getAllFacts,
  mergeFacts,
  deleteFact,
  FactsResponse,
} from '../api/facts';
import { Modal } from '../components/ui/Modal';
import { HeaderSection } from '../components/Header';
import { useDialogState } from '../contexts/DialogStateContext';
import { setDocumentTitle } from '../utils/title';
import { useAuth } from '../contexts/AuthContext';

export function FactPage() {
  const { hasSubscription } = useAuth();
  const [facts, setFacts] = useState<FactWithCard[]>([]);
  const [loading, setLoading] = useState(true);
  const [error, setError] = useState<string | null>(null);
  const [filterText, setFilterText] = useState('');
  const [searchTerm, setSearchTerm] = useState(''); // Actual search term sent to API
  const [sortBy, setSortBy] = useState<'created_at' | 'fact'>('created_at');
  const [sortDirection, setSortDirection] = useState<'asc' | 'desc'>('desc');
  const [currentPage, setCurrentPage] = useState(1);
  const [itemsPerPage] = useState(20);
  const [totalPages, setTotalPages] = useState(1);
  const [totalItems, setTotalItems] = useState(0);
  const [selectedFacts, setSelectedFacts] = useState<number[]>([]);
  const [selectionMode, setSelectionMode] = useState(false);

  const {
    setSelectedFact,
    setShowFactDialog,
    setSelectedEntity,
    setShowEntityDialog,
  } = useDialogState();

  const [showConfirmDialog, setShowConfirmDialog] = useState(false);
  const [isMerging, setIsMerging] = useState(false);

  const [showDeleteConfirm, setShowDeleteConfirm] = useState(false);
  const [isDeleting, setIsDeleting] = useState(false);

  const handleDeleteClick = () => {
    if (selectedFacts.length === 0) return;
    setShowDeleteConfirm(true);
  };

  const fetchFacts = async (
    page: number = currentPage,
    search: string = searchTerm,
  ) => {
    setLoading(true);
    try {
      const data = await getAllFacts(page, itemsPerPage, search);
      setFacts(data.facts);
      setCurrentPage(data.page);
      setTotalPages(data.total_pages);
      setTotalItems(data.total);
      setError(null);
    } catch (err) {
      setError('Failed to load facts');
      console.error('Error loading facts:', err);
    } finally {
      setLoading(false);
    }
  };

  const handleConfirmDelete = async () => {
    setIsDeleting(true);
    try {
      for (let id of selectedFacts) {
        await deleteFact(id);
      }
      await fetchFacts(currentPage, searchTerm);
      setSelectedFacts([]);
    } catch (err) {
      setError('Failed to delete facts');
      console.error('Error deleting facts:', err);
    } finally {
      setIsDeleting(false);
      setShowDeleteConfirm(false);
    }
  };

  const handleMergeClick = () => {
    if (selectedFacts.length < 2) return;
    setShowConfirmDialog(true);
  };

  const handleConfirmMerge = async () => {
    if (selectedFacts.length < 2) return;
    setShowConfirmDialog(false);
    setIsMerging(true);
    const baseFact = selectedFacts[0];

    try {
      // Merge all other facts into the first one
      for (let i = 1; i < selectedFacts.length; i++) {
        await mergeFacts(baseFact, selectedFacts[i]);
      }
      setSelectedFacts([]);
      await fetchFacts(currentPage, searchTerm);
    } catch (err) {
      setError('Failed to merge facts');
      console.error('Error merging facts:', err);
    } finally {
      setIsMerging(false);
    }
  };

  const handleSearch = () => {
    setSearchTerm(filterText);
    setCurrentPage(1);
  };

  const handleKeyPress = (e: React.KeyboardEvent) => {
    if (e.key === 'Enter') {
      handleSearch();
    }
  };

  useEffect(() => {
    setDocumentTitle('Facts');
    fetchFacts(1, '');
  }, []);

  useEffect(() => {
    fetchFacts(currentPage, searchTerm);
  }, [currentPage]);

  useEffect(() => {
    // Fetch when searchTerm changes (triggered by Enter press)
    setCurrentPage(1);
    fetchFacts(1, searchTerm);
  }, [searchTerm]);

  // For now, removing client-side sorting and filtering
  // These will be implemented as server-side features later
  const currentFacts = facts;

  if (loading) return <div className="p-4">Loading facts...</div>;
  if (error) return <div className="p-4 text-red-600">{error}</div>;

  const handleFactClick = (fact: FactWithCard, event: React.MouseEvent) => {
    if (selectionMode || event.ctrlKey || event.metaKey) {
      event.preventDefault();
      setSelectedFacts((prev) => {
        const idx = prev.indexOf(fact.id);
        if (idx !== -1) {
          return prev.filter((id) => id !== fact.id);
        } else {
          return [...prev, fact.id];
        }
      });
    } else {
      setSelectedFact(fact);
      setShowFactDialog(true);
    }
  };

  const toggleSelectionMode = () => {
    setSelectionMode(!selectionMode);
    if (selectionMode) setSelectedFacts([]);
  };

  return (
    <div className="p-4">
      <HeaderSection text="Facts" />

      {!hasSubscription && (
        <div className="text-center text-gray-500 mt-8">
          Automatic fact extraction is a Pro feature. You are currently viewing
          default facts.
          <br />
          <Link to="/app/subscribe" className="text-blue-500 hover:underline">
            Upgrade to Pro to automatically populate this page from your notes.
          </Link>
        </div>
      )}
      <div className="mb-4 flex gap-2 items-center">
        <button
          onClick={toggleSelectionMode}
          className={`px-4 py-2 rounded-lg text-sm font-medium transition-colors ${
            selectionMode
              ? 'bg-blue-500 text-white hover:bg-blue-600'
              : 'bg-white border border-gray-300 text-gray-700 hover:bg-gray-50'
          }`}
        >
          {selectionMode ? 'Exit Selection' : 'Select Mode'}
        </button>
        {selectionMode && selectedFacts.length > 0 && (
          <span className="text-sm text-gray-600">
            {selectedFacts.length} selected
          </span>
        )}
        {selectionMode && selectedFacts.length > 1 && (
          <button
            onClick={handleMergeClick}
            disabled={isMerging}
            className="px-4 py-2 bg-blue-500 text-white rounded hover:bg-blue-600 text-sm font-medium"
          >
            {isMerging ? 'Merging...' : 'Merge Selected'}
          </button>
        )}
        {selectionMode && selectedFacts.length > 0 && (
          <button
            onClick={handleDeleteClick}
            disabled={isDeleting}
            className="px-4 py-2 bg-red-500 text-white rounded hover:bg-red-600 text-sm font-medium"
          >
            {isDeleting ? 'Deleting...' : 'Delete Selected'}
          </button>
        )}
        <input
          type="text"
          placeholder="Search facts... (Press Enter to search)"
          value={filterText}
          onChange={(e) => setFilterText(e.target.value)}
          onKeyPress={handleKeyPress}
          className="px-3 py-2 border rounded w-64"
        />
        <button
          onClick={handleSearch}
          className="px-4 py-2 bg-blue-500 text-white rounded hover:bg-blue-600 text-sm font-medium"
        >
          Search
        </button>
        {/* Sorting temporarily disabled - will implement server-side sorting later */}
        {/* <select
                    value={sortBy}
                    onChange={(e) => setSortBy(e.target.value as any)}
                    className="px-3 py-2 border rounded"
                >
                    <option value="created_at">Sort by Created</option>
                    <option value="fact">Sort by Text</option>
                </select>
                <button
                    onClick={() =>
                        setSortDirection(sortDirection === "asc" ? "desc" : "asc")
                    }
                    className="px-3 py-2 border rounded"
                >
                    {sortDirection === "asc" ? "↑" : "↓"}
                </button> */}
      </div>

      <table className="min-w-full border divide-y divide-gray-200">
        <thead className="bg-gray-50">
          <tr>
            {selectionMode && (
              <th className="px-4 py-2">
                <input
                  type="checkbox"
                  checked={
                    selectedFacts.length === facts.length && facts.length > 0
                  }
                  onChange={(e) => {
                    if (e.target.checked) {
                      setSelectedFacts(facts.map((f) => f.id));
                    } else {
                      setSelectedFacts([]);
                    }
                  }}
                />
              </th>
            )}
            <th className="px-4 py-2 text-left text-sm font-medium text-gray-700">
              Fact
            </th>
            <th className="px-4 py-2 text-left text-sm font-medium text-gray-700">
              Card
            </th>
            <th className="px-4 py-2 text-left text-sm font-medium text-gray-700">
              Created At
            </th>
          </tr>
        </thead>
        <tbody className="divide-y divide-gray-200">
          {facts.map((f) => {
            const isSelected = selectedFacts.includes(f.id);
            return (
              <tr
                key={f.id}
                className={`hover:bg-gray-50 ${
                  isSelected ? 'bg-blue-100' : ''
                }`}
                onClick={(e) => handleFactClick(f, e)}
              >
                {selectionMode && (
                  <td className="px-4 py-2">
                    <input
                      type="checkbox"
                      checked={isSelected}
                      onChange={(e) => {
                        e.stopPropagation();
                        handleFactClick(f, e as any);
                      }}
                    />
                  </td>
                )}
                <td className="px-4 py-2 text-sm text-gray-800 cursor-pointer hover:underline">
                  {f.fact}
                </td>
                <td className="px-4 py-2 text-sm text-gray-800">
                  {f.card && (
                    <Link
                      to={`/app/card/${f.card.id}`}
                      onClick={(e) => e.stopPropagation()}
                      className="inline-flex items-center text-sm text-blue-600 hover:text-blue-800 hover:underline"
                    >
                      <div className="w-3 h-3 mr-1 text-gray-400">
                        <CardIcon />
                      </div>
                      [{f.card.card_id}] {f.card.title}
                    </Link>
                  )}
                </td>
                <td className="px-4 py-2 text-sm text-gray-800">
                  {new Date(f.created_at).toLocaleString()}
                </td>
              </tr>
            );
          })}
        </tbody>
      </table>

      {facts.length === 0 && !loading && (
        <div className="text-center text-gray-500 mt-8">No facts found</div>
      )}

      {totalItems > 0 && (
        <div className="flex justify-center gap-4 mt-4">
          <button
            onClick={() => setCurrentPage(currentPage - 1)}
            disabled={currentPage === 1}
            className="px-4 py-2 text-sm font-medium text-gray-700 bg-white border rounded disabled:opacity-50"
          >
            Previous
          </button>
          <span className="flex items-center">
            Page {currentPage} of {totalPages} ({totalItems} total facts)
          </span>
          <button
            onClick={() => setCurrentPage(currentPage + 1)}
            disabled={currentPage === totalPages}
            className="px-4 py-2 text-sm font-medium text-gray-700 bg-white border rounded disabled:opacity-50"
          >
            Next
          </button>
        </div>
      )}
      {showConfirmDialog && (
        <Modal
          open={showConfirmDialog}
          onClose={() => setShowConfirmDialog(false)}
          size="md"
          dialogClassName="z-50"
        >
          <h2 className="text-lg font-semibold mb-4">Confirm Merge</h2>
          <div className="mb-4">
            <p className="font-medium text-green-600 mb-2">
              Primary Fact (will be kept):
              <br />
              {facts.find((f) => f.id === selectedFacts[0])?.fact}
            </p>
            <p className="text-gray-600 mb-2">
              The following facts will be merged into the primary:
            </p>
            <ul className="list-disc pl-5">
              {selectedFacts.slice(1).map((id) => {
                const fact = facts.find((f) => f.id === id);
                return fact ? (
                  <li key={id} className="text-gray-700">
                    {fact.fact}
                  </li>
                ) : null;
              })}
            </ul>
          </div>
          <p className="text-red-600 text-sm mb-4">
            This action cannot be undone. The merged facts will be deleted.
          </p>
          <div className="flex justify-end gap-4">
            <button
              onClick={() => setShowConfirmDialog(false)}
              className="px-4 py-2 text-gray-600 hover:text-gray-800"
            >
              Cancel
            </button>
            <button
              onClick={handleConfirmMerge}
              className="px-4 py-2 bg-blue-500 text-white rounded hover:bg-blue-600"
            >
              Merge
            </button>
          </div>
        </Modal>
      )}

      {showDeleteConfirm && (
        <Modal
          open={showDeleteConfirm}
          onClose={() => setShowDeleteConfirm(false)}
          size="md"
          dialogClassName="z-50"
        >
          <h2 className="text-lg font-semibold mb-4">Confirm Delete</h2>
          <p className="text-gray-700 mb-4">
            Are you sure you want to delete {selectedFacts.length} fact(s)? This
            action cannot be undone.
          </p>
          <div className="flex justify-end gap-4">
            <button
              onClick={() => setShowDeleteConfirm(false)}
              className="px-4 py-2 text-gray-600 hover:text-gray-800"
            >
              Cancel
            </button>
            <button
              onClick={handleConfirmDelete}
              className="px-4 py-2 bg-red-500 text-white rounded hover:bg-red-600"
            >
              Delete
            </button>
          </div>
        </Modal>
      )}
    </div>
  );
}
