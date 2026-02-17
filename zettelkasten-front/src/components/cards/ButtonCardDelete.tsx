import React from "react";
import { Button } from "../Button";
import { Card } from "../../models/Card";
import { deleteCard } from "../../api/cards";
import { useToast } from "../toast/ToastContext";

interface ButtonCardDeleteProps {
  card: Card;
  onSuccess?: () => void;
}

export function ButtonCardDelete({ card, onSuccess }: ButtonCardDeleteProps) {
  const { showToast } = useToast();

  function handleDeleteButtonClick() {
    if (
      window.confirm(
        "Are you sure you want to delete this card? This cannot be reversed"
      )
    ) {
      deleteCard(card.id)
        .then(() => {
          showToast("success", "Card Deleted", "The card has been deleted successfully");
          onSuccess?.();
        })
        .catch((error) =>
          showToast("error", "Delete Failed", "Unable to delete card. Does it have backlinks, children or files?")
        );
    }
  }

  return <Button onClick={handleDeleteButtonClick} children={"Delete"} />;
}
