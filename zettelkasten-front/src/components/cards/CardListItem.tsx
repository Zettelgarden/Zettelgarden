import React, { useState } from "react";
import { PartialCard, Card } from "../../models/Card";
import { CardPreviewWindow } from "./CardPreviewWindow";
import { CardLink } from "./CardLink";
import { PlusCircleIcon } from "../../assets/icons/PlusCircleIcon";
import { formatDate } from "../../utils/dates";
import { usePartialCardContext } from "../../contexts/CardContext";
import { useNavigate } from "react-router-dom";
import { Menu } from "@headlessui/react";
import { CardIdDiscoveryDialog } from "./CardIdDiscoveryDialog";
import { getCard, saveExistingCard } from "../../api/cards";

interface CardListItemProps {
  card: PartialCard;
  showAddButton?: boolean;
}

export function CardListItem({
  card,
  showAddButton = true,
}: CardListItemProps) {
  const [showHover, setShowHover] = useState(false);
  const [mousePosition, setMousePosition] = useState({ x: 0, y: 0 });
  const [showRecategoryDialog, setShowRecategoryDialog] = useState(false);

  const { setLastCard } = usePartialCardContext();

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

      // Optionally refresh the page or emit an event to refresh the card list
      window.location.reload();
    } catch (error) {
      console.error("Failed to update card:", error);
      // TODO: Show error message to user
    }
  }

  return (
    <div key={card.id} className="card-item py-2 px-2.5 flex w-full text-sm items-center">
      <div className="pr-4">
        <span
          onMouseEnter={handleMouseEnter}
          onMouseLeave={() => setShowHover(false)}
        >
          <CardLink
            card={card}
            handleViewBacklink={(id: number) => { }}
            showTitle={true}
          />
        </span>
      </div>

      <div className="flex-grow">
        {showAddButton && (
          <span onClick={handleAddCardClick}>
            <PlusCircleIcon />
          </span>
        )}
      </div>

      <div className="flex text-xs">{formatDate(card.created_at.toISOString())}</div>

      {/* Hamburger Menu */}
      <Menu as="div" className="relative">
        <Menu.Button className="rounded hover:bg-gray-100 transition-colors">
          <svg
            className="w-4 h-4 text-gray-500"
            fill="none"
            stroke="currentColor"
            viewBox="0 0 24 24"
          >
            <path
              strokeLinecap="round"
              strokeLinejoin="round"
              strokeWidth={2}
              d="M12 5v.01M12 12v.01M12 19v.01M12 6a1 1 0 110-2 1 1 0 010 2zm0 7a1 1 0 110-2 1 1 0 010 2zm0 7a1 1 0 110-2 1 1 0 010 2z"
            />
          </svg>
        </Menu.Button>

        <Menu.Items className="absolute right-0 z-10 mt-1 w-32 origin-top-right bg-white border border-gray-200 rounded-md shadow-lg focus:outline-none">
          <div className="py-1">
            <Menu.Item>
              {({ active }) => (
                <button
                  onClick={handleEditClick}
                  className={`${active ? 'bg-gray-100' : ''
                    } flex w-full items-center px-3 py-2 text-sm text-gray-700 hover:bg-gray-100`}
                >
                  <svg
                    className="w-4 h-4 mr-2"
                    fill="none"
                    stroke="currentColor"
                    viewBox="0 0 24 24"
                  >
                    <path
                      strokeLinecap="round"
                      strokeLinejoin="round"
                      strokeWidth={2}
                      d="M11 5H6a2 2 0 00-2 2v11a2 2 0 002 2h11a2 2 0 002-2v-5m-1.414-9.414a2 2 0 112.828 2.828L11.828 15H9v-2.828l8.586-8.586z"
                    />
                  </svg>
                  Edit
                </button>
              )}
            </Menu.Item>

            {card.card_id === "" && (
              <Menu.Item>
                {({ active }) => (
                  <button
                    onClick={handleRecategoryClick}
                    className={`${active ? 'bg-gray-100' : ''
                      } flex w-full items-center px-3 py-2 text-sm text-gray-700 hover:bg-gray-100`}
                  >
                    <svg
                      className="w-4 h-4 mr-2"
                      fill="none"
                      stroke="currentColor"
                      viewBox="0 0 24 24"
                    >
                      <path
                        strokeLinecap="round"
                        strokeLinejoin="round"
                        strokeWidth={2}
                        d="M7 7h.01M7 3h5c.512 0 1.024.195 1.414.586l7 7a2 2 0 010 2.828l-7 7a2 2 0 01-2.828 0l-7-7A1.994 1.994 0 013 12V7a4 4 0 014-4z"
                      />
                    </svg>
                    Recategory
                  </button>
                )}
              </Menu.Item>
            )}
          </div>
        </Menu.Items>
      </Menu>

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
