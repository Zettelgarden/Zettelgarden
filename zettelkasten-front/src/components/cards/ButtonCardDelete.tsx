import React from "react";
import { Button } from "../Button";
import { Card } from "../../models/Card";
import { deleteCard } from "../../api/cards";

interface ButtonCardDeleteProps {
  card: Card;
  setMessage: (message: string) => void;
  onSuccess?: () => void;
}

export function ButtonCardDelete({ card, setMessage, onSuccess }: ButtonCardDeleteProps) {
  function handleDeleteButtonClick() {
    if (
      window.confirm(
        "Are you sure you want to delete this card? This cannot be reversed"
      )
    ) {
      deleteCard(card.id)
        .then(() => {
          setMessage("Card deleted successfully");
          onSuccess?.();
        })
        .catch((error) =>
          setMessage(
            "Unable to delete card. Does it have backlinks, children or files?"
          )
        );
    }
  }

  return <Button onClick={handleDeleteButtonClick} children={"Delete"} />;
}
