import React from "react";

interface ChatUtilityBarProps {
  hasLastCleared: boolean;
  isSending: boolean;
  onClear: () => void;
  onRestoreLast: () => void;
  onInstructions: () => void;
  hasSubscription?: boolean;
}

export function ChatUtilityBar({
  hasLastCleared,
  isSending,
  onClear,
  onRestoreLast,
  onInstructions,
  hasSubscription = true,
}: ChatUtilityBarProps) {
  return (
    <div className="bg-white border-b border-gray-200 px-4 py-2">
      <div className="flex items-center justify-between max-w-6xl mx-auto">
        <div className="flex items-center gap-2">
          <h1 className="text-lg font-semibold text-gray-900">Chat</h1>
          {!hasSubscription && (
            <span className="text-xs bg-yellow-100 text-yellow-700 px-2 py-0.5 rounded-full font-medium">
              PRO
            </span>
          )}
        </div>
        <div className="flex items-center gap-2">
          <button
            onClick={onInstructions}
            aria-label="View chat instructions"
            className="text-gray-600 hover:text-gray-900 px-3 py-1.5 text-sm font-medium rounded-lg hover:bg-gray-100 transition-colors flex items-center gap-1.5 focus:outline-none focus:ring-2 focus:ring-blue-500"
          >
            <svg className="w-4 h-4" fill="none" stroke="currentColor" viewBox="0 0 24 24">
              <path strokeLinecap="round" strokeLinejoin="round" strokeWidth={2} d="M12 6.253v13m0-13C10.832 5.477 9.246 5 7.5 5S4.168 5.477 3 6.253v13C4.168 18.477 5.754 18 7.5 18s3.332.477 4.5 1.253m0-13C13.168 5.477 14.754 5 16.5 5c1.746 0 3.332.477 4.5 1.253v13C19.832 18.477 18.246 18 16.5 18c-1.746 0-3.332.477-4.5 1.253" />
            </svg>
            Instructions
          </button>
          <button
            onClick={onClear}
            disabled={isSending}
            aria-label="Clear chat history"
            className="text-gray-600 hover:text-gray-900 px-3 py-1.5 text-sm font-medium rounded-lg hover:bg-gray-100 transition-colors disabled:opacity-50 disabled:cursor-not-allowed focus:outline-none focus:ring-2 focus:ring-blue-500"
          >
            Clear
          </button>
          {hasLastCleared && (
            <button
              onClick={onRestoreLast}
              aria-label="Restore last cleared chat"
              className="text-blue-600 hover:text-blue-700 px-3 py-1.5 text-sm font-medium rounded-lg hover:bg-blue-50 transition-colors focus:outline-none focus:ring-2 focus:ring-blue-500"
            >
              Restore Last
            </button>
          )}
        </div>
      </div>
    </div>
  );
}
