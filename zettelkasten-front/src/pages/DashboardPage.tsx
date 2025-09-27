import React, { useEffect, useState } from "react";
import { CardList } from "../components/cards/CardList";
import { setDocumentTitle } from "../utils/title";
import { useAuth } from "../contexts/AuthContext";
import { semanticSearchCardsPaginated, getUnsortedCards } from "../api/cards";
import { PartialCard } from "../models/Card";
import { ChatInput } from "../components/chat/ChatInput";

export function DashboardPage() {
  const [recentCards, setRecentCards] = useState<PartialCard[]>([]);
  const [unsortedCards, setUnsortedCards] = useState<PartialCard[]>([]);
  const [chatInput, setChatInput] = useState("");
  const [isLoadingCards, setIsLoadingCards] = useState<boolean>(true);
  const { hasSubscription, isLoading } = useAuth();
  const subscriptionEnabled =
    import.meta.env.VITE_FEATURE_SUBSCRIPTION === "true";

  useEffect(() => {
    setDocumentTitle("Index");

    const fetchDashboardData = async () => {
      setIsLoadingCards(true);
      try {
        // Fetch recent cards using pagination (only get 10)
        const recentResponse = await semanticSearchCardsPaginated(
          "", // empty search term
          false, // fullText
          false, // showEntities
          false, // showFacts
          true, // showCards
          "sortCreatedNewOld", // sortBy recent
          "classic", // searchType
          false, // rerank
          1, // page
          10 // perPage - only get 10 cards
        );

        const recentCards: PartialCard[] = recentResponse.results.map((result) => ({
          id: Number(result.metadata?.id) || 0,
          card_id: result.metadata?.card_id || "",
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
        setRecentCards(recentCards);

        // Fetch unsorted cards using the dedicated API endpoint
        const unsortedResponse = await getUnsortedCards(1, 10);
        setUnsortedCards(unsortedResponse.cards);

      } catch (error) {
        console.error("Error fetching dashboard data:", error);
        // Set empty arrays on error to prevent infinite loading
        setRecentCards([]);
        setUnsortedCards([]);
      } finally {
        setIsLoadingCards(false);
      }
    };

    fetchDashboardData();
  }, []);

  const handleChatSubmit = async (referencedCards?: string[]) => {
    if (!chatInput.trim()) return;

    const messageToSend = chatInput.trim();

    // Navigate to chat page with message and referenced cards as URL params
    const params = new URLSearchParams();
    params.set('message', messageToSend);
    if (referencedCards && referencedCards.length > 0) {
      params.set('cards', referencedCards.join(','));
    }

    window.location.href = `/app/chat?${params.toString()}`;
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

            {/* Quick Chat Box - only show for subscribers */}
            {hasSubscription && (
              <div className="max-w-4xl mx-auto mb-8">
                <div className="relative">
                  {/* Unified Input Container */}
                  <div className="relative border border-gray-300 rounded-2xl bg-white shadow-sm hover:shadow-md transition-all duration-200 focus-within:border-blue-500 focus-within:ring-2 focus-within:ring-blue-500/20">
                    {/* Input Area with Controls */}
                    <div className="flex items-center gap-3 p-4">
                      {/* Main Input */}
                      <div className="flex-1 relative">
                        <ChatInput
                          value={chatInput}
                          onChange={setChatInput}
                          onSubmit={handleChatSubmit}
                          onCardReference={handleCardReference}
                          placeholder="Ask your knowledge base anything..."
                          disabled={false}
                          isLoading={false}
                          submitButtonText=""
                          multiline={false}
                          className="border-0 rounded-none p-0"
                        />
                      </div>

                      {/* Send Button */}
                      <button
                        onClick={() => handleChatSubmit()}
                        disabled={!chatInput.trim()}
                        className="p-2.5 bg-black hover:bg-gray-800 text-white rounded-xl transition-all duration-200 disabled:opacity-50 disabled:cursor-not-allowed disabled:hover:bg-black flex items-center justify-center min-w-[44px]"
                      >
                        <svg className="w-4 h-4" fill="none" stroke="currentColor" viewBox="0 0 24 24">
                          <path strokeLinecap="round" strokeLinejoin="round" strokeWidth={2} d="M12 19l9 2-9-18-9 18 9-2zm0 0v-8" />
                        </svg>
                      </button>
                    </div>

                    {/* Helper text at bottom */}
                    <div className="px-4 pb-3">
                      <div className="flex items-center justify-between text-xs text-gray-500">
                        <div className="flex items-center gap-2">
                          <div className="w-1.5 h-1.5 bg-green-500 rounded-full"></div>
                          <span>Ask anything about your knowledge base</span>
                        </div>
                        <div className="text-gray-400">
                          Press Enter to search
                        </div>
                      </div>
                    </div>
                  </div>
                </div>
              </div>
            )}
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

        <div className="flex-shrink-0 md:w-8/12 border-r p-4 overflow-hidden">
          <a href="/app/search?recent=true">
            <span className="font-bold">Recent Cards</span>
          </a>
          {isLoadingCards ? (
            <div className="flex justify-center py-8">
              <div className="text-gray-500">Loading recent cards...</div>
            </div>
          ) : (
            <CardList sort={false} cards={recentCards} />
          )}
          <hr />
        </div>

        {/* Right Section */}
        <div className="flex-shrink-0 md:w-4/12 border-l p-4 overflow-hidden">
          <div>
            <span className="font-bold">Unsorted Cards</span>
            {isLoadingCards ? (
              <div className="flex justify-center py-8">
                <div className="text-gray-500">Loading unsorted cards...</div>
              </div>
            ) : (
              <CardList cards={unsortedCards} />
            )}
          </div>
          <hr />
        </div>
      </div>
    </div>
  );
}
