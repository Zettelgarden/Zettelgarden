import React, { useEffect, useRef, useState, useCallback } from "react";
import Markdown from "react-markdown";
import remarkGfm from "remark-gfm";
import rehypeSanitize from "rehype-sanitize";
import { ChatMessage } from "../../api/chat";
import { parseMessageContent } from "../../utils/chatUtils";
import { CardsSection } from "./CardsSection";
import { TasksSection } from "./TasksSection";
import { ChatInput } from "./ChatInput";
import { ToolResultCard } from "./ToolResultCard";
import { EditMessageDialog } from "./EditMessageDialog";
import { BacklinkDialog } from "../cards/BacklinkDialog";
import { useChat } from "../../hooks/useChat";
import { editUserMessage } from "../../api/chat";
import { getCard } from "../../api/cards";
import { PartialCard } from "../../models/Card";

// Relative time component with auto-update and absolute time on hover
const RelativeTime = ({ timestamp }: { timestamp: string }) => {
  const [relativeTime, setRelativeTime] = useState("");
  const [absoluteTime, setAbsoluteTime] = useState("");

  const updateRelativeTime = useCallback(() => {
    const now = new Date();
    const then = new Date(timestamp);
    const diffMs = now.getTime() - then.getTime();
    const diffSecs = Math.floor(diffMs / 1000);
    const diffMins = Math.floor(diffSecs / 60);
    const diffHours = Math.floor(diffMins / 60);
    const diffDays = Math.floor(diffHours / 24);

    if (diffSecs < 60) {
      setRelativeTime("just now");
    } else if (diffMins < 60) {
      setRelativeTime(`${diffMins}m ago`);
    } else if (diffHours < 24) {
      setRelativeTime(`${diffHours}h ago`);
    } else if (diffDays < 7) {
      setRelativeTime(`${diffDays}d ago`);
    } else {
      // For older messages, show the date
      setRelativeTime(then.toLocaleDateString());
    }
  }, [timestamp]);

  useEffect(() => {
    // Calculate absolute time once
    const then = new Date(timestamp);
    setAbsoluteTime(then.toLocaleString());

    // Initial update
    updateRelativeTime();

    // Update every minute for messages less than an hour old
    const interval = setInterval(updateRelativeTime, 60000);

    return () => clearInterval(interval);
  }, [timestamp, updateRelativeTime]);

  return (
    <span title={absoluteTime} className="cursor-help">
      {relativeTime}
    </span>
  );
};

// Streaming cursor component with smooth animation
const StreamingCursor = () => (
  <span className="inline-flex items-center ml-1">
    <span className="flex space-x-0.5">
      <span className="w-1 h-1 bg-blue-500 rounded-full animate-[ping_1s_ease-in-out_infinite]"></span>
      <span className="w-1 h-1 bg-blue-500 rounded-full animate-[ping_1s_ease-in-out_0.2s_infinite]"></span>
      <span className="w-1 h-1 bg-blue-500 rounded-full animate-[ping_1s_ease-in-out_0.4s_infinite]"></span>
    </span>
  </span>
);

// Skeleton loader for message content (removed - too busy)
// const MessageSkeleton = () => (
//   <div className="space-y-3 animate-pulse">
//     <div className="h-4 bg-gray-200 rounded w-3/4"></div>
//     <div className="h-4 bg-gray-200 rounded w-1/2"></div>
//     <div className="h-4 bg-gray-200 rounded w-5/6"></div>
//   </div>
// );

// Tool call loading indicator
const ToolCallLoading = () => (
  <div className="flex items-center gap-2 px-3 py-2 bg-amber-50 border border-amber-200 rounded-lg">
    <div className="flex space-x-1">
      <div className="w-2 h-2 bg-amber-500 rounded-full animate-bounce"></div>
      <div className="w-2 h-2 bg-amber-500 rounded-full animate-bounce" style={{ animationDelay: '0.1s' }}></div>
      <div className="w-2 h-2 bg-amber-500 rounded-full animate-bounce" style={{ animationDelay: '0.2s' }}></div>
    </div>
    <span className="text-sm text-amber-700 font-medium">Running tools...</span>
  </div>
);

interface ChatInterfaceProps {
  chatHook: ReturnType<typeof useChat>;
  onCardClick: (cardPk: string) => void;
  onTaskClick: (taskId: number) => void;
  placeholder?: string;
  compact?: boolean;
  onRegenerateMessage?: (messageId: string) => void;
  onRetryToolCall?: (messageId: string, conversationId: string) => void;
}

export function ChatInterface({
  chatHook,
  onCardClick,
  onTaskClick,
  placeholder = "Ask about your cards... Type @ to mention a card",
  compact = false,
  onRegenerateMessage,
  onRetryToolCall,
}: ChatInterfaceProps) {
  const {
    messages,
    messageInput,
    isSending,
    error,
    selectedModel,
    collapsedToolResults,
    retryingToolIds,
    failedMessage,
    currentConversation,
    streamingMessageId,
    activeToolCalls,
    setMessageInput,
    sendMessage,
    handleCardReference,
    toggleToolResult,
    retryFailedMessage,
    retryTool,
    referencedCards,
    setReferencedCards,
    clearChat,
  } = chatHook;

  const messagesEndRef = useRef<HTMLDivElement>(null);
  const inputContainerRef = useRef<HTMLDivElement>(null);
  const [showToolCallLoading, setShowToolCallLoading] = useState(false);

  // Edit dialog state
  const [editDialog, setEditDialog] = useState({
    isOpen: false,
    messageId: "",
    initialContent: "",
  });
  const [isEditing, setIsEditing] = useState(false);

  // Card reference state
  const [showBacklinkDialog, setShowBacklinkDialog] = useState(false);
  const [referencedCardDetails, setReferencedCardDetails] = useState<PartialCard[]>([]);

  // Fetch card details when referenced cards change
  useEffect(() => {
    const fetchCardDetails = async () => {
      if (!referencedCards || referencedCards.length === 0) {
        setReferencedCardDetails([]);
        return;
      }

      try {
        const details = await Promise.all(
          referencedCards.map(cardId => getCard(cardId))
        );
        // Filter out any error responses
        const validCards = details.filter(card => !('error' in card));
        setReferencedCardDetails(validCards);
      } catch (error) {
        console.error('Failed to fetch card details:', error);
      }
    };

    fetchCardDetails();
  }, [referencedCards]);

  // Check if a message is editable (within 5 minutes)
  const isMessageEditable = useCallback((message: ChatMessage) => {
    if (message.role !== "user") return false;
    const createdAt = new Date(message.created_at);
    const now = new Date();
    const diffMs = now.getTime() - createdAt.getTime();
    const fiveMinutes = 5 * 60 * 1000;
    return diffMs < fiveMinutes;
  }, []);

  useEffect(() => {
    scrollToBottom();
  }, [messages]);

  // Focus input when chat is cleared (no messages)
  useEffect(() => {
    if (messages.length === 0 && inputContainerRef.current) {
      // Find the input element within the container
      const inputElement = inputContainerRef.current.querySelector('textarea, input') as HTMLTextAreaElement | HTMLInputElement;
      if (inputElement && document.activeElement !== inputElement) {
        inputElement.focus();
      }
    }
  }, [messages.length]);

  const scrollToBottom = () => {
    messagesEndRef.current?.scrollIntoView({ behavior: "smooth" });
  };

  const handleOpenEditDialog = (message: ChatMessage) => {
    setEditDialog({
      isOpen: true,
      messageId: message.id,
      initialContent: message.content || "",
    });
  };

  const handleCloseEditDialog = () => {
    setEditDialog({
      isOpen: false,
      messageId: "",
      initialContent: "",
    });
  };

  // Handler for adding card via backlink dialog
  const handleAddBacklink = (card: PartialCard) => {
    const newReferencedCards = [...(referencedCards || []), String(card.id)];
    setReferencedCards(newReferencedCards);
    setShowBacklinkDialog(false);
  };

  // Handler for removing a referenced card
  const handleRemoveReferencedCard = (cardId: string) => {
    const newReferencedCards = (referencedCards || []).filter(id => id !== cardId);
    setReferencedCards(newReferencedCards);
  };

  const handleSaveEdit = async (newContent: string) => {
    if (!currentConversation || !editDialog.messageId) return;

    setIsEditing(true);
    try {
      const result = await editUserMessage(currentConversation.id, editDialog.messageId, { content: newContent });

      // Update the messages in the chat hook
      chatHook.setMessages(result.messages || []);

      // Close the dialog
      handleCloseEditDialog();

      // Trigger regeneration by sending a message to continue the conversation
      // The backend has deleted all messages after the edited one, so we need to regenerate
      if (result.messages && result.messages.length > 0) {
        // Find the last user message (the one we just edited)
        const lastUserMessage = [...result.messages].reverse().find(m => m.role === "user");
        if (lastUserMessage && lastUserMessage.id === editDialog.messageId) {
          // Trigger regeneration by streaming a new response
          // This will create a new assistant message
          await chatHook.sendMessageToConversation(
            currentConversation.id,
            newContent,
            lastUserMessage.referenced_cards
          );
        }
      }
    } catch (error) {
      console.error("Failed to edit message:", error);
      // Handle error - could show a toast notification
    } finally {
      setIsEditing(false);
    }
  };

  const getStatusIndicator = (status: string, hasContent: boolean = false, isStreaming: boolean = false) => {
    // For streaming with content, show inline cursor indicator instead
    if (status === 'processing' && hasContent && isStreaming) {
      return null; // Will show streaming cursor inline with content
    }

    switch (status) {
      case 'pending':
        return (
          <div className="flex items-center gap-2 text-amber-600 text-sm font-medium">
            <div className="relative flex h-3 w-3">
              <span className="animate-ping absolute inline-flex h-full w-full rounded-full bg-amber-400 opacity-75"></span>
              <span className="relative inline-flex rounded-full h-3 w-3 bg-amber-500"></span>
            </div>
            <span>Pending...</span>
          </div>
        );
      case 'processing':
        // Show skeleton for empty processing messages
        if (!hasContent) {
          return (
            <div className="flex items-center gap-3 text-blue-600 text-sm font-medium">
              <div className="flex items-center gap-2">
                <div className="flex space-x-1">
                  <div className="w-2 h-2 bg-blue-500 rounded-full animate-bounce"></div>
                  <div className="w-2 h-2 bg-blue-500 rounded-full animate-bounce" style={{ animationDelay: '0.15s' }}></div>
                  <div className="w-2 h-2 bg-blue-500 rounded-full animate-bounce" style={{ animationDelay: '0.3s' }}></div>
                </div>
                <span>Thinking</span>
              </div>
            </div>
          );
        }
        return null;
      case 'failed':
        return (
          <div className="flex items-center gap-2 text-red-600 text-sm font-medium">
            <svg className="w-4 h-4" fill="currentColor" viewBox="0 0 20 20">
              <path fillRule="evenodd" d="M10 18a8 8 0 100-16 8 8 0 000 16zM8.707 7.293a1 1 0 00-1.414 1.414L8.586 10l-1.293 1.293a1 1 0 101.414 1.414L10 11.414l1.293 1.293a1 1 0 001.414-1.414L11.414 10l1.293-1.293a1 1 0 00-1.414-1.414L10 8.586 8.707 7.293z" clipRule="evenodd" />
            </svg>
            <span>Failed</span>
          </div>
        );
      default:
        return null;
    }
  };

  const formatMessageContent = (message: ChatMessage) => {
    if (message.role === "tool" && message.content) {
      const isCollapsed = collapsedToolResults.has(message.id);
      const isRetrying = retryingToolIds.has(message.id);

      try {
        const toolResult = JSON.parse(message.content);
        const toolName = message._metadata?.tool_name || "Tool";
        const conversationId = currentConversation?.id || message.conversation_id;

        return (
          <ToolResultCard
            messageId={message.id}
            toolName={toolName}
            result={toolResult}
            metadata={message._metadata}
            isCollapsed={isCollapsed}
            onToggle={() => toggleToolResult(message.id)}
            onRetry={onRetryToolCall || retryTool}
            conversationId={conversationId}
            isRetrying={isRetrying}
          />
        );
      } catch {
        // Fallback for non-JSON content
        const toolName = message._metadata?.tool_name || "Tool";
        const conversationId = currentConversation?.id || message.conversation_id;

        return (
          <ToolResultCard
            messageId={message.id}
            toolName={toolName}
            result={{ output: message.content }}
            isCollapsed={isCollapsed}
            onToggle={() => toggleToolResult(message.id)}
            conversationId={conversationId}
          />
        );
      }
    }

    // For assistant messages
    if (message.role === "assistant") {
      const hasContent = message.content && message.content.trim().length > 0;
      const isStreaming = message.id === streamingMessageId;

      // Show status indicator for non-completed messages without content
      if (!hasContent && message.status !== 'completed') {
        return (
          <div className="flex items-center justify-center py-4">
            {getStatusIndicator(message.status, false, isStreaming)}
          </div>
        );
      }

      if (hasContent) {
        const { text, cards, tasks } = parseMessageContent(message.content!);

        return (
          <div>
            <div className="prose prose-sm max-w-none">
              <Markdown remarkPlugins={[remarkGfm]} rehypePlugins={[rehypeSanitize]}>
                {text}
              </Markdown>
              {isStreaming && <StreamingCursor />}
            </div>
            <CardsSection cards={cards} onCardClick={onCardClick} />
            <TasksSection tasks={tasks} onTaskClick={onTaskClick} />
          </div>
        );
      }
    }

    // For user messages
    if (message.role === "user" && message.content) {
      const filteredContent = message.content.replace(/<referenced cards>[\s\S]*?<\/referenced cards>/g, '').trim();
      return (
        <div className="whitespace-pre-wrap break-words leading-relaxed">
          {filteredContent}
        </div>
      );
    }

    return (
      <div className="whitespace-pre-wrap break-words leading-relaxed">
        {message.content}
      </div>
    );
  };

  const getMessageStyle = (role: string) => {
    const baseStyle = "shadow-sm border break-words transform transition-all duration-200 hover:shadow-md";
    switch (role) {
      case "user":
        return `bg-gradient-to-br from-blue-500 to-blue-600 text-white ml-auto max-w-[80%] text-right rounded-2xl rounded-br-md ${baseStyle}`;
      case "assistant":
        return `bg-gradient-to-br from-white to-gray-50 text-gray-900 mr-auto max-w-[80%] rounded-2xl rounded-bl-md border-gray-200 ${baseStyle}`;
      case "tool":
        return `mr-auto max-w-[90%] ${baseStyle}`;
      default:
        return `bg-gradient-to-br from-gray-50 to-gray-100 text-gray-600 mr-auto max-w-[80%] rounded-2xl border-gray-200 ${baseStyle}`;
    }
  };

  const inputPadding = compact ? "p-3" : "p-4";
  const messageSpacing = compact ? "space-y-4" : "space-y-6";
  const messagesPadding = compact ? "p-3 sm:p-4" : "p-4 sm:p-6";
  const inputBorder = compact ? "border-t border-gray-200" : "bg-white border-t border-gray-200";
  const textSize = compact ? "text-sm" : "text-sm";

  return (
    <div className="flex flex-col h-full min-h-0">
      {/* Messages */}
      <div className={`flex-1 overflow-y-auto ${messagesPadding} ${messageSpacing} bg-white min-h-0`}>
        {failedMessage && (
          <div className={`bg-gradient-to-r from-orange-50 to-amber-50 border border-orange-200 rounded-2xl shadow-sm p-4 ${compact ? 'px-3 py-3' : 'px-6 py-4'}`}>
            <div className="flex items-center justify-between gap-3">
              <div className="flex-1">
                <div className="flex items-center gap-2 text-orange-800 mb-2">
                  <svg className="w-5 h-5" fill="none" stroke="currentColor" viewBox="0 0 24 24">
                    <path strokeLinecap="round" strokeLinejoin="round" strokeWidth={2} d="M12 9v2m0 4h.01m-6.938 4h13.856c1.54 0 2.502-1.667 1.732-3L13.732 4c-.77-1.333-2.694-1.333-3.464 0L3.34 16c-.77 1.333.192 3 1.732 3z" />
                  </svg>
                  <span className="font-medium text-sm">Message failed to send</span>
                </div>
                <div className="bg-white rounded-lg p-3 text-sm text-gray-700 border border-orange-100 whitespace-pre-wrap break-words">
                  {failedMessage.content}
                </div>
                {failedMessage.referencedCards && failedMessage.referencedCards.length > 0 && (
                  <div className="mt-2 text-xs text-gray-500">
                    Referenced cards: {failedMessage.referencedCards.join(', ')}
                  </div>
                )}
              </div>
              <button
                onClick={() => retryFailedMessage?.()}
                disabled={isSending}
                className="flex-shrink-0 flex items-center gap-2 bg-orange-600 hover:bg-orange-700 text-white px-4 py-2 rounded-lg text-sm font-medium transition-colors duration-200 disabled:opacity-50 disabled:cursor-not-allowed"
              >
                <svg className="w-4 h-4" fill="none" stroke="currentColor" viewBox="0 0 24 24">
                  <path strokeLinecap="round" strokeLinejoin="round" strokeWidth={2} d="M4 4v5h.582m15.356 2A8.001 8.001 0 004.582 9m0 0H9m11 11v-5h-.581m0 0a8.003 8.003 0 01-15.357-2m15.357 2H15" />
                </svg>
                Retry
              </button>
            </div>
          </div>
        )}

        {error && (
          <div className={`bg-gradient-to-r from-red-50 to-pink-50 border border-red-200 text-red-700 px-6 py-4 rounded-2xl shadow-sm flex items-center gap-3 ${compact ? 'px-4 py-3' : ''}`}>
            <span className="text-xl">⚠️</span>
            <div>
              <div className="font-medium">Error</div>
              <div className={compact ? "text-xs" : "text-sm"}>{error}</div>
            </div>
          </div>
        )}

        {/* Tool call loading indicator */}
        {activeToolCalls && activeToolCalls.size > 0 && (
          <ToolCallLoading />
        )}

        {messages.map((message) => {
          return (
            <div key={message.id} className="flex items-start gap-3 group">
              <div className={`flex flex-col ${message.role === "user" ? "items-end w-full" : "flex-1"}`}>
                <div className={`${getMessageStyle(message.role)} ${message.role === "tool" ? "" : `py-2 px-4 ${textSize}`}`}>
                  {message.role !== "tool" && (
                    <div className={`text-xs mb-2 flex items-center gap-2 ${message.role === "user" ? "text-blue-100 justify-end" : "text-gray-500"}`}>
                      <span className="font-medium capitalize">{message.role}</span>
                      <span>•</span>
                      <RelativeTime timestamp={message.created_at} />
                    </div>
                  )}
                  {formatMessageContent(message)}
                </div>

                {/* Regenerate button for assistant messages */}
                {message.role === "assistant" && message.status === "completed" && onRegenerateMessage && (
                  <div className="mt-2 flex justify-start group">
                    <button
                      onClick={() => onRegenerateMessage(message.id)}
                      className="opacity-0 group-hover:opacity-100 focus-within:opacity-100 transition-opacity duration-200 text-gray-500 hover:text-gray-700 text-xs flex items-center gap-1 px-2 py-1 rounded hover:bg-gray-100"
                      title="Regenerate this message"
                    >
                      <svg className="w-3 h-3" fill="none" stroke="currentColor" viewBox="0 0 24 24">
                        <path strokeLinecap="round" strokeLinejoin="round" strokeWidth={2} d="M4 4v5h.582m15.356 2A8.001 2A8.001 0 004.582 9m0 0H9m11 11v-5h-.581m0 0a8.003 8.003 0 01-15.357-2m15.357 2H15" />
                      </svg>
                      Regenerate
                    </button>
                  </div>
                )}

                {/* Edit button for user messages (within 5 minutes) */}
                {message.role === "user" && isMessageEditable(message) && (
                  <div className="mt-2 flex justify-end group">
                    <button
                      onClick={() => handleOpenEditDialog(message)}
                      disabled={isEditing}
                      className="opacity-0 group-hover:opacity-100 focus-within:opacity-100 transition-opacity duration-200 text-blue-100 hover:text-white text-xs flex items-center gap-1 px-2 py-1 rounded hover:bg-blue-400/30 disabled:opacity-50 disabled:cursor-not-allowed"
                      title="Edit this message"
                    >
                      <svg className="w-3 h-3" fill="none" stroke="currentColor" viewBox="0 0 24 24">
                        <path strokeLinecap="round" strokeLinejoin="round" strokeWidth={2} d="M11 5H6a2 2 0 00-2 2v11a2 2 0 002 2h11a2 2 0 002-2v-5m-1.414-9.414a2 2 0 112.828 2.828L11.828 15H9v-2.828l8.586-8.586z" />
                      </svg>
                      Edit
                    </button>
                  </div>
                )}
              </div>
            </div>
          );
        })}

        <div ref={messagesEndRef} />
      </div>

      {/* Input */}
      <div className={inputBorder}>
        <div className="relative w-full sm:max-w-4xl mx-auto">
          <div className="relative border border-gray-300 rounded-2xl bg-white shadow-sm hover:shadow-md transition-all duration-200 focus-within:border-blue-500 focus-within:ring-2 focus-within:ring-blue-500/20">
            {/* Referenced Cards Display */}
            {referencedCardDetails.length > 0 && (
              <div className={`flex flex-wrap gap-2 px-4 pt-3 ${compact ? 'px-3 pt-2' : ''}`}>
                {referencedCardDetails.map((card) => (
                  <div
                    key={card.id}
                    className="inline-flex items-center gap-1.5 px-2 py-1 bg-blue-50 border border-blue-200 rounded-lg text-xs text-blue-700 group hover:bg-blue-100 transition-colors"
                  >
                    <span className="font-medium">[{card.card_id}]</span>
                    <span className="truncate max-w-[100px] sm:max-w-[150px]">{card.title}</span>
                    <button
                      onClick={() => handleRemoveReferencedCard(String(card.id))}
                      className="ml-0.5 text-blue-500 hover:text-blue-700 opacity-0 group-hover:opacity-100 md:opacity-0 focus-within:opacity-100 transition-opacity"
                      title="Remove card reference"
                    >
                      <svg className="w-3 h-3" fill="none" stroke="currentColor" viewBox="0 0 24 24">
                        <path strokeLinecap="round" strokeLinejoin="round" strokeWidth={2} d="M6 18L18 6M6 6l12 12" />
                      </svg>
                    </button>
                  </div>
                ))}
              </div>
            )}
            <div className={`flex items-end gap-2 sm:gap-3 ${inputPadding}`} ref={inputContainerRef}>
              <div className="flex-1 relative">
                <ChatInput
                  value={messageInput}
                  onChange={setMessageInput}
                  onSubmit={sendMessage}
                  onCardReference={handleCardReference}
                  onClearCommand={clearChat}
                  placeholder={placeholder}
                  disabled={isSending}
                  isLoading={isSending}
                  submitButtonText=""
                  multiline={true}
                  className="border-0 rounded-none p-0"
                />
              </div>

              <div className="flex items-center gap-2 flex-shrink-0">
                {/* Add Card Reference Button */}
                <button
                  onClick={() => setShowBacklinkDialog(true)}
                  disabled={isSending}
                  className="p-2.5 text-gray-500 hover:text-gray-700 hover:bg-gray-100 rounded-xl transition-all duration-200 disabled:opacity-50 disabled:cursor-not-abled disabled:hover:bg-transparent"
                  title="Add card reference"
                >
                  <svg className="w-4 h-4" fill="none" stroke="currentColor" viewBox="0 0 24 24">
                    <path strokeLinecap="round" strokeLinejoin="round" strokeWidth={2} d="M12 4v16m8-8H4" />
                  </svg>
                </button>

                {/* Send Button */}
                <button
                  onClick={() => sendMessage()}
                  disabled={!messageInput.trim() || isSending}
                  className={`p-2.5 bg-black hover:bg-gray-800 text-white rounded-xl transition-all duration-200 disabled:opacity-50 disabled:cursor-not-allowed disabled:hover:bg-black flex items-center justify-center ${compact ? 'min-w-[36px] p-2' : 'min-w-[44px]'}`}
                >
                  {isSending ? (
                    <div className={`border-2 border-white border-t-transparent rounded-full animate-spin ${compact ? 'w-3 h-3' : 'w-4 h-4'}`}></div>
                  ) : (
                    <svg className={compact ? "w-3 h-3" : "w-4 h-4"} fill="none" stroke="currentColor" viewBox="0 0 24 24">
                      <path strokeLinecap="round" strokeLinejoin="round" strokeWidth={2} d="M12 19l9 2-9-18-9 18 9-2zm0 0v-8" />
                    </svg>
                  )}
                </button>
              </div>
            </div>
          </div>
        </div>
      </div>

      {/* Edit Message Dialog */}
      <EditMessageDialog
        isOpen={editDialog.isOpen}
        initialContent={editDialog.initialContent}
        onSave={handleSaveEdit}
        onCancel={handleCloseEditDialog}
        isLoading={isEditing}
      />

      {/* Backlink Dialog */}
      {showBacklinkDialog && (
        <BacklinkDialog
          onClose={() => setShowBacklinkDialog(false)}
          onSelect={handleAddBacklink}
          setMessage={() => {}}
          excludeCardId={undefined}
        />
      )}
    </div>
  );
}