import React, { useEffect } from "react";
import { useAuth } from "../../contexts/AuthContext";
import { Card } from "../../models/Card";
import { useChatSidebarContext } from "../../contexts/ChatSidebarContext";
import { useChat } from "../../hooks/useChat";
import { ChatInterface } from "./ChatInterface";

interface ChatSidebarProps {
  card: Card;
}

export function ChatSidebar({ card }: ChatSidebarProps) {
  const { hasSubscription } = useAuth();
  const { setChatSidebarCard } = useChatSidebarContext();

  const chatHook = useChat();

  // Create a conversation when the card changes
  useEffect(() => {
    createNewConversationForCard();
  }, [card.id]);

  const createNewConversationForCard = async () => {
    try {
      const conversation = await chatHook.createNewConversation(`Chat about ${card.card_id} - ${card.title}`);

      // Send initial message with card context
      const initialMessage = `I want to chat about this card: [${card.card_id}] - ${card.title}`;
      await chatHook.sendMessageToConversation(conversation.id, initialMessage, [card.id.toString()]);
    } catch (error) {
      console.error("Failed to create conversation for card:", error);
    }
  };

  const handleCardClick = (cardPk: string) => {
    window.open(`/app/card/${encodeURIComponent(cardPk)}`, '_blank');
  };

  const handleTaskClick = (taskId: number) => {
    // Could implement task dialog here if needed
  };

  return (
    <div className="flex flex-col h-full bg-white">
      {/* Header */}
      <div className="bg-green-100 border-b border-green-200 p-3">
        <div className="flex items-center justify-between">
          <div className="flex items-center gap-2">
            <span className="text-xs font-medium uppercase tracking-wide text-green-700">
              Chat about: [{card.card_id}]
            </span>
            <span className="text-sm font-medium truncate text-green-800">
              {card.title}
            </span>
          </div>
          <button
            onClick={() => setChatSidebarCard(null)}
            className="text-green-600 hover:text-green-800 p-1 rounded"
          >
            <svg className="w-4 h-4" fill="none" stroke="currentColor" viewBox="0 0 24 24">
              <path strokeLinecap="round" strokeLinejoin="round" strokeWidth={2} d="M6 18L18 6M6 6l12 12" />
            </svg>
          </button>
        </div>
      </div>

      {!hasSubscription ? (
        <div className="flex-1 flex items-center justify-center p-4">
          <div className="text-center text-gray-500">
            <p className="mb-2">Chat is a Pro feature.</p>
            <p className="text-sm">Upgrade to Pro to chat about your cards.</p>
          </div>
        </div>
      ) : (
        <ChatInterface
          chatHook={chatHook}
          onCardClick={handleCardClick}
          onTaskClick={handleTaskClick}
          placeholder="Ask about this card..."
          compact={true}
          showModelDropdown={false}
        />
      )}
    </div>
  );
}