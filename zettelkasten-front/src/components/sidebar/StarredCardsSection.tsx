import React, { useState, useEffect } from "react";
import { Card } from "../../models/Card";
import { CardItem } from "../cards/CardItem";
import { getStarredCards, unstarCard } from "../../api/cards";
import { useLocation } from "react-router-dom";
import { useToast } from "../toast/ToastContext";

interface StarredCardsSectionProps {
  onShowStarCardDialog: () => void;
}

export function StarredCardsSection({ onShowStarCardDialog }: StarredCardsSectionProps) {
  const [starredCards, setStarredCards] = useState<Card[]>([]);
  const location = useLocation();
  const { showToast } = useToast();

  const handleUnstarCard = (cardId: number) => {
    unstarCard(cardId)
      .then(() => {
        // Refresh the starred cards list after unstarring
        refreshStarredCards();
        // Show a success message
        showToast("success", "Card unstarred successfully");
      })
      .catch(error => {
        console.error("Error unstarring card:", error);
        showToast("error", "Failed to unstar card", "Please try again");
      });
  };

  const refreshStarredCards = () => {
    getStarredCards()
      .then((cards) => {
        setStarredCards(cards);
      })
      .catch(error => {
        console.error("Error fetching starred cards:", error);
      });
  };

  useEffect(() => {
    refreshStarredCards();
  }, []);

  // Refresh starred items when location changes (navigation occurs)
  useEffect(() => {
    refreshStarredCards();
  }, [location.pathname]);

  return (
    <>
      <hr />
      <div className="p-2">
        <div className="flex items-center justify-between mb-2 px-2">
          <h3 className="text-xs font-semibold text-gray-500 uppercase tracking-wider">
            Starred Cards
          </h3>
          <button
            onClick={onShowStarCardDialog}
            className="min-w-[44px] min-h-[44px] md:min-w-0 md:min-h-0 flex items-center justify-center text-gray-400 hover:text-blue-500 rounded-full"
            title="Star a card"
          >
            +
          </button>
        </div>
        {starredCards.length > 0 ? (
          <ul className="space-y-0.5">
            {starredCards.map((card) => (
              <li key={card.id} className="text-sm group relative">
                <div className="flex items-center">
                  <div className="flex-grow min-w-0">
                    <CardItem card={card} />
                  </div>
                  <button
                    onClick={() => handleUnstarCard(card.id)}
                    className="absolute right-2 text-gray-400 hover:text-red-500 opacity-0 group-hover:opacity-100 transition-opacity min-w-[44px] min-h-[44px] md:min-w-0 md:min-h-0 flex items-center justify-center"
                    aria-label={`Unstar "${card.title}"`}
                    title="Unstar card"
                  >
                    ×
                  </button>
                </div>
              </li>
            ))}
          </ul>
        ) : (
          <p className="text-xs text-gray-400 px-2">No starred cards yet</p>
        )}
      </div>
    </>
  );
}