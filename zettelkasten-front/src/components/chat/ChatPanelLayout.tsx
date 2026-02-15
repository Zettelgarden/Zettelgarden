import React, { useState } from "react";
import { ChatInterface } from "./ChatInterface";
import { ChatUtilityBar } from "./ChatUtilityBar";
import { useIsDesktop } from "../../hooks/useWindowSize";
import { useUIState } from "../../contexts/UIStateContext";
import { useAuth } from "../../contexts/AuthContext";
import { TaskDialog } from "../tasks/TaskDialog";
import { useChat } from "../../hooks/useChat";
import { getChatModels, ChatModel, getChatInstructions, updateChatInstructions } from "../../api/chat";
import { useToast } from "../toast/ToastContext";

interface ChatPanelLayoutProps {
  children: React.ReactNode;
}

export const ChatPanelLayout: React.FC<ChatPanelLayoutProps> = ({
  children
}) => {
  const [isExpanded, setIsExpanded] = useState(false);
  const isDesktop = useIsDesktop(1024);
  const { setIsChatOpen } = useUIState();
  const { hasSubscription } = useAuth();
  const { showToast } = useToast();

  // Chat state
  const [selectedTaskId, setSelectedTaskId] = useState<number | null>(null);
  const [showTaskDialog, setShowTaskDialog] = useState(false);
  const [regeneratingMessageIds, setRegeneratingMessageIds] = useState<Set<string>>(new Set());
  const [showClearConfirm, setShowClearConfirm] = useState(false);
  const [showSettingsDialog, setShowSettingsDialog] = useState(false);
  const [availableModels, setAvailableModels] = useState<ChatModel[]>([]);
  const [instructions, setInstructions] = useState("");
  const [instructionsHasChanges, setInstructionsHasChanges] = useState(false);
  const [isSavingInstructions, setIsSavingInstructions] = useState(false);

  // Use the shared chat hook
  const chatHook = useChat({
    initialModel: localStorage.getItem('chatSelectedModel') || "google/gemini-2.5-flash",
    enableStreaming: true,
  });

  // Initialize chat on mount
  React.useEffect(() => {
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
      const { getConversations } = await import("../../api/chat");
      const conversations = await getConversations();
      if (conversations && conversations.length > 0) {
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
      const { regenerateMessage } = await import("../../api/chat");
      await regenerateMessage(chatHook.currentConversation.id, messageId);
      await chatHook.refreshMessages(chatHook.currentConversation.id);
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
    <div className="flex flex-col lg:flex-row h-full">
      {/* Main Content Pane - Left side on desktop, top on mobile */}
      <div className={`
        w-full lg:w-1/2
        border-b lg:border-b-0 lg:border-r border-gray-200
        overflow-y-auto
        ${isExpanded ? 'h-1/3 md:h-1/2 lg:h-full' : 'flex-1 lg:h-full'}
        transition-all duration-300 ease-in-out
      `}>
        <div className="h-full">
          {children}
        </div>
      </div>

      {/* Chat Panel Pane - Right side on desktop, collapsible bottom on mobile */}
      <div className={`
        w-full lg:w-1/2
        ${isExpanded ? 'h-2/3 md:h-1/2' : 'h-auto lg:h-full'}
        transition-all duration-300 ease-in-out
      `}>
        <div className="h-full bg-green-50 flex flex-col">
          {/* Desktop Header */}
          <div className="hidden lg:flex bg-green-100 px-3 py-2 border-b border-green-200 items-center justify-between">
            <div className="flex items-center gap-2">
              <svg className="h-4 w-4 text-green-600" fill="none" stroke="currentColor" viewBox="0 0 24 24">
                <path strokeLinecap="round" strokeLinejoin="round" strokeWidth={2} d="M8 12h.01M12 12h.01M16 12h.01M21 12c0 4.418-4.03 8-9 8a9.863 9.863 0 01-4.255-.949L3 20l1.395-3.72C3.512 15.042 3 13.574 3 12c0-4.418 4.03-8 9-8s9 3.582 9 8z" />
              </svg>
              <span className="text-xs font-semibold uppercase tracking-wide text-green-700">
                Chat Panel
              </span>
            </div>
            <button
              onClick={() => setIsChatOpen(false)}
              className="text-green-600 hover:text-green-800 hover:bg-green-200 px-2 py-1 rounded text-sm flex items-center gap-1"
              title="Close chat"
            >
              <svg className="w-4 h-4" fill="none" stroke="currentColor" viewBox="0 0 24 24">
                <path strokeLinecap="round" strokeLinejoin="round" strokeWidth={2} d="M6 18L18 6M6 6l12 12" />
              </svg>
              Close
            </button>
          </div>

          {/* Mobile collapse/expand button */}
          <div className="lg:hidden bg-green-100 p-2 border-b border-green-200">
            <button
              onClick={() => setIsExpanded(!isExpanded)}
              className="flex items-center justify-between w-full text-green-700"
            >
              <div className="flex items-center gap-2">
                <svg className="h-3 w-3 text-green-600" fill="none" stroke="currentColor" viewBox="0 0 24 24">
                  <path strokeLinecap="round" strokeLinejoin="round" strokeWidth={2} d="M8 12h.01M12 12h.01M16 12h.01M21 12c0 4.418-4.03 8-9 8a9.863 9.863 0 01-4.255-.949L3 20l1.395-3.72C3.512 15.042 3 13.574 3 12c0-4.418 4.03-8 9-8s9 3.582 9 8z" />
                </svg>
                <span className="text-xs font-medium uppercase tracking-wide">
                  Chat Panel
                </span>
              </div>
              <div className="flex items-center gap-2">
                <button
                  onClick={(e) => { e.stopPropagation(); setIsChatOpen(false); }}
                  className="text-green-600 hover:text-green-800 p-1"
                  title="Close chat"
                >
                  <svg className="w-4 h-4" fill="none" stroke="currentColor" viewBox="0 0 24 24">
                    <path strokeLinecap="round" strokeLinejoin="round" strokeWidth={2} d="M6 18L18 6M6 6l12 12" />
                  </svg>
                </button>
                <svg
                  className={`w-4 h-4 transition-transform ${isExpanded ? 'rotate-180' : ''}`}
                  fill="none"
                  stroke="currentColor"
                  viewBox="0 0 24 24"
                >
                  <path strokeLinecap="round" strokeLinejoin="round" strokeWidth={2} d="M19 9l-7 7-7-7" />
                </svg>
              </div>
            </button>
          </div>

          {/* Chat content - conditionally shown on mobile */}
          {(isExpanded || isDesktop) && (
            <div className="flex-1 flex flex-col min-h-0 overflow-hidden">
              <ChatUtilityBar
                hasLastCleared={!!chatHook.lastClearedSession}
                isSending={chatHook.isSending}
                onClear={chatHook.clearChat}
                onRestoreLast={chatHook.restoreLastCleared}
                onSettings={() => setShowSettingsDialog(true)}
                hasSubscription={hasSubscription}
              />
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
                  <div className="text-center text-gray-600 max-w-lg mx-auto p-8">
                    <h3 className="text-xl font-semibold text-gray-900 mb-2">Welcome to Chat</h3>
                    <p className="text-gray-600">
                      Ask questions, explore your knowledge base, and discover connections.
                    </p>
                  </div>
                </div>
              )}
            </div>
          )}
        </div>
      </div>

      {/* Settings Dialog */}
      {showSettingsDialog && (
        <div className="fixed inset-0 bg-gray-900/60 flex items-center justify-center z-50 p-4">
          <div className="bg-white rounded-3xl shadow-2xl max-w-2xl w-full max-h-[85vh] flex flex-col overflow-hidden">
            <div className="p-6 border-b border-gray-100 bg-green-50">
              <div className="flex items-center justify-between">
                <div>
                  <h2 className="text-xl font-semibold text-gray-900">Chat Settings</h2>
                  <p className="text-sm text-gray-600 mt-1">Customize your AI chat experience</p>
                </div>
                <button
                  onClick={() => {
                    if (instructionsHasChanges) {
                      const confirmClose = window.confirm("You have unsaved changes to your instructions. Are you sure you want to close?");
                      if (!confirmClose) return;
                    }
                    setShowSettingsDialog(false);
                  }}
                  className="text-gray-400 hover:text-gray-600 p-2.5 rounded-xl hover:bg-gray-100"
                >
                  <svg className="w-5 h-5" fill="none" stroke="currentColor" viewBox="0 0 24 24">
                    <path strokeLinecap="round" strokeLinejoin="round" strokeWidth={2} d="M6 18L18 6M6 6l12 12" />
                  </svg>
                </button>
              </div>
            </div>

            <div className="flex-1 p-6 overflow-y-auto space-y-6">
              {/* Model Section */}
              <div>
                <div className="flex items-center gap-2 mb-3">
                  <div className="p-1.5 rounded-lg bg-green-100 text-green-600">
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
                    className="w-full px-4 py-3 border-2 border-gray-200 rounded-xl focus:ring-2 focus:ring-green-500/20 focus:border-green-500 bg-white"
                  >
                    {availableModels.map((model) => (
                      <option key={model.value} value={model.value}>{model.label}</option>
                    ))}
                  </select>
                  <div className="absolute right-3 top-1/2 -translate-y-1/2 pointer-events-none text-gray-500">
                    <svg className="w-4 h-4" fill="none" stroke="currentColor" viewBox="0 0 24 24">
                      <path strokeLinecap="round" strokeLinejoin="round" strokeWidth={2} d="M19 9l-7 7-7-7" />
                    </svg>
                  </div>
                </div>
              </div>

              {/* Instructions Section */}
              <div>
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
                    <span className="text-xs font-medium text-amber-700 bg-amber-50 border border-amber-200 rounded-full px-2 py-1">
                      Unsaved
                    </span>
                  )}
                </div>
                <textarea
                  value={instructions}
                  onChange={(e) => {
                    setInstructions(e.target.value);
                    setInstructionsHasChanges(true);
                  }}
                  placeholder="Enter custom instructions for the AI."
                  className="w-full min-h-[140px] px-4 py-3 border-2 border-gray-200 rounded-xl focus:ring-2 focus:ring-green-500/20 focus:border-green-500 resize-y text-sm bg-white"
                  maxLength={10000}
                />
              </div>
            </div>

            <div className="p-6 border-t border-gray-100 bg-gray-50/50 flex justify-between">
              <button
                onClick={() => {
                  if (instructionsHasChanges) {
                    const confirmClose = window.confirm("You have unsaved changes. Are you sure?");
                    if (!confirmClose) return;
                  }
                  setShowSettingsDialog(false);
                }}
                className="px-5 py-2.5 text-sm font-medium text-gray-600 hover:text-gray-800 hover:bg-gray-200/50 rounded-xl"
              >
                Cancel
              </button>
              <div className="flex gap-3">
                {instructionsHasChanges && (
                  <button
                    onClick={() => {
                      loadInstructions();
                      showToast("info", "Reset", "Instructions reset.");
                    }}
                    className="px-4 py-2.5 text-sm font-medium text-gray-600 hover:text-gray-800 border-2 border-gray-300 hover:border-gray-400 rounded-xl"
                  >
                    Reset
                  </button>
                )}
                <button
                  onClick={handleSaveInstructions}
                  disabled={isSavingInstructions || !instructionsHasChanges}
                  className="px-6 py-2.5 bg-green-600 hover:bg-green-700 text-white rounded-xl font-medium disabled:opacity-50 disabled:cursor-not-allowed flex items-center gap-2"
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
                      <span>Save</span>
                    </>
                  )}
                </button>
              </div>
            </div>
          </div>
        </div>
      )}

      {/* Clear Chat Confirmation */}
      {showClearConfirm && (
        <div className="fixed inset-0 bg-gray-900/60 flex items-center justify-center z-50 p-4">
          <div className="bg-white rounded-3xl shadow-2xl max-w-md w-full overflow-hidden">
            <div className="p-6 bg-amber-50 border-b border-amber-100">
              <div className="flex items-center gap-4">
                <div className="flex-shrink-0 w-12 h-12 bg-amber-500 rounded-2xl flex items-center justify-center">
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
              <p className="text-gray-600 mb-5">
                This will clear your current chat session and start a new one. You can restore it later using "Restore Last".
              </p>

              <div className="flex gap-3 justify-end">
                <button
                  onClick={() => setShowClearConfirm(false)}
                  className="px-5 py-2.5 text-sm font-medium text-gray-700 bg-white border-2 border-gray-300 hover:border-gray-400 rounded-xl"
                >
                  Cancel
                </button>
                <button
                  onClick={handleClearConfirmed}
                  className="px-5 py-2.5 text-sm font-medium text-white bg-amber-500 hover:bg-amber-600 rounded-xl"
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
};
