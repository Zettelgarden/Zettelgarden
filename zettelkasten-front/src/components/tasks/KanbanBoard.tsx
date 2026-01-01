import React from "react";
import { DragDropContext, Droppable, Draggable, DropResult } from "@hello-pangea/dnd";
import { Task, TaskStatus } from "../../models/Task";
import { TaskListItem } from "./TaskListItem";
import { saveExistingTask } from "../../api/tasks";
import { useTaskContext } from "../../contexts/TaskContext";

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
  const { setRefreshTasks } = useTaskContext();

  // Group tasks by status
  const tasksByStatus = tasks.reduce((acc, task) => {
    const status = task.status || "todo";
    if (!acc[status]) {
      acc[status] = [];
    }
    acc[status].push(task);
    return acc;
  }, {} as Record<TaskStatus, Task[]>);

  const onDragEnd = async (result: DropResult) => {
    if (!result.destination) return;

    const sourceStatus = result.source.droppableId as TaskStatus;
    const destStatus = result.destination.droppableId as TaskStatus;

    // No change if dropped in the same column
    if (sourceStatus === destStatus) return;

    const draggedId = result.draggableId;
    const task = tasks.find(t => t.id.toString() === draggedId);
    if (!task) return;

    // Update task status
    const updatedTask = { ...task, status: destStatus };

    // Sync is_complete with status
    if (destStatus === "done") {
      updatedTask.is_complete = true;
    } else {
      updatedTask.is_complete = false;
    }

    try {
      // Persist changes
      const response = await saveExistingTask(updatedTask);
      if (!("error" in response)) {
        setRefreshTasks(true);
      }
    } catch (err) {
      console.error("Failed to save updated task after drag-and-drop:", err);
    }
  };

  return (
    <DragDropContext onDragEnd={onDragEnd}>
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

              {/* Column Tasks - Droppable Zone */}
              <Droppable droppableId={column.status}>
                {(dropProvided, snapshot) => (
                  <div
                    ref={dropProvided.innerRef}
                    {...dropProvided.droppableProps}
                    className={`flex-1 p-2 space-y-2 overflow-y-auto max-h-[600px] transition-colors ${
                      snapshot.isDraggingOver
                        ? "bg-blue-50"
                        : ""
                    }`}
                  >
                    {columnTasks.length > 0 ? (
                      columnTasks.map((task, index) => (
                        <Draggable
                          key={task.id.toString()}
                          draggableId={task.id.toString()}
                          index={index}
                        >
                          {(dragProvided, dragSnapshot) => (
                            <div
                              ref={dragProvided.innerRef}
                              {...dragProvided.draggableProps}
                              {...dragProvided.dragHandleProps}
                              className={`bg-white rounded border shadow-sm transition-shadow ${
                                dragSnapshot.isDragging
                                  ? "border-blue-400 shadow-lg"
                                  : "border-gray-200 hover:shadow-md"
                              }`}
                            >
                              <TaskListItem
                                task={task}
                                onTagClick={onTagClick}
                                hideMatrixTags={false}
                              />
                            </div>
                          )}
                        </Draggable>
                      ))
                    ) : (
                      <div className="text-center text-gray-400 text-sm py-8">
                        No tasks
                      </div>
                    )}
                    {dropProvided.placeholder}
                  </div>
                )}
              </Droppable>
            </div>
          );
        })}
      </div>
    </DragDropContext>
  );
}
