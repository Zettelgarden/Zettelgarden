import React, { useState, useEffect } from "react";
import {
  ChatConversation,
  getConversations,
  deleteConversation as apiDeleteConversation,
  starConversation as apiStarConversation,
  regenerateMessage as apiRegenerateMessage,
} from "../api/chat";
import { setDocumentTitle } from "../utils/title";
import { Button } from "../components/Button";
import { useChatContext } from "../contexts/ChatContext";
import { ChatInterface } from "../components/chat/ChatInterface";
import { InstructionsMenu } from "../components/chat/InstructionsMenu";
import { TaskDialog } from "../components/tasks/TaskDialog";
import { Link } from "react-router-dom";
import { useAuth } from "../contexts/AuthContext";
import { useChat } from "../hooks/useChat";
import { useToast } from "../components/toast/ToastContext";

interface ChatPageProps { }

export function ChatPage({ }: ChatPageProps) {
  // ChatPage-specific state
  const [conversations, setConversations] = useState<ChatConversation[]>([]);
  const [sidebarOpen, setSidebarOpen] = useState(false);
  const [showAllRecent, setShowAllRecent] = useState(false);
  const [showInstructionsMenu, setShowInstructionsMenu] = useState(false);
  const [selectedTaskId, setSelectedTaskId] = useState<number | null>(null);
  const [showTaskDialog, setShowTaskDialog] = useState(false);

  // Loading states for async operations
  const [loadingConversations, setLoadingConversations] = useState(false);
  const [loadingConversationIds, setLoadingConversationIds] = useState<Set<string>>(new Set());
  const [deletingConversationIds, setDeletingConversationIds] = useState<Set<string>>(new Set());
  const [starringConversationIds, setStarringConversationIds] = useState<Set<string>>(new Set());
  const [regeneratingMessageIds, setRegeneratingMessageIds] = useState<Set<string>>(new Set());

  const { conversationId, setConversationId } = useChatContext();
  const { hasSubscription } = useAuth();
  const { showToast } = useToast();

  // Use the shared chat hook
  const chatHook = useChat({
    onConversationChange: (conversation) => {
      // This callback is called whenever the current conversation changes
      // We can use it to sync the conversation ID context
      if (conversation) {
        setConversationId(conversation.id);
      }
    },
    onConversationCreated: (conversation) => {
      // When a draft conversation becomes real (first message sent),
      // add it to the conversations list and refresh
      setConversations(prev => [conversation, ...prev]);
    }
  });

  useEffect(() => {
    setDocumentTitle("Chat");
    loadConversations();
    handleUrlParams();
  }, []);

  const handleUrlParams = async () => {
    const urlParams = new URLSearchParams(window.location.search);
    const message = urlParams.get('message');
    const cardsParam = urlParams.get('cards');
    const newParam = urlParams.get('new');

    if (message) {
      // Clear URL params to avoid re-triggering
      window.history.replaceState({}, '', '/app/chat');

      // Parse referenced cards
      const referencedCards = cardsParam ? cardsParam.split(',').filter(Boolean) : undefined;

      // Create new conversation and send message
      await createNewConversationWithMessage(message, referencedCards);
    } else if (newParam === 'true') {
      // Clear URL params to avoid re-triggering
      window.history.replaceState({}, '', '/app/chat');

      // Create new empty conversation
      await createNewConversation();
    }
  };

  const createNewConversationWithMessage = async (message: string, referencedCards?: string[]) => {
    try {
      const newConv = await chatHook.createNewConversation("", chatHook.selectedModel);

      setConversations(prev => [newConv, ...prev]);

      // Send the message directly without setting the input
      await chatHook.sendMessageToConversation(newConv.id, message, referencedCards);

      showToast("success", "Conversation started", "Your message has been sent to the AI.");
    } catch (error) {
      console.error("Failed to create conversation with message:", error);
      showToast("error", "Failed to start conversation", "Unable to create new chat with your message.");
    }
  };

  // Load specific conversation if set in context
  useEffect(() => {
    if (conversationId && conversationId !== chatHook.currentConversation?.id) {
      loadConversation(conversationId);
    }
  }, [conversationId]);

  const loadConversations = async () => {
    setLoadingConversations(true);
    try {
      const convs = await getConversations();

      if (!convs) {
        setConversations([]);
        return;
      }

      setConversations(convs);

      // If no current conversation and we have conversations, load the most recent one
      if (!chatHook.currentConversation && convs.length > 0) {
        await loadConversation(convs[0].id);
      }
    } catch (error) {
      console.error("Failed to load conversations:", error);
      showToast("error", "Failed to load conversations", "Please try refreshing the page.");
    } finally {
      setLoadingConversations(false);
    }
  };

  const loadConversation = async (conversationId: string) => {
    setLoadingConversationIds(prev => {
      const next = new Set(prev);
      next.add(conversationId);
      return next;
    });

    try {
      await chatHook.loadConversation(conversationId);
    } catch (error) {
      console.error("Failed to load conversation:", error);
      showToast("error", "Failed to load conversation", "The selected chat could not be loaded.");
    } finally {
      setLoadingConversationIds(prev => {
        const next = new Set(prev);
        next.delete(conversationId);
        return next;
      });
    }
  };

  const createNewConversation = async () => {
    try {
      const newConv = await chatHook.createNewConversation("", chatHook.selectedModel);

      // Draft conversations are not added to the conversations list until a message is sent
      // The chat interface will still show the current conversation
      showToast("success", "New conversation created");
    } catch (error) {
      console.error("Failed to create conversation:", error);
      showToast("error", "Failed to create new conversation", "Please try again.");
    }
  };


  const deleteConversation = async (conversationId: string) => {
    if (deletingConversationIds.has(conversationId)) return;
    if (!confirm("Are you sure you want to delete this conversation?")) return;

    setDeletingConversationIds(prev => {
      const next = new Set(prev);
      next.add(conversationId);
      return next;
    });

    try {
      await apiDeleteConversation(conversationId);

      let remaining: ChatConversation[] = [];
      setConversations(prev => {
        remaining = prev.filter(c => c.id !== conversationId);
        return remaining;
      });

      if (chatHook.currentConversation?.id === conversationId) {
        if (remaining.length > 0) {
          await loadConversation(remaining[0].id);
        } else {
          chatHook.setCurrentConversation(null);
          chatHook.setMessages([]);
          setConversationId("");
        }
      }

      showToast("success", "Conversation deleted");
    } catch (error) {
      console.error("Failed to delete conversation:", error);
      showToast("error", "Failed to delete conversation", "The chat could not be deleted.");
    } finally {
      setDeletingConversationIds(prev => {
        const next = new Set(prev);
        next.delete(conversationId);
        return next;
      });
    }
  };

  const starConversation = async (conversationId: string) => {
    if (starringConversationIds.has(conversationId)) return;

    setStarringConversationIds(prev => {
      const next = new Set(prev);
      next.add(conversationId);
      return next;
    });

    try {
      const updatedConv = await apiStarConversation(conversationId);
      setConversations(prev => prev.map(c => (c.id === conversationId ? updatedConv : c)));
      if (chatHook.currentConversation?.id === conversationId) {
        chatHook.setCurrentConversation(updatedConv);
      }
      showToast("success", updatedConv.starred ? "Conversation starred" : "Conversation unstarred");
    } catch (error) {
      console.error("Failed to star conversation:", error);
      showToast("error", "Failed to star conversation", "Unable to toggle star status.");
    } finally {
      setStarringConversationIds(prev => {
        const next = new Set(prev);
        next.delete(conversationId);
        return next;
      });
    }
  };

  const handleCardClick = (cardPk: string) => {
    // Navigate to the card page using the card_id
    window.open(`/app/card/${encodeURIComponent(cardPk)}`, '_blank');
  };

  const handleTaskClick = (taskId: number) => {
    setSelectedTaskId(taskId);
    setShowTaskDialog(true);
  };

  const handleTaskDialogClose = () => {
    setShowTaskDialog(false);
    setSelectedTaskId(null);
  };

  const handleTagClick = (tag: string) => {
    // No-op for now, could filter tasks by tag in the future
  };

  const handleRegenerateMessage = async (messageId: string) => {
    if (!chatHook.currentConversation) return;
    if (regeneratingMessageIds.has(messageId)) return;

    setRegeneratingMessageIds(prev => {
      const next = new Set(prev);
      next.add(messageId);
      return next;
    });

    try {
      await apiRegenerateMessage(chatHook.currentConversation.id, messageId);

      // Refresh messages to get the updated message with pending status
      await chatHook.refreshMessages(chatHook.currentConversation.id);

      // Start polling for updates
      chatHook.startPolling(chatHook.currentConversation.id);

      showToast("info", "Regenerating response...", "AI is creating a new reply.");
    } catch (error) {
      console.error("Failed to regenerate message:", error);
      showToast("error", "Failed to regenerate message", "Unable to create a new response.");
    } finally {
      setRegeneratingMessageIds(prev => {
        const next = new Set(prev);
        next.delete(messageId);
        return next;
      });
    }
  };

  // Separate starred and recent conversations
  // Include draft conversation in the display list if it exists and has no messages
  const displayConversations = chatHook.isDraftConversation && chatHook.messages.length === 0
    ? [chatHook.currentConversation!, ...conversations]
    : conversations;

  const starredConversations = displayConversations.filter(conv => conv.starred);
  const allRecentConversations = displayConversations.filter(conv => !conv.starred);

  // Limit recent conversations display
  const RECENT_LIMIT = 25;
  const displayedRecentConversations = showAllRecent
    ? allRecentConversations
    : allRecentConversations.slice(0, RECENT_LIMIT);
  const remainingRecentCount = allRecentConversations.length - RECENT_LIMIT;

  return (
    <div className="flex h-screen bg-white">
      {/* Conversations Sidebar */}
      <div className={`${sidebarOpen ? 'w-80' : 'w-0'} bg-gray-50 border-r border-gray-200 flex flex-col transition-all duration-300 overflow-hidden`}>
        <div className="pt-4 px-4 pb-2">
          <div className="flex items-center justify-between mb-6">
            <Button
              onClick={createNewConversation}
              disabled={chatHook.isLoading}
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
                    <div
                      key={conv.id}
                      onClick={() => {
                        if (
                          loadingConversationIds.has(conv.id) ||
                          deletingConversationIds.has(conv.id) ||
                          starringConversationIds.has(conv.id)
                        ) {
                          return;
                        }
                        loadConversation(conv.id);
                      }}
                      className={`group relative p-2 mx-1 mb-1 rounded-lg transition-all duration-200 hover:bg-white ${chatHook.currentConversation?.id === conv.id ? 'bg-white shadow-sm' : ''} ${(loadingConversationIds.has(conv.id) || deletingConversationIds.has(conv.id) || starringConversationIds.has(conv.id)) ? 'opacity-60 cursor-not-allowed' : 'cursor-pointer'}`}
                    >
                      <div className="flex items-center justify-between">
                        <div className="flex-1 min-w-0">
                          <h3 className="font-medium text-gray-900 text-sm truncate">
                            {conv.title || "Untitled Chat"}
                          </h3>
                        </div>
                        <div className={`flex items-center space-x-1 ml-2 ${loadingConversationIds.has(conv.id) ? 'opacity-100' : 'opacity-0 group-hover:opacity-100'} transition-opacity`}>
                          {loadingConversationIds.has(conv.id) && (
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
                            disabled={loadingConversationIds.has(conv.id) || deletingConversationIds.has(conv.id) || starringConversationIds.has(conv.id)}
                            onClick={(e) => {
                              e.stopPropagation();
                              starConversation(conv.id);
                            }}
                            className={`text-sm p-1 rounded hover:bg-gray-100 transition-colors disabled:opacity-50 disabled:cursor-not-allowed ${conv.starred ? 'text-yellow-500' : 'text-gray-500 hover:text-yellow-500'
                              }`}
                          >
                            <svg className="w-3 h-3" fill={conv.starred ? "currentColor" : "none"} stroke="currentColor" viewBox="0 0 24 24">
                              <path strokeLinecap="round" strokeLinejoin="round" strokeWidth={2} d="M11.049 2.927c.3-.921 1.603-.921 1.902 0l1.519 4.674a1 1 0 00.95.69h4.915c.969 0 1.371 1.24.588 1.81l-3.976 2.888a1 1 0 00-.363 1.118l1.518 4.674c.3.922-.755 1.688-1.538 1.118l-3.976-2.888a1 1 0 00-1.176 0l-3.976 2.888c-.783.57-1.838-.197-1.538-1.118l1.518-4.674a1 1 0 00-.363-1.118l-3.976-2.888c-.784-.57-.38-1.81.588-1.81h4.914a1 1 0 00.951-.69l1.519-4.674z" />
                            </svg>
                          </button>
                          <button
                            disabled={loadingConversationIds.has(conv.id) || deletingConversationIds.has(conv.id) || starringConversationIds.has(conv.id)}
                            onClick={(e) => {
                              e.stopPropagation();
                              deleteConversation(conv.id);
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
                    <div
                      key={conv.id}
                      onClick={() => {
                        if (
                          loadingConversationIds.has(conv.id) ||
                          deletingConversationIds.has(conv.id) ||
                          starringConversationIds.has(conv.id)
                        ) {
                          return;
                        }
                        loadConversation(conv.id);
                      }}
                      className={`group relative p-2 mx-1 mb-1 rounded-lg transition-all duration-200 hover:bg-white ${chatHook.currentConversation?.id === conv.id ? 'bg-white shadow-sm' : ''} ${(loadingConversationIds.has(conv.id) || deletingConversationIds.has(conv.id) || starringConversationIds.has(conv.id)) ? 'opacity-60 cursor-not-allowed' : 'cursor-pointer'}`}
                    >
                      <div className="flex items-center justify-between">
                        <div className="flex-1 min-w-0">
                          <h3 className="font-medium text-gray-900 text-sm truncate">
                            {conv.title || "Untitled Chat"}
                          </h3>
                        </div>
                        <div className={`flex items-center space-x-1 ml-2 ${loadingConversationIds.has(conv.id) ? 'opacity-100' : 'opacity-0 group-hover:opacity-100'} transition-opacity`}>
                          {loadingConversationIds.has(conv.id) && (
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
                            disabled={loadingConversationIds.has(conv.id) || deletingConversationIds.has(conv.id) || starringConversationIds.has(conv.id)}
                            onClick={(e) => {
                              e.stopPropagation();
                              starConversation(conv.id);
                            }}
                            className={`text-sm p-1 rounded hover:bg-gray-100 transition-colors disabled:opacity-50 disabled:cursor-not-allowed ${conv.starred ? 'text-yellow-500' : 'text-gray-500 hover:text-yellow-500'
                              }`}
                          >
                            <svg className="w-3 h-3" fill={conv.starred ? "currentColor" : "none"} stroke="currentColor" viewBox="0 0 24 24">
                              <path strokeLinecap="round" strokeLinejoin="round" strokeWidth={2} d="M11.049 2.927c.3-.921 1.603-.921 1.902 0l1.519 4.674a1 1 0 00.95.69h4.915c.969 0 1.371 1.24.588 1.81l-3.976 2.888a1 1 0 00-.363 1.118l1.518 4.674c.3.922-.755 1.688-1.538 1.118l-3.976-2.888a1 1 0 00-1.176 0l-3.976 2.888c-.783.57-1.838-.197-1.538-1.118l1.518-4.674a1 1 0 00-.363-1.118l-3.976-2.888c-.784-.57-.38-1.81.588-1.81h4.914a1 1 0 00.951-.69l1.519-4.674z" />
                            </svg>
                          </button>
                          <button
                            disabled={loadingConversationIds.has(conv.id) || deletingConversationIds.has(conv.id) || starringConversationIds.has(conv.id)}
                            onClick={(e) => {
                              e.stopPropagation();
                              deleteConversation(conv.id);
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

      {/* Chat Area */}
      <div className="flex-1 flex flex-col">
        {chatHook.currentConversation ? (
          <>
            {/* Chat Header */}
            <div className="bg-white border-b border-gray-200 p-2 shadow-sm">
              <div className="flex items-center gap-3">
                {!sidebarOpen && (
                  <button
                    onClick={() => setSidebarOpen(true)}
                    className="text-gray-500 hover:text-gray-700 p-2 rounded-lg hover:bg-gray-100 transition-colors mr-2"
                  >
                    <svg className="w-5 h-5" fill="none" stroke="currentColor" viewBox="0 0 24 24">
                      <path strokeLinecap="round" strokeLinejoin="round" strokeWidth={2} d="M4 6h16M4 12h16M4 18h16" />
                    </svg>
                  </button>
                )}
                <div className="flex-1">
                  <h2 className="text-lg font-semibold text-gray-900">
                    {chatHook.currentConversation.title || "Untitled Chat"}
                  </h2>
                  <p className="text-xs text-gray-500 flex items-center gap-2">
                    <span className="w-2 h-2 bg-green-500 rounded-full"></span>
                    Model: {chatHook.currentConversation.model}
                  </p>
                </div>
                <button
                  onClick={() => setShowInstructionsMenu(true)}
                  className="text-gray-500 hover:text-gray-700 p-2 rounded-lg hover:bg-gray-100 transition-colors"
                  title="Chat Instructions"
                >
                  <svg className="w-5 h-5" fill="none" stroke="currentColor" viewBox="0 0 24 24">
                    <path strokeLinecap="round" strokeLinejoin="round" strokeWidth={2} d="M12 6.253v13m0-13C10.832 5.477 9.246 5 7.5 5S4.168 5.477 3 6.253v13C4.168 18.477 5.754 18 7.5 18s3.332.477 4.5 1.253m0-13C13.168 5.477 14.754 5 16.5 5c1.746 0 3.332.477 4.5 1.253v13C19.832 18.477 18.246 18 16.5 18c-1.746 0-3.332.477-4.5 1.253" />
                  </svg>
                </button>
              </div>
            </div>

            {/* Chat Interface */}
            <ChatInterface
              chatHook={chatHook}
              onCardClick={handleCardClick}
              onTaskClick={handleTaskClick}
              onRegenerateMessage={handleRegenerateMessage}
              placeholder="Ask about your cards... Type @ to mention a card"
              compact={false}
              showModelDropdown={true}
            />
          </>
        ) : (
          <div className="flex-1 flex flex-col bg-white">
            {!sidebarOpen && (
              <div className="p-6">
                <button
                  onClick={() => setSidebarOpen(true)}
                  className="text-gray-500 hover:text-gray-700 p-2 rounded-lg hover:bg-gray-100 transition-colors"
                >
                  <svg className="w-5 h-5" fill="none" stroke="currentColor" viewBox="0 0 24 24">
                    <path strokeLinecap="round" strokeLinejoin="round" strokeWidth={2} d="M4 6h16M4 12h16M4 18h16" />
                  </svg>
                </button>
              </div>
            )}
            <div className="flex-1 flex items-center justify-center">
              <div className="text-center text-gray-500 max-w-md mx-auto p-8">
                <div className="w-16 h-16 mx-auto mb-6 rounded-lg bg-gray-100 flex items-center justify-center">
                  <svg className="w-8 h-8 text-gray-600" fill="none" stroke="currentColor" viewBox="0 0 24 24">
                    <path strokeLinecap="round" strokeLinejoin="round" strokeWidth={2} d="M8 12h.01M12 12h.01M16 12h.01M21 12c0 4.418-4.03 8-9 8a9.863 9.863 0 01-4.255-.949L3 20l1.395-3.72C3.512 15.042 3 13.574 3 12c0-4.418 4.03-8 9-8s9 3.582 9 8z" />
                  </svg>
                </div>
                <h3 className="text-xl font-semibold text-gray-900 mb-3">Welcome to Chat</h3>
                <p className="text-gray-600 mb-6 leading-relaxed">Create a new conversation to start chatting with your knowledge base.</p>
                {!hasSubscription && (
                  <div className="text-center text-gray-500 mb-6 p-4 bg-gray-50 rounded-lg">
                    AI Agents are a Pro feature.
                    <br />
                    <Link to="/app/subscribe" className="text-blue-500 hover:underline">
                      Upgrade to Pro to unlock intelligent AI agents that can work with your knowledge base.
                    </Link>
                  </div>
                )}
                <Button
                  onClick={createNewConversation}
                  disabled={chatHook.isLoading || !hasSubscription}
                  className="bg-black hover:bg-gray-800 text-white rounded-lg px-6 py-3 transition-colors duration-200 disabled:opacity-50"
                >
                  <span className="flex items-center gap-2">
                    <svg className="w-4 h-4" fill="none" stroke="currentColor" viewBox="0 0 24 24">
                      <path strokeLinecap="round" strokeLinejoin="round" strokeWidth={2} d="M12 4v16m8-8H4" />
                    </svg>
                    <span className="font-medium">Start New Chat</span>
                  </span>
                </Button>
              </div>
            </div>
          </div>
        )}
      </div>

      {/* Instructions Menu */}
      <InstructionsMenu
        isOpen={showInstructionsMenu}
        onClose={() => setShowInstructionsMenu(false)}
      />

      {/* Task Dialog */}
      <TaskDialog
        taskId={selectedTaskId}
        isOpen={showTaskDialog}
        onClose={handleTaskDialogClose}
      />

    </div>
  );
}