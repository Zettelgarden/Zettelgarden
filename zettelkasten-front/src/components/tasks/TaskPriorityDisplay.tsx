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
            return { text: "No Priority", color: "gray" };
        }

        switch (task.priority) {
            case "A":
                return { text: "Priority A", color: "red" };
            case "B":
                return { text: "Priority B", color: "orange" };
            case "C":
                return { text: "Priority C", color: "blue" };
            default:
                return { text: task.priority, color: "gray" };
        }
    };

    const priorityDisplay = getPriorityDisplay();

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
        <div className="relative">
            <span
                onClick={() => setShowPriorityMenu(!showPriorityMenu)}
                className="task-priority cursor-pointer"
                style={{ color: priorityDisplay.color }}
            >
                {priorityDisplay.text}
            </span>

            {showPriorityMenu && (
                <div className="absolute z-10 mt-1 bg-white rounded-md shadow-lg py-1 w-32">
                    <div
                        className="px-4 py-2 hover:bg-gray-100 cursor-pointer"
                        onClick={() => setPriority("A")}
                    >
                        <span style={{ color: "red" }}>Priority A</span>
                    </div>
                    <div
                        className="px-4 py-2 hover:bg-gray-100 cursor-pointer"
                        onClick={() => setPriority("B")}
                    >
                        <span style={{ color: "orange" }}>Priority B</span>
                    </div>
                    <div
                        className="px-4 py-2 hover:bg-gray-100 cursor-pointer"
                        onClick={() => setPriority("C")}
                    >
                        <span style={{ color: "blue" }}>Priority C</span>
                    </div>
                    <div
                        className="px-4 py-2 hover:bg-gray-100 cursor-pointer"
                        onClick={() => setPriority(null)}
                    >
                        <span style={{ color: "gray" }}>No Priority</span>
                    </div>
                </div>
            )}
        </div>
    );
}
