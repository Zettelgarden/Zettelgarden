import React from "react";

interface TaskDropdownProps {
  isOpen: boolean;
  onToggle: (e: React.MouseEvent) => void;
  onClose: () => void;
  display: {
    icon?: string;
    text: string;
    color: string;
  };
  menuClassName?: string;
  children: React.ReactNode;
}

/**
 * Consistent dropdown menu wrapper for Task*Display components.
 * Provides:
 * - Clickable display badge with icon, text, and color styling
 * - Dropdown menu container with consistent styling
 * - Click-outside and stopPropagation handling
 */
export function TaskDropdown({
  isOpen,
  onToggle,
  onClose,
  display,
  menuClassName = "min-w-[140px]",
  children,
}: TaskDropdownProps) {
  return (
    <div className="relative inline-block">
      <span
        onClick={onToggle}
        className="cursor-pointer inline-flex items-center gap-1 px-2 py-0.5 rounded-md text-xs font-medium transition-colors hover:opacity-80"
        style={{
          backgroundColor: display.color + "20",
          color: display.color,
          border: `1px solid ${display.color}40`,
        }}
      >
        {display.icon && <span>{display.icon}</span>}
        <span>{display.text}</span>
      </span>

      {isOpen && (
        <div
          className={`absolute z-20 mt-1 bg-white rounded-md shadow-lg py-1 border border-gray-200 ${menuClassName}`}
          onClick={(e) => {
            e.stopPropagation();
            onClose();
          }}
        >
          {children}
        </div>
      )}
    </div>
  );
}
