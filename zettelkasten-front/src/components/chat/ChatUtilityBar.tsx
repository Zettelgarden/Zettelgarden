import React from "react";

interface ChatUtilityBarProps {
  hasLastCleared: boolean;
  isSending: boolean;
  onClear: () => void;
  onRestoreLast: () => void;
  onSettings: () => void;
  onShowKeyboardHelp?: () => void;
  hasSubscription?: boolean;
}

export function ChatUtilityBar({
  hasLastCleared,
  isSending,
  onClear,
  onRestoreLast,
  onSettings,
  onShowKeyboardHelp,
  hasSubscription = true,
}: ChatUtilityBarProps) {
  return (
    <div className="bg-white border-b border-gray-200 px-3 py-2.5 sm:px-4 sm:py-3" role="banner">
      <div className="flex items-center justify-between max-w-6xl mx-auto">
        <div className="flex items-center gap-2.5">
          <div className="w-8 h-8 rounded-xl bg-blue-600 flex items-center justify-center shadow-md" aria-hidden="true">
            <svg className="w-4 h-4 text-white" fill="none" stroke="currentColor" viewBox="0 0 24 24" aria-hidden="true">
              <path strokeLinecap="round" strokeLinejoin="round" strokeWidth={2} d="M8 12h.01M12 12h.01M16 12h.01M21 12c0 4.418-4.03 8-9 8a9.863 9.863 0 01-4.255-.949L3 20l1.395-3.72C3.512 15.042 3 13.574 3 12c0-4.418 4.03-8 9-8s9 3.582 9 8z" />
            </svg>
          </div>
          <h1 className="text-base sm:text-lg font-semibold text-gray-900">Chat</h1>
          {!hasSubscription && (
            <span className="inline-flex items-center px-2 py-0.5 text-xs font-semibold bg-amber-100 text-amber-700 rounded-full border border-amber-200">
              PRO
            </span>
          )}
        </div>
        <div className="flex items-center gap-1.5 sm:gap-2">
          {onShowKeyboardHelp && (
            <button
              onClick={onShowKeyboardHelp}
              aria-label="Show keyboard shortcuts"
              className="group text-gray-500 hover:text-gray-700 p-2.5 min-w-[44px] min-h-[44px] flex items-center justify-center rounded-xl hover:bg-gray-100 transition-all duration-200 focus:outline-none focus:ring-2 focus:ring-blue-500/50 focus:bg-gray-100"
              title="Keyboard shortcuts (?)"
              type="button"
            >
              <svg className="w-4 h-4 text-gray-600" fill="none" stroke="currentColor" viewBox="0 0 24 24">
                <path strokeLinecap="round" strokeLinejoin="round" strokeWidth={2} d="M9.663 17h4.673M12 3v1m6.364 1.636l-.707.707M21 12c0 4.418-4.03 8-9 8a9.863 9.863 0 01-4.255-.949L3.34 16c-.77-1.333-2.694-1.333-3.464 0L3.34 16c-.77-1.333.192 3 1.732 3z" />
              </svg>
              <span className="ml-1 text-xs font-medium text-gray-600 group-hover:text-gray-800 hidden sm:inline">?</span>
            </button>
          )}
          <button
            onClick={onSettings}
            aria-label="Chat settings"
            className="group relative text-gray-500 hover:text-gray-700 p-2.5 min-w-[44px] min-h-[44px] flex items-center justify-center rounded-xl hover:bg-gray-100 transition-all duration-200 focus:outline-none focus:ring-2 focus:ring-blue-500/50 focus:bg-gray-100"
            title="Chat Settings"
            type="button"
          >
            <svg className="w-4 h-4 transition-transform duration-200 group-hover:rotate-45" fill="none" stroke="currentColor" viewBox="0 0 24 24">
              <path strokeLinecap="round" strokeLinejoin="round" strokeWidth={2} d="M10.325 4.317c.426-1.756 2.924 1.756 3.35 0a1.724 1.732 3L13.732 4c-.77-1.333-2.694-1.333-3.464 0L3.34 16c-.77-1.333-2.694-1.333-3.464 0L3.34 16c-.77-1.333.192 3 1.732 3z" />
            </svg>
          </button>
          <button
            onClick={onClear}
            disabled={isSending}
            aria-label="Clear chat history"
            className="group text-gray-600 hover:text-gray-900 px-4 py-3 min-h-[44px] text-sm font-medium rounded-xl hover:bg-gray-100 active:bg-gray-200 transition-all duration-200 disabled:opacity-50 disabled:cursor-not-allowed disabled:hover:bg-transparent"
          >
            Clear
          </button>
          {hasLastCleared && (
            <button
              onClick={onRestoreLast}
              aria-label="Restore last cleared chat"
              className="group inline-flex items-center gap-1.5 px-4 py-3 min-h-[44px] text-sm font-medium text-blue-600 bg-blue-50 hover:bg-blue-100 active:bg-blue-200 rounded-xl transition-colors duration-200 focus:outline-none focus:ring-2 focus:ring-blue-500/50"
            >
              <svg className="w-3.5 h-3.5 transition-transform duration-200 group-hover:-rotate-12" fill="none" stroke="currentColor" viewBox="0 0 24 24">
                <path strokeLinecap="round" strokeLinejoin="round" strokeWidth={2} d="M4 4v5h.582m15.356 2A8.001 8.001 0 004.582 9m0 0H9m11 11v-5h-.581m0 0a8.003 8.003 0 01-15.357-2m15.357 2H15" />
              </svg>
              <span className="hidden sm:inline">Restore Last</span>
            </button>
          )}
        </div>
      </div>
    </div>
  );
}
