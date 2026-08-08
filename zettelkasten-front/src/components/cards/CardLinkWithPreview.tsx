import React, { useState } from 'react';

import { Card } from '../../models/Card';
import { CardPreviewWindow } from './CardPreviewWindow';
import { CardLink } from './CardLink';

interface CardLinkWithPreviewProps {
  currentCard: Card;
  card_id: string;
  displayText?: string;
  resolveTitle?: boolean;
  handleViewBacklink: (card_id: number) => void;
}

export function CardLinkWithPreview({
  currentCard,
  card_id,
  displayText,
  resolveTitle,
  handleViewBacklink,
}: CardLinkWithPreviewProps) {
  const [showHover, setShowHover] = useState(false);
  const linkedCard = currentCard.references
    .filter((x) => x !== null)
    .find((linked) => linked.card_id === card_id);
  const [mousePosition, setMousePosition] = useState({ x: 0, y: 0 });

  const handleMouseEnter = (e: React.MouseEvent) => {
    setMousePosition({ x: e.clientX, y: e.clientY });
    setShowHover(true);
  };

  return (
    <span
      onMouseEnter={handleMouseEnter}
      onMouseLeave={() => setShowHover(false)}
      className="inline-block"
    >
      {linkedCard && (
        <CardLink
          card={linkedCard}
          handleViewBacklink={handleViewBacklink}
          showTitle={false}
          displayText={
            resolveTitle && linkedCard?.title ? linkedCard.title : displayText
          }
        />
      )}
      {showHover && linkedCard && (
        <CardPreviewWindow
          cardPK={linkedCard.id}
          mousePosition={mousePosition}
        />
      )}
    </span>
  );
}
