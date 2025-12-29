import React, { useEffect, useState } from "react";
import { Task } from "../../models/Task";

import {
    getToday,
    getTomorrow,
    getNextWeek,
    getNextMonday,
    isFriday,
} from "../../utils/dates";
import { saveExistingTask } from "../../api/tasks";
import { useTaskContext } from "../../contexts/TaskContext";

interface BulkTaskDateDisplayProps {
    tasks: Task[];
    setShowBulkEdit: (show: boolean) => void;
}

export function BulkTaskDateDisplay({
    tasks,
    setShowBulkEdit,
}: BulkTaskDateDisplayProps) {
    const { setRefreshTasks } = useTaskContext();
    const [selectedDate, setSelectedDate] = useState<string>("");

    async function updateTasks(newDate: Date | null) {
        const promises = tasks.map((task) => {
            const updatedTask = { ...task, scheduled_date: newDate };
            return saveExistingTask(updatedTask);
        });

        try {
            await Promise.all(promises);
            setRefreshTasks(true);
            setShowBulkEdit(false);
        } catch (error) {
            console.error("Failed to bulk update tasks", error);
            alert("Failed to bulk update tasks. Please try again.");
        }
    }

    async function handleScheduledDateChange(
        e: React.ChangeEvent<HTMLInputElement>,
    ) {
        const newDate = new Date(e.target.value);
        const localTimezoneOffset = newDate.getTimezoneOffset();
        const utcDate = new Date(newDate.getTime() + localTimezoneOffset * 60000);
        updateTasks(utcDate);
    }

    async function setNoDate() {
        updateTasks(null);
    }
    async function setToday() {
        updateTasks(getToday());
    }
    async function setTomorrow() {
        updateTasks(getTomorrow());
    }
    async function setNextWeek() {
        updateTasks(getNextWeek());
    }

    async function setNextMonday() {
        updateTasks(getNextMonday());
    }

    return (
        <div className="dropdown relative">
            <div className="absolute right-0 mt-1 w-48 bg-white border border-slate-300 rounded shadow-lg p-3 z-20">
                <h3 className="text-sm font-semibold mb-3 text-gray-700">
                    Edit Date ({tasks.length} tasks)
                </h3>
                <div className="flex flex-col space-y-2">
                    {" "}
                    {/* This creates vertical spacing between children */}
                    <button onClick={setNoDate} className="w-full">
                        No Date
                    </button>
                    <button onClick={setToday} className="w-full">
                        Today
                    </button>
                    <button onClick={setTomorrow} className="w-full">
                        Tomorrow
                    </button>
                    {isFriday() ? (
                        <button onClick={setNextMonday} className="w-full">
                            Next Monday
                        </button>
                    ) : (
                        <div></div>
                    )}
                    <button onClick={setNextWeek} className="w-full">
                        Next Week
                    </button>
                    <input
                        aria-label="Date"
                        type="date"
                        className="p-2 w-full"
                        value={selectedDate}
                        onChange={handleScheduledDateChange}
                    />
                </div>
            </div>
        </div>
    );
}
