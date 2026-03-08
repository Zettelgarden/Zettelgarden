import React, { useState, useMemo } from 'react';
import { format, startOfMonth, endOfMonth, startOfWeek, endOfWeek, eachDayOfInterval, addMonths, subMonths, isSameDay, isSameMonth } from 'date-fns';
import { HabitLog } from '../../models/habit';

interface HabitCalendarProps {
  logs: HabitLog[];
  onDayClick?: (date: Date, logs: HabitLog[]) => void;
}

export const HabitCalendar: React.FC<HabitCalendarProps> = ({ logs, onDayClick }) => {
  const [currentMonth, setCurrentMonth] = useState(new Date());

  // Create a set of dates with check-ins for quick lookup
  const checkedDates = useMemo(() => {
    const dateMap = new Map<string, HabitLog[]>();
    logs.forEach(log => {
      const dateKey = format(new Date(log.completed_at), 'yyyy-MM-dd');
      if (!dateMap.has(dateKey)) {
        dateMap.set(dateKey, []);
      }
      dateMap.get(dateKey)!.push(log);
    });
    return dateMap;
  }, [logs]);

  // Generate calendar grid
  const calendarDays = useMemo(() => {
    const monthStart = startOfMonth(currentMonth);
    const monthEnd = endOfMonth(currentMonth);
    const gridStart = startOfWeek(monthStart, { weekStartsOn: 0 });
    const gridEnd = endOfWeek(monthEnd, { weekStartsOn: 0 });
    return eachDayOfInterval({ start: gridStart, end: gridEnd });
  }, [currentMonth]);

  const weekDayNames = ['Sun', 'Mon', 'Tue', 'Wed', 'Thu', 'Fri', 'Sat'];

  const handlePrevMonth = () => setCurrentMonth(subMonths(currentMonth, 1));
  const handleNextMonth = () => setCurrentMonth(addMonths(currentMonth, 1));
  const handleToday = () => setCurrentMonth(new Date());

  const handleDayClick = (day: Date) => {
    const dateKey = format(day, 'yyyy-MM-dd');
    const dayLogs = checkedDates.get(dateKey) || [];
    onDayClick?.(day, dayLogs);
  };

  const today = new Date();

  return (
    <div className="bg-white rounded-lg border border-gray-200 p-3">
      {/* Header */}
      <div className="flex items-center justify-between mb-3">
        <button
          onClick={handlePrevMonth}
          className="p-1.5 hover:bg-gray-100 rounded min-h-[32px] min-w-[32px]"
          aria-label="Previous month"
        >
          ‹
        </button>
        <div className="flex items-center gap-2">
          <h3 className="text-sm font-semibold text-gray-800">
            {format(currentMonth, 'MMMM yyyy')}
          </h3>
          <button
            onClick={handleToday}
            className="text-xs px-2 py-0.5 bg-gray-100 hover:bg-gray-200 rounded text-gray-600"
          >
            Today
          </button>
        </div>
        <button
          onClick={handleNextMonth}
          className="p-1.5 hover:bg-gray-100 rounded min-h-[32px] min-w-[32px]"
          aria-label="Next month"
        >
          ›
        </button>
      </div>

      {/* Weekday headers */}
      <div className="grid grid-cols-7 mb-1">
        {weekDayNames.map(day => (
          <div key={day} className="text-center text-xs font-medium text-gray-500 py-1">
            {day}
          </div>
        ))}
      </div>

      {/* Calendar grid */}
      <div className="grid grid-cols-7 gap-0.5">
        {calendarDays.map((day, index) => {
          const dateKey = format(day, 'yyyy-MM-dd');
          const hasCheckin = checkedDates.has(dateKey);
          const isToday = isSameDay(day, today);
          const isCurrentMonth = isSameMonth(day, currentMonth);

          return (
            <button
              key={index}
              onClick={() => handleDayClick(day)}
              className={`
                aspect-square p-0.5 text-xs rounded-sm transition-colors
                ${!isCurrentMonth ? 'text-gray-300' : 'text-gray-700'}
                ${isToday ? 'ring-1 ring-blue-500' : ''}
                ${hasCheckin ? 'bg-green-500 text-white hover:bg-green-600' : 'hover:bg-gray-100'}
              `}
              title={hasCheckin ? `Checked in on ${format(day, 'MMM d')}` : format(day, 'MMM d')}
            >
              {format(day, 'd')}
            </button>
          );
        })}
      </div>

      {/* Legend */}
      <div className="flex items-center justify-center gap-4 mt-3 pt-2 border-t border-gray-100">
        <div className="flex items-center gap-1">
          <div className="w-3 h-3 bg-green-500 rounded-sm" />
          <span className="text-xs text-gray-500">Checked in</span>
        </div>
        <div className="flex items-center gap-1">
          <div className="w-3 h-3 ring-1 ring-blue-500 rounded-sm" />
          <span className="text-xs text-gray-500">Today</span>
        </div>
      </div>
    </div>
  );
};
