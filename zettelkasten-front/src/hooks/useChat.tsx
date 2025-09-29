import { useState, useEffect, useRef } from "react";
import {
  ChatConversation,
  ChatMessage,
  createConversation,
  getConversation,
  sendMessage as apiSendMessage,
  sendMessageStream,
  getConversationStatus,
  StreamEvent,
} from "../api/chat";
import { useChatContext } from "../contexts/ChatContext";

export interface UseChatOptions {
  onConversationChange?: (conversation: ChatConversation | null) => void;
  initialModel?: string;
  enableStreaming?: boolean;
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
  const [streamingMessageId, setStreamingMessageId] = useState<string | null>(null);

  const pollingIntervalRef = useRef<NodeJS.Timeout | null>(null);
  const streamingContentRef = useRef<string>("");
  const { setConversationId } = useChatContext();
  const enableStreaming = options.enableStreaming ?? true; // Default to enabled

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
    setError(null);

    try {
      if (enableStreaming) {
        // Use streaming
        streamingContentRef.current = "";
        let userMessage: ChatMessage | null = null;
        let assistantMessage: ChatMessage | null = null;

        await sendMessageStream(
          conversationId,
          message,
          (event: StreamEvent) => {
            switch (event.type) {
              case 'messages':
                // Initial messages received
                userMessage = event.data.user_message;
                assistantMessage = event.data.assistant_message;

                if (userMessage && assistantMessage) {
                  setMessages(prev => [...prev, userMessage!, assistantMessage!]);
                  setStreamingMessageId(assistantMessage.id);
                }
                break;

              case 'title':
                // Update conversation title
                if (currentConversation) {
                  setCurrentConversation({
                    ...currentConversation,
                    title: event.data.title
                  });
                }
                break;

              case 'content':
                // Append content delta
                streamingContentRef.current += event.data.delta;

                // Update the assistant message in the messages array
                if (assistantMessage) {
                  setMessages(prev => prev.map(msg =>
                    msg.id === assistantMessage!.id
                      ? { ...msg, content: streamingContentRef.current, status: 'processing' as const }
                      : msg
                  ));
                }
                break;

              case 'tool_call':
                // Tool call initiated - could show a loading indicator
                console.log('Tool call:', event.data.name, event.data.arguments);
                break;

              case 'tool_result':
                // Tool result received - add as a tool message
                const toolMessage: ChatMessage = {
                  id: `tool-${event.data.tool_call_id}`,
                  conversation_id: conversationId,
                  role: 'tool',
                  content: JSON.stringify(event.data.result),
                  tool_call_id: event.data.tool_call_id,
                  sequence_number: 0,
                  status: 'completed',
                  created_at: new Date().toISOString()
                };
                setMessages(prev => [...prev, toolMessage]);
                break;

              case 'error':
                setError(event.data.error);
                break;

              case 'done':
                // Streaming complete - mark assistant message as completed
                setStreamingMessageId(null);
                streamingContentRef.current = "";
                if (assistantMessage) {
                  setMessages(prev => prev.map(msg =>
                    msg.id === assistantMessage!.id
                      ? { ...msg, status: 'completed' as const }
                      : msg
                  ));
                }
                break;
            }
          },
          referencedCards?.length ? referencedCards : undefined,
          selectedModel
        );
      } else {
        // Use polling (legacy)
        await apiSendMessage(conversationId, message, referencedCards?.length ? referencedCards : undefined, selectedModel);
        await refreshMessages(conversationId);
        startPolling(conversationId);
      }
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