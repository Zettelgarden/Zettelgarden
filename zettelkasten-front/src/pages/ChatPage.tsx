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
import { ConversationSidebar } from "../components/chat/ConversationSidebar";

interface ChatPageProps { }

export function ChatPage({ }: ChatPageProps) {
  // ChatPage-specific state
  const [conversations, setConversations] = useState<ChatConversation[]>([]);
  const [sidebarOpen, setSidebarOpen] = useState(false);
  const [showAllRecent, setShowAllRecent] = useState(false);
  const [showInstructionsMenu, setShowInstructionsMenu] = useState(false);
  const [selectedTaskId, setSelectedTaskId] = useState<number | null>(null);
  const [showTaskDialog, setShowTaskDialog] = useState(false);
  const [searchQuery, setSearchQuery] = useState("");
  const [showStarredOnly, setShowStarredOnly] = useState(false);

  // Loading states for async operations
  const [loadingConversations, setLoadingConversations] = useState(false);
  const [loadingConversationIds, setLoadingConversationIds] = useState<Set<string>>(new Set());
  const [deletingConversationIds, setDeletingConversationIds] = useState<Set<string>>(new Set());
  const [starringConversationIds, setStarringConversationIds] = useState<Set<string>>(new Set());
  const [regeneratingMessageIds, setRegeneratingMessageIds] = useState<Set<string>>(new Set());

  const { conversationId, setConversationId } = useChatContext();
  const { hasSubscription } = useAuth();
  const { showToast } = useToast();

  // Track the last synced conversation ID to prevent circular updates
  const lastSyncedConversationIdRef = React.useRef<string | null>(null);

  // Use the shared chat hook
  const chatHook = useChat({
    onConversationChange: (conversation) => {
      // Only update the context conversationId if it actually changed
      // This prevents a circular update loop where:
      // 1. loadConversation updates currentConversation
      // 2. onConversationChange fires and calls setConversationId
      // 3. The effect depending on conversationId + currentConversation re-runs
      if (conversation && conversation.id !== lastSyncedConversationIdRef.current) {
        lastSyncedConversationIdRef.current = conversation.id;
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

    // Clear any previous conversationId from context to avoid loading old chats
    // This ensures the user starts with a clean slate when navigating to /app/chat
    setConversationId("");

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

  // Track previous conversationId to detect actual changes
  const prevConversationIdRef = React.useRef<string | null>(null);

  // Load specific conversation if set in context
  useEffect(() => {
    // Skip loading draft conversations (they start with "draft-") as they don't exist in the backend yet
    // Only load when conversationId actually changes (not just the conversation object)
    if (conversationId && conversationId !== prevConversationIdRef.current && !conversationId.startsWith('draft-')) {
      if (conversationId !== chatHook.currentConversation?.id) {
        loadConversation(conversationId);
      }
      prevConversationIdRef.current = conversationId;
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

  // Include draft conversation in the display list if it exists and has no messages
  const displayConversations = chatHook.isDraftConversation && chatHook.messages.length === 0
    ? [chatHook.currentConversation!, ...conversations]
    : conversations;

  // Filter conversations based on search query and starred filter
  const filteredConversations = displayConversations.filter(conv => {
    // Apply starred-only filter
    if (showStarredOnly && !conv.starred) {
      return false;
    }

    // Apply search filter
    if (searchQuery.trim()) {
      const query = searchQuery.toLowerCase();
      const title = (conv.title || "").toLowerCase();
      return title.includes(query);
    }

    return true;
  });

  return (
    <div className="flex h-screen bg-white">
      {/* Conversations Sidebar */}
      <ConversationSidebar
        sidebarOpen={sidebarOpen}
        setSidebarOpen={setSidebarOpen}
        conversations={conversations}
        currentConversationId={chatHook.currentConversation?.id || null}
        loadingConversations={loadingConversations}
        loadingConversationIds={loadingConversationIds}
        deletingConversationIds={deletingConversationIds}
        starringConversationIds={starringConversationIds}
        showAllRecent={showAllRecent}
        setShowAllRecent={setShowAllRecent}
        displayConversations={filteredConversations}
        isLoading={chatHook.isLoading}
        searchQuery={searchQuery}
        setSearchQuery={setSearchQuery}
        showStarredOnly={showStarredOnly}
        setShowStarredOnly={setShowStarredOnly}
        onCreateNewConversation={createNewConversation}
        onLoadConversation={loadConversation}
        onStarConversation={starConversation}
        onDeleteConversation={deleteConversation}
      />

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