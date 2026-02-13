import React, { useState } from "react";
import { ToolError, ToolResultMetadata } from "../../api/chat";

interface ToolResultCardProps {
  messageId: string;
  toolName: string;
  result: Record<string, any>;
  metadata?: ToolResultMetadata;
  isCollapsed: boolean;
  onToggle: () => void;
  onRetry?: (messageId: string, conversationId: string) => void;
  conversationId: string;
  isRetrying?: boolean;
}

const errorTypeLabels: Record<string, { label: string; icon: string }> = {
  network: { label: "Network Error", icon: "🌐" },
  validation: { label: "Validation Error", icon: "⚠️" },
  database: { label: "Database Error", icon: "💾" },
  not_found: { label: "Not Found", icon: "🔍" },
  permission: { label: "Permission Error", icon: "🔒" },
  rate_limit: { label: "Rate Limited", icon: "⏱️" },
  timeout: { label: "Timeout", icon: "⏰" },
  unknown: { label: "Error", icon: "❌" },
};

export function ToolResultCard({
  messageId,
  toolName,
  result,
  metadata,
  isCollapsed,
  onToggle,
  onRetry,
  conversationId,
  isRetrying = false,
}: ToolResultCardProps) {
  const [showFullJson, setShowFullJson] = useState(false);

  // Check if result contains an error
  const errorData = result.error;
  const hasError = !!errorData;

  // Parse error data if present
  let toolError: ToolError | null = null;
  if (hasError) {
    if (typeof errorData === "string") {
      // Old format: just an error string
      toolError = {
        type: "unknown",
        message: errorData,
        retryable: true,
        tool_name: toolName,
      };
    } else if (typeof errorData === "object") {
      // New format: structured error object
      toolError = errorData as ToolError;
    }
  }

  const isError = hasError || metadata?.has_error;
  const bgColor = isError ? "bg-red-50 border-red-200" : "bg-amber-50 border-amber-200";

  const textColor = isError ? "text-red-700" : "text-amber-700";
  const bgHover = isError ? "hover:bg-red-100" : "hover:bg-amber-100";

  // Format tool name for display
  const displayName = toolName
    .split("_")
    .map((word) => word.charAt(0).toUpperCase() + word.slice(1))
    .join(" ");

  // Extract error type for badge
  const errorType = toolError?.type || "unknown";
  const errorTypeInfo = errorTypeLabels[errorType] || errorTypeLabels.unknown;

  return (
    <div className={`${bgColor} border rounded-xl shadow-sm overflow-hidden`}>
      <button
        onClick={onToggle}
        className={`w-full px-4 py-3 min-h-[44px] text-left transition-colors duration-200 ${bgHover} hover:shadow-sm`}
      >
        <div className={`flex items-center justify-between ${textColor}`}>
          <div className="flex items-center gap-2">
            <span className="text-lg">{isError ? errorTypeInfo.icon : "🔧"}</span>
            <span className="font-medium text-sm">{displayName}</span>
            {isError && toolError && (
              <span className="px-2.5 py-0.5 text-xs bg-red-200 text-red-800 rounded-full font-medium border border-red-300">
                {errorTypeInfo.label}
              </span>
            )}
          </div>
          <svg
            className={`w-4 h-4 transition-transform duration-200 ${isCollapsed ? "" : "rotate-180"}`}
            fill="none"
            stroke="currentColor"
            viewBox="0 0 24 24"
          >
            <path strokeLinecap="round" strokeLinejoin="round" strokeWidth={2} d="M19 9l-7 7-7-7" />
          </svg>
        </div>

        {/* Error summary - always visible for errors */}
        {isError && toolError && !isCollapsed && (
          <div className="mt-2 pl-8">
            <div className="text-sm font-medium text-red-800">
              {toolError.message}
            </div>
            {toolError.suggestion && (
              <div className="mt-2 inline-flex items-start gap-2 px-3 py-2 bg-red-100 border border-red-200 rounded-lg">
                <svg className="w-3.5 h-3.5 mt-0.5 flex-shrink-0 text-red-600" fill="currentColor" viewBox="0 0 20 20">
                  <path fillRule="evenodd" d="M18 10a8 8 0 11-16 0 8 8 0 0116 0zm-7-4a1 1 0 11-2 0 1 1 0 012 0zM9 9a1 1 0 000 2v3a1 1 0 001 1h1a1 1 0 100-2v-3a1 1 0 00-1-1H9z" clipRule="evenodd" />
                </svg>
                <span className="text-xs text-red-700">{toolError.suggestion}</span>
              </div>
            )}
          </div>
        )}
      </button>

      {!isCollapsed && (
        <div className="px-4 pb-3">
          {/* Retry button for retryable errors */}
          {isError && toolError?.retryable && onRetry && (
            <div className="mb-3 pl-8">
              <button
                onClick={() => onRetry(messageId, conversationId)}
                disabled={isRetrying}
                className="flex items-center gap-2 bg-red-500 hover:bg-red-600 text-white px-4 py-3 min-h-[44px] rounded-xl text-sm font-medium transition-colors duration-200 shadow-md disabled:opacity-50 disabled:cursor-not-allowed"
              >
                {isRetrying ? (
                  <>
                    <div className="w-3 h-3 border-2 border-white border-t-transparent rounded-full animate-spin"></div>
                    <span>Retrying...</span>
                  </>
                ) : (
                  <>
                    <svg className="w-3 h-3" fill="none" stroke="currentColor" viewBox="0 0 24 24">
                      <path strokeLinecap="round" strokeLinejoin="round" strokeWidth={2} d="M4 4v5h.582m15.356 2A8.001 8.001 0 004.582 9m0 0H9m11 11v-5h-.581m0 0a8.003 8.003 0 01-15.357-2m15.357 2H15" />
                    </svg>
                    <span>Retry Tool</span>
                  </>
                )}
              </button>
            </div>
          )}

          {/* JSON details */}
          <div className="pl-8">
            <div className="flex items-center justify-between mb-2">
              <span className="text-xs font-medium text-gray-600">
                {isError ? "Error Details" : "Tool Output"}
              </span>
              {Object.keys(result).length > 5 && (
                <button
                  onClick={() => setShowFullJson(!showFullJson)}
                  className="text-xs text-gray-500 hover:text-gray-700 transition-colors"
                >
                  {showFullJson ? "Hide" : "Show"} Full JSON
                </button>
              )}
            </div>
            <pre
              className={`text-xs overflow-x-auto whitespace-pre-wrap break-words font-mono p-2 rounded border ${
                isError
                  ? "bg-red-50 text-red-800 border-red-100"
                  : "bg-amber-50 text-amber-800 border-amber-100"
              }`}
            >
              {JSON.stringify(result, null, 2)}
            </pre>

            {/* Show arguments if available */}
            {metadata?.arguments && Object.keys(metadata.arguments).length > 0 && (
              <div className="mt-2">
                <button
                  onClick={() => setShowFullJson(!showFullJson)}
                  className="text-xs text-gray-500 hover:text-gray-700 transition-colors flex items-center gap-1"
                >
                  <svg
                    className={`w-3 h-3 transition-transform ${showFullJson ? "rotate-90" : ""}`}
                    fill="none"
                    stroke="currentColor"
                    viewBox="0 0 24 24"
                  >
                    <path strokeLinecap="round" strokeLinejoin="round" strokeWidth={2} d="M9 5l7 7-7 7" />
                  </svg>
                  Arguments Used
                </button>
                {showFullJson && (
                  <pre
                    className="text-xs text-gray-700 overflow-x-auto whitespace-pre-wrap break-words font-mono bg-gray-50 p-2 rounded border border-gray-200 mt-1"
                  >
                    {JSON.stringify(metadata.arguments, null, 2)}
                  </pre>
                )}
              </div>
            )}
          </div>
        </div>
      )}
    </div>
  );
}
