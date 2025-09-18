import React, { useState, useEffect, useRef } from "react";
import {
  ChatConversation,
  ChatMessage,
  createConversation,
  getConversations,
  getConversation,
  sendMessage as apiSendMessage,
  deleteConversation as apiDeleteConversation,
  starConversation as apiStarConversation,
} from "../api/chat";
import { setDocumentTitle } from "../utils/title";
import { Button } from "../components/Button";
import { useChatContext } from "../contexts/ChatContext";
import { renderTextWithCardLinks, parseMessageContent } from "../utils/chatUtils";
import { CardsSection } from "../components/chat/CardsSection";

interface ChatPageProps { }

export function ChatPage({ }: ChatPageProps) {
  const [conversations, setConversations] = useState<ChatConversation[]>([]);
  const [currentConversation, setCurrentConversation] = useState<ChatConversation | null>(null);
  const [messages, setMessages] = useState<ChatMessage[]>([]);
  const [messageInput, setMessageInput] = useState("");
  const [isLoading, setIsLoading] = useState(false);
  const [isSending, setIsSending] = useState(false);
  const [error, setError] = useState<string | null>(null);
  const [sidebarOpen, setSidebarOpen] = useState(false);
  const [selectedModel, setSelectedModel] = useState("gpt-4o-mini");
  const messagesEndRef = useRef<HTMLDivElement>(null);
  const { conversationId, setConversationId } = useChatContext();

  useEffect(() => {
    setDocumentTitle("Chat");
    loadConversations();
  }, []);

  useEffect(() => {
    scrollToBottom();
  }, [messages]);

  // Load specific conversation if set in context
  useEffect(() => {
    if (conversationId && conversationId !== currentConversation?.id) {
      loadConversation(conversationId);
    }
  }, [conversationId]);

  const scrollToBottom = () => {
    messagesEndRef.current?.scrollIntoView({ behavior: "smooth" });
  };

  const loadConversations = async () => {
    try {
      setIsLoading(true);
      const convs = await getConversations();

      console.log("convos", convs)
      if (!convs) {
        setConversations([]);
        return
      } else {
        setConversations(convs);
      }

      // If no current conversation and we have conversations, load the most recent one
      if (!currentConversation && convs.length > 0) {
        await loadConversation(convs[0].id);
      }
    } catch (error) {
      console.error("Failed to load conversations:", error);
      setError("Failed to load conversations");
    } finally {
      setIsLoading(false);
    }
  };

  const loadConversation = async (conversationId: string) => {
    try {
      setIsLoading(true);
      const data = await getConversation(conversationId);
      setCurrentConversation(data.conversation);
      setMessages(data.messages || []);
      setConversationId(conversationId);
      setError(null);
    } catch (error) {
      console.error("Failed to load conversation:", error);
      setError("Failed to load conversation");
    } finally {
      setIsLoading(false);
    }
  };

  const createNewConversation = async () => {
    try {
      setIsLoading(true);
      const newConv = await createConversation({
        title: "",
        model: selectedModel
      });

      setConversations(prev => [newConv, ...prev]);
      setCurrentConversation(newConv);
      setMessages([]);
      setConversationId(newConv.id);
      setError(null);
    } catch (error) {
      console.error("Failed to create conversation:", error);
      setError("Failed to create conversation");
    } finally {
      setIsLoading(false);
    }
  };

  const sendMessage = async () => {
    if (!messageInput.trim() || !currentConversation || isSending) return;

    const userMessage = messageInput.trim();
    setMessageInput("");
    setIsSending(true);

    try {
      // Add user message to UI immediately
      const tempUserMessage: ChatMessage = {
        id: `temp-${Date.now()}`,
        conversation_id: currentConversation.id,
        role: "user",
        content: userMessage,
        sequence_number: messages.length + 1,
        created_at: new Date().toISOString(),
      };
      setMessages(prev => [...prev, tempUserMessage]);

      // Send to API
      const newMessages = await apiSendMessage(currentConversation.id, userMessage);

      // Reload the full conversation to get all messages including tool calls
      await loadConversation(currentConversation.id);

      // Update conversations list to reflect new message count
      await loadConversations();

    } catch (error) {
      console.error("Failed to send message:", error);
      setError("Failed to send message");
      // Remove the temporary user message on error
      setMessages(prev => prev.slice(0, -1));
    } finally {
      setIsSending(false);
    }
  };

  const deleteConversation = async (conversationId: string) => {
    if (!confirm("Are you sure you want to delete this conversation?")) return;

    try {
      await apiDeleteConversation(conversationId);
      setConversations(prev => prev.filter(c => c.id !== conversationId));

      if (currentConversation?.id === conversationId) {
        const remaining = conversations.filter(c => c.id !== conversationId);
        if (remaining.length > 0) {
          await loadConversation(remaining[0].id);
        } else {
          setCurrentConversation(null);
          setMessages([]);
          setConversationId("");
        }
      }
    } catch (error) {
      console.error("Failed to delete conversation:", error);
      setError("Failed to delete conversation");
    }
  };

  const starConversation = async (conversationId: string) => {
    try {
      const updatedConv = await apiStarConversation(conversationId);
      setConversations(prev =>
        prev.map(c => c.id === conversationId ? updatedConv : c)
      );
      if (currentConversation?.id === conversationId) {
        setCurrentConversation(updatedConv);
      }
    } catch (error) {
      console.error("Failed to star conversation:", error);
      setError("Failed to star conversation");
    }
  };

  const handleKeyPress = (e: React.KeyboardEvent<HTMLTextAreaElement>) => {
    if (e.key === 'Enter' && !e.shiftKey) {
      e.preventDefault();
      sendMessage();
    }
  };

  const handleCardClick = (cardPk: string) => {
    // Navigate to the card page using the card_id
    window.open(`/app/card/${encodeURIComponent(cardPk)}`, '_blank');
  };

  const getRoleIcon = (role: string) => {
    switch (role) {
      case "user":
        return "👤";
      case "assistant":
        return "🤖";
      case "tool":
        return "🔧";
      default:
        return "💬";
    }
  };

  const formatMessageContent = (message: ChatMessage) => {
    if (message.role === "tool" && message.content) {
      try {
        const toolResult = JSON.parse(message.content);
        return (
          <div className="bg-gradient-to-br from-amber-50 to-yellow-50 border border-amber-200 rounded-lg p-4 shadow-sm">
            <div className="flex items-center gap-2 mb-3 text-amber-700">
              <span className="text-lg">🔧</span>
              <span className="font-medium text-sm">Tool Output</span>
            </div>
            <pre className="text-xs text-amber-800 overflow-x-auto whitespace-pre-wrap break-words font-mono bg-amber-50/50 p-2 rounded border">
              {JSON.stringify(toolResult, null, 2)}
            </pre>
          </div>
        );
      } catch {
        return (
          <div className="bg-gradient-to-br from-amber-50 to-yellow-50 border border-amber-200 rounded-lg p-4 shadow-sm">
            <div className="flex items-center gap-2 mb-3 text-amber-700">
              <span className="text-lg">🔧</span>
              <span className="font-medium text-sm">Tool Output</span>
            </div>
            <pre className="text-xs text-amber-800 whitespace-pre-wrap break-words font-mono bg-amber-50/50 p-2 rounded border">
              {message.content}
            </pre>
          </div>
        );
      }
    }

    // For assistant messages, parse and render card references as clickable links
    if (message.role === "assistant" && message.content) {
      console.log(message.content)
      const { text, cards } = parseMessageContent(message.content);

      return (
        <div>
          <div className="whitespace-pre-wrap break-words leading-relaxed">
            {renderTextWithCardLinks(text, handleCardClick)}
          </div>
          <CardsSection cards={cards} onCardClick={handleCardClick} />
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

  const availableModels = [
    { value: "gpt-4o-mini", label: "GPT-4o Mini" },
    { value: "gpt-4o", label: "GPT-4o" },
    { value: "openai/gpt-5", label: "GPT-5" },
  ];

  return (
    <div className="flex h-screen bg-white">
      {/* Conversations Sidebar */}
      <div className={`${sidebarOpen ? 'w-80' : 'w-0'} bg-gray-50 border-r border-gray-200 flex flex-col transition-all duration-300 overflow-hidden`}>
        <div className="p-4">
          <div className="flex items-center justify-between mb-6">
            <Button
              onClick={createNewConversation}
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

          {/* Model Selection */}
          <div className="mb-6">
            <label className="text-xs font-medium text-gray-600 block mb-2 uppercase tracking-wider">Model (Dev)</label>
            <select
              value={selectedModel}
              onChange={(e) => setSelectedModel(e.target.value)}
              className="w-full text-sm bg-white border border-gray-300 text-gray-900 rounded-lg px-3 py-2 focus:outline-none focus:ring-2 focus:ring-blue-500 focus:border-blue-500"
            >
              {availableModels.map((model) => (
                <option key={model.value} value={model.value}>
                  {model.label}
                </option>
              ))}
            </select>
          </div>

        </div>

        {/* Conversations List */}
        <div className="flex-1 overflow-y-auto px-2">
          {conversations.length === 0 ? (
            <div className="p-4 text-gray-500 text-center text-sm">
              No conversations yet
            </div>
          ) : (
            conversations.map((conv) => (
              <div
                key={conv.id}
                onClick={() => loadConversation(conv.id)}
                className={`group relative p-3 mx-2 mb-1 rounded-lg cursor-pointer transition-all duration-200 hover:bg-white ${
                  currentConversation?.id === conv.id ? 'bg-white shadow-sm' : ''
                }`}
              >
                <div className="flex items-center justify-between">
                  <div className="flex-1 min-w-0">
                    <h3 className="font-medium text-gray-900 text-sm truncate">
                      {conv.title || "Untitled Chat"}
                    </h3>
                    <div className="text-xs text-gray-500 mt-1">
                      {conv.message_count || 0} messages • {new Date(conv.updated_at).toLocaleDateString()}
                    </div>
                  </div>
                  <div className="flex items-center space-x-1 ml-2 opacity-0 group-hover:opacity-100 transition-opacity">
                    <button
                      onClick={(e) => {
                        e.stopPropagation();
                        starConversation(conv.id);
                      }}
                      className={`text-sm p-1 rounded hover:bg-gray-100 transition-colors ${
                        conv.starred ? 'text-yellow-500' : 'text-gray-500 hover:text-yellow-500'
                      }`}
                    >
                      <svg className="w-3 h-3" fill={conv.starred ? "currentColor" : "none"} stroke="currentColor" viewBox="0 0 24 24">
                        <path strokeLinecap="round" strokeLinejoin="round" strokeWidth={2} d="M11.049 2.927c.3-.921 1.603-.921 1.902 0l1.519 4.674a1 1 0 00.95.69h4.915c.969 0 1.371 1.24.588 1.81l-3.976 2.888a1 1 0 00-.363 1.118l1.518 4.674c.3.922-.755 1.688-1.538 1.118l-3.976-2.888a1 1 0 00-1.176 0l-3.976 2.888c-.783.57-1.838-.197-1.538-1.118l1.518-4.674a1 1 0 00-.363-1.118l-3.976-2.888c-.784-.57-.38-1.81.588-1.81h4.914a1 1 0 00.951-.69l1.519-4.674z" />
                      </svg>
                    </button>
                    <button
                      onClick={(e) => {
                        e.stopPropagation();
                        deleteConversation(conv.id);
                      }}
                      className="text-gray-500 hover:text-red-500 text-sm p-1 rounded hover:bg-gray-100 transition-colors"
                    >
                      <svg className="w-3 h-3" fill="none" stroke="currentColor" viewBox="0 0 24 24">
                        <path strokeLinecap="round" strokeLinejoin="round" strokeWidth={2} d="M19 7l-.867 12.142A2 2 0 0116.138 21H7.862a2 2 0 01-1.995-1.858L5 7m5 4v6m4-6v6m1-10V4a1 1 0 00-1-1h-4a1 1 0 00-1 1v3M4 7h16" />
                      </svg>
                    </button>
                  </div>
                </div>
              </div>
            ))
          )}
        </div>
      </div>

      {/* Chat Area */}
      <div className="flex-1 flex flex-col">
        {currentConversation ? (
          <>
            {/* Chat Header */}
            <div className="bg-white border-b border-gray-200 p-6 shadow-sm">
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
                <div className="w-10 h-10 rounded-full bg-gradient-to-br from-blue-500 to-blue-600 flex items-center justify-center shadow-sm">
                  <span className="text-white text-lg">🤖</span>
                </div>
                <div>
                  <h2 className="text-xl font-semibold text-gray-900">
                    {currentConversation.title || "Untitled Chat"}
                  </h2>
                  <p className="text-sm text-gray-500 flex items-center gap-2">
                    <span className="w-2 h-2 bg-green-500 rounded-full"></span>
                    Model: {currentConversation.model}
                  </p>
                </div>
              </div>
            </div>

            {/* Messages */}
            <div className="flex-1 overflow-y-auto p-6 space-y-6 bg-white">
              {error && (
                <div className="bg-gradient-to-r from-red-50 to-pink-50 border border-red-200 text-red-700 px-6 py-4 rounded-2xl shadow-sm flex items-center gap-3">
                  <span className="text-xl">\u26a0\ufe0f</span>
                  <div>
                    <div className="font-medium">Error</div>
                    <div className="text-sm">{error}</div>
                  </div>
                </div>
              )}

              {messages.map((message) => (
                <div key={message.id} className="flex items-start gap-3 group">
                  {message.role !== "user" && (
                    <div className="flex-shrink-0 w-8 h-8 rounded-full bg-gradient-to-br from-gray-100 to-gray-200 flex items-center justify-center shadow-sm">
                      <span className="text-sm">{getRoleIcon(message.role)}</span>
                    </div>
                  )}
                  <div className={`flex flex-col ${message.role === "user" ? "items-end w-full" : "flex-1"}`}>
                    {message.role === "user" && (
                      <div className="flex-shrink-0 w-8 h-8 rounded-full bg-gradient-to-br from-blue-500 to-blue-600 flex items-center justify-center shadow-sm mb-2">
                        <span className="text-sm text-white">{getRoleIcon(message.role)}</span>
                      </div>
                    )}
                    <div className={`${getMessageStyle(message.role)} ${message.role === "tool" ? "" : "p-4"}`}>
                      {message.role !== "tool" && (
                        <div className={`text-xs mb-2 flex items-center gap-2 ${message.role === "user" ? "text-blue-100 justify-end" : "text-gray-500"}`}>
                          <span className="font-medium capitalize">{message.role}</span>
                          <span>•</span>
                          <span>{new Date(message.created_at).toLocaleTimeString()}</span>
                        </div>
                      )}
                      {formatMessageContent(message)}
                    </div>
                  </div>
                </div>
              ))}

              {isSending && (
                <div className="flex items-start gap-3 group">
                  <div className="flex-shrink-0 w-8 h-8 rounded-full bg-gradient-to-br from-gray-100 to-gray-200 flex items-center justify-center shadow-sm">
                    <span className="text-sm">🤖</span>
                  </div>
                  <div className="flex-1">
                    <div className="bg-gradient-to-br from-white to-gray-50 rounded-2xl rounded-bl-md border border-gray-200 p-4 shadow-sm">
                      <div className="flex items-center gap-3 text-gray-600">
                        <div className="flex space-x-1">
                          <div className="w-2 h-2 bg-blue-500 rounded-full animate-bounce"></div>
                          <div className="w-2 h-2 bg-blue-500 rounded-full animate-bounce" style={{ animationDelay: '0.1s' }}></div>
                          <div className="w-2 h-2 bg-blue-500 rounded-full animate-bounce" style={{ animationDelay: '0.2s' }}></div>
                        </div>
                        <span className="text-sm font-medium">Assistant is thinking...</span>
                      </div>
                    </div>
                  </div>
                </div>
              )}

              <div ref={messagesEndRef} />
            </div>

            {/* Message Input */}
            <div className="bg-white border-t border-gray-200 p-6">
              <div className="flex items-end gap-3">
                <textarea
                  value={messageInput}
                  onChange={(e) => setMessageInput(e.target.value)}
                  onKeyPress={handleKeyPress}
                  placeholder="Ask about your cards..."
                  className="flex-1 resize-none rounded-2xl border border-gray-300 px-4 py-3 text-sm focus:outline-none focus:ring-2 focus:ring-blue-500 focus:border-transparent shadow-sm transition-all duration-200 hover:border-gray-400 focus:shadow-md"
                  rows={3}
                  disabled={isSending}
                />
                <Button
                  onClick={sendMessage}
                  disabled={!messageInput.trim() || isSending}
                  className="px-6 py-3 bg-gradient-to-r from-blue-500 to-blue-600 hover:from-blue-600 hover:to-blue-700 text-white rounded-2xl shadow-sm transition-all duration-200 hover:shadow-md transform hover:scale-105 disabled:opacity-50 disabled:transform-none disabled:hover:shadow-sm"
                >
                  {isSending ? (
                    <div className="flex items-center gap-2">
                      <div className="w-4 h-4 border-2 border-white border-t-transparent rounded-full animate-spin"></div>
                      <span>Sending</span>
                    </div>
                  ) : (
                    <div className="flex items-center gap-2">
                      <span>Send</span>
                      <span>→</span>
                    </div>
                  )}
                </Button>
              </div>
            </div>
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
                <Button
                  onClick={createNewConversation}
                  disabled={isLoading}
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
    </div>
  );
}