import React, { useState, useEffect } from "react";
import { SearchResultList } from "../../components/cards/SearchResultList";
import { CardListItem } from "./CardListItem";
import { TriangleDownIcon } from "../../assets/icons/TriangleDown";
import { TriangleRightIcon } from "../../assets/icons/TriangleRight";
import { getCardChildren } from "../../api/cards";

import { Card, PartialCard } from "../../models/Card";

interface ChildrenCardsProps {
  allChildren: PartialCard[];
  card: PartialCard;
}

export function ChildrenCards({ allChildren, card }: ChildrenCardsProps) {
  const [openCards, setOpenCards] = useState<Record<string, boolean>>({});
  const [childCards, setChildCards] = useState<PartialCard[]>([]);
  const [loadedChildren, setLoadedChildren] = useState<Record<string, PartialCard[]>>({});
  const [loading, setLoading] = useState<Record<string, boolean>>({});

  async function handleIconClick(cardId: string, id: number) {
    const wasOpen = openCards[cardId];

    setOpenCards((prevOpenCards) => ({
      ...prevOpenCards,
      [cardId]: !prevOpenCards[cardId],
    }));

    // If opening the card and we haven't loaded its children yet, load them
    if (!wasOpen && !loadedChildren[cardId]) {
      setLoading((prev) => ({ ...prev, [cardId]: true }));
      try {
        const children = await getCardChildren(id.toString());
        setLoadedChildren((prev) => ({ ...prev, [cardId]: children }));
      } catch (error) {
        console.error('Failed to load children for card:', cardId, error);
        setLoadedChildren((prev) => ({ ...prev, [cardId]: [] }));
      } finally {
        setLoading((prev) => ({ ...prev, [cardId]: false }));
      }
    }
  }

  useEffect(() => {
    let cards = allChildren.filter((c) => c.parent_id === card.id);
    setChildCards(cards);
  }, [allChildren, card]);

  return (
    <div className="w-full">
      <ul>
        {childCards.map((c, index) => (
          <li key={index} className="flex flex-col">
            <div className="flex items-center">
              <span
                className="mr-2 cursor-pointer"
                onClick={() => handleIconClick(c.card_id, c.id)}
              >
                {openCards[c.card_id] ? <TriangleDownIcon /> : <TriangleRightIcon />}
              </span>
              <CardListItem card={c} />
            </div>
            {openCards[c.card_id] && (
              <div className="ml-6">
                {loading[c.card_id] ? (
                  <div className="text-gray-500 text-sm">Loading children...</div>
                ) : (
                  <ChildrenCards
                    allChildren={loadedChildren[c.card_id] || []}
                    card={c}
                  />
                )}
              </div>
            )}
          </li>
        ))}
      </ul>
    </div>
  );
}
