import React, { useMemo, useState, useEffect } from 'react';
import { fetchTasks } from '../../api/tasks';
import { filterTasks, filterTasksByDateView, parseTaskQuery } from '../../utils/tasks';
import { compareDates } from '../../utils/dates';
import { TaskListItem } from '../tasks/TaskListItem';
import { Task } from '../../models/Task';

interface DynamicTaskListProps {
  query: string;
}

export const DynamicTaskList: React.FC<DynamicTaskListProps> = ({ query }) => {
  const [allTasks, setAllTasks] = useState<Task[]>([]);
  const [isLoading, setIsLoading] = useState(true);

  // Fetch tasks with backend filtering based on query
  useEffect(() => {
    const loadTasks = async () => {
      setIsLoading(true);
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
        setAllTasks(tasks);
      } catch (error) {
        console.error('Error fetching tasks:', error);
        setAllTasks([]);
      } finally {
        setIsLoading(false);
      }
    };

    loadTasks();
  }, [query]);

  // Parse query and apply client-side filters (text/tag search and date views)
  const filteredTasks = useMemo(() => {
    try {
      const params = parseTaskQuery(query);

      // Apply text/tag/priority filters
      let filtered = filterTasks(allTasks, params.searchTerms.join(' '));

      // Only apply date view filters if no specific date was provided
      // (specific dates are already filtered by the backend)
      if (!params.specificDate && params.dateView !== 'all') {
        filtered = filtered.filter(task =>
          filterTasksByDateView(task, params.dateView, params.showCompleted)
        );
      }

      return filtered;
    } catch (error) {
      console.error('Error filtering tasks:', error);
      return [];
    }
  }, [query, allTasks]);

  // Handle loading state
  if (isLoading) {
    return (
      <div className="bg-gray-50 rounded-lg p-4 my-4 border border-gray-200">
        <p className="text-sm text-gray-500 italic">
          Loading tasks...
        </p>
      </div>
    );
  }

  // Handle no results
  if (filteredTasks.length === 0) {
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
    <div className="bg-white rounded-lg shadow-sm my-4 border border-gray-200">
      <div className="p-4">
        <div className="text-xs text-gray-500 mb-2">
          Tasks matching: "{query}" ({filteredTasks.length})
        </div>
        <div className="space-y-2">
          {filteredTasks.map(task => (
            <div key={task.id} className="border-b border-gray-100 last:border-0 pb-2 last:pb-0">
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
  );
};
