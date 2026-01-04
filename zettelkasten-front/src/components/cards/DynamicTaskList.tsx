import React, { useMemo } from 'react';
import { useTaskContext } from '../../contexts/TaskContext';
import { filterTasks, filterTasksByDateView } from '../../utils/tasks';
import { TaskListItem } from '../tasks/TaskListItem';
import { Task } from '../../models/Task';

interface DynamicTaskListProps {
  query: string;
}

interface FilterParams {
  searchTerms: string[];
  dateView: "all" | "today" | "tomorrow";
  showCompleted: boolean;
}

function parseQuery(query: string): FilterParams {
  // Default values
  const params: FilterParams = {
    searchTerms: [],
    dateView: "all",
    showCompleted: false,
  };

  // Split query into tokens
  const tokens = query.split(' ').map(t => t.trim()).filter(t => t !== '');

  // Extract date view if present
  const dateViewToken = tokens.find(t =>
    t === 'today' || t === 'tomorrow' || t === 'all'
  );
  if (dateViewToken) {
    params.dateView = dateViewToken as "all" | "today" | "tomorrow";
    // Remove from tokens
    const index = tokens.indexOf(dateViewToken);
    tokens.splice(index, 1);
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
  const { tasks } = useTaskContext();

  // Parse query and filter tasks
  const filteredTasks = useMemo(() => {
    try {
      const params = parseQuery(query);

      // First apply text/tag/priority filters
      let filtered = filterTasks(tasks, params.searchTerms.join(' '));

      // Then apply date view filters
      filtered = filtered.filter(task =>
        filterTasksByDateView(task, params.dateView, params.showCompleted)
      );

      return filtered;
    } catch (error) {
      console.error('Error filtering tasks:', error);
      return [];
    }
  }, [query, tasks]);

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
