import React from "react";
import { Task, TaskStatus } from "../../models/Task";
import { TaskListItem } from "./TaskListItem";

interface KanbanBoardProps {
  tasks: Task[];
  onTagClick: (tag: string) => void;
}

interface KanbanColumn {
  status: TaskStatus;
  title: string;
  color: string;
  icon: string;
}

const columns: KanbanColumn[] = [
  { status: "todo", title: "To Do", color: "#6B7280", icon: "⭕" },
  { status: "in_progress", title: "In Progress", color: "#3B82F6", icon: "🔄" },
  { status: "blocked", title: "Blocked", color: "#EF4444", icon: "🚫" },
  { status: "done", title: "Done", color: "#10B981", icon: "✅" },
];

export function KanbanBoard({ tasks, onTagClick }: KanbanBoardProps) {
  // Group tasks by status
  const tasksByStatus = tasks.reduce((acc, task) => {
    const status = task.status || "todo";
    if (!acc[status]) {
      acc[status] = [];
    }
    acc[status].push(task);
    return acc;
  }, {} as Record<TaskStatus, Task[]>);

  return (
    <div className="grid grid-cols-1 md:grid-cols-2 lg:grid-cols-4 gap-4">
      {columns.map((column) => {
        const columnTasks = tasksByStatus[column.status] || [];

        return (
          <div
            key={column.status}
            className="flex flex-col bg-gray-50 rounded-lg border border-gray-200"
          >
            {/* Column Header */}
            <div
              className="p-3 rounded-t-lg border-b-2"
              style={{
                backgroundColor: column.color + "15",
                borderBottomColor: column.color,
              }}
            >
              <div className="flex items-center justify-between">
                <div className="flex items-center gap-2">
                  <span className="text-lg">{column.icon}</span>
                  <h3
                    className="font-semibold text-sm"
                    style={{ color: column.color }}
                  >
                    {column.title}
                  </h3>
                </div>
                <span
                  className="text-xs font-medium px-2 py-0.5 rounded-full"
                  style={{
                    backgroundColor: column.color + "20",
                    color: column.color,
                  }}
                >
                  {columnTasks.length}
                </span>
              </div>
            </div>

            {/* Column Tasks */}
            <div className="flex-1 p-2 space-y-2 overflow-y-auto max-h-[600px]">
              {columnTasks.length > 0 ? (
                columnTasks.map((task) => (
                  <div
                    key={task.id}
                    className="bg-white rounded border border-gray-200 shadow-sm hover:shadow-md transition-shadow"
                  >
                    <TaskListItem
                      task={task}
                      onTagClick={onTagClick}
                      hideMatrixTags={false}
                    />
                  </div>
                ))
              ) : (
                <div className="text-center text-gray-400 text-sm py-8">
                  No tasks
                </div>
              )}
            </div>
          </div>
        );
      })}
    </div>
  );
}
