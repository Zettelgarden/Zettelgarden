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
        title: `Chat ${new Date().toLocaleDateString()}`,
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
    <div className="flex h-screen bg-gradient-to-br from-gray-50 to-gray-100">
      {/* Conversations Sidebar */}
      <div className={`${sidebarOpen ? 'w-80' : 'w-0'} bg-white border-r border-gray-200 flex flex-col shadow-lg transition-all duration-300 overflow-hidden`}>
        <div className="p-6 border-b border-gray-200 bg-gradient-to-r from-blue-50 to-indigo-50">
          <div className="flex items-center justify-between mb-4">
            <h1 className="text-2xl font-bold text-gray-900 flex items-center gap-2">
              <span className="text-2xl">💬</span>
              Chat
            </h1>
            <div className="flex items-center gap-2">
              <Button
                onClick={createNewConversation}
                disabled={isLoading}
                className="text-sm bg-gradient-to-r from-blue-500 to-blue-600 hover:from-blue-600 hover:to-blue-700 text-white rounded-xl px-4 py-2 shadow-sm transition-all duration-200 hover:shadow-md transform hover:scale-105"
              >
                <span className="flex items-center gap-2">
                  <span>+</span>
                  <span>New Chat</span>
                </span>
              </Button>
              <button
                onClick={() => setSidebarOpen(false)}
                className="text-gray-500 hover:text-gray-700 p-2 rounded-lg hover:bg-white/50 transition-colors"
              >
                ✕
              </button>
            </div>
          </div>

          {/* Model Selection */}
          <div className="px-6 pb-4">
            <label className="text-sm font-medium text-gray-700 block mb-2">Model</label>
            <select
              value={selectedModel}
              onChange={(e) => setSelectedModel(e.target.value)}
              className="w-full text-sm border border-gray-300 rounded-lg px-3 py-2 focus:outline-none focus:ring-2 focus:ring-blue-500 focus:border-transparent bg-white"
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
        <div className="flex-1 overflow-y-auto">
          {conversations.length === 0 ? (
            <div className="p-4 text-gray-500 text-center">
              No conversations yet. Create your first chat!
            </div>
          ) : (
            conversations.map((conv) => (
              <div
                key={conv.id}
                onClick={() => loadConversation(conv.id)}
                className={`p-4 border-b border-gray-100 cursor-pointer transition-all duration-200 hover:bg-gray-50 hover:shadow-sm ${currentConversation?.id === conv.id ? 'bg-gradient-to-r from-blue-50 to-indigo-50 border-blue-200 shadow-sm' : ''
                  }`}
              >
                <div className="flex justify-between items-start mb-2">
                  <h3 className="font-medium text-gray-900 text-sm truncate flex-1">
                    {conv.title || "Untitled Chat"}
                  </h3>
                  <div className="flex items-center space-x-1 ml-2">
                    {conv.starred && (
                      <span className="text-yellow-500 text-sm">★</span>
                    )}
                    <button
                      onClick={(e) => {
                        e.stopPropagation();
                        starConversation(conv.id);
                      }}
                      className="text-gray-400 hover:text-yellow-500 text-sm"
                    >
                      {conv.starred ? "★" : "☆"}
                    </button>
                    <button
                      onClick={(e) => {
                        e.stopPropagation();
                        deleteConversation(conv.id);
                      }}
                      className="text-gray-400 hover:text-red-500 text-sm"
                    >
                      ×
                    </button>
                  </div>
                </div>
                <div className="text-xs text-gray-500">
                  {conv.message_count || 0} messages • {new Date(conv.updated_at).toLocaleDateString()}
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
            <div className="flex-1 overflow-y-auto p-6 space-y-6 bg-gradient-to-b from-gray-50 to-white">
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
                          <div className="w-2 h-2 bg-blue-500 rounded-full animate-bounce" style={{animationDelay: '0.1s'}}></div>
                          <div className="w-2 h-2 bg-blue-500 rounded-full animate-bounce" style={{animationDelay: '0.2s'}}></div>
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
          <div className="flex-1 flex flex-col bg-gradient-to-b from-gray-50 to-white">
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
                <div className="w-24 h-24 mx-auto mb-6 rounded-full bg-gradient-to-br from-blue-100 to-indigo-100 flex items-center justify-center shadow-lg">
                  <span className="text-4xl">\ud83d\udcac</span>
                </div>
                <h3 className="text-2xl font-bold text-gray-900 mb-3">Welcome to Chat</h3>
                <p className="text-gray-600 mb-6 leading-relaxed">Create a new conversation to start chatting with your knowledge base. Ask questions, get insights, and explore your cards with AI assistance.</p>
                <Button
                  onClick={createNewConversation}
                  disabled={isLoading}
                  className="bg-gradient-to-r from-blue-500 to-blue-600 hover:from-blue-600 hover:to-blue-700 text-white rounded-xl px-8 py-3 shadow-lg transition-all duration-200 hover:shadow-xl transform hover:scale-105 disabled:opacity-50 disabled:transform-none"
                >
                  <span className="flex items-center gap-3">
                    <span className="text-lg">\ud83c\udf86</span>
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