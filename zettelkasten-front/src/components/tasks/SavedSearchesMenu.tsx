import React, { useEffect, useRef, useState } from 'react';
import type { SortField, SortDirection, ViewMode } from '../../types/taskPage';
import type { TaskSavedSearch } from '../../models/TaskSavedSearch';
import { useTaskSavedSearches } from '../../hooks/useTaskSavedSearches';
import { useToast } from '../toast/ToastContext';

export interface SavedSearchesMenuProps {
  /** Current page state, used to snapshot when saving and to detect the active search. */
  filterString: string;
  sortField: SortField;
  sortDirection: SortDirection;
  viewMode: ViewMode;
  /** Apply a saved search to the page. */
  onApply: (search: TaskSavedSearch) => void;
}

function matchesCurrent(
  search: TaskSavedSearch,
  filterString: string,
  sortField: SortField,
  sortDirection: SortDirection,
  viewMode: ViewMode,
): boolean {
  return (
    search.filter_string === filterString &&
    search.sort_field === sortField &&
    search.sort_direction === sortDirection &&
    search.view_mode === viewMode
  );
}

/**
 * Dropdown that lists a user's saved task searches, lets them apply one,
 * save the current filter+sort+view as a new search, update an applied
 * search, or delete one. Data is backend-synced via useTaskSavedSearches.
 */
export function SavedSearchesMenu({
  filterString,
  sortField,
  sortDirection,
  viewMode,
  onApply,
}: SavedSearchesMenuProps) {
  const {
    searches = [],
    isLoading,
    isError,
    create,
    update,
    remove,
  } = useTaskSavedSearches();
  const { showToast } = useToast();

  const [open, setOpen] = useState(false);
  const [saving, setSaving] = useState(false);
  const [busy, setBusy] = useState(false);
  const [name, setName] = useState('');
  /** The id of the search most recently applied; drives the "Update" affordance. */
  const [appliedId, setAppliedId] = useState<number | null>(null);
  const nameInputRef = useRef<HTMLInputElement>(null);

  const activeSearch = searches.find((s) =>
    matchesCurrent(s, filterString, sortField, sortDirection, viewMode),
  );
  const appliedSearch =
    appliedId != null ? searches.find((s) => s.id === appliedId) : undefined;
  const isDirty =
    !!appliedSearch &&
    !matchesCurrent(
      appliedSearch,
      filterString,
      sortField,
      sortDirection,
      viewMode,
    );

  useEffect(() => {
    if (saving) nameInputRef.current?.focus();
  }, [saving]);

  function handleApply(search: TaskSavedSearch) {
    onApply(search);
    setAppliedId(search.id);
    setOpen(false);
  }

  function startSave() {
    setName(filterString.trim() || 'Untitled search');
    setSaving(true);
  }

  function confirmSave() {
    const trimmed = name.trim();
    if (!trimmed) {
      showToast('error', 'Name required', 'Enter a name for this search');
      return;
    }
    setBusy(true);
    create({
      name: trimmed,
      filter_string: filterString,
      sort_field: sortField,
      sort_direction: sortDirection,
      view_mode: viewMode,
    })
      .then(() => {
        showToast(
          'success',
          'Search saved',
          `"${trimmed}" is now available on all your devices`,
        );
        setSaving(false);
      })
      .catch((err: Error) => showToast('error', 'Save failed', err.message))
      .finally(() => setBusy(false));
  }

  function handleUpdate() {
    if (!appliedSearch) return;
    setBusy(true);
    update(appliedSearch.id, {
      filter_string: filterString,
      sort_field: sortField,
      sort_direction: sortDirection,
      view_mode: viewMode,
    })
      .then(() => {
        showToast(
          'success',
          'Search updated',
          `"${appliedSearch.name}" updated`,
        );
        setOpen(false);
      })
      .catch((err: Error) => showToast('error', 'Update failed', err.message))
      .finally(() => setBusy(false));
  }

  function handleDelete(search: TaskSavedSearch) {
    setBusy(true);
    remove(search.id)
      .then(() => {
        if (appliedId === search.id) setAppliedId(null);
        showToast('success', 'Search deleted', `"${search.name}" removed`);
      })
      .catch((err: Error) => showToast('error', 'Delete failed', err.message))
      .finally(() => setBusy(false));
  }

  return (
    <div className="relative">
      <button
        type="button"
        onClick={() => setOpen((o) => !o)}
        className="h-9 inline-flex items-center gap-1 px-2.5 border border-slate-300 rounded-md text-sm bg-white hover:bg-slate-50 text-slate-600 transition-colors"
        title="Saved searches"
        aria-haspopup="menu"
        aria-expanded={open}
      >
        <svg
          xmlns="http://www.w3.org/2000/svg"
          viewBox="0 0 20 20"
          fill="currentColor"
          className={`w-4 h-4 ${
            activeSearch ? 'text-blue-600' : 'text-slate-400'
          }`}
        >
          <path
            fillRule="evenodd"
            d="M5.5 3A1.5 1.5 0 004 4.5v13a.75.75 0 001.14.643L10 15.341l4.86 2.802A.75.75 0 0016 17.5v-13A1.5 1.5 0 0014.5 3h-9z"
            clipRule="evenodd"
          />
        </svg>
        <span className="hidden lg:inline">
          {activeSearch ? activeSearch.name : 'Saved'}
        </span>
      </button>

      {open && (
        <>
          <div className="fixed inset-0 z-20" onClick={() => setOpen(false)} />
          <div className="absolute top-full right-0 mt-2 bg-white border border-slate-200 rounded-lg shadow-xl z-30 w-72">
            <div className="flex items-center justify-between px-3 py-2 border-b border-slate-100">
              <h4 className="font-semibold text-sm text-slate-800">
                Saved searches
              </h4>
              <button
                onClick={() => setOpen(false)}
                className="text-slate-400 hover:text-slate-600"
                aria-label="Close"
              >
                <svg
                  xmlns="http://www.w3.org/2000/svg"
                  viewBox="0 0 20 20"
                  fill="currentColor"
                  className="w-4 h-4"
                >
                  <path d="M6.28 5.22a.75.75 0 00-1.06 1.06L8.94 10l-3.72 3.72a.75.75 0 101.06 1.06L10 11.06l3.72 3.72a.75.75 0 101.06-1.06L11.06 10l3.72-3.72a.75.75 0 00-1.06-1.06L10 8.94 6.28 5.22z" />
                </svg>
              </button>
            </div>

            <div className="max-h-72 overflow-y-auto">
              {isLoading ? (
                <p className="px-3 py-3 text-sm text-slate-400">Loading…</p>
              ) : isError ? (
                <p className="px-3 py-3 text-sm text-red-500">
                  Failed to load searches
                </p>
              ) : searches.length === 0 ? (
                <p className="px-3 py-3 text-sm text-slate-400">
                  No saved searches yet. Save one to recall it on any device.
                </p>
              ) : (
                <ul className="py-1">
                  {searches.map((search) => {
                    const isActive = activeSearch?.id === search.id;
                    return (
                      <li key={search.id}>
                        <div className="group flex items-center px-2">
                          <button
                            onClick={() => handleApply(search)}
                            className={`flex-grow text-left px-2 py-1.5 rounded text-sm truncate ${
                              isActive
                                ? 'bg-blue-50 text-blue-700 font-medium'
                                : 'text-slate-700 hover:bg-slate-50'
                            }`}
                            title={search.filter_string || '(no filter)'}
                          >
                            {search.name}
                            {search.filter_string && (
                              <span className="block text-xs text-slate-400 font-normal truncate">
                                {search.filter_string}
                              </span>
                            )}
                          </button>
                          <button
                            onClick={() => handleDelete(search)}
                            disabled={busy}
                            className="ml-1 text-slate-300 hover:text-red-500 opacity-0 group-hover:opacity-100 transition-opacity px-1 disabled:opacity-50"
                            aria-label={`Delete "${search.name}"`}
                            title="Delete search"
                          >
                            <svg
                              xmlns="http://www.w3.org/2000/svg"
                              viewBox="0 0 20 20"
                              fill="currentColor"
                              className="w-4 h-4"
                            >
                              <path d="M6.28 5.22a.75.75 0 00-1.06 1.06L8.94 10l-3.72 3.72a.75.75 0 101.06 1.06L10 11.06l3.72 3.72a.75.75 0 101.06-1.06L11.06 10l3.72-3.72a.75.75 0 00-1.06-1.06L10 8.94 6.28 5.22z" />
                            </svg>
                          </button>
                        </div>
                      </li>
                    );
                  })}
                </ul>
              )}
            </div>

            <div className="border-t border-slate-100 p-2 space-y-2">
              {isDirty && appliedSearch && (
                <button
                  onClick={handleUpdate}
                  disabled={busy}
                  className="w-full text-left px-2 py-1.5 rounded text-sm text-amber-700 hover:bg-amber-50 disabled:opacity-50"
                  title="Overwrite this search with the current filter"
                >
                  Update “{appliedSearch.name}”
                </button>
              )}

              {saving ? (
                <div className="space-y-2">
                  {/* Preview exactly what will be saved, so the name can't be
                      confused with the filter (a common mistake). */}
                  <div className="bg-slate-50 rounded p-2 text-xs text-slate-600 space-y-0.5">
                    <div className="flex gap-1">
                      <span className="text-slate-400 shrink-0">Filter:</span>
                      <span className="font-mono break-all">
                        {filterString.trim() || (
                          <span className="text-slate-400 italic">
                            (no filter — saves all tasks)
                          </span>
                        )}
                      </span>
                    </div>
                    <div className="flex gap-1">
                      <span className="text-slate-400 shrink-0">Sort:</span>
                      <span>
                        {sortField} ({sortDirection})
                      </span>
                      <span className="text-slate-400 ml-2">View:</span>
                      <span>{viewMode}</span>
                    </div>
                  </div>
                  <div className="flex items-center gap-2">
                    <input
                      ref={nameInputRef}
                      type="text"
                      value={name}
                      onChange={(e) => setName(e.target.value)}
                      onKeyDown={(e) => {
                        if (e.key === 'Enter') confirmSave();
                        if (e.key === 'Escape') setSaving(false);
                      }}
                      placeholder="Search name"
                      className="flex-grow h-8 px-2 border border-slate-300 rounded text-sm focus:outline-none focus:ring-1 focus:ring-blue-400"
                    />
                    <button
                      onClick={confirmSave}
                      disabled={busy}
                      className="h-8 px-2 bg-blue-600 text-white rounded text-sm hover:bg-blue-700 disabled:opacity-50"
                    >
                      Save
                    </button>
                  </div>
                </div>
              ) : (
                <button
                  onClick={startSave}
                  className="w-full text-left px-2 py-1.5 rounded text-sm text-blue-700 hover:bg-blue-50 flex items-center gap-1.5"
                >
                  <svg
                    xmlns="http://www.w3.org/2000/svg"
                    viewBox="0 0 20 20"
                    fill="currentColor"
                    className="w-4 h-4"
                  >
                    <path d="M10.75 4.75a.75.75 0 00-1.5 0v4.5h-4.5a.75.75 0 000 1.5h4.5v4.5a.75.75 0 001.5 0v-4.5h4.5a.75.75 0 000-1.5h-4.5v-4.5z" />
                  </svg>
                  Save current search
                </button>
              )}
            </div>
          </div>
        </>
      )}
    </div>
  );
}
