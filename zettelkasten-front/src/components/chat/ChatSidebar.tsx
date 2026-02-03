import React, { useEffect, useState } from "react";
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
  const { setChatSidebarCard, setRefreshTrigger } = useUIState();

  const chatHook = useChat({
    initialModel: localStorage.getItem('chatSelectedModel') || "google/gemini-2.5-flash",
    enableStreaming: true,
  });
  const [initialized, setInitialized] = useState(false);

  // Initialize chat session for this card
  useEffect(() => {
    const initializeChat = async () => {
      try {
        // Try to load the most recent conversation for this card
        const conversations = await getConversations(card.id);

        if (conversations.length > 0) {
          // Load the most recent conversation
          await chatHook.loadConversation(conversations[0].id);
        } else {
          // Create a new conversation for this card
          const conversation = await chatHook.createNewConversation(
            `Chat about ${card.card_id} - ${card.title}`,
            undefined,
            card.id
          );

          // Set up the initial message
          const initialMessage = `I want to chat about this card: [${card.card_id}] - ${card.title}`;
          chatHook.setMessageInput(initialMessage);
          chatHook.handleCardReference([card.id.toString()]);
          await chatHook.sendMessage();
        }
        setInitialized(true);
      } catch (error) {
        console.error("Failed to initialize chat for card:", error);
        setInitialized(true);
      }
    };

    initializeChat();
  }, [card.id]);

  // Monitor chat messages for card operations and trigger refreshes
  useEffect(() => {
    const { messages } = chatHook;

    // Look for recent tool messages that indicate card operations
    const recentMessages = messages.slice(-10);

    recentMessages.forEach(message => {
      if (message.role === 'tool' && message.content) {
        try {
          const toolResult = JSON.parse(message.content);

          if (toolResult.operation === 'card_updated' || toolResult.operation === 'card_created') {
            const operationCardPK = toolResult.card_pk;

            // If the operation was on the current card, trigger a refresh
            if (operationCardPK === card.id) {
              console.log(`Card ${card.id} was ${toolResult.operation} via chat, triggering refresh`);
              setRefreshTrigger(card.id.toString());
            } else if (toolResult.operation === 'card_created') {
              console.log(`New card ${operationCardPK} was created via chat, refreshing current card ${card.id} for potential relationships`);
              setRefreshTrigger(card.id.toString());
            }
          }
        } catch (e) {
          // Ignore parsing errors for non-JSON tool results
        }
      }
    });
  }, [chatHook.messages, card.id, setRefreshTrigger]);

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
        <div className="flex items-center justify-between">
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
          />
        </div>
      )}
    </div>
  );
}
