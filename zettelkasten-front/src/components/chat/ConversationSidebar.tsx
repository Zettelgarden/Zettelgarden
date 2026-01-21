import React from "react";
import { ChatConversation } from "../../api/chat";
import { Button } from "../Button";

interface ConversationSidebarProps {
  // State
  sidebarOpen: boolean;
  setSidebarOpen: (open: boolean) => void;
  conversations: ChatConversation[];
  currentConversationId: string | null;

  // Loading states
  loadingConversations: boolean;
  loadingConversationIds: Set<string>;
  deletingConversationIds: Set<string>;
  starringConversationIds: Set<string>;

  // Conversation display options
  showAllRecent: boolean;
  setShowAllRecent: (show: boolean) => void;
  displayConversations: ChatConversation[];
  isLoading: boolean;

  // Callbacks
  onCreateNewConversation: () => void;
  onLoadConversation: (conversationId: string) => void;
  onStarConversation: (conversationId: string) => void;
  onDeleteConversation: (conversationId: string) => void;
}

export function ConversationSidebar({
  sidebarOpen,
  setSidebarOpen,
  conversations,
  currentConversationId,
  loadingConversations,
  loadingConversationIds,
  deletingConversationIds,
  starringConversationIds,
  showAllRecent,
  setShowAllRecent,
  displayConversations,
  isLoading,
  onCreateNewConversation,
  onLoadConversation,
  onStarConversation,
  onDeleteConversation,
}: ConversationSidebarProps) {
  // Separate starred and recent conversations
  const starredConversations = displayConversations.filter(conv => conv.starred);
  const allRecentConversations = displayConversations.filter(conv => !conv.starred);

  // Limit recent conversations display
  const RECENT_LIMIT = 25;
  const displayedRecentConversations = showAllRecent
    ? allRecentConversations
    : allRecentConversations.slice(0, RECENT_LIMIT);
  const remainingRecentCount = allRecentConversations.length - RECENT_LIMIT;

  return (
    <div className={`${sidebarOpen ? 'w-80' : 'w-0'} bg-gray-50 border-r border-gray-200 flex flex-col transition-all duration-300 overflow-hidden`}>
      <div className="pt-4 px-4 pb-2">
        <div className="flex items-center justify-between mb-6">
          <Button
            onClick={onCreateNewConversation}
            disabled={isLoading}
            className="flex-1 mr-3 bborder border-gray-300 rounded-lg px-4 py-2.5 text-sm font-medium duration-200 flex items-center justify-center gap-2"
          >

            New chat
          </Button>
          <button
            onClick={() => setSidebarOpen(false)}
            className="text-gray-500 hover:text-gray-700 p-2 rounded-lg hover:bg-gray-100 transition-colors"
          >
            <svg className="w-5 h-5" fill="none" stroke="currentColor" viewBox="0 0 24 24">
              <path strokeLinecap="round" strokeLinejoin="round" strokeWidth={2} d="M6 18L18 6M6 6l12 12" />
            </svg>
          </button>
        </div>
      </div>

      {/* Conversations List */}
      <div className="flex-1 overflow-y-auto px-2">
        {loadingConversations ? (
          <div className="p-4 text-gray-500 text-center text-sm">
            Loading conversations...
          </div>
        ) : conversations.length === 0 ? (
          <div className="p-4 text-gray-500 text-center text-sm">
            No conversations yet
          </div>
        ) : (
          <>
            {/* Starred Conversations Section */}
            {starredConversations.length > 0 && (
              <div className="mb-4">
                <div className="px-2 py-1 mb-2">
                  <h4 className="text-xs font-semibold text-gray-600 uppercase tracking-wider flex items-center gap-2">
                    <svg className="w-3 h-3 text-yellow-500" fill="currentColor" viewBox="0 0 24 24">
                      <path strokeLinecap="round" strokeLinejoin="round" strokeWidth={2} d="M11.049 2.927c.3-.921 1.603-.921 1.902 0l1.519 4.674a1 1 0 00.95.69h4.915c.969 0 1.371 1.24.588 1.81l-3.976 2.888a1 1 0 00-.363 1.118l1.518 4.674c.3.922-.755 1.688-1.538 1.118l-3.976-2.888a1 1 0 00-1.176 0l-3.976 2.888c-.783.57-1.838-.197-1.538-1.118l1.518-4.674a1 1 0 00-.363-1.118l-3.976-2.888c-.784-.57-.38-1.81.588-1.81h4.914a1 1 0 00.951-.69l1.519-4.674z" />
                    </svg>
                    Starred
                  </h4>
                </div>
                {starredConversations.map((conv) => (
                  <ConversationItem
                    key={conv.id}
                    conversation={conv}
                    isCurrent={currentConversationId === conv.id}
                    isLoading={loadingConversationIds.has(conv.id)}
                    isDeleting={deletingConversationIds.has(conv.id)}
                    isStarring={starringConversationIds.has(conv.id)}
                    onLoad={() => onLoadConversation(conv.id)}
                    onStar={() => onStarConversation(conv.id)}
                    onDelete={() => onDeleteConversation(conv.id)}
                  />
                ))}
              </div>
            )}

            {/* Recent Conversations Section */}
            {allRecentConversations.length > 0 && (
              <div>
                <div className="px-2 py-1 mb-2">
                  <h4 className="text-xs font-semibold text-gray-600 uppercase tracking-wider flex items-center gap-2">
                    <svg className="w-3 h-3 text-gray-500" fill="none" stroke="currentColor" viewBox="0 0 24 24">
                      <path strokeLinecap="round" strokeLinejoin="round" strokeWidth={2} d="M12 8v4l3 3m6-3a9 9 0 11-18 0 9 9 0 0118 0z" />
                    </svg>
                    Recent
                  </h4>
                </div>
                {displayedRecentConversations.map((conv) => (
                  <ConversationItem
                    key={conv.id}
                    conversation={conv}
                    isCurrent={currentConversationId === conv.id}
                    isLoading={loadingConversationIds.has(conv.id)}
                    isDeleting={deletingConversationIds.has(conv.id)}
                    isStarring={starringConversationIds.has(conv.id)}
                    onLoad={() => onLoadConversation(conv.id)}
                    onStar={() => onStarConversation(conv.id)}
                    onDelete={() => onDeleteConversation(conv.id)}
                  />
                ))}

                {/* Show More/Less Button */}
                {remainingRecentCount > 0 && (
                  <div className="px-2 mt-2">
                    <button
                      onClick={() => setShowAllRecent(!showAllRecent)}
                      className="w-full text-left px-2 py-2 text-xs text-gray-500 hover:text-gray-700 hover:bg-gray-100 rounded-lg transition-colors flex items-center justify-between"
                    >
                      <span>
                        {showAllRecent
                          ? 'Show less'
                          : `Show ${remainingRecentCount} more`}
                      </span>
                      <svg
                        className={`w-3 h-3 transition-transform ${showAllRecent ? 'rotate-180' : ''}`}
                        fill="none"
                        stroke="currentColor"
                        viewBox="0 0 24 24"
                      >
                        <path strokeLinecap="round" strokeLinejoin="round" strokeWidth={2} d="M19 9l-7 7-7-7" />
                      </svg>
                    </button>
                  </div>
                )}
              </div>
            )}
          </>
        )}
      </div>
    </div>
  );
}

interface ConversationItemProps {
  conversation: ChatConversation;
  isCurrent: boolean;
  isLoading: boolean;
  isDeleting: boolean;
  isStarring: boolean;
  onLoad: () => void;
  onStar: () => void;
  onDelete: () => void;
}

function ConversationItem({
  conversation,
  isCurrent,
  isLoading,
  isDeleting,
  isStarring,
  onLoad,
  onStar,
  onDelete,
}: ConversationItemProps) {
  const isDisabled = isLoading || isDeleting || isStarring;

  return (
    <div
      onClick={isDisabled ? undefined : onLoad}
      className={`group relative p-2 mx-1 mb-1 rounded-lg transition-all duration-200 hover:bg-white ${isCurrent ? 'bg-white shadow-sm' : ''} ${isDisabled ? 'opacity-60 cursor-not-allowed' : 'cursor-pointer'}`}
    >
      <div className="flex items-center justify-between">
        <div className="flex-1 min-w-0">
          <h3 className="font-medium text-gray-900 text-sm truncate">
            {conversation.title || "Untitled Chat"}
          </h3>
        </div>
        <div className={`flex items-center space-x-1 ml-2 ${isLoading ? 'opacity-100' : 'opacity-0 group-hover:opacity-100'} transition-opacity`}>
          {isLoading && (
            <svg
              className="w-3 h-3 text-gray-400 animate-spin"
              viewBox="0 0 24 24"
            >
              <circle
                className="opacity-25"
                cx="12"
                cy="12"
                r="10"
                stroke="currentColor"
                strokeWidth="4"
                fill="none"
              />
              <path
                className="opacity-75"
                fill="currentColor"
                d="M4 12a8 8 0 018-8v3a5 5 0 00-5 5H4z"
              />
            </svg>
          )}
          <button
            disabled={isDisabled}
            onClick={(e) => {
              e.stopPropagation();
              onStar();
            }}
            className={`text-sm p-1 rounded hover:bg-gray-100 transition-colors disabled:opacity-50 disabled:cursor-not-allowed ${conversation.starred ? 'text-yellow-500' : 'text-gray-500 hover:text-yellow-500'
              }`}
          >
            <svg className="w-3 h-3" fill={conversation.starred ? "currentColor" : "none"} stroke="currentColor" viewBox="0 0 24 24">
              <path strokeLinecap="round" strokeLinejoin="round" strokeWidth={2} d="M11.049 2.927c.3-.921 1.603-.921 1.902 0l1.519 4.674a1 1 0 00.95.69h4.915c.969 0 1.371 1.24.588 1.81l-3.976 2.888a1 1 0 00-.363 1.118l1.518 4.674c.3.922-.755 1.688-1.538 1.118l-3.976-2.888a1 1 0 00-1.176 0l-3.976 2.888c-.783.57-1.838-.197-1.538-1.118l1.518-4.674a1 1 0 00-.363-1.118l-3.976-2.888c-.784-.57-.38-1.81.588-1.81h4.914a1 1 0 00.951-.69l1.519-4.674z" />
            </svg>
          </button>
          <button
            disabled={isDisabled}
            onClick={(e) => {
              e.stopPropagation();
              onDelete();
            }}
            className="text-gray-500 hover:text-red-500 text-sm p-1 rounded hover:bg-gray-100 transition-colors disabled:opacity-50 disabled:cursor-not-allowed"
          >
            <svg className="w-3 h-3" fill="none" stroke="currentColor" viewBox="0 0 24 24">
              <path strokeLinecap="round" strokeLinejoin="round" strokeWidth={2} d="M19 7l-.867 12.142A2 2 0 0116.138 21H7.862a2 2 0 01-1.995-1.858L5 7m5 4v6m4-6v6m1-10V4a1 1 0 00-1-1h-4a1 1 0 00-1 1v3M4 7h16" />
            </svg>
          </button>
        </div>
      </div>
    </div>
  );
}
