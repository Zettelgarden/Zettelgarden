import React, { useState } from "react";
import { PartialCard, Card } from "../../models/Card";
import { CardPreviewWindow } from "./CardPreviewWindow";
import { CardLink } from "./CardLink";
import { PlusCircleIcon } from "../../assets/icons/PlusCircleIcon";
import { formatDate } from "../../utils/dates";
import { useUIState } from "../../contexts/UIStateContext";
import { useTagContext } from "../../contexts/TagContext";
import { useNavigate } from "react-router-dom";
import { Menu } from "@headlessui/react";
import { CardIdDiscoveryDialog } from "./CardIdDiscoveryDialog";
import { CardListMenu } from "./CardListMenu";
import { getCard, saveExistingCard } from "../../api/cards";
import { togglePartialCardStar } from "../../utils/cardActions";

interface CardListItemProps {
  card: PartialCard;
  showAddButton?: boolean;
  onCardUpdate?: () => void;
}

export function CardListItem({
  card,
  showAddButton = true,
  onCardUpdate,
}: CardListItemProps) {
  const [showHover, setShowHover] = useState(false);
  const [mousePosition, setMousePosition] = useState({ x: 0, y: 0 });
  const [showRecategoryDialog, setShowRecategoryDialog] = useState(false);

  const { setLastCard } = useUIState();
  const { tags } = useTagContext();

  const navigate = useNavigate();

  const handleMouseEnter = (e: React.MouseEvent) => {
    setMousePosition({ x: e.clientX, y: e.clientY });
    setShowHover(true);
  };

  function handleAddCardClick() {
    setLastCard(card);
    navigate("/app/card/new");
  }

  function handleEditClick() {
    navigate(`/app/card/${card.id}/edit`);
  }

  function handleRecategoryClick() {
    setShowRecategoryDialog(true);
  }

  async function handleStarToggle() {
    try {
      await togglePartialCardStar(card, onCardUpdate);
    } catch (error) {
      console.error("Failed to toggle star:", error);
    }
  }

  const handleAddTag = async (tagName: string) => {
    try {
      // Fetch the full card data
      const fullCard = await getCard(card.id.toString());

      // Add the tag to the card body
      const editedCard: Card = {
        ...fullCard,
        body: fullCard.body + "\n\n#" + tagName,
      };

      // Save the updated card
      await saveExistingCard(editedCard);

      console.log(`Tag #${tagName} added to card ${card.id}`);

      // Trigger refresh if callback provided
      if (onCardUpdate) {
        onCardUpdate();
      }
    } catch (error) {
      console.error("Failed to add tag to card:", error);
    }
  };

  const handleRemoveTag = async (tagName: string) => {
    try {
      // Fetch the full card data
      const fullCard = await getCard(card.id.toString());

      // Remove the tag from the card body using regex
      const tagRegex = new RegExp(`\\n*#${tagName}\\b`, 'g');
      const editedCard: Card = {
        ...fullCard,
        body: fullCard.body.replace(tagRegex, ''),
      };

      // Save the updated card
      await saveExistingCard(editedCard);

      console.log(`Tag #${tagName} removed from card ${card.id}`);

      // Trigger refresh if callback provided
      if (onCardUpdate) {
        onCardUpdate();
      }
    } catch (error) {
      console.error("Failed to remove tag from card:", error);
    }
  };

  async function handleCardIdSelection(newCardId: string) {
    try {
      // Fetch the full card data
      const fullCard = await getCard(card.id.toString());

      // Update the card_id
      const updatedCard: Card = {
        ...fullCard,
        card_id: newCardId,
      };

      // Save the updated card
      await saveExistingCard(updatedCard);

      // Update the local card data
      card.card_id = newCardId;

      // Close the dialog
      setShowRecategoryDialog(false);

      // Trigger refresh if callback provided
      if (onCardUpdate) {
        onCardUpdate();
      }
    } catch (error) {
      console.error("Failed to update card:", error);
      // TODO: Show error message to user
    }
  }

  return (
    <div key={card.id} className="text-palette-darkest py-1.5 px-2 flex w-full text-sm items-center hover:bg-gray-50 transition-colors duration-150 rounded-lg">
      <div className="pr-3 flex-1 min-w-0 overflow-hidden">
        <span
          onMouseEnter={handleMouseEnter}
          onMouseLeave={() => setShowHover(false)}
        >
          <CardLink
            card={card}
            handleViewBacklink={(id: number) => { }}
            showTitle={true}
            showTags={true}
            onRemoveTag={handleRemoveTag}
            showTagRemoval={true}
          />
        </span>
      </div>

      <div className="flex text-xs flex-shrink-0 mr-2 w-20">{formatDate(card.created_at.toISOString())}</div>

      {/* Hamburger Menu */}
      <CardListMenu
        cardId={card.id}
        onEditClick={handleEditClick}
        onAddTag={handleAddTag}
        onRecategoryClick={handleRecategoryClick}
        showRecategory={card.card_id === ""}
        tags={tags}
        isStarred={card.is_starred ?? false}
        onToggleStar={handleStarToggle}
      />

      {showHover && card && (
        <CardPreviewWindow cardPK={card.id} mousePosition={mousePosition} />
      )}

      {/* Recategory Dialog */}
      {showRecategoryDialog && (
        <CardIdDiscoveryDialog
          onClose={() => setShowRecategoryDialog(false)}
          onSelectId={handleCardIdSelection}
        />
      )}
    </div>
  );
}
