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
} from "./QuickTagPopover";
import { useSubtaskDisplayMode } from "../../hooks/useSubtaskDisplayMode";
import type {
  TaskData,
  ExternalEventsState,
  ViewSettings,
  ViewSettingsSetters,
  DialogState,
  DialogSetters,
  SelectionState,
  SelectionActions,
  FilterState,
  FilterSetters,
  FilterInputState,
  FilterInputSetters,
  FilterInputHandlers,
  NavigationActions,
  ExternalEventsSetters,
  DesktopHandlers,
  ViewMode,
} from "../../types/taskPage";

interface TaskDesktopLayoutProps {
  // Grouped: Task data
  taskData: TaskData;

  // Grouped: External events
  externalEventsState: ExternalEventsState;

  // Grouped: View settings
  viewSettings: ViewSettings;

  // Grouped: View settings setters
  viewSettingsSetters: ViewSettingsSetters;

  // Grouped: Dialog state
  dialogState: DialogState;

  // Grouped: Dialog setters
  dialogSetters: DialogSetters;

  // Grouped: Selection state
  selectionState: SelectionState;

  // Grouped: Selection actions
  selectionActions: SelectionActions;

  // Grouped: Filter state
  filterState: FilterState;

  // Grouped: Filter setters
  filterSetters: FilterSetters;

  // Grouped: Filter input state
  filterInputState: FilterInputState;

  // Grouped: Filter input setters
  filterInputSetters: FilterInputSetters;

  // Grouped: Filter input handlers
  filterInputHandlers: FilterInputHandlers;

  // Grouped: Navigation actions
  navigationActions: NavigationActions;

  // Grouped: External events setters
  externalEventsSetters: ExternalEventsSetters;

  // Grouped: Desktop handlers
  handlers: DesktopHandlers;
}

/**
 * Desktop two-column enhanced layout for Tasks
 * Layout: Main content (toolbar + view) | Quick lists sidebar (optional)
 */
export function TaskDesktopLayout({
  taskData,
  externalEventsState,
  viewSettings,
  viewSettingsSetters,
  dialogState,
  dialogSetters,
  selectionState,
  selectionActions,
  filterState,
  filterSetters,
  filterInputState,
  filterInputSetters,
  filterInputHandlers,
  navigationActions,
  externalEventsSetters,
  handlers,
}: TaskDesktopLayoutProps) {
  const { subtaskMode, setSubtaskMode } = useSubtaskDisplayMode();

  // Destructure for easier access
  const { tasks, tags, userTimezone, isLoading, tasksToDisplay, paginatedTasks, totalTasksForDateView, totalPages } = taskData;
  const { externalEvents, isLoadingEvents } = externalEventsState;
  const { dateView, viewMode, sortField, sortDirection, calendarViewMode, calendarCurrentDate, currentPage, itemsPerPage } = viewSettings;
  const { showCreateTaskWindow, selectedTaskId, isTaskDialogOpen, createTaskStatus, calendarSelectedDate } = dialogState;
  const { selectMode, selectedTaskIds } = selectionState;
  const { filterString, showFilterHelp, showCompleted } = filterState;
  const { filterInputRef, cursorPosition, filterTrigger, isFilterFocused } = filterInputState;
  const { setFilterTrigger, setIsFilterFocused } = filterInputSetters;
  const { onFilterChange, onSelectQuickTag, onRefreshFilterTriggerFromInput } = filterInputHandlers;
  const { navigateCalendar, toggleSortDirection } = navigationActions;
  const { setRefreshTasks } = externalEventsSetters;

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
                  onChange={onFilterChange}
                  onKeyUp={(e) => onRefreshFilterTriggerFromInput(e.currentTarget)}
                  onClick={(e) => onRefreshFilterTriggerFromInput(e.currentTarget)}
                  onFocus={(e) => {
                    setIsFilterFocused(true);
                    onRefreshFilterTriggerFromInput(e.currentTarget);
                  }}
                  onBlur={() => setIsFilterFocused(false)}
                  placeholder="Filter... try #tag or status:todo"
                  className="h-9 w-full pl-3 pr-8 border border-slate-300 rounded-md text-sm"
                />
                <FilterHelpButton
                  showHelp={showFilterHelp}
                  onToggle={() => filterSetters.setShowFilterHelp(!showFilterHelp)}
                />
                <FilterHelpPopover
                  visible={showFilterHelp}
                  onClose={() => filterSetters.setShowFilterHelp(false)}
                />
              </div>

              <QuickTagPopover
                open={Boolean(filterTrigger && isFilterFocused)}
                anchorInputRef={filterInputRef}
                titleValue={filterString}
                cursorPosition={cursorPosition}
                trigger={filterTrigger}
                onSelectTag={onSelectQuickTag}
                onRequestClose={() => setFilterTrigger(null)}
              />
              <div className="h-9 flex items-center">
                <SearchTagDropdown
                  tags={tags}
                  handleTagClick={handlers.onTagClick}
                />
              </div>
            </div>
            {/* Center section: View mode toggle */}
            <div className="flex items-center gap-2">
              <ViewModeToggle
                value={viewMode}
                onChange={(mode) => viewSettingsSetters.setViewMode(mode)}
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
                  onClick={() => viewSettingsSetters.setShowDisplayMenu(!viewSettings.showDisplayMenu)}
                >
                  Display ▾
                </Button>
                {viewSettings.showDisplayMenu && (
                  <div className="absolute right-0 mt-1 w-64 bg-white border border-slate-300 rounded shadow-lg p-3 z-20">
                    <div className="mb-2">
                      <label className="block text-xs font-semibold mb-1">Date Range</label>
                      <select
                        className="w-full p-1 border border-slate-300 rounded-md text-sm"
                        value={dateView}
                        onChange={handlers.onDateChange}
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
                        onChange={(e) => viewSettingsSetters.setViewMode(e.target.value as ViewMode)}
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
                          onChange={handlers.onShowCompletedChange}
                          className="rounded"
                        />
                        Show Completed Tasks
                      </label>
                    </div>
                    <div className="mb-2">
                      <label className="block text-xs font-semibold mb-1">Subtask Display</label>
                      <div className="space-y-1">
                        <label className="flex items-center gap-2 text-xs cursor-pointer">
                          <input
                            type="radio"
                            name="subtaskMode"
                            checked={subtaskMode === 'nested'}
                            onChange={() => setSubtaskMode('nested')}
                            className="rounded"
                          />
                          <span>Nested</span>
                        </label>
                        <label className="flex items-center gap-2 text-xs cursor-pointer">
                          <input
                            type="radio"
                            name="subtaskMode"
                            checked={subtaskMode === 'flat'}
                            onChange={() => setSubtaskMode('flat')}
                            className="rounded"
                          />
                          <span>Flat</span>
                        </label>
                        <label className="flex items-center gap-2 text-xs cursor-pointer">
                          <input
                            type="radio"
                            name="subtaskMode"
                            checked={subtaskMode === 'hidden'}
                            onChange={() => setSubtaskMode('hidden')}
                            className="rounded"
                          />
                          <span>Hidden</span>
                        </label>
                      </div>
                    </div>
                    <div className="mb-2">
                      <label className="flex items-center gap-2 text-xs font-semibold">
                        <input
                          type="checkbox"
                          checked={selectMode}
                          onChange={selectionActions.toggleSelectMode}
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
                          onChange={handlers.onSortFieldChange}
                        >
                          <option value="updated_at">Updated</option>
                          <option value="title">Name</option>
                          <option value="priority">Priority</option>
                          <option value="status">Status</option>
                          <option value="scheduled_date">Scheduled Date</option>
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
                  dialogSetters.setCreateTaskStatus(undefined);
                  dialogSetters.setCalendarSelectedDate(null);
                  dialogSetters.setShowCreateTaskWindow(!showCreateTaskWindow);
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
                      onClick={() => viewSettingsSetters.setViewMode("list")}
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
                    onViewModeChange={viewSettingsSetters.setCalendarViewMode}
                    onTaskClick={(taskId) => {
                      dialogSetters.setSelectedTaskId(taskId);
                      dialogSetters.setIsTaskDialogOpen(true);
                    }}
                    onCreateTask={(date) => {
                      dialogSetters.setCalendarSelectedDate(date);
                      dialogSetters.setCreateTaskStatus(undefined);
                      dialogSetters.setShowCreateTaskWindow(true);
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
                    onTagClick={handlers.onTagClick}
                    tasks={paginatedTasks}
                    selectMode={selectMode}
                    selectedTaskIds={selectedTaskIds}
                    onTaskSelect={selectionActions.toggleTaskSelection}
                    subtaskMode={subtaskMode}
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
                    dialogSetters.setCreateTaskStatus(undefined);
                    dialogSetters.setCalendarSelectedDate(null);
                    dialogSetters.setShowCreateTaskWindow(true);
                  }}
                  onClearFilters={() => filterSetters.setFilterString("")}
                  onShowCompleted={() => filterSetters.setShowCompleted(true)}
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
                      onClick={() => viewSettingsSetters.setCurrentPage(1)}
                      disabled={currentPage === 1}
                      className="px-3 py-1 text-sm border border-slate-300 rounded disabled:opacity-50 disabled:cursor-not-allowed"
                    >
                      First
                    </Button>
                    <Button
                      onClick={() => viewSettingsSetters.setCurrentPage(Math.max(1, currentPage - 1))}
                      disabled={currentPage === 1}
                      className="px-3 py-1 text-sm border border-slate-300 rounded disabled:opacity-50 disabled:cursor-not-allowed"
                    >
                      Previous
                    </Button>
                    <span className="text-sm text-slate-600">
                      Page {currentPage} of {totalPages}
                    </span>
                    <Button
                      onClick={() => viewSettingsSetters.setCurrentPage(Math.min(totalPages, currentPage + 1))}
                      disabled={currentPage === totalPages}
                      className="px-3 py-1 text-sm border border-slate-300 rounded disabled:opacity-50 disabled:cursor-not-allowed"
                    >
                      Next
                    </Button>
                    <Button
                      onClick={() => viewSettingsSetters.setCurrentPage(totalPages)}
                      disabled={currentPage === totalPages}
                      className="px-3 py-1 text-sm border border-slate-300 rounded disabled:opacity-50 disabled:cursor-not-allowed"
                    >
                      Last
                    </Button>
                    <select
                      value={itemsPerPage}
                      onChange={(e) => viewSettingsSetters.setItemsPerPage(Number(e.target.value))}
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
              onTagClick={handlers.onTagClick}
              tasks={tasksToDisplay}
              onAddTaskWithStatus={handlers.onAddTaskWithStatus}
              selectMode={selectMode}
              selectedTaskIds={selectedTaskIds}
              onTaskSelect={selectionActions.toggleTaskSelection}
              subtaskMode={subtaskMode}
            />
          ) : (
            <EisenhowerMatrix
              onTagClick={handlers.onTagClick}
              tasks={tasksToDisplay}
              onAddTaskWithTags={(tags: string[]) => {
                filterSetters.setFilterString(tags.join(" "));
                dialogSetters.setShowCreateTaskWindow(true);
              }}
              selectMode={selectMode}
              selectedTaskIds={selectedTaskIds}
              onTaskSelect={selectionActions.toggleTaskSelection}
              subtaskMode={subtaskMode}
            />
          )}
        </div>
      </div>

      {/* Dialogs and Overlays */}
      {showCreateTaskWindow && (
        <CreateTaskWindow
          currentCard={null}
          setShowTaskWindow={dialogSetters.setShowCreateTaskWindow}
          currentFilter={filterString}
          initialStatus={createTaskStatus}
          initialDate={calendarSelectedDate || undefined}
        />
      )}

      <TaskDialog
        taskId={selectedTaskId}
        isOpen={isTaskDialogOpen}
        onClose={handlers.onCloseTaskDialog}
      />

      <TaskSelectionOverlay
        tasks={tasksToDisplay}
        selectMode={selectMode}
        selectedTaskIds={selectedTaskIds}
        onSelectAll={() => selectionActions.selectAllTasks(paginatedTasks.map(t => t.id))}
        onClearSelection={selectionActions.clearSelection}
        onToggleSelectMode={selectionActions.toggleSelectMode}
      />
    </div>
  );
}
