import React, { ChangeEvent, useEffect, useRef, useState } from "react";
import { useNavigate } from "react-router-dom";
import { Task } from "../../models/Task";
import { Tag } from "../../models/Tags";
import { ExternalEvent } from "../../models/ExternalEvent";
import { MobileTopBar } from "../layout/MobileTopBar";
import { TaskFiltersSheet } from "./TaskFiltersSheet";
import { QuickTagPopover, type QuickTagTrigger, getQuickTagTrigger, applyQuickTagSelection } from "./QuickTagPopover";
import { SearchTagDropdown } from "../tags/SearchTagDropdown";
import { TaskList } from "./TaskList";
import { TaskListSkeleton } from "./TaskListSkeleton";
import { TaskEmptyState, getEmptyStateType } from "./TaskEmptyState";
import { FilterHelpButton, FilterHelpPopover } from "./FilterHelpButton";
import { EisenhowerMatrix } from "./EisenhowerMatrix";
import { KanbanBoard } from "./KanbanBoard";
import { CalendarViewWrapper } from "../../components/calendar/CalendarView";
import { TaskSelectionOverlay } from "./TaskSelectionOverlay";
import { TaskDialog } from "./TaskDialog";
import { CreateTaskWindow } from "./CreateTaskWindow";

type SortField = "updated_at" | "title" | "priority" | "status" | "id" | "scheduled_date";
type SortDirection = "asc" | "desc";
type ViewMode = "list" | "matrix" | "kanban" | "calendar";
type TaskMobileView = 'list' | 'filters';

interface TaskMobileLayoutProps {
  // Mobile view state
  mobileView: TaskMobileView;
  setMobileView: (view: TaskMobileView) => void;

  // Task data
  tasks: Task[];
  tasksToDisplay: Task[];
  paginatedTasks: Task[];
  totalTasksForDateView: number;
  totalPages: number;
  tags: Tag[];
  userTimezone: string;
  isLoading?: boolean;

  // Filter and settings state
  dateView: string;
  viewMode: ViewMode;
  showCompleted: boolean;
  filterString: string;
  sortField: SortField;
  sortDirection: SortDirection;
  calendarViewMode: "month" | "week";
  calendarCurrentDate: Date;
  currentPage: number;
  itemsPerPage: number;
  showFilterHelp: boolean;
  showDisplayMenu?: boolean;
  selectMode: boolean;
  selectedTaskIds: Set<number>;

  // Calendar events
  externalEvents: ExternalEvent[];
  isLoadingEvents: boolean;

  // Dialog states
  showCreateTaskWindow: boolean;
  selectedTaskId: number | null;
  isTaskDialogOpen: boolean;
  createTaskStatus: string | undefined;
  calendarSelectedDate: Date | null;

  // Setters
  setDateView: (view: string) => void;
  setViewMode: (mode: ViewMode) => void;
  setShowCompleted: (show: boolean) => void;
  setFilterString: (filter: string) => void;
  setSortField: (field: SortField) => void;
  setSortDirection: (direction: SortDirection) => void;
  setCurrentPage: (page: number) => void;
  setItemsPerPage: (items: number) => void;
  setShowFilterHelp: (show: boolean) => void;
  setSelectMode: (mode: boolean) => void;
  setSelectedTaskIds: (ids: Set<number>) => void;
  setCalendarViewMode: (mode: "month" | "week") => void;
  setCalendarCurrentDate: (date: Date) => void;
  setShowCreateTaskWindow: (show: boolean) => void;
  setSelectedTaskId: (taskId: number | null) => void;
  setIsTaskDialogOpen: (open: boolean) => void;
  setCreateTaskStatus: (status: string | undefined) => void;
  setCalendarSelectedDate: (date: Date | null) => void;
  setExternalEvents: (events: ExternalEvent[]) => void;
  setIsLoadingEvents: (loading: boolean) => void;
  setRefreshTasks: (refresh: boolean) => void;
  setShowDisplayMenu?: (show: boolean) => void;

  // Actions
  toggleSortDirection: () => void;
  toggleSelectMode: () => void;
  toggleTaskSelection: (taskId: number) => void;
  selectAllTasks: (taskIds: number[]) => void;
  clearSelection: () => void;
  navigateCalendar: (direction: "prev" | "next" | "today") => void;

  // Handlers
  onMenuClick: () => void;
  onTagClick: (tag: string) => void;
  onAddTaskWithStatus: (status: string) => void;
  onCloseTaskDialog: () => void;
  onTaskClick: (taskId: number) => void;

  // Filter handlers
  onFilterChange: (e: ChangeEvent<HTMLInputElement | HTMLTextAreaElement>) => void;
  onRefreshFilterTriggerFromInput: (input: HTMLInputElement) => void;
  onSelectQuickTag: (tagName: string) => void;
  filterInputRef: React.RefObject<HTMLInputElement>;
  cursorPosition: number;
  filterTrigger: QuickTagTrigger | null;
  setFilterTrigger: (trigger: QuickTagTrigger | null) => void;
  isFilterFocused: boolean;
  setIsFilterFocused: (focused: boolean) => void;
}

/**
 * Mobile layout wrapper for the Tasks page.
 * Handles mobile-specific navigation, filtering, and view rendering.
 *
 * Features:
 * - MobileTopBar with hamburger menu and filter button
 * - Filters bottom sheet with display options
 * - Responsive task list/content rendering
 * - Quick tag autocomplete in filter input
 * - Smooth transitions between list and filter views
 */
export function TaskMobileLayout({
  mobileView,
  setMobileView,
  tasks,
  tasksToDisplay,
  paginatedTasks,
  totalTasksForDateView,
  tags,
  userTimezone,
  isLoading = false,
  dateView,
  viewMode,
  showCompleted,
  filterString,
  sortField,
  sortDirection,
  calendarViewMode,
  calendarCurrentDate,
  currentPage,
  itemsPerPage,
  showFilterHelp,
  selectMode,
  selectedTaskIds,
  externalEvents,
  isLoadingEvents,
  showCreateTaskWindow,
  selectedTaskId,
  isTaskDialogOpen,
  createTaskStatus,
  calendarSelectedDate,
  setDateView,
  setViewMode,
  setShowCompleted,
  setSortField,
  setSortDirection,
  setCurrentPage,
  setItemsPerPage,
  setShowFilterHelp,
  setSelectMode,
  setSelectedTaskIds,
  setCalendarViewMode,
  setCalendarCurrentDate,
  setFilterString,
  setShowCreateTaskWindow,
  setSelectedTaskId,
  setIsTaskDialogOpen,
  setCreateTaskStatus,
  setCalendarSelectedDate,
  setExternalEvents,
  setIsLoadingEvents,
  setRefreshTasks,
  toggleSortDirection,
  toggleSelectMode,
  toggleTaskSelection,
  selectAllTasks,
  clearSelection,
  navigateCalendar,
  onMenuClick,
  onTagClick,
  onAddTaskWithStatus,
  onCloseTaskDialog,
  onTaskClick,
  onFilterChange,
  onRefreshFilterTriggerFromInput,
  onSelectQuickTag,
  filterInputRef,
  cursorPosition,
  filterTrigger,
  setFilterTrigger,
  isFilterFocused,
  setIsFilterFocused,
}: TaskMobileLayoutProps) {
  const navigate = useNavigate();

  // Local state for CreateTaskWindow - it uses currentCard and setShowTaskWindow pattern
  const [currentCard, setCurrentCard] = useState<null>(null);

  // Handle filter input changes (delegated from parent)
  const handleFilterChange = (e: ChangeEvent<HTMLInputElement | HTMLTextAreaElement>) => {
    onFilterChange(e);
  };

  // Refresh filter trigger from input
  const refreshFilterTriggerFromInput = (input: HTMLInputElement) => {
    onRefreshFilterTriggerFromInput(input);
  };

  // Handle quick tag selection
  const handleSelectQuickTag = (selectedTagName: string) => {
    onSelectQuickTag(selectedTagName);
  };

  // Handle add task click
  const handleAddTaskClick = () => {
    setCreateTaskStatus(undefined);
    setCalendarSelectedDate(null);
    setShowCreateTaskWindow(true);
  };

  // Handle task click for opening task dialog
  const handleTaskClick = (taskId: number) => {
    setSelectedTaskId(taskId);
    setIsTaskDialogOpen(true);
  };

  // Handle create task from calendar date
  const handleCreateTaskFromDate = (date: Date) => {
    setCalendarSelectedDate(date);
    setShowCreateTaskWindow(true);
  };

  // Handle task moved (for calendar drag and drop)
  const handleTaskMoved = () => {
    setRefreshTasks(true);
  };

  // Handle external event change
  const handleExternalEventChange = () => {
    setRefreshTasks(true);
  };

  // Render current view based on viewMode
  const renderCurrentView = () => {
    switch (viewMode) {
      case "calendar":
        return (
          <CalendarViewWrapper
            tasks={tasksToDisplay}
            externalEvents={externalEvents}
            currentDate={calendarCurrentDate}
            viewMode={calendarViewMode}
            onNavigate={navigateCalendar}
            onViewModeChange={setCalendarViewMode}
            onTaskClick={handleTaskClick}
            onCreateTask={handleCreateTaskFromDate}
            onTaskMoved={handleTaskMoved}
            onExternalEventChange={handleExternalEventChange}
            timezone={userTimezone}
          />
        );
      case "list":
        if (isLoading) {
          return <TaskListSkeleton count={6} />;
        }
        if (paginatedTasks.length === 0) {
          return (
            <TaskEmptyState
              type={getEmptyStateType({
                totalTasks: tasks.length,
                filteredTasks: tasksToDisplay.length,
                hasActiveFilter: filterString.trim().length > 0,
                showCompleted,
              }) || "no-tasks"}
              onAddTask={() => {
                setCreateTaskStatus(undefined);
                setCalendarSelectedDate(null);
                setShowCreateTaskWindow(true);
              }}
              onClearFilters={() => setFilterString("")}
              onShowCompleted={() => setShowCompleted(true)}
            />
          );
        }
        return (
          <TaskList
            tasks={paginatedTasks}
            onTagClick={onTagClick}
            hideMatrixTags={false}
            selectMode={selectMode}
            selectedTaskIds={selectedTaskIds}
            onTaskSelect={toggleTaskSelection}
          />
        );
      case "kanban":
        return (
          <KanbanBoard
            tasks={tasksToDisplay}
            onTagClick={onTagClick}
            onAddTaskWithStatus={onAddTaskWithStatus}
            selectMode={selectMode}
            selectedTaskIds={selectedTaskIds}
            onTaskSelect={toggleTaskSelection}
          />
        );
      case "matrix":
        return (
          <EisenhowerMatrix
            tasks={tasksToDisplay}
            onTagClick={onTagClick}
            onAddTaskWithTags={(tags) => {
              // For matrix view, we'll just use the first tag's value for status
              // or fall back to the status handler
              onAddTaskWithStatus(tags[0] || "todo");
            }}
            selectMode={selectMode}
            selectedTaskIds={selectedTaskIds}
            onTaskSelect={toggleTaskSelection}
          />
        );
      default:
        return (
          <TaskList
            tasks={paginatedTasks}
            onTagClick={onTagClick}
            hideMatrixTags={false}
            selectMode={selectMode}
            selectedTaskIds={selectedTaskIds}
            onTaskSelect={toggleTaskSelection}
          />
        );
    }
  };

  // Mobile List View
  if (mobileView === 'list') {
    return (
      <div className="flex flex-col flex-1 h-full">
        {/* Mobile Top Bar */}
        <MobileTopBar
          title="Tasks"
          badge={selectMode ? selectedTaskIds.size : undefined}
          onMenuClick={onMenuClick}
          actions={
            <div className="flex items-center gap-1">
              {/* Filter button */}
              <button
                onClick={() => setMobileView('filters')}
                className="p-2 -mr-2 text-gray-600 hover:text-gray-900 hover:bg-gray-100 rounded-lg"
                aria-label="Open filters"
              >
                <svg className="w-6 h-6" fill="none" viewBox="0 0 24 24" stroke="currentColor">
                  <path strokeLinecap="round" strokeLinejoin="round" strokeWidth={2} d="M3 4a1 1 0 011-1h16a1 1 0 011 1v2.586a1 1 0 01-.293.707l-6.414 6.414a1 1 0 00-.293.707V17l-4 4v-6.586a1 1 0 00-.293-.707L3.293 7.293A1 1 0 013 6.586V4z" />
                </svg>
              </button>
              {/* Add task button */}
              <button
                onClick={handleAddTaskClick}
                className="p-2 -mr-2 text-blue-600 hover:text-blue-700 hover:bg-blue-50 rounded-lg"
                aria-label="Add task"
              >
                <svg className="w-6 h-6" fill="none" viewBox="0 0 24 24" stroke="currentColor">
                  <path strokeLinecap="round" strokeLinejoin="round" strokeWidth={2} d="M12 4v16m8-8H4" />
                </svg>
              </button>
            </div>
          }
        />

        {/* Filter Input Section */}
        <div className="bg-slate-100 p-3 border-b border-slate-300">
          <div className="flex items-center gap-2">
            <div className="relative flex-grow">
              <input
                ref={filterInputRef}
                type="text"
                value={filterString}
                onChange={handleFilterChange}
                onKeyUp={(e) => refreshFilterTriggerFromInput(e.currentTarget)}
                onClick={(e) => refreshFilterTriggerFromInput(e.currentTarget)}
                onFocus={(e) => {
                  setIsFilterFocused(true);
                  refreshFilterTriggerFromInput(e.currentTarget);
                }}
                onBlur={() => setIsFilterFocused(false)}
                placeholder="Filter... try #tag or status:todo"
                className="h-9 w-full pl-3 pr-8 border border-slate-300 rounded-md text-sm"
              />
              <FilterHelpButton
                showHelp={showFilterHelp}
                onToggle={() => setShowFilterHelp(!showFilterHelp)}
              />
              <FilterHelpPopover
                visible={showFilterHelp}
                onClose={() => setShowFilterHelp(false)}
              />
            </div>

            <QuickTagPopover
              open={Boolean(filterTrigger && isFilterFocused)}
              anchorInputRef={filterInputRef}
              titleValue={filterString}
              cursorPosition={cursorPosition}
              trigger={filterTrigger}
              onSelectTag={handleSelectQuickTag}
              onRequestClose={() => setFilterTrigger(null)}
            />
            <div className="h-9 flex items-center">
              <SearchTagDropdown
                tags={tags}
                handleTagClick={onTagClick}
              />
            </div>
          </div>

          {/* Task count badge */}
          <div className="mt-2 flex items-center gap-2">
            <span className="bg-slate-200 text-slate-700 px-2 py-0.5 rounded-full text-xs whitespace-nowrap">
              {tasksToDisplay.length}/{totalTasksForDateView}
              {dateView === "today"
                ? " today"
                : dateView === "tomorrow"
                  ? " tomorrow"
                  : dateView === "overdue"
                    ? " overdue"
                    : dateView === "this_week"
                      ? " this week"
                      : dateView === "no_date"
                        ? " no date"
                        : ""} {totalTasksForDateView === 1 ? "task" : "tasks"}
            </span>
            {selectMode && (
              <span className="bg-blue-100 text-blue-700 px-2 py-0.5 rounded-full text-xs whitespace-nowrap">
                {selectedTaskIds.size} selected
              </span>
            )}
          </div>
        </div>

        {/* Task content area */}
        <div className="flex-1 bg-white flex flex-col overflow-y-auto">
          {renderCurrentView()}
        </div>

        {/* Modals and overlays */}
        <TaskSelectionOverlay
          tasks={tasksToDisplay}
          selectMode={selectMode}
          selectedTaskIds={selectedTaskIds}
          onSelectAll={() => selectAllTasks(tasksToDisplay.map(t => t.id))}
          onClearSelection={clearSelection}
          onToggleSelectMode={toggleSelectMode}
        />
        <TaskDialog
          taskId={selectedTaskId}
          isOpen={isTaskDialogOpen}
          onClose={onCloseTaskDialog}
        />
        {showCreateTaskWindow && (
          <CreateTaskWindow
            currentCard={currentCard}
            setShowTaskWindow={setShowCreateTaskWindow}
            currentFilter={filterString}
            initialStatus={createTaskStatus}
            initialDate={calendarSelectedDate || undefined}
          />
        )}
      </div>
    );
  }

  // Mobile Filters View
  if (mobileView === 'filters') {
    return (
      <>
        {/* Background content - shows list behind the filter overlay */}
        <div className="flex flex-col flex-1 h-full opacity-50">
          <MobileTopBar title="Tasks" onMenuClick={onMenuClick} />
          <div className="flex-1 overflow-y-auto">
            {renderCurrentView()}
          </div>
        </div>

        {/* Filters Bottom Sheet */}
        <TaskFiltersSheet
          isOpen={true}
          onClose={() => setMobileView('list')}
          dateView={dateView}
          viewMode={viewMode}
          showCompleted={showCompleted}
          sortField={sortField}
          sortDirection={sortDirection}
          selectMode={selectMode}
          onDateViewChange={setDateView}
          onViewModeChange={setViewMode}
          onShowCompletedChange={() => setShowCompleted(!showCompleted)}
          onSortFieldChange={setSortField}
          onSortDirectionToggle={toggleSortDirection}
          onSelectModeToggle={toggleSelectMode}
          onApply={() => setMobileView('list')}
        />
      </>
    );
  }

  return null;
}

export default TaskMobileLayout;
