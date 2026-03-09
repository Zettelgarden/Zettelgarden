import React from "react";

type ViewMode = "list" | "matrix" | "kanban" | "calendar";

interface ViewModeToggleProps {
  value: ViewMode;
  onChange: (mode: ViewMode) => void;
}

const VIEW_MODES: { value: ViewMode; label: string; icon: string; shortcut: string }[] = [
  { value: "list", label: "List", icon: "☰", shortcut: "1" },
  { value: "matrix", label: "Matrix", icon: "⊞", shortcut: "2" },
  { value: "kanban", label: "Kanban", icon: "▦", shortcut: "3" },
  { value: "calendar", label: "Calendar", icon: "📅", shortcut: "4" },
];

export function ViewModeToggle({ value, onChange }: ViewModeToggleProps) {
  return (
    <div className="inline-flex rounded-md border border-slate-300 bg-white overflow-hidden">
      {VIEW_MODES.map((mode) => (
        <button
          key={mode.value}
          type="button"
          onClick={() => onChange(mode.value)}
          className={`
            px-2.5 py-1 text-xs font-medium transition-colors
            ${value === mode.value
              ? "bg-blue-600 text-white"
              : "bg-white text-slate-600 hover:bg-slate-50"
            }
            ${mode.value !== VIEW_MODES[0].value ? "border-l border-slate-300" : ""}
          `}
          title={`${mode.label} (${mode.shortcut})`}
        >
          <span className="hidden sm:inline">{mode.label}</span>
          <span className="sm:hidden">{mode.icon}</span>
        </button>
      ))}
    </div>
  );
}

/**
 * Compact version for smaller screens
 */
export function ViewModeToggleCompact({ value, onChange }: ViewModeToggleProps) {
  return (
    <div className="inline-flex rounded border border-slate-300 bg-white overflow-hidden">
      {VIEW_MODES.map((mode) => (
        <button
          key={mode.value}
          type="button"
          onClick={() => onChange(mode.value)}
          className={`
            px-2 py-1 text-sm transition-colors
            ${value === mode.value
              ? "bg-blue-600 text-white"
              : "bg-white text-slate-600 hover:bg-slate-50"
            }
            ${mode.value !== VIEW_MODES[0].value ? "border-l border-slate-300" : ""}
          `}
          title={`${mode.label} (${mode.shortcut})`}
        >
          {mode.icon}
        </button>
      ))}
    </div>
  );
}
