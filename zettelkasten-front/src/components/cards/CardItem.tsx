import React, { useState } from 'react';
import { Link } from 'react-router-dom';
import { PartialCard } from '../../models/Card';

import { CardPreviewWindow } from './CardPreviewWindow';
import { CardLink } from './CardLink';

interface CardItemProps {
  card: PartialCard;
}

export function CardItem({ card }: CardItemProps) {
  const [showHover, setShowHover] = useState(false);
  const [mousePosition, setMousePosition] = useState({ x: 0, y: 0 });

  const handleMouseEnter = (e: React.MouseEvent) => {
    setMousePosition({ x: e.clientX, y: e.clientY });
    setShowHover(true);
  };

  return (
    <div key={card.id} className="text-palette-darkest py-1.5 px-2 text-sm">
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
      {showHover && card && (
        <CardPreviewWindow cardPK={card.id} mousePosition={mousePosition} />
      )}
    </div>
  );
}
