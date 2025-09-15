import React from "react";
import { Button } from "../Button";
import { PinIcon } from "../../assets/icons/PinIcon";
import { Card } from "../../models/Card";

interface PinButtonProps {
  card: Card;
  isPinned: boolean;
  onTogglePin: () => void;
}

export const PinButton: React.FC<PinButtonProps> = ({
  card,
  isPinned,
  onTogglePin
}) => {
  return (
    <div title={isPinned ? 'Unpin Card' : 'Pin Card'}>
      <Button
        onClick={onTogglePin}
        variant="outline"
        className={`${
          isPinned
            ? 'text-blue-600 bg-blue-50 border-blue-200 hover:bg-blue-100'
            : 'text-gray-600 hover:text-gray-800 hover:bg-gray-50'
        }`}
      >
        <PinIcon className="h-4 w-4" filled={isPinned} />
      </Button>
    </div>
  );
};