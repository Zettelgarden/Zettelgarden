import React from "react";
import { HeaderSubSection } from "../Header";
import { CardItem } from "./CardItem";
import { RelatedCard } from "../../models/Card";

interface RelatedCardsProps {
  relatedCards: RelatedCard[];
  onCardClick: (cardId: number) => void;
}

export function RelatedCards({ relatedCards, onCardClick }: RelatedCardsProps) {
  if (relatedCards.length === 0) {
    return null;
  }

  return (
    <div>
      <HeaderSubSection text="Related Cards" />
      <ul className="mt-2 space-y-1">
        {relatedCards.map((rc) => (
          <li key={rc.card.id} onClick={() => onCardClick(rc.card.id)} className="cursor-pointer">
            <div className="flex items-center justify-between">
              <div className="flex-1">
                <CardItem card={rc.card} />
              </div>
              {rc.score > 0 && <span className="text-xs text-gray-400 ml-2 shrink-0">{rc.score.toFixed(1)}</span>}
            </div>
          </li>
        ))}
      </ul>
      <hr className="my-4" />
    </div>
  );
}
