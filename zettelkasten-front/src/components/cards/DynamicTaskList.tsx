import React, { useMemo, useState, useEffect } from 'react';
import { fetchTasks } from '../../api/tasks';
import { filterTasks, filterTasksByDateView } from '../../utils/tasks';
import { compareDates } from '../../utils/dates';
import { TaskListItem } from '../tasks/TaskListItem';
import { Task } from '../../models/Task';

interface DynamicTaskListProps {
  query: string;
}

interface FilterParams {
  searchTerms: string[];
  dateView: "all" | "today" | "tomorrow";
  showCompleted: boolean;
  specificDate: Date | null;
}

function parseQuery(query: string): FilterParams {
  // Default values
  const params: FilterParams = {
    searchTerms: [],
    dateView: "all",
    showCompleted: false,
    specificDate: null,
  };

  // Split query into tokens
  const tokens = query.split(' ').map(t => t.trim()).filter(t => t !== '');

  // Extract specific date if present (date:YYYY-MM-DD)
  const dateTokenIndex = tokens.findIndex(t => t.startsWith('date:'));
  if (dateTokenIndex !== -1) {
    const dateToken = tokens[dateTokenIndex];
    const dateString = dateToken.substring('date:'.length);
    try {
      // Parse ISO date format (YYYY-MM-DD)
      const parsedDate = new Date(dateString);
      if (!isNaN(parsedDate.getTime())) {
        params.specificDate = parsedDate;
      }
    } catch (error) {
      console.error('Invalid date format:', dateString);
    }
    // Remove from tokens
    tokens.splice(dateTokenIndex, 1);
  }

  // Extract date view if present (only if no specific date)
  if (!params.specificDate) {
    const dateViewToken = tokens.find(t =>
      t === 'today' || t === 'tomorrow' || t === 'all'
    );
    if (dateViewToken) {
      params.dateView = dateViewToken as "all" | "today" | "tomorrow";
      // Remove from tokens
      const index = tokens.indexOf(dateViewToken);
      tokens.splice(index, 1);
    }
  }

  // Check for completed flag
  const completedIndex = tokens.indexOf('completed');
  if (completedIndex !== -1) {
    params.showCompleted = true;
    tokens.splice(completedIndex, 1);
  }

  // Remaining tokens are search terms
  params.searchTerms = tokens;

  return params;
}

export const DynamicTaskList: React.FC<DynamicTaskListProps> = ({ query }) => {
  const [allTasks, setAllTasks] = useState<Task[]>([]);
  const [isLoading, setIsLoading] = useState(true);

  // Fetch all tasks (including completed) whenever the component mounts or query changes
  useEffect(() => {
    const loadTasks = async () => {
      setIsLoading(true);
      try {
        // Always fetch with showCompleted=true to get all tasks
        const tasks = await fetchTasks(true);
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

  // Parse query and filter tasks
  const filteredTasks = useMemo(() => {
    try {
      const params = parseQuery(query);

      // First apply text/tag/priority filters
      let filtered = filterTasks(allTasks, params.searchTerms.join(' '));

      // Apply specific date filter if present
      if (params.specificDate) {
        filtered = filtered.filter(task => {
          if (params.showCompleted) {
            // When "completed" flag is present, match against completed_at
            return compareDates(task.completed_at, params.specificDate);
          } else {
            // By default, match against scheduled_date
            return compareDates(task.scheduled_date, params.specificDate);
          }
        });
      } else {
        // Apply date view filters (today, tomorrow, all)
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
