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
  getUsageQuotas,
  UsageQuota
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
  const [usageQuotas, setUsageQuotas] = useState<UsageQuota[]>([]);
  const [error, setError] = useState<string | null>(null);
  const messagesEndRef = useRef<HTMLDivElement>(null);
  const { conversationId, setConversationId } = useChatContext();

  useEffect(() => {
    setDocumentTitle("Chat");
    loadConversations();
    loadUsageQuotas();
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
        model: "gpt-4o-mini"
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

  const loadUsageQuotas = async () => {
    try {
      const quotas = await getUsageQuotas();
      setUsageQuotas(quotas);
    } catch (error) {
      console.error("Failed to load usage quotas:", error);
    }
  };

  const handleKeyPress = (e: React.KeyboardEvent<HTMLTextAreaElement>) => {
    if (e.key === 'Enter' && !e.shiftKey) {
      e.preventDefault();
      sendMessage();
    }
  };

  const formatMessageContent = (message: ChatMessage) => {
    if (message.role === "tool" && message.content) {
      try {
        const toolResult = JSON.parse(message.content);
        return (
          <pre className="bg-yellow-50 p-3 rounded text-sm overflow-x-auto">
            {JSON.stringify(toolResult, null, 2)}
          </pre>
        );
      } catch {
        return <pre className="bg-yellow-50 p-3 rounded text-sm">{message.content}</pre>;
      }
    }
    return <div className="whitespace-pre-wrap">{message.content}</div>;
  };

  const getMessageStyle = (role: string) => {
    switch (role) {
      case "user":
        return "bg-blue-500 text-white ml-auto max-w-[80%] text-right";
      case "assistant":
        return "bg-gray-100 text-gray-900 mr-auto max-w-[80%]";
      case "tool":
        return "bg-yellow-50 border border-yellow-200 text-yellow-800 mr-auto max-w-[90%] text-xs";
      default:
        return "bg-gray-50 text-gray-600 mr-auto max-w-[80%]";
    }
  };

  return (
    <div className="flex h-screen bg-gray-50">
      {/* Conversations Sidebar */}
      <div className="w-80 bg-white border-r border-gray-200 flex flex-col">
        <div className="p-4 border-b border-gray-200">
          <div className="flex items-center justify-between mb-4">
            <h1 className="text-xl font-semibold text-gray-900">Chat</h1>
            <Button
              onClick={createNewConversation}
              disabled={isLoading}
              className="text-sm"
            >
              + New Chat
            </Button>
          </div>

          {/* Usage Quotas */}
          {usageQuotas && usageQuotas.length > 0 && (
            <div className="space-y-2">
              <h3 className="text-sm font-medium text-gray-700">Daily Usage</h3>
              {usageQuotas.map((quota) => (
                <div key={quota.quota_type} className="text-xs">
                  <div className="flex justify-between text-gray-600">
                    <span>{quota.quota_type.replace('_', ' ')}</span>
                    <span>{quota.current_usage}/{quota.max_limit}</span>
                  </div>
                  <div className="w-full bg-gray-200 rounded-full h-1.5">
                    <div
                      className="bg-blue-500 h-1.5 rounded-full"
                      style={{ width: `${(quota.current_usage / quota.max_limit) * 100}%` }}
                    ></div>
                  </div>
                </div>
              ))}
            </div>
          )}
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
                className={`p-4 border-b border-gray-100 cursor-pointer hover:bg-gray-50 ${currentConversation?.id === conv.id ? 'bg-blue-50 border-blue-200' : ''
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
            <div className="bg-white border-b border-gray-200 p-4">
              <h2 className="text-lg font-medium text-gray-900">
                {currentConversation.title || "Untitled Chat"}
              </h2>
              <p className="text-sm text-gray-500">
                Model: {currentConversation.model}
              </p>
            </div>

            {/* Messages */}
            <div className="flex-1 overflow-y-auto p-4 space-y-4">
              {error && (
                <div className="bg-red-50 border border-red-200 text-red-700 px-4 py-3 rounded">
                  {error}
                </div>
              )}

              {messages.map((message) => (
                <div
                  key={message.id}
                  className={`p-3 rounded-lg ${getMessageStyle(message.role)}`}
                >
                  <div className="text-xs opacity-70 mb-1">
                    {message.role} • {new Date(message.created_at).toLocaleTimeString()}
                  </div>
                  {formatMessageContent(message)}
                </div>
              ))}

              {isSending && (
                <div className="flex items-center space-x-2 text-gray-500">
                  <div className="animate-spin rounded-full h-4 w-4 border-b-2 border-gray-500"></div>
                  <span className="text-sm">Thinking...</span>
                </div>
              )}

              <div ref={messagesEndRef} />
            </div>

            {/* Message Input */}
            <div className="bg-white border-t border-gray-200 p-4">
              <div className="flex space-x-2">
                <textarea
                  value={messageInput}
                  onChange={(e) => setMessageInput(e.target.value)}
                  onKeyPress={handleKeyPress}
                  placeholder="Ask about your cards..."
                  className="flex-1 resize-none rounded-lg border border-gray-300 px-3 py-2 text-sm focus:outline-none focus:ring-2 focus:ring-blue-500 focus:border-transparent"
                  rows={3}
                  disabled={isSending}
                />
                <Button
                  onClick={sendMessage}
                  disabled={!messageInput.trim() || isSending}
                  className="px-4 py-2"
                >
                  {isSending ? "Sending..." : "Send"}
                </Button>
              </div>
            </div>
          </>
        ) : (
          <div className="flex-1 flex items-center justify-center">
            <div className="text-center text-gray-500">
              <h3 className="text-lg font-medium mb-2">Welcome to Chat</h3>
              <p className="mb-4">Create a new conversation to start chatting with your knowledge base.</p>
              <Button onClick={createNewConversation} disabled={isLoading}>
                Start New Chat
              </Button>
            </div>
          </div>
        )}
      </div>
    </div>
  );
}