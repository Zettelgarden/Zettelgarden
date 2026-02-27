import React, { useEffect, useState } from "react";
import { useNavigate } from "react-router-dom";
import { CardList } from "../components/cards/CardList";
import { setDocumentTitle } from "../utils/title";
import { useAuth } from "../contexts/AuthContext";
import { useUIState } from "../contexts/UIStateContext";
import { semanticSearchCardsPaginated, getUnsortedCards } from "../api/cards";
import { PartialCard } from "../models/Card";
import { ChatInput } from "../components/chat/ChatInput";
import { AddArticleDialog } from "../components/cards/AddArticleDialog";
import { useDialogState } from "../contexts/DialogStateContext";
import { MobileTopBar } from "../components/layout/MobileTopBar";

export function DashboardPage() {
  const navigate = useNavigate();
  const { toggleMobileSidebar } = useUIState();
  const [recentCards, setRecentCards] = useState<PartialCard[]>([]);
  const [unsortedCards, setUnsortedCards] = useState<PartialCard[]>([]);
  const [chatInput, setChatInput] = useState("");
  const [isLoadingCards, setIsLoadingCards] = useState<boolean>(true);
  const { hasSubscription, isLoading } = useAuth();
  const [showAddArticleDialog, setShowAddArticleDialog] = useState(false);
  const { setShowCreateTaskWindow } = useDialogState();

  const setMessage = (message: string) => {
    console.log("Message:", message);
    // TODO: Could integrate with a toast notification system here
  };
  const subscriptionEnabled =
    import.meta.env.VITE_FEATURE_SUBSCRIPTION === "true";

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
        false, // showEmails
        "sortCreatedNewOld", // sortBy recent
        "classic", // searchType
        false, // rerank
        1, // page
        10 // perPage - only get 10 cards
      );
      console.log("recent", recentResponse)

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
      console.log("recent", recentCards)
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

  useEffect(() => {
    setDocumentTitle("Index");
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

    navigate(`/app/chat?${params.toString()}`);
  };

  const handleCardReference = (cardIds: string[]) => {
    // Cards are handled directly by ChatInput component
  };

  const handleNewStandardCard = () => {
    navigate("/app/card/new", { state: { cardType: "standard" } });
  };

  const handleNewChat = () => {
    navigate("/app/chat?new=true");
  };

  const handleNewTask = () => {
    setShowCreateTaskWindow(true);
  };

  const handleAddArticle = () => {
    setShowAddArticleDialog(true);
  };

  return (
    <div>
      {/* Mobile Top Bar */}
      <MobileTopBar
        title="Dashboard"
        onMenuClick={toggleMobileSidebar}
      />

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


          {/* Mobile Quick Actions - Only visible on small screens */}
          <div className="md:hidden border-t border-gray-200 bg-gray-50 px-4 py-6">
            <div className="grid grid-cols-4 gap-3">
              <button
                onClick={handleNewStandardCard}
                className="flex flex-col items-center p-3 bg-white rounded-lg shadow-sm hover:shadow-md hover:bg-gray-50 border border-gray-200 transition-all duration-200"
              >
                <svg className="w-6 h-6 text-blue-600 mb-1" fill="none" stroke="currentColor" viewBox="0 0 24 24">
                  <path strokeLinecap="round" strokeLinejoin="round" strokeWidth={2} d="M12 4v16m8-8H4" />
                </svg>
                <span className="text-xs font-medium text-gray-700">Card</span>
              </button>

              <button
                onClick={handleAddArticle}
                className="flex flex-col items-center p-3 bg-white rounded-lg shadow-sm hover:shadow-md hover:bg-gray-50 border border-gray-200 transition-all duration-200"
              >
                <svg className="w-6 h-6 text-green-600 mb-1" fill="none" stroke="currentColor" viewBox="0 0 24 24">
                  <path strokeLinecap="round" strokeLinejoin="round" strokeWidth={2} d="M9 12h6m-6 4h6m2 5H7a2 2 0 01-2-2V5a2 2 0 012-2h5.586a1 1 0 01.707.293l5.414 5.414a1 1 0 01.293.707V19a2 2 0 01-2 2z" />
                </svg>
                <span className="text-xs font-medium text-gray-700">Article</span>
              </button>

              <button
                onClick={handleNewTask}
                className="flex flex-col items-center p-3 bg-white rounded-lg shadow-sm hover:shadow-md hover:bg-gray-50 border border-gray-200 transition-all duration-200"
              >
                <svg className="w-6 h-6 text-orange-600 mb-1" fill="none" stroke="currentColor" viewBox="0 0 24 24">
                  <path strokeLinecap="round" strokeLinejoin="round" strokeWidth={2} d="M9 5H7a2 2 0 00-2 2v10a2 2 0 002 2h8a2 2 0 002-2V7a2 2 0 00-2-2h-2M9 5a2 2 0 002 2h2a2 2 0 002-2M9 5a2 2 0 012-2h2a2 2 0 012 2m-6 9l2 2 4-4" />
                </svg>
                <span className="text-xs font-medium text-gray-700">Task</span>
              </button>

              {hasSubscription && (
                <button
                  onClick={handleNewChat}
                  className="flex flex-col items-center p-3 bg-white rounded-lg shadow-sm hover:shadow-md hover:bg-gray-50 border border-gray-200 transition-all duration-200"
                >
                  <svg className="w-6 h-6 text-purple-600 mb-1" fill="none" stroke="currentColor" viewBox="0 0 24 24">
                    <path strokeLinecap="round" strokeLinejoin="round" strokeWidth={2} d="M8 12h.01M12 12h.01M16 12h.01M21 12c0 4.418-4.03 8-9 8a9.863 9.863 0 01-4.255-.949L3 20l1.395-3.72C3.512 15.042 3 13.574 3 12c0-4.418 4.03 8 9 8s9 3.582 9 8z" />
                  </svg>
                  <span className="text-xs font-medium text-gray-700">Chat</span>
                </button>
              )}

              {!hasSubscription && (
                <button
                  onClick={() => window.location.href = "/app/subscription"}
                  className="flex flex-col items-center p-3 bg-gray-100 rounded-lg shadow-sm hover:shadow-md cursor-not-allowed border border-gray-200 transition-all duration-200"
                  disabled
                >
                  <svg className="w-6 h-6 text-gray-400 mb-1" fill="none" stroke="currentColor" viewBox="0 0 24 24">
                    <path strokeLinecap="round" strokeLinejoin="round" strokeWidth={2} d="M12 15v2m-6 4h12a2 2 0 002-2v-6a2 2 0 00-2-2H6a2 2 0 00-2 2v6a2 2 0 002 2zm10-10V7a4 4 0 00-8 0v4h8z" />
                  </svg>
                  <span className="text-xs font-medium text-gray-400">Chat</span>
                </button>
              )}
            </div>
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
            <CardList sort={false} cards={recentCards} onCardUpdate={fetchDashboardData} />
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
              <CardList cards={unsortedCards} onCardUpdate={fetchDashboardData} />
            )}
          </div>
          <hr />
        </div>
      </div>

      {/* Modal dialogs */}
      <AddArticleDialog
        show={showAddArticleDialog}
        onClose={() => setShowAddArticleDialog(false)}
      />
    </div>
  );
}
