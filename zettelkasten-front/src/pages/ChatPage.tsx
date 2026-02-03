import React, { useState, useEffect } from "react";
import {
  regenerateMessage as apiRegenerateMessage,
  getChatModels,
  ChatModel,
  getChatInstructions,
  updateChatInstructions,
} from "../api/chat";
import { setDocumentTitle } from "../utils/title";
import { ChatInterface } from "../components/chat/ChatInterface";
import { ChatUtilityBar } from "../components/chat/ChatUtilityBar";
import { TaskDialog } from "../components/tasks/TaskDialog";
import { Link } from "react-router-dom";
import { useAuth } from "../contexts/AuthContext";
import { useChat } from "../hooks/useChat";
import { useToast } from "../components/toast/ToastContext";

interface ChatPageProps { }

export function ChatPage({ }: ChatPageProps) {
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
    initializeChatSession();
    handleUrlParams();
    loadAvailableModels();
    loadInstructions();
  }, []);

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

  const initializeChatSession = async () => {
    // Always start with a fresh session or load existing
    if (!chatHook.currentConversation) {
      await chatHook.createNewConversation("", chatHook.selectedModel);
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
    <div className="flex flex-col h-screen bg-white">
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
          <div className="flex-1 flex items-center justify-center bg-white">
            <div className="text-center text-gray-500 max-w-md mx-auto p-8">
              <div className="w-16 h-16 mx-auto mb-6 rounded-lg bg-gray-100 flex items-center justify-center">
                <svg className="w-8 h-8 text-gray-600" fill="none" stroke="currentColor" viewBox="0 0 24 24">
                  <path strokeLinecap="round" strokeLinejoin="round" strokeWidth={2} d="M8 12h.01M12 12h.01M16 12h.01M21 12c0 4.418-4.03 8-9 8a9.863 9.863 0 01-4.255-.949L3 20l1.395-3.72C3.512 15.042 3 13.574 3 12c0-4.418 4.03-8 9-8s9 3.582 9 8z" />
                </svg>
              </div>
              <h3 className="text-xl font-semibold text-gray-900 mb-3">Welcome to Chat</h3>
              <p className="text-gray-600 mb-6 leading-relaxed">Start typing to chat with your knowledge base.</p>
              {!hasSubscription && (
                <div className="text-center text-gray-500 mb-6 p-4 bg-gray-50 rounded-lg">
                  AI Agents are a Pro feature.
                  <br />
                  <Link to="/app/subscribe" className="text-blue-500 hover:underline">
                    Upgrade to Pro to unlock intelligent AI agents that can work with your knowledge base.
                  </Link>
                </div>
              )}
            </div>
          </div>
        )}
      </div>

      {/* Unified Settings Dialog */}
      {showSettingsDialog && (
        <div className="fixed inset-0 bg-black bg-opacity-50 flex items-center justify-center z-50 p-4">
          <div className="bg-white rounded-2xl shadow-xl max-w-2xl w-full max-h-[80vh] flex flex-col">
            {/* Header */}
            <div className="p-6 border-b border-gray-200">
              <div className="flex items-center justify-between">
                <div>
                  <h2 className="text-xl font-semibold text-gray-900">Chat Settings</h2>
                  <p className="text-sm text-gray-600 mt-1">
                    Customize your chat experience
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
                  className="text-gray-400 hover:text-gray-600 p-2 rounded-lg hover:bg-gray-100 transition-colors"
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
              <div>
                <h3 className="text-sm font-semibold text-gray-900 mb-3">Model</h3>
                <select
                  value={chatHook.selectedModel}
                  onChange={(e) => handleModelChange(e.target.value)}
                  className="w-full px-3 py-2 border border-gray-300 rounded-lg focus:ring-2 focus:ring-blue-500 focus:border-blue-500"
                >
                  {availableModels.map((model) => (
                    <option key={model.value} value={model.value}>
                      {model.label}
                    </option>
                  ))}
                </select>
                <p className="text-xs text-gray-500 mt-2">
                  This model will be used for all new chat conversations.
                </p>
              </div>

              {/* Instructions Section */}
              <div>
                <div className="flex items-center justify-between mb-3">
                  <h3 className="text-sm font-semibold text-gray-900">Instructions</h3>
                  {instructionsHasChanges && (
                    <span className="text-xs text-amber-600 flex items-center gap-1">
                      <div className="w-2 h-2 bg-amber-500 rounded-full"></div>
                      Unsaved changes
                    </span>
                  )}
                </div>
                <textarea
                  value={instructions}
                  onChange={(e) => {
                    setInstructions(e.target.value);
                    setInstructionsHasChanges(true);
                  }}
                  placeholder="Enter custom instructions for the AI. These will be included with every chat message."
                  className="w-full h-48 px-3 py-2 border border-gray-300 rounded-lg focus:outline-none focus:ring-2 focus:ring-blue-500 focus:border-transparent resize-none text-sm"
                  style={{ fontFamily: 'monospace' }}
                  maxLength={10000}
                />
                <div className="mt-2 flex justify-between items-center">
                  <span className={`text-xs ${instructions.length > 8000 ? 'text-red-600 font-medium' : instructions.length > 6000 ? 'text-amber-600' : 'text-gray-500'}`}>
                    {instructions.length}/10000 characters
                  </span>
                </div>
              </div>
            </div>

            {/* Footer */}
            <div className="p-6 border-t border-gray-200">
              <div className="flex justify-between items-center">
                <button
                  onClick={() => {
                    if (instructionsHasChanges) {
                      const confirmClose = window.confirm("You have unsaved changes to your instructions. Are you sure you want to close?");
                      if (!confirmClose) return;
                    }
                    setShowSettingsDialog(false);
                  }}
                  className="px-4 py-2 text-sm text-gray-600 hover:text-gray-800 transition-colors"
                >
                  Close
                </button>
                <div className="flex gap-3">
                  {instructionsHasChanges && (
                    <button
                      onClick={() => {
                        loadInstructions();
                        showToast("info", "Reset", "Instructions reset to saved value.");
                      }}
                      className="px-4 py-2 text-sm text-gray-600 hover:text-gray-800 border border-gray-300 rounded-lg transition-colors"
                    >
                      Reset
                    </button>
                  )}
                  <button
                    onClick={handleSaveInstructions}
                    disabled={isSavingInstructions || !instructionsHasChanges}
                    className="px-6 py-2 bg-blue-500 hover:bg-blue-600 text-white rounded-lg font-medium transition-colors disabled:opacity-50 disabled:cursor-not-allowed flex items-center gap-2"
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
                        <span>Save Instructions</span>
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
        <div className="fixed inset-0 bg-black bg-opacity-50 flex items-center justify-center z-50 p-4">
          <div className="bg-white rounded-2xl shadow-xl max-w-md w-full p-6">
            <div className="flex items-center gap-3 mb-4">
              <div className="flex-shrink-0 w-10 h-10 bg-yellow-100 rounded-full flex items-center justify-center">
                <svg className="w-5 h-5 text-yellow-600" fill="none" stroke="currentColor" viewBox="0 0 24 24">
                  <path strokeLinecap="round" strokeLinejoin="round" strokeWidth={2} d="M12 9v2m0 4h.01m-6.938 4h13.856c1.54 0 2.502-1.667 1.732-3L13.732 4c-.77-1.333-2.694-1.333-3.464 0L3.34 16c-.77 1.333.192 3 1.732 3z" />
                </svg>
              </div>
              <h3 className="text-lg font-semibold text-gray-900">Clear Chat?</h3>
            </div>
            <p className="text-gray-600 mb-6">
              This will clear your current chat session and start a new one. You can restore it later using the "Restore Last" button.
            </p>
            <div className="bg-gray-50 rounded-lg p-4 mb-6">
              <p className="text-sm text-gray-500">
                Current session has <strong>{chatHook.messages.length} message{chatHook.messages.length !== 1 ? 's' : ''}</strong>.
                {chatHook.messages.length > 0 && (
                  <span className="block mt-2 text-xs text-gray-400">
                    This will be archived and can be restored later.
                  </span>
                )}
              </p>
            </div>
            <div className="flex gap-3 justify-end">
              <button
                onClick={() => setShowClearConfirm(false)}
                className="px-4 py-2 text-sm font-medium text-gray-700 bg-white border border-gray-300 rounded-lg hover:bg-gray-50 focus:outline-none focus:ring-2 focus:ring-blue-500"
              >
                Cancel
              </button>
              <button
                onClick={handleClearConfirmed}
                className="px-4 py-2 text-sm font-medium text-white bg-yellow-600 hover:bg-yellow-700 rounded-lg focus:outline-none focus:ring-2 focus:ring-yellow-500"
              >
                Clear Chat
              </button>
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