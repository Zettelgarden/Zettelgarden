import React, { useEffect, useState } from "react";
import { Transition } from "@headlessui/react";
import { getCard } from "../../api/cards";
import { Card } from "../../models/Card";
import { CardBody } from "./CardBody";
import { isErrorResponse } from "../../models/common";
import { formatDate } from "../../utils/dates";

interface CardPreviewWindowProps {
  cardPK: number;
  mousePosition: { x: number; y: number };
}

export function CardPreviewWindow({
  cardPK,
  mousePosition,
}: CardPreviewWindowProps) {
  const [viewingCard, setViewingCard] = useState<Card | null>(null);
  const [error, setError] = useState("");
  const [isVisible, setIsVisible] = useState(false);

  async function fetchCard(id: string) {
    let refreshed = await getCard(id);

    if (isErrorResponse(refreshed)) {
      setError(refreshed["error"]);
    } else {
      setViewingCard(refreshed);
      setIsVisible(true);
    }
  }

  useEffect(() => {
    setIsVisible(false);
    if (cardPK !== undefined && cardPK !== null) {
      fetchCard(cardPK.toString());
    } else {
      setError("Invalid card ID");
    }
  }, [cardPK]);

  const windowHeight = window.innerHeight;
  const windowWidth = window.innerWidth;
  const previewHeight = 400;
  const previewWidth = 500;

  const topPosition =
    mousePosition.y + previewHeight > windowHeight
      ? Math.max(10, mousePosition.y - previewHeight - 20)
      : mousePosition.y + 10;

  const leftPosition =
    mousePosition.x + previewWidth > windowWidth
      ? Math.max(10, mousePosition.x - previewWidth - 20)
      : mousePosition.x + 10;

  return (
    <Transition
      show={isVisible}
      enter="transition-opacity duration-150"
      enterFrom="opacity-0"
      enterTo="opacity-100"
      leave="transition-opacity duration-100"
      leaveFrom="opacity-100"
      leaveTo="opacity-0"
    >
      <div
        className="fixed z-50 bg-white rounded-lg shadow-2xl border border-gray-200 p-4 max-w-xl"
        style={{
          top: topPosition,
          left: leftPosition,
          maxHeight: "400px",
          width: "500px",
        }}
      >
        {error && (
          <div className="bg-red-50 border border-red-200 rounded-md p-3 mb-3">
            <div className="text-red-700 text-sm">{error}</div>
          </div>
        )}
        {viewingCard && (
          <div className="space-y-3">
            <div className="border-b border-gray-200 pb-3">
              <div className="flex items-center gap-2 mb-1">
                <span className="font-semibold text-blue-600 text-sm">
                  [{viewingCard.card_id}]
                </span>
                <span className="text-gray-700 font-medium text-sm truncate">
                  {viewingCard.title}
                </span>
              </div>
              <p className="text-xs text-gray-500">
                {formatDate(viewingCard.created_at.toISOString())}
              </p>
            </div>
            <div className="overflow-y-auto max-h-64 prose prose-sm max-w-none">
              <CardBody viewingCard={viewingCard} entities={viewingCard.entities} />
            </div>
          </div>
        )}
      </div>
    </Transition>
  );
}
