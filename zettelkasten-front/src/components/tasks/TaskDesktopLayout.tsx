import React, { ChangeEvent } from "react";
import { Task } from "../../models/Task";
import { Tag } from "../../models/Tags";
import { ExternalEvent } from "../../models/ExternalEvent";
import { EisenhowerMatrix } from "./EisenhowerMatrix";
import { KanbanBoard } from "./KanbanBoard";
import { TaskList } from "./TaskList";
import { TaskListSkeleton } from "./TaskListSkeleton";
import { TaskEmptyState, getEmptyStateType } from "./TaskEmptyState";
import { FilterHelpButton, FilterHelpPopover } from "./FilterHelpButton";
import { ViewModeToggle } from "./ViewModeToggle";
import { TaskSelectionOverlay } from "./TaskSelectionOverlay";
import { CreateTaskWindow } from "./CreateTaskWindow";
import { TaskDialog } from "./TaskDialog";
import { CalendarViewWrapper } from "../../components/calendar/CalendarView";
import { ErrorBoundary } from "../../components/ErrorBoundary";
import { Button } from "../../components/Button";
import { SearchTagDropdown } from "../../components/tags/SearchTagDropdown";
import {
  QuickTagPopover,
  type QuickTagTrigger,
  getQuickTagTrigger,
  applyQuickTagSelection,
} from "./QuickTagPopover";
import {
  getStartOfMonthInTimezone,
  getEndOfMonthInTimezone,
  getStartOfWeekInTimezone,
  getEndOfWeekInTimezone,
} from "../../utils/dates";

type SortField = "updated_at" | "title" | "priority" | "status" | "id" | "scheduled_date" | "due_date";
type SortDirection = "asc" | "desc";
type ViewMode = "list" | "matrix" | "kanban" | "calendar";

interface TaskDesktopLayoutProps {
  // Task data
  tasks: Task[];
  tags: Tag[];
  showCompleted: boolean;
  userTimezone: string;
  isLoading?: boolean;
  filterInputRef?: React.RefObject<HTMLInputElement>;

  // Filtered and paginated tasks
  tasksToDisplay: Task[];
  paginatedTasks: Task[];
  totalTasksForDateView: number;
  totalPages: number;

  // View settings
  dateView: string;
  filterString: string;
  sortField: SortField;
  sortDirection: SortDirection;
  viewMode: ViewMode;
  calendarViewMode: "month" | "week";
  calendarCurrentDate: Date;
  currentPage: number;
  itemsPerPage: number;
  showFilterHelp: boolean;
  showDisplayMenu: boolean;
  selectMode: boolean;
  selectedTaskIds: Set<number>;

  // External calendar events
  externalEvents: ExternalEvent[];
  isLoadingEvents: boolean;

  // Dialog states
  showCreateTaskWindow: boolean;
  selectedTaskId: number | null;
  isTaskDialogOpen: boolean;
  createTaskStatus: string | undefined;
  calendarSelectedDate: Date | null;

  // Setters
  setShowCompleted: (show: boolean) => void;
  setRefreshTasks: (refresh: boolean) => void;

  // Settings methods (from useTaskPageSettings hook)
  setDateView: (dateView: string | ((prev: string) => string)) => void;
  setFilterString: (filterString: string | ((prev: string) => string)) => void;
  setSortField: (sortField: SortField | ((prev: SortField) => SortField)) => void;
  setSortDirection: (sortDirection: SortDirection | ((prev: SortDirection) => SortDirection)) => void;
  setViewMode: (viewMode: ViewMode | ((prev: ViewMode) => ViewMode)) => void;
  setCalendarViewMode: (viewMode: "month" | "week" | ((prev: "month" | "week") => "month" | "week")) => void;
  setCalendarCurrentDate: (date: Date | ((prev: Date) => Date)) => void;
  setCurrentPage: (page: number | ((prev: number) => number)) => void;
  setItemsPerPage: (itemsPerPage: number | ((prev: number) => number)) => void;
  setShowFilterHelp: (show: boolean | ((prev: boolean) => boolean)) => void;
  setShowDisplayMenu: (show: boolean | ((prev: boolean) => boolean)) => void;
  setSelectMode: (selectMode: boolean | ((prev: boolean) => boolean)) => void;
  setSelectedTaskIds: (taskIds: Set<number> | ((prev: Set<number>) => Set<number>)) => void;
  setShowCreateTaskWindow: (show: boolean | ((prev: boolean) => boolean)) => void;
  setSelectedTaskId: (taskId: number | null) => void;
  setIsTaskDialogOpen: (open: boolean | ((prev: boolean) => boolean)) => void;
  setCreateTaskStatus: (status: string | undefined) => void;
  setCalendarSelectedDate: (date: Date | null) => void;
  setExternalEvents: (events: ExternalEvent[] | ((prev: ExternalEvent[]) => ExternalEvent[])) => void;
  setIsLoadingEvents: (loading: boolean | ((prev: boolean) => boolean)) => void;

  // Actions
  toggleSortDirection: () => void;
  toggleSelectMode: () => void;
  toggleTaskSelection: (taskId: number) => void;
  selectAllTasks: (taskIds: number[]) => void;
  clearSelection: () => void;
  navigateCalendar: (direction: "prev" | "next" | "today") => void;

  // Event handlers
  onTagClick: (tag: string) => void;
  onAddTaskWithStatus: (status: string) => void;
  onCloseTaskDialog: () => void;
  onDateChange: (e: ChangeEvent<HTMLSelectElement>) => void;
  onSortFieldChange: (e: ChangeEvent<HTMLSelectElement>) => void;
  onShowCompletedChange: () => void;
  onFilterChange: (e: ChangeEvent<HTMLInputElement | HTMLTextAreaElement>) => void;
  onSelectQuickTag: (tagName: string) => void;
  onRefreshFilterTriggerFromInput: (input: HTMLInputElement) => void;
}

/**
 * Desktop two-column enhanced layout for Tasks
 * Layout: Main content (toolbar + view) | Quick lists sidebar (optional)
 */
export function TaskDesktopLayout({
  tasks,
  tags,
  showCompleted,
  userTimezone,
  isLoading = false,
  filterInputRef: externalFilterInputRef,
  tasksToDisplay,
  paginatedTasks,
  totalTasksForDateView,
  totalPages,
  dateView,
  filterString,
  sortField,
  sortDirection,
  viewMode,
  calendarViewMode,
  calendarCurrentDate,
  currentPage,
  itemsPerPage,
  showFilterHelp,
  showDisplayMenu,
  selectMode,
  selectedTaskIds,
  externalEvents,
  isLoadingEvents,
  showCreateTaskWindow,
  selectedTaskId,
  isTaskDialogOpen,
  createTaskStatus,
  calendarSelectedDate,
  setShowCompleted,
  setRefreshTasks,
  setDateView,
  setFilterString,
  setSortField,
  setSortDirection,
  setViewMode,
  setCalendarViewMode,
  setCalendarCurrentDate,
  setCurrentPage,
  setItemsPerPage,
  setShowFilterHelp,
  setShowDisplayMenu,
  setSelectMode,
  setSelectedTaskIds,
  setShowCreateTaskWindow,
  setSelectedTaskId,
  setIsTaskDialogOpen,
  setCreateTaskStatus,
  setCalendarSelectedDate,
  setExternalEvents,
  setIsLoadingEvents,
  toggleSortDirection,
  toggleSelectMode,
  toggleTaskSelection,
  selectAllTasks,
  clearSelection,
  navigateCalendar,
  onTagClick,
  onAddTaskWithStatus,
  onCloseTaskDialog,
  onDateChange,
  onSortFieldChange,
  onShowCompletedChange,
  onFilterChange,
  onSelectQuickTag,
  onRefreshFilterTriggerFromInput,
}: TaskDesktopLayoutProps) {
  // Refs for quick tag autocomplete - use external ref if provided, otherwise internal
  const internalFilterInputRef = React.useRef<HTMLInputElement>(null);
  const filterInputRef = externalFilterInputRef || internalFilterInputRef;
  const [cursorPosition, setCursorPosition] = React.useState(0);
  const [filterTrigger, setFilterTrigger] = React.useState<QuickTagTrigger | null>(null);
  const [isFilterFocused, setIsFilterFocused] = React.useState(false);

  const handleSelectQuickTag = (selectedTagName: string) => {
    if (!filterTrigger) return;

    const res = applyQuickTagSelection({
      title: filterString,
      trigger: filterTrigger,
      selectedTagName,
    });

    setCursorPosition(res.nextCursor);

    if (!res.didInsert) {
      setFilterTrigger(null);
      return;
    }

    setFilterString(res.nextTitle);
    setFilterTrigger(null);

    // Restore focus + cursor after React updates the controlled input.
    requestAnimationFrame(() => {
      const input = filterInputRef.current;
      if (!input) return;
      input.focus();
      input.setSelectionRange(res.nextCursor, res.nextCursor);
    });
  };

  const handleFilterChange = (e: ChangeEvent<HTMLInputElement | HTMLTextAreaElement>) => {
    const nextValue = e.target.value;
    const nextCursor = (e.target as HTMLInputElement).selectionStart ?? nextValue.length;

    setCursorPosition(nextCursor);
    setFilterTrigger(getQuickTagTrigger(nextValue, nextCursor));

    setFilterString(nextValue);
  };

  const refreshFilterTriggerFromInput = (input: HTMLInputElement) => {
    const cursor = input.selectionStart ?? 0;
    setCursorPosition(cursor);
    setFilterTrigger(getQuickTagTrigger(input.value, cursor));
  };

  return (
    <div className="hidden md:flex flex-row h-screen overflow-hidden">
      {/* Main Content Area */}
      <div className="flex-1 flex flex-col overflow-hidden">
        {/* Toolbar */}
        <div className="bg-slate-100 p-3 border-b border-slate-300 flex-shrink-0">
          <div className="flex flex-col sm:flex-row flex-wrap items-stretch sm:items-center justify-between gap-2 sm:gap-3">
            {/* Left section: Filter */}
            <div className="flex flex-wrap items-center gap-2 flex-grow min-w-0 w-full sm:w-auto">
              <div className="relative flex-grow w-full sm:max-w-md">
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
            {/* Center section: View mode toggle */}
            <div className="flex items-center gap-2">
              <ViewModeToggle
                value={viewMode}
                onChange={(mode) => setViewMode(mode)}
              />
            </div>
            {/* Right section: Count, Display menu, Actions */}
            <div className="flex items-center gap-3 flex-shrink-0">
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
              {/* Display dropdown */}
              <div className="relative">
                <Button
                  className="h-9 px-3 text-sm bg-slate-300 rounded-md"
                  onClick={() => setShowDisplayMenu(!showDisplayMenu)}
                >
                  Display ▾
                </Button>
                {showDisplayMenu && (
                  <div className="absolute right-0 mt-1 w-64 bg-white border border-slate-300 rounded shadow-lg p-3 z-20">
                    <div className="mb-2">
                      <label className="block text-xs font-semibold mb-1">Date Range</label>
                      <select
                        className="w-full p-1 border border-slate-300 rounded-md text-sm"
                        value={dateView}
                        onChange={onDateChange}
                      >
                        <option value="today">Today</option>
                        <option value="tomorrow">Tomorrow</option>
                        <option value="this_week">This Week</option>
                        <option value="overdue">Overdue</option>
                        <option value="no_date">No Date</option>
                        <option value="all">All</option>
                      </select>
                    </div>
                    <div className="mb-2">
                      <label className="block text-xs font-semibold mb-1">View Mode</label>
                      <select
                        className="w-full p-1 border border-slate-300 rounded-md text-sm"
                        value={viewMode}
                        onChange={(e) => setViewMode(e.target.value as ViewMode)}
                      >
                        <option value="list">List View</option>
                        <option value="matrix">Eisenhower Matrix</option>
                        <option value="kanban">Kanban Board</option>
                        <option value="calendar">Calendar View</option>
                      </select>
                    </div>
                    <div className="mb-2">
                      <label className="flex items-center gap-2 text-xs font-semibold">
                        <input
                          type="checkbox"
                          checked={showCompleted}
                          onChange={onShowCompletedChange}
                          className="rounded"
                        />
                        Show Completed Tasks
                      </label>
                    </div>
                    <div className="mb-2">
                      <label className="flex items-center gap-2 text-xs font-semibold">
                        <input
                          type="checkbox"
                          checked={selectMode}
                          onChange={toggleSelectMode}
                          className="rounded"
                        />
                        Select Mode
                      </label>
                    </div>
                    <div>
                      <label className="block text-xs font-semibold mb-1">Sort By</label>
                      <div className="flex items-center gap-1">
                        <select
                          id="sort-select"
                          className="flex-grow p-1 border border-slate-300 rounded-md text-sm"
                          value={sortField}
                          onChange={onSortFieldChange}
                        >
                          <option value="updated_at">Updated</option>
                          <option value="title">Name</option>
                          <option value="priority">Priority</option>
                          <option value="status">Status</option>
                          <option value="scheduled_date">Scheduled Date</option>
                          <option value="due_date">Due Date</option>
                          <option value="id">ID</option>
                        </select>
                        <Button onClick={toggleSortDirection} className="p-1 text-xs border border-slate-300 rounded-md">
                          {sortDirection === "asc" ? "↑" : "↓"}
                        </Button>
                      </div>
                    </div>
                  </div>
                )}
              </div>
              <Button
                onClick={() => {
                  setCreateTaskStatus(undefined);
                  setCalendarSelectedDate(null);
                  setShowCreateTaskWindow(!showCreateTaskWindow);
                }}
                className="h-9 bg-blue-600 hover:bg-blue-700 text-white rounded-md px-3 text-lg font-bold flex items-center justify-center pb-2"
                title="Add Task (N)"
              >
                +
              </Button>
              <span
                className="text-slate-400 text-xs cursor-help"
                title="Press ? for keyboard shortcuts"
              >
                ⌨️
              </span>
            </div>
          </div>
        </div>

        {/* Task Content Area */}
        <div className="flex-1 overflow-auto p-4">
          {viewMode === "calendar" ? (
            <ErrorBoundary
              fallback={
                <div className="p-4 m-4 border border-red-300 rounded bg-red-50">
                  <h2 className="text-lg font-semibold text-red-800 mb-2">Calendar Error</h2>
                  <p className="text-red-600 mb-3">
                    We encountered an error while displaying the calendar. Please try refreshing the page or switching to a different view.
                  </p>
                  <div className="flex gap-2">
                    <button
                      className="px-4 py-2 bg-red-600 text-white rounded hover:bg-red-700 transition-colors"
                      onClick={() => window.location.reload()}
                    >
                      Refresh Page
                    </button>
                    <button
                      className="px-4 py-2 border border-slate-300 text-slate-700 rounded hover:bg-slate-50 transition-colors"
                      onClick={() => setViewMode("list")}
                    >
                      Switch to List View
                    </button>
                  </div>
                </div>
              }
            >
              {isLoading ? (
                <div className="flex items-center justify-center h-64">
                  <div className="flex flex-col items-center gap-3">
                    <div className="w-8 h-8 border-3 border-blue-600 border-t-transparent rounded-full animate-spin" />
                    <span className="text-sm text-slate-500">Loading calendar...</span>
                  </div>
                </div>
              ) : (
                <>
                  {isLoadingEvents && (
                    <div className="mb-4 p-3 bg-blue-50 border border-blue-200 rounded-md flex items-center gap-2">
                      <div className="w-4 h-4 border-2 border-blue-600 border-t-transparent rounded-full animate-spin" aria-hidden="true"></div>
                      <span className="text-sm text-blue-700">Loading external events...</span>
                    </div>
                  )}
                  <CalendarViewWrapper
                    tasks={tasksToDisplay}
                    externalEvents={externalEvents}
                    currentDate={calendarCurrentDate}
                    viewMode={calendarViewMode}
                    onNavigate={navigateCalendar}
                    onViewModeChange={setCalendarViewMode}
                    onTaskClick={(taskId) => {
                      setSelectedTaskId(taskId);
                      setIsTaskDialogOpen(true);
                    }}
                    onCreateTask={(date) => {
                      setCalendarSelectedDate(date);
                      setCreateTaskStatus(undefined);
                      setShowCreateTaskWindow(true);
                    }}
                    onTaskMoved={() => {
                      setRefreshTasks(true);
                    }}
                    timezone={userTimezone}
                  />
                </>
              )}
            </ErrorBoundary>
          ) : viewMode === "list" ? (
            <>
              {isLoading ? (
                <TaskListSkeleton count={8} />
              ) : paginatedTasks.length > 0 ? (
                <ul>
                  <TaskList
                    onTagClick={onTagClick}
                    tasks={paginatedTasks}
                    selectMode={selectMode}
                    selectedTaskIds={selectedTaskIds}
                    onTaskSelect={toggleTaskSelection}
                  />
                </ul>
              ) : (
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
              )}
              {/* Pagination controls - only show if there are tasks and multiple pages */}
              {tasksToDisplay.length > 0 && totalPages > 1 && (
                <div className="mt-4 flex flex-col sm:flex-row items-center justify-between gap-3 border-t pt-4">
                  <div className="text-sm text-slate-600">
                    Showing {((currentPage - 1) * itemsPerPage) + 1}-{Math.min(currentPage * itemsPerPage, tasksToDisplay.length)} of {tasksToDisplay.length} {tasksToDisplay.length === 1 ? "task" : "tasks"}
                  </div>
                  <div className="flex items-center gap-2">
                    <Button
                      onClick={() => setCurrentPage(1)}
                      disabled={currentPage === 1}
                      className="px-3 py-1 text-sm border border-slate-300 rounded disabled:opacity-50 disabled:cursor-not-allowed"
                    >
                      First
                    </Button>
                    <Button
                      onClick={() => setCurrentPage(Math.max(1, currentPage - 1))}
                      disabled={currentPage === 1}
                      className="px-3 py-1 text-sm border border-slate-300 rounded disabled:opacity-50 disabled:cursor-not-allowed"
                    >
                      Previous
                    </Button>
                    <span className="text-sm text-slate-600">
                      Page {currentPage} of {totalPages}
                    </span>
                    <Button
                      onClick={() => setCurrentPage(Math.min(totalPages, currentPage + 1))}
                      disabled={currentPage === totalPages}
                      className="px-3 py-1 text-sm border border-slate-300 rounded disabled:opacity-50 disabled:cursor-not-allowed"
                    >
                      Next
                    </Button>
                    <Button
                      onClick={() => setCurrentPage(totalPages)}
                      disabled={currentPage === totalPages}
                      className="px-3 py-1 text-sm border border-slate-300 rounded disabled:opacity-50 disabled:cursor-not-allowed"
                    >
                      Last
                    </Button>
                    <select
                      value={itemsPerPage}
                      onChange={(e) => setItemsPerPage(Number(e.target.value))}
                      className="ml-2 px-2 py-1 text-sm border border-slate-300 rounded"
                    >
                      <option value={25}>25 per page</option>
                      <option value={50}>50 per page</option>
                      <option value={100}>100 per page</option>
                      <option value={200}>200 per page</option>
                    </select>
                  </div>
                </div>
              )}
            </>
          ) : viewMode === "kanban" ? (
            <KanbanBoard
              onTagClick={onTagClick}
              tasks={tasksToDisplay}
              onAddTaskWithStatus={onAddTaskWithStatus}
              selectMode={selectMode}
              selectedTaskIds={selectedTaskIds}
              onTaskSelect={toggleTaskSelection}
            />
          ) : (
            <EisenhowerMatrix
              onTagClick={onTagClick}
              tasks={tasksToDisplay}
              onAddTaskWithTags={(tags: string[]) => {
                setFilterString(tags.join(" "));
                setShowCreateTaskWindow(true);
              }}
              selectMode={selectMode}
              selectedTaskIds={selectedTaskIds}
              onTaskSelect={toggleTaskSelection}
            />
          )}
        </div>
      </div>

      {/* Dialogs and Overlays */}
      {showCreateTaskWindow && (
        <CreateTaskWindow
          currentCard={null}
          setShowTaskWindow={setShowCreateTaskWindow}
          currentFilter={filterString}
          initialStatus={createTaskStatus}
          initialDate={calendarSelectedDate || undefined}
        />
      )}

      <TaskDialog
        taskId={selectedTaskId}
        isOpen={isTaskDialogOpen}
        onClose={onCloseTaskDialog}
      />

      <TaskSelectionOverlay
        tasks={tasksToDisplay}
        selectMode={selectMode}
        selectedTaskIds={selectedTaskIds}
        onSelectAll={() => selectAllTasks(paginatedTasks.map(t => t.id))}
        onClearSelection={clearSelection}
        onToggleSelectMode={toggleSelectMode}
      />
    </div>
  );
}
