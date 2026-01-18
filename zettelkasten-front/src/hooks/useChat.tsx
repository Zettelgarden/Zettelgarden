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
  onConversationCreated?: (conversation: ChatConversation) => void;
  initialModel?: string;
  enableStreaming?: boolean;
}

export function useChat(options: UseChatOptions = {}) {
  const [currentConversation, setCurrentConversation] = useState<ChatConversation | null>(null);
  const [isDraftConversation, setIsDraftConversation] = useState(false);
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
  const activeStreamConversationRef = useRef<string | null>(null);
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
    // Clear active stream reference when creating new conversation
    activeStreamConversationRef.current = null;
    setStreamingMessageId(null);
    streamingContentRef.current = "";

    // Create a draft conversation locally without calling the API
    const draftConv: ChatConversation = {
      id: `draft-${Date.now()}`,
      user_id: 0, // Placeholder for draft conversations
      title: title || "",
      model: model || selectedModel,
      primary_card_id: primaryCardId || undefined,
      created_at: new Date().toISOString(),
      updated_at: new Date().toISOString(),
      starred: false
    };

    setCurrentConversation(draftConv);
    setIsDraftConversation(true);
    setMessages([]);
    setError(null);

    return draftConv;
  };

  const loadConversation = async (conversationId: string) => {
    console.log("conversation", conversationId)
    if (conversationId == "") {
      return
    }
    try {
      setIsLoading(true);

      // Stop any existing polling when switching conversations
      stopPolling();

      // Clear active stream reference to ignore events from old conversation
      activeStreamConversationRef.current = null;
      setStreamingMessageId(null);
      streamingContentRef.current = "";

      const data = await getConversation(conversationId);
      setCurrentConversation(data.conversation);
      setMessages(data.messages || []);
      setError(null);

      // Check if there are pending/processing messages and start polling if needed
      if (!data.messages) {
        return
      }
      const hasPendingOrProcessing = data.messages.some(msg =>
        msg.status === 'pending' || msg.status === 'processing'
      );
      if (hasPendingOrProcessing) {
        startPolling(conversationId);
      }
    } catch (error) {
      console.error("Failed to load conversation:", error);
      setError("Failed to load conversationz");
    } finally {
      setIsLoading(false);
    }
  };

  const sendMessageToConversation = async (conversationId: string, message: string, referencedCards?: string[]) => {
    setIsSending(true);
    setError(null);

    // Mark this conversation as having an active stream
    activeStreamConversationRef.current = conversationId;

    // Optimistically add user message immediately
    const optimisticUserMessage: ChatMessage = {
      id: `temp-user-${Date.now()}`,
      conversation_id: conversationId,
      role: 'user',
      content: message,
      sequence_number: 0,
      status: 'completed',
      created_at: new Date().toISOString()
    };

    const optimisticAssistantMessage: ChatMessage = {
      id: `temp-assistant-${Date.now()}`,
      conversation_id: conversationId,
      role: 'assistant',
      content: '',
      sequence_number: 0,
      status: 'processing',
      created_at: new Date().toISOString()
    };

    // Add optimistic messages immediately
    setMessages(prev => [...prev, optimisticUserMessage, optimisticAssistantMessage]);
    setStreamingMessageId(optimisticAssistantMessage.id);

    try {
      if (enableStreaming) {
        // Use streaming
        streamingContentRef.current = "";

        // Use refs to store message IDs to avoid closure issues
        const messageIdsRef = {
          userMessageId: null as string | null,
          assistantMessageId: optimisticAssistantMessage.id as string | null
        };

        let receivedMessages = false;
        let streamError = false;

        await sendMessageStream(
          conversationId,
          message,
          (event: StreamEvent) => {
            // Ignore events if we've switched to a different conversation
            if (activeStreamConversationRef.current !== conversationId) {
              console.log('Ignoring stream event for old conversation:', conversationId);
              return;
            }

            switch (event.type) {
              case 'messages':
                // Initial messages received - replace optimistic with real
                receivedMessages = true;
                const userMessage = event.data.user_message;
                const assistantMessage = event.data.assistant_message;

                if (userMessage && assistantMessage) {
                  messageIdsRef.userMessageId = userMessage.id;
                  messageIdsRef.assistantMessageId = assistantMessage.id;

                  // Replace optimistic messages with real ones
                  setMessages(prev =>
                    prev.map(msg => {
                      if (msg.id === optimisticUserMessage.id) return userMessage;
                      if (msg.id === optimisticAssistantMessage.id) return assistantMessage;
                      return msg;
                    })
                  );
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
                if (messageIdsRef.assistantMessageId) {
                  setMessages(prev => prev.map(msg =>
                    msg.id === messageIdsRef.assistantMessageId
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
                streamError = true;
                setError(event.data.error);
                break;

              case 'done':
                // Streaming complete - mark assistant message as completed
                setStreamingMessageId(null);
                streamingContentRef.current = "";
                if (messageIdsRef.assistantMessageId) {
                  setMessages(prev => prev.map(msg =>
                    msg.id === messageIdsRef.assistantMessageId
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

        // After streaming completes, refresh messages from server to ensure consistency
        // This is important as a fallback if streaming had issues
        if (!receivedMessages || streamError) {
          console.log('Stream did not receive messages or had error, refreshing from server');
          // Remove optimistic messages before refreshing
          setMessages(prev => prev.filter(msg =>
            msg.id !== optimisticUserMessage.id && msg.id !== optimisticAssistantMessage.id
          ));
        }
        await refreshMessages(conversationId);
      } else {
        // Use polling (legacy)
        // Remove optimistic messages before making API call
        setMessages(prev => prev.filter(msg =>
          msg.id !== optimisticUserMessage.id && msg.id !== optimisticAssistantMessage.id
        ));
        await apiSendMessage(conversationId, message, referencedCards?.length ? referencedCards : undefined, selectedModel);
        await refreshMessages(conversationId);
        startPolling(conversationId);
      }
    } catch (error) {
      console.error("Failed to send message:", error);
      setError("Failed to send message");

      // On error, remove optimistic messages and refresh from server
      setMessages(prev => prev.filter(msg =>
        msg.id !== optimisticUserMessage.id && msg.id !== optimisticAssistantMessage.id
      ));

      try {
        await refreshMessages(conversationId);
      } catch (refreshError) {
        console.error("Failed to refresh messages after error:", refreshError);
      }
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

    // If this is a draft conversation, we need to create it in the backend first
    let conversationId = currentConversation.id;
    if (isDraftConversation) {
      try {
        setIsLoading(true);

        const newConv = await createConversation({
          title: currentConversation.title,
          model: currentConversation.model,
          primary_card_id: currentConversation.primary_card_id
        });

        setCurrentConversation(newConv);
        setIsDraftConversation(false);
        setConversationId(newConv.id);
        conversationId = newConv.id;

        // Notify that a conversation was created from a draft
        if (options.onConversationCreated) {
          options.onConversationCreated(newConv);
        }

        // Refresh messages for the new conversation to ensure consistency
        await refreshMessages(conversationId);
      } catch (error) {
        console.error("Failed to create conversation:", error);
        setError("Failed to create conversation");
        return;
      } finally {
        setIsLoading(false);
      }
    }

    await sendMessageToConversation(conversationId, userMessage, cardIds.length > 0 ? cardIds : undefined);
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
    isDraftConversation,
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