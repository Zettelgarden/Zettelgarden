import React, { useState, useRef, useEffect } from "react";
import { createPortal } from "react-dom";
import { Task } from "../../models/Task";
import { format } from "date-fns-tz";
import { useAuth } from "../../contexts/AuthContext";

interface TaskHoverCardProps {
  task: Task;
  children: React.ReactNode;
}

const PRIORITY_CONFIG = {
  A: { color: "#EF4444", icon: "🔴", label: "High" },
  B: { color: "#F59E0B", icon: "🟠", label: "Medium" },
  C: { color: "#3B82F6", icon: "🔵", label: "Low" },
} as const;

export function TaskHoverCard({ task, children }: TaskHoverCardProps) {
  const [isVisible, setIsVisible] = useState(false);
  const [position, setPosition] = useState({ top: 0, left: 0 });
  const triggerRef = useRef<HTMLDivElement>(null);
  const hoverCardRef = useRef<HTMLDivElement>(null);
  const { user } = useAuth();
  const userTimezone = user?.timezone || "UTC";

  // Calculate position when hover card becomes visible
  useEffect(() => {
    if (!isVisible || !triggerRef.current) return;

    const triggerRect = triggerRef.current.getBoundingClientRect();
    const hoverCardWidth = 280;
    const hoverCardHeight = 200;
    const padding = 8;

    let left = triggerRect.right + padding;
    let top = triggerRect.top;

    // Check if hover card would go off the right edge
    if (left + hoverCardWidth > window.innerWidth - padding) {
      left = triggerRect.left - hoverCardWidth - padding;
    }

    // If still off-screen, position below
    if (left < padding) {
      left = Math.min(padding, triggerRect.left);
      top = triggerRect.bottom + padding;
    }

    // Check if hover card would go off the bottom edge
    if (top + hoverCardHeight > window.innerHeight - padding) {
      top = Math.max(padding, window.innerHeight - hoverCardHeight - padding);
    }

    setPosition({ top, left });
  }, [isVisible]);

  // Handle mouse events
  const handleMouseEnter = () => {
    setIsVisible(true);
  };

  const handleMouseLeave = (e: React.MouseEvent) => {
    // Check if we're moving to the hover card
    if (hoverCardRef.current && hoverCardRef.current.contains(e.relatedTarget as Node)) {
      return;
    }
    setIsVisible(false);
  };

  // Get description preview (first 150 chars)
  const descriptionPreview = task.description
    ? task.description.length > 150
      ? task.description.slice(0, 150) + "..."
      : task.description
    : null;

  // Format dates
  const formatDate = (date: Date | string | null) => {
    if (!date) return null;
    try {
      return format(new Date(date), "MMM d, yyyy", { timeZone: userTimezone });
    } catch {
      return null;
    }
  };

  const scheduledDate = formatDate(task.scheduled_date);
  const dueDate = formatDate(task.due_date);

  // Get priority display
  const priorityDisplay = task.priority
    ? PRIORITY_CONFIG[task.priority as keyof typeof PRIORITY_CONFIG] || { icon: "○", label: task.priority, color: "#6B7280" }
    : null;

  // Don't show hover card if there's nothing to preview
  const hasPreview = descriptionPreview || scheduledDate || dueDate || priorityDisplay || task.tags.length > 0;

  if (!hasPreview) {
    return <>{children}</>;
  }

  return (
    <>
      <div
        ref={triggerRef}
        onMouseEnter={handleMouseEnter}
        onMouseLeave={handleMouseLeave}
        className="relative"
      >
        {children}
      </div>

      {isVisible && createPortal(
        <div
          ref={hoverCardRef}
          className="fixed z-50 bg-white rounded-lg shadow-lg border border-gray-200 p-3 w-70"
          style={{
            top: position.top,
            left: position.left,
          }}
          onMouseEnter={() => setIsVisible(true)}
          onMouseLeave={() => setIsVisible(false)}
        >
          {/* Task Title */}
          <h4 className="font-medium text-sm text-gray-900 mb-2 line-clamp-2">
            {task.title}
          </h4>

          {/* Description Preview */}
          {descriptionPreview && (
            <p className="text-xs text-gray-600 mb-2 leading-relaxed">
              {descriptionPreview}
            </p>
          )}

          {/* Metadata */}
          <div className="flex flex-wrap gap-2 text-xs">
            {/* Scheduled Date */}
            {scheduledDate && (
              <span className="inline-flex items-center gap-1 px-2 py-0.5 bg-blue-50 text-blue-700 rounded">
                📅 {scheduledDate}
              </span>
            )}

            {/* Due Date */}
            {dueDate && (
              <span className="inline-flex items-center gap-1 px-2 py-0.5 bg-orange-50 text-orange-700 rounded">
                ⏰ {dueDate}
              </span>
            )}

            {/* Priority */}
            {priorityDisplay && (
              <span
                className="inline-flex items-center gap-1 px-2 py-0.5 rounded"
                style={{ backgroundColor: priorityDisplay.color + "15", color: priorityDisplay.color }}
              >
                {priorityDisplay.icon} {priorityDisplay.label}
              </span>
            )}
          </div>

          {/* Tags */}
          {task.tags.length > 0 && (
            <div className="flex flex-wrap gap-1 mt-2">
              {task.tags.slice(0, 5).map((tag) => (
                <span
                  key={tag.name}
                  className="inline-block px-1.5 py-0.5 text-xs bg-gray-100 text-gray-600 rounded"
                >
                  #{tag.name}
                </span>
              ))}
              {task.tags.length > 5 && (
                <span className="text-xs text-gray-400">
                  +{task.tags.length - 5} more
                </span>
              )}
            </div>
          )}

          {/* Hint */}
          <div className="mt-2 pt-2 border-t border-gray-100 text-xs text-gray-400">
            Click to open task
          </div>
        </div>,
        document.body
      )}
    </>
  );
}
