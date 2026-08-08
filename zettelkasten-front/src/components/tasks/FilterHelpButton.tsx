import React from 'react';

interface FilterHelpButtonProps {
  showHelp: boolean;
  onToggle: () => void;
}

export function FilterHelpButton({
  showHelp,
  onToggle,
}: FilterHelpButtonProps) {
  return (
    <button
      type="button"
      onClick={onToggle}
      className="absolute right-2 top-1/2 -translate-y-1/2 text-slate-400 hover:text-slate-600 cursor-pointer transition-colors rounded px-1"
      title="Filter tips"
      aria-label="Show filter tips"
      aria-expanded={showHelp}
    >
      <svg
        xmlns="http://www.w3.org/2000/svg"
        viewBox="0 0 20 20"
        fill="currentColor"
        className="w-4 h-4"
      >
        <path
          fillRule="evenodd"
          d="M18 10a8 8 0 11-16 0 8 8 0 0116 0zM8.94 6.94a.75.75 0 11-1.061-1.061 3 3 0 112.871 5.026v.345a.75.75 0 01-1.5 0v-.5c0-.72.57-1.172 1.081-1.287A1.5 1.5 0 108.94 6.94zM10 15a1 1 0 100-2 1 1 0 000 2z"
          clipRule="evenodd"
        />
      </svg>
    </button>
  );
}

interface FilterHelpPopoverProps {
  visible: boolean;
  onClose: () => void;
}

export function FilterHelpPopover({
  visible,
  onClose,
}: FilterHelpPopoverProps) {
  if (!visible) return null;

  return (
    <>
      {/* Backdrop to close on click outside */}
      <div className="fixed inset-0 z-20" onClick={onClose} />
      <div className="absolute top-full mt-2 left-0 bg-white p-4 border border-slate-200 rounded-lg shadow-xl z-30 w-80">
        <div className="flex items-center justify-between mb-3">
          <h4 className="font-semibold text-slate-800">Filter Tips</h4>
          <button
            onClick={onClose}
            className="text-slate-400 hover:text-slate-600"
          >
            <svg
              xmlns="http://www.w3.org/2000/svg"
              viewBox="0 0 20 20"
              fill="currentColor"
              className="w-4 h-4"
            >
              <path d="M6.28 5.22a.75.75 0 00-1.06 1.06L8.94 10l-3.72 3.72a.75.75 0 101.06 1.06L10 11.06l3.72 3.72a.75.75 0 101.06-1.06L11.06 10l3.72-3.72a.75.75 0 00-1.06-1.06L10 8.94 6.28 5.22z" />
            </svg>
          </button>
        </div>
        <ul className="space-y-2 text-sm text-slate-600">
          <li className="flex items-start gap-2">
            <span className="text-slate-400 font-mono text-xs mt-0.5">
              text
            </span>
            <span>
              Search by title (e.g.,{' '}
              <code className="bg-slate-100 px-1 rounded text-xs">meeting</code>
              )
            </span>
          </li>
          <li className="flex items-start gap-2">
            <span className="text-blue-500 font-mono text-xs mt-0.5">#tag</span>
            <span>Filter by tag</span>
          </li>
          <li className="flex items-start gap-2">
            <span className="text-purple-500 font-mono text-xs mt-0.5">
              priority:X
            </span>
            <span>A, B, C, or D</span>
          </li>
          <li className="flex items-start gap-2">
            <span className="text-green-600 font-mono text-xs mt-0.5">
              status:X
            </span>
            <span>todo, in_progress, blocked, done</span>
          </li>
          <li className="flex items-start gap-2">
            <span className="text-orange-500 font-mono text-xs mt-0.5">
              date:X
            </span>
            <span>today, tomorrow, or YYYY-MM-DD</span>
          </li>
          <li className="flex items-start gap-2">
            <span className="text-slate-500 font-mono text-xs mt-0.5">
              show:completed
            </span>
            <span>Include completed tasks</span>
          </li>
          <li className="flex items-start gap-2">
            <span className="text-pink-500 font-mono text-xs mt-0.5">
              has:reminder
            </span>
            <span>Tasks with reminders</span>
          </li>
          <li className="flex items-start gap-2">
            <span className="text-red-500 font-mono text-xs mt-0.5">!term</span>
            <span>Negate any filter</span>
          </li>
        </ul>
        <div className="mt-3 pt-3 border-t border-slate-100 text-xs text-slate-400">
          Combine multiple filters for precise results
        </div>
      </div>
    </>
  );
}
