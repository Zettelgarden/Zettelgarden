import { useState, useEffect, useRef } from "react";
import {
  ChatConversation,
  ChatMessage,
  createConversation,
  getConversation,
  sendMessage as apiSendMessage,
  getConversationStatus,
} from "../api/chat";
import { useChatContext } from "../contexts/ChatContext";

export interface UseChatOptions {
  onConversationChange?: (conversation: ChatConversation | null) => void;
  initialModel?: string;
}

export function useChat(options: UseChatOptions = {}) {
  const [currentConversation, setCurrentConversation] = useState<ChatConversation | null>(null);
  const [messages, setMessages] = useState<ChatMessage[]>([]);
  const [messageInput, setMessageInput] = useState("");
  const [isLoading, setIsLoading] = useState(false);
  const [isSending, setIsSending] = useState(false);
  const [error, setError] = useState<string | null>(null);
  const [selectedModel, setSelectedModel] = useState(() => {
    return options.initialModel || localStorage.getItem('chatSelectedModel') || "google/gemini-2.5-flash";
  });
  const [collapsedToolResults, setCollapsedToolResults] = useState<Set<string>>(new Set());
  const [referencedCards, setReferencedCards] = useState<string[]>([]);
  const [isPolling, setIsPolling] = useState(false);
  const [showModelDropdown, setShowModelDropdown] = useState(false);

  const pollingIntervalRef = useRef<NodeJS.Timeout | null>(null);
  const { setConversationId } = useChatContext();

  // Update model in localStorage when it changes
  useEffect(() => {
    localStorage.setItem('chatSelectedModel', selectedModel);
  }, [selectedModel]);

  // Reset collapsed state when messages change and add new tool results as collapsed by default
  useEffect(() => {
    const newCollapsedSet = new Set<string>();
    messages.forEach(msg => {
      if (msg.role === "tool") {
        newCollapsedSet.add(msg.id);
      }
    });
    setCollapsedToolResults(newCollapsedSet);
  }, [messages]);

  // Cleanup polling on unmount
  useEffect(() => {
    return () => {
      stopPolling();
    };
  }, []);

  // Notify parent when conversation changes
  useEffect(() => {
    if (options.onConversationChange) {
      options.onConversationChange(currentConversation);
    }
  }, [currentConversation, options.onConversationChange]);

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

  const createNewConversation = async (title: string = "", model?: string, primaryCardId?: number) => {
    try {
      setIsLoading(true);
      const newConv = await createConversation({
        title,
        model: model || selectedModel,
        primary_card_id: primaryCardId
      });

      setCurrentConversation(newConv);
      setMessages([]);
      setConversationId(newConv.id);
      setError(null);

      return newConv;
    } catch (error) {
      console.error("Failed to create conversation:", error);
      setError("Failed to create conversation");
      throw error;
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

  const sendMessageToConversation = async (conversationId: string, message: string, referencedCards?: string[]) => {
    setIsSending(true);

    try {
      await apiSendMessage(conversationId, message, referencedCards?.length ? referencedCards : undefined, selectedModel);
      await refreshMessages(conversationId);
      startPolling(conversationId);
    } catch (error) {
      console.error("Failed to send message:", error);
      setError("Failed to send message");
    } finally {
      setIsSending(false);
    }
  };

  const sendMessage = async (passedReferencedCards?: string[]) => {
    if (!messageInput.trim() || !currentConversation || isSending) return;

    const userMessage = messageInput.trim();
    const cardIds = passedReferencedCards || referencedCards;

    setMessageInput("");
    setReferencedCards([]);

    await sendMessageToConversation(currentConversation.id, userMessage, cardIds.length > 0 ? cardIds : undefined);
  };

  const handleCardReference = (cardIds: string[]) => {
    setReferencedCards(cardIds);
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

  return {
    // State
    currentConversation,
    messages,
    messageInput,
    isLoading,
    isSending,
    error,
    selectedModel,
    collapsedToolResults,
    referencedCards,
    isPolling,
    showModelDropdown,

    // Setters
    setCurrentConversation,
    setMessages,
    setMessageInput,
    setError,
    setSelectedModel,
    setReferencedCards,
    setShowModelDropdown,

    // Actions
    createNewConversation,
    loadConversation,
    sendMessage,
    sendMessageToConversation,
    handleCardReference,
    toggleToolResult,
    refreshMessages,
    startPolling,
    stopPolling,
  };
}