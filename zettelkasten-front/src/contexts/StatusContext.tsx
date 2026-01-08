import React, { createContext, useState, useContext, useEffect, ReactNode } from 'react';
import { TaskStatus } from '../models/TaskStatus';
import { fetchTaskStatuses } from '../api/taskStatuses';

interface StatusContextType {
  statuses: TaskStatus[];
  loading: boolean;
  error: string | null;
  refreshStatuses: () => Promise<void>;
  getStatusByName: (name: string) => TaskStatus | undefined;
  getDefaultStatus: () => TaskStatus | undefined;
  getCompleteStatus: () => TaskStatus | undefined;
}

const StatusContext = createContext<StatusContextType | undefined>(undefined);

export const StatusProvider: React.FC<{ children: ReactNode }> = ({ children }) => {
  const [statuses, setStatuses] = useState<TaskStatus[]>([]);
  const [loading, setLoading] = useState(true);
  const [error, setError] = useState<string | null>(null);

  const loadStatuses = async () => {
    try {
      setLoading(true);
      setError(null);
      const fetchedStatuses = await fetchTaskStatuses();
      setStatuses(fetchedStatuses);
    } catch (err) {
      console.error('Error fetching task statuses:', err);
      setError('Failed to load task statuses');
      // Fallback to default statuses if fetch fails
      setStatuses([
        {
          id: 0,
          user_id: 0,
          name: 'todo',
          display_name: 'To Do',
          color: '#6B7280',
          icon: '⭕',
          position: 0,
          is_default: true,
          is_complete_state: false,
          created_at: new Date(),
          updated_at: new Date(),
        },
        {
          id: 1,
          user_id: 0,
          name: 'in_progress',
          display_name: 'In Progress',
          color: '#3B82F6',
          icon: '🔄',
          position: 1,
          is_default: false,
          is_complete_state: false,
          created_at: new Date(),
          updated_at: new Date(),
        },
        {
          id: 2,
          user_id: 0,
          name: 'blocked',
          display_name: 'Blocked',
          color: '#EF4444',
          icon: '🚫',
          position: 2,
          is_default: false,
          is_complete_state: false,
          created_at: new Date(),
          updated_at: new Date(),
        },
        {
          id: 3,
          user_id: 0,
          name: 'done',
          display_name: 'Done',
          color: '#10B981',
          icon: '✅',
          position: 3,
          is_default: false,
          is_complete_state: true,
          created_at: new Date(),
          updated_at: new Date(),
        },
      ]);
    } finally {
      setLoading(false);
    }
  };

  useEffect(() => {
    loadStatuses();
  }, []);

  const getStatusByName = (name: string): TaskStatus | undefined => {
    return statuses.find(status => status.name === name);
  };

  const getDefaultStatus = (): TaskStatus | undefined => {
    return statuses.find(status => status.is_default);
  };

  const getCompleteStatus = (): TaskStatus | undefined => {
    return statuses.find(status => status.is_complete_state);
  };

  const value: StatusContextType = {
    statuses,
    loading,
    error,
    refreshStatuses: loadStatuses,
    getStatusByName,
    getDefaultStatus,
    getCompleteStatus,
  };

  return <StatusContext.Provider value={value}>{children}</StatusContext.Provider>;
};

export const useStatus = (): StatusContextType => {
  const context = useContext(StatusContext);
  if (context === undefined) {
    throw new Error('useStatus must be used within a StatusProvider');
  }
  return context;
};
