import React, { useMemo, useState, useEffect, useRef } from 'react';
import { fetchTasks } from '../../api/tasks';
import {
  filterTasks,
  filterTasksByDateView,
  parseTaskQuery,
} from '../../utils/tasks';
import { TaskListItem } from '../tasks/TaskListItem';
import { Task } from '../../models/Task';
import { CreateTaskWindow } from '../tasks/CreateTaskWindow';

const taskQueryCache = new Map<string, Task[]>();

interface DynamicTaskListProps {
  query: string;
}

export const DynamicTaskList: React.FC<DynamicTaskListProps> = ({ query }) => {
  const [allTasks, setAllTasks] = useState<Task[]>(
    () => taskQueryCache.get(query) ?? [],
  );
  const [isLoading, setIsLoading] = useState(() => !taskQueryCache.has(query));
  const [isRefreshing, setIsRefreshing] = useState(false);
  const hasLoadedOnceRef = useRef(taskQueryCache.has(query));
  const [showTaskWindow, setShowTaskWindow] = useState(false);

  // Fetch tasks with backend filtering based on query
  useEffect(() => {
    let cancelled = false;

    const loadTasks = async () => {
      const cached = taskQueryCache.get(query);
      if (cached) {
        setAllTasks(cached);
        hasLoadedOnceRef.current = true;
        setIsLoading(false);
      }

      // Avoid blanking out the list while refreshing.
      if (!hasLoadedOnceRef.current) {
        setIsLoading(true);
      } else {
        setIsRefreshing(true);
      }

      if (cancelled) return;

      try {
        const params = parseTaskQuery(query);

        // Determine which date parameter to use
        let scheduledDate: Date | null = null;
        let completedDate: Date | null = null;

        if (params.specificDate) {
          if (params.showCompleted) {
            // When "completed" flag is present, filter by completed_at
            completedDate = params.specificDate;
          } else {
            // By default, filter by scheduled_date
            scheduledDate = params.specificDate;
          }
        }

        const tasks = await fetchTasks({
          showCompleted: params.showCompleted,
          scheduledDate,
          completedDate,
        });

        if (cancelled) return;
        taskQueryCache.set(query, tasks);
        setAllTasks(tasks);
      } catch (error) {
        if (cancelled) return;
        console.error('Error fetching tasks:', error);
        if (!hasLoadedOnceRef.current) {
          setAllTasks([]);
        }
      } finally {
        if (cancelled) return;
        setIsLoading(false);
        setIsRefreshing(false);
        hasLoadedOnceRef.current = true;
      }
    };

    loadTasks();

    return () => {
      cancelled = true;
    };
  }, [query]);

  const shouldShowInitialLoading = isLoading && !hasLoadedOnceRef.current;

  // Parse query and apply client-side filters (text/tag search and date views)
  const filteredTasks = useMemo(() => {
    try {
      const params = parseTaskQuery(query);

      // Apply text/tag/priority filters
      let filtered = filterTasks(allTasks, params.searchTerms.join(' '));

      // Only apply date view filters if no specific date was provided
      // (specific dates are already filtered by the backend)
      if (!params.specificDate && params.dateView !== 'all') {
        filtered = filtered.filter((task) =>
          filterTasksByDateView(task, params.dateView, params.showCompleted),
        );
      }

      return filtered;
    } catch (error) {
      console.error('Error filtering tasks:', error);
      return [];
    }
  }, [query, allTasks]);

  const shouldShowNoResults =
    !shouldShowInitialLoading && !isRefreshing && filteredTasks.length === 0;

  // If we're refreshing, keep showing the previous tasks (don’t flash the full loading state).

  // Handle loading state
  if (shouldShowInitialLoading) {
    return (
      <div className="bg-gray-50 rounded-lg p-4 my-4 border border-gray-200">
        <p className="text-sm text-gray-500 italic">Loading tasks...</p>
      </div>
    );
  }

  // Handle no results
  if (shouldShowNoResults) {
    return (
      <div className="bg-gray-50 rounded-lg p-4 my-4 border border-gray-200">
        <p className="text-sm text-gray-500 italic">
          No tasks match query: "{query}"
        </p>
      </div>
    );
  }

  // Render task list
  return (
    <>
      <div className="bg-white rounded-lg shadow-sm my-4 border border-gray-200">
        <div className="p-4">
          <div className="flex justify-between items-center mb-2">
            <div className="text-xs text-gray-500">
              Tasks matching: "{query}" ({filteredTasks.length})
              {isRefreshing && (
                <span className="ml-2 italic">Refreshing...</span>
              )}
            </div>
            <button
              onClick={() => setShowTaskWindow(true)}
              className="px-3 py-1 text-xs bg-blue-500 text-white rounded hover:bg-blue-600 transition-colors"
            >
              + New Task
            </button>
          </div>
          <div className="space-y-2">
            {filteredTasks.map((task) => (
              <div
                key={task.id}
                className="border-b border-gray-100 last:border-0 pb-2 last:pb-0"
              >
                <TaskListItem
                  task={task}
                  onTagClick={(tag: string) => {}}
                  hideMatrixTags={false}
                />
              </div>
            ))}
          </div>
        </div>
      </div>
      {showTaskWindow && (
        <CreateTaskWindow
          currentCard={null}
          setShowTaskWindow={setShowTaskWindow}
          currentFilter={query}
        />
      )}
    </>
  );
};
