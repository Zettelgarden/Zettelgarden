import React from 'react';
import { Task } from '../../models/Task';
import { useOptimisticTaskUpdate } from '../../hooks/useOptimisticTaskUpdate';
import { TaskDropdown } from './TaskDropdown';
import { useStatus } from '../../contexts/StatusContext';

interface TaskStatusDisplayProps {
  task: Task;
  setTask: (task: Task) => void;
  saveOnChange: boolean;
}

export function TaskStatusDisplay({
  task,
  setTask,
  saveOnChange,
}: TaskStatusDisplayProps) {
  const { updateTask } = useOptimisticTaskUpdate({
    task,
    setTask,
    saveOnChange,
    errorMessagePrefix: 'Failed to update task status',
  });
  const { statuses, getStatusByName } = useStatus();

  // Get status config from dynamic statuses
  const statusDisplay = React.useMemo(() => {
    const currentStatus = getStatusByName(task.status);
    return currentStatus
      ? {
          text: currentStatus.display_name,
          color: currentStatus.color,
          icon: currentStatus.icon,
        }
      : { text: 'Unknown', color: '#6B7280', icon: '⭕' };
  }, [task.status, getStatusByName]);

  async function setStatus(statusName: string, close: () => void) {
    const statusConfig = getStatusByName(statusName);
    const editedTask = { ...task, status: statusName };

    // Sync is_complete with status based on is_complete_state
    if (statusConfig) {
      editedTask.is_complete = statusConfig.is_complete_state;
    }

    await updateTask(editedTask);
    close();
  }

  return (
    <div className="relative inline-block pr-2">
      <TaskDropdown display={statusDisplay}>
        {({ close }) => (
          <>
            {statuses.map((status) => (
              <div
                key={status.id}
                className="px-2 py-1 min-h-[26px] hover:bg-gray-100 cursor-pointer flex items-center gap-1.5 text-xs overflow-hidden"
                onClick={() => setStatus(status.name, close)}
                style={{ color: status.color }}
              >
                <span>{status.icon}</span>
                <span className="truncate">{status.display_name}</span>
              </div>
            ))}
          </>
        )}
      </TaskDropdown>
    </div>
  );
}
