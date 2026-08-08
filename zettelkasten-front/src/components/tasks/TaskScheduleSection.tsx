import React from 'react';
import { Task } from '../../models/Task';
import { TaskDateDisplay } from './TaskDateDisplay';
import { TaskPriorityDisplay } from './TaskPriorityDisplay';
import { TaskReminderDisplay } from './TaskReminderDisplay';
import { TaskStatusDisplay } from './TaskStatusDisplay';
import { TaskTagDisplay } from './TaskTagDisplay';

interface TaskScheduleSectionProps {
  task: Task;
  setTask: (task: Task) => void;
  saveOnChange: boolean;
  showTagEditor: boolean;
  setShowTagEditor: (show: boolean) => void;
  onRemoveTag: (tagName: string) => Promise<void>;
}

export function TaskScheduleSection({
  task,
  setTask,
  saveOnChange,
  showTagEditor,
  setShowTagEditor,
  onRemoveTag,
}: TaskScheduleSectionProps) {
  return (
    <div className="flex flex-col sm:flex-row items-stretch sm:items-center gap-3 flex-wrap">
      <TaskStatusDisplay
        task={task}
        setTask={setTask}
        saveOnChange={saveOnChange}
      />
      <TaskDateDisplay
        task={task}
        setTask={setTask}
        saveOnChange={saveOnChange}
      />
      <TaskPriorityDisplay
        task={task}
        setTask={setTask}
        saveOnChange={saveOnChange}
      />
      <TaskReminderDisplay
        task={task}
        setTask={setTask}
        saveOnChange={saveOnChange}
      />
      <TaskTagDisplay
        task={task}
        tags={task.tags}
        onTagClick={() => {}}
        onRemoveTag={onRemoveTag}
      />
      <button
        onClick={() => setShowTagEditor(!showTagEditor)}
        className="text-sm text-blue-600 hover:text-blue-800 font-medium"
      >
        {showTagEditor ? '- Hide Tags' : '+ Add Tags'}
      </button>
    </div>
  );
}
