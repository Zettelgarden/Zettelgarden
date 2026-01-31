import { useState, useEffect, useMemo, useCallback, Dispatch, SetStateAction } from "react";

const STORAGE_KEY = "taskPageSettings";

type SortField = "updated_at" | "title" | "priority" | "status" | "id" | "scheduled_date" | "due_date";
type SortDirection = "asc" | "desc";
type ViewMode = "list" | "matrix" | "kanban" | "calendar";

interface TaskPageSettings {
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
}

interface TaskPageSettingsReturn extends TaskPageSettings {
  setDateView: Dispatch<SetStateAction<string>>;
  setFilterString: Dispatch<SetStateAction<string>>;
  setSortField: Dispatch<SetStateAction<SortField>>;
  setSortDirection: Dispatch<SetStateAction<SortDirection>>;
  setViewMode: Dispatch<SetStateAction<ViewMode>>;
  setCalendarViewMode: Dispatch<SetStateAction<"month" | "week">>;
  setCalendarCurrentDate: Dispatch<SetStateAction<Date>>;
  navigateCalendar: (direction: "prev" | "next" | "today") => void;
  setCurrentPage: Dispatch<SetStateAction<number>>;
  setItemsPerPage: Dispatch<SetStateAction<number>>;
  setShowFilterHelp: Dispatch<SetStateAction<boolean>>;
  setShowDisplayMenu: Dispatch<SetStateAction<boolean>>;
  setSelectMode: Dispatch<SetStateAction<boolean>>;
  setSelectedTaskIds: Dispatch<SetStateAction<Set<number>>>;
  toggleSortDirection: () => void;
  toggleSelectMode: () => void;
  toggleTaskSelection: (taskId: number) => void;
  selectAllTasks: (taskIds: number[]) => void;
  clearSelection: () => void;
  resetPage: () => void;
}

export function useTaskPageSettings(): TaskPageSettingsReturn {
  // Load saved settings from localStorage
  const savedSettings = useMemo(() => {
    try {
      return JSON.parse(localStorage.getItem(STORAGE_KEY) || "{}");
    } catch {
      return {};
    }
  }, []);

  // Initialize state with saved values or defaults
  const [dateView, setDateView] = useState<string>(savedSettings.dateView || "today");
  const [filterString, setFilterString] = useState<string>(savedSettings.filterString || "");
  const [sortField, setSortField] = useState<SortField>(savedSettings.sortField || "priority");
  const [sortDirection, setSortDirection] = useState<SortDirection>(savedSettings.sortDirection || "asc");
  const [viewMode, setViewMode] = useState<ViewMode>(savedSettings.viewMode || "list");
  const [calendarViewMode, setCalendarViewMode] = useState<"month" | "week">(savedSettings.calendarViewMode || "month");
  const [calendarCurrentDate, setCalendarCurrentDate] = useState<Date>(savedSettings.calendarCurrentDate ? new Date(savedSettings.calendarCurrentDate) : new Date());
  const [currentPage, setCurrentPage] = useState<number>(1);
  const [itemsPerPage, setItemsPerPage] = useState<number>(savedSettings.itemsPerPage || 50);
  const [showFilterHelp, setShowFilterHelp] = useState<boolean>(false);
  const [showDisplayMenu, setShowDisplayMenu] = useState<boolean>(false);
  const [selectMode, setSelectMode] = useState<boolean>(false);
  const [selectedTaskIds, setSelectedTaskIds] = useState<Set<number>>(new Set());

  // Persist settings to localStorage whenever they change
  useEffect(() => {
    const settings = {
      dateView,
      viewMode,
      calendarViewMode,
      calendarCurrentDate: calendarCurrentDate.toISOString(),
      filterString,
      sortField,
      sortDirection,
      itemsPerPage
    };
    localStorage.setItem(STORAGE_KEY, JSON.stringify(settings));
  }, [dateView, viewMode, calendarViewMode, calendarCurrentDate, filterString, sortField, sortDirection, itemsPerPage]);

  // Reset to page 1 when filters change
  useEffect(() => {
    setCurrentPage(1);
  }, [dateView, filterString, sortField, sortDirection, viewMode]);

  // Clear selection when switching to matrix view
  useEffect(() => {
    if (viewMode === "matrix") {
      setSelectMode(false);
      setSelectedTaskIds(new Set());
    }
  }, [viewMode]);

  // Action functions
  const toggleSortDirection = useCallback(() => {
    setSortDirection(prev => prev === "asc" ? "desc" : "asc");
  }, []);

  const toggleSelectMode = useCallback(() => {
    setSelectMode(prev => {
      const newValue = !prev;
      // Clear selections when exiting select mode
      if (!newValue) {
        setSelectedTaskIds(new Set());
      }
      return newValue;
    });
  }, []);

  const toggleTaskSelection = useCallback((taskId: number) => {
    setSelectedTaskIds(prev => {
      const newSet = new Set(prev);
      if (newSet.has(taskId)) {
        newSet.delete(taskId);
      } else {
        newSet.add(taskId);
      }
      return newSet;
    });
  }, []);

  const selectAllTasks = useCallback((taskIds: number[]) => {
    setSelectedTaskIds(new Set(taskIds));
  }, []);

  const clearSelection = useCallback(() => {
    setSelectedTaskIds(new Set());
  }, []);

  const resetPage = useCallback(() => {
    setCurrentPage(1);
  }, []);

  const navigateCalendar = useCallback((direction: "prev" | "next" | "today") => {
    if (direction === "today") {
      setCalendarCurrentDate(new Date());
    } else if (direction === "prev") {
      setCalendarCurrentDate(prev => {
        const newDate = new Date(prev);
        if (calendarViewMode === "month") {
          newDate.setMonth(newDate.getMonth() - 1);
        } else {
          newDate.setDate(newDate.getDate() - 7);
        }
        return newDate;
      });
    } else if (direction === "next") {
      setCalendarCurrentDate(prev => {
        const newDate = new Date(prev);
        if (calendarViewMode === "month") {
          newDate.setMonth(newDate.getMonth() + 1);
        } else {
          newDate.setDate(newDate.getDate() + 7);
        }
        return newDate;
      });
    }
  }, [calendarViewMode]);

  return {
    // State
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

    // Setters
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

    // Actions
    toggleSortDirection,
    toggleSelectMode,
    toggleTaskSelection,
    selectAllTasks,
    clearSelection,
    resetPage,
    navigateCalendar,
  };
}
