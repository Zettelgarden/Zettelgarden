import React, { useState, useEffect } from "react";
import {
  regenerateMessage as apiRegenerateMessage,
  getChatModels,
  ChatModel,
  getChatInstructions,
  updateChatInstructions,
  getConversations,
} from "../api/chat";
import { setDocumentTitle } from "../utils/title";
import { ChatInterface } from "../components/chat/ChatInterface";
import { ChatUtilityBar } from "../components/chat/ChatUtilityBar";
import { TaskDialog } from "../components/tasks/TaskDialog";
import { Link, useLocation, useNavigate } from "react-router-dom";
import { useAuth } from "../contexts/AuthContext";
import { useChat } from "../hooks/useChat";
import { useToast } from "../components/toast/ToastContext";
import { useUIState } from "../contexts/UIStateContext";
import { useIsDesktop } from "../hooks/useWindowSize";

interface ChatPageProps { }

export function ChatPage({ }: ChatPageProps) {
  const location = useLocation();
  const navigate = useNavigate();
  const isDesktop = useIsDesktop(1024);
  const { setIsChatOpen } = useUIState();

  // Redirect desktop users to home with chat panel open
  useEffect(() => {
    if (isDesktop) {
      setIsChatOpen(true);
      navigate('/app', { replace: true });
    }
  }, [isDesktop, setIsChatOpen, navigate]);

  // Show loading message while redirecting on desktop
  if (isDesktop) {
    return (
      <div className="flex items-center justify-center h-screen bg-gray-50">
        <div className="text-gray-600">Opening chat...</div>
      </div>
    );
  }

  // ChatPage-specific state
  const [selectedTaskId, setSelectedTaskId] = useState<number | null>(null);
  const [showTaskDialog, setShowTaskDialog] = useState(false);
  const [regeneratingMessageIds, setRegeneratingMessageIds] = useState<Set<string>>(new Set());
  const [showClearConfirm, setShowClearConfirm] = useState(false);
  const [showSettingsDialog, setShowSettingsDialog] = useState(false);
  const [availableModels, setAvailableModels] = useState<ChatModel[]>([]);
  const [instructions, setInstructions] = useState("");
  const [instructionsHasChanges, setInstructionsHasChanges] = useState(false);
  const [isSavingInstructions, setIsSavingInstructions] = useState(false);

  const { hasSubscription } = useAuth();
  const { showToast } = useToast();

  // Use the shared chat hook
  const chatHook = useChat({
    initialModel: localStorage.getItem('chatSelectedModel') || "google/gemini-2.5-flash",
    enableStreaming: true,
  });

  useEffect(() => {
    setDocumentTitle("Chat");
    // Try to load the most recent conversation, then fall back to last cleared session, then create a draft
    const initializeChat = async () => {
      const loaded = await loadMostRecentConversation();
      if (!loaded && chatHook.lastClearedSession) {
        await chatHook.restoreLastCleared();
      } else if (!loaded && !chatHook.currentConversation) {
        chatHook.createNewConversation("", chatHook.selectedModel);
      }
    };
    initializeChat();
    loadAvailableModels();
    loadInstructions();
  }, []);

  // Handle URL params changing (e.g., when navigating from dashboard with a message)
  useEffect(() => {
    const urlParams = new URLSearchParams(location.search);
    const message = urlParams.get('message');
    const cardsParam = urlParams.get('cards');
    const newParam = urlParams.get('new');

    if (message) {
      // Clear URL params to avoid re-triggering
      window.history.replaceState({}, '', '/app/chat');

      // Parse referenced cards
      const referencedCards = cardsParam ? cardsParam.split(',').filter(Boolean) : undefined;

      // Send message to current session
      if (chatHook.currentConversation) {
        chatHook.sendMessageToConversation(chatHook.currentConversation.id, message, referencedCards);
        showToast("success", "Message sent", "Your message has been sent to the AI.");
      }
    } else if (newParam === 'true') {
      // Clear URL params to avoid re-triggering
      window.history.replaceState({}, '', '/app/chat');

      // Check if there are messages to clear before showing confirmation
      if (chatHook.messages.length > 0) {
        setShowClearConfirm(true);
      } else {
        // No messages to lose, just clear
        chatHook.clearChat();
      }
    }
    // eslint-disable-next-line react-hooks/exhaustive-deps
  }, [location.search]);

  const loadAvailableModels = async () => {
    try {
      const models = await getChatModels();
      setAvailableModels(models);
    } catch (error) {
      console.error("Failed to load chat models:", error);
    }
  };

  const loadInstructions = async () => {
    try {
      const response = await getChatInstructions();
      setInstructions(response.instructions || "");
      setInstructionsHasChanges(false);
    } catch (error) {
      console.error("Failed to load instructions:", error);
    }
  };

  const loadMostRecentConversation = async () => {
    try {
      const conversations = await getConversations();
      if (conversations && conversations.length > 0) {
        // Sort by updated_at descending and get the most recent
        const mostRecent = conversations.sort((a, b) =>
          new Date(b.updated_at).getTime() - new Date(a.updated_at).getTime()
        )[0];
        await chatHook.loadConversation(mostRecent.id);
        return true;
      }
      return false;
    } catch (error) {
      console.error("Failed to load conversations:", error);
      return false;
    }
  };

  const handleSaveInstructions = async () => {
    try {
      setIsSavingInstructions(true);
      await updateChatInstructions(instructions);
      setInstructionsHasChanges(false);
      showToast("success", "Instructions saved", "Your chat instructions have been updated.");
    } catch (error) {
      console.error("Failed to save instructions:", error);
      showToast("error", "Failed to save", "Could not save your instructions.");
    } finally {
      setIsSavingInstructions(false);
    }
  };

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

      // Send message to current session
      if (chatHook.currentConversation) {
        await chatHook.sendMessageToConversation(chatHook.currentConversation.id, message, referencedCards);
        showToast("success", "Message sent", "Your message has been sent to the AI.");
      }
    } else if (newParam === 'true') {
      // Clear URL params to avoid re-triggering
      window.history.replaceState({}, '', '/app/chat');

      // Check if there are messages to clear before showing confirmation
      if (chatHook.messages.length > 0) {
        setShowClearConfirm(true);
      } else {
        // No messages to lose, just clear
        await chatHook.clearChat();
      }
    }
  };

  const handleClearConfirmed = async () => {
    setShowClearConfirm(false);
    await chatHook.clearChat();
    showToast("success", "Chat cleared", "Started a new chat session.");
  };

  const handleModelChange = (model: string) => {
    chatHook.setSelectedModel(model);
    localStorage.setItem('chatSelectedModel', model);
    window.dispatchEvent(new CustomEvent('chatModelChange', { detail: model }));
    showToast("success", "Model updated", `Chat model changed to ${model}`);
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

  return (
    <div className="flex flex-col h-screen bg-gray-50">
      {/* Utility Bar */}
      <ChatUtilityBar
        hasLastCleared={!!chatHook.lastClearedSession}
        isSending={chatHook.isSending}
        onClear={chatHook.clearChat}
        onRestoreLast={chatHook.restoreLastCleared}
        onSettings={() => setShowSettingsDialog(true)}
        hasSubscription={hasSubscription}
      />

      {/* Chat Area - full width */}
      <div className="flex-1 flex flex-col overflow-hidden">
        {chatHook.currentConversation ? (
          <ChatInterface
            chatHook={chatHook}
            onCardClick={handleCardClick}
            onTaskClick={handleTaskClick}
            onRegenerateMessage={handleRegenerateMessage}
            placeholder="Ask about your cards... Type @ to mention a card, /clear to clear chat"
          />
        ) : (
          <div className="flex-1 flex items-center justify-center">
            <div className="text-center text-gray-600 max-w-lg mx-auto p-8 animate-fade-in">
              {/* Animated icon container */}
              <div className="w-20 h-20 mx-auto mb-8 rounded-2xl bg-blue-600 flex items-center justify-center shadow-lg">
                <svg className="w-10 h-10 text-white" fill="none" stroke="currentColor" viewBox="0 0 24 24">
                  <path strokeLinecap="round" strokeLinejoin="round" strokeWidth={2} d="M8 12h.01M12 12h.01M16 12h.01M21 12c0 4.418-4.03 8-9 8a9.863 9.863 0 01-4.255-.949L3 20l1.395-3.72C3.512 15.042 3 13.574 3 12c0-4.418 4.03-8 9-8s9 3.582 9 8z" />
                </svg>
              </div>
              <h3 className="text-2xl font-semibold text-gray-900 mb-3">Welcome to Chat</h3>
              <p className="text-gray-600 mb-8 leading-relaxed text-base">
                Ask questions, explore your knowledge base, and discover connections between your cards.
              </p>
              {!hasSubscription && (
                <div className="inline-flex flex-col items-center gap-2 p-5 bg-amber-50 border border-amber-200 rounded-2xl shadow-sm">
                  <div className="flex items-center gap-2 text-amber-800 font-medium">
                    <svg className="w-5 h-5" fill="none" stroke="currentColor" viewBox="0 0 24 24">
                      <path strokeLinecap="round" strokeLinejoin="round" strokeWidth={2} d="M9.663 17h4.673M12 3v1m6.364 1.636l-.707.707M21 12h-1M4 12H3m3.343-5.657l-.707-.707m2.828 9.9a5 5 0 117.072 0l-.548.547A3.374 3.374 0 0014 18.469V19a2 2 0 11-4 0v-.531c0-.895-.356-1.754-.988-2.386l-.548-.547z" />
                    </svg>
                    <span>AI Chat is a Pro feature</span>
                  </div>
                  <Link
                    to="/app/subscribe"
                    className="inline-flex items-center gap-2 px-5 py-2.5 bg-amber-500 hover:bg-amber-600 text-white font-medium rounded-xl transition-colors duration-200 shadow-md"
                  >
                    <svg className="w-4 h-4" fill="none" stroke="currentColor" viewBox="0 0 24 24">
                      <path strokeLinecap="round" strokeLinejoin="round" strokeWidth={2} d="M13 10V3L4 14h7v7l9-11h-7z" />
                    </svg>
                    Upgrade to Pro
                  </Link>
                </div>
              )}
            </div>
          </div>
        )}
      </div>

      {/* Unified Settings Dialog */}
      {showSettingsDialog && (
        <div className="fixed inset-0 bg-gray-900/60 flex items-center justify-center z-50 p-4 animate-dialog-fade-in">
          <div className="bg-white rounded-3xl shadow-2xl max-w-2xl w-full max-h-[85vh] flex flex-col overflow-hidden animate-dialog-slide-up">
            {/* Header */}
            <div className="p-6 border-b border-gray-100 bg-blue-50">
              <div className="flex items-center justify-between">
                <div>
                  <h2 className="text-xl font-semibold text-gray-900">Chat Settings</h2>
                  <p className="text-sm text-gray-600 mt-1">
                    Customize your AI chat experience
                  </p>
                </div>
                <button
                  onClick={() => {
                    if (instructionsHasChanges) {
                      const confirmClose = window.confirm("You have unsaved changes to your instructions. Are you sure you want to close?");
                      if (!confirmClose) return;
                    }
                    setShowSettingsDialog(false);
                  }}
                  className="text-gray-400 hover:text-gray-600 p-2.5 rounded-xl hover:bg-gray-100 transition-all duration-200 hover:scale-105"
                >
                  <svg className="w-5 h-5" fill="none" stroke="currentColor" viewBox="0 0 24 24">
                    <path strokeLinecap="round" strokeLinejoin="round" strokeWidth={2} d="M6 18L18 6M6 6l12 12" />
                  </svg>
                </button>
              </div>
            </div>

            {/* Content */}
            <div className="flex-1 p-6 overflow-y-auto space-y-6">
              {/* Model Section */}
              <div className="group">
                <div className="flex items-center gap-2 mb-3">
                  <div className="p-1.5 rounded-lg bg-blue-100 text-blue-600">
                    <svg className="w-4 h-4" fill="none" stroke="currentColor" viewBox="0 0 24 24">
                      <path strokeLinecap="round" strokeLinejoin="round" strokeWidth={2} d="M9.663 17h4.673M12 3v1m6.364 1.636l-.707.707M21 12h-1M4 12H3m3.343-5.657l-.707-.707m2.828 9.9a5 5 0 117.072 0l-.548.547A3.374 3.374 0 0014 18.469V19a2 2 0 11-4 0v-.531c0-.895-.356-1.754-.988-2.386l-.548-.547z" />
                    </svg>
                  </div>
                  <h3 className="text-sm font-semibold text-gray-900">AI Model</h3>
                </div>
                <div className="relative">
                  <select
                    value={chatHook.selectedModel}
                    onChange={(e) => handleModelChange(e.target.value)}
                    className="w-full px-4 py-3 border-2 border-gray-200 rounded-xl focus:ring-2 focus:ring-blue-500/20 focus:border-blue-500 bg-white transition-all duration-200 appearance-none cursor-pointer hover:border-gray-300"
                  >
                    {availableModels.map((model) => (
                      <option key={model.value} value={model.value}>
                        {model.label}
                      </option>
                    ))}
                  </select>
                  <div className="absolute right-3 top-1/2 -translate-y-1/2 pointer-events-none text-gray-500">
                    <svg className="w-4 h-4" fill="none" stroke="currentColor" viewBox="0 0 24 24">
                      <path strokeLinecap="round" strokeLinejoin="round" strokeWidth={2} d="M19 9l-7 7-7-7" />
                    </svg>
                  </div>
                </div>
                <p className="text-xs text-gray-500 mt-2.5 flex items-center gap-1.5">
                  <svg className="w-3.5 h-3.5 text-gray-400" fill="none" stroke="currentColor" viewBox="0 0 24 24">
                    <path strokeLinecap="round" strokeLinejoin="round" strokeWidth={2} d="M13 16h-1v-4h-1m1-4h.01M21 12a9 9 0 11-18 0 9 9 0 0118 0z" />
                  </svg>
                  This model will be used for all new chat conversations
                </p>
              </div>

              {/* Instructions Section */}
              <div className="group">
                <div className="flex items-center justify-between mb-3">
                  <div className="flex items-center gap-2">
                    <div className="p-1.5 rounded-lg bg-indigo-100 text-indigo-600">
                      <svg className="w-4 h-4" fill="none" stroke="currentColor" viewBox="0 0 24 24">
                        <path strokeLinecap="round" strokeLinejoin="round" strokeWidth={2} d="M9 12h6m-6 4h6m2 5H7a2 2 0 01-2-2V5a2 2 0 012-2h5.586a1 1 0 01.707.293l5.414 5.414a1 1 0 01.293.707V19a2 2 0 01-2 2z" />
                      </svg>
                    </div>
                    <h3 className="text-sm font-semibold text-gray-900">System Instructions</h3>
                  </div>
                  {instructionsHasChanges && (
                    <span className="inline-flex items-center gap-1.5 px-2.5 py-1 text-xs font-medium text-amber-700 bg-amber-50 border border-amber-200 rounded-full animate-pulse-soft">
                      <div className="w-1.5 h-1.5 bg-amber-500 rounded-full"></div>
                      Unsaved changes
                    </span>
                  )}
                </div>
                <div className="relative">
                  <textarea
                    value={instructions}
                    onChange={(e) => {
                      setInstructions(e.target.value);
                      setInstructionsHasChanges(true);
                    }}
                    placeholder="Enter custom instructions for the AI. These will be included with every chat message."
                    className="w-full min-h-[140px] max-h-[40vh] sm:h-48 sm:max-h-none px-4 py-3 border-2 border-gray-200 rounded-xl focus:ring-2 focus:ring-blue-500/20 focus:border-blue-500 resize-y text-sm bg-white transition-all duration-200 hover:border-gray-300 font-mono leading-relaxed"
                    maxLength={10000}
                  />
                  {/* Character count badge */}
                  <div className={`absolute bottom-3 right-3 px-2 py-1 rounded-lg text-xs font-medium transition-colors ${
                    instructions.length > 8000
                      ? 'bg-red-100 text-red-700'
                      : instructions.length > 6000
                      ? 'bg-amber-100 text-amber-700'
                      : 'bg-gray-100 text-gray-500'
                  }`}>
                    {instructions.length.toLocaleString()} / 10,000
                  </div>
                </div>
                <div className="mt-2.5 flex items-center gap-2 text-xs text-gray-500">
                  <svg className="w-3.5 h-3.5 text-gray-400" fill="none" stroke="currentColor" viewBox="0 0 24 24">
                    <path strokeLinecap="round" strokeLinejoin="round" strokeWidth={2} d="M13 16h-1v-4h-1m1-4h.01M21 12a9 9 0 11-18 0 9 9 0 0118 0z" />
                  </svg>
                  Custom instructions help the AI understand your preferences and context
                </div>
              </div>
            </div>

            {/* Footer */}
            <div className="p-6 border-t border-gray-100 bg-gray-50/50">
              <div className="flex justify-between items-center">
                <button
                  onClick={() => {
                    if (instructionsHasChanges) {
                      const confirmClose = window.confirm("You have unsaved changes to your instructions. Are you sure you want to close?");
                      if (!confirmClose) return;
                    }
                    setShowSettingsDialog(false);
                  }}
                  className="px-5 py-2.5 text-sm font-medium text-gray-600 hover:text-gray-800 hover:bg-gray-200/50 rounded-xl transition-all duration-200"
                >
                  Cancel
                </button>
                <div className="flex gap-3">
                  {instructionsHasChanges && (
                    <button
                      onClick={() => {
                        loadInstructions();
                        showToast("info", "Reset", "Instructions reset to saved value.");
                      }}
                      className="px-4 py-2.5 text-sm font-medium text-gray-600 hover:text-gray-800 border-2 border-gray-300 hover:border-gray-400 rounded-xl transition-all duration-200"
                    >
                      Reset
                    </button>
                  )}
                  <button
                    onClick={handleSaveInstructions}
                    disabled={isSavingInstructions || !instructionsHasChanges}
                    className="px-6 py-2.5 bg-blue-600 hover:bg-blue-700 text-white rounded-xl font-medium transition-colors duration-200 disabled:opacity-50 disabled:cursor-not-allowed flex items-center gap-2 shadow-lg"
                  >
                    {isSavingInstructions ? (
                      <>
                        <div className="w-4 h-4 border-2 border-white border-t-transparent rounded-full animate-spin"></div>
                        <span>Saving...</span>
                      </>
                    ) : (
                      <>
                        <svg className="w-4 h-4" fill="none" stroke="currentColor" viewBox="0 0 24 24">
                          <path strokeLinecap="round" strokeLinejoin="round" strokeWidth={2} d="M5 13l4 4L19 7" />
                        </svg>
                        <span>Save Changes</span>
                      </>
                    )}
                  </button>
                </div>
              </div>
            </div>
          </div>
        </div>
      )}

      {/* Clear Chat Confirmation Dialog */}
      {showClearConfirm && (
        <div className="fixed inset-0 bg-gray-900/60 flex items-center justify-center z-50 p-4 animate-dialog-fade-in">
          <div className="bg-white rounded-3xl shadow-2xl max-w-md w-full overflow-hidden animate-dialog-slide-up">
            {/* Header */}
            <div className="p-6 bg-amber-50 border-b border-amber-100">
              <div className="flex items-center gap-4">
                <div className="flex-shrink-0 w-12 h-12 bg-amber-500 rounded-2xl flex items-center justify-center shadow-lg">
                  <svg className="w-6 h-6 text-white" fill="none" stroke="currentColor" viewBox="0 0 24 24">
                    <path strokeLinecap="round" strokeLinejoin="round" strokeWidth={2.5} d="M12 9v2m0 4h.01m-6.938 4h13.856c1.54 0 2.502-1.667 1.732-3L13.732 4c-.77-1.333-2.694-1.333-3.464 0L3.34 16c-.77 1.333.192 3 1.732 3z" />
                  </svg>
                </div>
                <div>
                  <h3 className="text-lg font-semibold text-gray-900">Clear Chat?</h3>
                  <p className="text-sm text-gray-600 mt-0.5">This action can be undone</p>
                </div>
              </div>
            </div>

            <div className="p-6">
              <p className="text-gray-600 mb-5 leading-relaxed">
                This will clear your current chat session and start a new one. You can restore it later using the "Restore Last" button.
              </p>

              {/* Message count card */}
              <div className="bg-blue-50 border border-blue-200 rounded-2xl p-4 mb-6">
                <div className="flex items-center justify-between">
                  <div>
                    <p className="text-xs font-medium text-blue-700 uppercase tracking-wide">Current Session</p>
                    <p className="text-2xl font-bold text-blue-900 mt-1">
                      {chatHook.messages.length}
                      <span className="text-lg font-normal text-blue-700 ml-1">
                        message{chatHook.messages.length !== 1 ? 's' : ''}
                      </span>
                    </p>
                  </div>
                  <div className="w-12 h-12 bg-blue-100 rounded-xl flex items-center justify-center">
                    <svg className="w-6 h-6 text-blue-600" fill="none" stroke="currentColor" viewBox="0 0 24 24">
                      <path strokeLinecap="round" strokeLinejoin="round" strokeWidth={2} d="M8 12h.01M12 12h.01M16 12h.01M21 12c0 4.418-4.03 8-9 8a9.863 9.863 0 01-4.255-.949L3 20l1.395-3.72C3.512 15.042 3 13.574 3 12c0-4.418 4.03-8 9-8s9 3.582 9 8z" />
                    </svg>
                  </div>
                </div>
                {chatHook.messages.length > 0 && (
                  <div className="mt-3 pt-3 border-t border-blue-200/50 flex items-center gap-2 text-xs text-blue-600">
                    <svg className="w-3.5 h-3.5" fill="none" stroke="currentColor" viewBox="0 0 24 24">
                      <path strokeLinecap="round" strokeLinejoin="round" strokeWidth={2} d="M5 13l4 4L19 7" />
                    </svg>
                    This will be archived and can be restored later
                  </div>
                )}
              </div>

              <div className="flex gap-3 justify-end">
                <button
                  onClick={() => setShowClearConfirm(false)}
                  className="px-5 py-2.5 text-sm font-medium text-gray-700 bg-white border-2 border-gray-300 hover:border-gray-400 rounded-xl transition-all duration-200 hover:bg-gray-50"
                >
                  Cancel
                </button>
                <button
                  onClick={handleClearConfirmed}
                  className="px-5 py-2.5 text-sm font-medium text-white bg-amber-500 hover:bg-amber-600 rounded-xl transition-colors duration-200 shadow-lg"
                >
                  Clear Chat
                </button>
              </div>
            </div>
          </div>
        </div>
      )}

      {/* Task Dialog */}
      <TaskDialog
        taskId={selectedTaskId}
        isOpen={showTaskDialog}
        onClose={handleTaskDialogClose}
      />
    </div>
  );
}