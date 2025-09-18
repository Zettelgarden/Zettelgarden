import React, { useEffect, useState } from "react";
import { CardList } from "../components/cards/CardList";
import { setDocumentTitle } from "../utils/title";
import { useAuth } from "../contexts/AuthContext";
import { semanticSearchCards } from "../api/cards";
import { PartialCard } from "../models/Card";
import { createConversation, sendMessage } from "../api/chat";
import { useChatContext } from "../contexts/ChatContext";
import { ChatInput } from "../components/chat/ChatInput";

export function DashboardPage() {
  const [recentCards, setRecentCards] = useState<PartialCard[]>([]);
  const [unsortedCards, setUnsortedCards] = useState<PartialCard[]>([]);
  const [chatInput, setChatInput] = useState("");
  const [isCreatingChat, setIsCreatingChat] = useState(false);
  const { hasSubscription, isLoading } = useAuth();
  const { setConversationId } = useChatContext();
  const subscriptionEnabled =
    import.meta.env.VITE_FEATURE_SUBSCRIPTION === "true";

  useEffect(() => {
    setDocumentTitle("Index");

    // Fetch recent cards
    semanticSearchCards("", false, false, false, "sortByDate").then((results) => {
      const cards: PartialCard[] = results.map((result) => ({
        id: Number(result.metadata?.id) || 0,
        card_id: result.metadata.card_id,
        title: result.title,
        body: result.preview || "",
        tags: result.tags || [],
        is_deleted: false,
        created_at: new Date(result.created_at),
        updated_at: new Date(result.updated_at),
        parent_id: result.metadata?.parent_id || 0,
        user_id: 0,
        link: "",
        parent: null,
      }));
      setRecentCards(cards.slice(0, 10));
      const unsorted = cards.filter((card) => card.card_id === "");
      setUnsortedCards(unsorted.slice(0, 10));
    });
  }, []);

  const handleChatSubmit = async (referencedCards?: string[]) => {
    if (!chatInput.trim() || isCreatingChat) return;

    const messageToSend = chatInput.trim();
    setIsCreatingChat(true);

    try {
      // Create a new conversation
      const conversation = await createConversation({
        title: "", // Will be auto-generated from the first message
        model: "gpt-4o-mini"
      });

      // Set the conversation ID in context and navigate immediately
      setConversationId(conversation.id);

      // Navigate to chat page right away
      window.location.href = "/app/chat";

      // Send the initial message asynchronously (this will happen in the background)
      sendMessage(conversation.id, messageToSend, referencedCards).catch(error => {
        console.error("Failed to send initial message:", error);
      });

    } catch (error) {
      console.error("Failed to create chat:", error);
      alert("Failed to start chat. Please try again.");
      setIsCreatingChat(false);
    }
  };

  const handleCardReference = (cardIds: string[]) => {
    // Cards are handled directly by ChatInput component
  };

  return (
    <div>
      {/* Main Content Section */}
      <div className="p-2">
        <div className="">
          <div className="mb-8">
            <div className="text-center">

            <h1 className="text-3xl font-bold text-gray-900 mb-3 p-10">
              Welcome to Zettelgarden 🌱
            </h1>
            <p className="text-lg text-gray-600 max-w-full mb-8">
              Your personal space for growing ideas. Create cards, connect
              thoughts, and watch your knowledge garden flourish.
            </p>
            </div>

            {/* Quick Chat Box */}
            <div className="max-w-4xl mx-auto mb-8">
              <div className="bg-white rounded-2xl shadow-lg border border-gray-200 p-6">
                <ChatInput
                  value={chatInput}
                  onChange={setChatInput}
                  onSubmit={handleChatSubmit}
                  onCardReference={handleCardReference}
                  placeholder="Ask your knowledge base anything..."
                  disabled={isCreatingChat}
                  isLoading={isCreatingChat}
                  submitButtonText="Chat"
                  multiline={false}
                />
              </div>
            </div>
          </div>
          {!isLoading && !hasSubscription && subscriptionEnabled && (
            <div className="bg-blue-50 border-t border-b border-blue-200 text-blue-800 px-4 py-3 text-center">
              <p>
                <strong>Unlock powerful AI features!</strong> Subscribe now to
                enable summarization, entity extraction, and fact analysis.
                <a
                  href="/app/subscription"
                  className="ml-2 bg-blue-500 text-white font-bold py-1 px-3 rounded hover:bg-blue-600"
                >
                  Upgrade
                </a>
              </p>
            </div>
          )}
        </div>
      </div>
      <div className="flex flex-col md:flex-row border-t">
        {/* Left Section */}

        <div className="flex-grow md:w-8/12 border-r p-4">
          <a href="/app/search?recent=true">
            <span className="font-bold">Recent Cards</span>
          </a>
          <CardList sort={false} cards={recentCards} />
          <hr />
        </div>

        {/* Right Section */}
        <div className="flex-shrink-0 md:w-4/12 border-l p-4">
          <div>
            <span className="font-bold">Unsorted Cards</span>
            <CardList cards={unsortedCards} />
          </div>
          <hr />
        </div>
      </div>
    </div>
  );
}
