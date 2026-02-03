import React, { useState, useEffect } from "react";
import {
  regenerateMessage as apiRegenerateMessage,
} from "../api/chat";
import { setDocumentTitle } from "../utils/title";
import { ChatInterface } from "../components/chat/ChatInterface";
import { ChatUtilityBar } from "../components/chat/ChatUtilityBar";
import { InstructionsMenu } from "../components/chat/InstructionsMenu";
import { TaskDialog } from "../components/tasks/TaskDialog";
import { Link } from "react-router-dom";
import { useAuth } from "../contexts/AuthContext";
import { useChat } from "../hooks/useChat";
import { useToast } from "../components/toast/ToastContext";

interface ChatPageProps { }

export function ChatPage({ }: ChatPageProps) {
  // ChatPage-specific state
  const [showInstructionsMenu, setShowInstructionsMenu] = useState(false);
  const [selectedTaskId, setSelectedTaskId] = useState<number | null>(null);
  const [showTaskDialog, setShowTaskDialog] = useState(false);
  const [regeneratingMessageIds, setRegeneratingMessageIds] = useState<Set<string>>(new Set());

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
  }, []);

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
      // For rolling session, just clear existing chat
      await chatHook.clearChat();
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

  return (
    <div className="flex flex-col h-screen bg-white">
      {/* Utility Bar */}
      <ChatUtilityBar
        hasLastCleared={!!chatHook.lastClearedSession}
        isSending={chatHook.isSending}
        onClear={chatHook.clearChat}
        onRestoreLast={chatHook.restoreLastCleared}
        onInstructions={() => setShowInstructionsMenu(true)}
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