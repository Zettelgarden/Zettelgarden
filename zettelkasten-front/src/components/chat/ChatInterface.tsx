import React, { useEffect, useRef } from "react";
import Markdown from "react-markdown";
import remarkGfm from "remark-gfm";
import { ChatMessage } from "../../api/chat";
import { parseMessageContent } from "../../utils/chatUtils";
import { CardsSection } from "./CardsSection";
import { TasksSection } from "./TasksSection";
import { ChatInput } from "./ChatInput";
import { useChat } from "../../hooks/useChat";

interface ChatInterfaceProps {
  chatHook: ReturnType<typeof useChat>;
  onCardClick: (cardPk: string) => void;
  onTaskClick: (taskId: number) => void;
  placeholder?: string;
  compact?: boolean;
  showModelDropdown?: boolean;
  availableModels?: { value: string; label: string }[];
  onRegenerateMessage?: (messageId: string) => void;
}

export function ChatInterface({
  chatHook,
  onCardClick,
  onTaskClick,
  placeholder = "Ask about your cards... Type @ to mention a card",
  compact = false,
  showModelDropdown = true,
  availableModels = [
    { value: "google/gemini-2.5-flash", label: "google/gemini-2.5-flash" },
    { value: "google/gemini-2.5-flash-lite", label: "google/gemini-2.5-flash-lite" },
    { value: "google/gemini-2.5-pro", label: "google/gemini-2.5-pro" },
    { value: "gpt-4o-mini", label: "GPT-4o Mini" },
    { value: "openai/gpt-5", label: "GPT-5" },
    { value: "anthropic/claude-sonnet-4", label: "anthropic/claude-sonnet-4" },
  ],
  onRegenerateMessage
}: ChatInterfaceProps) {
  const {
    messages,
    messageInput,
    isSending,
    error,
    selectedModel,
    collapsedToolResults,
    showModelDropdown: internalShowModelDropdown,
    setMessageInput,
    setSelectedModel,
    setShowModelDropdown,
    sendMessage,
    handleCardReference,
    toggleToolResult,
  } = chatHook;

  const messagesEndRef = useRef<HTMLDivElement>(null);
  const modelDropdownRef = useRef<HTMLDivElement>(null);

  useEffect(() => {
    scrollToBottom();
  }, [messages]);

  // Handle clicking outside the model dropdown
  useEffect(() => {
    const handleClickOutside = (event: MouseEvent) => {
      if (modelDropdownRef.current && !modelDropdownRef.current.contains(event.target as Node)) {
        setShowModelDropdown(false);
      }
    };

    if (internalShowModelDropdown) {
      document.addEventListener('mousedown', handleClickOutside);
      return () => {
        document.removeEventListener('mousedown', handleClickOutside);
      };
    }
  }, [internalShowModelDropdown, setShowModelDropdown]);

  const scrollToBottom = () => {
    messagesEndRef.current?.scrollIntoView({ behavior: "smooth" });
  };

  const getStatusIndicator = (status: string) => {
    switch (status) {
      case 'pending':
        return (
          <div className="flex items-center gap-2 text-amber-600 text-xs">
            <div className="w-2 h-2 bg-amber-500 rounded-full animate-pulse"></div>
            <span>Pending...</span>
          </div>
        );
      case 'processing':
        return (
          <div className="flex items-center gap-2 text-blue-600 text-xs">
            <div className="flex space-x-1">
              <div className="w-1.5 h-1.5 bg-blue-500 rounded-full animate-bounce"></div>
              <div className="w-1.5 h-1.5 bg-blue-500 rounded-full animate-bounce" style={{ animationDelay: '0.1s' }}></div>
              <div className="w-1.5 h-1.5 bg-blue-500 rounded-full animate-bounce" style={{ animationDelay: '0.2s' }}></div>
            </div>
            <span>Processing...</span>
          </div>
        );
      case 'failed':
        return (
          <div className="flex items-center gap-2 text-red-600 text-xs">
            <div className="w-2 h-2 bg-red-500 rounded-full"></div>
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

      try {
        const toolResult = JSON.parse(message.content);
        return (
          <div className="bg-gradient-to-br from-amber-50 to-yellow-50 border border-amber-200 rounded-lg shadow-sm">
            <button
              onClick={() => toggleToolResult(message.id)}
              className="w-full px-4 py-1 text-left hover:bg-amber-100/50 transition-colors rounded-lg"
            >
              <div className="flex items-center justify-between text-amber-700">
                <div className="flex items-center gap-2">
                  <span className="text-lg">🔧</span>
                  <span className="font-medium text-sm">Tool Output</span>
                </div>
                <svg
                  className={`w-4 h-4 transition-transform duration-200 ${isCollapsed ? '' : 'rotate-180'}`}
                  fill="none"
                  stroke="currentColor"
                  viewBox="0 0 24 24"
                >
                  <path strokeLinecap="round" strokeLinejoin="round" strokeWidth={2} d="M19 9l-7 7-7-7" />
                </svg>
              </div>
            </button>
            {!isCollapsed && (
              <div className="px-4 pb-4">
                <pre className="text-xs text-amber-800 overflow-x-auto whitespace-pre-wrap break-words font-mono bg-amber-50/50 p-2 rounded border">
                  {JSON.stringify(toolResult, null, 2)}
                </pre>
              </div>
            )}
          </div>
        );
      } catch {
        return (
          <div className="bg-gradient-to-br from-amber-50 to-yellow-50 border border-amber-200 rounded-lg shadow-sm">
            <button
              onClick={() => toggleToolResult(message.id)}
              className="w-full p-4 text-left hover:bg-amber-100/50 transition-colors rounded-lg"
            >
              <div className="flex items-center justify-between text-amber-700">
                <div className="flex items-center gap-2">
                  <span className="text-lg">🔧</span>
                  <span className="font-medium text-sm">Tool Output</span>
                </div>
                <svg
                  className={`w-4 h-4 transition-transform duration-200 ${isCollapsed ? '' : 'rotate-180'}`}
                  fill="none"
                  stroke="currentColor"
                  viewBox="0 0 24 24"
                >
                  <path strokeLinecap="round" strokeLinejoin="round" strokeWidth={2} d="M19 9l-7 7-7-7" />
                </svg>
              </div>
            </button>
            {!isCollapsed && (
              <div className="px-4 pb-4">
                <pre className="text-xs text-amber-800 whitespace-pre-wrap break-words font-mono bg-amber-50/50 p-2 rounded border">
                  {message.content}
                </pre>
              </div>
            )}
          </div>
        );
      }
    }

    // For assistant messages
    if (message.role === "assistant") {
      if (message.status !== 'completed') {
        return (
          <div className="flex items-center justify-center py-4">
            {getStatusIndicator(message.status)}
          </div>
        );
      }

      if (message.content) {
        const { text, cards, tasks } = parseMessageContent(message.content);

        return (
          <div>
            <div className="prose prose-sm max-w-none">
              <Markdown remarkPlugins={[remarkGfm]}>
                {text}
              </Markdown>
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
  const messagesPadding = compact ? "p-4" : "p-6";
  const inputBorder = compact ? "border-t border-gray-200" : "bg-white border-t border-gray-200";
  const textSize = compact ? "text-sm" : "text-sm";

  return (
    <div className="flex flex-col h-full">
      {/* Messages */}
      <div className={`flex-1 overflow-y-auto ${messagesPadding} ${messageSpacing} bg-white`}>
        {error && (
          <div className={`bg-gradient-to-r from-red-50 to-pink-50 border border-red-200 text-red-700 px-6 py-4 rounded-2xl shadow-sm flex items-center gap-3 ${compact ? 'px-4 py-3' : ''}`}>
            <span className="text-xl">⚠️</span>
            <div>
              <div className="font-medium">Error</div>
              <div className={compact ? "text-xs" : "text-sm"}>{error}</div>
            </div>
          </div>
        )}

        {messages.map((message) => (
          <div key={message.id} className="flex items-start gap-3 group">
            <div className={`flex flex-col ${message.role === "user" ? "items-end w-full" : "flex-1"}`}>
              <div className={`${getMessageStyle(message.role)} ${message.role === "tool" ? "" : `py-2 px-4 ${textSize}`}`}>
                {message.role !== "tool" && (
                  <div className={`text-xs mb-2 flex items-center gap-2 ${message.role === "user" ? "text-blue-100 justify-end" : "text-gray-500"}`}>
                    <span className="font-medium capitalize">{message.role}</span>
                    <span>•</span>
                    <span>{new Date(message.created_at).toLocaleTimeString()}</span>
                  </div>
                )}
                {formatMessageContent(message)}
              </div>

              {/* Regenerate button for assistant messages */}
              {message.role === "assistant" && message.status === "completed" && onRegenerateMessage && (
                <div className="mt-2 flex justify-start">
                  <button
                    onClick={() => onRegenerateMessage(message.id)}
                    className="opacity-0 group-hover:opacity-100 transition-opacity duration-200 text-gray-500 hover:text-gray-700 text-xs flex items-center gap-1 px-2 py-1 rounded hover:bg-gray-100"
                    title="Regenerate this message"
                  >
                    <svg className="w-3 h-3" fill="none" stroke="currentColor" viewBox="0 0 24 24">
                      <path strokeLinecap="round" strokeLinejoin="round" strokeWidth={2} d="M4 4v5h.582m15.356 2A8.001 8.001 0 004.582 9m0 0H9m11 11v-5h-.581m0 0a8.003 8.003 0 01-15.357-2m15.357 2H15" />
                    </svg>
                    Regenerate
                  </button>
                </div>
              )}
            </div>
          </div>
        ))}

        <div ref={messagesEndRef} />
      </div>

      {/* Input */}
      <div className={inputBorder}>
        <div className={`relative max-w-4xl mx-auto ${compact ? '' : 'max-w-4xl'}`}>
          <div className="relative border border-gray-300 rounded-2xl bg-white shadow-sm hover:shadow-md transition-all duration-200 focus-within:border-blue-500 focus-within:ring-2 focus-within:ring-blue-500/20">
            <div className={`flex items-end gap-3 ${inputPadding}`}>
              <div className="flex-1 relative">
                <ChatInput
                  value={messageInput}
                  onChange={setMessageInput}
                  onSubmit={sendMessage}
                  onCardReference={handleCardReference}
                  placeholder={placeholder}
                  disabled={isSending}
                  isLoading={isSending}
                  submitButtonText=""
                  multiline={true}
                  className="border-0 rounded-none p-0"
                />
              </div>

              <div className="flex items-center gap-2 flex-shrink-0">
                {/* Model Dropdown */}
                {showModelDropdown && internalShowModelDropdown && (
                  <div ref={modelDropdownRef} className="absolute bottom-16 left-4 z-10">
                    <div className="bg-white border border-gray-200 rounded-lg shadow-lg min-w-[200px] max-h-60 overflow-y-auto">
                      {availableModels.map((model) => (
                        <button
                          key={model.value}
                          onClick={() => {
                            setSelectedModel(model.value);
                            setShowModelDropdown(false);
                          }}
                          className={`w-full text-left px-3 py-2 text-xs hover:bg-gray-50 transition-colors ${
                            selectedModel === model.value
                              ? 'bg-blue-50 text-blue-700 font-medium'
                              : 'text-gray-700'
                          }`}
                        >
                          {model.label}
                        </button>
                      ))}
                    </div>
                  </div>
                )}

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

            {/* Model indicator */}
            {showModelDropdown && (
              <div className={`px-4 pb-3 ${compact ? 'px-3 pb-2' : ''}`}>
                <div className="flex items-center justify-between text-xs text-gray-500">
                  <button
                    onClick={() => setShowModelDropdown(!internalShowModelDropdown)}
                    className="flex items-center gap-2 hover:text-gray-700 transition-colors cursor-pointer rounded-md px-2 py-1 hover:bg-gray-50"
                  >
                    <div className="w-1.5 h-1.5 bg-green-500 rounded-full"></div>
                    <span>Using {availableModels.find(m => m.value === selectedModel)?.label}</span>
                    <svg
                      className={`w-3 h-3 transition-transform duration-200 ${internalShowModelDropdown ? 'rotate-180' : ''}`}
                      fill="none"
                      stroke="currentColor"
                      viewBox="0 0 24 24"
                    >
                      <path strokeLinecap="round" strokeLinejoin="round" strokeWidth={2} d="M19 9l-7 7-7-7" />
                    </svg>
                  </button>
                  {!compact && (
                    <div className="text-gray-400">
                      Press Enter to send • Shift+Enter for new line
                    </div>
                  )}
                </div>
              </div>
            )}
          </div>
        </div>
      </div>
    </div>
  );
}