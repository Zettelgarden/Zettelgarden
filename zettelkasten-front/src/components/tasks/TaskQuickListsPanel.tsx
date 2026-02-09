import React, { useState, useMemo } from "react";
import { Task } from "../../models/Task";
import { useDialogState } from "../../contexts/DialogStateContext";
import { useAuth } from "../../contexts/AuthContext";
import { getToday, getNextWeek, compareDatesInTimezone, isPast } from "../../utils/dates";

interface QuickList {
  id: string;
  title: string;
  description: string;
  tasks: Task[];
  icon: string;
}

interface TaskQuickListsPanelProps {
  tasks: Task[];
  onTagClick?: (tag: string) => void;
  className?: string;
  isCollapsible?: boolean;
  defaultCollapsed?: boolean;
  showOnMobile?: boolean; // For detail view on mobile
}

export function TaskQuickListsPanel({
  tasks,
  onTagClick,
  className = "",
  isCollapsible = true,
  defaultCollapsed = false,
  showOnMobile = false,
}: TaskQuickListsPanelProps) {
  const { user } = useAuth();
  const { setSelectedTaskId, setShowTaskDialog } = useDialogState();
  const [isCollapsed, setIsCollapsed] = useState(defaultCollapsed);
  const [expandedLists, setExpandedLists] = useState<Set<string>>(new Set());

  const userTimezone = user?.timezone || "UTC";

  // Filter tasks into quick lists
  const quickLists = useMemo(() => {
    const today = getToday(userTimezone);
    const nextWeek = getNextWeek(userTimezone);
    const now = new Date();

    // Helper to check if a date is today
    const isToday = (date: Date | null): boolean => {
      if (!date) return false;
      return compareDatesInTimezone(date, today, userTimezone);
    };

    // Helper to check if a date is within the next 7 days
    const isThisWeek = (date: Date | null): boolean => {
      if (!date) return false;
      const taskDate = new Date(date);
      return taskDate >= today && taskDate < nextWeek;
    };

    // Helper to check if a task is overdue
    const isOverdue = (task: Task): boolean => {
      if (task.is_complete || task.is_deleted) return false;
      if (!task.due_date) return false;
      return isPast(task.due_date, userTimezone);
    };

    // Helper to check if task is high priority
    const isHighPriority = (task: Task): boolean => {
      return task.priority === "A" && !task.is_complete && !task.is_deleted;
    };

    // Filter tasks for each quick list
    const todayTasks = tasks.filter(
      (task) =>
        !task.is_complete &&
        !task.is_deleted &&
        (isToday(task.due_date) || isToday(task.scheduled_date))
    );

    const thisWeekTasks = tasks.filter(
      (task) =>
        !task.is_complete &&
        !task.is_deleted &&
        (isThisWeek(task.due_date) || isThisWeek(task.scheduled_date)) &&
        !isToday(task.due_date) &&
        !isToday(task.scheduled_date)
    );

    const overdueTasks = tasks.filter(isOverdue);

    const highPriorityTasks = tasks.filter(isHighPriority);

    const lists: QuickList[] = [
      {
        id: "today",
        title: "Today",
        description: "Due or scheduled for today",
        tasks: todayTasks,
        icon: "📅",
      },
      {
        id: "week",
        title: "This Week",
        description: "Due or scheduled within 7 days",
        tasks: thisWeekTasks,
        icon: "📆",
      },
      {
        id: "overdue",
        title: "Overdue",
        description: "Past due date",
        tasks: overdueTasks,
        icon: "⚠️",
      },
      {
        id: "high-priority",
        title: "High Priority",
        description: "Priority A tasks",
        tasks: highPriorityTasks,
        icon: "🔴",
      },
    ];

    return lists;
  }, [tasks, userTimezone]);

  const toggleListExpanded = (listId: string) => {
    setExpandedLists((prev) => {
      const newSet = new Set(prev);
      if (newSet.has(listId)) {
        newSet.delete(listId);
      } else {
        newSet.add(listId);
      }
      return newSet;
    });
  };

  const handleTaskClick = (taskId: number) => {
    setSelectedTaskId(taskId);
    setShowTaskDialog(true);
  };

  const handleCompleteToggle = (task: Task, e: React.MouseEvent) => {
    e.stopPropagation();
    // This would integrate with existing task completion logic
    // For now, just opening the dialog
    handleTaskClick(task.id);
  };

  if (isCollapsed) {
    return (
      <div className={`bg-white border-l border-slate-200 ${className}`}>
        <button
          onClick={() => setIsCollapsed(false)}
          className="w-full p-3 text-left hover:bg-slate-50 transition-colors"
          aria-label="Expand quick lists panel"
        >
          <span className="text-slate-500">» Quick Lists</span>
        </button>
      </div>
    );
  }

  // Responsive classes: hide on mobile unless showOnMobile is true
  const responsiveClasses = showOnMobile ? "" : "hidden lg:block";

  return (
    <div className={`bg-white border-l border-slate-200 overflow-y-auto ${responsiveClasses} ${className}`}>
      {/* Header */}
      <div className="sticky top-0 bg-white border-b border-slate-200 p-3 flex items-center justify-between z-10">
        <div>
          <h2 className="font-semibold text-slate-800">Quick Lists</h2>
          <p className="text-xs text-slate-500">Fast access to key tasks</p>
        </div>
        {isCollapsible && (
          <button
            onClick={() => setIsCollapsed(true)}
            className="text-slate-400 hover:text-slate-600 p-1"
            aria-label="Collapse panel"
          >
            «
          </button>
        )}
      </div>

      {/* Quick Lists */}
      <div className="p-3 space-y-3">
        {quickLists.map((list) => {
          const isExpanded = expandedLists.has(list.id);
          const displayTasks = isExpanded ? list.tasks : list.tasks.slice(0, 3);
          const hasMore = list.tasks.length > 3;

          return (
            <div
              key={list.id}
              className="border border-slate-200 rounded-lg overflow-hidden"
            >
              {/* List Header */}
              <button
                onClick={() => toggleListExpanded(list.id)}
                className="w-full p-2 bg-slate-50 hover:bg-slate-100 transition-colors flex items-center justify-between text-left"
                aria-expanded={isExpanded}
              >
                <div className="flex items-center gap-2">
                  <span className="text-lg" role="img" aria-label={list.title}>
                    {list.icon}
                  </span>
                  <div>
                    <div className="font-medium text-sm text-slate-700">
                      {list.title}
                      <span className="ml-1 px-1.5 py-0.5 bg-slate-200 text-slate-600 rounded-full text-xs">
                        {list.tasks.length}
                      </span>
                    </div>
                    <div className="text-xs text-slate-500">{list.description}</div>
                  </div>
                </div>
                <span className="text-slate-400">
                  {isExpanded ? "▼" : "▶"}
                </span>
              </button>

              {/* Task Items */}
              {list.tasks.length > 0 ? (
                <div className="divide-y divide-slate-100">
                  {displayTasks.map((task) => (
                    <div
                      key={task.id}
                      onClick={() => handleTaskClick(task.id)}
                      className="p-2 hover:bg-slate-50 cursor-pointer transition-colors"
                    >
                      <div className="flex items-start gap-2">
                        <button
                          onClick={(e) => handleCompleteToggle(task, e)}
                          className={`mt-0.5 w-4 h-4 rounded border flex-shrink-0 flex items-center justify-center transition-colors ${
                            task.is_complete
                              ? "bg-green-500 border-green-500 text-white"
                              : "border-slate-300 hover:border-green-500"
                          }`}
                          aria-label={
                            task.is_complete ? "Mark as incomplete" : "Mark as complete"
                          }
                        >
                          {task.is_complete && (
                            <svg
                              className="w-3 h-3"
                              fill="none"
                              stroke="currentColor"
                              viewBox="0 0 24 24"
                            >
                              <path
                                strokeLinecap="round"
                                strokeLinejoin="round"
                                strokeWidth={3}
                                d="M5 13l4 4L19 7"
                              />
                            </svg>
                          )}
                        </button>
                        <div className="flex-grow min-w-0">
                          <p
                            className={`text-sm truncate ${
                              task.is_complete ? "line-through text-slate-400" : "text-slate-700"
                            }`}
                          >
                            {task.title.replace(/#[\w-]+/g, "")}
                          </p>
                          {task.due_date && (
                            <p className="text-xs text-slate-500">
                              Due: {new Date(task.due_date).toLocaleDateString()}
                            </p>
                          )}
                        </div>
                      </div>
                    </div>
                  ))}
                  {hasMore && !isExpanded && (
                    <button
                      onClick={() => toggleListExpanded(list.id)}
                      className="w-full p-2 text-xs text-blue-600 hover:text-blue-800 hover:bg-blue-50 transition-colors"
                    >
                      +{list.tasks.length - 3} more
                    </button>
                  )}
                </div>
              ) : (
                <div className="p-4 text-center text-sm text-slate-400">
                  No tasks
                </div>
              )}
            </div>
          );
        })}

        {/* Empty State */}
        {quickLists.every((list) => list.tasks.length === 0) && (
          <div className="text-center py-8">
            <p className="text-slate-400 text-sm">No tasks in quick lists</p>
            <p className="text-slate-400 text-xs mt-1">
              Tasks with due dates or priorities will appear here
            </p>
          </div>
        )}
      </div>
    </div>
  );
}
