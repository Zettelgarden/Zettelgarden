import React from 'react';
import { Task } from '../../models/Task';

interface TaskDependenciesSectionProps {
  task: Task;
  mode: 'create' | 'edit';
  showDependencyEditor: boolean;
  setShowDependencyEditor: (show: boolean) => void;
  dependencyFilter: string;
  setDependencyFilter: (filter: string) => void;
  tasks: Task[];
  onAddDependency: (blockingTaskId: number) => Promise<void>;
  onRemoveDependency: (blockingTaskId: number) => Promise<void>;
}

export function TaskDependenciesSection({
  task,
  mode,
  showDependencyEditor,
  setShowDependencyEditor,
  dependencyFilter,
  setDependencyFilter,
  tasks,
  onAddDependency,
  onRemoveDependency,
}: TaskDependenciesSectionProps) {
  async function handleRemoveDependency(blockingTaskId: number) {
    await onRemoveDependency(blockingTaskId);
  }

  async function handleAddDependency(blockingTaskId: number) {
    await onAddDependency(blockingTaskId);
  }

  return (
    <>
      {/* Blocked By Section (edit mode only) */}
      {mode === 'edit' && task.blocked_by && task.blocked_by.length > 0 && (
        <div className="flex items-center gap-2 flex-wrap">
          <span className="text-sm font-medium text-gray-700">Blocked by:</span>
          {task.blocked_by.map((blockingTask) => (
            <div
              key={blockingTask.id}
              className="inline-flex items-center gap-1 px-2 py-1 bg-orange-100 text-orange-800 rounded text-sm"
            >
              <span className={blockingTask.is_complete ? 'line-through' : ''}>
                {blockingTask.title}
              </span>
              <button
                onClick={() => handleRemoveDependency(blockingTask.id)}
                className="ml-1 hover:text-orange-600"
                title="Remove blocker"
              >
                x
              </button>
            </div>
          ))}
        </div>
      )}

      {/* Add Blocker Button (edit mode only) */}
      {mode === 'edit' && task.id > 0 && (
        <button
          onClick={() => {
            setShowDependencyEditor(!showDependencyEditor);
            if (showDependencyEditor) {
              setDependencyFilter('');
            }
          }}
          className="text-sm text-blue-600 hover:text-blue-800 font-medium w-fit"
        >
          {showDependencyEditor ? '- Hide Blockers' : '+ Add Blocker'}
        </button>
      )}

      {/* Dependency Editor (edit mode only) */}
      {mode === 'edit' && showDependencyEditor && (
        <div className="border border-gray-200 rounded-lg p-4 bg-gray-50 space-y-3">
          <label className="block text-sm font-medium text-gray-700">
            Select tasks that block this task
          </label>
          <input
            type="text"
            value={dependencyFilter}
            onChange={(e) => setDependencyFilter(e.target.value)}
            placeholder="Filter tasks..."
            className="w-full px-3 py-2 border border-gray-300 rounded-md focus:outline-none focus:ring-2 focus:ring-blue-500"
          />
          <div className="max-h-48 overflow-y-auto border border-gray-200 rounded-md p-3 bg-white">
            {(() => {
              const availableTasks = tasks
                .filter((t) => t.id !== task.id && !t.is_complete)
                .filter(
                  (t) =>
                    dependencyFilter === '' ||
                    t.title
                      .toLowerCase()
                      .includes(dependencyFilter.toLowerCase()),
                );

              if (availableTasks.length === 0) {
                return (
                  <p className="text-gray-500 text-sm text-center py-2">
                    {dependencyFilter
                      ? 'No tasks match your search'
                      : 'No available tasks'}
                  </p>
                );
              }

              return (
                <div className="space-y-2">
                  {availableTasks.map((t) => {
                    const isBlocking =
                      task.blocked_by?.some((bt) => bt.id === t.id) || false;
                    return (
                      <button
                        key={t.id}
                        onClick={() => {
                          if (isBlocking) {
                            handleRemoveDependency(t.id);
                          } else {
                            handleAddDependency(t.id);
                          }
                        }}
                        className={`w-full text-left px-3 py-2 rounded transition-colors ${
                          isBlocking
                            ? 'bg-orange-100 text-orange-800 border border-orange-300'
                            : 'bg-gray-100 text-gray-700 hover:bg-gray-200'
                        }`}
                      >
                        <div className="flex items-center justify-between">
                          <span>{t.title}</span>
                          {isBlocking && (
                            <span className="text-xs">Blocking</span>
                          )}
                        </div>
                      </button>
                    );
                  })}
                </div>
              );
            })()}
          </div>
        </div>
      )}
    </>
  );
}
