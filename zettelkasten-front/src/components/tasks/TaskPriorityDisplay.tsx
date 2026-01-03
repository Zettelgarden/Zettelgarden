import React, { useState, useEffect } from "react";
import { Task } from "../../models/Task";
import { saveExistingTask } from "../../api/tasks";
import { useTaskContext } from "../../contexts/TaskContext";

interface TaskPriorityDisplayProps {
    task: Task;
    setTask: (task: Task) => void;
    saveOnChange: boolean;
}

export function TaskPriorityDisplay({
    task,
    setTask,
    saveOnChange,
}: TaskPriorityDisplayProps) {
    const { setRefreshTasks, updateTask: updateTaskInContext } = useTaskContext();
    const [showPriorityMenu, setShowPriorityMenu] = useState<boolean>(false);

    // Get display text and color based on priority
    const getPriorityDisplay = () => {
        if (!task.priority) {
            return { text: "No Priority", color: "#6B7280", icon: "○" };
        }

        switch (task.priority) {
            case "A":
                return { text: "Priority A", color: "#EF4444", icon: "🔴" };
            case "B":
                return { text: "Priority B", color: "#F59E0B", icon: "🟠" };
            case "C":
                return { text: "Priority C", color: "#3B82F6", icon: "🔵" };
            default:
                return { text: task.priority, color: "#6B7280", icon: "○" };
        }
    };

    const priorityDisplay = getPriorityDisplay();

    // Close menu when clicking outside
    useEffect(() => {
        const handleClickOutside = () => setShowPriorityMenu(false);
        if (showPriorityMenu) {
            document.addEventListener("click", handleClickOutside);
            return () => document.removeEventListener("click", handleClickOutside);
        }
    }, [showPriorityMenu]);

    async function updateTask(editedTask: Task) {
        if (saveOnChange) {
            // Optimistic update: update UI immediately
            updateTaskInContext(editedTask);

            // Send update to server in background
            try {
                const response = await saveExistingTask(editedTask);
                if ("error" in response) {
                    // Rollback on error
                    updateTaskInContext(task);
                    console.error("Failed to update task priority:", response.error);
                }
            } catch (error) {
                // Rollback on network error
                updateTaskInContext(task);
                console.error("Failed to update task priority:", error);
            }
        } else {
            setTask(editedTask);
        }
    }

    async function setPriority(priority: string | null) {
        let editedTask = { ...task, priority };
        updateTask(editedTask);
        setShowPriorityMenu(false);
    }

    return (
        <div className="relative inline-block">
            <span
                onClick={(e) => {
                    e.stopPropagation();
                    setShowPriorityMenu(!showPriorityMenu);
                }}
                className="cursor-pointer inline-flex items-center gap-1 px-2 py-0.5 rounded-md text-xs font-medium transition-colors hover:opacity-80"
                style={{
                    backgroundColor: priorityDisplay.color + "20",
                    color: priorityDisplay.color,
                    border: `1px solid ${priorityDisplay.color}40`,
                }}
            >
                <span>{priorityDisplay.icon}</span>
                <span>{priorityDisplay.text}</span>
            </span>

            {showPriorityMenu && (
                <div
                    className="absolute z-20 mt-1 bg-white rounded-md shadow-lg py-1 min-w-[140px] border border-gray-200"
                    onClick={(e) => e.stopPropagation()}
                >
                    <div
                        className="px-3 py-2 hover:bg-gray-100 cursor-pointer flex items-center gap-2 text-sm"
                        onClick={() => setPriority("A")}
                        style={{ color: "#EF4444" }}
                    >
                        <span>🔴</span>
                        <span>Priority A</span>
                    </div>
                    <div
                        className="px-3 py-2 hover:bg-gray-100 cursor-pointer flex items-center gap-2 text-sm"
                        onClick={() => setPriority("B")}
                        style={{ color: "#F59E0B" }}
                    >
                        <span>🟠</span>
                        <span>Priority B</span>
                    </div>
                    <div
                        className="px-3 py-2 hover:bg-gray-100 cursor-pointer flex items-center gap-2 text-sm"
                        onClick={() => setPriority("C")}
                        style={{ color: "#3B82F6" }}
                    >
                        <span>🔵</span>
                        <span>Priority C</span>
                    </div>
                    <div
                        className="px-3 py-2 hover:bg-gray-100 cursor-pointer flex items-center gap-2 text-sm"
                        onClick={() => setPriority(null)}
                        style={{ color: "#6B7280" }}
                    >
                        <span>○</span>
                        <span>No Priority</span>
                    </div>
                </div>
            )}
        </div>
    );
}
