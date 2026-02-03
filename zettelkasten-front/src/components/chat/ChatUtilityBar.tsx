import React from "react";

interface ChatUtilityBarProps {
  hasLastCleared: boolean;
  isSending: boolean;
  onClear: () => void;
  onRestoreLast: () => void;
  onSettings: () => void;
  hasSubscription?: boolean;
}

export function ChatUtilityBar({
  hasLastCleared,
  isSending,
  onClear,
  onRestoreLast,
  onSettings,
  hasSubscription = true,
}: ChatUtilityBarProps) {
  return (
    <div className="bg-white border-b border-gray-200 px-3 py-2 sm:px-4 sm:py-2">
      <div className="flex items-center justify-between max-w-6xl mx-auto">
        <div className="flex items-center gap-2">
          <h1 className="text-base sm:text-lg font-semibold text-gray-900">Chat</h1>
          {!hasSubscription && (
            <span className="text-xs bg-yellow-100 text-yellow-700 px-2 py-0.5 rounded-full font-medium">
              PRO
            </span>
          )}
        </div>
        <div className="flex items-center gap-1 sm:gap-2">
          <button
            onClick={onSettings}
            aria-label="Chat settings"
            className="text-gray-600 hover:text-gray-900 p-1.5 rounded-lg hover:bg-gray-100 transition-colors focus:outline-none focus:ring-2 focus:ring-blue-500"
            title="Chat Settings"
          >
            <svg className="w-4 h-4" fill="none" stroke="currentColor" viewBox="0 0 24 24">
              <path strokeLinecap="round" strokeLinejoin="round" strokeWidth={2} d="M10.325 4.317c.426-1.756 2.924-1.756 3.35 0a1.724 1.724 0 002.573 1.066c1.543-.94 3.31.826 2.37 2.37a1.724 1.724 0 001.065 2.572c1.756.426 1.756 2.924 0 3.35a1.724 1.724 0 00-1.066 2.573c.94 1.543-.826 3.31-2.37 2.37a1.724 1.724 0 00-2.572 1.065c-.426 1.756-2.924 1.756-3.35 0a1.724 1.724 0 00-2.573-1.066c-1.543.94-3.31-.826-2.37-2.37a1.724 1.724 0 00-1.065-2.572c-1.756-.426-1.756-2.924 0-3.35a1.724 1.724 0 001.066-2.573c-.94-1.543.826-3.31 2.37-2.37.996.608 2.296.07 2.572-1.065z" />
              <path strokeLinecap="round" strokeLinejoin="round" strokeWidth={2} d="M15 12a3 3 0 11-6 0 3 3 0 016 0z" />
            </svg>
          </button>
          <button
            onClick={onClear}
            disabled={isSending}
            aria-label="Clear chat history"
            className="text-gray-600 hover:text-gray-900 px-2 py-1.5 sm:px-3 sm:py-1.5 text-sm font-medium rounded-lg hover:bg-gray-100 transition-colors disabled:opacity-50 disabled:cursor-not-allowed focus:outline-none focus:ring-2 focus:ring-blue-500"
          >
            Clear
          </button>
          {hasLastCleared && (
            <button
              onClick={onRestoreLast}
              aria-label="Restore last cleared chat"
              className="text-blue-600 hover:text-blue-700 px-2 py-1.5 sm:px-3 sm:py-1.5 text-sm font-medium rounded-lg hover:bg-blue-50 transition-colors focus:outline-none focus:ring-2 focus:ring-blue-500"
            >
              <span className="hidden sm:inline">Restore Last</span>
              <span className="sm:hidden">Restore</span>
            </button>
          )}
        </div>
      </div>
    </div>
  );
}
