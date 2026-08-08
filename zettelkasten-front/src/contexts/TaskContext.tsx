import React, { createContext, useState, useEffect, useContext } from 'react';
import { Task } from '../models/Task';

import { fetchTasks } from '../api/tasks';

const TaskContext = createContext<TaskContextType | undefined>(undefined);

interface TaskContextType {
  tasks: Task[];
  isLoading: boolean;
  refreshTasks: boolean;
  setRefreshTasks: (refresh: boolean) => void;
  getTasks: () => Promise<void>;
  existingTags: string[];
  showCompleted: boolean;
  setShowCompleted: (refresh: boolean) => void;
  updateTask: (updatedTask: Task) => void;
}
interface TaskProviderProps {
  children: React.ReactNode;
  testing?: boolean; // Add this line
  testTasks?: Task[];
}

export const TaskProvider: React.FC<TaskProviderProps> = ({
  children,
  testing = false,
  testTasks = [],
}) => {
  const [tasks, setTasks] = useState<Task[]>([]);
  const [isLoading, setIsLoading] = useState(true);
  const [refreshTasks, setRefreshTasks] = useState(false);
  const [existingTags, setExistingTags] = useState<string[]>([]);
  const [showCompleted, setShowCompleted] = useState<boolean>(false);

  const extractTags = async (data: Task[]) => {
    let tagSet = new Set<string>();

    data.forEach((task) => {
      const tagsInTitle = task.title.match(/(^|\s)#\w+(\s|$)/g);

      if (tagsInTitle) {
        tagsInTitle.forEach((tag) => tagSet.add(tag));
      }
    });
    const sortedTags = Array.from(tagSet).sort();

    setExistingTags(sortedTags);
  };
  const getTasks = async () => {
    setIsLoading(true);
    try {
      const data = await fetchTasks({ showCompleted });
      setTasks(data);
      extractTags(data);
    } finally {
      setIsLoading(false);
      setRefreshTasks(false);
    }
  };

  const updateTask = (updatedTask: Task) => {
    setTasks((prevTasks) => {
      const newTasks = prevTasks.map((task) =>
        task.id === updatedTask.id ? updatedTask : task,
      );
      extractTags(newTasks);
      return newTasks;
    });
  };
  useEffect(() => {
    if (testing) {
      setTasks(testTasks);
      extractTags(testTasks);
      return;
    }
    // Load tasks immediately when showCompleted changes or on mount
    getTasks();

    const intervalId = setInterval(() => {
      getTasks();
    }, 60000);

    return () => clearInterval(intervalId); // Cleanup on component unmount
  }, [refreshTasks, showCompleted]);

  return (
    <TaskContext.Provider
      value={{
        tasks,
        isLoading,
        refreshTasks,
        setRefreshTasks,
        getTasks,
        existingTags,
        showCompleted,
        setShowCompleted,
        updateTask,
      }}
    >
      {children}
    </TaskContext.Provider>
  );
};

export const useTaskContext = () => {
  const context = useContext(TaskContext);
  if (context === undefined) {
    throw new Error('useTaskContext must be used within a TaskProvider');
  }
  return context;
};
