import React, { useEffect, useState, useRef } from "react";
import { useAuth } from "../../contexts/AuthContext";
import { Card } from "../../models/Card";
import { useUIState } from "../../contexts/UIStateContext";
import { useChat } from "../../hooks/useChat";
import { ChatInterface } from "./ChatInterface";
import { getConversations, ChatConversation } from "../../api/chat";

interface ChatSidebarProps {
  card: Card;
}

export function ChatSidebar({ card }: ChatSidebarProps) {
  const { hasSubscription } = useAuth();
  const { setChatSidebarCard, refreshCard } = useUIState();

  const chatHook = useChat({
    onConversationChange: (conversation) => {
      // Update selected conversation ID whenever the current conversation changes
      if (conversation) {
        setSelectedConversationId(conversation.id);
      }
    },
    onConversationCreated: async (conversation) => {
      // Refresh the conversation list after a draft is converted to real conversation
      const conversations = await getConversations(card.id);
      setCardConversations(conversations);
    }
  });
  const [cardConversations, setCardConversations] = useState<ChatConversation[]>([]);
  const [selectedConversationId, setSelectedConversationId] = useState<string | null>(null);
  const [isLoadingConversations, setIsLoadingConversations] = useState(true);
  const [showConversationMenu, setShowConversationMenu] = useState(false);
  const menuRef = useRef<HTMLDivElement>(null);

  // Load existing conversations for this card
  useEffect(() => {
    const loadConversations = async () => {
      try {
        setIsLoadingConversations(true);
        const conversations = await getConversations(card.id);
        setCardConversations(conversations);

        // If there are existing conversations, load the most recent one
        if (conversations.length > 0 && !selectedConversationId) {
          await loadExistingConversation(conversations[0].id);
          setSelectedConversationId(conversations[0].id);
        } else if (conversations.length === 0) {
          // Only create a new conversation if there are no existing ones
          await createNewConversationForCard();
        }
      } catch (error) {
        console.error("Failed to load conversations for card:", error);
        // Fall back to creating a new conversation
        await createNewConversationForCard();
      } finally {
        setIsLoadingConversations(false);
      }
    };

    loadConversations();
  }, [card.id]);

  // Close menu when clicking outside
  useEffect(() => {
    const handleClickOutside = (event: MouseEvent) => {
      if (menuRef.current && !menuRef.current.contains(event.target as Node)) {
        setShowConversationMenu(false);
      }
    };

    if (showConversationMenu) {
      document.addEventListener("mousedown", handleClickOutside);
      return () => {
        document.removeEventListener("mousedown", handleClickOutside);
      };
    }
  }, [showConversationMenu]);

  // Monitor chat messages for card operations and trigger refreshes
  useEffect(() => {
    const { messages } = chatHook;

    // Look for recent tool messages that indicate card operations
    const recentMessages = messages.slice(-10); // Check last 10 messages

    recentMessages.forEach(message => {
      if (message.role === 'tool' && message.content) {
        try {
          const toolResult = JSON.parse(message.content);

          // Check if this is a card operation
          if (toolResult.operation === 'card_updated' || toolResult.operation === 'card_created') {
            const operationCardPK = toolResult.card_pk;

            // If the operation was on the current card, trigger a refresh
            if (operationCardPK === card.id) {
              console.log(`Card ${card.id} was ${toolResult.operation} via chat, triggering refresh`);
              refreshCard(card.id.toString());
            } else if (toolResult.operation === 'card_created') {
              // For new cards, always refresh the current card as it might affect relationships
              console.log(`New card ${operationCardPK} was created via chat, refreshing current card ${card.id} for potential relationships`);
              refreshCard(card.id.toString());
            }
          }
        } catch (e) {
          // Ignore parsing errors for non-JSON tool results
        }
      }
    });
  }, [chatHook.messages, card.id, refreshCard]);

  const loadExistingConversation = async (conversationId: string) => {
    try {
      await chatHook.loadConversation(conversationId);
    } catch (error) {
      console.error("Failed to load conversation:", error);
    }
  };

  const createNewConversationForCard = async () => {
    try {
      const conversation = await chatHook.createNewConversation(
        `Chat about ${card.card_id} - ${card.title}`,
        undefined, // use default model
        card.id    // primary card ID
      );

      setSelectedConversationId(conversation.id);

      // Set up the initial message using the hook's sendMessage mechanism
      // This properly handles draft conversations by creating them on the backend first
      const initialMessage = `I want to chat about this card: [${card.card_id}] - ${card.title}`;
      chatHook.setMessageInput(initialMessage);
      chatHook.handleCardReference([card.id.toString()]);

      // Send the message - this will create the conversation on the backend
      // The onConversationCreated callback will handle updating the conversation ID
      await chatHook.sendMessage();
    } catch (error) {
      console.error("Failed to create conversation for card:", error);
    }
  };

  const handleConversationChange = async (conversationId: string) => {
    setShowConversationMenu(false);
    if (conversationId === "new") {
      await createNewConversationForCard();
    } else {
      setSelectedConversationId(conversationId);
      await loadExistingConversation(conversationId);
    }
  };

  const getSelectedConversation = () => {
    return cardConversations.find(c => c.id === selectedConversationId);
  };

  const handleCardClick = (cardPk: string) => {
    window.open(`/app/card/${encodeURIComponent(cardPk)}`, '_blank');
  };

  const handleTaskClick = (taskId: number) => {
    // Could implement task dialog here if needed
  };

  return (
    <div className="flex flex-col h-full bg-white">
      {/* Fixed Header */}
      <div className="flex-shrink-0 bg-white rounded-lg p-3 shadow-sm border-b border-gray-200">
        <div className="flex items-center justify-between mb-2">
          <div className="flex items-center gap-2 flex-1 min-w-0">
            <span className="font-bold text-gray-600 whitespace-nowrap">
              Chat:
            </span>
            <span className="text-blue-600">
              [{card.card_id}]
            </span>
            <span className="text-gray-600 truncate">
              - {card.title}
            </span>
          </div>
          <button
            onClick={() => setChatSidebarCard(null)}
            className="text-gray-500 hover:text-gray-700 p-1 rounded flex-shrink-0"
            title="Close chat"
          >
            <svg className="w-5 h-5" fill="none" stroke="currentColor" viewBox="0 0 24 24">
              <path strokeLinecap="round" strokeLinejoin="round" strokeWidth={2} d="M6 18L18 6M6 6l12 12" />
            </svg>
          </button>
        </div>

        {/* Conversation Selector */}
        {!isLoadingConversations && hasSubscription && (
          <div className="flex gap-2 justify-end relative" ref={menuRef}>
            <button
              onClick={() => setShowConversationMenu(!showConversationMenu)}
              className="inline-flex justify-center rounded-md border border-gray-300 shadow-sm px-2 py-2 bg-white text-sm font-medium text-gray-700 hover:bg-gray-50 focus:outline-none focus:ring-2 focus:ring-offset-2 focus:ring-indigo-500 transition-colors flex-shrink-0"
              title="Switch conversation"
            >
              <svg className="w-5 h-5" fill="none" stroke="currentColor" viewBox="0 0 24 24">
                <path strokeLinecap="round" strokeLinejoin="round" strokeWidth={2} d="M4 6h16M4 12h16M4 18h16" />
              </svg>
            </button>

            <button
              onClick={() => handleConversationChange("new")}
              className="inline-flex items-center gap-1.5 rounded-md border border-gray-300 shadow-sm px-3 py-2 bg-white text-sm font-medium text-gray-700 hover:bg-gray-50 focus:outline-none focus:ring-2 focus:ring-offset-2 focus:ring-indigo-500 transition-colors flex-shrink-0"
              title="Start new conversation"
            >
              <svg className="w-4 h-4" fill="none" stroke="currentColor" viewBox="0 0 24 24">
                <path strokeLinecap="round" strokeLinejoin="round" strokeWidth={2} d="M12 4v16m8-8H4" />
              </svg>
              New Chat
            </button>

            {/* Popup Menu */}
            {showConversationMenu && (
              <div className="absolute top-full right-0 mt-2 bg-white border border-gray-200 rounded-lg shadow-lg z-50 max-h-80 overflow-y-auto min-w-[320px]">
                {cardConversations.length > 0 ? (
                  cardConversations.map((conv) => (
                    <button
                      key={conv.id}
                      onClick={() => handleConversationChange(conv.id)}
                      className={`w-full text-left px-4 py-3 hover:bg-gray-100 border-b border-gray-100 last:border-b-0 transition-colors ${
                        conv.id === selectedConversationId ? "bg-gray-50" : ""
                      }`}
                    >
                      <div className="flex items-start justify-between gap-2">
                        <div className="flex-1 min-w-0">
                          <div className="text-sm font-medium text-gray-900 truncate">
                            {conv.title || "Untitled conversation"}
                          </div>
                          <div className="text-xs text-gray-500 mt-1 flex items-center gap-2">
                            <span>{conv.message_count || 0} messages</span>
                            <span>•</span>
                            <span>{new Date(conv.updated_at).toLocaleDateString()}</span>
                          </div>
                        </div>
                        {conv.id === selectedConversationId && (
                          <svg className="w-5 h-5 text-blue-600 flex-shrink-0" fill="currentColor" viewBox="0 0 20 20">
                            <path fillRule="evenodd" d="M16.707 5.293a1 1 0 010 1.414l-8 8a1 1 0 01-1.414 0l-4-4a1 1 0 011.414-1.414L8 12.586l7.293-7.293a1 1 0 011.414 0z" clipRule="evenodd" />
                          </svg>
                        )}
                      </div>
                    </button>
                  ))
                ) : (
                  <div className="px-4 py-3 text-sm text-gray-500 text-center">
                    No conversations yet
                  </div>
                )}
              </div>
            )}
          </div>
        )}
      </div>

      {!hasSubscription ? (
        <div className="flex-1 flex items-center justify-center p-4 overflow-hidden">
          <div className="text-center text-gray-500">
            <p className="mb-2">Chat is a Pro feature.</p>
            <p className="text-sm">Upgrade to Pro to chat about your cards.</p>
          </div>
        </div>
      ) : (
        <div className="flex-1 overflow-hidden">
          <ChatInterface
            chatHook={chatHook}
            onCardClick={handleCardClick}
            onTaskClick={handleTaskClick}
            placeholder="Ask about this card..."
            compact={true}
            showModelDropdown={true}
          />
        </div>
      )}
    </div>
  );
}