import React from "react";

interface KeyboardShortcutsHelpProps {
  visible: boolean;
  onClose: () => void;
}

interface ShortcutGroup {
  title: string;
  shortcuts: { key: string; description: string; modifier?: string }[];
}

const SHORTCUT_GROUPS: ShortcutGroup[] = [
  {
    title: "General",
    shortcuts: [
      { key: "?", description: "Show keyboard shortcuts" },
      { key: "Esc", description: "Close dialogs and menus" },
    ],
  },
  {
    title: "Tasks",
    shortcuts: [
      { key: "N", description: "Create new task" },
      { key: "F", description: "Focus filter input" },
    ],
  },
  {
    title: "Views",
    shortcuts: [
      { key: "1", description: "Switch to List view" },
      { key: "2", description: "Switch to Matrix view" },
      { key: "3", description: "Switch to Kanban view" },
    ],
  },
  {
    title: "Kanban Board",
    shortcuts: [
      { key: "↑", description: "Move to previous card" },
      { key: "↓", description: "Move to next card" },
      { key: "←", description: "Move to previous column" },
      { key: "→", description: "Move to next column" },
      { key: "j", description: "Move to next card (vim)" },
      { key: "k", description: "Move to previous card (vim)" },
      { key: "h", description: "Move to previous column (vim)" },
      { key: "l", description: "Move to next column (vim)" },
      { key: "Enter", description: "Open focused task" },
      { key: "Esc", description: "Clear selection" },
    ],
  },
];

export function KeyboardShortcutsHelp({ visible, onClose }: KeyboardShortcutsHelpProps) {
  if (!visible) return null;

  return (
    <>
      {/* Backdrop */}
      <div
        className="fixed inset-0 bg-black/50 z-50"
        onClick={onClose}
      />

      {/* Modal */}
      <div className="fixed inset-0 z-50 flex items-center justify-center p-4">
        <div
          className="bg-white rounded-lg shadow-2xl max-w-md w-full p-6"
          onClick={(e) => e.stopPropagation()}
        >
          <div className="flex items-center justify-between mb-4">
            <h2 className="text-lg font-semibold text-slate-800">
              Keyboard Shortcuts
            </h2>
            <button
              onClick={onClose}
              className="text-slate-400 hover:text-slate-600 transition-colors"
            >
              <svg
                xmlns="http://www.w3.org/2000/svg"
                viewBox="0 0 20 20"
                fill="currentColor"
                className="w-5 h-5"
              >
                <path d="M6.28 5.22a.75.75 0 00-1.06 1.06L8.94 10l-3.72 3.72a.75.75 0 101.06 1.06L10 11.06l3.72 3.72a.75.75 0 101.06-1.06L11.06 10l3.72-3.72a.75.75 0 00-1.06-1.06L10 8.94 6.28 5.22z" />
              </svg>
            </button>
          </div>

          <div className="space-y-4">
            {SHORTCUT_GROUPS.map((group) => (
              <div key={group.title}>
                <h3 className="text-xs font-semibold text-slate-500 uppercase tracking-wider mb-2">
                  {group.title}
                </h3>
                <ul className="space-y-1">
                  {group.shortcuts.map((shortcut) => (
                    <li
                      key={shortcut.key}
                      className="flex items-center justify-between py-1"
                    >
                      <span className="text-sm text-slate-600">
                        {shortcut.description}
                      </span>
                      <kbd className="px-2 py-0.5 text-xs font-mono bg-slate-100 border border-slate-200 rounded text-slate-700">
                        {shortcut.modifier ? `${shortcut.modifier}+` : ""}
                        {shortcut.key}
                      </kbd>
                    </li>
                  ))}
                </ul>
              </div>
            ))}
          </div>

          <div className="mt-6 pt-4 border-t border-slate-100">
            <p className="text-xs text-slate-400 text-center">
              Press <kbd className="px-1 py-0.5 text-xs font-mono bg-slate-100 border border-slate-200 rounded">?</kbd> or <kbd className="px-1 py-0.5 text-xs font-mono bg-slate-100 border border-slate-200 rounded">Esc</kbd> to close
            </p>
          </div>
        </div>
      </div>
    </>
  );
}

/**
 * Hook to manage keyboard shortcuts
 */
export function useKeyboardShortcuts(handlers: {
  onShowHelp?: () => void;
  onCloseHelp?: () => void;
  onNewTask?: () => void;
  onFocusFilter?: () => void;
  onSwitchView?: (view: "list" | "matrix" | "kanban") => void;
  onEscape?: () => void;
}) {
  const [showHelp, setShowHelp] = React.useState(false);

  React.useEffect(() => {
    const handleKeyDown = (event: KeyboardEvent) => {
      // Ignore if typing in an input field (unless it's Escape)
      const target = event.target as HTMLElement;
      const isInputField =
        target.tagName === "INPUT" ||
        target.tagName === "TEXTAREA" ||
        target.isContentEditable;

      // Escape always works
      if (event.key === "Escape") {
        event.preventDefault();
        if (showHelp) {
          setShowHelp(false);
          handlers.onCloseHelp?.();
        } else {
          handlers.onEscape?.();
        }
        return;
      }

      // Don't trigger shortcuts when typing in input fields
      if (isInputField) {
        return;
      }

      // Ignore system shortcuts
      if (event.metaKey || event.ctrlKey || event.altKey) {
        return;
      }

      switch (event.key) {
        case "?":
          event.preventDefault();
          setShowHelp((prev) => !prev);
          if (!showHelp) {
            handlers.onShowHelp?.();
          }
          break;
        case "n":
        case "N":
          event.preventDefault();
          handlers.onNewTask?.();
          break;
        case "f":
        case "F":
          event.preventDefault();
          handlers.onFocusFilter?.();
          break;
        case "1":
          event.preventDefault();
          handlers.onSwitchView?.("list");
          break;
        case "2":
          event.preventDefault();
          handlers.onSwitchView?.("matrix");
          break;
        case "3":
          event.preventDefault();
          handlers.onSwitchView?.("kanban");
          break;
        case "4":
          event.preventDefault();
          break;
      }
    };

    document.addEventListener("keydown", handleKeyDown);
    return () => document.removeEventListener("keydown", handleKeyDown);
  }, [showHelp, handlers]);

  return { showHelp, setShowHelp };
}
