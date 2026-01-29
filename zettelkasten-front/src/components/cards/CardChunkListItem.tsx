import React, { useState } from "react";
import { CardChunk } from "../../models/Card";
import { CardPreviewWindow } from "./CardPreviewWindow";
import { CardLink } from "./CardLink";
import { PlusCircleIcon } from "../../assets/icons/PlusCircleIcon";
import { formatDate } from "../../utils/dates";
import { useUIState } from "../../contexts/UIStateContext";
import { useNavigate } from "react-router-dom";

interface CardChunkListItemProps {
  card: CardChunk;
  showAddButton?: boolean;
}

export function CardChunkListItem({
  card,
  showAddButton = true,
}: CardChunkListItemProps) {
  const [showHover, setShowHover] = useState(false);
  const [mousePosition, setMousePosition] = useState({ x: 0, y: 0 });

  const { setLastCard } = useUIState();

  const navigate = useNavigate();

  const handleMouseEnter = (e: React.MouseEvent) => {
    setMousePosition({ x: e.clientX, y: e.clientY });
    setShowHover(true);
  };

  function handleAddCardClick() {
    setLastCard(card);
    navigate("/app/card/new");
  }

  console.log(card)
  return (
    <div
      key={card.id}
      className="text-palette-darkest py-1 px-2.5 flex justify-between items-start w-full"
    >
      {/* Left side - blue tag */}
      <div className="flex-shrink-0"></div>

      {/* Middle section - title and text */}
      <div className="flex-grow">
        <div className="flex flex-col">
          <span
            onMouseEnter={handleMouseEnter}
            onMouseLeave={() => setShowHover(false)}
          >
            <CardLink
              card={card}
              handleViewBacklink={(id: number) => {}}
              showTitle={true}
            />
          </span>
          <div className="mt-1 px-4 text-sm italic">{card.ranking} - {card.body}</div>
        </div>
        {showAddButton && (
          <span onClick={handleAddCardClick}>
            <PlusCircleIcon />
          </span>
        )}
      </div>

      {/* Right side - date */}
      <div className="flex-shrink-0 text-xs">
        {formatDate(card.created_at.toISOString())}
      </div>

      {showHover && card && (
        <CardPreviewWindow cardPK={card.id} mousePosition={mousePosition} />
      )}
    </div>
  );
}
