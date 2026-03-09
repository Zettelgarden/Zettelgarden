import React, { ChangeEvent } from "react";
import { MobileBottomSheet } from "../layout/MobileBottomSheet";
import { Button } from "../Button";

interface TaskFiltersSheetProps {
  isOpen: boolean;
  onClose: () => void;
  // Filter settings
  dateView: string;
  viewMode: "list" | "matrix" | "kanban" | "calendar";
  showCompleted: boolean;
  sortField: "updated_at" | "title" | "priority" | "status" | "id" | "scheduled_date" | "due_date";
  sortDirection: "asc" | "desc";
  selectMode: boolean;
  // Handlers
  onDateViewChange: (value: string) => void;
  onViewModeChange: (value: "list" | "matrix" | "kanban" | "calendar") => void;
  onShowCompletedChange: () => void;
  onSortFieldChange: (value: "updated_at" | "title" | "priority" | "status" | "id" | "scheduled_date" | "due_date") => void;
  onSortDirectionToggle: () => void;
  onSelectModeToggle: () => void;
  onApply?: () => void;
}

export function TaskFiltersSheet({
  isOpen,
  onClose,
  dateView,
  viewMode,
  showCompleted,
  sortField,
  sortDirection,
  selectMode,
  onDateViewChange,
  onViewModeChange,
  onShowCompletedChange,
  onSortFieldChange,
  onSortDirectionToggle,
  onSelectModeToggle,
  onApply,
}: TaskFiltersSheetProps) {
  const handleDateViewChange = (e: ChangeEvent<HTMLSelectElement>) => {
    onDateViewChange(e.target.value);
  };

  const handleViewModeChange = (e: ChangeEvent<HTMLSelectElement>) => {
    onViewModeChange(e.target.value as "list" | "matrix" | "kanban" | "calendar");
  };

  const handleSortFieldChange = (e: ChangeEvent<HTMLSelectElement>) => {
    onSortFieldChange(e.target.value as "updated_at" | "title" | "priority" | "status" | "id" | "scheduled_date" | "due_date");
  };

  const handleApply = () => {
    if (onApply) {
      onApply();
    }
    onClose();
  };

  const handleShowCompletedClick = () => {
    onShowCompletedChange();
  };

  const handleSelectModeClick = () => {
    onSelectModeToggle();
  };

  const handleSortDirectionClick = () => {
    onSortDirectionToggle();
  };

  return (
    <MobileBottomSheet
      isOpen={isOpen}
      onClose={onClose}
      title="Display & Filters"
      showCloseButton={true}
      maxHeight="85vh"
    >
      <div className="p-4 space-y-5">
        {/* Date Range Section */}
        <section>
          <label className="block text-sm font-semibold text-gray-900 mb-2">
            Date Range
          </label>
          <select
            className="w-full p-3 border border-gray-300 rounded-lg text-base bg-white focus:ring-2 focus:ring-blue-500 focus:border-blue-500 min-h-[48px]"
            value={dateView}
            onChange={handleDateViewChange}
          >
            <option value="today">Today</option>
            <option value="tomorrow">Tomorrow</option>
            <option value="this_week">This Week</option>
            <option value="overdue">Overdue</option>
            <option value="no_date">No Date</option>
            <option value="all">All</option>
          </select>
        </section>

        {/* View Mode Section */}
        <section>
          <label className="block text-sm font-semibold text-gray-900 mb-2">
            View Mode
          </label>
          <select
            className="w-full p-3 border border-gray-300 rounded-lg text-base bg-white focus:ring-2 focus:ring-blue-500 focus:border-blue-500 min-h-[48px]"
            value={viewMode}
            onChange={handleViewModeChange}
          >
            <option value="list">List View</option>
            <option value="matrix">Eisenhower Matrix</option>
            <option value="kanban">Kanban Board</option>
            <option value="calendar">Calendar View</option>
          </select>
        </section>

        {/* Toggles Section */}
        <section className="space-y-3">
          <label className="block text-sm font-semibold text-gray-900">
            Options
          </label>

          {/* Show Completed Toggle */}
          <button
            type="button"
            onClick={handleShowCompletedClick}
            className="w-full p-4 border border-gray-300 rounded-lg text-left flex items-center justify-between bg-white hover:bg-gray-50 active:bg-gray-100 min-h-[56px] transition-colors"
          >
            <span className="text-base text-gray-900">Show Completed Tasks</span>
            <div className={`w-12 h-7 rounded-full transition-colors flex items-center px-1 ${showCompleted ? 'bg-blue-600' : 'bg-gray-300'}`}>
              <div className={`w-5 h-5 bg-white rounded-full shadow transition-transform ${showCompleted ? 'translate-x-5' : 'translate-x-0'}`} />
            </div>
          </button>

          {/* Select Mode Toggle */}
          <button
            type="button"
            onClick={handleSelectModeClick}
            className="w-full p-4 border border-gray-300 rounded-lg text-left flex items-center justify-between bg-white hover:bg-gray-50 active:bg-gray-100 min-h-[56px] transition-colors"
          >
            <span className="text-base text-gray-900">Select Mode</span>
            <div className={`w-12 h-7 rounded-full transition-colors flex items-center px-1 ${selectMode ? 'bg-blue-600' : 'bg-gray-300'}`}>
              <div className={`w-5 h-5 bg-white rounded-full shadow transition-transform ${selectMode ? 'translate-x-5' : 'translate-x-0'}`} />
            </div>
          </button>
        </section>

        {/* Sort Section */}
        <section>
          <label className="block text-sm font-semibold text-gray-900 mb-2">
            Sort By
          </label>
          <div className="flex items-center gap-2">
            <select
              className="flex-grow p-3 border border-gray-300 rounded-lg text-base bg-white focus:ring-2 focus:ring-blue-500 focus:border-blue-500 min-h-[48px]"
              value={sortField}
              onChange={handleSortFieldChange}
            >
              <option value="updated_at">Updated</option>
              <option value="title">Name</option>
              <option value="priority">Priority</option>
              <option value="status">Status</option>
              <option value="scheduled_date">Scheduled Date</option>
              <option value="due_date">Due Date</option>
              <option value="id">ID</option>
            </select>
            <button
              type="button"
              onClick={handleSortDirectionClick}
              className="p-3 border border-gray-300 rounded-lg bg-white hover:bg-gray-50 active:bg-gray-100 min-h-[48px] min-w-[48px] flex items-center justify-center transition-colors"
              aria-label={`Sort ${sortDirection === 'asc' ? 'ascending' : 'descending'}`}
            >
              <span className="text-xl font-semibold text-gray-700">
                {sortDirection === "asc" ? "↑" : "↓"}
              </span>
            </button>
          </div>
        </section>

        {/* Action Buttons */}
        <div className="flex gap-3 pt-2">
          <Button
            onClick={onClose}
            variant="outline"
            className="flex-1 min-h-[48px] text-base"
          >
            Cancel
          </Button>
          <Button
            onClick={handleApply}
            variant="primary"
            className="flex-1 min-h-[48px] text-base"
          >
            Apply
          </Button>
        </div>
      </div>
    </MobileBottomSheet>
  );
}
