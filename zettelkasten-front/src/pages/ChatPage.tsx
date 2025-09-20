import React, { useState, useEffect, useRef } from "react";
import Markdown from "react-markdown";
import remarkGfm from "remark-gfm";
import {
  ChatConversation,
  ChatMessage,
  createConversation,
  getConversations,
  getConversation,
  sendMessage as apiSendMessage,
  deleteConversation as apiDeleteConversation,
  starConversation as apiStarConversation,
  getConversationStatus,
} from "../api/chat";
import { setDocumentTitle } from "../utils/title";
import { Button } from "../components/Button";
import { useChatContext } from "../contexts/ChatContext";
import { parseMessageContent } from "../utils/chatUtils";
import { CardsSection } from "../components/chat/CardsSection";
import { ChatInput } from "../components/chat/ChatInput";
import { Link } from "react-router-dom";
import { useAuth } from "../contexts/AuthContext";

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
  const [selectedModel, setSelectedModel] = useState("google/gemini-2.5-pro");
  const [collapsedToolResults, setCollapsedToolResults] = useState<Set<string>>(new Set());
  const [showAllRecent, setShowAllRecent] = useState(false);
  const [referencedCards, setReferencedCards] = useState<string[]>([]);
  const [isPolling, setIsPolling] = useState(false);
  const messagesEndRef = useRef<HTMLDivElement>(null);
  const pollingIntervalRef = useRef<NodeJS.Timeout | null>(null);
  const { conversationId, setConversationId } = useChatContext();
  const { hasSubscription } = useAuth();

  useEffect(() => {
    setDocumentTitle("Chat");
    loadConversations();
    handleUrlParams();
  }, []);

  const handleUrlParams = async () => {
    const urlParams = new URLSearchParams(window.location.search);
    const message = urlParams.get('message');
    const cardsParam = urlParams.get('cards');

    if (message) {
      // Clear URL params to avoid re-triggering
      window.history.replaceState({}, '', '/app/chat');

      // Parse referenced cards
      const referencedCards = cardsParam ? cardsParam.split(',').filter(Boolean) : undefined;

      // Create new conversation and send message
      await createNewConversationWithMessage(message, referencedCards);
    }
  };

  const createNewConversationWithMessage = async (message: string, referencedCards?: string[]) => {
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

      // Set message input and send it
      setMessageInput(message);
      await sendMessageToConversation(newConv.id, message, referencedCards);
    } catch (error) {
      console.error("Failed to create conversation with message:", error);
      setError("Failed to create conversation");
    } finally {
      setIsLoading(false);
    }
  };

  const sendMessageToConversation = async (conversationId: string, message: string, referencedCards?: string[]) => {
    setIsSending(true);

    try {
      // Send to API with referenced cards - this now returns immediately with user message and pending assistant message
      await apiSendMessage(conversationId, message, referencedCards?.length ? referencedCards : undefined, selectedModel);

      // Reload conversation to get the new messages with correct status
      await refreshMessages(conversationId);

      // Start polling for status updates
      startPolling(conversationId);

      // Update conversations list to reflect new message count
      await loadConversations();

    } catch (error) {
      console.error("Failed to send message:", error);
      setError("Failed to send message");
    } finally {
      setIsSending(false);
    }
  };

  const startPolling = (conversationId: string) => {
    if (pollingIntervalRef.current) {
      clearInterval(pollingIntervalRef.current);
    }

    setIsPolling(true);

    pollingIntervalRef.current = setInterval(async () => {
      try {
        const status = await getConversationStatus(conversationId);

        // If no pending or processing messages, stop polling and reload just the messages
        if (!status.has_pending && !status.has_processing) {
          stopPolling();
          await refreshMessages(conversationId);
        }
      } catch (error) {
        console.error("Failed to check conversation status:", error);
        // Continue polling even if status check fails
      }
    }, 2000); // Poll every 2 seconds
  };

  const stopPolling = () => {
    if (pollingIntervalRef.current) {
      clearInterval(pollingIntervalRef.current);
      pollingIntervalRef.current = null;
    }
    setIsPolling(false);
  };

  const refreshMessages = async (conversationId: string) => {
    try {
      const data = await getConversation(conversationId);
      setMessages(data.messages || []);
    } catch (error) {
      console.error("Failed to refresh messages:", error);
    }
  };

  // Cleanup polling on unmount
  useEffect(() => {
    return () => {
      stopPolling();
    };
  }, []);

  useEffect(() => {
    scrollToBottom();
    // Reset collapsed state when messages change and add new tool results as collapsed by default
    const newCollapsedSet = new Set<string>();
    messages.forEach(msg => {
      if (msg.role === "tool") {
        newCollapsedSet.add(msg.id);
      }
    });
    setCollapsedToolResults(newCollapsedSet);
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

      // Stop any existing polling when switching conversations
      stopPolling();

      const data = await getConversation(conversationId);
      setCurrentConversation(data.conversation);
      setMessages(data.messages || []);
      setConversationId(conversationId);
      setError(null);

      // Check if there are pending/processing messages and start polling if needed
      const hasPendingOrProcessing = data.messages.some(msg =>
        msg.status === 'pending' || msg.status === 'processing'
      );
      if (hasPendingOrProcessing) {
        startPolling(conversationId);
      }
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

  const sendMessage = async (passedReferencedCards?: string[]) => {
    if (!messageInput.trim() || !currentConversation || isSending) return;

    const userMessage = messageInput.trim();
    const cardIds = passedReferencedCards || referencedCards;

    setMessageInput("");
    setReferencedCards([]); // Clear referenced cards
    setIsSending(true);

    try {
      // Send to API with referenced cards - this now returns immediately with user message and pending assistant message
      await apiSendMessage(currentConversation.id, userMessage, cardIds.length > 0 ? cardIds : undefined, selectedModel);

      // Reload conversation to get the new messages with correct status
      await refreshMessages(currentConversation.id);

      // Start polling for status updates
      startPolling(currentConversation.id);

      // Update conversations list to reflect new message count
      await loadConversations();

    } catch (error) {
      console.error("Failed to send message:", error);
      setError("Failed to send message");
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

  const handleCardReference = (cardIds: string[]) => {
    setReferencedCards(cardIds);
  };

  const handleCardClick = (cardPk: string) => {
    // Navigate to the card page using the card_id
    window.open(`/app/card/${encodeURIComponent(cardPk)}`, '_blank');
  };

  const toggleToolResult = (messageId: string) => {
    setCollapsedToolResults(prev => {
      const newSet = new Set(prev);
      if (newSet.has(messageId)) {
        newSet.delete(messageId);
      } else {
        newSet.add(messageId);
      }
      return newSet;
    });
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

  const filterReferencedCardsSection = (content: string) => {
    // Remove content between <referenced cards> and </referenced cards> tags
    return content.replace(/<referenced cards>[\s\S]*?<\/referenced cards>/g, '').trim();
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

    // For assistant messages, parse and render card references as clickable links with markdown
    if (message.role === "assistant") {
      // Show status indicator for pending, processing, or failed messages
      if (message.status !== 'completed') {
        return (
          <div className="flex items-center justify-center py-4">
            {getStatusIndicator(message.status)}
          </div>
        );
      }

      if (message.content) {
        console.log(message.content)
        const { text, cards } = parseMessageContent(message.content);

        return (
          <div>
            <div className="prose prose-sm max-w-none">
              <Markdown
                remarkPlugins={[remarkGfm]}
              >
                {text}
              </Markdown>
            </div>
            <CardsSection cards={cards} onCardClick={handleCardClick} />
          </div>
        );
      }
    }

    // For user messages, filter out referenced cards section
    if (message.role === "user" && message.content) {
      const filteredContent = filterReferencedCardsSection(message.content);
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

  const availableModels = [
    { value: "google/gemini-2.5-flash", label: "google/gemini-2.5-flash" },
    { value: "google/gemini-2.5-flash-lite", label: "google/gemini-2.5-flash-lite" },
    { value: "google/gemini-2.5-pro", label: "google/gemini-2.5-pro" },
    { value: "gpt-4o-mini", label: "GPT-4o Mini" },
    { value: "openai/gpt-5", label: "GPT-5" },
    { value: "anthropic/claude-sonnet-4", label: "anthropic/claude-sonnet-4" },

  ];

  // Separate starred and recent conversations
  const starredConversations = conversations.filter(conv => conv.starred);
  const allRecentConversations = conversations.filter(conv => !conv.starred);

  // Limit recent conversations display
  const RECENT_LIMIT = 25;
  const displayedRecentConversations = showAllRecent
    ? allRecentConversations
    : allRecentConversations.slice(0, RECENT_LIMIT);
  const remainingRecentCount = allRecentConversations.length - RECENT_LIMIT;

  return (
    <div className="flex h-screen bg-white">
      {/* Conversations Sidebar */}
      <div className={`${sidebarOpen ? 'w-80' : 'w-0'} bg-gray-50 border-r border-gray-200 flex flex-col transition-all duration-300 overflow-hidden`}>
        <div className="pt-4 px-4 pb-2">
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


        </div>

        {/* Conversations List */}
        <div className="flex-1 overflow-y-auto px-2">
          {conversations.length === 0 ? (
            <div className="p-4 text-gray-500 text-center text-sm">
              No conversations yet
            </div>
          ) : (
            <>
              {/* Starred Conversations Section */}
              {starredConversations.length > 0 && (
                <div className="mb-4">
                  <div className="px-2 py-1 mb-2">
                    <h4 className="text-xs font-semibold text-gray-600 uppercase tracking-wider flex items-center gap-2">
                      <svg className="w-3 h-3 text-yellow-500" fill="currentColor" viewBox="0 0 24 24">
                        <path strokeLinecap="round" strokeLinejoin="round" strokeWidth={2} d="M11.049 2.927c.3-.921 1.603-.921 1.902 0l1.519 4.674a1 1 0 00.95.69h4.915c.969 0 1.371 1.24.588 1.81l-3.976 2.888a1 1 0 00-.363 1.118l1.518 4.674c.3.922-.755 1.688-1.538 1.118l-3.976-2.888a1 1 0 00-1.176 0l-3.976 2.888c-.783.57-1.838-.197-1.538-1.118l1.518-4.674a1 1 0 00-.363-1.118l-3.976-2.888c-.784-.57-.38-1.81.588-1.81h4.914a1 1 0 00.951-.69l1.519-4.674z" />
                      </svg>
                      Starred
                    </h4>
                  </div>
                  {starredConversations.map((conv) => (
                    <div
                      key={conv.id}
                      onClick={() => loadConversation(conv.id)}
                      className={`group relative p-2 mx-1 mb-1 rounded-lg cursor-pointer transition-all duration-200 hover:bg-white ${currentConversation?.id === conv.id ? 'bg-white shadow-sm' : ''
                        }`}
                    >
                      <div className="flex items-center justify-between">
                        <div className="flex-1 min-w-0">
                          <h3 className="font-medium text-gray-900 text-sm truncate">
                            {conv.title || "Untitled Chat"}
                          </h3>
                        </div>
                        <div className="flex items-center space-x-1 ml-2 opacity-0 group-hover:opacity-100 transition-opacity">
                          <button
                            onClick={(e) => {
                              e.stopPropagation();
                              starConversation(conv.id);
                            }}
                            className={`text-sm p-1 rounded hover:bg-gray-100 transition-colors ${conv.starred ? 'text-yellow-500' : 'text-gray-500 hover:text-yellow-500'
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
                  ))}
                </div>
              )}

              {/* Recent Conversations Section */}
              {allRecentConversations.length > 0 && (
                <div>
                  <div className="px-2 py-1 mb-2">
                    <h4 className="text-xs font-semibold text-gray-600 uppercase tracking-wider flex items-center gap-2">
                      <svg className="w-3 h-3 text-gray-500" fill="none" stroke="currentColor" viewBox="0 0 24 24">
                        <path strokeLinecap="round" strokeLinejoin="round" strokeWidth={2} d="M12 8v4l3 3m6-3a9 9 0 11-18 0 9 9 0 0118 0z" />
                      </svg>
                      Recent
                    </h4>
                  </div>
                  {displayedRecentConversations.map((conv) => (
                    <div
                      key={conv.id}
                      onClick={() => loadConversation(conv.id)}
                      className={`group relative p-2 mx-1 mb-1 rounded-lg cursor-pointer transition-all duration-200 hover:bg-white ${currentConversation?.id === conv.id ? 'bg-white shadow-sm' : ''
                        }`}
                    >
                      <div className="flex items-center justify-between">
                        <div className="flex-1 min-w-0">
                          <h3 className="font-medium text-gray-900 text-sm truncate">
                            {conv.title || "Untitled Chat"}
                          </h3>
                        </div>
                        <div className="flex items-center space-x-1 ml-2 opacity-0 group-hover:opacity-100 transition-opacity">
                          <button
                            onClick={(e) => {
                              e.stopPropagation();
                              starConversation(conv.id);
                            }}
                            className={`text-sm p-1 rounded hover:bg-gray-100 transition-colors ${conv.starred ? 'text-yellow-500' : 'text-gray-500 hover:text-yellow-500'
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
                  ))}

                  {/* Show More/Less Button */}
                  {remainingRecentCount > 0 && (
                    <div className="px-2 mt-2">
                      <button
                        onClick={() => setShowAllRecent(!showAllRecent)}
                        className="w-full text-left px-2 py-2 text-xs text-gray-500 hover:text-gray-700 hover:bg-gray-100 rounded-lg transition-colors flex items-center justify-between"
                      >
                        <span>
                          {showAllRecent
                            ? 'Show less'
                            : `Show ${remainingRecentCount} more`}
                        </span>
                        <svg
                          className={`w-3 h-3 transition-transform ${showAllRecent ? 'rotate-180' : ''}`}
                          fill="none"
                          stroke="currentColor"
                          viewBox="0 0 24 24"
                        >
                          <path strokeLinecap="round" strokeLinejoin="round" strokeWidth={2} d="M19 9l-7 7-7-7" />
                        </svg>
                      </button>
                    </div>
                  )}
                </div>
              )}
            </>
          )}
        </div>
      </div>

      {/* Chat Area */}
      <div className="flex-1 flex flex-col">
        {currentConversation ? (
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
                <div>
                  <h2 className="text-lg font-semibold text-gray-900">
                    {currentConversation.title || "Untitled Chat"}
                  </h2>
                  <p className="text-xs text-gray-500 flex items-center gap-2">
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

                  <div className={`flex flex-col ${message.role === "user" ? "items-end w-full" : "flex-1"}`}>

                    <div className={`${getMessageStyle(message.role)} ${message.role === "tool" ? "" : "py-2 px-4 text-sm"}`}>
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


              <div ref={messagesEndRef} />
            </div>

            {/* Message Input */}
            <div className="bg-white border-t border-gray-200 p-6">
              {/* Model Selection */}
              <div className="mb-4">
                <div className="flex items-center gap-3 mb-3">
                  <label className="text-sm font-medium text-gray-700 flex items-center gap-2">
                    <svg className="w-4 h-4 text-gray-500" fill="none" stroke="currentColor" viewBox="0 0 24 24">
                      <path strokeLinecap="round" strokeLinejoin="round" strokeWidth={2} d="M9.75 17L9 20l-1 1h8l-1-1-.75-3M3 13h18M5 17h14a2 2 0 002-2V5a2 2 0 00-2-2H5a2 2 0 00-2 2v10a2 2 0 002 2z" />
                    </svg>
                    Model:
                  </label>
                  <select
                    value={selectedModel}
                    onChange={(e) => setSelectedModel(e.target.value)}
                    className="text-sm bg-white border border-gray-300 text-gray-900 rounded-lg px-3 py-2 focus:outline-none focus:ring-2 focus:ring-blue-500 focus:border-blue-500 min-w-[200px]"
                  >
                    {availableModels.map((model) => (
                      <option key={model.value} value={model.value}>
                        {model.label}
                      </option>
                    ))}
                  </select>
                </div>
              </div>

              <ChatInput
                value={messageInput}
                onChange={setMessageInput}
                onSubmit={sendMessage}
                onCardReference={handleCardReference}
                placeholder="Ask about your cards... Type @ to mention a card"
                disabled={isSending}
                isLoading={isSending}
                submitButtonText="Send"
                multiline={true}
              />
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
                  disabled={isLoading || !hasSubscription}
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