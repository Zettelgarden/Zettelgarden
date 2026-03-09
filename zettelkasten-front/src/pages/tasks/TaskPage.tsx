import React, { useEffect, ChangeEvent, useState, useRef } from "react";
import { TaskList } from "../../components/tasks/TaskList";
import { TaskSelectionOverlay } from "../../components/tasks/TaskSelectionOverlay";
import { SearchTagDropdown } from "../../components/tags/SearchTagDropdown";
import { CreateTaskWindow } from "../../components/tasks/CreateTaskWindow";
import { TaskDialog } from "../../components/tasks/TaskDialog";
import { useTaskContext } from "../../contexts/TaskContext";
import { useTagContext } from "../../contexts/TagContext";
import { useAuth } from "../../contexts/AuthContext";
import { setDocumentTitle } from "../../utils/title";
import { Button } from "../../components/Button";
import { useDialogState } from "../../contexts/DialogStateContext";
import { EisenhowerMatrix } from "../../components/tasks/EisenhowerMatrix";
import { KanbanBoard } from "../../components/tasks/KanbanBoard";
import { useTaskPageSettings } from "../../hooks/useTaskPageSettings";
import { useTaskFiltering } from "../../hooks/useTaskFiltering";
import { CalendarViewWrapper } from "../../components/calendar/CalendarView";
import { useNavigate } from "react-router-dom";
import { getExternalEvents } from "../../api/externalEvents";
import { ExternalEvent } from "../../models/ExternalEvent";
import { ErrorBoundary } from "../../components/ErrorBoundary";
import { TaskDesktopLayout } from "../../components/tasks/TaskDesktopLayout";
import { TaskMobileLayout } from "../../components/tasks/TaskMobileLayout";
import { KeyboardShortcutsHelp, useKeyboardShortcuts } from "../../components/tasks/KeyboardShortcutsHelp";
import { useResponsiveLayout } from "../../hooks/useResponsiveLayout";
import { useUIState } from "../../contexts/UIStateContext";

type TaskMobileView = 'list' | 'filters';
import {
  getStartOfMonthInTimezone,
  getEndOfMonthInTimezone,
  getStartOfWeekInTimezone,
  getEndOfWeekInTimezone,
} from "../../utils/dates";
import { parseTaskQuery, updateQueryDateView, updateQueryShowCompleted } from "../../utils/tasks";
import {
  QuickTagPopover,
  type QuickTagTrigger,
  getQuickTagTrigger,
  applyQuickTagSelection,
} from "../../components/tasks/QuickTagPopover";

interface TaskListProps { }

export function TaskPage({ }: TaskListProps) {
  const { tasks, isLoading, showCompleted, setShowCompleted, setRefreshTasks } = useTaskContext();
  const { tags } = useTagContext();
  const { user } = useAuth();
  const { showCreateTaskWindow, setShowCreateTaskWindow } = useDialogState();
  const navigate = useNavigate();
  const { toggleMobileSidebar } = useUIState();
  const userTimezone = user?.timezone || "UTC";

  // Responsive layout state
  const isMobile = typeof window !== 'undefined' && window.innerWidth < 768;
  const [mobileView, setMobileView] = useState<TaskMobileView>('list');
  const [showQuickListsPanel, setShowQuickListsPanel] = useState(false);
  const [isFilterFocused, setIsFilterFocused] = useState(false);

  // State for task dialog
  const [selectedTaskId, setSelectedTaskId] = useState<number | null>(null);
  const [isTaskDialogOpen, setIsTaskDialogOpen] = useState(false);
  const [createTaskStatus, setCreateTaskStatus] = useState<string | undefined>(undefined);
  const [calendarSelectedDate, setCalendarSelectedDate] = useState<Date | null>(null);

  // External calendar events state
  const [externalEvents, setExternalEvents] = useState<ExternalEvent[]>([]);
  const [isLoadingEvents, setIsLoadingEvents] = useState(false);

  // Ref to prevent infinite loops when syncing query and UI
  const isInternalUpdate = useRef(false);

  // Ref for abort controller to cancel pending external events requests
  const abortControllerRef = useRef<AbortController | null>(null);

  // Filter input quick tag autocomplete state
  const filterInputRef = useRef<HTMLInputElement>(null);
  const [cursorPosition, setCursorPosition] = useState(0);
  const [filterTrigger, setFilterTrigger] = useState<QuickTagTrigger | null>(null);

  // Use custom hooks for settings and filtering
  const settings = useTaskPageSettings();
  const {
    tasksToDisplay,
    paginatedTasks,
    totalPages,
    totalTasksForDateView
  } = useTaskFiltering({
    tasks,
    dateView: settings.dateView,
    showCompleted,
    filterString: settings.filterString,
    sortField: settings.sortField,
    sortDirection: settings.sortDirection,
    viewMode: settings.viewMode,
    currentPage: settings.currentPage,
    itemsPerPage: settings.itemsPerPage,
    timezone: userTimezone,
  });

  // Sync UI controls with query keywords in filter string
  useEffect(() => {
    if (isInternalUpdate.current) {
      isInternalUpdate.current = false;
      return;
    }

    const parsed = parseTaskQuery(settings.filterString);

    // Update dateView if it differs from parsed value
    if (parsed.dateView !== settings.dateView) {
      isInternalUpdate.current = true;
      settings.setDateView(parsed.dateView);
    }

    // Update showCompleted if it differs from parsed value
    if (parsed.showCompleted !== showCompleted) {
      isInternalUpdate.current = true;
      setShowCompleted(parsed.showCompleted);
    }
  }, [settings.filterString]);

  // Load external calendar events when in calendar view
  useEffect(() => {
    async function loadExternalEvents() {
      if (settings.viewMode !== "calendar") {
        setExternalEvents([]);
        setIsLoadingEvents(false);
        return;
      }

      // Cancel any pending request
      if (abortControllerRef.current) {
        abortControllerRef.current.abort();
      }

      // Create new abort controller for this request
      const abortController = new AbortController();
      abortControllerRef.current = abortController;

      setIsLoadingEvents(true);

      try {
        // Calculate date range based on view mode and current date
        const currentDate = settings.calendarCurrentDate;
        let start: Date;
        let end: Date;

        if (settings.calendarViewMode === "month") {
          start = getStartOfMonthInTimezone(currentDate, userTimezone);
          end = getEndOfMonthInTimezone(currentDate, userTimezone);
        } else {
          // Week view
          start = getStartOfWeekInTimezone(currentDate, userTimezone, 0);
          end = getEndOfWeekInTimezone(currentDate, userTimezone, 0);
        }

        const events = await getExternalEvents(start, end, abortController.signal);
        // Only update state if this request wasn't aborted
        if (!abortController.signal.aborted) {
          setExternalEvents(events);
        }
      } catch (err) {
        // Only handle error if not due to abort
        if (!abortController.signal.aborted) {
          console.error("Failed to load external events:", err);
          setExternalEvents([]);
        }
      } finally {
        // Only clear loading state if this request wasn't aborted
        if (!abortController.signal.aborted) {
          setIsLoadingEvents(false);
        }
      }
    }

    loadExternalEvents();

    // Cleanup function to abort pending requests on unmount
    return () => {
      if (abortControllerRef.current) {
        abortControllerRef.current.abort();
      }
    };
  }, [settings.viewMode, settings.calendarViewMode, settings.calendarCurrentDate, userTimezone]);

  function handleFilterChange(
    e: ChangeEvent<HTMLInputElement | HTMLTextAreaElement>,
  ) {
    const nextValue = e.target.value;
    const nextCursor = (e.target as HTMLInputElement).selectionStart ?? nextValue.length;

    setCursorPosition(nextCursor);
    setFilterTrigger(getQuickTagTrigger(nextValue, nextCursor));

    settings.setFilterString(nextValue);
  }

  function refreshFilterTriggerFromInput(input: HTMLInputElement) {
    const cursor = input.selectionStart ?? 0;
    setCursorPosition(cursor);
    setFilterTrigger(getQuickTagTrigger(input.value, cursor));
  }

  function handleSelectQuickTag(selectedTagName: string) {
    if (!filterTrigger) return;

    const res = applyQuickTagSelection({
      title: settings.filterString,
      trigger: filterTrigger,
      selectedTagName,
    });

    setCursorPosition(res.nextCursor);

    if (!res.didInsert) {
      setFilterTrigger(null);
      return;
    }

    settings.setFilterString(res.nextTitle);
    setFilterTrigger(null);

    // Restore focus + cursor after React updates the controlled input.
    requestAnimationFrame(() => {
      const input = filterInputRef.current;
      if (!input) return;
      input.focus();
      input.setSelectionRange(res.nextCursor, res.nextCursor);
    });
  }

  function handleDateChange(e: ChangeEvent<HTMLSelectElement>) {
    isInternalUpdate.current = true;
    const newDateView = e.target.value;
    settings.setDateView(newDateView);
    // Update filter string to include the date view keyword
    settings.setFilterString(updateQueryDateView(settings.filterString, newDateView));
  }

  function handleSortFieldChange(e: ChangeEvent<HTMLSelectElement>) {
    settings.setSortField(e.target.value as "updated_at" | "title" | "priority" | "status" | "id" | "scheduled_date" | "due_date");
  }

  function handleShowCompletedChange() {
    isInternalUpdate.current = true;
    const newShowCompleted = !showCompleted;
    setShowCompleted(newShowCompleted);
    // Update filter string to include/remove the completed keyword
    settings.setFilterString(updateQueryShowCompleted(settings.filterString, newShowCompleted));
  }

  function toggleShowTaskWindow() {
    setCreateTaskStatus(undefined);
    setCalendarSelectedDate(null); // Reset calendar date when opening from toolbar
    setShowCreateTaskWindow(!showCreateTaskWindow);
  }

  function handleAddTaskWithStatus(status: string) {
    setCreateTaskStatus(status);
    setShowCreateTaskWindow(true);
  }

  function handleTagClick(tag: string) {
    settings.setFilterString("#" + tag);
  }

  function handleCloseTaskDialog() {
    setIsTaskDialogOpen(false);
    setSelectedTaskId(null);
    // Remove taskId parameter from URL
    const params = new URLSearchParams(location.search);
    params.delete("taskId");
    const newSearch = params.toString();
    navigate(`/app/tasks${newSearch ? `?${newSearch}` : ""}`, { replace: true });
  }

  function handleTaskClick(taskId: number) {
    setSelectedTaskId(taskId);
    setIsTaskDialogOpen(true);
  }

  // Keyboard shortcuts
  const { showHelp, setShowHelp } = useKeyboardShortcuts({
    onNewTask: () => {
      setCreateTaskStatus(undefined);
      setCalendarSelectedDate(null);
      setShowCreateTaskWindow(true);
    },
    onSwitchView: (view) => {
      settings.setViewMode(view);
    },
    onEscape: () => {
      setShowCreateTaskWindow(false);
    },
  });

  useEffect(() => {
    setDocumentTitle("Tasks");

    const params = new URLSearchParams(location.search);
    const term = params.get("term");
    if (term) {
      settings.setFilterString(term);
    }

    // Check for taskId parameter to open specific task dialog
    const taskIdParam = params.get("taskId");
    if (taskIdParam) {
      const taskId = parseInt(taskIdParam, 10);
      if (!isNaN(taskId)) {
        setSelectedTaskId(taskId);
        setIsTaskDialogOpen(true);
      }
    }
  }, [settings.setFilterString]);

  return (
    <div className="h-screen flex flex-col md:overflow-hidden">
      <KeyboardShortcutsHelp
        visible={showHelp}
        onClose={() => setShowHelp(false)}
      />
      {isMobile ? (
        <TaskMobileLayout
          mobileView={mobileView}
          setMobileView={setMobileView}
          // Task data
          tasks={tasks}
          tasksToDisplay={tasksToDisplay}
          paginatedTasks={paginatedTasks}
          totalTasksForDateView={totalTasksForDateView}
          totalPages={totalPages}
          tags={tags}
          userTimezone={userTimezone}
          isLoading={isLoading}
          // Settings
          dateView={settings.dateView}
          filterString={settings.filterString}
          sortField={settings.sortField}
          sortDirection={settings.sortDirection}
          viewMode={settings.viewMode}
          currentPage={settings.currentPage}
          itemsPerPage={settings.itemsPerPage}
          calendarViewMode={settings.calendarViewMode}
          calendarCurrentDate={settings.calendarCurrentDate}
          showCompleted={showCompleted}
          // UI state
          showFilterHelp={settings.showFilterHelp}
          showDisplayMenu={settings.showDisplayMenu}
          selectMode={settings.selectMode}
          selectedTaskIds={settings.selectedTaskIds}
          // External events
          externalEvents={externalEvents}
          isLoadingEvents={isLoadingEvents}
          // Dialog states
          showCreateTaskWindow={showCreateTaskWindow}
          selectedTaskId={selectedTaskId}
          isTaskDialogOpen={isTaskDialogOpen}
          createTaskStatus={createTaskStatus}
          calendarSelectedDate={calendarSelectedDate}
          // Setters
          setRefreshTasks={setRefreshTasks}
          setShowCreateTaskWindow={setShowCreateTaskWindow}
          setSelectedTaskId={setSelectedTaskId}
          setIsTaskDialogOpen={setIsTaskDialogOpen}
          setCreateTaskStatus={setCreateTaskStatus}
          setCalendarSelectedDate={setCalendarSelectedDate}
          setExternalEvents={setExternalEvents}
          setIsLoadingEvents={setIsLoadingEvents}
          // Settings setters
          setDateView={settings.setDateView}
          setFilterString={settings.setFilterString}
          setSortField={settings.setSortField}
          setSortDirection={settings.setSortDirection}
          setViewMode={settings.setViewMode}
          setCurrentPage={settings.setCurrentPage}
          setItemsPerPage={settings.setItemsPerPage}
          setCalendarViewMode={settings.setCalendarViewMode}
          setCalendarCurrentDate={settings.setCalendarCurrentDate}
          setShowFilterHelp={settings.setShowFilterHelp}
          setShowDisplayMenu={settings.setShowDisplayMenu}
          setSelectMode={settings.setSelectMode}
          setSelectedTaskIds={settings.setSelectedTaskIds}
          // Actions
          toggleSelectMode={settings.toggleSelectMode}
          toggleTaskSelection={settings.toggleTaskSelection}
          selectAllTasks={settings.selectAllTasks}
          clearSelection={settings.clearSelection}
          toggleSortDirection={settings.toggleSortDirection}
          navigateCalendar={settings.navigateCalendar}
          setShowCompleted={setShowCompleted}
          // Handlers
          onMenuClick={toggleMobileSidebar}
          onTagClick={handleTagClick}
          onAddTaskWithStatus={handleAddTaskWithStatus}
          onCloseTaskDialog={handleCloseTaskDialog}
          onTaskClick={handleTaskClick}
          // Filter handlers
          onFilterChange={handleFilterChange}
          onRefreshFilterTriggerFromInput={refreshFilterTriggerFromInput}
          onSelectQuickTag={handleSelectQuickTag}
          filterInputRef={filterInputRef}
          cursorPosition={cursorPosition}
          filterTrigger={filterTrigger}
          setFilterTrigger={setFilterTrigger}
          isFilterFocused={isFilterFocused}
          setIsFilterFocused={setIsFilterFocused}
        />
      ) : (
        <TaskDesktopLayout
          showQuickListsPanel={showQuickListsPanel}
          // Task data
          tasks={tasks}
          tags={tags}
          userTimezone={userTimezone}
          isLoading={isLoading}
          // Filtered tasks
          tasksToDisplay={tasksToDisplay}
          paginatedTasks={paginatedTasks}
          totalTasksForDateView={totalTasksForDateView}
          totalPages={totalPages}
          // Settings
          dateView={settings.dateView}
          filterString={settings.filterString}
          sortField={settings.sortField}
          sortDirection={settings.sortDirection}
          viewMode={settings.viewMode}
          currentPage={settings.currentPage}
          itemsPerPage={settings.itemsPerPage}
          calendarViewMode={settings.calendarViewMode}
          calendarCurrentDate={settings.calendarCurrentDate}
          showCompleted={showCompleted}
          // UI state
          showFilterHelp={settings.showFilterHelp}
          showDisplayMenu={settings.showDisplayMenu}
          selectMode={settings.selectMode}
          selectedTaskIds={settings.selectedTaskIds}
          // External events
          externalEvents={externalEvents}
          isLoadingEvents={isLoadingEvents}
          // Dialog states
          showCreateTaskWindow={showCreateTaskWindow}
          selectedTaskId={selectedTaskId}
          isTaskDialogOpen={isTaskDialogOpen}
          createTaskStatus={createTaskStatus}
          calendarSelectedDate={calendarSelectedDate}
          // Setters
          setRefreshTasks={setRefreshTasks}
          setShowCreateTaskWindow={setShowCreateTaskWindow}
          setSelectedTaskId={setSelectedTaskId}
          setIsTaskDialogOpen={setIsTaskDialogOpen}
          setCreateTaskStatus={setCreateTaskStatus}
          setCalendarSelectedDate={setCalendarSelectedDate}
          setExternalEvents={setExternalEvents}
          setIsLoadingEvents={setIsLoadingEvents}
          // Settings setters
          setDateView={settings.setDateView}
          setFilterString={settings.setFilterString}
          setSortField={settings.setSortField}
          setSortDirection={settings.setSortDirection}
          setViewMode={settings.setViewMode}
          setCurrentPage={settings.setCurrentPage}
          setItemsPerPage={settings.setItemsPerPage}
          setCalendarViewMode={settings.setCalendarViewMode}
          setCalendarCurrentDate={settings.setCalendarCurrentDate}
          setShowFilterHelp={settings.setShowFilterHelp}
          setShowDisplayMenu={settings.setShowDisplayMenu}
          setSelectMode={settings.setSelectMode}
          setSelectedTaskIds={settings.setSelectedTaskIds}
          toggleSelectMode={settings.toggleSelectMode}
          toggleTaskSelection={settings.toggleTaskSelection}
          selectAllTasks={settings.selectAllTasks}
          clearSelection={settings.clearSelection}
          toggleSortDirection={settings.toggleSortDirection}
          setShowCompleted={setShowCompleted}
          navigateCalendar={settings.navigateCalendar}
          // Handlers
          onTagClick={handleTagClick}
          onAddTaskWithStatus={handleAddTaskWithStatus}
          onCloseTaskDialog={handleCloseTaskDialog}
          onDateChange={handleDateChange}
          onSortFieldChange={handleSortFieldChange}
          onShowCompletedChange={handleShowCompletedChange}
          onFilterChange={handleFilterChange}
          onSelectQuickTag={handleSelectQuickTag}
          onRefreshFilterTriggerFromInput={refreshFilterTriggerFromInput}
        />
      )}
    </div>
  );
}
