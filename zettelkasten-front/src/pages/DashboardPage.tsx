import React, { useEffect, useState } from "react";
import { CardList } from "../components/cards/CardList";
import { setDocumentTitle } from "../utils/title";
import { useAuth } from "../contexts/AuthContext";
import { semanticSearchCards } from "../api/cards";
import { PartialCard } from "../models/Card";

export function DashboardPage() {
  const [recentCards, setRecentCards] = useState<PartialCard[]>([]);
  const [unsortedCards, setUnsortedCards] = useState<PartialCard[]>([]);
  const { hasSubscription, isLoading } = useAuth();
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

  return (
    <div>
      {/* Main Content Section */}
      <div className="p-2">
        <div className="text-center">
          <div className="mb-8">
            <h1 className="text-3xl font-bold text-gray-900 mb-3 p-10">
              Welcome to Zettelgarden 🌱
            </h1>
            <p className="text-lg text-gray-600 max-w-full">
              Your personal space for growing ideas. Create cards, connect
              thoughts, and watch your knowledge garden flourish.
            </p>
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
