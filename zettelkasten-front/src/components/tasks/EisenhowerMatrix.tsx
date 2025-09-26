import React from "react";
import { DragDropContext, Droppable, Draggable, DropResult } from "@hello-pangea/dnd";
import { Task } from "../../models/Task";
import { TaskList } from "./TaskList";

interface EisenhowerMatrixProps {
    tasks: Task[];
    onTagClick: (tag: string) => void;
    onAddTaskWithTags?: (tags: string[]) => void;
}

function normalizeTag(tag: string) {
    return tag.replace(/^#/, "").toLowerCase();
}

function getQuadrant(task: Task): 1 | 2 | 3 | 4 {
    const tags = (task.tags || []).map((t: any) =>
        typeof t === "string" ? normalizeTag(t) : (t && t.name ? normalizeTag(String(t.name)) : "")
    );
    const isImportant = tags.includes("important");
    const isUrgent = tags.includes("urgent");

    if (isImportant && isUrgent) return 1;
    if (isImportant && !isUrgent) return 2;
    if (!isImportant && isUrgent) return 3;
    return 4; // default quadrant if no labels or both false
}

import { saveExistingTask } from "../../api/tasks";
import { useTaskContext } from "../../contexts/TaskContext";

export function EisenhowerMatrix({ tasks, onTagClick, onAddTaskWithTags }: EisenhowerMatrixProps) {
    const { setRefreshTasks } = useTaskContext();
    const q1 = tasks.filter(t => getQuadrant(t) === 1);
    const q2 = tasks.filter(t => getQuadrant(t) === 2);
    const q3 = tasks.filter(t => getQuadrant(t) === 3);
    const q4 = tasks.filter(t => getQuadrant(t) === 4);

    const quadrantTitles: Record<number, string> = {
        1: "Do First (Important and Urgent)",
        2: "Schedule (Important and Not Urgent)",
        3: "Delegate (Not Important and Urgent)",
        4: "Eliminate (Not Important and Not Urgent)",
    };

    const quadrantData: Record<number, Task[]> = { 1: q1, 2: q2, 3: q3, 4: q4 };

    const quadrantTags: Record<number, string[]> = {
        1: ["#urgent", "#important"],
        2: ["#important"],
        3: ["#urgent"],
        4: [],
    };

    function updateTaskTagsForQuadrant(task: Task, quadrant: number): Task {
        // Remove #urgent / #important from title
        let newTitle = task.title
            .replace(/(^|\s)#urgent(\s|$)/gi, " ")
            .replace(/(^|\s)#important(\s|$)/gi, " ")
            .trim();

        // Add tags for this quadrant directly to the title string
        if (quadrant === 1) newTitle += " #urgent #important";
        if (quadrant === 2) newTitle += " #important";
        if (quadrant === 3) newTitle += " #urgent";
        // quadrant 4 gets none

        // Normalize spacing
        newTitle = newTitle.replace(/\s+/g, " ").trim();

        return { ...task, title: newTitle };
    }

    const onDragEnd = async (result: DropResult) => {
        console.log("?")
        if (!result.destination) return;
        console.log("done")
        const sourceQ = parseInt(result.source.droppableId);
        const destQ = parseInt(result.destination.droppableId);
        if (sourceQ === destQ) return;
        const draggedId = result.draggableId;
        const task = tasks.find(t => t.id.toString() === draggedId);
        if (!task) return;
        const updatedTask = updateTaskTagsForQuadrant(task, destQ);

        try {
            // Persist changes
            const response = await saveExistingTask(updatedTask);
            if (!("error" in response)) {
                task.tags = updatedTask.tags;
                setRefreshTasks(true);
            }
        } catch (err) {
            console.error("Failed to save updated task after drag-and-drop:", err);
        }
    };

    const quadrantBox = (quadrant: number) => (
        <Droppable droppableId={quadrant.toString()}>
            {(dropProvided, snapshot) => (
                <div
                    ref={dropProvided.innerRef}
                    {...dropProvided.droppableProps}
                    className={`border-2 p-2 rounded min-h-[150px] transition-colors duration-200 ${snapshot.isDraggingOver
                        ? "border-blue-500 bg-blue-50"
                        : "border-slate-400 bg-white"
                        }`}
                >
                    <div className="flex flex-col sm:flex-row justify-between items-start sm:items-center gap-2 sm:gap-0 mb-2">
                        <h3 className="font-bold">{quadrantTitles[quadrant]} ({quadrantData[quadrant].length})</h3>
                        {typeof onAddTaskWithTags === "function" && (
                            <button
                                onClick={() => onAddTaskWithTags(quadrantTags[quadrant])}
                                className="text-xs bg-blue-500 hover:bg-blue-600 text-white px-2 py-1 rounded"
                            >
                                + Add Task
                            </button>
                        )}
                    </div>
                    {quadrantData[quadrant].length > 0 ? (
                        quadrantData[quadrant].map((task, idx) => (
                            <Draggable key={task.id.toString()} draggableId={task.id.toString()} index={idx}>
                                {(dragProvided) => (
                                    <div
                                        ref={dragProvided.innerRef}
                                        {...dragProvided.draggableProps}
                                        {...dragProvided.dragHandleProps}
                                        className={`mb-2 border-2 rounded transition-shadow duration-150 ${snapshot.isDraggingOver
                                            ? "shadow-lg border-blue-400"
                                            : "border-transparent"
                                            }`}
                                    >
                                        <TaskList onTagClick={onTagClick} tasks={[task]} hideMatrixTags={true} />
                                    </div>
                                )}
                            </Draggable>
                        ))
                    ) : (
                        <div className="text-slate-500 text-sm">No tasks</div>
                    )}
                    {dropProvided.placeholder}
                </div>
            )}
        </Droppable>
    );

    return (
        <DragDropContext onDragEnd={onDragEnd}>

            <div className="grid grid-cols-1 sm:grid-cols-2 gap-4">
                {quadrantBox(1)}
                {quadrantBox(2)}
                {quadrantBox(3)}
                {quadrantBox(4)}
            </div>
        </DragDropContext>
    );
}
