import React, { useEffect, ChangeEvent, useState, useRef } from "react";
import { TaskList } from "../../components/tasks/TaskList";
import { TaskPageOptionsMenu } from "../../components/tasks/TaskPageOptionsMenu";
import { CreateTaskWindow } from "../../components/tasks/CreateTaskWindow";
import { TaskDialog } from "../../components/tasks/TaskDialog";
import { useTaskContext } from "../../contexts/TaskContext";
import { useTagContext } from "../../contexts/TagContext";
import { setDocumentTitle } from "../../utils/title";
import { Button } from "../../components/Button";
import { useShortcutContext } from "../../contexts/ShortcutContext";
import { EisenhowerMatrix } from "../../components/tasks/EisenhowerMatrix";
import { KanbanBoard } from "../../components/tasks/KanbanBoard";
import { useTaskPageSettings } from "../../hooks/useTaskPageSettings";
import { useTaskFiltering } from "../../hooks/useTaskFiltering";
import { useNavigate } from "react-router-dom";
import { parseTaskQuery, updateQueryDateView, updateQueryShowCompleted } from "../../utils/tasks";

interface TaskListProps { }

export function TaskPage({ }: TaskListProps) {
  const { tasks, showCompleted, setShowCompleted } = useTaskContext();
  const { tags } = useTagContext();
  const { showCreateTaskWindow, setShowCreateTaskWindow } = useShortcutContext();
  const navigate = useNavigate();

  // State for task dialog
  const [selectedTaskId, setSelectedTaskId] = useState<number | null>(null);
  const [isTaskDialogOpen, setIsTaskDialogOpen] = useState(false);

  // Ref to prevent infinite loops when syncing query and UI
  const isInternalUpdate = useRef(false);

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

  function handleFilterChange(
    e: ChangeEvent<HTMLInputElement | HTMLTextAreaElement>,
  ) {
    settings.setFilterString(e.target.value);
  }

  function handleDateChange(e: ChangeEvent<HTMLSelectElement>) {
    isInternalUpdate.current = true;
    const newDateView = e.target.value;
    settings.setDateView(newDateView);
    // Update filter string to include the date view keyword
    settings.setFilterString(updateQueryDateView(settings.filterString, newDateView));
  }

  function handleSortFieldChange(e: ChangeEvent<HTMLSelectElement>) {
    settings.setSortField(e.target.value as "updated_at" | "title" | "priority" | "status" | "id" | "scheduled_date");
  }

  function handleShowCompletedChange() {
    isInternalUpdate.current = true;
    const newShowCompleted = !showCompleted;
    setShowCompleted(newShowCompleted);
    // Update filter string to include/remove the completed keyword
    settings.setFilterString(updateQueryShowCompleted(settings.filterString, newShowCompleted));
  }

  function toggleShowTaskWindow() {
    setShowCreateTaskWindow(!showCreateTaskWindow);
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

  const handleKeyPress = (event: KeyboardEvent) => {
    // if this is true, the user is using a system shortcut, don't do anything with it
    if (event.metaKey) {
      return;
    }
    if (event.key === "Escape") {
      event.preventDefault();
      setShowCreateTaskWindow(false);
      return;
    }
  };

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

    document.addEventListener("keydown", handleKeyPress);
    return () => {
      document.removeEventListener("keydown", handleKeyPress);
    };
  }, [setShowCreateTaskWindow, settings.setFilterString]); // Added setShowCreateTaskWindow to dependency array

  return (
    <div>
      {/* Redesigned toolbar header */}
      <div className="bg-slate-100 p-3 border-b border-slate-300">
        <div className="flex flex-col sm:flex-row flex-wrap items-stretch sm:items-center justify-between gap-2 sm:gap-3">
          {/* Left section: Filter */}
          <div className="flex flex-wrap items-center gap-2 flex-grow min-w-0 w-full sm:w-auto">
            <div className="relative flex-grow w-full sm:max-w-md">
              <input
                type="text"
                value={settings.filterString}
                onChange={handleFilterChange}
                placeholder="Filter tasks..."
                className="h-9 w-full pl-3 pr-8 border border-slate-300 rounded-md text-sm"
              />
              <span
                className="absolute right-2 top-1/2 -translate-y-1/2 text-slate-500 cursor-help"
                onMouseEnter={() => settings.setShowFilterHelp(true)}
                onMouseLeave={() => settings.setShowFilterHelp(false)}
              >
                ?
              </span>
              {settings.showFilterHelp && (
                <div className="absolute top-full mt-2 left-0 bg-white p-3 border border-slate-300 rounded shadow-lg z-20 w-auto min-w-[280px]">
                  <h4 className="font-semibold mb-2 text-slate-700">Filter Options:</h4>
                  <ul className="list-none space-y-1 text-sm text-slate-600">
                    <li><strong>Text:</strong> e.g., <code>meeting</code></li>
                    <li><strong>Tag:</strong> <code>#tagName</code></li>
                    <li><strong>Priority:</strong> <code>priority:A</code></li>
                    <li><strong>Status:</strong> <code>status:todo</code>, <code>status:in_progress</code>, <code>status:blocked</code>, <code>status:done</code></li>
                    <li><strong>Date:</strong> <code>date:today</code>, <code>date:tomorrow</code>, <code>date:2026-01-05</code></li>
                    <li><strong>Show Completed:</strong> <code>show:completed</code></li>
                    <li><strong>Reminder:</strong> <code>has:reminder</code></li>
                    <li><strong>Negate:</strong> prepend <code>!</code></li>
                  </ul>
                </div>
              )}
            </div>
          </div>
          {/* Right section: Count, Display menu, Actions */}
          <div className="flex items-center gap-3 flex-shrink-0">
            <span className="bg-slate-200 text-slate-700 px-2 py-0.5 rounded-full text-xs whitespace-nowrap">
              {tasksToDisplay.length}/{totalTasksForDateView}
              {settings.dateView === "today"
                ? " today"
                : settings.dateView === "tomorrow"
                  ? " tomorrow"
                  : ""} tasks
            </span>
            {settings.selectMode && (
              <span className="bg-blue-100 text-blue-700 px-2 py-0.5 rounded-full text-xs whitespace-nowrap">
                {settings.selectedTaskIds.size} selected
              </span>
            )}
            {/* Display dropdown */}
            <div className="relative">
              <Button
                className="h-9 px-3 text-sm bg-slate-300 rounded-md"
                onClick={() => settings.setShowDisplayMenu((prev: boolean) => !prev)}
              >
                Display ▾
              </Button>
              {settings.showDisplayMenu && (
                <div className="absolute right-0 mt-1 w-64 bg-white border border-slate-300 rounded shadow-lg p-3 z-20">
                  <div className="mb-2">
                    <label className="block text-xs font-semibold mb-1">Date Range</label>
                    <select
                      className="w-full p-1 border border-slate-300 rounded-md text-sm"
                      value={settings.dateView}
                      onChange={handleDateChange}
                    >
                      <option value="today">Today</option>
                      <option value="tomorrow">Tomorrow</option>
                      <option value="all">All</option>
                    </select>
                  </div>
                  <div className="mb-2">
                    <label className="block text-xs font-semibold mb-1">View Mode</label>
                    <select
                      className="w-full p-1 border border-slate-300 rounded-md text-sm"
                      value={settings.viewMode}
                      onChange={(e) => settings.setViewMode(e.target.value as "list" | "matrix" | "kanban")}
                    >
                      <option value="list">List View</option>
                      <option value="matrix">Eisenhower Matrix</option>
                      <option value="kanban">Kanban Board</option>
                    </select>
                  </div>
                  <div className="mb-2">
                    <label className="flex items-center gap-2 text-xs font-semibold">
                      <input
                        type="checkbox"
                        checked={showCompleted}
                        onChange={handleShowCompletedChange}
                        className="rounded"
                      />
                      Show Completed Tasks
                    </label>
                  </div>
                  <div>
                    <label className="block text-xs font-semibold mb-1">Sort By</label>
                    <div className="flex items-center gap-1">
                      <select
                        id="sort-select"
                        className="flex-grow p-1 border border-slate-300 rounded-md text-sm"
                        value={settings.sortField}
                        onChange={handleSortFieldChange}
                      >
                        <option value="updated_at">Updated</option>
                        <option value="title">Name</option>
                        <option value="priority">Priority</option>
                        <option value="status">Status</option>
                        <option value="scheduled_date">Scheduled Date</option>
                        <option value="id">ID</option>
                      </select>
                      <Button onClick={settings.toggleSortDirection} className="p-1 text-xs border border-slate-300 rounded-md">
                        {settings.sortDirection === "asc" ? "↑" : "↓"}
                      </Button>
                    </div>
                  </div>
                </div>
              )}
            </div>
            <Button onClick={toggleShowTaskWindow} className="h-9 bg-blue-600 hover:bg-blue-700 text-white rounded-md px-3 text-sm">
              Add Task
            </Button>
            <div className="h-9 flex items-center">
              <TaskPageOptionsMenu
                tags={tags}
                handleTagClick={handleTagClick}
                tasks={tasksToDisplay}
                selectMode={settings.selectMode}
                selectedTaskIds={settings.selectedTaskIds}
                onSelectAll={() => settings.selectAllTasks(paginatedTasks.map(t => t.id))}
                onClearSelection={settings.clearSelection}
                onToggleSelectMode={settings.toggleSelectMode}
              />
            </div>
          </div>
        </div>
      </div>
      <div>
        {showCreateTaskWindow && (
          <CreateTaskWindow
            currentCard={null}
            setShowTaskWindow={setShowCreateTaskWindow}
            currentFilter={settings.filterString}
          />
        )}
      </div>
      {/* Task Dialog */}
      <TaskDialog
        taskId={selectedTaskId}
        isOpen={isTaskDialogOpen}
        onClose={handleCloseTaskDialog}
        onTagClick={handleTagClick}
      />
      <div className="p-4">
        {settings.viewMode === "list" ? (
          <>
            <ul>
              {paginatedTasks.length > 0 ? (
                <TaskList
                  onTagClick={handleTagClick}
                  tasks={paginatedTasks}
                  selectMode={settings.selectMode}
                  selectedTaskIds={settings.selectedTaskIds}
                  onTaskSelect={settings.toggleTaskSelection}
                />
              ) : (
                <div className="flex justify-center items-center">
                  No tasks, you're done for the day!
                </div>
              )}
            </ul>
            {/* Pagination controls - only show if there are tasks and multiple pages */}
            {tasksToDisplay.length > 0 && totalPages > 1 && (
              <div className="mt-4 flex flex-col sm:flex-row items-center justify-between gap-3 border-t pt-4">
                <div className="text-sm text-slate-600">
                  Showing {((settings.currentPage - 1) * settings.itemsPerPage) + 1}-{Math.min(settings.currentPage * settings.itemsPerPage, tasksToDisplay.length)} of {tasksToDisplay.length} tasks
                </div>
                <div className="flex items-center gap-2">
                  <Button
                    onClick={() => settings.setCurrentPage(1)}
                    disabled={settings.currentPage === 1}
                    className="px-3 py-1 text-sm border border-slate-300 rounded disabled:opacity-50 disabled:cursor-not-allowed"
                  >
                    First
                  </Button>
                  <Button
                    onClick={() => settings.setCurrentPage(prev => Math.max(1, prev - 1))}
                    disabled={settings.currentPage === 1}
                    className="px-3 py-1 text-sm border border-slate-300 rounded disabled:opacity-50 disabled:cursor-not-allowed"
                  >
                    Previous
                  </Button>
                  <span className="text-sm text-slate-600">
                    Page {settings.currentPage} of {totalPages}
                  </span>
                  <Button
                    onClick={() => settings.setCurrentPage(prev => Math.min(totalPages, prev + 1))}
                    disabled={settings.currentPage === totalPages}
                    className="px-3 py-1 text-sm border border-slate-300 rounded disabled:opacity-50 disabled:cursor-not-allowed"
                  >
                    Next
                  </Button>
                  <Button
                    onClick={() => settings.setCurrentPage(totalPages)}
                    disabled={settings.currentPage === totalPages}
                    className="px-3 py-1 text-sm border border-slate-300 rounded disabled:opacity-50 disabled:cursor-not-allowed"
                  >
                    Last
                  </Button>
                  <select
                    value={settings.itemsPerPage}
                    onChange={(e) => settings.setItemsPerPage(Number(e.target.value))}
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
        ) : settings.viewMode === "kanban" ? (
          <KanbanBoard
            onTagClick={handleTagClick}
            tasks={tasksToDisplay}
          />
        ) : (
          <EisenhowerMatrix
            onTagClick={handleTagClick}
            tasks={tasksToDisplay}
            onAddTaskWithTags={(tags: string[]) => {
              settings.setFilterString(tags.join(" "));
              setShowCreateTaskWindow(true);
            }}
          />
        )}
      </div>
    </div >
  );
}
