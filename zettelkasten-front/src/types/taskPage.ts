import type { RefObject, ChangeEvent } from "react";
import type { Task } from "../models/Task";
import type { Tag } from "../models/Tags";
import type { ExternalEvent } from "../models/ExternalEvent";
import type { QuickTagTrigger } from "../components/tasks/QuickTagPopover";

// Type aliases for clarity
export type SortField = "updated_at" | "title" | "priority" | "status" | "id" | "scheduled_date" | "due_date";
export type SortDirection = "asc" | "desc";
export type ViewMode = "list" | "matrix" | "kanban" | "calendar";
export type CalendarViewMode = "month" | "week";
export type TaskMobileView = "list" | "filters";

/**
 * Task data passed from TaskPage to layout components
 */
export interface TaskData {
  tasks: Task[];
  tags: Tag[];
  userTimezone: string;
  isLoading?: boolean;
  tasksToDisplay: Task[];
  paginatedTasks: Task[];
  totalTasksForDateView: number;
  totalPages: number;
}

/**
 * External calendar events state
 */
export interface ExternalEventsState {
  externalEvents: ExternalEvent[];
  isLoadingEvents: boolean;
}

/**
 * View settings (sorting, pagination, calendar, etc.)
 */
export interface ViewSettings {
  dateView: string;
  viewMode: ViewMode;
  sortField: SortField;
  sortDirection: SortDirection;
  calendarViewMode: CalendarViewMode;
  calendarCurrentDate: Date;
  currentPage: number;
  itemsPerPage: number;
  showDisplayMenu: boolean;
}

/**
 * View settings setters
 */
export interface ViewSettingsSetters {
  setDateView: (view: string | ((prev: string) => string)) => void;
  setViewMode: (mode: ViewMode | ((prev: ViewMode) => ViewMode)) => void;
  setSortField: (field: SortField | ((prev: SortField) => SortField)) => void;
  setSortDirection: (direction: SortDirection | ((prev: SortDirection) => SortDirection)) => void;
  setCalendarViewMode: (mode: CalendarViewMode | ((prev: CalendarViewMode) => CalendarViewMode)) => void;
  setCalendarCurrentDate: (date: Date | ((prev: Date) => Date)) => void;
  setCurrentPage: (page: number | ((prev: number) => number)) => void;
  setItemsPerPage: (items: number | ((prev: number) => number)) => void;
  setShowDisplayMenu: (show: boolean | ((prev: boolean) => boolean)) => void;
}

/**
 * Dialog state (modals, popups, etc.)
 */
export interface DialogState {
  showCreateTaskWindow: boolean;
  selectedTaskId: number | null;
  isTaskDialogOpen: boolean;
  createTaskStatus: string | undefined;
  calendarSelectedDate: Date | null;
}

/**
 * Dialog state setters
 */
export interface DialogSetters {
  setShowCreateTaskWindow: (show: boolean | ((prev: boolean) => boolean)) => void;
  setSelectedTaskId: (taskId: number | null) => void;
  setIsTaskDialogOpen: (open: boolean | ((prev: boolean) => boolean)) => void;
  setCreateTaskStatus: (status: string | undefined) => void;
  setCalendarSelectedDate: (date: Date | null) => void;
}

/**
 * Selection state (bulk operations)
 */
export interface SelectionState {
  selectMode: boolean;
  selectedTaskIds: Set<number>;
}

/**
 * Selection state setters and actions
 */
export interface SelectionActions {
  setSelectMode: (mode: boolean | ((prev: boolean) => boolean)) => void;
  setSelectedTaskIds: (ids: Set<number> | ((prev: Set<number>) => Set<number>)) => void;
  toggleSelectMode: () => void;
  toggleTaskSelection: (taskId: number) => void;
  selectAllTasks: (taskIds: number[]) => void;
  clearSelection: () => void;
}

/**
 * Filter state
 */
export interface FilterState {
  filterString: string;
  showFilterHelp: boolean;
  showCompleted: boolean;
}

/**
 * Filter state setters
 */
export interface FilterSetters {
  setFilterString: (filter: string | ((prev: string) => string)) => void;
  setShowFilterHelp: (show: boolean | ((prev: boolean) => boolean)) => void;
  setShowCompleted: (show: boolean | ((prev: boolean) => boolean)) => void;
}

/**
 * Quick tag autocomplete state for filter input
 */
export interface FilterInputState {
  filterInputRef: RefObject<HTMLInputElement>;
  cursorPosition: number;
  filterTrigger: QuickTagTrigger | null;
  isFilterFocused: boolean;
}

/**
 * Filter input setters
 */
export interface FilterInputSetters {
  setFilterTrigger: (trigger: QuickTagTrigger | null) => void;
  setIsFilterFocused: (focused: boolean) => void;
}

/**
 * Filter input handlers
 */
export interface FilterInputHandlers {
  onFilterChange: (e: ChangeEvent<HTMLInputElement | HTMLTextAreaElement>) => void;
  onSelectQuickTag: (tagName: string) => void;
  onRefreshFilterTriggerFromInput: (input: HTMLInputElement) => void;
}

/**
 * Navigation actions
 */
export interface NavigationActions {
  navigateCalendar: (direction: "prev" | "next" | "today") => void;
  toggleSortDirection: () => void;
}

/**
 * External events setters
 */
export interface ExternalEventsSetters {
  setExternalEvents: (events: ExternalEvent[] | ((prev: ExternalEvent[]) => ExternalEvent[])) => void;
  setIsLoadingEvents: (loading: boolean | ((prev: boolean) => boolean)) => void;
  setRefreshTasks: (refresh: boolean) => void;
}

/**
 * Event handlers passed to layout components
 */
export interface TaskPageHandlers {
  onTagClick: (tag: string) => void;
  onAddTaskWithStatus: (status: string) => void;
  onCloseTaskDialog: () => void;
}

/**
 * Desktop-specific handlers
 */
export interface DesktopHandlers extends TaskPageHandlers {
  onDateChange: (e: ChangeEvent<HTMLSelectElement>) => void;
  onSortFieldChange: (e: ChangeEvent<HTMLSelectElement>) => void;
  onShowCompletedChange: () => void;
}

/**
 * Mobile-specific state
 */
export interface MobileState {
  mobileView: TaskMobileView;
}

/**
 * Mobile-specific setters
 */
export interface MobileSetters {
  setMobileView: (view: TaskMobileView) => void;
}

/**
 * Mobile-specific handlers
 */
export interface MobileHandlers extends TaskPageHandlers {
  onMenuClick: () => void;
  onTaskClick: (taskId: number) => void;
}
